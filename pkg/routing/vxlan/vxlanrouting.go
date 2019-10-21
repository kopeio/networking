package vxlan

import (
	"fmt"
	"io/ioutil"
	"net"
	"syscall"

	"github.com/vishvananda/netlink"
	"k8s.io/klog"
	"kope.io/networking/pkg/routing"
	"kope.io/networking/pkg/routing/netutil"
)

type VxlanRoutingProvider struct {
	overlayCIDR *net.IPNet

	monitor *NetlinkMonitor

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

func listenArp(link netlink.Link) error {
	sysctlPath := "/proc/sys/net/ipv4/neigh/" + link.Attrs().Name + "/app_solicit"
	err := ioutil.WriteFile(sysctlPath, []byte("3"), 0666)
	if err != nil {
		return fmt.Errorf("error writing setting sysctl for ARP events: %v", err)
	}
	return nil
}

func (p *VxlanRoutingProvider) EnsureLink(me net.IP, cidr *net.IPNet) (netlink.Link, error) {
	name := fmt.Sprintf("vxlan%d", p.vxlanID)

	macAddress := mapToMAC(cidr.IP)

	// TODO: Check if exists first?
	link := &netlink.Vxlan{
		LinkAttrs: netlink.LinkAttrs{
			Name:         name,
			MTU:          p.mtu,
			HardwareAddr: macAddress,
		},
		VxlanId:      p.vxlanID,
		VtepDevIndex: p.vtepIndex,
		SrcAddr:      me,
		Port:         p.vxlanPort,
	}

	err := netlink.LinkAdd(link)
	if err != nil {
		// TODO: Reconfigure link?
		klog.Warningf("Unable to create link; will reuse existing link: %v", err)
	} else {
		klog.V(2).Infof("NETLINK: ip link set %s address %s", link.Name, macAddress)
		err = netlink.LinkSetHardwareAddr(link, macAddress)
		if err != nil {
			return nil, fmt.Errorf("failed to `ip link set %s address %s`: %v", link.Name, macAddress, err)
		}
	}

	// We need the link index
	found, err := netlink.LinkByName(link.Name)
	if err != nil {
		return nil, fmt.Errorf("error retrieving link %q: %v", link.Name, err)
	}

	if found.Attrs().MTU != p.mtu {
		klog.V(2).Infof("NETLINK: ip link set %s mtu %d", link.Name, p.mtu)
		err = netlink.LinkSetMTU(link, p.mtu)
		if err != nil {
			return nil, fmt.Errorf("failed to `ip link set %s mtu %d`: %v", link.Name, p.mtu, err)
		}
	}

	// ip addr add $cidr dev $link
	linkCIDR := &net.IPNet{
		IP:   cidr.IP,
		Mask: p.overlayCIDR.Mask,
	}
	err = netutil.EnsureLinkAddresses(link, []*netlink.Addr{
		{
			IPNet: linkCIDR,
			Label: link.Name,
			Flags: 128, // ???
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to `ip addr add %s dev link %s`: %v", cidr, link.Name, err)
	}

	// ip link set $link up
	klog.V(2).Infof("NETLINK: ip link set %s up", link.Name)
	err = netlink.LinkSetUp(link)
	if err != nil {
		return nil, fmt.Errorf("failed to `ip link set %s up`: %s", link.Name, err)
	}

	return found, nil
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

		err = listenArp(link)
		if err != nil {
			return err
		}

		monitor, err := NewNetlinkMonitor(link.Attrs().Index)
		if err != nil {
			return err
		}
		err = monitor.Start()
		if err != nil {
			return err
		}
		p.monitor = monitor
	}

	linkIndex := p.link.Attrs().Index

	var neighs []*netlink.Neigh
	var routes []*netlink.Route

	// route whole overlay CIDR to vxlan
	{
		r := &netlink.Route{
			LinkIndex: linkIndex,
			Dst:       p.overlayCIDR,
			Protocol:  syscall.RTPROT_BOOT,
			Table:     syscall.RT_TABLE_MAIN,
			Type:      syscall.RTN_UNICAST,
		}
		routes = append(routes, r)
	}

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

		// bridge fdb add to <remote-mac> dst <remote-ip> dev vxlan1
		{
			n := &netlink.Neigh{
				LinkIndex:    linkIndex,
				State:        netlink.NUD_PERMANENT,
				Family:       syscall.AF_BRIDGE,
				Flags:        netlink.NTF_SELF,
				IP:           remote.Address,
				HardwareAddr: remoteMAC,
			}

			neighs = append(neighs, n)
		}
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
