// provide functions to handle operations around rootfs
package root

import (
	"os"
	"path/filepath"
	"syscall"
)

func PivotRoot(newroot string, fileperms os.FileMode) error {
	// pivot_root requires newroot to be a mount point; bind it onto itself
	if err := syscall.Mount(newroot, newroot, "", syscall.MS_BIND|syscall.MS_REC, ""); err != nil {
		return err
	}

	// Mount the new /proc NOW, while the old root's fully-visible /proc is
	// still attached. In a userns the kernel only allows a fresh procfs if it
	// can validate it against an existing fully-visible proc; once we detach
	// the old root below, that reference is gone and the mount EPERMs.
	proc := filepath.Join(newroot, "proc")
	if err := os.MkdirAll(proc, fileperms); err != nil {
		return err
	}
	if err := syscall.Mount("proc", proc, "proc", 0, ""); err != nil {
		return err
	}

	// put_old must sit inside newroot
	putold := filepath.Join(newroot, ".put_old")
	if err := os.MkdirAll(putold, fileperms); err != nil {
		return err
	}

	// swap the root: newroot becomes /, old root parked at /.put_old
	if err := syscall.PivotRoot(newroot, putold); err != nil {
		return err
	}

	// cwd still points into the old root; move into the new one
	if err := syscall.Chdir("/"); err != nil {
		return err
	}

	// THE line that matters: detach the old root so the host tree is gone
	if err := syscall.Unmount("/.put_old", syscall.MNT_DETACH); err != nil {
		return err
	}
	return os.Remove("/.put_old")
}
