package netutil

import (
	"bytes"
	"fmt"

	"github.com/golang/glog"
	"github.com/kopeio/route-controller/pkg/routecontroller"
	"github.com/vishvananda/netlink"
)

type NeighTable struct {
	linkName  string
	linkIndex int
}

func NewNeighTable(linkName string, linkIndex int) (*NeighTable, error) {
	t := &NeighTable{
		linkName:  linkName,
		linkIndex: linkIndex,
	}

	return t, nil
}

func (t *NeighTable) Ensure(expected []*netlink.Neigh) error {
	glog.V(2).Infof("NETLINK: ip neigh show dev %s", t.linkName)
	glog.V(2).Infof("link index=%d", t.linkIndex)
	actualList, err := netlink.NeighList(t.linkIndex, netlink.FAMILY_ALL)
	if err != nil {
		return fmt.Errorf("error listing layer2 config: %v", err)
	}

	for _, e := range expected {
		e.LinkIndex = t.linkIndex
	}

	glog.Warningf("TODO: using strings as layer2 key is inefficient")
	actualMap := make(map[string]*netlink.Neigh)
	for i := range actualList {
		a := &actualList[i]
		if a.IP == nil {
			glog.Errorf("ignoring unexpected layer2 entry with no IP: %v", a)
			continue
		}
		k := a.IP.String()
		actualMap[k] = a
		glog.Infof("Actual layer2 entry: %v", routecontroller.AsJsonString(a))
	}

	expectedMap := make(map[string]*netlink.Neigh)

	for _, e := range expected {
		if e.IP == nil {
			glog.Errorf("ignoring unexpected layer2 entry with no IP: %v", e)
			continue
		}
		k := e.IP.String()
		expectedMap[k] = e
		glog.Infof("Expected layer2 entry: %v", routecontroller.AsJsonString(e))
	}

	var upsert []*netlink.Neigh
	var remove []*netlink.Neigh

	for k, e := range expectedMap {
		a := actualMap[k]

		if a == nil {
			upsert = append(upsert, e)
			continue
		}

		if !neighEqual(a, e) {
			glog.Infof("neigh change for %s:\n\t%s\n\t%s", k, routecontroller.AsJsonString(a), routecontroller.AsJsonString(e))
			//remove = append(remove, a)
			upsert = append(upsert, e)
		}
	}

	for k, a := range actualMap {
		e := expectedMap[k]

		if e == nil {
			remove = append(remove, a)
			continue
		}
	}

	if len(remove) != 0 {
		for _, r := range remove {
			glog.Infof("Skipping neigh delete: %v", routecontroller.AsJsonString(r))
			//glog.V(2).Infof("NETLINK: ip neigh delete %v", routecontroller.AsJsonString(r))
			//glog.V(2).Infof(" full neigh: %v", routecontroller.AsJsonString(r))
			//err := netlink.NeighDel(r)
			//if err != nil {
			//	return fmt.Errorf("error removing layer2 entry: %v", err)
			//}
		}
	}

	if len(upsert) != 0 {
		for _, r := range upsert {
			glog.V(2).Infof("NETLINK: ip neigh replace to %s lladdr %s dev %d", r.IP, r.HardwareAddr, r.LinkIndex)
			glog.V(2).Infof(" full neigh: %v", routecontroller.AsJsonString(r))
			err := netlink.NeighSet(r)
			if err != nil {
				return fmt.Errorf("error creating layer2 entry %v: %v", r, err)
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
