// Package dpapi securely stores the secret-key path in the Windows Credential
// Manager (encrypted under the current user's account) so the key file can live
// outside the vault folder while its location is remembered across restarts.
//
// Only the path is stored — never the key bytes. The key file itself is read
// in place at unlock time and never copied into the vault directory.
package dpapi

import "errors"

// ErrNotFound reports that no key path has been stored yet.
var ErrNotFound = errors.New("dpapi: no stored key path")

// ErrNotSupported reports that DPAPI is unavailable on this platform.
var ErrNotSupported = errors.New("dpapi: only supported on Windows")