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
	"fmt"
	"github.com/ghodss/yaml"
	"github.com/spf13/pflag"
	"io/ioutil"
	"os"
	"time"
)

type Options struct {
	Provider       string `json:"provider"`
	TargetLinkName string `json:"targetLinkName"`

	ResyncPeriod time.Duration `json:"resyncPeriod"`

	NodeName       string `json:"nodeName"`
	MachineIDPath  string `json:"machineIDPath"`
	SystemUUIDPath string `json:"systemUUIDPath"`
	BootIDPath     string `json:"bootIDPath"`

	IPSEC IPSECOptions `json:"ipsec"`

	LogLevel *int `json:"logLevel"`
}

type IPSECOptions struct {
	Authentication string `json:"authentication"`
	Encryption     string `json:"encryption"`
	Encapsulation  string `json:"encapsulation"`
}

func (o *Options) InitDefaults() {
	logLevel := 1
	o.LogLevel = &logLevel

	o.ResyncPeriod = 30 * time.Second
	o.TargetLinkName = "eth0"
	o.Provider = "vxlan"

	o.SystemUUIDPath = "/sys/class/dmi/id/product_uuid"

	o.IPSEC.Authentication = "sha1"
	o.IPSEC.Encapsulation = "udp"
	o.IPSEC.Encryption = "aes"
}

func (options *Options) AddFlags(flags *pflag.FlagSet) {
	flags.DurationVar(&options.ResyncPeriod, "sync-period", options.ResyncPeriod,
		`Relist and confirm cloud resources this often.`)

	//healthzPort = flags.Int("healthz-port", healthPort, "port for healthz endpoint.")

	//kubeConfig = flags.String("kubeconfig", "", "Path to kubeconfig file with authorization information.")

	flags.StringVar(&options.NodeName, "node-name", options.NodeName, "name of this node")

	flags.StringVar(&options.MachineIDPath, "machine-id", options.MachineIDPath, "path to file containing machine id (as set in node status)")
	flags.StringVar(&options.SystemUUIDPath, "system-uuid", options.SystemUUIDPath, "path to file containing system-uuid (as set in node status)")
	flags.StringVar(&options.BootIDPath, "boot-id", options.BootIDPath, "path to file containing boot-id (as set in node status)")

	flags.StringVar(&options.Provider, "provider", options.Provider, "route backend to use")

	flags.StringVar(&options.TargetLinkName, "target", options.TargetLinkName, "network link to use for actual packet transport")

	flags.StringVar(&options.IPSEC.Encryption, "ipsec-encryption", options.IPSEC.Encryption, "encryption method to use (for IPSEC)")
	flags.StringVar(&options.IPSEC.Authentication, "ipsec-authentication", options.IPSEC.Authentication, "authentication method to use (for IPSEC)")
	flags.StringVar(&options.IPSEC.Encapsulation, "ipsec-encapsulation", options.IPSEC.Encapsulation, "encapsulation method to use (for IPSEC)")

	// I can't figure out how to get a serviceaccount in a manifest-controlled pod
	//inCluster = flags.Bool("running-in-cluster", true,
	//	`Optional, if this controller is running in a kubernetes cluster, use the
	//	 pod secrets for creating a Kubernetes client.`)

	//flags.BoolVar(options.Profiling, "profiling", options.Profiling, `Enable profiling via web interface host:port/debug/pprof/`)
}

func (options *Options) LoadFrom(p string) error {
	data, err := ioutil.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			return err
		}
		return fmt.Errorf("error reading file %q: %v", p, err)
	}

	err = yaml.Unmarshal(data, options)
	if err != nil {
		return fmt.Errorf("error parsing file %q: %v", p, err)
	}
	return nil
}
