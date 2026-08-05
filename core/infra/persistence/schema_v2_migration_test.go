package persistence

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSchemaV1MigrationCreatesBackupAndPersistsInitialWorkspace(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "aivo.db")
	writeSchemaV1ConfigFixture(t, dbPath)

	store, err := Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if !store.db.Migrator().HasColumn(&appConfigRow{}, "initial_workspace_path") {
		t.Fatal("schema v2 initial workspace column is missing")
	}
	var version int
	if err := store.db.Model(&schemaVersionRow{}).Select("MAX(version)").Scan(&version).Error; err != nil {
		t.Fatal(err)
	}
	if version != latestSchemaVersion {
		t.Fatalf("schema version = %d, want %d", version, latestSchemaVersion)
	}
	cfg, err := store.LoadConfig(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.Initialized {
		t.Fatal("v1 initialized state was not preserved")
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}

	backupPath := migrationBackupPath(dbPath, 1)
	if err := verifySQLiteBackup(backupPath); err != nil {
		t.Fatalf("backup is not recoverable: %v", err)
	}
	before, err := os.Stat(backupPath)
	if err != nil {
		t.Fatal(err)
	}
	reopened, err := Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	after, err := os.Stat(backupPath)
	if err != nil {
		t.Fatal(err)
	}
	if !before.ModTime().Equal(after.ModTime()) || before.Size() != after.Size() {
		t.Fatal("idempotent reopen rewrote the v1 backup")
	}
}

func TestSchemaV1MigrationRefusesInvalidExistingBackupBeforeMutation(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "aivo.db")
	writeSchemaV1ConfigFixture(t, dbPath)
	backupPath := migrationBackupPath(dbPath, 1)
	if err := os.WriteFile(backupPath, []byte("invalid backup"), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := Open(dbPath)
	if err == nil || !strings.Contains(err.Error(), "verify schema v1 migration backup") {
		t.Fatalf("error = %v", err)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	rows, err := db.Query("PRAGMA table_info(app_config)")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var cid, notNull, primaryKey int
		var name, columnType string
		var defaultValue any
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &primaryKey); err != nil {
			t.Fatal(err)
		}
		if name == "initial_workspace_path" {
			t.Fatal("schema mutated despite invalid backup")
		}
	}
}

func writeSchemaV1ConfigFixture(t *testing.T, path string) {
	t.Helper()
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	_, err = db.Exec(`
CREATE TABLE schema_version (version INTEGER PRIMARY KEY, applied_at TEXT NOT NULL);
INSERT INTO schema_version(version, applied_at) VALUES (1, '2026-01-01T00:00:00Z');
CREATE TABLE app_config (
  id INTEGER PRIMARY KEY,
  initialized INTEGER NOT NULL DEFAULT 0,
  updated_at TEXT NOT NULL
);
INSERT INTO app_config(id, initialized, updated_at) VALUES (1, 1, '2026-01-01T00:00:00Z');
`)
	if err != nil {
		t.Fatal(err)
	}
}
