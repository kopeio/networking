package mockrouting

import (
	"fmt"
	"github.com/golang/glog"
	"github.com/kopeio/route-controller/pkg/routecontroller"
	"github.com/kopeio/route-controller/pkg/routecontroller/routingproviders"
	"k8s.io/kubernetes/pkg/api"
	"net"
	"strings"
)

type MockRoutingProvider struct {
}

var _ routingproviders.RoutingProvider = &MockRoutingProvider{}

func NewMockRoutingProvider() (*MockRoutingProvider, error) {
	p := &MockRoutingProvider{}
	return p, nil
}

func (p *MockRoutingProvider) EnsureCIDRs(me *api.Node, allNodes []api.Node) error {
	fmt.Printf("=============\n")
	fmt.Printf("EnsureCIDRs\n")

	meInternalIP := routecontroller.FindInternalIPAddress(me)
	if meInternalIP == "" {
		glog.Infof("self-node does not yet have internalIP; delaying configuration")
		return nil
	}

	cidrMap := routecontroller.BuildCIDRMap(me, allNodes)

	for remoteCIDRString, destIP := range cidrMap {
		fmt.Printf("\t%s\t%s\n", remoteCIDRString, destIP)

		_, remoteCIDR, err := net.ParseCIDR(remoteCIDRString)
		if err != nil {
			return fmt.Errorf("error parsing PodCidr %q: %v", remoteCIDRString, err)
		}

		// See e.g. http://lartc.org/howto/lartc.tunnel.gre.html
		name := "gre-" + strings.Replace(remoteCIDR.IP.String(), ".", "-", -1)
		fmt.Printf("\t\tip tunnel add %s mode gre remote %s local %s ttl 255\n", name, destIP, meInternalIP)
		fmt.Printf("\t\tip link set %s up\n", name)
		//fmt.Printf("\t\tip addr add %s dev %s\n", me, name)
		fmt.Printf("\t\tip route add %s dev %s\n", remoteCIDR.String(), name)

		//And when you want to remove the tunnel on router A:
		//ip link set netb down
		//ip tunnel del netb

	}
	fmt.Printf("=============\n")

	return nil
}
