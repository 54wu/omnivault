package vault

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/54wu/omnivault/internal/dpapi"
)

// envKeyPath returns the secret-key path from the OVAULT_KEY_PATH env var,
// normally set by GetTheKey.bat for the lifetime of that launch.
func envKeyPath() string {
	return strings.TrimSpace(os.Getenv("OVAULT_KEY_PATH"))
}

// SecretKeyPath resolves where the secret key lives for a vault rooted at dir,
// trying the same order everywhere the key is read or written:
//
//	1. $OVAULT_KEY_PATH (GetTheKey.bat)
//	2. the path remembered in the OS credential store (DPAPI)
//	3. the legacy in-vault <dir>/secret.key
//
// It never returns an empty string. The key stays OUTSIDE the vault folder when
// an external path is configured; otherwise it falls back to the legacy spot so
// existing vaults and plain double-click first runs keep working.
func SecretKeyPath(dir string) string {
	if p := envKeyPath(); p != "" {
		return p
	}
	if p, err := dpapi.Read(); err == nil && p != "" {
		return p
	}
	return filepath.Join(dir, "secret.key")
}