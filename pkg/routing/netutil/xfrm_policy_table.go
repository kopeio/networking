package netutil

import (
	"fmt"

	"github.com/golang/glog"
	"github.com/vishvananda/netlink"
	"kope.io/krouton/pkg/util"
)

type XfrmPolicyTable struct {
}

func (p *XfrmPolicyTable) Flush() error {
	return netlink.XfrmPolicyFlush()
}

func (t *XfrmPolicyTable) Ensure(expected []*netlink.XfrmPolicy) error {
	actual, err := netlink.XfrmPolicyList(netlink.FAMILY_ALL)
	if err != nil {
		return fmt.Errorf("error listing xfrm policies: %v", err)
	}

	for _, p := range actual {
		glog.Infof("Actual Policy: %v", util.AsJsonString(p))
	}

	var create []*netlink.XfrmPolicy
	var remove []*netlink.XfrmPolicy
	var updates []*netlink.XfrmPolicy

	actualMatched := make([]bool, len(actual), len(actual))
	for _, e := range expected {
		var a *netlink.XfrmPolicy
		// TODO: Bucket by 'key' so we are not O(N^2)
		for i := range actual {
			p := &actual[i]
			if xfrmPolicyMatches(p, e) {
				if a != nil {
					glog.Warningf("Found duplicate matching policies: %v and %v", util.AsJsonString(p), util.AsJsonString(a))
				}
				a = p
				actualMatched[i] = true
			}
		}

		if a == nil {
			create = append(create, e)
			continue
		}

		// Avoid spurious changes
		e.Index = a.Index
		if !xfrmPolicyEqual(a, e) {
			glog.Infof("Policy changed:\n\t%s\n\t%s", a, e)
			updates = append(updates, e)
		}
	}

	for i := range actual {
		if actualMatched[i] {
			continue
		}
		remove = append(remove, &actual[i])
	}

	if len(create) != 0 {
		for _, p := range create {
			glog.Infof("creating policy %v", util.AsJsonString(p))
			err := netlink.XfrmPolicyAdd(p)
			if err != nil {
				return fmt.Errorf("error creating policy: %v", err)
			}
		}
	}

	if len(updates) != 0 {
		for _, p := range updates {
			glog.Infof("updating policy %v", util.AsJsonString(p))
			err := netlink.XfrmPolicyUpdate(p)
			if err != nil {
				return fmt.Errorf("error updating policy: %v", err)
			}
		}
	}

	if len(remove) != 0 {
		for _, p := range remove {
			glog.Infof("removing policy %v", util.AsJsonString(p))
			err := netlink.XfrmPolicyDel(p)
			if err != nil {
				return fmt.Errorf("error removing policy: %v", err)
			}
		}
	}

	return nil
}

func xfrmPolicyEqual(l *netlink.XfrmPolicy, r *netlink.XfrmPolicy) bool {
	if l.Dir != r.Dir {
		return false
	}
	if l.Priority != r.Priority {
		return false
	}
	if l.Index != r.Index {
		return false
	}
	if l.SrcPort != r.SrcPort {
		return false
	}
	if l.DstPort != r.DstPort {
		return false
	}
	if l.Proto != r.Proto {
		return false
	}
	if !ipnetEqual(l.Dst, r.Dst) {
		return false
	}
	if !ipnetEqual(l.Src, r.Src) {
		return false
	}
	if !xfrmMarkEqual(l.Mark, r.Mark) {
		return false
	}

	if len(l.Tmpls) != len(r.Tmpls) {
		return false
	}
	for i := range l.Tmpls {
		if !xfrmPolicyTmplEqual(&l.Tmpls[i], &r.Tmpls[i]) {
			return false
		}
	}

	return true
}

func xfrmMarkEqual(l *netlink.XfrmMark, r *netlink.XfrmMark) bool {
	if l == nil || r == nil {
		return (r == nil) == (l == nil)
	}
	return l.Mask == r.Mask && l.Value == r.Value
}

func xfrmPolicyTmplEqual(l *netlink.XfrmPolicyTmpl, r *netlink.XfrmPolicyTmpl) bool {
	if l == nil || r == nil {
		return (r == nil) == (l == nil)
	}
	if l.Proto != r.Proto {
		return false
	}
	if l.Mode != r.Mode {
		return false
	}
	if l.Spi != r.Spi {
		return false
	}
	if l.Reqid != r.Reqid {
		return false
	}
	if !l.Dst.Equal(r.Dst) {
		return false
	}
	if !l.Src.Equal(r.Src) {
		return false
	}
	return true
}

func xfrmPolicyMatches(a *netlink.XfrmPolicy, e *netlink.XfrmPolicy) bool {
	if a.Dir != e.Dir {
		return false
	}
	if a.SrcPort != e.SrcPort {
		return false
	}
	if a.DstPort != e.DstPort {
		return false
	}
	if a.Proto != e.Proto {
		return false
	}
	if !ipnetEqual(a.Dst, e.Dst) {
		return false
	}
	if !ipnetEqual(a.Src, e.Src) {
		return false
	}
	return true
}
