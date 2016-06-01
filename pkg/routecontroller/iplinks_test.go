package routecontroller

import (
	"fmt"
	"testing"
)

const iplinksOutput = `
1: lo: <LOOPBACK,UP,LOWER_UP> mtu 65536 qdisc noqueue state UNKNOWN mode DEFAULT group default \    link/loopback 00:00:00:00:00:00 brd 00:00:00:00:00:00
2: eth0: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 9001 qdisc pfifo_fast state UP mode DEFAULT group default qlen 1000\    link/ether 06:98:25:7d:7e:2b brd ff:ff:ff:ff:ff:ff
3: docker0: <NO-CARRIER,BROADCAST,MULTICAST,UP> mtu 1500 qdisc noqueue state DOWN mode DEFAULT group default \    link/ether 02:42:b2:5c:9b:07 brd ff:ff:ff:ff:ff:ff
4: cbr0: <BROADCAST,MULTICAST,PROMISC,UP,LOWER_UP> mtu 9001 qdisc htb state UP mode DEFAULT group default \    link/ether 52:04:79:84:84:b9 brd ff:ff:ff:ff:ff:ff
5: gre0@NONE: <NOARP> mtu 1476 qdisc noop state DOWN mode DEFAULT group default \    link/gre 0.0.0.0 brd 0.0.0.0
6: gretap0@NONE: <BROADCAST,MULTICAST> mtu 1462 qdisc noop state DOWN mode DEFAULT group default qlen 1000\    link/ether 00:00:00:00:00:00 brd ff:ff:ff:ff:ff:ff
7: gre-10-244-2-0@NONE: <POINTOPOINT,NOARP,UP,LOWER_UP> mtu 8977 qdisc noqueue state UNKNOWN mode DEFAULT group default \    link/gre 172.20.30.98 peer 172.20.30.99
8: gre-10-244-0-0@NONE: <POINTOPOINT,NOARP,UP,LOWER_UP> mtu 8977 qdisc noqueue state UNKNOWN mode DEFAULT group default \    link/gre 172.20.30.98 peer 172.20.17.122
10: veth25f1ef4: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 9001 qdisc noqueue master cbr0 state UP mode DEFAULT group default \    link/ether 82:60:07:8c:60:47 brd ff:ff:ff:ff:ff:ff
12: vethd40bfa9: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 9001 qdisc noqueue master cbr0 state UP mode DEFAULT group default \    link/ether 52:04:79:84:84:b9 brd ff:ff:ff:ff:ff:ff
14: vethe83113b: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 9001 qdisc noqueue master cbr0 state UP mode DEFAULT group default \    link/ether aa:ae:b2:f7:ae:ae brd ff:ff:ff:ff:ff:ff
16: veth16e3e30: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 9001 qdisc noqueue master cbr0 state UP mode DEFAULT group default \    link/ether 9a:09:65:68:f2:56 brd ff:ff:ff:ff:ff:ff
`

func TestParseIPLinks(t *testing.T) {
	links, err := parseIPLinks([]byte(iplinksOutput))
	if err != nil {
		t.Fatalf("error parsing ip links: %v", err)
	}

	for _, l := range links.Links {
		if l.Device == "" {
			t.Fatalf("Device not set in %v", l)
		}
		fmt.Printf("Link: %v\n", l)
	}

	if len(links.Links) != 12 {
		t.Fatalf("Expected 12 links, got %d", len(links.Links))
	}

	tunnel0 := links.FindByDevice("gre-10-244-0-0")
	if tunnel0 == nil {
		t.Fatalf("Unable to find tunnel gre-10-244-0-0")
	}

	if !tunnel0.Up {
		t.Fatalf("tunnel gre-10-244-0-0: should be UP: %v", tunnel0)
	}
	if !tunnel0.GRE {
		t.Fatalf("tunnel gre-10-244-0-0: should be GRE: %v", tunnel0)
	}
	if tunnel0.GRELocalAddress != "172.20.30.98" {
		t.Fatalf("tunnel gre-10-244-0-0: local should be 172.20.30.98: %v", tunnel0)
	}
	if tunnel0.Peer != "172.20.17.122" {
		t.Fatalf("tunnel gre-10-244-0-0: peer should be 172.20.17.122: %v", tunnel0)
	}

}
