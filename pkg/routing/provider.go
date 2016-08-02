package routing

type Provider interface {
	EnsureCIDRs(nodeMap *NodeMap) error
}
