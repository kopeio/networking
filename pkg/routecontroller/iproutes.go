package routecontroller

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/golang/glog"
	"os/exec"
	"strings"
)

type IPRoute struct {
	CIDR   string
	Device string
	Via    string
}

func (l *IPRoute) String() string {
	return AsJsonString(l)
}

type IPRoutes struct {
	Routes []*IPRoute
}

func (l *IPRoutes) FindByCIDR(cidr string) []*IPRoute {
	var matches []*IPRoute
	for _, r := range l.Routes {
		if r.CIDR == cidr {
			matches = append(matches, r)
		}
	}
	return matches
}

func QueryIPRoutes() (*IPRoutes, error) {
	// TODO: We could query netlink directly e.g. https://golang.org/src/net/interface_linux.go

	argv := []string{"ip", "-oneline", "route", "show"}
	humanArgv := strings.Join(argv, " ")

	glog.V(2).Infof("Running %q", humanArgv)
	cmd := exec.Command(argv[0], argv[1:]...)

	// TODO: Should we separate out stderr?
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("error running %q: %v", humanArgv, err)
	}

	return parseIPRoutes(out)
}

func parseIPRoutes(out []byte) (*IPRoutes, error) {
	//default via 172.20.0.1 dev eth0
	//10.244.0.0/24 dev gre-10-244-0-0  scope link
	//10.244.1.0/24 dev cbr0  proto kernel  scope link  src 10.244.1.1
	//10.244.2.0/24 dev gre-10-244-2-0  scope link
	//172.17.0.0/16 dev docker0  proto kernel  scope link  src 172.17.0.1
	//172.20.0.0/19 dev eth0  proto kernel  scope link  src 172.20.30.98
	// 10.244.1.0/24 via 172.20.21.42 dev eth0
	// 10.244.2.0/24 via 172.20.21.41 dev eth0

	var routes []*IPRoute

	buf := bytes.NewReader(out)
	scanner := bufio.NewScanner(buf)
	for scanner.Scan() {
		line := scanner.Text()

		// Remove \ character that replaced new lines
		line = strings.Replace(line, "\\", " ", -1)

		// Ignore empty lines
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 1 {
			glog.Warningf("Ignoring unparseable line: %q", line)
			continue
		}

		route := &IPRoute{}

		cidr := fields[0]
		route.CIDR = cidr

		for i := 1; i < len(fields); i += 2 {
			if (i + 1) >= len(fields) {
				glog.Warningf("ip route unparseable line: %q", line)
				continue
			}
			key := fields[i]
			value := fields[i+1]
			switch key {
			case "dev":
				route.Device = value
			case "via":
				route.Via = value
			}
		}

		routes = append(routes, route)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error parsing ip routes output: %v", err)
	}

	return &IPRoutes{Routes: routes}, nil

}

func (i *IPRoutes) EnsureRouteViaIP(cidr string, via string) error {
	var existing *IPRoute
	for _, r := range i.Routes {
		if r.CIDR == cidr {
			existing = r
			break
		}
	}

	if existing != nil {
		delete := false
		if existing.Via != via {
			glog.Infof("Route exists, but via does not match: %q vs %q", existing.Via, via)
			delete = true
		}

		if delete {
			glog.Infof("Deleting route, as settings not correct")
			err := i.DeleteRoute(existing)
			if err != nil {
				return fmt.Errorf("error deleting existing route: %v", err)
			}
			existing = nil
		}
	}

	if existing == nil {
		glog.Infof("Creating route: %s via %s", cidr, via)

		argv := []string{"ip", "route", "add", cidr, "via", via}
		humanArgv := strings.Join(argv, " ")

		glog.V(2).Infof("Running %q", humanArgv)
		cmd := exec.Command(argv[0], argv[1:]...)

		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("error running %q: %v: %q", humanArgv, err, string(out))
		}
	}

	return nil
}

func (i *IPRoutes) EnsureRouteToDevice(cidr string, device string) error {
	var existing *IPRoute
	for _, r := range i.Routes {
		if r.CIDR == cidr {
			existing = r
			break
		}
	}

	if existing != nil {
		delete := false
		if existing.Device != device {
			glog.Infof("Route exists, but device does not match: %q vs %q", existing.Device, device)
			delete = true
		}

		if delete {
			glog.Infof("Deleting route, as settings not correct")
			err := i.DeleteRoute(existing)
			if err != nil {
				return fmt.Errorf("error deleting existing route: %v", err)
			}
			existing = nil
		}
	}

	if existing == nil {
		glog.Infof("Creating route: %s %s", cidr, device)

		argv := []string{"ip", "route", "add", cidr, "dev", device}
		humanArgv := strings.Join(argv, " ")

		glog.V(2).Infof("Running %q", humanArgv)
		cmd := exec.Command(argv[0], argv[1:]...)

		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("error running %q: %v: %q", humanArgv, err, string(out))
		}
	}

	return nil
}

func (i *IPRoutes) DeleteRoute(r *IPRoute) error {
	{
		glog.Infof("Deleting route: %s")

		argv := []string{"ip", "route", "delete", r.CIDR}
		if r.Device != "" {
			argv = append(argv, "dev", r.Device)
		}
		if r.Via != "" {
			argv = append(argv, "via", r.Device)
		}

		humanArgv := strings.Join(argv, " ")

		glog.V(2).Infof("Running %q", humanArgv)
		cmd := exec.Command(argv[0], argv[1:]...)

		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("error running %q: %v: %q", humanArgv, err, string(out))
		}
	}

	return nil
}
