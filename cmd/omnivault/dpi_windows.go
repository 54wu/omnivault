//go:build windows

package main

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

// init opts the process into per-monitor V2 DPI awareness before any window is
// created. Without this, Windows bitmaps-scales the whole WebView window on
// high-DPI displays and the UI text looks blurry.
func init() {
	user32 := windows.NewLazySystemDLL("user32.dll")

	setDpiAwarenessContext := user32.NewProc("SetProcessDpiAwarenessContext")
	if err := setDpiAwarenessContext.Find(); err == nil {
		// DPI_AWARENESS_CONTEXT_PER_MONITOR_AWARE_V2 == (HANDLE)(-4)
		setDpiAwarenessContext.Call(uintptr(^uintptr(3)))
	} else if setProcDpiAware := user32.NewProc("SetProcessDPIAware"); setProcDpiAware.Find() == nil {
		// Fallback for older Windows: system DPI awareness.
		setProcDpiAware.Call()
	}

	// If we own the console alone (double-clicked, not launched from a shell),
	// detach immediately so the console window closes before it can flash.
	// CLI commands are launched from a terminal, where the console is shared
	// with the parent, so they keep their console.
	if ownsConsoleAlone() {
		freeConsole()
	}
}

// ownsConsoleAlone reports whether this process is the only process attached to
// its console. That is true when the exe was double-clicked (Windows created a
// fresh console just for it) and false when it was launched from a shell.
func ownsConsoleAlone() bool {
	kernel32 := windows.NewLazySystemDLL("kernel32.dll")
	proc := kernel32.NewProc("GetConsoleProcessList")
	if proc.Find() != nil {
		return false
	}
	var list [2]uint32
	n, _, _ := proc.Call(uintptr(unsafe.Pointer(&list[0])), uintptr(len(list)))
	return n == 1
}

// freeConsole detaches the process from its console window (kernel32
// FreeConsole). Used at the start of the app flow so the console that Windows
// creates on double-click closes immediately, leaving only the native window.
func freeConsole() {
	kernel32 := windows.NewLazySystemDLL("kernel32.dll")
	procFreeConsole := kernel32.NewProc("FreeConsole")
	if procFreeConsole.Find() == nil {
		procFreeConsole.Call()
	}
}

// setForegroundWindow brings the given native window (HWND) to the top of the
// z-order and gives it input focus. The app is a single window (no console), so
// this ensures it appears on top of any other windows and is ready to receive
// the password.
func setForegroundWindow(hwnd unsafe.Pointer) {
	if hwnd == nil {
		return
	}
	user32 := windows.NewLazySystemDLL("user32.dll")
	procSetForeground := user32.NewProc("SetForegroundWindow")
	if procSetForeground.Find() == nil {
		procSetForeground.Call(uintptr(hwnd))
	}
	procBringToTop := user32.NewProc("BringWindowToTop")
	if procBringToTop.Find() == nil {
		procBringToTop.Call(uintptr(hwnd))
	}
}

// maximizeWindow maximizes the given native window (HWND) via ShowWindow with
// SW_MAXIMIZE. Called by the UI bridge once the vault has been unlocked so the
// dossier fills the screen.
func maximizeWindow(hwnd unsafe.Pointer) {
	if hwnd == nil {
		return
	}
	user32 := windows.NewLazySystemDLL("user32.dll")
	procShow := user32.NewProc("ShowWindow")
	if procShow.Find() == nil {
		// SW_MAXIMIZE == 3
		procShow.Call(uintptr(hwnd), uintptr(3))
	}
}

// showFatalDialog displays an error message box. Used by fatal() in app (GUI)
// mode, where there is no console to print an error to.
func showFatalDialog(msg string) {
	user32 := windows.NewLazySystemDLL("user32.dll")
	procMessageBox := user32.NewProc("MessageBoxW")
	title, _ := windows.UTF16PtrFromString("OmniVault · 万象档案袋")
	text, _ := windows.UTF16PtrFromString(msg)
	// MB_OK | MB_ICONERROR
	procMessageBox.Call(0, uintptr(unsafe.Pointer(text)), uintptr(unsafe.Pointer(title)), 0x10)
}