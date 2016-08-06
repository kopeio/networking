package ipsecrouting

import (
	"github.com/kopeio/route-controller/pkg/routing"
	"github.com/vishvananda/netlink"
	"net"
)

type EncapsulationStrategy interface {
	Apply(s *netlink.XfrmState, src *routing.NodeInfo, dest *routing.NodeInfo)
}

type UdpEncapsulationStrategy struct {
}

var _ EncapsulationStrategy = &UdpEncapsulationStrategy{}

func (e *UdpEncapsulationStrategy) Apply(s *netlink.XfrmState, src *routing.NodeInfo, dest *routing.NodeInfo) {
	s.Encap = &netlink.XfrmStateEncap{
		Type:            netlink.XFRM_ENCAP_ESPINUDP,
		SrcPort:         4500,
		DstPort:         4500,
		OriginalAddress: net.IPv4(0, 0, 0, 0),
	}
}

type EspEncapsulationStrategy struct {
}

var _ EncapsulationStrategy = &EspEncapsulationStrategy{}

func (e *EspEncapsulationStrategy) Apply(s *netlink.XfrmState, src *routing.NodeInfo, dest *routing.NodeInfo) {
}
