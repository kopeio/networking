package routecontroller

import (
	"encoding/json"
	"fmt"
	"github.com/golang/glog"
	"k8s.io/kubernetes/pkg/api"
	"sort"
)

func FindInternalIPAddress(node *api.Node) string {
	var internalIPs []string
	for i := range node.Status.Addresses {
		address := &node.Status.Addresses[i]
		if address.Type == api.NodeInternalIP {
			internalIPs = append(internalIPs, address.Address)
		}
	}

	if len(internalIPs) == 0 {
		return ""
	}

	if len(internalIPs) != 1 {
		glog.Infof("arbitrarily choosing IP for node: %q", node.Name)
		sort.Strings(internalIPs) // At least choose consistently
	}

	internalIP := internalIPs[0]
	return internalIP
}

func BuildCIDRMap(me *api.Node, nodes []api.Node) map[string]string {
	cidrMap := make(map[string]string)

	meName := me.Name

	for j := range nodes {
		node := &nodes[j]

		name := node.Name
		if name == meName {
			continue
		}

		podCIDR := node.Spec.PodCIDR
		if podCIDR == "" {
			glog.Infof("skipping node with no CIDR: %q", name)
			continue
		}

		internalIP := FindInternalIPAddress(node)
		if internalIP == "" {
			glog.Infof("skipping node with no InternalIP Address: %q", name)
			continue
		}

		cidrMap[podCIDR] = internalIP
	}

	return cidrMap
}

func AsJsonString(o interface{}) string {
	b, err := json.Marshal(o)
	if err != nil {
		return fmt.Sprintf("error marshaling %T: %v", o, err)
	}
	return string(b)
}
