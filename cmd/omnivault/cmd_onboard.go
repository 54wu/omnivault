package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/54wu/omnivault/internal/vault"
)

var onboardFields = []struct {
	prompt  string
	fieldID string
}{
	{"Full name", "identity.full_name"},
	{"Email", "identity.email"},
	{"City", "addresses.home_city"},
	{"State", "addresses.home_state"},
	{"ZIP", "addresses.home_zip"},
	{"Country", "addresses.home_country"},
	{"Timezone", "preferences.timezone"},
}

func cmdOnboard() {
	dir := vaultDir()

	// Check if vault already exists
	if _, err := os.Stat(dir + "/vault.db"); err == nil {
		fatal("vault already initialized at %s — use 'omnivault ui' instead", dir)
	}

	fmt.Println("Create your vault")

	pw, err := promptPassword("  Profile password: ")
	if err != nil {
		fatal("reading password: %v", err)
	}
	if len(pw) < 8 {
		fatal("password must be at least 8 characters")
	}

	confirm, err := promptPassword("  Confirm: ")
	if err != nil {
		fatal("reading confirmation: %v", err)
	}
	if pw != confirm {
		fatal("passwords do not match")
	}

	sk, err := vault.Init(dir, pw)
	if err != nil {
		fatal("%v", err)
	}

	keyPath, err := saveSecretKey(sk)
	if err != nil {
		fatal("保存密钥失败: %v", err)
	}

	fmt.Println()
	fmt.Println("Your secret key (saved to " + keyPath + "):")
	fmt.Printf("  %s\n", sk)
	fmt.Println()

	// Open and unlock the vault in-process (no background daemon).
	v, err := vault.Open(dir)
	if err != nil {
		fatal("opening vault: %v", err)
	}
	if _, err := v.Unlock(pw, sk); err != nil {
		v.Close()
		fatal("unlocking vault: %v", err)
	}
	defer func() {
		v.Lock()
		v.Close()
	}()
	v.BackupNow()

	// Prompt for common fields
	fmt.Println("Let's add some basics (press Enter to skip any):")
	reader := bufio.NewReader(os.Stdin)
	saved := 0

	for _, f := range onboardFields {
		fmt.Printf("  %s: ", f.prompt)
		line, _ := reader.ReadString('\n')
		value := strings.TrimSpace(line)
		if value == "" {
			continue
		}

		if err := v.Set(f.fieldID, value, vault.DefaultSensitivity(f.fieldID)); err != nil {
			fmt.Fprintf(os.Stderr, "  Warning: could not save %s: %v\n", f.fieldID, err)
			continue
		}
		saved++
	}
	v.ScheduleBackup()

	fmt.Println()
	if saved > 0 {
		fmt.Printf("Done — %d field(s) saved. Your vault is ready.\n", saved)
	} else {
		fmt.Println("Done — your vault is ready.")
	}
	fmt.Println("Run 'omnivault ui' to open the interface, or 'omnivault status' to see what's stored.")
}