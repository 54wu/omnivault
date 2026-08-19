//go:build !windows

package main

import "unsafe"

// freeConsole is a no-op on non-Windows platforms.
func freeConsole() {}

// setForegroundWindow is a no-op on non-Windows platforms (the app only opens
// a native window on Windows).
func setForegroundWindow(hwnd unsafe.Pointer) {}

// maximizeWindow is a no-op on non-Windows platforms.
func maximizeWindow(hwnd unsafe.Pointer) {}

// showFatalDialog is a no-op on non-Windows platforms.
func showFatalDialog(msg string) {}