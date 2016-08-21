package main

import (
	"github.com/golang/glog"
	"github.com/kopeio/route-controller/pkg/routecontroller/routingproviders/ipsecrouting"
	"time"
)

func main() {
	port := 4500
	glog.Infof("Creating encap listener on port %d", port)
	l, err := ipsecrouting.NewUDPEncapListener(port)
	if err != nil {
		glog.Fatalf("error creating UDP encapsulation listener on port %d: %v", port, err)
	}
	for {
		time.Sleep(10 * time.Second)
	}
	l.Close()
}
