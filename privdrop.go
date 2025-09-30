//go:build !windows

package main

import (
	"fmt"
	"strconv"
	"syscall"
)

func dropPrivileges(uidStr, gidStr string) error {
	uid, err := strconv.Atoi(uidStr)
	if err != nil {
		return fmt.Errorf("invalid UID: %v", err)
	}
	gid, err := strconv.Atoi(gidStr)
	if err != nil {
		return fmt.Errorf("invalid GID: %v", err)
	}
	if err := syscall.Setgid(gid); err != nil {
		return fmt.Errorf("Setgid failed: %v", err)
	}
	if err := syscall.Setuid(uid); err != nil {
		return fmt.Errorf("Setuid failed: %v", err)
	}
	return nil
}
