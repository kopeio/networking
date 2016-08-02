package routing

import (
	"time"

	"github.com/golang/glog"
)

// Controller updates the routing provider, if any changes have been made
type Controller struct {
	nodeMap  *NodeMap
	provider Provider
}

// NewController creates a routing.Controller
func NewController(nodeMap *NodeMap, provider Provider) (*Controller, error) {
	c := &Controller{
		nodeMap:  nodeMap,
		provider: provider,
	}

	return c, nil
}

// Run starts the NodeController.
func (c *Controller) Run() {
	glog.Infof("starting node controller")

	go c.runWatcher()

	glog.Infof("shutting down node controller")
}

func (c *Controller) runWatcher() {
	for {
		err := c.provider.EnsureCIDRs(c.nodeMap)

		if err != nil {
			glog.Warningf("Unexpected error in provider controller, will retry: %v", err)
			time.Sleep(10 * time.Second)
		} else {
			time.Sleep(1 * time.Second)
		}
	}
}
