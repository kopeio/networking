package routingproviders

import "k8s.io/kubernetes/pkg/api"

type RoutingProvider interface {
	EnsureCIDRs(me *api.Node, allNodes []api.Node) error
}
