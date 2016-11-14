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
	"github.com/golang/glog"
	"github.com/spf13/pflag"
	"io/ioutil"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/rest"
	"kope.io/krouton/pkg/routing"
	"kope.io/krouton/pkg/routing/ipsec"
	"kope.io/krouton/pkg/routing/layer2"
	"kope.io/krouton/pkg/routing/vxlan"
	"kope.io/krouton/pkg/watchers"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

//const (
//	healthPort = 10249
//)

var (
	// value overwritten during build. This can be used to resolve issues.
	version = "0.5"
	gitRepo = "https://github.com/kopeio/krouton"

	flags = pflag.NewFlagSet("", pflag.ExitOnError)

	resyncPeriod = flags.Duration("sync-period", 30*time.Second,
		`Relist and confirm cloud resources this often.`)

	//healthzPort = flags.Int("healthz-port", healthPort, "port for healthz endpoint.")

	//kubeConfig = flags.String("kubeconfig", "", "Path to kubeconfig file with authorization information.")

	nodeName       = flags.String("node-name", "", "name of this node")
	machineIDPath  = flags.String("machine-id", "", "path to file containing machine id (as set in node status)")
	systemUUIDPath = flags.String("system-uuid", "", "path to file containing system-uuid (as set in node status)")
	bootIDPath     = flags.String("boot-id", "", "path to file containing boot-id (as set in node status)")
	providerID     = flags.String("provider", "gre", "route backend to use")

	targetLinkName = flags.String("target", "eth0", "network link to use for actual packet transport")

	ipsecEncryption     = flags.String("ipsec-encryption", "aes", "encryption method to use (for IPSEC)")
	ipsecAuthentication = flags.String("ipsec-authentication", "sha1", "authentication method to use (for IPSEC)")
	ipsecEncapsulation  = flags.String("ipsec-encapsulation", "udp", "encapsulation method to use (for IPSEC)")

	// I can't figure out how to get a serviceaccount in a manifest-controlled pod
	//inCluster = flags.Bool("running-in-cluster", true,
	//	`Optional, if this controller is running in a kubernetes cluster, use the
	//	 pod secrets for creating a Kubernetes client.`)

	profiling = flags.Bool("profiling", true, `Enable profiling via web interface host:port/debug/pprof/`)
)

func main() {
	// Trick to avoid 'logging before flag.Parse' warning
	goflag.CommandLine.Parse([]string{})

	goflag.Set("logtostderr", "true")

	flags.AddGoFlagSet(goflag.CommandLine)

	flags.Parse(os.Args)

	glog.Infof("Using build: %v - %v", gitRepo, version)

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
	if *machineIDPath != "" {
		b, err := ioutil.ReadFile(*machineIDPath)
		if err != nil {
			glog.Fatalf("error reading machine-id file %q: %v", *machineIDPath, err)
		}
		machineID := string(b)
		machineID = strings.TrimSpace(machineID)

		matcher = func(node *v1.Node) bool {
			return node.Status.NodeInfo.MachineID == machineID
		}
	} else if *systemUUIDPath != "" {
		b, err := ioutil.ReadFile(*systemUUIDPath)
		if err != nil {
			glog.Fatalf("error reading system-uuid file %q: %v", *systemUUIDPath, err)
		}
		systemUUID := string(b)
		systemUUID = strings.TrimSpace(systemUUID)
		matcher = func(node *v1.Node) bool {
			return node.Status.NodeInfo.SystemUUID == systemUUID
		}
	} else if *bootIDPath != "" {
		b, err := ioutil.ReadFile(*bootIDPath)
		if err != nil {
			glog.Fatalf("error reading boot-id file %q: %v", *bootIDPath, err)
		}
		bootID := string(b)
		bootID = strings.TrimSpace(bootID)
		matcher = func(node *v1.Node) bool {
			return node.Status.NodeInfo.BootID == bootID
		}
	} else {
		matchNodeName := *nodeName
		if matchNodeName == "" {
			hostname, err := os.Hostname()
			if err != nil {
				glog.Fatalf("error getting hostname: %v", err)
			}
			glog.Infof("Using hostname as node name: %q", hostname)
			matchNodeName = hostname
		}
		matcher = func(node *v1.Node) bool {
			return node.Name == matchNodeName
		}
	}

	nodeMap := routing.NewNodeMap(matcher)

	var provider routing.Provider
	switch *providerID {
	case "layer2":
		provider, err = layer2.NewLayer2RoutingProvider(*targetLinkName)
	case "gre":
		glog.Fatalf("GRE temporarily not enabled - until patch goes upstream")
		// provider, err = gre.NewGreRoutingProvider()
	case "vxlan":
		glog.Errorf("assuming overlay is 100.96.0.0/12")
		_, overlayCIDR, _ := net.ParseCIDR("100.96.0.0/12")
		provider, err = vxlan.NewVxlanRoutingProvider(overlayCIDR, *targetLinkName)
	case "ipsec":
		var authenticationStrategy ipsec.AuthenticationStrategy
		var encryptionStrategy ipsec.EncryptionStrategy
		var encapsulationStrategy ipsec.EncapsulationStrategy

		switch *ipsecEncryption {
		case "none":
			encryptionStrategy = &ipsec.PlaintextEncryptionStrategy{}
		case "aes":
			encryptionStrategy = &ipsec.AesEncryptionStrategy{}
		default:
			glog.Fatalf("unknown ipsec-encryption: %v", *ipsecEncryption)
		}
		switch *ipsecAuthentication {
		case "none":
			authenticationStrategy = &ipsec.PlaintextAuthenticationStrategy{}
		case "sha1":
			authenticationStrategy = &ipsec.HmacSha1AuthenticationStrategy{}
		default:
			glog.Fatalf("unknown ipsec-authentication: %v", *ipsecAuthentication)
		}
		switch *ipsecEncapsulation {
		case "udp":
			encapsulationStrategy = &ipsec.UdpEncapsulationStrategy{}
		case "esp":
			encapsulationStrategy = &ipsec.EspEncapsulationStrategy{}
		default:
			glog.Fatalf("unknown ipsec-encapsulation: %v", *ipsecEncapsulation)
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
		glog.Fatalf("provider not known: %q", *providerID)
	}

	if err != nil {
		glog.Fatalf("failed to build provider %q: %v", *providerID, err)
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

	rc, err := routing.NewController(nodeMap, provider)
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
