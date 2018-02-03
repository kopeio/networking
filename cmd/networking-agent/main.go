/*
Copyright 2015 The Kubernetes Authors All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	goflag "flag"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/golang/glog"
	"github.com/spf13/pflag"
	"k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"kope.io/networking"
	"kope.io/networking/pkg/routing"
	"kope.io/networking/pkg/routing/ipsec"
	"kope.io/networking/pkg/routing/layer2"
	"kope.io/networking/pkg/routing/vxlan"
	"kope.io/networking/pkg/routing/vxlan2"
	"kope.io/networking/pkg/watchers"
)

//const (
//	healthPort = 10249
//)

func main() {
	gitVersion := networking.GitVersion
	if gitVersion != "" {
		if len(gitVersion) > 6 {
			gitVersion = gitVersion[:6]
		}
		gitVersion = "git-" + gitVersion
	}

	fmt.Fprintf(os.Stdout, "kopeio-networking %v %v\n", networking.Version, gitVersion)

	options := &Options{}
	options.InitDefaults()

	err := options.LoadFrom("/config/config.yaml")
	if err != nil && !os.IsNotExist(err) {
		glog.Fatalf("error reading config file: %v", err)
	}

	flags := pflag.NewFlagSet("", pflag.ExitOnError)
	options.AddFlags(flags)

	// Trick to avoid 'logging before flag.Parse' warning
	goflag.CommandLine.Parse([]string{})

	goflag.Set("logtostderr", "true")

	flags.AddGoFlagSet(goflag.CommandLine)

	if options.LogLevel != nil {
		goflag.Set("v", strconv.Itoa(*options.LogLevel))
	}

	flags.Parse(os.Args)

	config, err := rest.InClusterConfig()
	if err != nil {
		glog.Errorf("error building client configuration: %v", err)
		os.Exit(1)
	}

	kubeClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		glog.Fatalf("error building REST client: %v", err)
	}

	var matcher func(node *v1.Node) bool
	if nodeName := os.Getenv("NODE_NAME"); nodeName != "" {
		// Passing NODE_NAME via downward API is preferred
		glog.Infof("will match node on name=%q", nodeName)
		matcher = func(node *v1.Node) bool {
			return node.Name == nodeName
		}
	} else if options.MachineIDPath != "" {
		glog.Warningf("using MachineIDPath is deprecated - prefer passing NODE_NAME via downward API")

		b, err := ioutil.ReadFile(options.MachineIDPath)
		if err != nil {
			glog.Fatalf("error reading machine-id file %q: %v", options.MachineIDPath, err)
		}
		machineID := string(b)
		machineID = strings.TrimSpace(machineID)

		glog.Infof("will match node on machineid=%q", machineID)
		matcher = func(node *v1.Node) bool {
			return node.Status.NodeInfo.MachineID == machineID
		}
	} else if options.SystemUUIDPath != "" {
		glog.Warningf("using SystemUUIDPath is deprecated - prefer passing NODE_NAME via downward API")

		b, err := ioutil.ReadFile(options.SystemUUIDPath)
		if err != nil {
			glog.Fatalf("error reading system-uuid file %q: %v", options.SystemUUIDPath, err)
		}
		systemUUID := string(b)
		systemUUID = strings.TrimSpace(systemUUID)

		// If the BIOS isn't correctly configured, we'll
		if systemUUID == "03000200-0400-0500-0006-000700080009" {
			glog.Fatalf("detected well-known invalid system-uuid 03000200-0400-0500-0006-000700080009")
		}

		glog.Infof("will match node on systemUUID=%q", systemUUID)
		matcher = func(node *v1.Node) bool {
			return node.Status.NodeInfo.SystemUUID == systemUUID
		}
	} else if options.BootIDPath != "" {
		glog.Warningf("using BootIDPath is deprecated - prefer passing NODE_NAME via downward API")

		b, err := ioutil.ReadFile(options.BootIDPath)
		if err != nil {
			glog.Fatalf("error reading boot-id file %q: %v", options.BootIDPath, err)
		}
		bootID := string(b)
		bootID = strings.TrimSpace(bootID)

		glog.Infof("will match node on bootID=%q", bootID)
		matcher = func(node *v1.Node) bool {
			return node.Status.NodeInfo.BootID == bootID
		}
	} else {
		glog.Warningf("using NodeName is deprecated - prefer passing NODE_NAME via downward API")

		matchNodeName := options.NodeName
		if matchNodeName == "" {
			hostname, err := os.Hostname()
			if err != nil {
				glog.Fatalf("error getting hostname: %v", err)
			}
			glog.Infof("Using hostname as node name: %q", hostname)
			matchNodeName = hostname
		}
		glog.Infof("will match node on name=%q", matchNodeName)
		matcher = func(node *v1.Node) bool {
			return node.Name == matchNodeName
		}
	}

	nodeMap := routing.NewNodeMap(matcher)

	targetLinkName := options.TargetLinkName
	if targetLinkName == "" {
		targetLinkName, err = findTargetLink()
		if targetLinkName == "" || err != nil {
			glog.Fatalf("unable to determine network device; pass --target to specify: %v", err)
		}
	}

	var provider routing.Provider
	switch options.Provider {
	case "layer2":
		provider, err = layer2.NewLayer2RoutingProvider(targetLinkName)
	case "gre":
		glog.Fatalf("GRE temporarily not enabled - until patch goes upstream")
	// provider, err = gre.NewGreRoutingProvider()
	case "vxlan-legacy":
		_, overlayCIDR, _ := net.ParseCIDR(options.PodCIDR)
		provider, err = vxlan.NewVxlanRoutingProvider(overlayCIDR, targetLinkName)
	case "vxlan":
		_, overlayCIDR, _ := net.ParseCIDR(options.PodCIDR)
		provider, err = vxlan2.NewVxlanRoutingProvider(overlayCIDR, targetLinkName)
	case "ipsec":
		var authenticationStrategy ipsec.AuthenticationStrategy
		var encryptionStrategy ipsec.EncryptionStrategy
		var encapsulationStrategy ipsec.EncapsulationStrategy

		switch options.IPSEC.Encryption {
		case "none":
			encryptionStrategy = &ipsec.PlaintextEncryptionStrategy{}
		case "aes":
			encryptionStrategy = &ipsec.AesEncryptionStrategy{}
		default:
			glog.Fatalf("unknown ipsec-encryption: %v", options.IPSEC.Encryption)
		}
		switch options.IPSEC.Authentication {
		case "none":
			authenticationStrategy = &ipsec.PlaintextAuthenticationStrategy{}
		case "sha1":
			authenticationStrategy = &ipsec.HmacSha1AuthenticationStrategy{}
		default:
			glog.Fatalf("unknown ipsec-authentication: %v", options.IPSEC.Authentication)
		}
		switch options.IPSEC.Encapsulation {
		case "udp":
			encapsulationStrategy = &ipsec.UdpEncapsulationStrategy{}
		case "esp":
			encapsulationStrategy = &ipsec.EspEncapsulationStrategy{}
		default:
			glog.Fatalf("unknown ipsec-encapsulation: %v", options.IPSEC.Encapsulation)
		}

		var ipsecProvider *ipsec.IpsecRoutingProvider
		ipsecProvider, err = ipsec.NewIpsecRoutingProvider(authenticationStrategy, encryptionStrategy, encapsulationStrategy)
		if err == nil {
			// TODO: This is only because state update is not working
			glog.Warningf("TODO Doing ip xfrm flush; remove!!")
			err := ipsecProvider.Flush()
			if err != nil {
				glog.Fatalf("cannot flush tables")
			}
		}
		provider = ipsecProvider

	default:
		glog.Fatalf("provider not known: %q", options.Provider)
	}

	if err != nil {
		glog.Fatalf("failed to build provider %q: %v", options.Provider, err)
	}

	//c, err := newRouteController(kubeClient, *resyncPeriod, *nodeName, bootID, systemUUID, machineID, provider)
	//if err != nil {
	//	glog.Fatalf("%v", err)
	//}

	c, err := watchers.NewNodeController(kubeClient, nodeMap)
	if err != nil {
		glog.Fatalf("Failed to build node controller: %v", err)
	}
	go c.Run()

	rc, err := routing.NewController(kubeClient, nodeMap, provider)
	if err != nil {
		glog.Fatalf("Failed to build routing controller: %v", err)
	}
	go rc.Run()
	//go registerHandlers(c)
	go handleSigterm(c)

	for {
		time.Sleep(30 * time.Second)
	}

}

//func registerHandlers(c *routeController) {
//	mux := http.NewServeMux()
//	// TODO: healthz
//	//healthz.InstallHandler(mux, lbc.nginx)
//
//	http.HandleFunc("/build", func(w http.ResponseWriter, r *http.Request) {
//		w.WriteHeader(http.StatusOK)
//		fmt.Fprint(w, "build: %v - %v", gitRepo, version)
//	})
//
//	http.HandleFunc("/stop", func(w http.ResponseWriter, r *http.Request) {
//		c.Stop()
//	})
//
//	if *profiling {
//		mux.HandleFunc("/debug/pprof/", pprof.Index)
//		mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
//		mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
//	}
//
//	server := &http.Server{
//		Addr:    fmt.Sprintf(":%v", *healthzPort),
//		Handler: mux,
//	}
//	glog.Fatal(server.ListenAndServe())
//}

func handleSigterm(c *watchers.NodeController) {
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGTERM)
	<-signalChan
	glog.Infof("Received SIGTERM, shutting down")

	exitCode := 0
	if err := c.Stop(); err != nil {
		glog.Infof("Error during shutdown %v", err)
		exitCode = 1
	}
	glog.Infof("Exiting with %v", exitCode)
	os.Exit(exitCode)
}

// findTargetLink attempts to discover the correct network interface
func findTargetLink() (string, error) {
	networkInterfaces, err := net.Interfaces()
	if err != nil {
		return "", fmt.Errorf("error querying interfaces to determine primary network interface: %v", err)
	}

	var candidates []string
	for i := range networkInterfaces {
		networkInterface := &networkInterfaces[i]
		flags := networkInterface.Flags
		name := networkInterface.Name

		if (flags & net.FlagLoopback) != 0 {
			glog.V(2).Infof("Ignoring interface %s - loopback", name)
			continue
		}

		// Not a lot else to go on...
		if !strings.HasPrefix(name, "eth") && !strings.HasPrefix(name, "en") {
			glog.V(2).Infof("Ignoring interface %s - name does not look like ethernet device", name)
			continue
		}

		candidates = append(candidates, networkInterface.Name)
	}

	if len(candidates) == 0 {
		return "", fmt.Errorf("unable to determine interface (no interfaces found)")
	}

	if len(candidates) == 1 {
		return candidates[0], nil
	}

	return "", fmt.Errorf("unable to determine interface (multiple interfaces found: %s)", strings.Join(candidates, ","))
}
