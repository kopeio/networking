package gre

import (
	"fmt"

	"github.com/golang/glog"
	"github.com/kopeio/route-controller/pkg/routing"
	"github.com/kopeio/route-controller/pkg/routing/netutil"
	"github.com/vishvananda/netlink"
	"net"
	"strings"
	"syscall"
)

const tunnelTTL = 255 // TODO: What is the correct value for a GRE tunnel?

type GreRoutingProvider struct {
	lastVersionApplied uint64

	routeTable *netutil.RouteTable
	links      *netutil.Links
}

var _ routing.Provider = &GreRoutingProvider{}

func NewGreRoutingProvider() (*GreRoutingProvider, error) {
	p := &GreRoutingProvider{
		routeTable: &netutil.RouteTable{},
		links:      &netutil.Links{},
	}

	return p, nil
}

func (p *GreRoutingProvider) Close() error {
	return nil
}

func (p *GreRoutingProvider) EnsureCIDRs(nodeMap *routing.NodeMap) error {
	if p.lastVersionApplied != 0 && nodeMap.IsVersion(p.lastVersionApplied) {
		return nil
	}

	me, allNodes, version := nodeMap.Snapshot()

	if me == nil {
		return fmt.Errorf("Cannot find local node")
	}

	prefix := "k8s-"

	var tunnels []netlink.Link

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

		tunnelName := prefix + strings.Replace(remote.PodCIDR.IP.String(), ".", "-", -1)

		// ip tunnel add $name mode gre remote $remoteIP local $localIP`
		{
			t := &netlink.Gre{
				LinkAttrs: netlink.LinkAttrs{
					Name: tunnelName,
				},
				Local:  me.Address,
				Remote: remote.Address,
				Ttl:    tunnelTTL,
			}
			tunnels = append(tunnels, t)
		}
	}

	tunnelMap, err := p.links.Ensure(tunnels, prefix)
	if err != nil {
		return fmt.Errorf("error configuring tunnels: %v", err)
	}

	// Make sure all links are up
	for _, l := range tunnelMap {
		if (l.Attrs().Flags & net.FlagUp) != 0 {
			continue
		}
		name := l.Attrs().Name
		// ip link set $name up
		err := netlink.LinkSetUp(l)
		if err != nil {
			return fmt.Errorf("error from `ip link set %s up`: %v", name, err)
		}
	}

	// TODO: Set MTU?

	var routes []*netlink.Route

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

		tunnelName := prefix + strings.Replace(remote.PodCIDR.IP.String(), ".", "-", -1)
		tunnel := tunnelMap[tunnelName]
		if tunnel == nil {
			return fmt.Errorf("tunnel not found after being created: %q", tunnelName)
		}

		// ip route add $remoteCidr dev $tunnel
		{
			r := &netlink.Route{
				LinkIndex: tunnel.Attrs().Index,
				Dst:       remote.PodCIDR,
				Protocol:  syscall.RTPROT_BOOT,
				Table:     syscall.RT_TABLE_MAIN,
				Type:      syscall.RTN_UNICAST,
			}
			routes = append(routes, r)
		}
	}

	err = p.routeTable.Ensure(nil, routes, false)
	if err != nil {
		return fmt.Errorf("error applying route table: %v", err)
	}

	p.lastVersionApplied = version

	return nil
}
