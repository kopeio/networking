# Simple Networking Provider for Kubernetes

**Status: Moving from experimental to alpha.**

kope-routing is the easiest networking controller for kubernetes.

It is kubernetes-native, meaning that it uses the Kubernetes API to manage state (no second source
of truth), and installation is simply a matter of installing a daemonset.

Kubernetes already allocates a CIDR to each Node, and the route controller simply configures
linux native networking with that CIDR -> node mapping.  It runs on every node, and for example
with Layer2 networking, will effectively call `ip route add $nodeCIDR via $nodeIP` for every other
node.  Though the other modes are less simple to configure, they all boil down to pretty standard
`ip link` manipulation (though VXLAN has an ARP helper process).   Alongside each routing option there is
documentation on what is actually going on under the covers.

kops-routing is not tied to a particular cloud provider.  VXLAN and IPSEC should work
everywhere UDP is supported!

## Transport Choices

kops-routing is also relatively transport agnostic, supporting several different modes:

* `layer2` requires Layer-2 networking connectivity, and is thus primarily useful on bare
metal (though it also works on AWS in a single AZ & subnet).  It sets up a `ip route` for
each node ( [more details](pkg/routing/layer2/README.md) ).  Performance is excellent,
but the Layer 2 connectivity requirement means it cannot be used everywhere.

* `vxlan` requires only UDP connectivity, but also requires a user-space component
so that the kernel can perform the equivalent of ARP requests to remote machines (
[more details](pkg/routing/vxlan/README.md) ).  The user-space component is included
in the daemonset, of course!

* `ipsec` only requires UDP connectivity (though can also work over the native IPSEC
protocols).  Encryption is optional, so plaintext IPSEC is really an alternative way
of doing insecure tunneling (like vxlan).

** ipsec encryption currently makes no attempt to choose secure keys.  Do not consider
it secure! **

## Configuration

Bring up your cluster as normal!  We recommend [kops](https://github.com/kubernetes/kops) if
you are on AWS, but networking is now separate from installation.

* the controller-manager should have `--allocate-node-cidrs=true` and `--configure-cloud-routes=false`.  We
want it to allocate a CIDR to each Node, but the daemonset will configure connectivity.

Your cluster should start without networking, but pods on different nodes will not
be able to communicate with each other.  They might not even be able to reach the API server.
But that is OK, because kubelets talk to the master over the "real" network, not the overlay
network.  In addition, the designers of kubernetes set it up so that pods can talk to the API
before the overlay network is in place (the `kubernetes` service is specially routed).  So
we can run daemonsets, they will schedule, bring up the network and all is well.

Daemonsets are included!

Simply create the appropriate daemonset:

* `layer2`: `kubectl create -f https://raw.githubusercontent.com/kopeio/kope-routing/master/k8s/layer2.yaml`
* `vxlan`: `kubectl create -f https://raw.githubusercontent.com/kopeio/kope-routing/master/k8s/vxlan.yaml`
* `ipsec-plaintext`: `kubectl create -f https://raw.githubusercontent.com/kopeio/kope-routing/master/k8s/ipsec-plaintext.yaml`
* `ipsec-encrypted` (not yet secure!): `kubectl create -f https://raw.githubusercontent.com/kopeio/kope-routing/master/k8s/ipsec-encrypted.yaml`


You can of course clone this repository and work from the filesystem instead.



