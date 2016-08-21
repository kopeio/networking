## GRE

GRE routing sets up a full mesh of GRE tunnels, from each node to every other node.

For each remote node, it adds a tunnel:

`ip tunnel add $name mode gre remote $remoteIP local $localIP`

And then we route packets to the destination CIDR over the tunnel:

`ip route add $remoteCIDR via $remoteIP`
