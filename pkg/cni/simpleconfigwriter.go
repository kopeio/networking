package cni

import (
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"strings"

	"k8s.io/klog/v2"
)

// SimpleConfigWriter writes to a single cni config file
type SimpleConfigWriter struct {
	Path string
}

const cniConfig = `
{
  "cniVersion":      "0.3.1",
  "name":            "k8s-pod-network",
  "type":            "bridge",
  "bridge":          "kopeio",
  "isDefaultGateway": true,
  "ipMasq": true,
  "ipam": {
    "type":   "host-local",
    "name":   "kopeio",
    "subnet": "{{PodCIDR}}"
  }
}
`

func (w *SimpleConfigWriter) WriteCNIConfig(podCIDR *net.IPNet) error {
	b, err := ioutil.ReadFile(w.Path)
	if err != nil {
		if os.IsNotExist(err) {
			b = nil
		} else {
			return fmt.Errorf("error reading cni config %s: %v", w.Path, err)
		}
	}

	podCIDRString := ""
	if podCIDR != nil {
		podCIDRString = podCIDR.String()
	}

	expected := cniConfig
	expected = strings.ReplaceAll(expected, "{{PodCIDR}}", podCIDRString)

	existing := string(b)
	if existing == expected {
		klog.V(4).Infof("cni config at %s matches expected", w.Path)
		return nil
	}

	if err := ioutil.WriteFile(w.Path, []byte(expected), 0644); err != nil {
		return fmt.Errorf("error writing cni config %s: %v", w.Path, err)
	}
	klog.Infof("wrote cni config to %s", w.Path)

	return nil
}
