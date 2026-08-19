package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/54wu/omnivault/internal/store"
)

// cmdBackup copies ONLY the encrypted vault database to a sync destination
// (e.g. a cloud-synced folder), keeping the secret key on this device.
// This is the "cloud sync + key separation" model: the safe can be carried,
// the key never leaves you. Usage: omnivault backup <dest-file-or-dir>
func cmdBackup() {
	if len(os.Args) < 3 {
		fatal("usage: omnivault backup <dest>  (file path or a directory to sync vault.db into)")
	}
	dest := os.Args[2]

	dir := vaultDir()
	dbPath := filepath.Join(dir, "vault.db")
	if _, err := os.Stat(dbPath); err != nil {
		fatal("vault database not found at %s", dbPath)
	}

	// Checkpoint WAL so vault.db is self-contained before copying.
	db, err := store.Open(dbPath)
	if err != nil {
		fatal("open database for checkpoint: %v", err)
	}
	if err := db.Checkpoint(); err != nil {
		fatal("checkpoint: %v", err)
	}
	db.Close()

	// If dest is a directory, write vault.db inside it.
	if info, err := os.Stat(dest); err == nil && info.IsDir() {
		dest = filepath.Join(dest, "vault.db")
	}

	if err := copyFile(dbPath, dest); err != nil {
		fatal("backup: %v", err)
	}
	fmt.Printf("Backup written to %s (encrypted vault.db only; secret key NOT included)\n", dest)
	fmt.Println("Sync this file/folder to other devices. Your secret key stays here.")
}
