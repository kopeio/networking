package layer2routing

import (
	"fmt"
	"github.com/golang/glog"
	"github.com/kopeio/route-controller/pkg/routecontroller"
	"github.com/kopeio/route-controller/pkg/routecontroller/routingproviders"
	"k8s.io/kubernetes/pkg/api"
	"net"
)

type Layer2RoutingProvider struct {
}

var _ routingproviders.RoutingProvider = &Layer2RoutingProvider{}

func NewLayer2RoutingProvider() (*Layer2RoutingProvider, error) {
	p := &Layer2RoutingProvider{}
	return p, nil
}

func (p *Layer2RoutingProvider) EnsureCIDRs(me *api.Node, allNodes []api.Node) error {
	meInternalIP := routecontroller.FindInternalIPAddress(me)
	if meInternalIP == "" {
		glog.Infof("self-node does not yet have internalIP; delaying configuration")
		return nil
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

		err = routes.EnsureRouteViaIP(remoteCIDR.String(), destIP)
		if err != nil {
			return err
		}

		// TODO: Delete any extra routes?
	}

	return nil
}
