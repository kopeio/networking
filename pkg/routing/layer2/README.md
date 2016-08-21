## Layer2

Layer2 routing is very simple, but does not work on most clouds.  It may be a good option for bare metal.

It does work on AWS VPC but only within a single AZ, and you must turn off src/dest checks on all instances.

For each remote node, it adds a route:

`ip route add $remoteCidr via $remoteIP`