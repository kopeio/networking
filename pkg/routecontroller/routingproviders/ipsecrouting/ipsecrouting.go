package ipsecrouting

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net"
	"syscall"

	"github.com/golang/glog"
	"github.com/kopeio/route-controller/pkg/routecontroller"
	"github.com/kopeio/route-controller/pkg/routecontroller/routingproviders"
	"github.com/vishvananda/netlink"
	"k8s.io/kubernetes/pkg/api"
)

const (
	XFRM_PROTO_UDP netlink.Proto = syscall.IPPROTO_UDP
)

var ipnetAll *net.IPNet = &net.IPNet{
	Mask: net.IPv4Mask(0, 0, 0, 0),
	IP:   net.IPv4(0, 0, 0, 0),
}

const NoByteCountLimit = uint64(0xffffffffffffffff)
const NoPacketCountLimit = uint64(0xffffffffffffffff)

var noLimits = netlink.XfrmStateLimits{
	ByteSoft:    NoByteCountLimit,
	ByteHard:    NoByteCountLimit,
	PacketSoft:  NoPacketCountLimit,
	PacketHard:  NoPacketCountLimit,
	TimeSoft:    0,
	TimeHard:    0,
	TimeUseSoft: 0,
	TimeUseHard: 0,
}

type IpsecRoutingProvider struct {
}

var _ routingproviders.RoutingProvider = &IpsecRoutingProvider{}

func NewIpsecRoutingProvider() (*IpsecRoutingProvider, error) {
	p := &IpsecRoutingProvider{}
	return p, nil
}

func (p *IpsecRoutingProvider) EnsureCIDRs(me *api.Node, allNodes []api.Node) error {
	meNodeNumeral, err := computeNodeNumeral(me.Spec.PodCIDR)
	if err != nil {
		return err
	}

	meInternalIP := routecontroller.FindInternalIPAddress(me)
	if meInternalIP == "" {
		glog.Infof("self-node does not yet have internalIP; delaying configuration")
		return nil
	}

	myIP := net.ParseIP(meInternalIP)
	if myIP == nil {
		return fmt.Errorf("cannot parse my IP %q", meInternalIP)
	}

	// TODO: Can we / should we share these (do things by CIDR?)

	{
		actualList, err := netlink.XfrmStateList(netlink.FAMILY_ALL)
		if err != nil {
			return fmt.Errorf("error listing xfrm state: %v", err)
		}

		actualMap := make(map[int]*netlink.XfrmState)
		for i := range actualList {
			a := &actualList[i]
			actualMap[a.Spi] = a
			glog.Infof("Actual State: %v", routecontroller.AsJsonString(a))
		}

		expected := make(map[int]*netlink.XfrmState)

		cidrMap := routecontroller.BuildCIDRMap(me, allNodes)

		for remoteCIDRString, remoteIPString := range cidrMap {
			remoteNodeNumeral, err := computeNodeNumeral(remoteCIDRString)
			if err != nil {
				return err
			}

			remoteIP := net.ParseIP(remoteIPString)
			if remoteIP == nil {
				return fmt.Errorf("cannot parse remote IP %q", remoteIPString)
			}

			// dir isn't explicit in state rules, but we use it to avoid code duplication
			for _, dir := range []netlink.Dir{netlink.XFRM_DIR_IN, netlink.XFRM_DIR_OUT} {
				glog.Errorf("Using hard-coded (and stupid) encryption keys - NO SECURITY ")
				ahKey := []byte{0x1, 0x2, 0x3, 0x4, 0x5, 0x6, 0x7, 0x8, 0x9, 0xa}
				espKey := []byte{0x4, 0x5, 0x9, 0x4, 0x3, 0x6, 0x7, 0x8, 0x9, 0xa,
					0x4, 0x5, 0x9, 0x4, 0x3, 0x6, 0x7, 0x8, 0x9, 0xa}

				{

					// AH outbound
					// TODO: Does this need to be XFRM_MODE_TUNNEL??
					p := &netlink.XfrmState{
						Proto: netlink.XFRM_PROTO_AH,
						Mode:  netlink.XFRM_MODE_TUNNEL,
						Auth: &netlink.XfrmStateAlgo{
							Name:        "hmac(sha1)",
							Key:         ahKey,
							TruncateLen: 96,
						},
					}

					p.Limits = noLimits

					if dir == netlink.XFRM_DIR_OUT {
						p.Src = myIP
						p.Dst = remoteIP

						spi := uint32(0xc0000000)
						spi |= meNodeNumeral << 16
						spi |= remoteNodeNumeral << 2
						spi |= 0x0
						p.Spi = int(spi)
					} else {
						p.Src = remoteIP
						p.Dst = myIP

						spi := uint32(0xc0000000)
						spi |= remoteNodeNumeral << 16
						spi |= meNodeNumeral << 2
						spi |= 0x0
						p.Spi = int(spi)
					}
					expected[p.Spi] = p
				}

				{
					// ESP outbound
					// TODO: Does this need to be XFRM_MODE_TUNNEL??
					p := &netlink.XfrmState{
						Proto: netlink.XFRM_PROTO_ESP,
						Mode:  netlink.XFRM_MODE_TUNNEL,
						Crypt: &netlink.XfrmStateAlgo{
							Name: "rfc3686(ctr(aes))",
							Key:  espKey,
						},
						Encap: &netlink.XfrmStateEncap{
							Type:            netlink.XFRM_ENCAP_ESPINUDP,
							SrcPort:         4500,
							DstPort:         4500,
							OriginalAddress: net.IPv4(0, 0, 0, 0),
						},
					}
					p.Limits = noLimits

					if dir == netlink.XFRM_DIR_OUT {
						p.Src = myIP
						p.Dst = remoteIP

						spi := uint32(0xc0000000)
						spi |= meNodeNumeral << 16
						spi |= remoteNodeNumeral << 2
						spi |= 0x1
						p.Spi = int(spi)
					} else {
						p.Src = remoteIP
						p.Dst = myIP

						spi := uint32(0xc0000000)
						spi |= remoteNodeNumeral << 16
						spi |= meNodeNumeral << 2
						spi |= 0x1
						p.Spi = int(spi)
					}

					expected[p.Spi] = p
				}
			}
		}

		var create []*netlink.XfrmState
		var remove []*netlink.XfrmState
		var updates []*netlink.XfrmState

		// TODO: Figure out what the actual key is!

		for spi, e := range expected {
			a := actualMap[spi]

			if a == nil {
				create = append(create, e)
				continue
			}

			if !xfrmStateEqual(a, e) {
				glog.Infof("State change for %d:\n\t%s\n\t%s", spi, routecontroller.AsJsonString(a), routecontroller.AsJsonString(e))
				updates = append(updates, e)
			}
		}

		for spi, a := range expected {
			e := expected[spi]

			if e == nil {
				remove = append(remove, a)
				continue
			}
		}

		if len(create) != 0 {
			for _, p := range create {
				glog.Infof("creating state %v", routecontroller.AsJsonString(p))
				err := netlink.XfrmStateAdd(p)
				if err != nil {
					glog.Warningf("error creating state: %v", err)
				}
			}
		}
		if len(updates) != 0 {
			for _, p := range updates {
				glog.Infof("updating state %v", routecontroller.AsJsonString(p))
				err := netlink.XfrmStateUpdate(p)
				if err != nil {
					glog.Warningf("error updating state: %v", err)
				}
			}
		}

		if len(remove) != 0 {
			for _, p := range remove {
				glog.Infof("removing state %v", routecontroller.AsJsonString(p))
				err := netlink.XfrmStateDel(p)
				if err != nil {
					glog.Warningf("error removing state: %v", err)
				}
			}
		}
		glog.Errorf("Need to implement state add / remove")
	}

	{
		actual, err := netlink.XfrmPolicyList(netlink.FAMILY_ALL)
		if err != nil {
			return fmt.Errorf("error listing xfrm policies: %v", err)
		}

		for _, p := range actual {
			glog.Infof("Actual Policy: %v", routecontroller.AsJsonString(p))
		}

		var expected []*netlink.XfrmPolicy

		// No IPSEC for IPSEC over UDP (port 4500)
		for _, dir := range []netlink.Dir{netlink.XFRM_DIR_IN, netlink.XFRM_DIR_OUT, netlink.XFRM_DIR_FWD} {
			p := &netlink.XfrmPolicy{}
			p.Src = ipnetAll
			p.Dst = ipnetAll
			p.DstPort = 4500
			p.Dir = dir
			p.Proto = XFRM_PROTO_UDP
			p.Priority = 200

			expected = append(expected, p)
		}

		// If nothing else matches: no encryption
		for _, dir := range []netlink.Dir{netlink.XFRM_SOCKET_IN, netlink.XFRM_SOCKET_OUT} {
			p := &netlink.XfrmPolicy{}
			p.Src = ipnetAll
			p.Dst = ipnetAll
			p.Dir = dir
			p.Priority = 0

			expected = append(expected, p)
		}

		_, meCIDR, err := net.ParseCIDR(me.Spec.PodCIDR)
		if err != nil {
			return fmt.Errorf("error parsing my PodCidr %q: %v", me.Spec.PodCIDR, err)
		}

		cidrMap := routecontroller.BuildCIDRMap(me, allNodes)

		for remoteCIDRString, remoteIPString := range cidrMap {
			_, remoteCIDR, err := net.ParseCIDR(remoteCIDRString)
			if err != nil {
				return fmt.Errorf("error parsing PodCidr %q: %v", remoteCIDRString, err)
			}

			remoteIP := net.ParseIP(remoteIPString)
			if remoteIP == nil {
				return fmt.Errorf("cannot parse remote IP %q", remoteIPString)
			}

			for _, dir := range []netlink.Dir{netlink.XFRM_DIR_IN, netlink.XFRM_DIR_OUT, netlink.XFRM_DIR_FWD} {
				p := &netlink.XfrmPolicy{}
				if dir == netlink.XFRM_DIR_OUT {
					p.Src = meCIDR
					p.Dst = remoteCIDR
				} else {
					p.Src = remoteCIDR
					p.Dst = meCIDR
				}
				p.Dir = dir
				p.Priority = 100

				p.Tmpls = []netlink.XfrmPolicyTmpl{
					{
						Proto: netlink.XFRM_PROTO_ESP,
						Mode:  netlink.XFRM_MODE_TUNNEL,
					},
				}

				t := &p.Tmpls[0]

				if dir == netlink.XFRM_DIR_OUT {
					t.Src = myIP
					t.Dst = remoteIP
				} else {
					t.Src = remoteIP
					t.Dst = myIP
				}

				expected = append(expected, p)
			}
		}

		var create []*netlink.XfrmPolicy
		var remove []*netlink.XfrmPolicy
		var updates []*netlink.XfrmPolicy

		actualMatched := make([]bool, len(actual), len(actual))
		for _, e := range expected {
			var a *netlink.XfrmPolicy
			// TODO: Bucket by 'key' so we are not O(N^2)
			for i := range actual {
				p := &actual[i]
				if xfrmPolicyMatches(p, e) {
					if a != nil {
						glog.Warningf("Found duplicate matching policies: %v and %v", routecontroller.AsJsonString(p), routecontroller.AsJsonString(a))
					}
					a = p
					actualMatched[i] = true
				}
			}

			if a == nil {
				create = append(create, e)
				continue
			}

			// Avoid spurious changes
			e.Index = a.Index
			if !xfrmPolicyEqual(a, e) {
				glog.Infof("Policy changed:\n\t%s\n\t%s", a, e)
				updates = append(updates, e)
			}
		}

		for i := range actual {
			if actualMatched[i] {
				continue
			}
			remove = append(remove, &actual[i])
		}

		if len(create) != 0 {
			for _, p := range create {
				glog.Infof("creating policy %v", routecontroller.AsJsonString(p))
				err := netlink.XfrmPolicyAdd(p)
				if err != nil {
					glog.Warningf("error creating policy: %v", err)
				}
			}
		}

		if len(updates) != 0 {
			for _, p := range updates {
				glog.Infof("updating policy %v", routecontroller.AsJsonString(p))
				err := netlink.XfrmPolicyUpdate(p)
				if err != nil {
					glog.Warningf("error updating policy: %v", err)
				}
			}
		}

		if len(remove) != 0 {
			for _, p := range remove {
				glog.Infof("removing policy %v", routecontroller.AsJsonString(p))
				err := netlink.XfrmPolicyDel(p)
				if err != nil {
					glog.Warningf("error removing policy: %v", err)
				}
			}
		}
	}

	return nil
}

func xfrmPolicyEqual(l *netlink.XfrmPolicy, r *netlink.XfrmPolicy) bool {
	if l.Dir != r.Dir {
		return false
	}
	if l.Priority != r.Priority {
		return false
	}
	if l.Index != r.Index {
		return false
	}
	if l.SrcPort != r.SrcPort {
		return false
	}
	if l.DstPort != r.DstPort {
		return false
	}
	if l.Proto != r.Proto {
		return false
	}
	if !ipnetEqual(l.Dst, r.Dst) {
		return false
	}
	if !ipnetEqual(l.Src, r.Src) {
		return false
	}
	if !xfrmMarkEqual(l.Mark, r.Mark) {
		return false
	}

	if len(l.Tmpls) != len(r.Tmpls) {
		return false
	}
	for i := range l.Tmpls {
		if !xfrmPolicyTmplEqual(&l.Tmpls[i], &r.Tmpls[i]) {
			return false
		}
	}

	return true
}

func xfrmStateEqual(l *netlink.XfrmState, r *netlink.XfrmState) bool {
	if l.Proto != r.Proto {
		return false
	}
	if l.Mode != r.Mode {
		return false
	}
	if l.Spi != r.Spi {
		return false
	}
	if l.Reqid != r.Reqid {
		return false
	}
	if l.ReplayWindow != r.ReplayWindow {
		return false
	}
	if !l.Dst.Equal(r.Dst) {
		return false
	}
	if !l.Src.Equal(r.Src) {
		return false
	}

	if l.Limits != r.Limits {
		//!xfrmStateLimitsEqual(l.Limits, r.Limits)
		return false
	}

	if !xfrmMarkEqual(l.Mark, r.Mark) {
		return false
	}
	if !xfrmStateAlgoEqual(l.Auth, r.Auth) {
		return false
	}
	if !xfrmStateAlgoEqual(l.Crypt, r.Crypt) {
		return false
	}
	if !xfrmStateAlgoEqual(l.Aead, r.Aead) {
		return false
	}

	if !xfrmStateEncapEqual(l.Encap, r.Encap) {
		return false
	}

	return true
}

func xfrmStateAlgoEqual(l *netlink.XfrmStateAlgo, r *netlink.XfrmStateAlgo) bool {
	if l == nil || r == nil {
		return (r == nil) == (l == nil)
	}
	if l.TruncateLen != r.TruncateLen {
		return false
	}
	if l.ICVLen != r.ICVLen {
		return false
	}
	if l.Name != r.Name {
		return false
	}
	if !bytes.Equal(l.Key, r.Key) {
		return false
	}
	return true
}

func xfrmStateEncapEqual(l *netlink.XfrmStateEncap, r *netlink.XfrmStateEncap) bool {
	if l == nil || r == nil {
		return (r == nil) == (l == nil)
	}
	if l.Type != r.Type {
		return false
	}
	if l.SrcPort != r.SrcPort {
		return false
	}
	if l.DstPort != r.DstPort {
		return false
	}

	return l.OriginalAddress.Equal(r.OriginalAddress)
}

//func xfrmStateLimitsEqual(l *netlink.XfrmStateLimits, r *netlink.XfrmStateLimits) bool {
//	if l == nil || r == nil {
//		return (r == nil) == (l == nil)
//	}
//	//ByteSoft    uint64
//	//ByteHard    uint64
//	//PacketSoft  uint64
//	//PacketHard  uint64
//	//TimeSoft    uint64
//	//TimeHard    uint64
//	//TimeUseSoft uint64
//	//TimeUseHard uint64
//	return *l == *r
//}

func xfrmMarkEqual(l *netlink.XfrmMark, r *netlink.XfrmMark) bool {
	if l == nil || r == nil {
		return (r == nil) == (l == nil)
	}
	return l.Mask == r.Mask && l.Value == r.Value
}

func xfrmPolicyTmplEqual(l *netlink.XfrmPolicyTmpl, r *netlink.XfrmPolicyTmpl) bool {
	if l == nil || r == nil {
		return (r == nil) == (l == nil)
	}
	if l.Proto != r.Proto {
		return false
	}
	if l.Mode != r.Mode {
		return false
	}
	if l.Spi != r.Spi {
		return false
	}
	if l.Reqid != r.Reqid {
		return false
	}
	if !l.Dst.Equal(r.Dst) {
		return false
	}
	if !l.Src.Equal(r.Src) {
		return false
	}
	return true
}

func xfrmPolicyMatches(a *netlink.XfrmPolicy, e *netlink.XfrmPolicy) bool {
	if a.Dir != e.Dir {
		return false
	}
	if a.SrcPort != e.SrcPort {
		return false
	}
	if a.DstPort != e.DstPort {
		return false
	}
	if a.Proto != e.Proto {
		return false
	}
	if !ipnetEqual(a.Dst, e.Dst) {
		return false
	}
	if !ipnetEqual(a.Src, e.Src) {
		return false
	}
	return true
}

func ipnetEqual(a *net.IPNet, e *net.IPNet) bool {
	if a == nil {
		return e == nil
	}
	if e == nil {
		return a == nil
	}
	if !bytes.Equal(a.Mask, e.Mask) {
		return false
	}
	if !a.IP.Equal(e.IP) {
		return false
	}
	return true
}

func computeNodeNumeral(podCIDRString string) (uint32, error) {
	_, podCIDR, err := net.ParseCIDR(podCIDRString)
	if err != nil {
		return 0, fmt.Errorf("error parsing PodCidr %q: %v", podCIDRString, err)
	}

	podCIDRv4 := podCIDR.IP.To4()
	if podCIDRv4 == nil {
		return 0, fmt.Errorf("expected IPv4 PodCidr %q: %v", podCIDRString, err)
	}
	v := binary.BigEndian.Uint32(podCIDRv4)
	ones, bits := podCIDR.Mask.Size()
	v = v >> uint32(bits-ones)

	// We allow 14 bits of pods... things will break if we go over this
	// TODO: We have all the nodes; detect if we go over
	v = v & 0x3fff

	glog.Infof("Mapped CIDR %q -> %d", podCIDRString, v)
	return v, nil
}
