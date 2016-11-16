package netutil

import (
	"fmt"

	"github.com/golang/glog"
	"github.com/vishvananda/netlink"
	"kope.io/networking/pkg/util"
	"strings"
)

type Links struct {
}

// Creates links to match expected; removing any links that match prefix but are not expected
// Returns the state of links matching expected
func (t *Links) Ensure(expected []netlink.Link, prefix string) (map[string]netlink.Link, error) {
	glog.V(2).Infof("NETLINK: ip links show")

	retMap := make(map[string]netlink.Link)

	actualMap := make(map[string]netlink.Link)
	{
		allLinks, err := netlink.LinkList()
		if err != nil {
			return nil, fmt.Errorf("error doing `ip route show`: %v", err)
		}
		for _, l := range allLinks {
			name := l.Attrs().Name
			if !strings.HasPrefix(name, prefix) {
				continue
			}
			actualMap[name] = l
			glog.V(4).Infof("Actual Link: %v", util.AsJsonString(l))
		}
	}

	expectedMap := make(map[string]netlink.Link)

	for _, e := range expected {
		glog.V(4).Infof("Expected route: %v", util.AsJsonString(e))

		name := e.Attrs().Name
		expectedMap[name] = e
	}

	var create []netlink.Link
	var remove []netlink.Link

	for k, e := range expectedMap {
		a := actualMap[k]

		if a == nil {
			create = append(create, e)
			continue
		}

		if !linkEqual(a, e) {
			glog.Warningf("NOT IMPLEMENTED change for link %s:\n\ta: %s\n\te: %s", k, util.AsJsonString(a), util.AsJsonString(e))
			//remove = append(remove, a)
			//create = append(create, e)
		}
		retMap[k] = a
	}

	for k, a := range actualMap {
		e := expectedMap[k]

		if e == nil {
			remove = append(remove, a)
			continue
		}
	}

	if len(remove) != 0 {
		for _, l := range remove {
			glog.Infof("NETLINK: ip link del %s", l.Attrs().Name)
			err := netlink.LinkDel(l)
			if err != nil {
				return nil, fmt.Errorf("error removing link: %v", err)
			}
		}
	}

	if len(create) != 0 {
		for _, l := range create {
			glog.Infof("NETLINK: ip link create %s", l.Attrs().Name)
			glog.V(2).Infof(" full link object: %v", util.AsJsonString(l))
			err := netlink.LinkAdd(l)
			if err != nil {
				return nil, fmt.Errorf("error creating link %v: %v", l, err)
			}

			retMap[l.Attrs().Name] = l
		}
	}

	return retMap, nil
}

func linkEqual(a, e netlink.Link) bool {
	if a.Type() != e.Type() {
		return false
	}
	aa := a.Attrs()
	ea := e.Attrs()
	if aa.Name != ea.Name {
		return false
	}
	switch a.Type() {
	//case "gre":
	//	return greLinkEqual(a.(*netlink.Gre), e.(*netlink.Gre))
	default:
		glog.Warningf("Unhandled type %q", a.Type())
		return true
	}
}

//func greLinkEqual(a, e *netlink.Gre) bool {
//	if !ipEqual(a.Local, e.Local) {
//		return false
//	}
//	if !ipEqual(a.Remote, e.Remote) {
//		return false
//	}
//	if a.Ttl != e.Ttl {
//		return false
//	}
//	if a.IFlags != e.IFlags {
//		return false
//	}
//	if a.OFlags != e.OFlags {
//		return false
//	}
//	if a.PMtuDisc != e.PMtuDisc {
//		return false
//	}
//	if a.Tos != e.Tos {
//		return false
//	}
//	if a.Link != e.Link {
//		return false
//	}
//	return true
//}
