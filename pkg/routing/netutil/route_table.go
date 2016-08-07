package netutil

import (
	"fmt"

	"github.com/golang/glog"
	"github.com/kopeio/route-controller/pkg/routecontroller"
	"github.com/vishvananda/netlink"
)

type RouteTable struct {
	link netlink.Link
}

func (t *RouteTable) Ensure(expected []*netlink.Route) error {
	glog.V(2).Infof("NETLINK: ip route show")
	actualList, err := netlink.RouteList(t.link, netlink.FAMILY_ALL)
	if err != nil {
		return fmt.Errorf("error doing `ip route show`: %v", err)
	}

	glog.Warningf("TODO: using strings as ipnet key is inefficient")
	actualMap := make(map[string]*netlink.Route)
	for i := range actualList {
		a := &actualList[i]
		if a.Dst == nil {
			glog.Errorf("ignoring unexpected route with no dst: %v", a)
			continue
		}
		k := a.Dst.String()
		actualMap[k] = a
		glog.Infof("Actual route: %v", routecontroller.AsJsonString(a))
	}

	expectedMap := make(map[string]*netlink.Route)

	var create []*netlink.Route
	var remove []*netlink.Route

	for _, e := range expected {
		if e.Dst == nil {
			glog.Errorf("ignoring unexpected route with no dst: %v", e)
			continue
		}
		k := e.Dst.String()
		expectedMap[k] = e
		glog.Infof("Expected route: %v", routecontroller.AsJsonString(e))

		// Note that we process expected in order
		// TODO: I guess we could sort via dependencies?
		a := actualMap[k]

		if a == nil {
			create = append(create, e)
			continue
		}

		if !routeEqual(a, e) {
			glog.Infof("State change for %s:\n\ta: %s\n\te: %s", k, routecontroller.AsJsonString(a), routecontroller.AsJsonString(e))
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

	if len(remove) != 0 {
		for _, r := range remove {
			// TODO: Remove if mismatch?
			glog.Infof("Not removing route: %v", routecontroller.AsJsonString(r))
			//	glog.Infof("NETLINK: ip route del %v", routecontroller.AsJsonString(r))
			//	err := netlink.RouteDel(r)
			//	if err != nil {
			//		return fmt.Errorf("error removing route: %v", err)
			//	}
		}
	}

	if len(create) != 0 {
		for _, r := range create {
			glog.Infof("NETLINK: ip route add %s via %s", r.Dst, r.Gw)
			glog.Infof(" full route object: %v", routecontroller.AsJsonString(r))
			err := netlink.RouteAdd(r)
			if err != nil {
				return fmt.Errorf("error creating route %v: %v", r, err)
			}
		}
	}

	return nil
}

func routeEqual(a, e *netlink.Route) bool {
	if a.LinkIndex != e.LinkIndex || a.ILinkIndex != e.ILinkIndex || a.Scope != e.Scope || a.Protocol != e.Protocol || a.Priority != e.Priority || a.Table != e.Table || a.Type != e.Type || a.Tos != e.Tos || a.Flags != e.Flags {
		return false
	}
	if !ipnetEqual(a.Dst, e.Dst) {
		return false
	}
	if !ipEqual(a.Src, e.Src) {
		return false
	}
	if !ipEqual(a.Gw, e.Gw) {
		return false
	}
	if len(a.MultiPath) != len(e.MultiPath) {
		return false
	}
	for i := range a.MultiPath {
		if !nexthopInfoEqual(a.MultiPath[i], e.MultiPath[i]) {
			return false
		}
	}
	return true
}

func nexthopInfoEqual(a, e *netlink.NexthopInfo) bool {
	if a.LinkIndex != e.LinkIndex || a.Hops != e.Hops {
		return false
	}
	if !ipEqual(a.Gw, e.Gw) {
		return false
	}
	return true
}
