#!/usr/bin/env bash
#
# cleanup  --  remove host state a demo run can leave behind.
#
# The runtime cleans up after itself on a clean exit, but a hard kill (Ctrl-C
# at the wrong moment, a must() panic before the defers finish) can strand the
# veth pair, the cgroup, the grown probe log, or a half-finished mount. Run
# this between takes; it is safe to run when there is nothing to clean.
#
# Needs root for the veth and cgroup (same as the runtime under `make sudo`),
# so it uses sudo when not already root.
set -u

C_GRN=$'\033[32m'
C_DIM=$'\033[2m'
C_RST=$'\033[0m'

# Keep these in step with the consts in cmd/demo/main.go.
ROOTFS=${ROOTFS:-_rootfs}
VETH=veth1
CGROUP_CTR="/sys/fs/cgroup/user.slice/user-$(id -u).slice/boxes.service/ctr1"
CGROUP_SVC="/sys/fs/cgroup/user.slice/user-$(id -u).slice/boxes.service"
LOG="${ROOTFS}/var/log/node.log"

SUDO=
[ "$(id -u)" -ne 0 ] && SUDO=sudo

done_msg() { printf '%s  %s%s\n' "$C_GRN" "$1" "$C_RST"; }
skip_msg() { printf '%s  %s%s\n' "$C_DIM" "$1" "$C_RST"; }

# --- host end of the veth pair (deleting it takes the container end too) ---
if ip link show "$VETH" >/dev/null 2>&1; then
    $SUDO ip link del "$VETH" && done_msg "deleted $VETH"
else
    skip_msg "no $VETH"
fi

# --- cgroup: child first, then the service dir (rmdir fails on a parent that
#     still has children) ---
for cg in "$CGROUP_CTR" "$CGROUP_SVC"; do
    if [ -d "$cg" ]; then
        $SUDO rmdir "$cg" && done_msg "removed ${cg##*/cgroup/}"
    else
        skip_msg "no ${cg##*/cgroup/}"
    fi
done

# --- unmount anything left under the rootfs in the HOST ns. Stage 00 has no
#     mount namespace, so the runtime's proc/sysfs/cgroup2/dev mounts DO
#     survive it and land here every run. Deepest first: /sys/fs/cgroup is
#     stacked on /sys, so /sys will not release until it goes. ---
for m in "${ROOTFS}/sys/fs/cgroup" "${ROOTFS}/sys" "${ROOTFS}/dev" "${ROOTFS}/proc" "${ROOTFS}"; do
    if mountpoint -q "$m" 2>/dev/null; then
        $SUDO umount -l "$m" && done_msg "unmounted $m"
    fi
done

# --- leftover pivot_root staging dir ---
if [ -d "${ROOTFS}/.put_old" ]; then
    $SUDO rmdir "${ROOTFS}/.put_old" 2>/dev/null && done_msg "removed ${ROOTFS}/.put_old"
fi

# --- probe log: reset so the webapp starts each take from a clean slate ---
if [ -f "$LOG" ]; then
    : >"$LOG" 2>/dev/null || $SUDO truncate -s 0 "$LOG"
    done_msg "cleared $LOG"
else
    skip_msg "no probe log"
fi

echo "clean"
