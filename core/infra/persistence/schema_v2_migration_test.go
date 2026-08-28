package persistence

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"aivo/core/domain"
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

func TestSchemaV4MigrationCreatesBackupAndGlobalToolPreferences(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "aivo.db")
	writeSchemaV4Fixture(t, dbPath)

	store, err := Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if !store.db.Migrator().HasTable(&globalToolPreferenceRow{}) {
		t.Fatal("schema v5 global_tool_preferences table is missing")
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	backupPath := migrationBackupPath(dbPath, 4)
	if err := verifySQLiteBackup(backupPath); err != nil {
		t.Fatalf("v4 backup is not recoverable: %v", err)
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
		t.Fatal("idempotent reopen rewrote the v4 backup")
	}
}

func TestSchemaV4MigrationRefusesInvalidBackupBeforeGlobalToolPreferenceMutation(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "aivo.db")
	writeSchemaV4Fixture(t, dbPath)
	backupPath := migrationBackupPath(dbPath, 4)
	if err := os.WriteFile(backupPath, []byte("invalid backup"), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := Open(dbPath)
	if err == nil || !strings.Contains(err.Error(), "verify schema v4 migration backup") {
		t.Fatalf("error = %v", err)
	}
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='global_tool_preferences'").Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatal("schema mutated despite invalid v4 backup")
	}
}

func TestSchemaV5MigrationCreatesBackupAndAgentModeDefinitions(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "aivo.db")
	writeSchemaV5Fixture(t, dbPath)

	store, err := Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if !store.db.Migrator().HasTable(&agentModeDefinitionRow{}) {
		t.Fatal("schema v6 agent_mode_definitions table is missing")
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	backupPath := migrationBackupPath(dbPath, 5)
	if err := verifySQLiteBackup(backupPath); err != nil {
		t.Fatalf("v5 backup is not recoverable: %v", err)
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
		t.Fatal("idempotent reopen rewrote the v5 backup")
	}
}

func TestSchemaV5MigrationRefusesInvalidBackupBeforeAgentModeMutation(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "aivo.db")
	writeSchemaV5Fixture(t, dbPath)
	backupPath := migrationBackupPath(dbPath, 5)
	if err := os.WriteFile(backupPath, []byte("invalid backup"), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := Open(dbPath)
	if err == nil || !strings.Contains(err.Error(), "verify schema v5 migration backup") {
		t.Fatalf("error = %v", err)
	}
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='agent_mode_definitions'").Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatal("schema mutated despite invalid v5 backup")
	}
}

func TestSchemaV6MigrationCreatesBackupAndRemovesAgentModeToolsets(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "aivo.db")
	writeSchemaV6Fixture(t, dbPath)

	store, err := Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	var definition string
	if err := store.db.Model(&agentModeDefinitionRow{}).Where("id = ?", "research").Pluck("definition", &definition).Error; err != nil {
		t.Fatal(err)
	}
	if strings.Contains(definition, "toolsets") {
		t.Fatalf("schema v7 retained toolsets: %s", definition)
	}
	backupPath := migrationBackupPath(dbPath, 6)
	if err := verifySQLiteBackup(backupPath); err != nil {
		t.Fatalf("v6 backup is not recoverable: %v", err)
	}
	backup, err := sql.Open("sqlite", backupPath)
	if err != nil {
		t.Fatal(err)
	}
	defer backup.Close()
	if err := backup.QueryRow("SELECT definition FROM agent_mode_definitions WHERE id = ?", "research").Scan(&definition); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(definition, "toolsets") {
		t.Fatalf("v6 backup did not preserve removed data: %s", definition)
	}
}

func TestSchemaV6MigrationRefusesInvalidBackupBeforeToolsetCleanup(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "aivo.db")
	writeSchemaV6Fixture(t, dbPath)
	backupPath := migrationBackupPath(dbPath, 6)
	if err := os.WriteFile(backupPath, []byte("invalid backup"), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := Open(dbPath)
	if err == nil || !strings.Contains(err.Error(), "verify schema v6 migration backup") {
		t.Fatalf("error = %v", err)
	}
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var definition string
	if err := db.QueryRow("SELECT definition FROM agent_mode_definitions WHERE id = ?", "research").Scan(&definition); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(definition, "toolsets") {
		t.Fatalf("schema mutated despite invalid v6 backup: %s", definition)
	}
}

func TestSchemaV6MigrationRollsBackMalformedAgentModeCleanup(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "aivo.db")
	writeSchemaV6Fixture(t, dbPath)
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("UPDATE agent_mode_definitions SET definition = ? WHERE id = ?", "{invalid", "research"); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	_, err = Open(dbPath)
	if err == nil || !strings.Contains(err.Error(), "decode agent mode research for schema v7") {
		t.Fatalf("error = %v", err)
	}
	db, err = sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var version int
	if err := db.QueryRow("SELECT MAX(version) FROM schema_version").Scan(&version); err != nil {
		t.Fatal(err)
	}
	if version != 6 {
		t.Fatalf("schema version = %d after failed cleanup, want 6", version)
	}
}

func TestSchemaV7MigrationCreatesBackupAndPreservesAgentModePayloads(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "aivo.db")
	writeSchemaV7Fixture(t, dbPath)

	store, err := Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	var version int
	if err := store.db.Model(&schemaVersionRow{}).Select("MAX(version)").Scan(&version).Error; err != nil {
		t.Fatal(err)
	}
	if version != latestSchemaVersion {
		t.Fatalf("schema version = %d, want %d", version, latestSchemaVersion)
	}
	var definition string
	if err := store.db.Model(&agentModeDefinitionRow{}).Where("id = ?", "research").Pluck("definition", &definition).Error; err != nil {
		t.Fatal(err)
	}
	const expected = `{"description":"Read-only research","displayName":"Research","id":"research","mode":"all","permissionScope":"read_only","promptId":"agent.research"}`
	if definition != expected {
		t.Fatalf("schema v9 agent prompt migration payload = %s", definition)
	}
	promptPath := filepath.Join(filepath.Dir(dbPath), "prompts", "overrides", "agent", "agent.research.md")
	prompt, err := os.ReadFile(promptPath)
	if err != nil || !strings.Contains(string(prompt), "Investigate carefully.") {
		t.Fatalf("schema v9 prompt extraction failed: %v %q", err, prompt)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	backupPath := migrationBackupPath(dbPath, 7)
	if err := verifySQLiteBackup(backupPath); err != nil {
		t.Fatalf("v7 backup is not recoverable: %v", err)
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
		t.Fatal("idempotent reopen rewrote the v7 backup")
	}
}

func TestSchemaV7MigrationRefusesInvalidBackupBeforeVersionMutation(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "aivo.db")
	writeSchemaV7Fixture(t, dbPath)
	backupPath := migrationBackupPath(dbPath, 7)
	if err := os.WriteFile(backupPath, []byte("invalid backup"), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := Open(dbPath)
	if err == nil || !strings.Contains(err.Error(), "verify schema v7 migration backup") {
		t.Fatalf("error = %v", err)
	}
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var version int
	if err := db.QueryRow("SELECT MAX(version) FROM schema_version").Scan(&version); err != nil {
		t.Fatal(err)
	}
	if version != 7 {
		t.Fatalf("schema version = %d after invalid backup, want 7", version)
	}
}

func TestSchemaV9MigrationCreatesBackupAndDefaultsAppName(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "aivo.db")
	writeSchemaV9Fixture(t, dbPath)

	store, err := Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if !store.db.Migrator().HasColumn(&appConfigRow{}, "app_name") {
		t.Fatal("schema v10 app name column is missing")
	}
	cfg, err := store.LoadConfig(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if cfg.AppName != "Aivo" || !cfg.Initialized || cfg.InitialWorkspacePath != "/synthetic/workspace" {
		t.Fatalf("migrated config = %#v", cfg)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	backupPath := migrationBackupPath(dbPath, 9)
	if err := verifySQLiteBackup(backupPath); err != nil {
		t.Fatalf("v9 backup is not recoverable: %v", err)
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
		t.Fatal("idempotent reopen rewrote the v9 backup")
	}
}

func TestSchemaV9MigrationRefusesInvalidBackupBeforeAppNameMutation(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "aivo.db")
	writeSchemaV9Fixture(t, dbPath)
	backupPath := migrationBackupPath(dbPath, 9)
	if err := os.WriteFile(backupPath, []byte("invalid backup"), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := Open(dbPath)
	if err == nil || !strings.Contains(err.Error(), "verify schema v9 migration backup") {
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
		if name == "app_name" {
			t.Fatal("schema mutated despite invalid v9 backup")
		}
	}
}

func TestSchemaV10MigrationCreatesBackupAndDefaultsPermissionMode(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "aivo.db")
	writeSchemaV10Fixture(t, dbPath)

	store, err := Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if !store.db.Migrator().HasColumn(&appConfigRow{}, "default_permission_mode") {
		t.Fatal("schema v11 default permission mode column is missing")
	}
	cfg, err := store.LoadConfig(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DefaultPermissionMode != domain.PermissionModeRequestApproval {
		t.Fatalf("default permission mode = %q, want request approval", cfg.DefaultPermissionMode)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	backupPath := migrationBackupPath(dbPath, 10)
	if err := verifySQLiteBackup(backupPath); err != nil {
		t.Fatal(err)
	}
}

func TestSchemaV10MigrationRefusesInvalidBackupBeforePermissionModeMutation(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "aivo.db")
	writeSchemaV10Fixture(t, dbPath)
	backupPath := migrationBackupPath(dbPath, 10)
	if err := os.WriteFile(backupPath, []byte("invalid backup"), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := Open(dbPath)
	if err == nil || !strings.Contains(err.Error(), "verify schema v10 migration backup") {
		t.Fatalf("error = %v", err)
	}
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM pragma_table_info('app_config') WHERE name = 'default_permission_mode'").Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatal("schema mutated despite invalid v10 backup")
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

func writeSchemaV4Fixture(t *testing.T, path string) {
	t.Helper()
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	_, err = db.Exec(`
CREATE TABLE schema_version (version INTEGER PRIMARY KEY, applied_at TEXT NOT NULL);
INSERT INTO schema_version(version, applied_at) VALUES (4, '2026-01-01T00:00:00Z');
CREATE TABLE extension_installs (
  id TEXT PRIMARY KEY,
  manifest TEXT NOT NULL,
  root_path TEXT NOT NULL UNIQUE,
  manifest_path TEXT NOT NULL,
  integrity TEXT NOT NULL,
  install_mode TEXT NOT NULL DEFAULT 'linked',
  enabled INTEGER NOT NULL DEFAULT 0,
  status TEXT NOT NULL,
  error TEXT,
  time_created TEXT NOT NULL,
  time_updated TEXT NOT NULL
);
`)
	if err != nil {
		t.Fatal(err)
	}
}

func writeSchemaV5Fixture(t *testing.T, path string) {
	t.Helper()
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	_, err = db.Exec(`
CREATE TABLE schema_version (version INTEGER PRIMARY KEY, applied_at TEXT NOT NULL);
INSERT INTO schema_version(version, applied_at) VALUES (5, '2026-01-01T00:00:00Z');
CREATE TABLE global_tool_preferences (
  name TEXT PRIMARY KEY,
  enabled INTEGER NOT NULL DEFAULT 0,
  time_created TEXT NOT NULL,
  time_updated TEXT NOT NULL
);
`)
	if err != nil {
		t.Fatal(err)
	}
}

func writeSchemaV6Fixture(t *testing.T, path string) {
	t.Helper()
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	_, err = db.Exec(`
CREATE TABLE schema_version (version INTEGER PRIMARY KEY, applied_at TEXT NOT NULL);
INSERT INTO schema_version(version, applied_at) VALUES (6, '2026-01-01T00:00:00Z');
CREATE TABLE agent_mode_definitions (
  id TEXT PRIMARY KEY,
  definition TEXT NOT NULL,
  time_created TEXT NOT NULL,
  time_updated TEXT NOT NULL
);
INSERT INTO agent_mode_definitions(id, definition, time_created, time_updated) VALUES (
  'research',
  '{"id":"research","displayName":"Research","description":"Read-only research","prompt":"Investigate carefully.","toolsets":["safe","web"],"permissionScope":"read_only","mode":"all"}',
  '2026-01-01T00:00:00Z',
  '2026-01-01T00:00:00Z'
);
`)
	if err != nil {
		t.Fatal(err)
	}
}

func writeSchemaV7Fixture(t *testing.T, path string) {
	t.Helper()
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	_, err = db.Exec(`
CREATE TABLE schema_version (version INTEGER PRIMARY KEY, applied_at TEXT NOT NULL);
INSERT INTO schema_version(version, applied_at) VALUES (7, '2026-01-01T00:00:00Z');
CREATE TABLE agent_mode_definitions (
  id TEXT PRIMARY KEY,
  definition TEXT NOT NULL,
  time_created TEXT NOT NULL,
  time_updated TEXT NOT NULL
);
INSERT INTO agent_mode_definitions(id, definition, time_created, time_updated) VALUES (
  'research',
  '{"id":"research","displayName":"Research","description":"Read-only research","prompt":"Investigate carefully.","permissionScope":"read_only","mode":"all"}',
  '2026-01-01T00:00:00Z',
  '2026-01-01T00:00:00Z'
);
`)
	if err != nil {
		t.Fatal(err)
	}
}

func writeSchemaV9Fixture(t *testing.T, path string) {
	t.Helper()
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	_, err = db.Exec(`
CREATE TABLE schema_version (version INTEGER PRIMARY KEY, applied_at TEXT NOT NULL);
INSERT INTO schema_version(version, applied_at) VALUES (9, '2026-01-01T00:00:00Z');
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

func writeSchemaV10Fixture(t *testing.T, path string) {
	t.Helper()
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	_, err = db.Exec(`
CREATE TABLE schema_version (version INTEGER PRIMARY KEY, applied_at TEXT NOT NULL);
INSERT INTO schema_version(version, applied_at) VALUES (10, '2026-01-01T00:00:00Z');
CREATE TABLE app_config (
  id INTEGER PRIMARY KEY,
  initialized INTEGER NOT NULL DEFAULT 0,
  app_name TEXT,
  initial_workspace_path TEXT,
  updated_at TEXT NOT NULL
);
INSERT INTO app_config(id, initialized, app_name, initial_workspace_path, updated_at)
VALUES (1, 1, 'Aivo', '/synthetic/workspace', '2026-01-01T00:00:00Z');
`)
	if err != nil {
		t.Fatal(err)
	}
}
