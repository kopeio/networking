package netutil

import (
	"bytes"
	"fmt"

	"github.com/golang/glog"
	"github.com/kopeio/route-controller/pkg/routecontroller"
	"github.com/vishvananda/netlink"
)

type XfrmStateTable struct {
}

func (p *XfrmStateTable) Flush() error {
	return netlink.XfrmStateFlush(0)
}

func (p *XfrmStateTable) Ensure(expectedList []*netlink.XfrmState) error {
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
	for _, e := range expectedList {
		if expected[e.Spi] != nil {
			glog.Fatalf("Found duplicate ESP SPI %d %v %v", e.Spi, e, expected[e.Spi])
		}
		expected[e.Spi] = e
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

	for spi, a := range actualMap {
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
				return fmt.Errorf("error creating state %v: %v", p, err)
			}
		}
	}
	if len(updates) != 0 {
		for _, p := range updates {
			glog.Infof("updating state %v", routecontroller.AsJsonString(p))
			err := netlink.XfrmStateUpdate(p)
			if err != nil {
				return fmt.Errorf("error updating state %v: %v", p, err)
			}
		}
	}

	if len(remove) != 0 {
		for _, p := range remove {
			glog.Infof("removing state %v", routecontroller.AsJsonString(p))
			err := netlink.XfrmStateDel(p)
			if err != nil {
				return fmt.Errorf("error removing state: %v", err)
			}
		}
	}

	return nil
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
