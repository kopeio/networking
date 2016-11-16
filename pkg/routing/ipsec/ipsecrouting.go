package ipsec

import (
	"encoding/binary"
	"fmt"
	"net"
	"syscall"

	"github.com/golang/glog"
	"github.com/vishvananda/netlink"
	"kope.io/networking/pkg/routing"
	"kope.io/networking/pkg/routing/netutil"
	"os/exec"
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
	authenticationStrategy AuthenticationStrategy
	encryptionStrategy     EncryptionStrategy
	encapsulationStrategy  EncapsulationStrategy

	udpEncapListener *UDPEncapListener

	xfrmPolicyTable *netutil.XfrmPolicyTable
	xfrmStateTable  *netutil.XfrmStateTable

	lastVersionApplied uint64
}

var _ routing.Provider = &IpsecRoutingProvider{}

func NewIpsecRoutingProvider(authenticationStrategy AuthenticationStrategy, encryptionStrategy EncryptionStrategy, encapsulationStrategy EncapsulationStrategy) (*IpsecRoutingProvider, error) {
	err := doModprobe()
	if err != nil {
		return nil, err
	}

	p := &IpsecRoutingProvider{
		authenticationStrategy: authenticationStrategy,
		encryptionStrategy:     encryptionStrategy,
		encapsulationStrategy:  encapsulationStrategy,

		xfrmPolicyTable: &netutil.XfrmPolicyTable{},
		xfrmStateTable:  &netutil.XfrmStateTable{},
	}

	// TODO: Refactor into encapsulationStrategy
	port := 4500
	glog.Infof("Creating encap listener on port %d", port)
	p.udpEncapListener, err = NewUDPEncapListener(port)
	if err != nil {
		return nil, fmt.Errorf("error creating UDP encapsulation listener on port %d: %v", port, err)
	}
	return p, nil
}

func (p *IpsecRoutingProvider) Flush() error {
	err := p.xfrmPolicyTable.Flush()
	if err != nil {
		return fmt.Errorf("error flushing xfrm policy table")
	}
	err = p.xfrmStateTable.Flush()
	if err != nil {
		return fmt.Errorf("error flushing xfrm policy table")
	}
	return nil
}

func (p *IpsecRoutingProvider) Close() error {
	if p.udpEncapListener != nil {
		err := p.udpEncapListener.Close()
		if err != nil {
			return err
		}
		p.udpEncapListener = nil
	}
	return nil
}

func doModprobe() error {
	modules := []string{"af_key",
		"ah4",
		"ipcomp",
		//"esp",
		//"xfrm4",
		"xfrm4_tunnel",
		//"tunnel",
	}
	for _, module := range modules {
		glog.Infof("Doing modprobe for module %v", module)
		out, err := exec.Command("/sbin/modprobe", module).CombinedOutput()
		outString := string(out)
		if err != nil {
			return fmt.Errorf("modprobe for module %q failed (%v): %s", module, err, outString)
		}
		if outString != "" {
			glog.Infof("Output from modprobe %s:\n%s", module, outString)
		}
	}
	return nil
}

func (p *IpsecRoutingProvider) EnsureCIDRs(nodeMap *routing.NodeMap) error {
	if p.lastVersionApplied != 0 && nodeMap.IsVersion(p.lastVersionApplied) {
		return nil
	}

	me, allNodes, version := nodeMap.Snapshot()

	if me == nil {
		return fmt.Errorf("Cannot find local node")
	}

	if me.PodCIDR == nil {
		return fmt.Errorf("No CIDR assigned to local node; cannot configure tunnels")
	}

	if me.Address == nil {
		return fmt.Errorf("No Address assigned to local node; cannot configure tunnels")
	}

	meNodeNumeral, err := computeNodeNumeral(me.PodCIDR)
	if err != nil {
		return err
	}

	{
		// TODO: Can we / should we share these (we can't do things by CIDR though, so it might be impossible)
		expected := make([]*netlink.XfrmState, 0, len(allNodes)*4)

		for i := range allNodes {
			remote := &allNodes[i]

			if remote.Name == me.Name {
				continue
			}

			if remote.Address == nil {
				glog.Infof("Node %q did not have address; ignoring", remote.Name)
				continue
			}
			if remote.PodCIDR == nil {
				glog.Infof("Node %q did not have PodCIDR; ignoring", remote.Name)
				continue
			}

			remoteNodeNumeral, err := computeNodeNumeral(remote.PodCIDR)
			if err != nil {
				return err
			}

			// dir isn't explicit in state rules, but we use it to avoid code duplication
			for _, dir := range []netlink.Dir{netlink.XFRM_DIR_IN, netlink.XFRM_DIR_OUT} {
				glog.Errorf("Using hard-coded (and stupid) encryption keys - NO SECURITY ")

				if p.authenticationStrategy.UseAH() {
					// AH outbound
					// TODO: Does this need to be XFRM_MODE_TUNNEL??
					s := &netlink.XfrmState{
						Proto: netlink.XFRM_PROTO_AH,
						Mode:  netlink.XFRM_MODE_TUNNEL,
					}

					s.Limits = noLimits

					if dir == netlink.XFRM_DIR_OUT {
						s.Src = me.Address
						s.Dst = remote.Address

						spi := uint32(0xc0000000)
						spi |= meNodeNumeral << 16
						spi |= remoteNodeNumeral << 2
						spi |= 0x0
						s.Spi = int(spi)

						p.authenticationStrategy.Apply(s, me, remote)
					} else {
						s.Src = remote.Address
						s.Dst = me.Address

						spi := uint32(0xc0000000)
						spi |= remoteNodeNumeral << 16
						spi |= meNodeNumeral << 2
						spi |= 0x0
						s.Spi = int(spi)

						p.authenticationStrategy.Apply(s, remote, me)
					}
					expected = append(expected, s)
				}

				if p.encryptionStrategy.UseESP() {
					// ESP outbound
					// TODO: Does this need to be XFRM_MODE_TUNNEL??
					s := &netlink.XfrmState{
						Proto: netlink.XFRM_PROTO_ESP,
						Mode:  netlink.XFRM_MODE_TUNNEL,
					}
					s.Limits = noLimits

					if dir == netlink.XFRM_DIR_OUT {
						s.Src = me.Address
						s.Dst = remote.Address

						spi := uint32(0xc0000000)
						spi |= meNodeNumeral << 16
						spi |= remoteNodeNumeral << 2
						spi |= 0x1
						s.Spi = int(spi)

						p.encryptionStrategy.Apply(s, me, remote)
						p.encapsulationStrategy.Apply(s, me, remote)
					} else {
						s.Src = remote.Address
						s.Dst = me.Address

						spi := uint32(0xc0000000)
						spi |= remoteNodeNumeral << 16
						spi |= meNodeNumeral << 2
						spi |= 0x1
						s.Spi = int(spi)

						p.encryptionStrategy.Apply(s, remote, me)
						p.encapsulationStrategy.Apply(s, remote, me)
					}
					expected = append(expected, s)
				}
			}
		}

		err := p.xfrmStateTable.Ensure(expected)
		if err != nil {
			return fmt.Errorf("error applying xfrm state: %v", err)
		}
	}

	{
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

		for _, remote := range allNodes {
			if remote.Name == me.Name {
				continue
			}

			if remote.Address == nil {
				glog.Infof("Node %q did not have address; ignoring", remote.Name)
				continue
			}
			if remote.PodCIDR == nil {
				glog.Infof("Node %q did not have PodCIDR; ignoring", remote.Name)
				continue
			}

			// TODO: Do we need forward??
			// TODO: Do we need to speciy that AH is required?  (and check that encryption is required)
			// TODO: Can we tie to a specific policy (or is that done by IP)
			for _, dir := range []netlink.Dir{netlink.XFRM_DIR_IN, netlink.XFRM_DIR_OUT, netlink.XFRM_DIR_FWD} {
				p := &netlink.XfrmPolicy{}
				if dir == netlink.XFRM_DIR_OUT {
					p.Src = me.PodCIDR
					p.Dst = remote.PodCIDR
				} else {
					p.Src = remote.PodCIDR
					p.Dst = me.PodCIDR
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
					t.Src = me.Address
					t.Dst = remote.Address
				} else {
					t.Src = remote.Address
					t.Dst = me.Address
				}

				expected = append(expected, p)
			}

			// TODO: Do we need forward??
			for _, dir := range []netlink.Dir{netlink.XFRM_DIR_IN, netlink.XFRM_DIR_OUT, netlink.XFRM_DIR_FWD} {
				p := &netlink.XfrmPolicy{}
				if dir == netlink.XFRM_DIR_OUT {
					p.Src = me.PodCIDR
					p.Dst = ipToIpnet(remote.Address)
				} else {
					p.Src = ipToIpnet(remote.Address)
					p.Dst = me.PodCIDR
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
					t.Src = me.Address
					t.Dst = remote.Address
				} else {
					t.Src = remote.Address
					t.Dst = me.Address
				}

				expected = append(expected, p)
			}

			// TODO: Do we need forward??
			for _, dir := range []netlink.Dir{netlink.XFRM_DIR_IN, netlink.XFRM_DIR_OUT, netlink.XFRM_DIR_FWD} {
				p := &netlink.XfrmPolicy{}
				if dir == netlink.XFRM_DIR_OUT {
					p.Src = ipToIpnet(me.Address)
					p.Dst = remote.PodCIDR
				} else {
					p.Src = remote.PodCIDR
					p.Dst = ipToIpnet(me.Address)
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
					t.Src = me.Address
					t.Dst = remote.Address
				} else {
					t.Src = remote.Address
					t.Dst = me.Address
				}

				expected = append(expected, p)
			}
		}

		err := p.xfrmPolicyTable.Ensure(expected)
		if err != nil {
			return fmt.Errorf("error applying xfrm policy: %v", err)
		}
	}

	p.lastVersionApplied = version

	return nil
}

func ipToIpnet(ip net.IP) *net.IPNet {
	return &net.IPNet{
		IP:   ip,
		Mask: net.IPv4Mask(255, 255, 255, 255),
	}
}

func computeNodeNumeral(podCIDR *net.IPNet) (uint32, error) {
	podCIDRv4 := podCIDR.IP.To4()
	if podCIDRv4 == nil {
		return 0, fmt.Errorf("expected IPv4 PodCidr %q", podCIDR)
	}
	v := binary.BigEndian.Uint32(podCIDRv4)
	ones, bits := podCIDR.Mask.Size()
	v = v >> uint32(bits-ones)

	// We allow 14 bits of pods... things will break if we go over this
	// TODO: We have all the nodes; detect if we go over
	v = v & 0x3fff

	glog.Infof("Mapped CIDR %q -> %d", podCIDR, v)
	return v, nil
}
