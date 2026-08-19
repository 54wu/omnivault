//go:build !windows

package main

import "os/exec"

// detachServerChild is a no-op on non-Windows platforms.
func detachServerChild(cmd *exec.Cmd) {}
