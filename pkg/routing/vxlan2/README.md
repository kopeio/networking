## VXLAN2

A simplified version of the original VXLAN configuration, that avoids the need for dynamic ARP queries.

A VXLAN device is a virtual device which can forward packets over UDP.

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

2. we set up a `bridge fdb`, for each remote node.  This associated the mac from 1 with the VXLAN endpoint IP - the
node's internal IP:

bridge fdb show

```
...
00:53:64:60:00:00 dev vxlan1 dst 172.20.110.17 self permanent
00:53:64:60:01:00 dev vxlan1 dst 172.20.108.12 self permanent
...
```

3. we set up ARP entries for VXLAN gateway IPs for each other node:

ip neigh show | sort

```
...
100.96.0.0 dev vxlan1 lladdr 00:53:64:60:00:00 PERMANENT
100.96.1.0 dev vxlan1 lladdr 00:53:64:60:01:00 PERMANENT
...
```

4. we set up a route table entry for the other CIDRs, pointing to them to the VXLAN gateway IP (.0).  The FDB
resolves that to the internal IP.

ip route
```
...
100.96.0.0/24 via 100.96.0.0 dev vxlan1 onlink 
100.96.1.0/24 via 100.96.1.0 dev vxlan1 onlink 
...
```
