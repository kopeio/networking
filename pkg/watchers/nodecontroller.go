package watchers

import (
	"fmt"
	"time"

	"github.com/golang/glog"

	"github.com/kopeio/route-controller/pkg/routing"
	"github.com/kopeio/route-controller/pkg/util"
	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/api/v1"
	client "k8s.io/kubernetes/pkg/client/clientset_generated/release_1_3/typed/core/v1"
	"k8s.io/kubernetes/pkg/fields"
	"k8s.io/kubernetes/pkg/labels"
	"k8s.io/kubernetes/pkg/watch"
)

// NodeController watches for nodes
type NodeController struct {
	util.Stoppable
	kubeClient *client.CoreClient
	nodeMap    *routing.NodeMap
}

// newNodeController creates a nodeController
func NewNodeController(kubeClient *client.CoreClient, nodeMap *routing.NodeMap) (*NodeController, error) {
	c := &NodeController{
		kubeClient: kubeClient,
		nodeMap:    nodeMap,
	}

	return c, nil
}

// Run starts the NodeController.
func (c *NodeController) Run() {
	glog.Infof("starting node controller")

	stopCh := c.StopChannel()
	go c.runWatcher(stopCh)

	<-stopCh
	glog.Infof("shutting down node controller")
}

func (c *NodeController) runWatcher(stopCh <-chan struct{}) {
	runOnce := func() (bool, error) {
		var listOpts api.ListOptions

		// We need to watch all the nodes
		listOpts.LabelSelector = labels.Everything()
		listOpts.FieldSelector = fields.Everything()

		nodeList, err := c.kubeClient.Nodes().List(listOpts)
		if err != nil {
			return false, fmt.Errorf("error listing nodes: %v", err)
		}
		for i := range nodeList.Items {
			node := &nodeList.Items[i]
			//glog.V(1).Infof("node list: %v", node.Name)
			c.nodeMap.UpdateNode(node)
		}
		c.nodeMap.MarkReady()

		listOpts.LabelSelector = labels.Everything()
		listOpts.FieldSelector = fields.Everything()

		listOpts.Watch = true
		listOpts.ResourceVersion = nodeList.ResourceVersion
		watcher, err := c.kubeClient.Nodes().Watch(listOpts)
		if err != nil {
			return false, fmt.Errorf("error watching nodes: %v", err)
		}
		ch := watcher.ResultChan()
		for {
			select {
			case <-stopCh:
				glog.Infof("Got stop signal")
				return true, nil
			case event, ok := <-ch:
				if !ok {
					glog.Infof("node watch channel closed")
					return false, nil
				}

				node := event.Object.(*v1.Node)
				glog.V(4).Infof("node changed: %s %v", event.Type, node.Name)

				switch event.Type {
				case watch.Added, watch.Modified:
					c.nodeMap.UpdateNode(node)

				case watch.Deleted:
					c.nodeMap.RemoveNode(node)
				}
			}
		}
	}

	for {
		stop, err := runOnce()
		if stop {
			return
		}

		if err != nil {
			glog.Warningf("Unexpected error in event watch, will retry: %v", err)
			time.Sleep(10 * time.Second)
		}
	}
}
