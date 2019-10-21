package cni

import "net"

type ConfigWriter interface {
	WriteCNIConfig(podCIDR *net.IPNet) error
}
