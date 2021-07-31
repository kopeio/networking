package vxlan2

import (
	"bytes"
	"fmt"
	"net"
	"syscall"

	"github.com/vishvananda/netlink"
	"k8s.io/klog/v2"
	"kope.io/networking/pkg/routing"
	"kope.io/networking/pkg/routing/netutil"
)

type VxlanRoutingProvider struct {
	overlayCIDR *net.IPNet

	vxlanID   int
	vtepIndex int
	vxlanPort int

	mtu int

	link       *netlink.Vxlan
	routeTable *netutil.RouteTable
	neighTable *netutil.NeighTable

	lastVersionApplied uint64
}

var _ routing.Provider = &VxlanRoutingProvider{}

func NewVxlanRoutingProvider(overlayCIDR *net.IPNet, deviceName string) (*VxlanRoutingProvider, error) {
	underlyingLink, err := netlink.LinkByName(deviceName)
	if err != nil {
		return nil, fmt.Errorf("error fetching target link %q: %v", deviceName, err)
	}
	if underlyingLink == nil {
		return nil, fmt.Errorf("target link not found %q", deviceName)
	}

	mtu := underlyingLink.Attrs().MTU - 100
	klog.Warningf("MTU hard-coded to underlying interface %s MTU - 100 = %d", deviceName, mtu)

	p := &VxlanRoutingProvider{
		overlayCIDR: overlayCIDR,

		vxlanID:   1,
		vtepIndex: 0,
		vxlanPort: 4789,

		mtu: mtu,
	}

	return p, nil
}

func (p *VxlanRoutingProvider) Close() error {
	return nil
}

func (p *VxlanRoutingProvider) EnsureLink(me net.IP, cidr *net.IPNet) (netlink.Link, error) {
	name := fmt.Sprintf("vxlan%d", p.vxlanID)

	macAddress := mapToMAC(cidr.IP)

	// TODO: Check if exists first?
	expected := &netlink.Vxlan{
		LinkAttrs: netlink.LinkAttrs{
			Name:         name,
			MTU:          p.mtu,
			HardwareAddr: macAddress,
		},
		Learning:     false,
		VxlanId:      p.vxlanID,
		VtepDevIndex: p.vtepIndex,
		SrcAddr:      me,
		Port:         p.vxlanPort,
	}

	actual, err := netlink.LinkByName(name)
	if err != nil {
		if _, ok := err.(netlink.LinkNotFoundError); ok {
			klog.V(2).Infof("link %q not found: %v", name, err)
			actual = nil
		} else {
			return nil, fmt.Errorf("error fetching link %q: %w", name, err)
		}
	}

	if actual == nil {
		if err := netlink.LinkAdd(expected); err != nil {
			klog.Infof("failed to create link %#v", expected)
			return nil, fmt.Errorf("unable to create link %q: %w", name, err)
		}

		// We need the link index
		found, err := netlink.LinkByName(expected.Name)
		if err != nil {
			return nil, fmt.Errorf("error retrieving link %q after creation: %w", expected.Name, err)
		}
		actual = found
	} else {
		// TODO: Check for differences and reconfigure?
		klog.V(2).Infof("existing link is %#v", actual)
		klog.Infof("reusing existing link %q", name)
	}

	if !bytes.Equal(actual.Attrs().HardwareAddr, macAddress) {
		klog.V(2).Infof("NETLINK: ip link set %s address %s", name, macAddress)
		if err := netlink.LinkSetHardwareAddr(actual, macAddress); err != nil {
			return nil, fmt.Errorf("failed to `ip link set %s address %s`: %w", name, macAddress, err)
		}
	}

	if actual.Attrs().MTU != p.mtu {
		klog.V(2).Infof("NETLINK: ip link set %s mtu %d", name, p.mtu)
		err = netlink.LinkSetMTU(actual, p.mtu)
		if err != nil {
			return nil, fmt.Errorf("failed to `ip link set %s mtu %d`: %w", name, p.mtu, err)
		}
	}

	// ip addr add $cidr dev $link
	linkCIDR := &net.IPNet{
		IP:   cidr.IP,
		Mask: net.CIDRMask(32, 32),
	}
	err = netutil.EnsureLinkAddresses(actual, []*netlink.Addr{
		{
			IPNet: linkCIDR,
			Label: name,
			Flags: 128, // ???
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to set address %s on link %s`: %w", cidr, name, err)
	}

	// ip link set $link up
	if actual.Attrs().Flags&net.FlagUp == 0 {
		klog.V(2).Infof("NETLINK: ip link set %s up", name)
		if err := netlink.LinkSetUp(actual); err != nil {
			return nil, fmt.Errorf("failed to `ip link set %s up`: %w", name, err)
		}
	}

	return actual, nil
}

func (p *VxlanRoutingProvider) EnsureCIDRs(nodeMap *routing.NodeMap) error {
	if p.lastVersionApplied != 0 && nodeMap.IsVersion(p.lastVersionApplied) {
		return nil
	}

	me, allNodes, version := nodeMap.Snapshot()

	if me == nil {
		return fmt.Errorf("Cannot find local node")
	}

	if me.PodCIDR == nil {
		return fmt.Errorf("No CIDR assigned to local node; cannot configure tunnels")
	}

	if me.Address == nil {
		return fmt.Errorf("No Address assigned to local node; cannot configure tunnels")
	}

	if p.link == nil {
		p.routeTable = &netutil.RouteTable{}

		link, err := p.EnsureLink(me.Address, me.PodCIDR)
		if err != nil {
			return err
		}
		p.link = link.(*netlink.Vxlan)
		p.neighTable, err = netutil.NewNeighTable(link.Attrs().Name, link.Attrs().Index)
		if err != nil {
			return err
		}
	}

	linkIndex := p.link.Attrs().Index

	var neighs []*netlink.Neigh
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

		remoteMAC := mapToMAC(remote.PodCIDR.IP)

		arp := &netlink.Neigh{
			LinkIndex:    linkIndex,
			Family:       netlink.FAMILY_V4,
			State:        netlink.NUD_PERMANENT,
			Type:         syscall.RTN_UNICAST,
			IP:           remote.PodCIDR.IP,
			HardwareAddr: remoteMAC,
		}
		neighs = append(neighs, arp)
		fdb := &netlink.Neigh{
			LinkIndex:    linkIndex,
			State:        netlink.NUD_PERMANENT,
			Family:       syscall.AF_BRIDGE,
			Flags:        netlink.NTF_SELF,
			IP:           remote.Address,
			HardwareAddr: remoteMAC,
		}
		neighs = append(neighs, fdb)

		route := &netlink.Route{
			LinkIndex: linkIndex,
			Scope:     netlink.SCOPE_UNIVERSE,
			Dst:       remote.PodCIDR,
			Gw:        remote.PodCIDR.IP,
			Protocol:  syscall.RTPROT_BOOT,
			Table:     syscall.RT_TABLE_MAIN,
			Type:      syscall.RTN_UNICAST,
		}
		route.SetFlag(syscall.RTNH_F_ONLINK)
		routes = append(routes, route)
	}

	err := p.neighTable.Ensure(p.link, neighs)
	if err != nil {
		return fmt.Errorf("error applying neigh table: %v", err)
	}

	// We are specifying a link scope, so we do delete routes
	deleteExtraRoutes := true
	err = p.routeTable.Ensure(p.link, routes, deleteExtraRoutes)
	if err != nil {
		return fmt.Errorf("error applying route table: %v", err)
	}

	p.lastVersionApplied = version

	return nil
}

func mapToMAC(ip net.IP) net.HardwareAddr {
	hw := make([]byte, 6, 6)
	// TODO: This is the "documentation" range - safe?
	hw[0] = 0x00
	hw[1] = 0x53
	ip4 := ip.To4()
	if ip4 == nil {
		klog.Fatalf("unexpected non-ipv4 IP: %v", ip)
	}
	hw[2] = ip4[0]
	hw[3] = ip4[1]
	hw[4] = ip4[2]
	hw[5] = ip4[3]

	mac := net.HardwareAddr(hw)
	klog.V(4).Infof("mapped ip %s -> mac %s", ip, mac)
	return mac
}
