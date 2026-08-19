//go:build windows

package dpapi

import (
	"os"
	"path/filepath"
	"syscall"
	"unsafe"
)

// The remembered value is the secret-key PATH, DPAPI-encrypted under the current
// user's profile and stored outside the vault folder. Storing an opaque blob via
// CryptProtectData is more robust than the picky CREDENTIALW marshaling.

// flags forbids any UI prompt during protect/unprotect.
const (
	flags = 0x1 // CRYPTPROTECT_UI_FORBIDDEN
)

var (
	modCrypt32    = syscall.NewLazyDLL("crypt32.dll")
	procProtect   = modCrypt32.NewProc("CryptProtectData")
	procUnprotect = modCrypt32.NewProc("CryptUnprotectData")
	modKernel32   = syscall.NewLazyDLL("kernel32.dll")
	procLocalFree = modKernel32.NewProc("LocalFree")
)

// dataBlob mirrors the native DATA_BLOB structure.
type dataBlob struct {
	cbData uint32
	pbData *byte
}

func blob(b []byte) *dataBlob {
	if len(b) == 0 {
		return &dataBlob{}
	}
	return &dataBlob{cbData: uint32(len(b)), pbData: &b[0]}
}

// dpapiFile returns the path of the DPAPI-encrypted key-path holder.
func dpapiFile() string {
	dir := os.Getenv("APPDATA")
	if dir == "" {
		home, _ := os.UserHomeDir()
		dir = filepath.Join(home, "AppData", "Roaming")
	}
	return filepath.Join(dir, "OmniVault", "keypath.dat")
}

// Store DPAPI-encrypts the key path and persists it to disk.
func Store(path string) error {
	in := blob([]byte(path))
	var out dataBlob
	r1, _, err := procProtect.Call(
		uintptr(unsafe.Pointer(in)), // pDataIn
		0,                           // szDataDescr = nil
		0,                           // pOptionalEntropy = nil
		0,                           // pvReserved = nil
		0,                           // pPromptStruct = nil
		uintptr(flags),              // dwFlags
		uintptr(unsafe.Pointer(&out)),
	)
	if r1 == 0 {
		return err
	}
	defer procLocalFree.Call(uintptr(unsafe.Pointer(out.pbData)))

	enc := unsafe.Slice(out.pbData, int(out.cbData))
	if err := os.MkdirAll(filepath.Dir(dpapiFile()), 0700); err != nil {
		return err
	}
	return os.WriteFile(dpapiFile(), enc, 0600)
}

// Read DPAPI-decrypts the stored key path, or returns ErrNotFound.
func Read() (string, error) {
	data, err := os.ReadFile(dpapiFile())
	if err != nil {
		if os.IsNotExist(err) {
			return "", ErrNotFound
		}
		return "", err
	}
	if len(data) == 0 {
		return "", ErrNotFound
	}

	in := blob(data)
	var out dataBlob
	r1, _, err := procUnprotect.Call(
		uintptr(unsafe.Pointer(in)), // pDataIn
		0,
		0, // pOptionalEntropy
		0, // pvReserved
		0, // pPromptStruct
		uintptr(flags),
		uintptr(unsafe.Pointer(&out)),
	)
	if r1 == 0 {
		return "", err
	}
	defer procLocalFree.Call(uintptr(unsafe.Pointer(out.pbData)))

	return string(unsafe.Slice(out.pbData, int(out.cbData))), nil
}

// Delete removes the stored key path, if present.
func Delete() error {
	err := os.Remove(dpapiFile())
	if err != nil {
		if os.IsNotExist(err) {
			return ErrNotFound
		}
		return err
	}
	return nil
}