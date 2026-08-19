package vault

import (
	"os"
	"path/filepath"
	"testing"
)

// TestBackupManagerVersioning verifies snapshots accumulate and pruning keeps
// only the most recent `keep` versions.
func TestBackupManagerVersioning(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "vault.db")
	if err := os.WriteFile(dbPath, []byte("seed"), 0600); err != nil {
		t.Fatal(err)
	}

	bm := NewBackupManager(dir, 3, nil)

	// Write distinct content and snapshot several times.
	for i := 0; i < 5; i++ {
		if err := os.WriteFile(dbPath, []byte("data-"+string(rune('a'+i))), 0600); err != nil {
			t.Fatal(err)
		}
		if err := bm.Snapshot(); err != nil {
			t.Fatalf("snapshot %d: %v", i, err)
		}
	}

	backups, err := bm.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(backups) != 3 {
		t.Fatalf("expected 3 retained backups after pruning, got %d", len(backups))
	}

	// Newest first: the last snapshot (data-e) must be the most recent.
	latest, err := os.ReadFile(backups[0].Path)
	if err != nil {
		t.Fatal(err)
	}
	if string(latest) != "data-e" {
		t.Fatalf("expected newest backup to contain 'data-e', got %q", latest)
	}
}

// TestBackupManagerResolve verifies lookup by full name and by timestamp.
func TestBackupManagerResolve(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "vault.db")
	if err := os.WriteFile(dbPath, []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}
	bm := NewBackupManager(dir, 3, nil)
	if err := bm.Snapshot(); err != nil {
		t.Fatal(err)
	}
	backups, err := bm.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(backups) != 1 {
		t.Fatalf("expected 1 backup, got %d", len(backups))
	}

	// Resolve by exact name.
	p1, err := bm.Resolve(backups[0].Name)
	if err != nil || p1 != backups[0].Path {
		t.Fatalf("resolve by name failed: %v (%s)", err, p1)
	}

	// Resolve by raw timestamp.
	ts := backups[0].Created.Format("20060102-150405")
	if len(ts) < 4 {
		t.Fatal("bad timestamp")
	}
	p2, err := bm.Resolve(ts)
	if err != nil || p2 != backups[0].Path {
		t.Fatalf("resolve by timestamp failed: %v (%s)", err, p2)
	}

	// Unknown name must error.
	if _, err := bm.Resolve("vault-19990101-000000.db"); err == nil {
		t.Fatal("expected error for unknown backup name")
	}
}

// TestBackupManagerRollback verifies a snapshot can be restored over the live db.
func TestBackupManagerRollback(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "vault.db")
	if err := os.WriteFile(dbPath, []byte("original"), 0600); err != nil {
		t.Fatal(err)
	}
	bm := NewBackupManager(dir, 3, nil)
	if err := bm.Snapshot(); err != nil {
		t.Fatal(err)
	}

	// Live db changes; rollback should restore the snapshot content.
	if err := os.WriteFile(dbPath, []byte("corrupted"), 0600); err != nil {
		t.Fatal(err)
	}
	backups, err := bm.List()
	if err != nil {
		t.Fatal(err)
	}
	if err := bm.RollbackFile(backups[0].Path); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "original" {
		t.Fatalf("expected rollback to restore 'original', got %q", got)
	}
}