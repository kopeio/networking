package vxlanrouting

import (
	"fmt"
	"net"
	"syscall"

	"github.com/golang/glog"
	"github.com/kopeio/route-controller/pkg/routing"
	"github.com/kopeio/route-controller/pkg/routing/netutil"
	"github.com/vishvananda/netlink"
)

type VxlanRoutingProvider struct {
	vxlanID      int
	vtepIndex    int
	vxlanSrcAddr net.IP
	vxlanPort    int

	routeTable *netutil.RouteTable
	neighTable *netutil.NeighTable

	lastVersionApplied uint64
}

var _ routing.Provider = &VxlanRoutingProvider{}

func NewVxlanRoutingProvider() (*VxlanRoutingProvider, error) {
	p := &VxlanRoutingProvider{
		vxlanID:      1,
		vtepIndex:    0,
		vxlanSrcAddr: nil, // TODO?
		vxlanPort:    4789,
	}

	return p, nil
}

func (p *VxlanRoutingProvider) Close() error {
	return nil
}

func (p *VxlanRoutingProvider) EnsureLink() (netlink.Link, error) {

	// VXLAN tunnels with layer-2 routing seems to work... tried manually with:

	// on each host:
	// ip link add vxlan1 type vxlan id 1 dev eth0 dstport 4789
	// ip link set vxlan1 address 54:8:64:40:02:01
	// ip addr add 10.244.0.0/32 dev vxlan1
	// ip link set up vxlan1

	// and then on each host, for each peer:
	// bridge fdb add to 54:08:64:40:01:01 dst 172.20.27.211 dev vxlan1
	// arp -i vxlan1 -s 10.244.100.1 54:08:64:40:01:01
	// ip route add 10.244.100.0/32 dev vxlan1
	// ip route add 10.244.100.0/24 via 10.244.100.0

	name := fmt.Sprintf("vxlan%d", p.vxlanID)

	var macAddress net.HardwareAddr

	link := &netlink.Vxlan{
		LinkAttrs: netlink.LinkAttrs{
			Name:         name,
			MTU:          mtu,
			HardwareAddr: macAddress,
		},
		VxlanId:      p.vxlanID,
		VtepDevIndex: p.vtepIndex,
		SrcAddr:      p.vxlanSrcAddr,
		Port:         p.vxlanPort,
	}

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

	link, err := p.EnsureLink()
	if err != nil {
		return err
	}

	linkIndex := link.Attrs().Index

	var neighs []*netlink.Neigh
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

		remoteMAC := mapToMAC(remote.Address)

		// bridge fdb add to 54:08:64:40:01:01 dst 172.20.27.211 dev vxlan1
		{
			n := &netlink.Neigh{
				LinkIndex:    linkIndex,
				Family:       syscall.AF_BRIDGE,
				State:        netlink.NUD_PERMANENT,
				Flags:        netlink.NTF_SELF,
				IP:           remote.Address,
				HardwareAddr: remoteMAC,
			}

			neighs = append(neighs, n)
		}
		// arp -i vxlan1 -s 10.244.100.1 54:08:64:40:01:01
		//{
		//	n := &netlink.Neigh{
		//		LinkIndex: linkIndex,
		//		IP:
		//	}
		//}

		// ip route add 10.244.100.0/32 dev vxlan1
		{
			r := &netlink.Route{
				LinkIndex: linkIndex,
				Dst: &net.IPNet{
					IP:   remote.PodCIDR.IP,
					Mask: net.IPv4Mask(255, 255, 255, 255),
				},
			}
			routes = append(routes, r)
		}

		// ip route add 10.244.100.0/24 via 10.244.100.0
		{
			r := &netlink.Route{
				Dst: remote.PodCIDR,
				Gw:  remote.PodCIDR.IP,
			}
			routes = append(routes, r)
		}
	}

	err = p.neighTable.Ensure(neighs)
	if err != nil {
		return fmt.Errorf("error applying neigh table: %v", err)
	}

	err = p.routeTable.Ensure(routes)
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
	if ip4 != nil {
		glog.Fatalf("unexpected non-ipv4 IP: %v", ip)
	}
	hw[2] = ip4[0]
	hw[3] = ip4[1]
	hw[4] = ip4[2]
	hw[5] = ip4[3]

	return net.HardwareAddr(hw)
}
