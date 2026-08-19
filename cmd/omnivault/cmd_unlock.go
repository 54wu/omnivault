package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"time"
)

func cmdUnlock() {
	// Probe the port first — catches stale servers even if the PID file is gone.
	if portHasVault() {
		if isVaultUnlocked() {
			fmt.Println("万象档案袋已解锁（服务正在运行）。")
			return
		}
		// Server running but vault auto-locked — re-unlock via API
		pw, err := promptPassword("档案密码: ")
		if err != nil {
			fatal("读取密码失败: %v", err)
		}
		sk, err := readSecretKey()
		if err != nil {
			fatal("%v", err)
		}
		reUnlock(pw, sk)
		return
	}

	// Nothing on the port — clean up any stale PID file
	removePID()

	pw, err := promptPassword("档案密码: ")
	if err != nil {
		fatal("读取密码失败: %v", err)
	}

	sk, err := readSecretKey()
	if err != nil {
		fatal("%v", err)
	}

	// Start background server
	exe, err := os.Executable()
	if err != nil {
		fatal("finding executable: %v", err)
	}

	cmd := exec.Command(exe, "serve", "--password-stdin")
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("VAULT_DIR=%s", vaultDir()),
	)
	detachServerChild(cmd)

	// Pass credentials via stdin pipe
	stdin, err := cmd.StdinPipe()
	if err != nil {
		fatal("creating stdin pipe: %v", err)
	}

	cmd.Stdout = nil
	cmd.Stderr = nil

	if err := cmd.Start(); err != nil {
		fatal("starting server: %v", err)
	}

	// Write credentials and close pipe
	fmt.Fprintf(stdin, "%s\n%s\n", pw, sk)
	stdin.Close()

	writePID(cmd.Process.Pid)

	// Watch the child process so a failed startup is reported immediately
	// instead of waiting out the full timeout.
	childDone := make(chan error, 1)
	go func() { childDone <- cmd.Wait() }()

	// Poll until the server is ready AND unlocked. The prior 3s timeout was too
	// short: the child must unlock (KDF), snapshot, and bind the port first, and
	// on a fresh boot the antivirus scanning the new exe adds further delay. We
	// now wait generously and only give up if the child exits or the timeout
	// elapses — never returning in a half-started state.
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	for {
		select {
		case err := <-childDone:
			// Child quit before becoming ready — surface the real failure.
			fatal("服务进程启动失败: %v", err)
		case <-ctx.Done():
			fatal("服务启动超时，未能确认解锁状态。可能原因：端口被占用或杀毒软件拦截，请关闭后重试。")
		case <-time.After(200 * time.Millisecond):
			resp, err := apiRequest("GET", "/vault/status", nil)
			if err != nil {
				continue // not ready yet
			}
			var status struct {
				Locked bool `json:"locked"`
			}
			json.NewDecoder(resp.Body).Decode(&status)
			resp.Body.Close()
			if !status.Locked {
				fmt.Println("万象档案袋已解锁，服务运行于", serverAddr())
				return
			}
			// Server responded but vault is locked — our spawn likely failed
			// and a stale server answered.
			fatal("%s 上的服务不是本次启动的（万象档案袋仍锁定）— 请先运行 'omnivault lock' 再 'omnivault unlock'", serverAddr())
		}
	}
}

// portHasVault probes the server address with GET /vault/status.
// Returns true if a vault server responds, false otherwise.
func portHasVault() bool {
	client := &http.Client{Timeout: time.Second}
	resp, err := client.Get(serverAddr() + "/vault/status")
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

func isVaultUnlocked() bool {
	resp, err := apiRequest("GET", "/vault/status", nil)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	var status struct {
		Locked bool `json:"locked"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return false
	}
	return !status.Locked
}

func reUnlock(password, secretKey string) {
	var answer string
	for attempt := 0; attempt < 4; attempt++ {
		body := map[string]string{
			"password":   password,
			"secret_key": secretKey,
		}
		if answer != "" {
			body["answer"] = answer
		}
		resp, err := apiRequest("POST", "/vault/unlock", body)
		if err != nil {
			fatal("重新解锁请求失败: %v", err)
		}

		var result struct {
			Token      string `json:"token"`
			Error      string `json:"error"`
			Constraint string `json:"constraint"`
			Question   string `json:"question"`
		}
		json.NewDecoder(resp.Body).Decode(&result)
		resp.Body.Close()

		// Vault demands the security answer after repeated wrong passwords.
		if resp.StatusCode == http.StatusLocked && result.Constraint == "security_required" {
			fmt.Printf("密保问题: %s\n", result.Question)
			ans, err := promptPassword("密保答案: ")
			if err != nil {
				fatal("读取答案失败: %v", err)
			}
			answer = ans
			continue
		}
		if resp.StatusCode == http.StatusLocked {
			fatal("%s", result.Error)
		}
		if resp.StatusCode >= 400 {
			fatal("解锁失败: %s", result.Error)
		}
		if result.Token == "" {
			fatal("解锁未返回令牌")
		}

		if err := writeSessionToken(result.Token); err != nil {
			fatal("写入会话失败: %v", err)
		}
		fmt.Println("万象档案袋已解锁，服务运行于", serverAddr())
		return
	}
	fatal("解锁失败：尝试次数过多")
}
