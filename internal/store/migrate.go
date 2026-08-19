package store

import "fmt"

// Schema versioning uses SQLite's PRAGMA user_version, a 32-bit integer kept
// inside the database file itself. Version 1 is the baseline schema created by
// createSchema in db.go. Every later structural change is an ordered migration
// appended to pendingMigrations below, so the target version for a build is
// always baseline + len(pendingMigrations). This is how upgrades stay safe: a
// vault never needs to be rebuilt, only opened and migrated forward in place.
//
// Rules for contributors:
//   - Never edit, renumber, or reorder an existing entry — that would orphan
//     already-migrated databases and corrupt their version history.
//   - Only append new versions with strictly increasing version numbers.
//   - Migrations must be additive and non-destructive (CREATE TABLE IF NOT
//     EXISTS, ALTER TABLE ... ADD COLUMN with defaults). They run inside a
//     single transaction, so a failed migration rolls back to the prior schema
//     and the vault stays usable.
const currentSchemaBaseline = 1

// migration upgrades the schema from (currentSchemaBaseline+n-1) to
// (currentSchemaBaseline+n). Statements execute in order inside one atomic
// transaction.
type migration struct {
	version    int
	statements []string
}

// migrations MUST stay ordered, contiguous, and append-only by version.
//
// Example of a future change (add a column, then a new table):
//
//	migrations = append(migrations, migration{
//	  version: 2,
//	  statements: []string{
//	    "ALTER TABLE vault_fields ADD COLUMN note TEXT NOT NULL DEFAULT ''",
//	    "CREATE TABLE IF NOT EXISTS vault_bookmarks (id TEXT PRIMARY KEY, field_id TEXT NOT NULL)",
//	  },
//	})
var migrations = []migration{}

// Migrated reports whether this connection applied at least one real schema
// migration (a version newer than the baseline) since it was opened. The caller
// (vault.Open) uses this to trigger a safety snapshot right after an upgrade.
func (d *DB) Migrated() bool { return d.migrated }

// applyMigrations brings an opened database to the schema version this build
// targets, and enforces compatibility:
//
//   - A fresh or legacy DB (user_version 0) is aligned onto the baseline marker.
//     The baseline tables are already ensured by createSchema; aligning just
//     records that version 1 has been reached.
//   - Each pending migration is applied in order, atomically, and only commits
//     (and bumps user_version) once every statement succeeds.
//   - If the database was created by a NEWER binary (user_version above our
//     target), it refuses to proceed rather than risk touching a schema it does
//     not understand. The user must upgrade OmniVault first.
func (d *DB) applyMigrations() error {
	cur, err := d.userVersion()
	if err != nil {
		return err
	}
	target := currentSchemaBaseline + len(migrations)

	if cur > target {
		return fmt.Errorf(
			"database schema version %d is newer than this build supports (%d): upgrade OmniVault before opening this vault",
			cur, target)
	}
	if cur > 0 && cur < currentSchemaBaseline {
		return fmt.Errorf("unsupported database schema version %d", cur)
	}

	if cur == 0 {
		if err := d.setUserVersion(currentSchemaBaseline); err != nil {
			return err
		}
		cur = currentSchemaBaseline
	}

	for cur < target {
		applied := false
		for _, m := range migrations {
			if m.version != cur+1 {
				continue
			}
			if err := d.applyOne(m); err != nil {
				return fmt.Errorf("migrate schema to version %d: %w", m.version, err)
			}
			d.migrated = true
			cur = m.version
			applied = true
			break
		}
		if !applied {
			return fmt.Errorf("missing migration for schema version %d", cur+1)
		}
	}
	return nil
}

// applyOne runs one migration's statements in a single transaction, then bumps
// user_version. The DDL is atomic, so a mid-way failure leaves the previous
// schema intact.
func (d *DB) applyOne(m migration) error {
	tx, err := d.conn.Begin()
	if err != nil {
		return err
	}
	for _, stmt := range m.statements {
		if _, err := tx.Exec(stmt); err != nil {
			tx.Rollback()
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	return d.setUserVersion(m.version)
}

func (d *DB) userVersion() (int, error) {
	var v int
	if err := d.conn.QueryRow("PRAGMA user_version").Scan(&v); err != nil {
		return 0, fmt.Errorf("read schema version: %w", err)
	}
	return v, nil
}

func (d *DB) setUserVersion(v int) error {
	// PRAGMA user_version cannot be parameterized with a placeholder, but v is
	// always one of our own constants from pendingMigrations/baseline, so direct
	// interpolation is safe here.
	if _, err := d.conn.Exec(fmt.Sprintf("PRAGMA user_version = %d", v)); err != nil {
		return fmt.Errorf("write schema version: %w", err)
	}
	return nil
}