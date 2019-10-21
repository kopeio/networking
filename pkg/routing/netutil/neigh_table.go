package netutil

import (
	"bytes"
	"fmt"

	"github.com/vishvananda/netlink"
	"k8s.io/klog"
	"kope.io/networking/pkg/util"
)

type NeighTable struct {
}

func NewNeighTable(linkName string, linkIndex int) (*NeighTable, error) {
	t := &NeighTable{}

	return t, nil
}

func (t *NeighTable) Ensure(link netlink.Link, expected []*netlink.Neigh) error {
	linkName := link.Attrs().Name
	linkIndex := link.Attrs().Index

	klog.V(2).Infof("NETLINK: ip neigh show dev %s", linkName)
	actualList, err := netlink.NeighList(linkIndex, netlink.FAMILY_ALL)
	if err != nil {
		return fmt.Errorf("error listing layer2 config: %v", err)
	}

	// TODO: using strings as layer2 key is inefficient
	actualMap := make(map[string]*netlink.Neigh)
	for i := range actualList {
		a := &actualList[i]
		if a.IP == nil {
			klog.Errorf("ignoring unexpected layer2 entry with no IP: %v", a)
			continue
		}
		k := a.IP.String()
		actualMap[k] = a
		klog.V(4).Infof("Actual layer2 entry: %v", util.AsJsonString(a))
	}

	expectedMap := make(map[string]*netlink.Neigh)

	for _, e := range expected {
		if e.IP == nil {
			klog.Errorf("ignoring unexpected layer2 entry with no IP: %v", e)
			continue
		}
		k := e.IP.String()
		expectedMap[k] = e
		klog.V(4).Infof("Expected layer2 entry: %v", util.AsJsonString(e))
	}

	var upsert []*netlink.Neigh

	for k, e := range expectedMap {
		a := actualMap[k]

		if a == nil {
			upsert = append(upsert, e)
			continue
		}

		if !neighEqual(a, e) {
			klog.V(2).Infof("neigh change for %s:\n\t%s\n\t%s", k, util.AsJsonString(a), util.AsJsonString(e))
			upsert = append(upsert, e)
		}
	}

	// We don't remove neighbour entries
	//var remove []*netlink.Neigh
	//for k, a := range actualMap {
	//	e := expectedMap[k]
	//
	//	if e == nil {
	//		remove = append(remove, a)
	//		continue
	//	}
	//}
	//
	//if len(remove) != 0 {
	//	for _, r := range remove {
	//		klog.Infof("NETLINK: ip neigh delete %v", routecontroller.AsJsonString(r))
	//		klog.V(2).Infof(" full neigh: %v", routecontroller.AsJsonString(r))
	//		err := netlink.NeighDel(r)
	//		if err != nil {
	//			return fmt.Errorf("error removing route entry: %v", err)
	//		}
	//	}
	//}

	if len(upsert) != 0 {
		for _, r := range upsert {
			klog.Infof("NETLINK: ip neigh replace to %s lladdr %s dev %d", r.IP, r.HardwareAddr, r.LinkIndex)
			klog.V(2).Infof(" full neigh: %v", util.AsJsonString(r))
			err := netlink.NeighSet(r)
			if err != nil {
				return fmt.Errorf("error doing `ip neigh replace to %s lladdr %s dev %d`: %v", r.IP, r.HardwareAddr, r.LinkIndex, err)
			}
		}
	}

	return nil
}

func neighEqual(a, e *netlink.Neigh) bool {
	if a.Type != e.Type || a.Family != e.Family || a.Flags != e.Flags || a.LinkIndex != e.LinkIndex || a.State != e.State {
		return false
	}
	if !a.IP.Equal(e.IP) {
		return false
	}
	if !bytes.Equal(a.HardwareAddr, e.HardwareAddr) {
		return false
	}
	return true
}
