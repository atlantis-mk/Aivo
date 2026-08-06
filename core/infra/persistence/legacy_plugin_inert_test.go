package persistence

import (
	"path/filepath"
	"testing"
)

func TestLegacyPluginRowsRemainUntouchedOnReopen(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "aivo.db")
	store, err := Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	legacy := pluginInstallRow{
		ID: "legacy-plugin", ManifestName: "legacy", Version: "1",
		RootPath: "/synthetic/legacy-plugin", ManifestPath: "/synthetic/legacy-plugin/aivo.plugin.json",
		Manifest: `{"id":"legacy-plugin","name":"legacy"}`, Enabled: 1, Status: "enabled",
		TimeCreated: "2026-08-01T00:00:00Z", TimeUpdated: "2026-08-01T00:00:00Z",
	}
	if err := store.db.Create(&legacy).Error; err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}

	reopened, err := Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	var actual pluginInstallRow
	if err := reopened.db.First(&actual, "id = ?", legacy.ID).Error; err != nil {
		t.Fatal(err)
	}
	if actual.Enabled != legacy.Enabled || actual.Status != legacy.Status || actual.Manifest != legacy.Manifest || actual.TimeUpdated != legacy.TimeUpdated {
		t.Fatalf("legacy row mutated on reopen: %#v", actual)
	}
}
