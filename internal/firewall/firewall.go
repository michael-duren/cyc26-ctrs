package firewall

import (
	"fmt"
	"os/exec"
)

// table is our own nftables table, same idea as docker's DOCKER chain:
// keep all our rules in one namespace we fully own, so cleanup is one delete
const table = "boxes-demo"

// Block rejects TCP traffic from this host to containerIP on every port
// except allowedPort. Mirrors docker's publish model: the only way in is
// the published port.
func Block(containerIP, allowedPort string) error {
	cmds := [][]string{
		// idempotent: add table/chain do nothing if they already exist,
		// flush clears rules left over from a previous run
		{"add", "table", "inet", table},
		{"flush", "table", "inet", table},
		{"add", "chain", "inet", table, "output",
			"{", "type", "filter", "hook", "output", "priority", "0", ";", "}"},
		{"add", "rule", "inet", table, "output",
			"ip", "daddr", containerIP,
			"tcp", "dport", "!=", allowedPort,
			"reject", "with", "tcp", "reset"},
	}
	for _, args := range cmds {
		if out, err := exec.Command("nft", args...).CombinedOutput(); err != nil {
			return fmt.Errorf("nft %v: %s: %w", args, out, err)
		}
	}
	return nil
}

// Cleanup deletes the whole table, removing every rule in it.
func Cleanup() error {
	if out, err := exec.Command("nft", "delete", "table", "inet", table).CombinedOutput(); err != nil {
		return fmt.Errorf("nft delete table: %s: %w", out, err)
	}
	return nil
}
