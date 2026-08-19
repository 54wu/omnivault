package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"

	"github.com/54wu/omnivault/internal/dpapi"
	"github.com/54wu/omnivault/internal/vault"
	"golang.org/x/term"
)

func vaultDir() string {
	if d := os.Getenv("VAULT_DIR"); d != "" {
		return d
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".omnivault")
}

func serverAddr() string {
	if a := os.Getenv("VAULT_ADDR"); a != "" {
		return a
	}
	return "http://127.0.0.1:7200"
}

func sessionPath() string {
	return filepath.Join(vaultDir(), ".session")
}

func pidPath() string {
	return filepath.Join(vaultDir(), "omnivault.pid")
}

func secretKeyPath() string {
	return filepath.Join(vaultDir(), "secret.key")
}

func readSessionToken() (string, error) {
	data, err := os.ReadFile(sessionPath())
	if err != nil {
		return "", fmt.Errorf("vault is not unlocked (no session file)")
	}
	return strings.TrimSpace(string(data)), nil
}

func writeSessionToken(token string) error {
	return os.WriteFile(sessionPath(), []byte(token+"\n"), 0600)
}

func removeSessionToken() {
	os.Remove(sessionPath())
}

func readSecretKey() (string, error) {
	p, err := resolveKeyPath()
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(p)
	if err != nil {
		return "", fmt.Errorf("读取密钥文件失败 (%s): %v", p, err)
	}
	return strings.TrimSpace(string(data)), nil
}

// keyPathFromArgs returns the key path passed via --key-path / --key, if any.
func keyPathFromArgs() string {
	for i := 1; i+1 < len(os.Args); i++ {
		if os.Args[i] == "--key-path" || os.Args[i] == "--key" {
			return os.Args[i+1]
		}
	}
	return ""
}

// resolveKeyPath returns the secret-key file path, trying in order:
//
//	1. --key-path / --key <path> command-line argument
//	2. $OVAULT_KEY_PATH environment variable (set by GetTheKey.bat)
//	3. the path remembered in the OS credential store (DPAPI on Windows)
//	4. the legacy vaultDir()/secret.key (existing vaults keep working)
//
// The key file lives OUTSIDE the vault folder; only its path is ever persisted.
func resolveKeyPath() (string, error) {
	if p := keyPathFromArgs(); p != "" {
		return p, nil
	}
	if p := envKeyPath(); p != "" {
		return p, nil
	}
	if p, err := dpapi.Read(); err == nil && p != "" {
		return p, nil
	}
	legacy := secretKeyPath()
	if _, err := os.Stat(legacy); err == nil {
		return legacy, nil
	}
	return "", fmt.Errorf("找不到密钥文件：请运行 GetTheKey.bat，或用 --key <路径> 指定 secret.key 的位置")
}

func envKeyPath() string {
	return strings.TrimSpace(os.Getenv("OVAULT_KEY_PATH"))
}

// rememberKeyPath stores the key path in the OS credential store (DPAPI).
func rememberKeyPath(p string) error {
	return dpapi.Store(p)
}

// forgetKeyPath removes the remembered key path.
func forgetKeyPath() error {
	return dpapi.Delete()
}

// writeSecretKey writes the secret key to the given absolute path (0600),
// creating parent directories as needed. The vault folder is never the target.
func writeSecretKey(path, sk string) error {
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("密钥路径为空")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return fmt.Errorf("创建密钥目录失败: %w", err)
	}
	return os.WriteFile(path, []byte(strings.TrimSpace(sk)+"\n"), 0600)
}

// saveSecretKey writes a freshly-generated secret key to the external path when
// one is configured (GetTheKey.bat's OVAULT_KEY_PATH or a remembered path);
// otherwise it falls back to the legacy in-vault location so a plain double-click
// first run still works. Returns the path written.
func saveSecretKey(sk string) (string, error) {
	path := envKeyPath()
	external := path != ""
	if path == "" {
		if p, err := dpapi.Read(); err == nil && p != "" {
			path = p
			external = true
		}
	}
	if path == "" {
		path = secretKeyPath() // legacy in-vault location
	}
	if err := writeSecretKey(path, sk); err != nil {
		return "", err
	}
	if external {
		rememberKeyPath(path) // best-effort
	}
	return path, nil
}

func writePID(pid int) error {
	return os.WriteFile(pidPath(), []byte(strconv.Itoa(pid)+"\n"), 0600)
}

func readPID() (int, error) {
	data, err := os.ReadFile(pidPath())
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(strings.TrimSpace(string(data)))
}

func removePID() {
	os.Remove(pidPath())
}

func promptPassword(prompt string) (string, error) {
	if pw := os.Getenv("VAULT_PASSWORD"); pw != "" {
		return pw, nil
	}
	fmt.Fprint(os.Stderr, prompt)
	pw, err := term.ReadPassword(int(syscall.Stdin))
	fmt.Fprintln(os.Stderr)
	if err != nil {
		return "", err
	}
	return string(pw), nil
}

// apiRequest makes an authenticated HTTP request to the vault server.
func apiRequest(method, path string, body any) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		buf, _ := json.Marshal(body)
		bodyReader = bytes.NewReader(buf)
	}

	url := serverAddr() + path
	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	token, err := readSessionToken()
	if err == nil {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	return http.DefaultClient.Do(req)
}

// apiResult decodes a JSON response or returns the error.
func apiResult(resp *http.Response, target any) error {
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		var errResp struct {
			Error string `json:"error"`
		}
		json.NewDecoder(resp.Body).Decode(&errResp)
		if errResp.Error != "" {
			return fmt.Errorf("%s", errResp.Error)
		}
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	if target != nil {
		return json.NewDecoder(resp.Body).Decode(target)
	}
	return nil
}

func fatal(msg string, args ...any) {
	full := fmt.Sprintf("错误: "+msg+"\n", args...)
	if len(os.Args) < 2 && runtime.GOOS == "windows" {
		// Double-clicked as a GUI app (no console): surface the error in a
		// message box instead of letting it vanish with the app.
		showFatalDialog(full)
	} else {
		fmt.Fprint(os.Stderr, full)
	}
	os.Exit(1)
}

// getPassword returns the profile password from the VAULT_PASSWORD env var or
// an interactive prompt. In the single-process model every command unlocks the
// vault directly, so the password is read here.
func getPassword() (string, error) {
	if pw := os.Getenv("VAULT_PASSWORD"); pw != "" {
		return pw, nil
	}
	return promptPassword("档案密码: ")
}

// openUnlocked opens the vault and unlocks it with prompted credentials.
// Returns the unlocked vault, its session token, and a cleanup func that
// locks and closes the vault. Suitable for the single-process model where each
// command operates on the vault directly (no background HTTP daemon).
func openUnlocked() (*vault.Vault, string, func(), error) {
	dir := vaultDir()
	v, err := vault.Open(dir)
	if err != nil {
		return nil, "", nil, err
	}
	pw, err := getPassword()
	if err != nil {
		v.Close()
		return nil, "", nil, err
	}
	sk, err := readSecretKey()
	if err != nil {
		v.Close()
		return nil, "", nil, err
	}
	token, err := v.Unlock(pw, sk)
	if err != nil {
		v.Close()
		return nil, "", nil, err
	}
	// Snapshot on unlock so the vault starts from a known-good state.
	v.BackupNow()
	cleanup := func() {
		v.StopBackup()
		v.Lock()
		v.Close()
	}
	return v, token, cleanup, nil
}

// runVault opens+unlocks the vault, runs fn against it, then locks and closes.
// fn runs while the vault is unlocked; any error is fatal.
func runVault(fn func(*vault.Vault) error) {
	v, _, cleanup, err := openUnlocked()
	if err != nil {
		fatal("%v", err)
	}
	defer cleanup()
	if err := fn(v); err != nil {
		fatal("%v", err)
	}
}

// copyFile copies a single file, preserving the source's permission bits.
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}
