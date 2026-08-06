package persistence

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDefaultManagedExtensionRootUsesPlatformApplicationData(t *testing.T) {
	configRoot, err := os.UserConfigDir()
	if err != nil {
		t.Fatal(err)
	}
	root, err := defaultManagedExtensionRoot()
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(configRoot, "Aivo", "Default", "Extensions")
	if root != want {
		t.Fatalf("managed root = %q, want %q", root, want)
	}
}

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

func TestSchemaV2MigrationCreatesBackupAndExtensionInstallTable(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "aivo.db")
	writeSchemaV2Fixture(t, dbPath)

	store, err := Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if !store.db.Migrator().HasTable(&extensionInstallRow{}) {
		t.Fatal("schema v3 extension_installs table is missing")
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	backupPath := migrationBackupPath(dbPath, 2)
	if err := verifySQLiteBackup(backupPath); err != nil {
		t.Fatalf("v2 backup is not recoverable: %v", err)
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
		t.Fatal("idempotent reopen rewrote the v2 backup")
	}
}

func TestSchemaV2MigrationRefusesInvalidBackupBeforeExtensionTableMutation(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "aivo.db")
	writeSchemaV2Fixture(t, dbPath)
	backupPath := migrationBackupPath(dbPath, 2)
	if err := os.WriteFile(backupPath, []byte("invalid backup"), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := Open(dbPath)
	if err == nil || !strings.Contains(err.Error(), "verify schema v2 migration backup") {
		t.Fatalf("error = %v", err)
	}
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='extension_installs'").Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatal("schema mutated despite invalid v2 backup")
	}
}

func TestSchemaV3MigrationCreatesBackupAndMarksExistingExtensionsLinked(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "aivo.db")
	writeSchemaV3Fixture(t, dbPath)

	store, err := Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if !store.db.Migrator().HasColumn(&extensionInstallRow{}, "install_mode") {
		t.Fatal("schema v4 extension install mode is missing")
	}
	install, err := store.GetExtensionInstall(context.Background(), "com.example.legacy")
	if err != nil {
		t.Fatal(err)
	}
	if install.InstallMode != "linked" {
		t.Fatalf("install mode = %q, want linked", install.InstallMode)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	backupPath := migrationBackupPath(dbPath, 3)
	if err := verifySQLiteBackup(backupPath); err != nil {
		t.Fatalf("v3 backup is not recoverable: %v", err)
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
		t.Fatal("idempotent reopen rewrote the v3 backup")
	}
}

func TestSchemaV3MigrationRefusesInvalidBackupBeforeInstallModeMutation(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "aivo.db")
	writeSchemaV3Fixture(t, dbPath)
	backupPath := migrationBackupPath(dbPath, 3)
	if err := os.WriteFile(backupPath, []byte("invalid backup"), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := Open(dbPath)
	if err == nil || !strings.Contains(err.Error(), "verify schema v3 migration backup") {
		t.Fatalf("error = %v", err)
	}
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	rows, err := db.Query("PRAGMA table_info(extension_installs)")
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
		if name == "install_mode" {
			t.Fatal("schema mutated despite invalid v3 backup")
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

func writeSchemaV2Fixture(t *testing.T, path string) {
	t.Helper()
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	_, err = db.Exec(`
CREATE TABLE schema_version (version INTEGER PRIMARY KEY, applied_at TEXT NOT NULL);
INSERT INTO schema_version(version, applied_at) VALUES (2, '2026-01-01T00:00:00Z');
CREATE TABLE app_config (
  id INTEGER PRIMARY KEY,
  initialized INTEGER NOT NULL DEFAULT 0,
  initial_workspace_path TEXT,
  updated_at TEXT NOT NULL
);
INSERT INTO app_config(id, initialized, initial_workspace_path, updated_at)
VALUES (1, 1, '/synthetic/workspace', '2026-01-01T00:00:00Z');
`)
	if err != nil {
		t.Fatal(err)
	}
}

func writeSchemaV3Fixture(t *testing.T, path string) {
	t.Helper()
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	_, err = db.Exec(`
CREATE TABLE schema_version (version INTEGER PRIMARY KEY, applied_at TEXT NOT NULL);
INSERT INTO schema_version(version, applied_at) VALUES (3, '2026-01-01T00:00:00Z');
CREATE TABLE extension_installs (
  id TEXT PRIMARY KEY,
  manifest TEXT NOT NULL,
  root_path TEXT NOT NULL UNIQUE,
  manifest_path TEXT NOT NULL,
  integrity TEXT NOT NULL,
  enabled INTEGER NOT NULL DEFAULT 0,
  status TEXT NOT NULL,
  error TEXT,
  time_created TEXT NOT NULL,
  time_updated TEXT NOT NULL
);
INSERT INTO extension_installs(
  id, manifest, root_path, manifest_path, integrity, enabled, status, error, time_created, time_updated
) VALUES (
  'com.example.legacy',
  '{"schemaVersion":2,"id":"com.example.legacy","name":"Legacy","version":"1","apiVersion":"2","runtime":{"type":"static"}}',
  '/synthetic/legacy', '/synthetic/legacy/aivo.extension.json',
  'aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa',
  0, 'stopped', '', '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z'
);
`)
	if err != nil {
		t.Fatal(err)
	}
}
