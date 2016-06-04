package main

import (
	"fmt"
	"reflect"
	"sync"
	"time"

	"github.com/golang/glog"

	"github.com/kopeio/route-controller/pkg/routecontroller"
	"github.com/kopeio/route-controller/pkg/routecontroller/routingproviders"
	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/client/cache"
	"k8s.io/kubernetes/pkg/client/record"
	client "k8s.io/kubernetes/pkg/client/unversioned"
	"k8s.io/kubernetes/pkg/controller/framework"
	"k8s.io/kubernetes/pkg/runtime"
	"k8s.io/kubernetes/pkg/watch"
)

var (
	keyFunc = framework.DeletionHandlingMetaNamespaceKeyFunc
)

// routeController watches the kubernetes api and adds/removes DNS entries
type routeController struct {
	selfNodeName   string
	selfMachineID  string
	selfSystemUUID string
	selfBootID     string
	provider       routingproviders.RoutingProvider

	client         *client.Client
	nodeController *framework.Controller
	nodeLister     cache.StoreToNodeLister

	recorder record.EventRecorder

	syncQueue *taskQueue

	// stopLock is used to enforce only a single call to Stop is active.
	// Needed because we allow stopping through an http endpoint and
	// allowing concurrent stoppers leads to stack traces.
	stopLock sync.Mutex
	shutdown bool
	stopCh   chan struct{}
}

// newRouteController creates a controller for node routes
func newRouteController(kubeClient *client.Client,
	resyncPeriod time.Duration,
	selfNodeName string,
	selfBootID string,
	selfSystemUUID string,
	selfMachineID string,
	provider routingproviders.RoutingProvider) (*routeController, error) {

	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(glog.Infof)
	eventBroadcaster.StartRecordingToSink(kubeClient.Events(""))

	c := routeController{
		selfNodeName:   selfNodeName,
		selfBootID:     selfBootID,
		selfSystemUUID: selfSystemUUID,
		selfMachineID:  selfMachineID,
		provider:       provider,
		client:         kubeClient,
		stopCh:         make(chan struct{}),
		recorder:       eventBroadcaster.NewRecorder(api.EventSource{Component: "loadbalancer-controller"}),
	}

	c.syncQueue = NewTaskQueue(c.sync)

	nodeEventHandler := framework.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			addNode := obj.(*api.Node)
			c.recorder.Eventf(addNode, api.EventTypeNormal, "CREATE", fmt.Sprintf("%s/%s", addNode.Namespace, addNode.Name))
			c.syncQueue.enqueue(obj)
		},
		DeleteFunc: func(obj interface{}) {
			delNode := obj.(*api.Node)
			c.recorder.Eventf(delNode, api.EventTypeNormal, "DELETE", fmt.Sprintf("%s/%s", delNode.Namespace, delNode.Name))
			c.syncQueue.enqueue(obj)
		},
		UpdateFunc: func(old, cur interface{}) {
			if !reflect.DeepEqual(old, cur) {
				updateNode := cur.(*api.Node)
				c.recorder.Eventf(updateNode, api.EventTypeNormal, "UPDATE", fmt.Sprintf("%s/%s", updateNode.Namespace, updateNode.Name))
				c.syncQueue.enqueue(cur)
			}
		},
	}

	c.nodeLister.Store, c.nodeController = framework.NewInformer(
		&cache.ListWatch{
			ListFunc:  nodeListFunc(c.client),
			WatchFunc: nodeWatchFunc(c.client),
		},
		&api.Node{}, resyncPeriod, nodeEventHandler)

	return &c, nil
}

func nodeListFunc(c *client.Client) func(api.ListOptions) (runtime.Object, error) {
	return func(opts api.ListOptions) (runtime.Object, error) {
		return c.Nodes().List(opts)
	}
}

func nodeWatchFunc(c *client.Client) func(options api.ListOptions) (watch.Interface, error) {
	return func(options api.ListOptions) (watch.Interface, error) {
		return c.Nodes().Watch(options)
	}
}

func (c *routeController) controllersInSync() bool {
	return c.nodeController.HasSynced()
}

func (c *routeController) sync(key string) error {
	if !c.controllersInSync() {
		c.syncQueue.requeue(key, fmt.Errorf("deferring sync till node controller has synced"))
		return nil
	}

	nodeList, err := c.nodeLister.List()
	if err != nil {
		// Impossible?
		return fmt.Errorf("error listing nodes: %v", err)
	}

	var me *api.Node
	if c.selfBootID != "" {
		for i := range nodeList.Items {
			node := &nodeList.Items[i]
			if node.Status.NodeInfo.BootID == c.selfBootID {
				if me != nil {
					glog.Fatalf("Found multiple nodes with boot-id: %q (%q and %q)", c.selfBootID, node.Name, me.Name)
				}
				me = node
			}
		}
		if me == nil {
			return fmt.Errorf("unable to find self-node boot-id: %q", c.selfBootID)
		}
	} else if c.selfSystemUUID != "" {
		for i := range nodeList.Items {
			node := &nodeList.Items[i]
			if node.Status.NodeInfo.SystemUUID == c.selfSystemUUID {
				if me != nil {
					glog.Fatalf("Found multiple nodes with system-uuid: %q (%q and %q)", c.selfSystemUUID, node.Name, me.Name)
				}
				me = node
			}
		}
		if me == nil {
			return fmt.Errorf("unable to find self-node system-uuid: %q", c.selfSystemUUID)
		}
	} else if c.selfMachineID != "" {
		for i := range nodeList.Items {
			node := &nodeList.Items[i]
			if node.Status.NodeInfo.MachineID == c.selfMachineID {
				if me != nil {
					glog.Fatalf("Found multiple nodes with machineid: %q (%q and %q)", c.selfMachineID, node.Name, me.Name)
				}
				me = node
			}
		}
		if me == nil {
			return fmt.Errorf("unable to find self-node machine-id: %q", c.selfMachineID)
		}
	} else if c.selfNodeName != "" {
		for i := range nodeList.Items {
			node := &nodeList.Items[i]
			if node.Name == c.selfNodeName {
				if me != nil {
					glog.Fatalf("Found multiple nodes with name: %q (%q and %q)", c.selfNodeName, node.Name, me.Name)
				}
				me = node
			}
		}
		if me == nil {
			return fmt.Errorf("unable to find self-node name: %q", c.selfNodeName)
		}
	} else {
		return fmt.Errorf("must set either node-name or machine-id")
	}

	err = c.provider.EnsureCIDRs(me, nodeList.Items)
	if err != nil {
		return fmt.Errorf("error while trying to create names: %v", err)
	}

	return nil
}

func (lbc *routeController) buildCIDRMap(nodes []api.Node) map[string]string {
	cidrMap := make(map[string]string)

	for j := range nodes {
		node := &nodes[j]

		name := node.Name

		podCIDR := node.Spec.PodCIDR
		if podCIDR == "" {
			glog.Infof("skipping node with no CIDR: %q", name)
			continue
		}

		internalIP := routecontroller.FindInternalIPAddress(node)
		if internalIP == "" {
			glog.Infof("skipping node with no InternalIP Address: %q", name)
			continue
		}

		cidrMap[podCIDR] = internalIP
	}

	return cidrMap
}

// Stop stops the route controller.
func (c *routeController) Stop() error {
	// Stop is invoked from the http endpoint.
	c.stopLock.Lock()
	defer c.stopLock.Unlock()

	// Only try draining the workqueue if we haven't already.
	if !c.shutdown {
		close(c.stopCh)
		glog.Infof("shutting down controller queues")
		c.shutdown = true
		c.syncQueue.shutdown()

		return nil
	}

	return fmt.Errorf("shutdown already in progress")
}

// Run starts the route controller.
func (c *routeController) Run() {
	glog.Infof("starting route controller")

	go c.nodeController.Run(c.stopCh)

	go c.syncQueue.run(time.Second, c.stopCh)

	<-c.stopCh
	glog.Infof("shutting down route controller")
}
