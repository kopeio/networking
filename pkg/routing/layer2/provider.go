package layer2

import (
	"fmt"

	"github.com/golang/glog"
	"github.com/kopeio/route-controller/pkg/routing"
	"github.com/kopeio/route-controller/pkg/routing/netutil"
	"github.com/vishvananda/netlink"
	"syscall"
)

type Layer2RoutingProvider struct {
	lastVersionApplied uint64

	routeTable     *netutil.RouteTable
	underlyingLink netlink.Link
}

var _ routing.Provider = &Layer2RoutingProvider{}

func NewLayer2RoutingProvider(deviceName string) (*Layer2RoutingProvider, error) {
	underlyingLink, err := netlink.LinkByName(deviceName)
	if err != nil {
		return nil, fmt.Errorf("error fetching target link %q: %v", deviceName, err)
	}
	if underlyingLink == nil {
		return nil, fmt.Errorf("target link not found %q", deviceName)
	}

	p := &Layer2RoutingProvider{
		routeTable:     &netutil.RouteTable{},
		underlyingLink: underlyingLink,
	}

	return p, nil
}

func (p *Layer2RoutingProvider) Close() error {
	return nil
}

func (p *Layer2RoutingProvider) EnsureCIDRs(nodeMap *routing.NodeMap) error {
	if p.lastVersionApplied != 0 && nodeMap.IsVersion(p.lastVersionApplied) {
		return nil
	}

	me, allNodes, version := nodeMap.Snapshot()

	if me == nil {
		return fmt.Errorf("Cannot find local node")
	}

	var routes []*netlink.Route

	underlyingLinkIndex := p.underlyingLink.Attrs().Index

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

		// ip route add $remoteCidr via $remoteIP
		{
			r := &netlink.Route{
				LinkIndex: underlyingLinkIndex,
				Dst:       remote.PodCIDR,
				Gw:        remote.Address,
				Protocol:  syscall.RTPROT_BOOT,
				Table:     syscall.RT_TABLE_MAIN,
				Type:      syscall.RTN_UNICAST,
			}
			routes = append(routes, r)
		}
	}

	err := p.routeTable.Ensure(nil, routes, false)
	if err != nil {
		return fmt.Errorf("error applying route table: %v", err)
	}

	p.lastVersionApplied = version

	return nil
}
