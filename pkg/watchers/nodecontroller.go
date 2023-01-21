package watchers

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
	"kope.io/networking/pkg/routing"
)

// NodeController watches for nodes
type NodeController struct {
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

	c.runWatcher(ctx)

	klog.Infof("exiting node controller")
}

func (c *NodeController) runWatcher(ctx context.Context) {
	runOnce := func() (bool, error) {
		var listOpts metav1.ListOptions

		// We need to watch all the nodes
		//listOpts.LabelSelector = labels.Everything()
		//listOpts.FieldSelector = fields.Everything()

		nodeList, err := c.kubeClient.CoreV1().Nodes().List(ctx, listOpts)
		if err != nil {
			return false, fmt.Errorf("error listing nodes: %w", err)
		}
		c.nodeMap.ReplaceAllNodes(nodeList.Items)
		c.nodeMap.MarkReady()

		listOpts.Watch = true
		listOpts.ResourceVersion = nodeList.ResourceVersion
		klog.Infof("starting node watch from %s", listOpts.ResourceVersion)
		watcher, err := c.kubeClient.CoreV1().Nodes().Watch(ctx, listOpts)
		if err != nil {
			return false, fmt.Errorf("error watching nodes: %w", err)
		}
		defer watcher.Stop()

		ch := watcher.ResultChan()
		for {
			select {
			case <-ctx.Done():
				klog.Infof("Got stop signal")
				return true, ctx.Err()
			case event, ok := <-ch:
				if !ok {
					klog.Infof("node watch channel closed")
					return false, nil
				}

				node, ok := event.Object.(*corev1.Node)
				if !ok {
					klog.Warningf("unexpected event:  type=%q, object=%T:%+v", event.Type, event.Object)
					return false, fmt.Errorf("object had unexpected type %T", event.Object)
				}
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
