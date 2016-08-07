package greRouting

//
//import (
//	"fmt"
//	"net"
//
//	"github.com/golang/glog"
//	"github.com/kopeio/route-controller/pkg/routing"
//	"github.com/kopeio/route-controller/pkg/routing/netutil"
//	"github.com/vishvananda/netlink"
//	"os"
//	"syscall"
//	"github.com/kopeio/route-controller/pkg/routecontroller"
//	"k8s.io/kops/_vendor/github.com/docker/docker/daemon/links"
//	"strings"
//)
//
//type GreRoutingProvider struct {
//	lastVersionApplied uint64
//
//	routeTable         *netutil.RouteTable
//	linkConfig         *netutil.LinkConfig
//
//}
//
//var _ routing.Provider = &GreRoutingProvider{}
//
//func NewGreRoutingProvider() (*GreRoutingProvider, error) {
//	p := &GreRoutingProvider{
//	}
//
//	return p, nil
//}
//
//func (p *GreRoutingProvider) Close() error {
//	return nil
//}
//
//func (p *GreRoutingProvider) EnsureCIDRs(nodeMap *routing.NodeMap) error {
//	if p.lastVersionApplied != 0 && nodeMap.IsVersion(p.lastVersionApplied) {
//		return nil
//	}
//
//	me, allNodes, version := nodeMap.Snapshot()
//
//	if me == nil {
//		return fmt.Errorf("Cannot find local node")
//	}
//
//	if me.PodCIDR == nil {
//		return fmt.Errorf("No CIDR assigned to local node; cannot configure tunnels")
//	}
//
//	if me.Address == nil {
//		return fmt.Errorf("No Address assigned to local node; cannot configure tunnels")
//	}
//
//
//	var tunnels []*netlink.Tunnel
//	var routes []*netlink.Route
//
//	for i := range allNodes {
//		remote := &allNodes[i]
//
//		if remote.Name == me.Name {
//			continue
//		}
//
//		if remote.Address == nil {
//			glog.Infof("Node %q did not have address; ignoring", remote.Name)
//			continue
//		}
//		if remote.PodCIDR == nil {
//			glog.Infof("Node %q did not have PodCIDR; ignoring", remote.Name)
//			continue
//		}
//
//
//		name := "gre-" + strings.Replace(remote.PodCIDR.IP.String(), ".", "-", -1)
//
//		p.linkConfig.Ensure(name)
//
//		tunnel := &netlink.Tunnel{
//		}
//		tunnels = append(tunnels, tunnel)
//
//
//		route := &netlink.Route{
//			LinkIndex: links[name].Index,
//			Dst: remote.PodCIDR,
//		}
//		routes = append(routes, route)
//	}
//
//	err := p.greLinks.Ensure(tunnels)
//	if err != nil {
//		return fmt.Errorf("error applying neigh table: %v", err)
//	}
//
//	err = p.routeTable.Ensure(routes)
//	if err != nil {
//		return fmt.Errorf("error applying route table: %v", err)
//	}
//
//	p.lastVersionApplied = version
//
//	return nil
//}
