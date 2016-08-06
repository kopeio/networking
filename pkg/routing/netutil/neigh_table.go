package netutil

import (
	"bytes"
	"fmt"

	"github.com/golang/glog"
	"github.com/kopeio/route-controller/pkg/routecontroller"
	"github.com/vishvananda/netlink"
)

type NeighTable struct {
	link netlink.Link
}

func NewNeighTable(link netlink.Link) (*NeighTable, error) {
	p := &NeighTable{
		link: link,
	}

	return p, nil
}

func (c *NeighTable) Ensure(expected []*netlink.Neigh) error {
	actualList, err := netlink.NeighList(c.link.Attrs().Index, netlink.FAMILY_ALL)
	if err != nil {
		return fmt.Errorf("error listing layer2 config: %v", err)
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

	var create []*netlink.Neigh
	var remove []*netlink.Neigh

	for k, e := range expectedMap {
		a := actualMap[k]

		if a == nil {
			create = append(create, e)
			continue
		}

		if !neighEqual(a, e) {
			glog.Infof("State change for %s:\n\t%s\n\t%s", k, routecontroller.AsJsonString(a), routecontroller.AsJsonString(e))
			remove = append(remove, a)
			create = append(create, e)
		}
	}

	for k, a := range actualMap {
		e := expectedMap[k]

		if e == nil {
			remove = append(remove, a)
			continue
		}
	}

	if len(create) != 0 {
		for _, r := range create {
			glog.Infof("creating route %v", routecontroller.AsJsonString(r))
			err := netlink.NeighAdd(r)
			if err != nil {
				glog.Warningf("error creating layer2 entry %v: %v", r, err)
			}
		}
	}

	if len(remove) != 0 {
		for _, r := range remove {
			glog.Infof("removing route %v", routecontroller.AsJsonString(r))
			err := netlink.NeighDel(r)
			if err != nil {
				glog.Warningf("error removing layer2 entry: %v", err)
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
