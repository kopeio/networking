package ipsec

import (
	"github.com/vishvananda/netlink"
	"kope.io/krouton/pkg/routing"
)

type EncryptionStrategy interface {
	Apply(s *netlink.XfrmState, src *routing.NodeInfo, dest *routing.NodeInfo)
	UseESP() bool
}

type AesEncryptionStrategy struct {
}

var _ EncryptionStrategy = &AesEncryptionStrategy{}

func (e *AesEncryptionStrategy) Apply(s *netlink.XfrmState, src *routing.NodeInfo, dest *routing.NodeInfo) {
	espKey := []byte{0x4, 0x5, 0x9, 0x4, 0x3, 0x6, 0x7, 0x8, 0x9, 0xa,
		0x4, 0x5, 0x9, 0x4, 0x3, 0x6, 0x7, 0x8, 0x9, 0xa}
	s.Crypt = &netlink.XfrmStateAlgo{
		Name: "rfc3686(ctr(aes))",
		Key:  espKey,
	}

	//http://lxr.free-electrons.com/source/net/xfrm/xfrm_algo.c#L540
}

func (e *AesEncryptionStrategy) UseESP() bool {
	return true
}

type PlaintextEncryptionStrategy struct {
}

var _ EncryptionStrategy = &PlaintextEncryptionStrategy{}

func (e *PlaintextEncryptionStrategy) Apply(s *netlink.XfrmState, src *routing.NodeInfo, dest *routing.NodeInfo) {
	s.Crypt = &netlink.XfrmStateAlgo{
		Name: "ecb(cipher_null)",
	}
}

func (e *PlaintextEncryptionStrategy) UseESP() bool {
	return true
}
