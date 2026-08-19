//go:build !windows

package main

import (
	"fmt"
	"os"
)

// On non-Windows platforms there is no WebView2/native UI window. The
// double-click entry point (cmdApp) and the "ui" command fall back to guiding
// the user to the HTTP server / daemon instead, so the binaries still build
// for darwin and linux.

func cmdApp() {
	fmt.Fprintln(os.Stderr, "OmniVault 图形界面目前仅支持 Windows。")
	fmt.Fprintln(os.Stderr, "请改用 HTTP 服务方式：先运行 'omnivault unlock' 解锁，")
	fmt.Fprintln(os.Stderr, "然后在浏览器打开 http://127.0.0.1:7200/ui")
	os.Exit(1)
}

func cmdUI() {
	cmdApp()
}