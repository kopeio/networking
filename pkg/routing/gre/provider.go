package gre

import (
	"fmt"
	"net"
	"syscall"

	"github.com/vishvananda/netlink"
	"k8s.io/klog/v2"
	"kope.io/networking/pkg/routing"
	"kope.io/networking/pkg/routing/netutil"
)

const tunnelTTL = 255 // TODO: What is the correct value for a GRE tunnel?

// Length of GRE tunnel name must be <= 15 characters
// Note that with hex we are _exactly_ 15 characters
const greLinkNamePrefix = "k8s-"
const greLinkNameFormat = "k8s-%02x-%02x-%02x-%02x"
const greLinkNameMaxLength = 15

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

func buildTunnelName(ip net.IP) string {
	ip4 := ip.To4()
	if ip4 == nil {
		klog.Warningf("cannot build tunnel name with non-ipv4 IP: %v", ip)
		return ""
	}
	name := fmt.Sprintf(greLinkNameFormat, ip4[0], ip4[1], ip4[2], ip4[3])
	if len(name) > greLinkNameMaxLength {
		klog.Warningf("generated link name that was longer than max: %q", name)
		return ""
	}
	return name
}

func (p *GreRoutingProvider) EnsureCIDRs(nodeMap *routing.NodeMap) error {
	if p.lastVersionApplied != 0 && nodeMap.IsVersion(p.lastVersionApplied) {
		return nil
	}

	me, allNodes, version := nodeMap.Snapshot()

	if me == nil {
		return fmt.Errorf("Cannot find local node")
	}

	var tunnels []netlink.Link

	for i := range allNodes {
		remote := &allNodes[i]

		if remote.Name == me.Name {
			continue
		}

		if remote.Address == nil {
			klog.Infof("Node %q did not have address; ignoring", remote.Name)
			continue
		}
		if remote.PodCIDR == nil {
			klog.Infof("Node %q did not have PodCIDR; ignoring", remote.Name)
			continue
		}

		tunnelName := buildTunnelName(remote.PodCIDR.IP)
		if tunnelName == "" {
			klog.Infof("Node %q has unacceptable PodCIDR %q", remote.Name, remote.PodCIDR.IP)
			continue
		}

		// ip tunnel add $name mode gre remote $remoteIP local $localIP`
		{
			t := &netlink.Gretun{
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

	tunnelMap, err := p.links.Ensure(tunnels, greLinkNamePrefix)
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
			klog.Infof("Node %q did not have address; ignoring", remote.Name)
			continue
		}
		if remote.PodCIDR == nil {
			klog.Infof("Node %q did not have PodCIDR; ignoring", remote.Name)
			continue
		}

		tunnelName := buildTunnelName(remote.PodCIDR.IP)
		if tunnelName == "" {
			klog.Infof("Node %q has unacceptable PodCIDR %q", remote.Name, remote.PodCIDR.IP)
			continue
		}

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
