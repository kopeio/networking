package netutil

import (
	"bytes"
	"net"
)

func ipnetEqual(a *net.IPNet, e *net.IPNet) bool {
	if a == nil {
		return e == nil
	}
	if e == nil {
		return a == nil
	}
	if !bytes.Equal(a.Mask, e.Mask) {
		return false
	}
	if !ipEqual(a.IP, e.IP) {
		return false
	}
	return true
}

func ipEqual(a net.IP, e net.IP) bool {
	if a == nil {
		return e == nil
	}
	if e == nil {
		return a == nil
	}
	return a.Equal(e)
}
