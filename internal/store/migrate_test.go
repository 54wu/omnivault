package store

import (
	"path/filepath"
	"strings"
	"testing"
)

// withMigrations temporarily extends the migrations table for one test and
// restores the original afterwards.
func withMigrations(t *testing.T, extra ...migration) {
	t.Helper()
	orig := migrations
	migrations = append(append([]migration{}, orig...), extra...)
	t.Cleanup(func() { migrations = orig })
}

func openAt(t *testing.T, path string) *DB {
	t.Helper()
	db, err := Open(path)
	if err != nil {
		t.Fatalf("open %s: %v", path, err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func readUserVersion(t *testing.T, db *DB) int {
	t.Helper()
	v, err := db.userVersion()
	if err != nil {
		t.Fatalf("read user_version: %v", err)
	}
	return v
}

func TestOpen_SetsBaselineVersion(t *testing.T) {
	db := openAt(t, filepath.Join(t.TempDir(), "v.db"))

	if got := readUserVersion(t, db); got != currentSchemaBaseline {
		t.Fatalf("expected baseline version %d, got %d", currentSchemaBaseline, got)
	}
	if db.Migrated() {
		t.Fatal("a fresh database must not be reported as migrated")
	}
}

func TestMigration_AppliedAndRecorded(t *testing.T) {
	withMigrations(t, migration{
		version:    2,
		statements: []string{"ALTER TABLE vault_fields ADD COLUMN note TEXT NOT NULL DEFAULT ''"},
	})

	db := openAt(t, filepath.Join(t.TempDir(), "v.db"))

	if got := readUserVersion(t, db); got != currentSchemaBaseline+1 {
		t.Fatalf("expected version %d after migration, got %d", currentSchemaBaseline+1, got)
	}
	if !db.Migrated() {
		t.Fatal("expected the migrated flag to be set")
	}

	rows, err := db.conn.Query("PRAGMA table_info(vault_fields)")
	if err != nil {
		t.Fatalf("query table_info: %v", err)
	}
	defer rows.Close()
	found := false
	for rows.Next() {
		var cid int
		var name, typ string
		var notnull, pk int
		var dflt any
		if err := rows.Scan(&cid, &name, &typ, &notnull, &dflt, &pk); err != nil {
			t.Fatalf("scan table_info: %v", err)
		}
		if name == "note" {
			found = true
		}
	}
	if !found {
		t.Fatal("migrated column 'note' not found in vault_fields")
	}
}

func TestMigration_IdempotentOnReopen(t *testing.T) {
	withMigrations(t, migration{
		version:    2,
		statements: []string{"ALTER TABLE vault_fields ADD COLUMN note TEXT NOT NULL DEFAULT ''"},
	})

	path := filepath.Join(t.TempDir(), "v.db")
	first := openAt(t, path)
	first.Close()

	// Reopening an already-migrated database must not re-run or flag a migration.
	second := openAt(t, path)
	if readUserVersion(t, second) != currentSchemaBaseline+1 {
		t.Fatal("expected version to persist across reopen")
	}
	if second.Migrated() {
		t.Fatal("reopening an up-to-date schema must not be reported as migrated")
	}
}

func TestMigration_FailureRollsBack(t *testing.T) {
	withMigrations(t, migration{
		version: 2,
		statements: []string{
			"ALTER TABLE vault_fields ADD COLUMN note TEXT NOT NULL DEFAULT ''",
			"THIS IS NOT VALID SQL",
		},
	})

	path := filepath.Join(t.TempDir(), "v.db")
	if _, err := Open(path); err == nil {
		t.Fatal("expected Open to fail when a migration statement is invalid")
	}

	// Reopen with migrations cleared so we can inspect the rolled-back state
	// instead of re-attempting the failing migration.
	saved := migrations
	migrations = nil
	defer func() { migrations = saved }()

	// The failed migration must have rolled back the good statement too, and the
	// version must remain at the baseline so a later successful run can retry.
	db := openAt(t, path)
	if got := readUserVersion(t, db); got != currentSchemaBaseline {
		t.Fatalf("expected version to stay at baseline after failure, got %d", got)
	}
	if db.Migrated() {
		t.Fatal("failed migration must not set the migrated flag")
	}
	var name string
	err := db.conn.QueryRow(
		"SELECT name FROM sqlite_master WHERE type='table' AND name='vault_fields'").Scan(&name)
	if err != nil {
		t.Fatalf("vault_fields should still exist after rollback: %v", err)
	}
	if hasColumn(t, db, "note") {
		t.Fatal("rolled-back column 'note' must not linger")
	}
}

func TestOpen_DowngradeGuard(t *testing.T) {
	if len(migrations) != 0 {
		t.Fatal("this test expects no future migrations present")
	}
	path := filepath.Join(t.TempDir(), "v.db")
	db := openAt(t, path)
	if _, err := db.conn.Exec("PRAGMA user_version = 99"); err != nil {
		t.Fatalf("seed higher version: %v", err)
	}
	db.Close()

	// A build targeting version 1 must refuse a database at version 99, rather
	// than risk reading a schema it does not understand.
	if _, err := Open(path); err == nil {
		t.Fatal("expected Open to reject a database from a newer build")
	} else if !strings.Contains(err.Error(), "newer than") {
		t.Fatalf("expected a downgrade-guard error, got: %v", err)
	}
}

func hasColumn(t *testing.T, db *DB, target string) bool {
	t.Helper()
	rows, err := db.conn.Query("PRAGMA table_info(vault_fields)")
	if err != nil {
		t.Fatalf("query table_info: %v", err)
	}
	defer rows.Close()
	for rows.Next() {
		var cid int
		var name, typ string
		var notnull, pk int
		var dflt any
		if err := rows.Scan(&cid, &name, &typ, &notnull, &dflt, &pk); err != nil {
			t.Fatalf("scan table_info: %v", err)
		}
		if name == target {
			return true
		}
	}
	return false
}