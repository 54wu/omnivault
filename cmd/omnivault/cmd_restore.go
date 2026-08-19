package main

import (
	"fmt"
	"os"
	"path/filepath"
)

// cmdRestore copies a synced vault.db (from a cloud folder or file) into the
// vault directory, replacing the local encrypted database. The secret key is
// never touched — on the new device you enter it once at unlock.
// Usage: omnivault restore <src-file-or-dir>
func cmdRestore() {
	if len(os.Args) < 3 {
		fatal("usage: omnivault restore <src>  (file path or a directory containing vault.db)")
	}
	src := os.Args[2]

	if portHasVault() {
		fatal("vault server is running — run 'omnivault lock' before restoring")
	}

	// If src is a directory, look for vault.db inside it.
	if info, err := os.Stat(src); err == nil && info.IsDir() {
		src = filepath.Join(src, "vault.db")
	}
	if _, err := os.Stat(src); err != nil {
		fatal("no vault.db found at %s", src)
	}

	dir := vaultDir()
	if err := os.MkdirAll(dir, 0700); err != nil {
		fatal("create vault dir: %v", err)
	}

	dbPath := filepath.Join(dir, "vault.db")
	if err := copyFile(src, dbPath); err != nil {
		fatal("restore: %v", err)
	}
	fmt.Printf("Vault database restored from %s (secret key on this device is kept)\n", src)
	fmt.Println("Run 'omnivault unlock' and enter your profile password + secret key.")
}
