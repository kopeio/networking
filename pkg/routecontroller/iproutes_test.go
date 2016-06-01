package routecontroller

import (
	"fmt"
	"testing"
)

const ipRoutesOutput = `
default via 172.20.0.1 dev eth0
10.244.0.0/24 dev gre-10-244-0-0  scope link
10.244.1.0/24 dev cbr0  proto kernel  scope link  src 10.244.1.1
10.244.2.0/24 dev gre-10-244-2-0  scope link
172.17.0.0/16 dev docker0  proto kernel  scope link  src 172.17.0.1
172.20.0.0/19 dev eth0  proto kernel  scope link  src 172.20.30.98
`

func TestParseIPRoutes(t *testing.T) {
	routes, err := parseIPRoutes([]byte(ipRoutesOutput))
	if err != nil {
		t.Fatalf("error parsing ip routes: %v", err)
	}

	for _, l := range routes.Routes {
		if l.Device == "" {
			t.Fatalf("Device not set in %v", l)
		}
		if l.CIDR == "" {
			t.Fatalf("CIDR not set in %v", l)
		}
		fmt.Printf("Route: %v\n", l)
	}

	if len(routes.Routes) != 6 {
		t.Fatalf("Expected 6 routes, got %d", len(routes.Routes))
	}

	matches := routes.FindByCIDR("10.244.2.0/24")
	if len(matches) != 1 {
		t.Fatalf("Did not find exactly one route for 10.244.2.0/24")
	}

	match := matches[0]
	if match.Device != "gre-10-244-2-0" {
		t.Fatalf("route for 10.244.2.0/24 should be gre-10-244-2-0")
	}
	if match.CIDR != "10.244.2.0/24" {
		t.Fatalf("route for 10.244.2.0/24 should be 10.244.2.0/24")
	}
}
