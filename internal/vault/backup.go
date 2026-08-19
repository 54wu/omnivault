package vault

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	// defaultBackupKeep is how many versioned backups are retained before the
	// oldest are pruned.
	defaultBackupKeep = 3
	// backupDebounce is how long a scheduled backup waits after the last write
	// before actually running, coalescing bursts of writes into one snapshot.
	backupDebounce = 3 * time.Second
	// backupPrefix is the filename prefix for versioned snapshot files.
	backupPrefix = "vault-"
	// backupSuffix is the file extension for snapshot files.
	backupSuffix = ".db"
)

// BackupInfo describes a single versioned backup snapshot.
type BackupInfo struct {
	Name    string    `json:"name"`
	Path    string    `json:"-"`
	Created time.Time `json:"created_at"`
	Size    int64     `json:"size"`
	ModTime time.Time `json:"-"`
}

// BackupManager writes versioned, encrypted snapshots of vault.db into a
// backups directory and prunes old ones. The secret key never enters the
// backups, preserving the "key stays on device" model.
type BackupManager struct {
	backupDir  string
	dbPath     string
	keep       int
	delay      time.Duration
	checkpoint func() error
	copyFn     func(src, dst string) error

	mu    sync.Mutex
	timer *time.Timer
}

// NewBackupManager creates a BackupManager rooted at vaultDir. check runFn is
// invoked to safely coalesce the WAL before each snapshot; keep is the maximum
// number of retained versions (<=0 falls back to the default).
func NewBackupManager(vaultDir string, keep int, checkpoint func() error) *BackupManager {
	if keep <= 0 {
		keep = defaultBackupKeep
	}
	return &BackupManager{
		backupDir:  filepath.Join(vaultDir, "backups"),
		dbPath:     filepath.Join(vaultDir, "vault.db"),
		keep:       keep,
		delay:      backupDebounce,
		checkpoint: checkpoint,
		copyFn:     copyFile,
	}
}

// Snapshot performs an immediate versioned backup. It is safe to call
// concurrently; a lock guards against overlapping snapshots.
func (b *BackupManager) Snapshot() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if _, err := os.Stat(b.dbPath); err != nil {
		return fmt.Errorf("vault database not found: %w", err)
	}
	if err := os.MkdirAll(b.backupDir, 0700); err != nil {
		return fmt.Errorf("create backups dir: %w", err)
	}
	if b.checkpoint != nil {
		if err := b.checkpoint(); err != nil {
			return fmt.Errorf("checkpoint before backup: %w", err)
		}
	}

	name := backupPrefix + time.Now().Format("20060102-150405") + backupSuffix
	dst := filepath.Join(b.backupDir, name)
	// Several snapshots may land in the same second; disambiguate with a
	// numeric suffix so they do not overwrite each other.
	for i := 1; ; i++ {
		if _, err := os.Stat(dst); os.IsNotExist(err) {
			break
		}
		name = backupPrefix + time.Now().Format("20060102-150405") + "-" + itoa(i) + backupSuffix
		dst = filepath.Join(b.backupDir, name)
	}
	if err := b.copyFn(b.dbPath, dst); err != nil {
		return fmt.Errorf("write backup: %w", err)
	}
	return b.prune()
}

// Schedule coalesces repeated calls into a single Snapshot executed after a
// short debounce window. Only the pending snapshot count matters, so bursts of
// writes produce one version.
func (b *BackupManager) Schedule() {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.timer != nil {
		b.timer.Stop()
	}
	b.timer = time.AfterFunc(b.delay, func() {
		if err := b.Snapshot(); err != nil {
			// Best-effort: a failed snapshot must not break the caller.
			fmt.Fprintf(os.Stderr, "backup: %v\n", err)
		}
	})
}

// Stop cancels any pending scheduled backup.
func (b *BackupManager) Stop() {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.timer != nil {
		b.timer.Stop()
		b.timer = nil
	}
}

// List returns all retained backup snapshots, newest first.
func (b *BackupManager) List() ([]BackupInfo, error) {
	entries, err := os.ReadDir(b.backupDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var backups []BackupInfo
	for _, e := range entries {
		if e.IsDir() || !strings.HasPrefix(e.Name(), backupPrefix) || !strings.HasSuffix(e.Name(), backupSuffix) {
			continue
		}
		ts, err := parseBackupTime(e.Name())
		if err != nil {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		backups = append(backups, BackupInfo{
			Name:    e.Name(),
			Path:    filepath.Join(b.backupDir, e.Name()),
			Created: ts,
			Size:    info.Size(),
			ModTime: info.ModTime(),
		})
	}

	// Newest first. ModTime disambiguates same-second snapshots reliably
	// (the numeric suffix is only a uniqueness aid, not an ordering key).
	sort.Slice(backups, func(i, j int) bool {
		return backups[i].ModTime.After(backups[j].ModTime)
	})
	return backups, nil
}

// Resolve returns the on-disk path for a version name or timestamp, or an
// error if no matching backup exists.
func (b *BackupManager) Resolve(name string) (string, error) {
	backups, err := b.List()
	if err != nil {
		return "", err
	}
	for _, bk := range backups {
		if bk.Name == name {
			return bk.Path, nil
		}
	}
	// Allow matching by raw timestamp (e.g. 20260814-153000).
	ts, err := parseBackupTime(backupPrefix + name + backupSuffix)
	if err == nil {
		for _, bk := range backups {
			if bk.Created.Equal(ts) {
				return bk.Path, nil
			}
		}
	}
	return "", fmt.Errorf("no backup matches %q", name)
}

// RollbackFile copies the selected snapshot over the live database. The caller
// is responsible for ensuring the server is stopped first.
func (b *BackupManager) RollbackFile(src string) error {
	if err := b.copyFn(src, b.dbPath); err != nil {
		return fmt.Errorf("restore backup: %w", err)
	}
	return nil
}

// prune removes the oldest snapshots beyond the retention limit.
func (b *BackupManager) prune() error {
	backups, err := b.List()
	if err != nil {
		return err
	}
	if len(backups) <= b.keep {
		return nil
	}
	for _, bk := range backups[b.keep:] {
		if err := os.Remove(bk.Path); err != nil {
			return err
		}
	}
	return nil
}

// parseBackupTime extracts the timestamp embedded in a snapshot filename,
// ignoring any same-second numeric disambiguation suffix (e.g. -2).
func parseBackupTime(name string) (time.Time, error) {
	trimmed := strings.TrimSuffix(strings.TrimPrefix(name, backupPrefix), backupSuffix)
	if t, err := time.ParseInLocation("20060102-150405", trimmed, time.Local); err == nil {
		return t, nil
	}
	// The timestamp contains hyphens, so only strip a trailing "-N" suffix
	// when the remainder still parses as a timestamp.
	if idx := strings.LastIndex(trimmed, "-"); idx > 0 {
		base := trimmed[:idx]
		if _, err := strconv.Atoi(trimmed[idx+1:]); err == nil {
			if t, err := time.ParseInLocation("20060102-150405", base, time.Local); err == nil {
				return t, nil
			}
		}
	}
	return time.Time{}, fmt.Errorf("invalid backup name %q", name)
}

// itoa converts a positive integer to its decimal string form.
func itoa(n int) string {
	return strconv.Itoa(n)
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