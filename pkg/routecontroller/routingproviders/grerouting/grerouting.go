package grerouting

import (
	"fmt"
	"github.com/golang/glog"
	"github.com/kopeio/route-controller/pkg/routecontroller"
	"github.com/kopeio/route-controller/pkg/routecontroller/routingproviders"
	"k8s.io/kubernetes/pkg/api"
	"net"
	"strings"
)

type GreRoutingProvider struct {
}

var _ routingproviders.RoutingProvider = &GreRoutingProvider{}

func NewGreRoutingProvider() (*GreRoutingProvider, error) {
	p := &GreRoutingProvider{}
	return p, nil
}

func (p *GreRoutingProvider) EnsureCIDRs(me *api.Node, allNodes []api.Node) error {
	meInternalIP := routecontroller.FindInternalIPAddress(me)
	if meInternalIP == "" {
		glog.Infof("self-node does not yet have internalIP; delaying configuration")
		return nil
	}

	links, err := routecontroller.QueryIPLinks()
	if err != nil {
		return err
	}

	routes, err := routecontroller.QueryIPRoutes()
	if err != nil {
		return err
	}

	cidrMap := routecontroller.BuildCIDRMap(me, allNodes)

	for remoteCIDRString, destIP := range cidrMap {
		_, remoteCIDR, err := net.ParseCIDR(remoteCIDRString)
		if err != nil {
			return fmt.Errorf("error parsing PodCidr %q: %v", remoteCIDRString, err)
		}

		name := "gre-" + strings.Replace(remoteCIDR.IP.String(), ".", "-", -1)

		err = links.EnsureGRETunnel(name, destIP, meInternalIP)
		if err != nil {
			return err
		}

		err = routes.EnsureRoute(remoteCIDR.String(), name)
		if err != nil {
			return err
		}

		// TODO: Delete any extra tunnels
	}

	return nil
}
