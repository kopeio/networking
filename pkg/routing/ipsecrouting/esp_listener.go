package ipsecrouting

import (
	"encoding/binary"
	"fmt"
	"syscall"
)

// See http://lxr.free-electrons.com/source/include/uapi/linux/udp.h
const UDP_ENCAP = 100
const UDP_ENCAP_ESPINUDP_NON_IKE = 1
const UDP_ENCAP_ESPINUDP = 2

// http://lxr.free-electrons.com/source/include/uapi/linux/ipsec.h
const (
	IPSEC_DIR_ANY      = 0
	IPSEC_DIR_INBOUND  = 1
	IPSEC_DIR_OUTBOUND = 2
	IPSEC_DIR_FWD      = 3
	IPSEC_DIR_MAX      = 4
	IPSEC_DIR_INVALID  = 5

	IPSEC_POLICY_DISCARD = 0
	IPSEC_POLICY_NONE    = 1
	IPSEC_POLICY_IPSEC   = 2
	IPSEC_POLICY_ENTRUST = 3
	IPSEC_POLICY_BYPASS  = 4
)

// http://lxr.free-electrons.com/source/include/uapi/linux/pfkeyv2.h
const (
	/* Extension Header values */
	SADB_EXT_RESERVED          = 0
	SADB_EXT_SA                = 1
	SADB_EXT_LIFETIME_CURRENT  = 2
	SADB_EXT_LIFETIME_HARD     = 3
	SADB_EXT_LIFETIME_SOFT     = 4
	SADB_EXT_ADDRESS_SRC       = 5
	SADB_EXT_ADDRESS_DST       = 6
	SADB_EXT_ADDRESS_PROXY     = 7
	SADB_EXT_KEY_AUTH          = 8
	SADB_EXT_KEY_ENCRYPT       = 9
	SADB_EXT_IDENTITY_SRC      = 10
	SADB_EXT_IDENTITY_DST      = 11
	SADB_EXT_SENSITIVITY       = 12
	SADB_EXT_PROPOSAL          = 13
	SADB_EXT_SUPPORTED_AUTH    = 14
	SADB_EXT_SUPPORTED_ENCRYPT = 15
	SADB_EXT_SPIRANGE          = 16
	SADB_X_EXT_KMPRIVATE       = 17
	SADB_X_EXT_POLICY          = 18
	SADB_X_EXT_SA2             = 19
	/* The next four entries are for setting up NAT Traversal */
	SADB_X_EXT_NAT_T_TYPE  = 20
	SADB_X_EXT_NAT_T_SPORT = 21
	SADB_X_EXT_NAT_T_DPORT = 22
	SADB_X_EXT_NAT_T_OA    = 23
	SADB_X_EXT_SEC_CTX     = 24
	/* Used with MIGRATE to pass @ to IKE for negotiation */
	SADB_X_EXT_KMADDRESS = 25
	SADB_X_EXT_FILTER    = 26
)

type UDPEncapListener struct {
	fd int
}

type SadbXPolicy struct {
	len            uint16
	ExtType        uint16
	Type           uint16
	Direction      uint8
	Reserved       uint8
	PolicyId       uint32
	PolicyPriority uint32
}

func (s *SadbXPolicy) Encode() []byte {
	b := make([]byte, 16, 16)

	o := binary.LittleEndian

	o.PutUint16(b[0:2], 2 /* length, in 8 byte counts */)
	o.PutUint16(b[2:4], s.ExtType)
	o.PutUint16(b[4:6], s.Type)
	b[6] = s.Direction
	b[7] = s.Reserved
	o.PutUint32(b[8:12], s.PolicyId)
	o.PutUint32(b[12:16], s.PolicyPriority)

	return b
}

func NewUDPEncapListener(port int) (*UDPEncapListener, error) {
	fd, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_DGRAM, 0)
	if err != nil {
		return nil, fmt.Errorf("error creating socket: %v", err)
	}

	var policy SadbXPolicy
	policy.ExtType = SADB_X_EXT_POLICY
	policy.Type = IPSEC_POLICY_BYPASS
	policy.Direction = IPSEC_DIR_OUTBOUND

	policyBytes := policy.Encode()

	err = syscall.SetsockoptString(fd, syscall.SOL_IP, syscall.IP_IPSEC_POLICY, string(policyBytes))
	if err != nil {
		return nil, fmt.Errorf("error setting IP_IPSEC_POLICY socket option (outbound): %v", err)
	}

	policy.ExtType = SADB_X_EXT_POLICY
	policy.Type = IPSEC_POLICY_BYPASS
	policy.Direction = IPSEC_DIR_INBOUND
	policyBytes = policy.Encode()

	err = syscall.SetsockoptString(fd, syscall.SOL_IP, syscall.IP_IPSEC_POLICY, string(policyBytes))
	if err != nil {
		return nil, fmt.Errorf("error setting IP_IPSEC_POLICY socket option (inbound): %v", err)
	}

	err = syscall.SetsockoptInt(fd, syscall.IPPROTO_UDP, UDP_ENCAP, UDP_ENCAP_ESPINUDP)
	if err != nil {
		return nil, fmt.Errorf("error setting UDP_ENCAP=UDP_ENCAP_ESPINUDP socket option: %v", err)
	}

	sa := &syscall.SockaddrInet4{}
	sa.Port = port

	err = syscall.Bind(fd, sa)
	if err != nil {
		return nil, fmt.Errorf("error binding socket: %v", err)
	}

	return &UDPEncapListener{fd: fd}, nil
}

func (l *UDPEncapListener) Close() error {
	if l.fd == 0 {
		return nil
	}
	err := syscall.Close(l.fd)
	if err != nil {
		return fmt.Errorf("error closing socket: %v", err)
	}
	l.fd = 0
	return nil
}
