package routing

import (
	"bytes"
	"net"
	"sort"
	"sync"

	v1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"
	"kope.io/networking/pkg/util"
)

type NodePredicate func(node *v1.Node) bool

type NodeMap struct {
	util.Stoppable
	mePredicate NodePredicate

	mutex   sync.Mutex
	ready   bool
	nodes   map[string]*NodeInfo
	version uint64
	me      *NodeInfo
}

func (m *NodeMap) IsVersion(version uint64) bool {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	return m.version == version
}

func (m *NodeMap) IsReady() bool {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	return m.ready
}

func (m *NodeMap) Snapshot() (*NodeInfo, []NodeInfo, uint64) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if !m.ready {
		return nil, nil, 0
	}

	nodes := make([]NodeInfo, 0, len(m.nodes))
	for _, node := range m.nodes {
		nodes = append(nodes, *node)
	}
	var me NodeInfo
	if m.me != nil {
		me = *m.me
	}
	return &me, nodes, m.version
}

func (m *NodeMap) MarkReady() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.ready = true
}

func (m *NodeMap) RemoveNode(node *v1.Node) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.removeNode(node.Name)
}

// removeNode removes the specified node; it assumes the lock is held
func (m *NodeMap) removeNode(nodeName string) {
	delete(m.nodes, nodeName)

	m.version++
}

func (m *NodeMap) UpdateNode(src *v1.Node) bool {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	return m.updateNode(src)
}

// updateNode updates the internal state for a particular node
// It assumes the mutex is held
func (m *NodeMap) updateNode(src *v1.Node) bool {
	changed := false
	name := src.Name

	node := m.nodes[name]
	if node == nil {
		node = &NodeInfo{Name: name}
		m.nodes[name] = node
		changed = true
	}
	if node.update(src) {
		changed = true
	}

	if m.me == nil {
		if m.mePredicate(src) {
			klog.Infof("identified self node: %q", src.Name)
			m.me = node
			changed = true
		}
	}

	if changed {
		klog.V(2).Infof("Node %q changed", name)
		m.version++
	}

	return changed
}

// ReplaceAllNodes takes a list of all nodes, and replaces the existing map with it
// Nodes not in the list will be removed
func (m *NodeMap) ReplaceAllNodes(nodes []v1.Node) bool {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	changed := false

	names := make(map[string]bool)
	for i := range nodes {
		node := &nodes[i]
		names[node.Name] = true

		if m.updateNode(node) {
			changed = true
		}
	}

	for k := range m.nodes {
		if !names[k] {
			m.removeNode(k)
			changed = true
		}
	}

	return changed
}

func NewNodeMap(mePredicate NodePredicate) *NodeMap {
	m := &NodeMap{
		nodes:       make(map[string]*NodeInfo),
		mePredicate: mePredicate,
	}
	return m
}

// NodeInfo contains the subset of the node information that we care about
type NodeInfo struct {
	Name             string
	Address          net.IP
	PodCIDR          *net.IPNet
	NetworkAvailable bool
}

func (n *NodeInfo) update(src *v1.Node) bool {
	changed := false

	name := src.Name

	cidr := src.Spec.PodCIDR
	if cidr == "" {
		klog.Infof("Node has no CIDR: %q", name)
		if n.PodCIDR != nil {
			changed = true
			n.PodCIDR = nil
		}
	} else {
		_, ipnet, err := net.ParseCIDR(cidr)
		if err != nil || ipnet == nil {
			klog.Warningf("Error parsing CIDR %q for node %q", cidr, name)
			if n.PodCIDR != nil {
				changed = true
				n.PodCIDR = nil
			}
		} else {
			if n.PodCIDR == nil || !ipnet.IP.Equal(n.PodCIDR.IP) || !bytes.Equal(n.PodCIDR.Mask, ipnet.Mask) {
				n.PodCIDR = ipnet
				changed = true
			}
		}
	}

	var internalIPs []string
	for i := range src.Status.Addresses {
		address := &src.Status.Addresses[i]
		if address.Type == v1.NodeInternalIP {
			internalIPs = append(internalIPs, address.Address)
		}
	}

	if len(internalIPs) == 0 {
		if n.Address != nil {
			n.Address = nil
			changed = true
		}
	} else {
		if len(internalIPs) != 1 {
			klog.Infof("arbitrarily choosing IP for node: %q", name)
			sort.Strings(internalIPs) // At least choose consistently
		}

		internalIP := internalIPs[0]
		a := net.ParseIP(internalIP)
		if a == nil {
			klog.Warningf("Unable to parse node address %q", internalIP)
			if n.Address != nil {
				n.Address = nil
				changed = true
			}
		} else if !n.Address.Equal(a) {
			n.Address = a
			changed = true
		}
	}

	{
		networkAvailable := true
		for _, condition := range src.Status.Conditions {
			if condition.Type == v1.NodeNetworkUnavailable {
				switch condition.Status {
				case v1.ConditionFalse:
					// Double negative: Not unavailable
					networkAvailable = true
				case v1.ConditionTrue:
					// It is true that it is unavailable => available=false
					networkAvailable = false
				case v1.ConditionUnknown:
					klog.V(2).Infof("NodeNetworkAvailable status was ConditionUnknown - assuming available")
				default:
					klog.Warningf("NodeNetworkAvailable status was %q - assuming available", condition.Status)
				}
			}
		}
		if networkAvailable != n.NetworkAvailable {
			n.NetworkAvailable = networkAvailable
			changed = true
		}
	}

	return changed
}
