package routecontroller

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/golang/glog"
	"os/exec"
	"strings"
)

const tunnelTTL = "255" // TODO: What is the correct value for a GRE tunnel?

type IPLink struct {
	Device          string
	Up              bool
	Peer            string
	GRE             bool
	GRELocalAddress string
}

func (l *IPLink) String() string {
	return AsJsonString(l)
}

type IPLinks struct {
	Links []*IPLink
}

func (i *IPLinks) FindByDevice(device string) *IPLink {
	for _, l := range i.Links {
		if l.Device == device {
			return l
		}
	}
	return nil
}

func (i *IPLinks) EnsureGRETunnel(name string, remoteIP string, localIP string) error {
	var existing *IPLink
	for _, link := range i.Links {
		if link.Device == name {
			existing = link
			break
		}
	}

	if existing != nil {
		delete := false
		if existing.GRELocalAddress != localIP {
			glog.Infof("Tunnel exists, but local IP does not match: %q vs %q", existing.GRELocalAddress, localIP)
			delete = true
		}
		if existing.Peer != remoteIP {
			glog.Infof("Tunnel exists, but remote IP does not match: %q vs %q", existing.Peer, remoteIP)
			delete = true
		}

		if delete {
			glog.Infof("Deleting tunnel, as settings not correct")
			err := i.DeleteTunnel(name)
			if err != nil {
				return fmt.Errorf("error deleting existing tunnel: %v", err)
			}
			existing = nil
		}
	}

	if existing == nil {
		glog.Infof("Creating GRE tunnel %s: %s -> %s", name, localIP, remoteIP)

		argv := []string{"ip", "tunnel", "add", name, "mode", "gre", "remote", remoteIP, "local", localIP, "ttl", tunnelTTL}
		humanArgv := strings.Join(argv, " ")

		glog.V(2).Infof("Running %q", humanArgv)
		cmd := exec.Command(argv[0], argv[1:]...)

		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("error running %q: %v: %q", humanArgv, err, string(out))
		}
	}

	if existing == nil || !existing.Up {
		glog.Infof("Setting GRE tunnel to up %s", name)

		argv := []string{"ip", "link", "set", name, "up"}
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

func (i *IPLinks) DeleteTunnel(name string) error {
	{
		glog.Infof("Setting GRE tunnel to down %s", name)

		argv := []string{"ip", "link", "set", name, "down"}
		humanArgv := strings.Join(argv, " ")

		glog.V(2).Infof("Running %q", humanArgv)
		cmd := exec.Command(argv[0], argv[1:]...)

		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("error running %q: %v: %q", humanArgv, err, string(out))
		}
	}

	{
		glog.Infof("Deleting GRE tunnel %s", name)

		argv := []string{"ip", "tunnel", "del", name}
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

func QueryIPLinks() (*IPLinks, error) {
	// TODO: We could query netlink directly e.g. https://golang.org/src/net/interface_linux.go

	//	1: lo: <LOOPBACK,UP,LOWER_UP> mtu 65536 qdisc noqueue state UNKNOWN mode DEFAULT group default \    link/loopback 00:00:00:00:00:00 brd 00:00:00:00:00:00
	//2: eth0: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 9001 qdisc pfifo_fast state UP mode DEFAULT group default qlen 1000\    link/ether 06:12:c7:ec:b9:7d brd ff:ff:ff:ff:ff:ff
	//3: docker0: <NO-CARRIER,BROADCAST,MULTICAST,UP> mtu 1500 qdisc noqueue state DOWN mode DEFAULT group default \    link/ether 02:42:50:29:a6:27 brd ff:ff:ff:ff:ff:ff
	//4: cbr0: <BROADCAST,MULTICAST,PROMISC,UP,LOWER_UP> mtu 9001 qdisc htb state UP mode DEFAULT group default \    link/ether 0a:a4:68:31:3f:6e brd ff:ff:ff:ff:ff:ff
	//6: veth06aec7b: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 9001 qdisc noqueue master cbr0 state UP mode DEFAULT group default \    link/ether 0a:a4:68:31:3f:6e brd ff:ff:ff:ff:ff:ff
	//7: gre0@NONE: <NOARP> mtu 1476 qdisc noop state DOWN mode DEFAULT group default \    link/gre 0.0.0.0 brd 0.0.0.0
	//8: gretap0@NONE: <BROADCAST,MULTICAST> mtu 1462 qdisc noop state DOWN mode DEFAULT group default qlen 1000\    link/ether 00:00:00:00:00:00 brd ff:ff:ff:ff:ff:ff
	//9: gre-10-244-1-0@NONE: <POINTOPOINT,NOARP,UP,LOWER_UP> mtu 8977 qdisc noqueue state UNKNOWN mode DEFAULT group default \    link/gre 172.20.30.99 peer 172.20.30.98
	//10: gre-10-244-0-0@NONE: <POINTOPOINT,NOARP,UP,LOWER_UP> mtu 8977 qdisc noqueue state UNKNOWN mode DEFAULT group default \    link/gre 172.20.30.99 peer 172.20.17.122
	//12: vetha04f9c1: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 9001 qdisc noqueue master cbr0 state UP mode DEFAULT group default \    link/ether 4a:98:0e:0f:04:95 brd ff:ff:ff:ff:ff:ff
	//14: veth04f4a8d: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 9001 qdisc noqueue master cbr0 state UP mode DEFAULT group default \    link/ether da:93:56:57:6b:b0 brd ff:ff:ff:ff:ff:ff

	argv := []string{"ip", "-oneline", "link", "show"}
	humanArgv := strings.Join(argv, " ")

	glog.V(2).Infof("Running %q", humanArgv)
	cmd := exec.Command(argv[0], argv[1:]...)

	// TODO: Should we separate out stderr?
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("error running %q: %v", humanArgv, err)
	}

	return parseIPLinks(out)
}

func parseIPLinks(out []byte) (*IPLinks, error) {
	var links []*IPLink

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
		if len(fields) < 3 {
			glog.Warningf("Ignoring unparseable line: %q", line)
			continue
		}

		link := &IPLink{}

		//1: lo: <LOOPBACK,UP,LOWER_UP> mtu 65536 qdisc noqueue state UNKNOWN mode DEFAULT group default qlen 1\    link/loopback 00:00:00:00:00:00 brd 00:00:00:00:00:00
		//9: gre-10-244-1-0@NONE: <POINTOPOINT,NOARP,UP,LOWER_UP> mtu 8977 qdisc noqueue state UNKNOWN mode DEFAULT group default \    link/gre 172.20.30.99 peer 172.20.30.98

		//index := fields[0]

		deviceName := fields[1]
		deviceName = strings.TrimSuffix(deviceName, ":")
		if atIndex := strings.IndexRune(deviceName, '@'); atIndex != -1 {
			deviceName = deviceName[:atIndex]
		}
		link.Device = deviceName

		for _, flag := range strings.Split(strings.Trim(fields[2], "<>"), ",") {
			if flag == "UP" {
				link.Up = true
			}
		}

		for i := 3; i < len(fields); i += 2 {
			if (i + 1) >= len(fields) {
				glog.Warningf("ip link unparseable line: %q", line)
				continue
			}
			key := fields[i]
			value := fields[i+1]
			switch key {
			case "link/gre":
				link.GRE = true
				link.GRELocalAddress = value
			case "peer":
				link.Peer = value
			}
		}

		links = append(links, link)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error parsing ip links output: %v", err)
	}

	return &IPLinks{Links: links}, nil

}
