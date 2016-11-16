package netutil

import (
	"fmt"

	"github.com/golang/glog"
	"github.com/vishvananda/netlink"
	"kope.io/networking/pkg/util"
)

func EnsureLinkAddresses(link netlink.Link, expected []*netlink.Addr) error {
	actualList, err := netlink.AddrList(link, netlink.FAMILY_ALL)
	if err != nil {
		return fmt.Errorf("error listing link addresses: %v", err)
	}

	actualMap := make(map[string]*netlink.Addr)
	for i := range actualList {
		a := &actualList[i]
		if a.IPNet == nil {
			glog.Errorf("ignoring unexpected address entry with no IP: %v", a)
			continue
		}
		k := a.IPNet.String()
		actualMap[k] = a
		glog.Infof("Actual address entry: %v", util.AsJsonString(a))
	}

	expectedMap := make(map[string]*netlink.Addr)

	for _, e := range expected {
		if e.IPNet == nil {
			glog.Errorf("ignoring unexpected address with no IP: %v", e)
			continue
		}
		k := e.IPNet.String()
		expectedMap[k] = e
		glog.Infof("Expected address entry: %v", util.AsJsonString(e))
	}

	var create []*netlink.Addr
	var remove []*netlink.Addr

	for k, e := range expectedMap {
		a := actualMap[k]

		if a == nil {
			create = append(create, e)
			continue
		}

		if !addrEqual(a, e) {
			glog.Infof("address change for %s:\n\t%s\n\t%s", k, util.AsJsonString(a), util.AsJsonString(e))
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
			glog.V(2).Infof("NETLINK: ip addr add %s dev link %s", r.IPNet, link.Attrs().Name)
			err := netlink.AddrAdd(link, r)
			if err != nil {
				return fmt.Errorf("error doing `ip addr add %s dev link %s`: %v", r.IPNet, link.Attrs().Name, err)
			}
		}
	}

	if len(remove) != 0 {
		for _, r := range remove {
			glog.V(2).Infof("NETLINK: ip addr del %s dev link %s", r.IPNet, link.Attrs().Name)
			err := netlink.AddrDel(link, r)
			if err != nil {
				return fmt.Errorf("error doing `ip addr del %s dev link %s address: %v", r.IPNet, link.Attrs().Name, err)
			}
		}
	}

	return nil
}

func addrEqual(a, e *netlink.Addr) bool {
	if a.Label != e.Label || a.Flags != e.Flags || a.Scope != e.Scope {
		return false
	}
	if !ipnetEqual(a.IPNet, e.IPNet) {
		return false
	}
	return true
}
