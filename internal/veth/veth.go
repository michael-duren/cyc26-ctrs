package veth

import (
	"errors"
	"fmt"

	"github.com/vishvananda/netlink"
)

const (
	veth1 = "veth1"
	veth2 = "veth2"
)

func CreateParentVeth(cpid int) error {
	veth := &netlink.Veth{
		LinkAttrs: netlink.LinkAttrs{
			Name: veth1,
		},
		PeerName:      veth2,
		PeerNamespace: netlink.NsPid(cpid),
	}

	CleanupVEth()
	if err := netlink.LinkAdd(veth); err != nil {
		return fmt.Errorf("create veth: %w", err)
	}

	link, err := netlink.LinkByName(veth1)
	if err != nil {
		return fmt.Errorf("resolve veth1 link: %w", err)
	}

	addr, err := netlink.ParseAddr("10.0.0.1/24")
	if err != nil {
		return fmt.Errorf("parse addr: %w", err)
	}
	if err := netlink.AddrAdd(link, addr); err != nil {
		return fmt.Errorf("add addr: %w", err)
	}
	if err := netlink.LinkSetUp(link); err != nil {
		return fmt.Errorf("set link to up: %w", err)
	}

	return nil
}

// CreateChildVeth assumed [CreateParentVeth] has been called
// finishes setting up the veth connection from the pov of
// the child
func CreateChildVeth() error {
	link, err := netlink.LinkByName(veth2)
	if err != nil {
		return fmt.Errorf("resolve netlink from veth2: %w", err)
	}
	addr, err := netlink.ParseAddr("10.0.0.2/24")
	if err != nil {
		return fmt.Errorf("parse addr: %w", err)
	}
	if err := netlink.AddrAdd(link, addr); err != nil {
		return fmt.Errorf("add addr veth2: %w", err)
	}
	if err := netlink.LinkSetUp(link); err != nil {
		return fmt.Errorf("set veth2 ↑: %w", err)
	}

	// NOTE: child p loopback is down by default node process would fail
	link, err = netlink.LinkByName("lo")
	if err != nil {
		return fmt.Errorf("resolve link from loopback: %w", err)
	}
	if err := netlink.LinkSetUp(link); err != nil {
		return fmt.Errorf("set loopback up: %w", err)
	}

	return nil
}

func CleanupVEth() {
	link, err := netlink.LinkByName(veth1)
	if err != nil {
		terr, ok := errors.AsType[netlink.LinkNotFoundError](err)
		if ok {
			return
		}
		fmt.Println("cleanup: lookup veth1", err, terr)
		return
	}
	if err := netlink.LinkDel(link); err != nil {
		fmt.Println("cleanup: deleting veth1", err)
	}
}
