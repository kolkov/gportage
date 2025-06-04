package state

import (
    "os/exec"
)

func CreateSnapshot() (string, error) {
    snapshotID := generateUUID()
    cmd := exec.Command("btrfs", "subvolume", "snapshot", "/", "/.snapshots/"+snapshotID)
    return snapshotID, cmd.Run()
}

func RollbackSnapshot(snapshotID string) error {
    cmd := exec.Command("btrfs", "subvolume", "set-default", snapshotID, "/")
    return cmd.Run()
}