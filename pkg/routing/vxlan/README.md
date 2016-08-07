## VXLAN

The setup is pretty simple, but there are some implementation complexities.

A VXLAN device is a virtual device which can forward packets over UDP.

We set it up with the CIDR for the whole pod network (not the service network), e.g. 100.96.0.0/12

Note that we set up the IP of the VXLAN to be an IP in our pod CIDR, otherwise traffic that originates from the host
 cannot be routed back.

So the question is: how do we tell the kernel that a particular pod IP should be routed to a particular VXLAN endpoint?
It is trivial to map the pod IP to the public IP, using the Kubernetes Nodes API.

There are two pieces to this:

1. we assign each vxlan interface a unique mac on each machine.  We actually encode the PodCIDR in here for sanity.

ip link show

```
...
9: vxlan1: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 1500 qdisc noqueue state UNKNOWN mode DEFAULT group default 
    link/ether 00:53:64:60:02:00 brd ff:ff:ff:ff:ff:ff
...
```

2. we set up a `bridge fdb`, for each remote node.  This associated the mac from 1 with the VXLAN endpoint IP

bridge fdb show

```
...
00:53:64:60:00:00 dev vxlan1 dst 172.20.110.17 self permanent
00:53:64:60:01:00 dev vxlan1 dst 172.20.108.12 self permanent
...
```

3. we set up ARP entries for every other pod, mapping to the vxlan MAC.  The VXLAN machine does the mapping to the endpoint IP and the encapsulation.

ip neigh show | sort

```
...
100.96.1.3 dev vxlan1 lladdr 00:53:64:60:01:00 STALE
100.96.1.4 dev vxlan1 lladdr 00:53:64:60:01:00 REACHABLE
100.96.1.5 dev vxlan1 lladdr 00:53:64:60:01:00 REACHABLE
100.96.2.3 dev cbr0 lladdr 02:42:64:60:02:03 STALE
100.96.2.4 dev cbr0 lladdr 02:42:64:60:02:04 REACHABLE
100.96.2.5 dev cbr0 lladdr 02:42:64:60:02:05 REACHABLE
100.96.2.6 dev cbr0 lladdr 02:42:64:60:02:06 REACHABLE
172.17.0.2 dev docker0 lladdr 02:42:ac:11:00:02 STALE
172.20.110.17 dev eth0 lladdr 0a:5b:03:7b:f0:c7 STALE
172.20.96.1 dev eth0 lladdr 0a:d6:10:07:62:d7 REACHABLE
...
```

The next question is how we set up those ARP entries.  We don't really want to set it up for every pod in advance
(though we could).  The answer is that the kernel can notify us over netlink when an L2 miss is happening, and we
inject the L2 mapping on demand.
