//go:build windows

package main

import (
	"os/exec"
	"syscall"
)

// detachServerChild runs the background server process without its own console
// window, so the parent's console can be fully closed once the UI opens.
func detachServerChild(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: 0x08000000, // CREATE_NO_WINDOW
	}
}
