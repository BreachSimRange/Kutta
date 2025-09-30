//go:build windows

package main

func dropPrivileges(user, group string) error {
    // No-op on Windows
    return nil
}
