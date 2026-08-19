//go:build !windows

package dpapi

// Store is unsupported off Windows.
func Store(path string) error { return ErrNotSupported }

// Read is unsupported off Windows.
func Read() (string, error) { return "", ErrNotSupported }

// Delete is unsupported off Windows.
func Delete() error { return ErrNotSupported }