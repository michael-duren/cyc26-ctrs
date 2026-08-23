package cgroup

import (
	"fmt"
	"os"
	"path/filepath"
)

const (
	cgrouppath = "/sys/fs/cgroup/user.slice/user-1000.slice/boxes.service"
	ctrpath    = cgrouppath + "/ctr1"
	fileperms  = 0o755
)

func Cg() error {
	if err := os.MkdirAll(cgrouppath, fileperms); err != nil {
		return fmt.Errorf("create service cgroup: %w", err)
	}
	if err := os.WriteFile(
		filepath.Join(cgrouppath, "cgroup.subtree_control"), []byte("+pids"), fileperms); err != nil {
		return fmt.Errorf("enable pids controller: %w", err)
	}

	if err := os.MkdirAll(ctrpath, fileperms); err != nil {
		return fmt.Errorf("create ctr cgroup: %w", err)
	}
	if err := os.WriteFile(filepath.Join(ctrpath, "pids.max"), []byte("50"), fileperms); err != nil {
		return fmt.Errorf("write pids.max: %w", err)
	}

	return nil
}

func CleanupCg() error {
	fmt.Println("running in cleanup")
	ctrpath := filepath.Join(cgrouppath, "ctr1")

	if err := os.Remove(ctrpath); err != nil {
		return err
	}

	return os.Remove(cgrouppath)
}
