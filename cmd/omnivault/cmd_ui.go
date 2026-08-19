//go:build windows

package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"unsafe"

	"github.com/jchv/go-webview2"
	"github.com/54wu/omnivault/internal/api"
	"github.com/54wu/omnivault/internal/vault"
)

// cmdApp is the double-click entry point: it opens the vault (creating it on
// first run) and shows the UI in a single process. No background daemon is
// started, so there is no port or network involved that could be intercepted.
func cmdApp() {
	cmdUI()
}

// cmdUI opens the vault in-process, unlocks it, and shows the UI in a native
// WebView2 window. The UI talks to the vault through a Go bridge (WebView2
// Bind) instead of HTTP, so no TCP port or named pipe is involved — the vault
// can never be unreachable due to network-filter drivers.
//
// The password is entered inside the window itself (a login / first-run
// screen), so this is a single window with no console. On first run it walks
// the user through creating the vault.
func cmdUI() {
	dir := vaultDir()

	// Detach the console that Windows creates on double-click so only the
	// native window remains (the password is entered inside the window).
	freeConsole()

	// Ensure the directory exists so the vault can be opened/created.
	if err := os.MkdirAll(dir, 0700); err != nil {
		fatal("创建档案库目录失败: %v", err)
	}

	// Open and check whether the vault is actually initialized. An uninitialized
	// state means either the file is missing or a previous attempt left an empty
	// shell — both are safe to recreate (the user has no real data yet).
	v, err := vault.Open(dir)
	if err != nil {
		fatal("打开档案库失败: %v", err)
	}
	status, serr := v.Status()
	if serr != nil {
		v.Close()
		fatal("读取档案库状态失败: %v", serr)
	}
	firstRun := !status.Initialized

	// Build the in-process API server (never starts a listener) and bridge it
	// into the WebView2 window. The bridge starts with an empty token; the
	// in-window login flow sets it once the user unlocks the vault.
	srv := api.New(v, "127.0.0.1:0")
	bridge := api.NewNativeBridge(srv.Handler(), "")
	serverToggle := &serverBridge{srv: srv}

	// currentV tracks the live vault handle; it is swapped after a first-run
	// init reopens the database. Cleanup always targets the latest handle.
	currentV := v
	defer func() {
		currentV.StopBackup()
		currentV.Lock()
		currentV.Close()
	}()

	login := &loginBridge{
		v:   v,
		dir: dir,
		setToken: bridge.SetToken,
		onInitComplete: func(nv *vault.Vault) {
			currentV = nv
			srv.SetVault(nv)
		},
	}

	if runtime.GOOS != "windows" || !openNativeWindow(bridge, login, serverToggle, api.UISource(), firstRun, dir) {
		// Non-Windows or no WebView2 runtime: fall back to a plain browser is
		// not possible for the in-process bridge, so report the limitation.
		fatal("webview2 不可用，无法在非 Windows / 无 runtime 环境打开界面")
	}
}

// loginBridge is bound into the WebView2 window as `nativeVaultLogin`. It
// performs the in-window unlock (and first-run creation). Each method returns a
// JSON string: {"ok":true,"data":{...}} or {"ok":false,"error":"..."}.
type loginBridge struct {
	v              *vault.Vault
	dir            string
	setToken       func(token string)
	onInitComplete func(nv *vault.Vault)
}

// Handle dispatches a login request. op is "login" (unlock an existing vault)
// or "create" (initialize a fresh vault on first run). WebView2 Bind only
// accepts functions, so the two operations are exposed through one entry point.
func (h *loginBridge) Handle(op, password string) string {
	switch op {
	case "create":
		return h.create(password)
	case "login", "":
		return h.login(password)
	}
	return loginError(fmt.Errorf("未知操作: %s", op))
}

// login unlocks an already-initialized vault with the given password and hands
// the session token to the native bridge.
func (h *loginBridge) login(password string) string {
	sk, err := readSecretKey()
	if err != nil {
		return loginError(err)
	}
	token, err := h.v.Unlock(password, sk)
	if err != nil {
		return loginError(err)
	}
	h.v.BackupNow()
	if h.setToken != nil {
		h.setToken(token)
	}
	return loginOK(map[string]any{"token": token})
}

// create initializes a fresh vault (first run) with the given password, then
// unlocks it. Returns the session token and the secret key for display.
func (h *loginBridge) create(password string) string {
	if len(password) < 8 {
		return loginError(fmt.Errorf("密码至少需要 8 个字符"))
	}
	// Drop the stale uninitialized handle and its empty shell so Init can build
	// a fresh database.
	h.v.StopBackup()
	h.v.Close()
	for _, f := range []string{"vault.db", "vault.db-wal", "vault.db-shm"} {
		os.Remove(filepath.Join(h.dir, f))
	}
	sk, err := vault.Init(h.dir, password)
	if err != nil {
		return loginError(err)
	}
	nv, err := vault.Open(h.dir)
	if err != nil {
		return loginError(err)
	}
	token, err := nv.Unlock(password, sk)
	if err != nil {
		nv.Close()
		return loginError(err)
	}
	nv.BackupNow()
	if h.onInitComplete != nil {
		h.onInitComplete(nv)
	}
	if h.setToken != nil {
		h.setToken(token)
	}
	// Write the freshly generated key to its configured location (external path
	// via OVAULT_KEY_PATH / remembered, else the legacy in-vault spot).
	keyPath, err := saveSecretKey(sk)
	if err != nil {
		keyPath = ""
	}
	return loginOK(map[string]any{"token": token, "secret_key": sk, "key_path": keyPath})
}

func loginOK(data map[string]any) string {
	out, _ := json.Marshal(map[string]any{"ok": true, "data": data})
	return string(out)
}

func loginError(err error) string {
	out, _ := json.Marshal(map[string]any{"ok": false, "error": err.Error()})
	return string(out)
}

// serverBridge is bound into the WebView2 window as `nativeVaultServer`. It
// toggles the in-process loopback listener (127.0.0.1:7200) so external
// MCP/HTTP consumers can reach the vault while the UI is open. The listener
// lives only as long as the process — closing the window stops it (关窗即关).
type serverBridge struct {
	srv *api.Server
}

// Set toggles the loopback listener. op is "status" | "on" | "off". Returns a
// JSON string: {"ok":true,"data":{...}} or {"ok":false,"error":"..."}.
func (h *serverBridge) Set(op string) string {
	switch op {
	case "status":
		return serverOK(map[string]any{"running": h.srv.LocalRunning()})
	case "on":
		if err := h.srv.StartLocal("127.0.0.1:7200"); err != nil {
			return serverError(fmt.Errorf("开启本地服务失败（端口可能被占用）: %v", err))
		}
		return serverOK(map[string]any{"running": true})
	case "off":
		h.srv.StopLocal()
		return serverOK(map[string]any{"running": false})
	}
	return serverError(fmt.Errorf("未知操作: %s", op))
}

func serverOK(data map[string]any) string {
	out, _ := json.Marshal(map[string]any{"ok": true, "data": data})
	return string(out)
}

func serverError(err error) string {
	out, _ := json.Marshal(map[string]any{"ok": false, "error": err.Error()})
	return string(out)
}

// windowBridge is bound into the WebView2 window as `nativeVaultMaximize`. It
// lets the UI maximize the native window once the vault has been unlocked, so
// the dossier fills the screen. No-op in a plain browser / test harness where
// the bridge is absent or the window handle is nil.
type windowBridge struct {
	getHWND func() unsafe.Pointer
}

// Maximize maximizes the native window (best-effort).
func (h *windowBridge) Maximize() string {
	maximizeWindow(h.getHWND())
	out, _ := json.Marshal(map[string]any{"ok": true})
	return string(out)
}

// openNativeWindow shows the UI in a native WebView2 window, binding the vault
// bridge and the login bridge and loading the embedded HTML directly (no
// network). Blocks until the window closes. Returns false if the WebView2
// runtime is unavailable.
func openNativeWindow(bridge *api.NativeBridge, login *loginBridge, serverToggle *serverBridge, html []byte, firstRun bool, dir string) (ok bool) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Fprintf(os.Stderr, "WebView2 不可用（%v）。\n", r)
			ok = false
		}
	}()

	// Use a persistent data directory inside the vault folder so the UI's
	// localStorage (person list, field order, custom fields) survives restarts.
	// Without this, WebView2 uses a fresh temp profile each launch and the UI
	// forgets which persons exist even though the field data is safe in vault.db.
	dataPath := filepath.Join(dir, "ui-data")
	if err := os.MkdirAll(dataPath, 0700); err != nil {
		dataPath = ""
	}

	// Persist the UI page as a file inside the data directory and navigate to
	// it. NavigateToString (used by SetHtml) loads from an opaque about:blank
	// origin where localStorage throws and is silently swallowed — the person
	// list would reset to empty on every launch. A real file:// origin lets
	// localStorage survive restarts. The file is rewritten on every launch so
	// updates to the embedded HTML take effect automatically.
	pagePath := ""
	if dataPath != "" {
		pagePath = filepath.Join(dataPath, "onboarding.html")
		if err := os.WriteFile(pagePath, html, 0600); err != nil {
			pagePath = ""
		}
	}
	pageURL := ""
	if pagePath != "" {
		u := url.URL{Scheme: "file", Path: filepath.ToSlash(pagePath)}
		pageURL = u.String()
	}

	w := webview2.NewWithOptions(webview2.WebViewOptions{
		Debug:     false,
		AutoFocus: true,
		DataPath:  dataPath,
		WindowOptions: webview2.WindowOptions{
			Title:  "OmniVault · 万象档案袋",
			Width:  1200,
			Height: 800,
			Center: true,
		},
	})
	if w == nil {
		fmt.Fprintln(os.Stderr, "未检测到 WebView2 运行时。")
		return false
	}
	defer w.Destroy()

	var currentHWND unsafe.Pointer
	winBridge := &windowBridge{getHWND: func() unsafe.Pointer { return currentHWND }}

	if err := w.Bind("nativeVault", bridge.Request); err != nil {
		fmt.Fprintf(os.Stderr, "绑定界面桥接失败: %v\n", err)
		return false
	}
	if err := w.Bind("nativeVaultLogin", login.Handle); err != nil {
		fmt.Fprintf(os.Stderr, "绑定登录桥接失败: %v\n", err)
		return false
	}
	if err := w.Bind("nativeVaultServer", serverToggle.Set); err != nil {
		fmt.Fprintf(os.Stderr, "绑定服务开关桥接失败: %v\n", err)
		return false
	}
	if err := w.Bind("nativeVaultMaximize", winBridge.Maximize); err != nil {
		fmt.Fprintf(os.Stderr, "绑定窗口最大化桥接失败: %v\n", err)
		return false
	}

	// Inject the first-run flag on every page load so the HTML shows the create
	// form instead of the unlock form. The session token is *not* injected here;
	// it is established by the login screen and carried over a #token= reload.
	w.Init("window.__omnivaultFirstRun = " + boolJSON(firstRun) + ";")

	fmt.Println("已打开万象档案袋原生窗口。")
	if pageURL != "" {
		w.Navigate(pageURL)
	} else {
		// Fallback if the page could not be written to disk.
		w.SetHtml(string(html))
	}

	// Bring the window to the foreground so it appears on top of any other
	// windows (it is the only window — there is no console anymore).
	w.Dispatch(func() {
		currentHWND = w.Window()
		setForegroundWindow(w.Window())
	})

	w.Run()
	return true
}

func boolJSON(b bool) string {
	if b {
		return "true"
	}
	return "false"
}