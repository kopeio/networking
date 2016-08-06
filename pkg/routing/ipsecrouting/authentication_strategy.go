package ipsecrouting

import (
	"github.com/kopeio/route-controller/pkg/routing"
	"github.com/vishvananda/netlink"
)

type AuthenticationStrategy interface {
	Apply(s *netlink.XfrmState, src *routing.NodeInfo, dest *routing.NodeInfo)
	UseAH() bool
}

type HmacSha1AuthenticationStrategy struct {
}

var _ AuthenticationStrategy = &HmacSha1AuthenticationStrategy{}

func (p *HmacSha1AuthenticationStrategy) Apply(s *netlink.XfrmState, src *routing.NodeInfo, dest *routing.NodeInfo) {
	ahKey := []byte{0x1, 0x2, 0x3, 0x4, 0x5, 0x6, 0x7, 0x8, 0x9, 0xa}

	// http://lxr.free-electrons.com/source/net/xfrm/xfrm_algo.c#L219
	s.Auth = &netlink.XfrmStateAlgo{
		Name:        "hmac(sha1)",
		Key:         ahKey,
		TruncateLen: 96,
	}
}

func (p *HmacSha1AuthenticationStrategy) UseAH() bool {
	return true
}

type PlaintextAuthenticationStrategy struct {
}

var _ AuthenticationStrategy = &PlaintextAuthenticationStrategy{}

func (p *PlaintextAuthenticationStrategy) Apply(s *netlink.XfrmState, src *routing.NodeInfo, dest *routing.NodeInfo) {
	s.Auth = &netlink.XfrmStateAlgo{
		Name: "digest_null",
	}
}

func (p *PlaintextAuthenticationStrategy) UseAH() bool {
	return true
}
