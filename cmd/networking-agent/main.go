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
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
	"kope.io/networking"
	"kope.io/networking/pkg/cni"
	"kope.io/networking/pkg/routing"
	"kope.io/networking/pkg/routing/gre"
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
	ctx := context.Background()

	if err := run(ctx); err != nil {
		klog.Fatalf("unexpected error: %v", err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
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
		return fmt.Errorf("error reading config file: %v", err)
	}

	flags := flag.NewFlagSet("", flag.ExitOnError)
	options.AddFlags(flags)

	klog.InitFlags(flags)

	if options.LogLevel != nil {
		flags.Set("v", strconv.Itoa(*options.LogLevel))
	}

	flags.Parse(os.Args)

	config, err := rest.InClusterConfig()
	if err != nil {
		return fmt.Errorf("error building client configuration: %v", err)
	}

	kubeClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("error building REST client: %v", err)
	}

	var matcher func(node *v1.Node) bool
	if nodeName := os.Getenv("NODE_NAME"); nodeName != "" {
		// Passing NODE_NAME via downward API is preferred
		klog.Infof("will match node on name=%q", nodeName)
		matcher = func(node *v1.Node) bool {
			return node.Name == nodeName
		}
	} else if options.MachineIDPath != "" {
		klog.Warningf("using MachineIDPath is deprecated - prefer passing NODE_NAME via downward API")

		b, err := ioutil.ReadFile(options.MachineIDPath)
		if err != nil {
			return fmt.Errorf("error reading machine-id file %q: %v", options.MachineIDPath, err)
		}
		machineID := string(b)
		machineID = strings.TrimSpace(machineID)

		klog.Infof("will match node on machineid=%q", machineID)
		matcher = func(node *v1.Node) bool {
			return node.Status.NodeInfo.MachineID == machineID
		}
	} else if options.SystemUUIDPath != "" {
		klog.Warningf("using SystemUUIDPath is deprecated - prefer passing NODE_NAME via downward API")

		b, err := ioutil.ReadFile(options.SystemUUIDPath)
		if err != nil {
			return fmt.Errorf("error reading system-uuid file %q: %v", options.SystemUUIDPath, err)
		}
		systemUUID := string(b)
		systemUUID = strings.TrimSpace(systemUUID)

		// If the BIOS isn't correctly configured, we'll
		if systemUUID == "03000200-0400-0500-0006-000700080009" {
			return fmt.Errorf("detected well-known invalid system-uuid 03000200-0400-0500-0006-000700080009")
		}

		klog.Infof("will match node on systemUUID=%q", systemUUID)
		matcher = func(node *v1.Node) bool {
			return node.Status.NodeInfo.SystemUUID == systemUUID
		}
	} else if options.BootIDPath != "" {
		klog.Warningf("using BootIDPath is deprecated - prefer passing NODE_NAME via downward API")

		b, err := ioutil.ReadFile(options.BootIDPath)
		if err != nil {
			return fmt.Errorf("error reading boot-id file %q: %v", options.BootIDPath, err)
		}
		bootID := string(b)
		bootID = strings.TrimSpace(bootID)

		klog.Infof("will match node on bootID=%q", bootID)
		matcher = func(node *v1.Node) bool {
			return node.Status.NodeInfo.BootID == bootID
		}
	} else {
		klog.Warningf("using NodeName is deprecated - prefer passing NODE_NAME via downward API")

		matchNodeName := options.NodeName
		if matchNodeName == "" {
			hostname, err := os.Hostname()
			if err != nil {
				return fmt.Errorf("error getting hostname: %v", err)
			}
			klog.Infof("Using hostname as node name: %q", hostname)
			matchNodeName = hostname
		}
		klog.Infof("will match node on name=%q", matchNodeName)
		matcher = func(node *v1.Node) bool {
			return node.Name == matchNodeName
		}
	}

	nodeMap := routing.NewNodeMap(matcher)

	targetLinkName := options.TargetLinkName
	if targetLinkName == "" {
		targetLinkName, err = findTargetLink()
		if targetLinkName == "" || err != nil {
			return fmt.Errorf("unable to determine network device; pass --target to specify: %v", err)
		}
	}

	var provider routing.Provider
	switch options.Provider {
	case "layer2":
		provider, err = layer2.NewLayer2RoutingProvider(targetLinkName)
	case "gre":
		provider, err = gre.NewGreRoutingProvider()
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
			return fmt.Errorf("unknown ipsec-encryption: %v", options.IPSEC.Encryption)
		}
		switch options.IPSEC.Authentication {
		case "none":
			authenticationStrategy = &ipsec.PlaintextAuthenticationStrategy{}
		case "sha1":
			authenticationStrategy = &ipsec.HmacSha1AuthenticationStrategy{}
		default:
			return fmt.Errorf("unknown ipsec-authentication: %v", options.IPSEC.Authentication)
		}
		switch options.IPSEC.Encapsulation {
		case "udp":
			encapsulationStrategy = &ipsec.UdpEncapsulationStrategy{}
		case "esp":
			encapsulationStrategy = &ipsec.EspEncapsulationStrategy{}
		default:
			return fmt.Errorf("unknown ipsec-encapsulation: %v", options.IPSEC.Encapsulation)
		}

		var ipsecProvider *ipsec.IpsecRoutingProvider
		ipsecProvider, err = ipsec.NewIpsecRoutingProvider(authenticationStrategy, encryptionStrategy, encapsulationStrategy)
		if err == nil {
			// TODO: This is only because state update is not working
			klog.Warningf("TODO Doing ip xfrm flush; remove!!")
			err := ipsecProvider.Flush()
			if err != nil {
				return fmt.Errorf("cannot flush tables")
			}
		}
		provider = ipsecProvider

	default:
		return fmt.Errorf("provider not known: %q", options.Provider)
	}

	if err != nil {
		return fmt.Errorf("failed to build provider %q: %v", options.Provider, err)
	}

	//c, err := newRouteController(kubeClient, *resyncPeriod, *nodeName, bootID, systemUUID, machineID, provider)
	//if err != nil {
	//	return fmt.Errorf("%v", err)
	//}

	var cniWriter cni.ConfigWriter
	if options.CNIConfigPath != "" {
		cniWriter = &cni.SimpleConfigWriter{Path: options.CNIConfigPath}
	}

	c, err := watchers.NewNodeController(kubeClient, nodeMap)
	if err != nil {
		return fmt.Errorf("Failed to build node controller: %v", err)
	}
	go c.Run(ctx)

	rc, err := routing.NewController(kubeClient, nodeMap, provider, cniWriter)
	if err != nil {
		return fmt.Errorf("Failed to build routing controller: %v", err)
	}
	go rc.Run(ctx)
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
//	klog.Fatal(server.ListenAndServe())
//}

func handleSigterm(c *watchers.NodeController) {
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGTERM)
	<-signalChan
	klog.Infof("Received SIGTERM, shutting down")

	exitCode := 0
	if err := c.Stop(); err != nil {
		klog.Infof("Error during shutdown %v", err)
		exitCode = 1
	}
	klog.Infof("Exiting with %v", exitCode)
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
			klog.V(2).Infof("Ignoring interface %s - loopback", name)
			continue
		}

		// Not a lot else to go on...
		if !strings.HasPrefix(name, "eth") && !strings.HasPrefix(name, "en") {
			klog.V(2).Infof("Ignoring interface %s - name does not look like ethernet device", name)
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
