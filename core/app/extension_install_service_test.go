package app

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"aivo/core/domain"
	"aivo/core/infra/persistence"
)

func TestExtensionInstallPreviewDoesNotExecuteAndRejectsChangedConfirmation(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "aivo.db")
	store, err := persistence.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	service := NewService(store)
	defer service.Shutdown()

	root := t.TempDir()
	marker := filepath.Join(root, "executed")
	writeTestFile(t, filepath.Join(root, "bin", "extension"), "#!/bin/sh\ntouch "+marker+"\n", 0o700)
	writeTestExtensionManifest(t, root, map[string]any{
		"schemaVersion": 2, "id": "com.example.preview", "name": "Preview", "version": "1", "apiVersion": "2",
		"runtime": map[string]any{"type": "process", "command": "bin/extension", "transport": "stdio"},
	})
	writeTestFile(t, filepath.Join(root, "lib", "runtime.js"), "export const value = 1;", 0o600)

	preview, err := service.PreviewExtensionInstall(context.Background(), domain.PreviewExtensionInstallInput{Path: root})
	if err != nil {
		t.Fatal(err)
	}
	if !preview.Summary.Executable || preview.Summary.RuntimeType != domain.ExtensionRuntimeProcess || preview.Integrity == "" {
		t.Fatalf("preview = %#v", preview)
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatalf("preview executed extension: %v", err)
	}

	writeTestFile(t, filepath.Join(root, "lib", "runtime.js"), "export const value = 2;", 0o600)
	if _, err := service.InstallExtension(context.Background(), domain.InstallExtensionInput{Path: root, Integrity: preview.Integrity}); err == nil || !strings.Contains(err.Error(), "changed after preview") {
		t.Fatalf("changed package error = %v", err)
	}
	if installs, err := service.ListExtensionInstalls(context.Background()); err != nil || len(installs) != 0 {
		t.Fatalf("installs = %#v, err = %v", installs, err)
	}
}

func TestExtensionInstallCopiesToManagedStorageAndIgnoresSourceChanges(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "aivo.db")
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "context", "guide.md"), "safe context", 0o600)
	writeTestExtensionManifest(t, root, map[string]any{
		"schemaVersion": 2, "id": "com.example.static_install", "name": "Static install", "version": "1", "apiVersion": "2",
		"runtime":     map[string]any{"type": "static"},
		"contributes": map[string]any{"contexts": []any{map[string]any{"id": "guide", "kind": "instructions", "path": "context/guide.md"}}},
	})

	store, err := persistence.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	service := NewService(store)
	preview, err := service.PreviewExtensionInstall(context.Background(), domain.PreviewExtensionInstallInput{Path: root})
	if err != nil {
		t.Fatal(err)
	}
	installed, err := service.InstallExtension(context.Background(), domain.InstallExtensionInput{Path: root, Integrity: preview.Integrity, Enable: true})
	if err != nil || !installed.Enabled || installed.ID != "com.example.static_install" {
		t.Fatalf("installed = %#v, err = %v", installed, err)
	}
	managedRoot, err := store.ManagedExtensionRoot()
	if err != nil {
		t.Fatal(err)
	}
	managedRoot, err = filepath.EvalSymlinks(managedRoot)
	if err != nil {
		t.Fatal(err)
	}
	if installed.InstallMode != domain.ExtensionInstallModeManaged || installed.RootPath == root || !pathWithin(managedRoot, installed.RootPath) {
		t.Fatalf("extension was not copied into managed storage: %#v", installed)
	}
	managedContext := filepath.Join(installed.RootPath, "context", "guide.md")
	if info, statErr := os.Stat(managedContext); statErr != nil {
		t.Fatalf("managed package is missing: %v", statErr)
	} else if info.Mode().Perm()&0o222 != 0 {
		t.Fatalf("managed package is not read-only: mode=%v", info.Mode())
	}
	service.Shutdown()
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}

	reopened, err := persistence.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	restored := NewService(reopened)
	installs, err := restored.ListExtensionInstalls(context.Background())
	if err != nil || len(installs) != 1 || !installs[0].Enabled || installs[0].Status != domain.ExtensionStateReady {
		t.Fatalf("restored installs = %#v, err = %v", installs, err)
	}
	writeTestFile(t, filepath.Join(root, "context", "guide.md"), "changed context", 0o600)
	installs, err = restored.ListExtensionInstalls(context.Background())
	if err != nil || len(installs) != 1 || !installs[0].Enabled || installs[0].Status != domain.ExtensionStateReady {
		t.Fatalf("source change affected managed install = %#v, err = %v", installs, err)
	}
	if err := os.Chmod(managedContext, 0o600); err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, managedContext, "tampered managed context", 0o600)
	installs, err = restored.ListExtensionInstalls(context.Background())
	if err != nil || len(installs) != 1 || installs[0].Enabled || installs[0].Status != domain.ExtensionStateError || !strings.Contains(installs[0].Error, "modified") {
		t.Fatalf("tampered managed install = %#v, err = %v", installs, err)
	}
	if err := restored.UninstallExtension(context.Background(), domain.ExtensionControlInput{ID: installs[0].ID}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "aivo.extension.json")); err != nil {
		t.Fatalf("uninstall deleted source: %v", err)
	}
	if _, err := os.Stat(filepath.Join(managedRoot, installed.ID)); !os.IsNotExist(err) {
		t.Fatalf("uninstall retained managed package: %v", err)
	}
	if items, err := restored.ListExtensionInstalls(context.Background()); err != nil || len(items) != 0 {
		t.Fatalf("remaining installs = %#v, err = %v", items, err)
	}
	restored.Shutdown()
	_ = reopened.Close()
}

func TestExtensionInstallPromotesVerifiedLinkedRecordIntoManagedStorage(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "aivo.db")
	root := t.TempDir()
	writeTestExtensionManifest(t, root, map[string]any{
		"schemaVersion": 2, "id": "com.example.linked", "name": "Linked", "version": "1", "apiVersion": "2",
		"runtime": map[string]any{"type": "static"},
	})
	loaded, err := LoadExtensionManifest(root)
	if err != nil {
		t.Fatal(err)
	}
	store, err := persistence.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	_, err = store.SaveExtensionInstall(context.Background(), domain.ExtensionInstall{
		ID: loaded.Manifest.ID, Manifest: loaded.Manifest, Summary: domain.ExtensionSummary(loaded.Manifest),
		InstallMode: domain.ExtensionInstallModeLinked, RootPath: loaded.Root, ManifestPath: loaded.ManifestPath,
		Integrity: loaded.Integrity, Enabled: false, Status: domain.ExtensionStateStopped,
	})
	if err != nil {
		t.Fatal(err)
	}
	service := NewService(store)
	defer service.Shutdown()
	defer store.Close()
	installs, err := service.ListExtensionInstalls(context.Background())
	if err != nil || len(installs) != 1 {
		t.Fatalf("installs = %#v, err = %v", installs, err)
	}
	managedRoot, err := store.ManagedExtensionRoot()
	if err != nil {
		t.Fatal(err)
	}
	managedRoot, err = filepath.EvalSymlinks(managedRoot)
	if err != nil {
		t.Fatal(err)
	}
	if installs[0].InstallMode != domain.ExtensionInstallModeManaged || installs[0].RootPath == root || !pathWithin(managedRoot, installs[0].RootPath) {
		t.Fatalf("linked install was not promoted: %#v", installs[0])
	}
	if _, err := os.Stat(filepath.Join(root, "aivo.extension.json")); err != nil {
		t.Fatalf("promotion modified source: %v", err)
	}
	if err := service.UninstallExtension(context.Background(), domain.ExtensionControlInput{ID: installs[0].ID}); err != nil {
		t.Fatal(err)
	}
}

func TestExtensionInstallMovesFormerManagedRootIntoPlatformStorage(t *testing.T) {
	databaseDir := t.TempDir()
	dbPath := filepath.Join(databaseDir, "aivo.db")
	sourceRoot := t.TempDir()
	writeTestExtensionManifest(t, sourceRoot, map[string]any{
		"schemaVersion": 2, "id": "com.example.relocated", "name": "Relocated", "version": "1", "apiVersion": "2",
		"runtime": map[string]any{"type": "static"},
	})
	store, err := persistence.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	service := NewService(store)
	preview, err := service.PreviewExtensionInstall(context.Background(), domain.PreviewExtensionInstallInput{Path: sourceRoot})
	if err != nil {
		t.Fatal(err)
	}
	legacyInstall, err := service.InstallExtension(context.Background(), domain.InstallExtensionInput{Path: sourceRoot, Integrity: preview.Integrity})
	if err != nil {
		t.Fatal(err)
	}
	service.Shutdown()

	platformRoot := filepath.Join(t.TempDir(), "Aivo", "Default", "Extensions")
	relocatedStore := &relocatedExtensionStore{Store: store, managedRoot: platformRoot}
	restored := NewService(relocatedStore)
	defer func() {
		restored.Shutdown()
		_ = store.Close()
	}()
	installs, err := restored.ListExtensionInstalls(context.Background())
	if err != nil || len(installs) != 1 {
		t.Fatalf("installs = %#v, err = %v", installs, err)
	}
	canonicalPlatformRoot, err := filepath.EvalSymlinks(platformRoot)
	if err != nil {
		t.Fatal(err)
	}
	if installs[0].RootPath == legacyInstall.RootPath || !pathWithin(canonicalPlatformRoot, installs[0].RootPath) {
		t.Fatalf("managed package was not relocated: %#v", installs[0])
	}
	if _, err := os.Stat(filepath.Dir(legacyInstall.RootPath)); !os.IsNotExist(err) {
		t.Fatalf("former managed extension directory remains: %v", err)
	}
	if _, err := os.Stat(filepath.Join(sourceRoot, "aivo.extension.json")); err != nil {
		t.Fatalf("relocation modified original source: %v", err)
	}
	if err := restored.UninstallExtension(context.Background(), domain.ExtensionControlInput{ID: installs[0].ID}); err != nil {
		t.Fatal(err)
	}
}

type relocatedExtensionStore struct {
	*persistence.Store
	managedRoot string
}

func (s *relocatedExtensionStore) ManagedExtensionRoot() (string, error) {
	if err := os.MkdirAll(s.managedRoot, 0o700); err != nil {
		return "", err
	}
	if err := os.Chmod(s.managedRoot, 0o700); err != nil {
		return "", err
	}
	return s.managedRoot, nil
}

func TestExtensionUninstallRefusesManagedPathOutsideOwnedRoot(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "aivo.db")
	outside := t.TempDir()
	marker := filepath.Join(outside, "keep.txt")
	writeTestFile(t, marker, "keep", 0o600)
	store, err := persistence.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	manifest := domain.ExtensionManifest{SchemaVersion: 2, ID: "com.example.outside", Name: "Outside", Version: "1", APIVersion: "2", Runtime: domain.ExtensionRuntime{Type: domain.ExtensionRuntimeStatic}}
	_, err = store.SaveExtensionInstall(context.Background(), domain.ExtensionInstall{
		ID: manifest.ID, Manifest: manifest, Summary: domain.ExtensionSummary(manifest), InstallMode: domain.ExtensionInstallModeManaged,
		RootPath: outside, ManifestPath: filepath.Join(outside, "aivo.extension.json"), Integrity: strings.Repeat("a", 64), Status: domain.ExtensionStateStopped,
	})
	if err != nil {
		t.Fatal(err)
	}
	service := NewService(store)
	defer service.Shutdown()
	defer store.Close()
	if err := service.UninstallExtension(context.Background(), domain.ExtensionControlInput{ID: manifest.ID}); err == nil || !strings.Contains(err.Error(), "managed extension path") {
		t.Fatalf("uninstall error = %v", err)
	}
	if _, err := os.Stat(marker); err != nil {
		t.Fatalf("outside file was removed: %v", err)
	}
}

func TestExtensionInstallPersistenceRoundTripKeepsValidatedManifest(t *testing.T) {
	store, err := persistence.Open(filepath.Join(t.TempDir(), "aivo.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	manifest := domain.ExtensionManifest{SchemaVersion: 2, ID: "com.example.roundtrip", Name: "Roundtrip", Version: "1", APIVersion: "2", Runtime: domain.ExtensionRuntime{Type: domain.ExtensionRuntimeStatic}}
	install := domain.ExtensionInstall{ID: manifest.ID, Manifest: manifest, Summary: domain.ExtensionSummary(manifest), InstallMode: domain.ExtensionInstallModeManaged, RootPath: "/synthetic/extension", ManifestPath: "/synthetic/extension/aivo.extension.json", Integrity: strings.Repeat("a", 64), Status: domain.ExtensionStateValidated}
	saved, err := store.SaveExtensionInstall(context.Background(), install)
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(saved)
	if strings.Contains(string(raw), "schemaVersion") || saved.Summary.Name != manifest.Name || saved.InstallMode != domain.ExtensionInstallModeManaged {
		t.Fatalf("public install leaked full manifest or lost summary: %s, %#v", raw, saved)
	}
}
