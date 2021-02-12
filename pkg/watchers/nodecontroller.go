package watchers

import (
	"context"
	"fmt"
	"time"

	v1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
	"kope.io/networking/pkg/routing"
	"kope.io/networking/pkg/util"
)

// NodeController watches for nodes
type NodeController struct {
	util.Stoppable
	kubeClient kubernetes.Interface
	nodeMap    *routing.NodeMap
}

// newNodeController creates a nodeController
func NewNodeController(kubeClient kubernetes.Interface, nodeMap *routing.NodeMap) (*NodeController, error) {
	c := &NodeController{
		kubeClient: kubeClient,
		nodeMap:    nodeMap,
	}

	return c, nil
}

// Run starts the NodeController.
func (c *NodeController) Run(ctx context.Context) {
	klog.Infof("starting node controller")

	stopCh := c.StopChannel()
	go c.runWatcher(ctx, stopCh)

	<-stopCh
	klog.Infof("shutting down node controller")
}

func (c *NodeController) runWatcher(ctx context.Context, stopCh <-chan struct{}) {
	runOnce := func() (bool, error) {
		var listOpts meta_v1.ListOptions

		// We need to watch all the nodes
		//listOpts.LabelSelector = labels.Everything()
		//listOpts.FieldSelector = fields.Everything()

		nodeList, err := c.kubeClient.CoreV1().Nodes().List(ctx, listOpts)
		if err != nil {
			return false, fmt.Errorf("error listing nodes: %v", err)
		}
		c.nodeMap.ReplaceAllNodes(nodeList.Items)
		c.nodeMap.MarkReady()

		//listOpts.LabelSelector = labels.Everything()
		//listOpts.FieldSelector = fields.Everything()

		listOpts.Watch = true
		listOpts.ResourceVersion = nodeList.ResourceVersion
		klog.Infof("doing node watch from %s", listOpts.ResourceVersion)
		watcher, err := c.kubeClient.CoreV1().Nodes().Watch(ctx, listOpts)
		if err != nil {
			return false, fmt.Errorf("error watching nodes: %v", err)
		}
		ch := watcher.ResultChan()
		for {
			select {
			case <-stopCh:
				klog.Infof("Got stop signal")
				return true, nil
			case event, ok := <-ch:
				if !ok {
					klog.Infof("node watch channel closed")
					return false, nil
				}

				node := event.Object.(*v1.Node)
				klog.V(4).Infof("node changed: %s %v", event.Type, node.Name)

				switch event.Type {
				case watch.Added, watch.Modified:
					c.nodeMap.UpdateNode(node)

				case watch.Deleted:
					c.nodeMap.RemoveNode(node)

				default:
					klog.Fatalf("unexpected type of watch event: %v", event)
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
			klog.Warningf("Unexpected error in node watch, will retry: %v", err)
			time.Sleep(10 * time.Second)
		}
	}
}
