package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"aivo/core/domain"
)

func (s *Service) PreviewExtensionInstall(_ context.Context, input domain.PreviewExtensionInstallInput) (domain.ExtensionInstallPreview, error) {
	loaded, err := LoadExtensionManifest(input.Path)
	if err != nil {
		return domain.ExtensionInstallPreview{}, err
	}
	preview := extensionInstallPreview(loaded)
	if store, ok := s.store.(extensionInstallStore); ok {
		if installs, listErr := store.ListExtensionInstalls(context.Background()); listErr == nil {
			for _, install := range installs {
				if install.ID == loaded.Manifest.ID {
					preview.Update = true
					break
				}
			}
		}
	}
	return preview, nil
}

func (s *Service) InstallExtension(ctx context.Context, input domain.InstallExtensionInput) (domain.ExtensionInstall, error) {
	store, ok := s.store.(extensionInstallStore)
	if !ok {
		return domain.ExtensionInstall{}, errors.New("extension installation store is unavailable")
	}
	loaded, err := LoadExtensionManifest(input.Path)
	if err != nil {
		return domain.ExtensionInstall{}, err
	}
	if strings.TrimSpace(input.Integrity) == "" || loaded.Integrity != strings.TrimSpace(input.Integrity) {
		return domain.ExtensionInstall{}, errors.New("extension package changed after preview; review it again before installing")
	}
	loaded, managedCreated, err := installManagedExtensionPackage(store, loaded)
	if err != nil {
		return domain.ExtensionInstall{}, err
	}

	existing := false
	if installs, listErr := store.ListExtensionInstalls(ctx); listErr == nil {
		for _, install := range installs {
			if install.ID == loaded.Manifest.ID {
				existing = true
				break
			}
		}
	}
	if existing && !input.Enable {
		if _, statusErr := s.extensionSupervisor.Status(loaded.Manifest.ID); statusErr == nil {
			if removeErr := s.extensionSupervisor.Remove(ctx, loaded.Manifest.ID); removeErr != nil {
				return domain.ExtensionInstall{}, removeErr
			}
		}
	}

	status, err := s.extensionSupervisor.Discover(loaded.Root)
	if err != nil {
		if managedCreated && !existing {
			_ = removeManagedGeneration(store, loaded)
		}
		return domain.ExtensionInstall{}, err
	}
	if status.Integrity != loaded.Integrity {
		if !existing {
			_ = s.extensionSupervisor.Remove(context.Background(), loaded.Manifest.ID)
			if managedCreated {
				_ = removeManagedGeneration(store, loaded)
			}
		}
		return domain.ExtensionInstall{}, errors.New("extension discovery integrity does not match the confirmed package")
	}
	status, err = s.extensionSupervisor.Trust(loaded.Manifest.ID, loaded.Integrity)
	if err != nil {
		if !existing {
			_ = s.extensionSupervisor.Remove(context.Background(), loaded.Manifest.ID)
			if managedCreated {
				_ = removeManagedGeneration(store, loaded)
			}
		}
		return domain.ExtensionInstall{}, err
	}
	install := domain.ExtensionInstall{
		ID: loaded.Manifest.ID, Manifest: loaded.Manifest, Summary: domain.ExtensionSummary(loaded.Manifest),
		InstallMode: domain.ExtensionInstallModeManaged,
		RootPath:    loaded.Root, ManifestPath: loaded.ManifestPath, Integrity: loaded.Integrity,
		Enabled: false, Status: status.State,
	}
	install, err = store.SaveExtensionInstall(ctx, install)
	if err != nil {
		if !existing {
			_ = s.extensionSupervisor.Remove(context.Background(), loaded.Manifest.ID)
			if managedCreated {
				_ = removeManagedGeneration(store, loaded)
			}
		}
		return domain.ExtensionInstall{}, err
	}
	if input.Enable {
		status, err = s.extensionSupervisor.Enable(ctx, loaded.Manifest.ID)
		if err != nil {
			install, _ = store.SetExtensionInstallState(ctx, loaded.Manifest.ID, false, domain.ExtensionStateError, err.Error())
			return install, err
		}
		install, err = store.SetExtensionInstallState(ctx, loaded.Manifest.ID, true, status.State, status.Error)
		if err != nil {
			_, _ = s.extensionSupervisor.Stop(context.Background(), loaded.Manifest.ID)
			return domain.ExtensionInstall{}, err
		}
	}
	s.refreshProviderExtensions("")
	return install, nil
}

func (s *Service) ListExtensionInstalls(ctx context.Context) ([]domain.ExtensionInstall, error) {
	store, ok := s.store.(extensionInstallStore)
	if !ok {
		return nil, errors.New("extension installation store is unavailable")
	}
	installs, err := store.ListExtensionInstalls(ctx)
	if err != nil {
		return nil, err
	}
	for index := range installs {
		install := installs[index]
		install, prepareErr := ensureManagedExtensionInstall(ctx, store, install)
		if prepareErr != nil {
			message := prepareErr.Error()
			if _, statusErr := s.extensionSupervisor.Status(install.ID); statusErr == nil {
				_, _ = s.extensionSupervisor.Stop(ctx, install.ID)
			}
			updated, updateErr := store.SetExtensionInstallState(ctx, install.ID, false, domain.ExtensionStateError, message)
			if updateErr == nil {
				installs[index] = updated
			}
			continue
		}
		installs[index] = install
		if status, statusErr := s.extensionSupervisor.Status(install.ID); statusErr == nil {
			install.Enabled = status.Enabled
			install.Status = status.State
			install.Error = status.Error
			installs[index] = install
		}
	}
	return installs, nil
}

func (s *Service) SetExtensionInstalledEnabled(ctx context.Context, input domain.SetExtensionEnabledInput) (domain.ExtensionInstall, error) {
	store, ok := s.store.(extensionInstallStore)
	if !ok {
		return domain.ExtensionInstall{}, errors.New("extension installation store is unavailable")
	}
	install, err := store.GetExtensionInstall(ctx, input.ID)
	if err != nil {
		return domain.ExtensionInstall{}, err
	}
	if !input.Enabled {
		if _, statusErr := s.extensionSupervisor.Status(install.ID); statusErr == nil {
			if _, err = s.extensionSupervisor.Stop(ctx, install.ID); err != nil {
				return domain.ExtensionInstall{}, err
			}
		}
		s.refreshProviderExtensions("")
		return store.SetExtensionInstallState(ctx, install.ID, false, domain.ExtensionStateStopped, "")
	}
	install, err = ensureManagedExtensionInstall(ctx, store, install)
	if err != nil {
		message := err.Error()
		updated, _ := store.SetExtensionInstallState(ctx, install.ID, false, domain.ExtensionStateError, message)
		return updated, errors.New(message)
	}
	if _, err = s.extensionSupervisor.Discover(install.RootPath); err != nil {
		return domain.ExtensionInstall{}, err
	}
	if _, err = s.extensionSupervisor.Trust(install.ID, install.Integrity); err != nil {
		return domain.ExtensionInstall{}, err
	}
	status, err := s.extensionSupervisor.Enable(ctx, install.ID)
	if err != nil {
		updated, _ := store.SetExtensionInstallState(ctx, install.ID, false, domain.ExtensionStateError, err.Error())
		return updated, err
	}
	s.refreshProviderExtensions("")
	return store.SetExtensionInstallState(ctx, install.ID, true, status.State, status.Error)
}

func (s *Service) UninstallExtension(ctx context.Context, input domain.ExtensionControlInput) error {
	store, ok := s.store.(extensionInstallStore)
	if !ok {
		return errors.New("extension installation store is unavailable")
	}
	install, err := store.GetExtensionInstall(ctx, input.ID)
	if err != nil {
		return err
	}
	if _, statusErr := s.extensionSupervisor.Status(install.ID); statusErr == nil {
		if err = s.extensionSupervisor.Remove(ctx, install.ID); err != nil {
			return err
		}
	}
	if install.InstallMode != domain.ExtensionInstallModeManaged {
		if err = store.DeleteExtensionInstall(ctx, install.ID); err != nil {
			return err
		}
		s.refreshProviderExtensions("")
		return nil
	}
	managedRoot, err := canonicalManagedExtensionRoot(store)
	if err != nil {
		return err
	}
	idRoot, err := validateManagedInstallPath(managedRoot, install)
	if err != nil {
		return err
	}
	quarantined := ""
	if _, statErr := os.Lstat(idRoot); statErr == nil {
		quarantined, err = reserveManagedRemovalPath(managedRoot)
		if err != nil {
			return err
		}
		if err = os.Rename(idRoot, quarantined); err != nil {
			return fmt.Errorf("prepare managed extension removal: %w", err)
		}
	} else if !errors.Is(statErr, os.ErrNotExist) {
		return statErr
	}
	if err = store.DeleteExtensionInstall(ctx, install.ID); err != nil {
		if quarantined != "" {
			_ = os.Rename(quarantined, idRoot)
		}
		return err
	}
	if quarantined != "" {
		if err = removeManagedTree(managedRoot, quarantined); err != nil {
			return fmt.Errorf("remove managed extension package: %w", err)
		}
	}
	s.refreshProviderExtensions("")
	return nil
}

func (s *Service) restoreInstalledExtensions(ctx context.Context) {
	store, ok := s.store.(extensionInstallStore)
	if !ok || s.extensionSupervisor == nil {
		return
	}
	cleanupManagedExtensionStaging(store)
	installs, err := store.ListExtensionInstalls(ctx)
	if err != nil {
		return
	}
	for index, install := range installs {
		install, prepareErr := ensureManagedExtensionInstall(ctx, store, install)
		if prepareErr != nil {
			message := prepareErr.Error()
			_, _ = store.SetExtensionInstallState(ctx, install.ID, false, domain.ExtensionStateError, message)
			continue
		}
		installs[index] = install
		if _, err = s.extensionSupervisor.Discover(install.RootPath); err != nil {
			_, _ = store.SetExtensionInstallState(ctx, install.ID, false, domain.ExtensionStateError, err.Error())
			continue
		}
		if _, err = s.extensionSupervisor.Trust(install.ID, install.Integrity); err != nil {
			_, _ = store.SetExtensionInstallState(ctx, install.ID, false, domain.ExtensionStateError, err.Error())
			continue
		}
		if !install.Enabled {
			continue
		}
		status, enableErr := s.extensionSupervisor.Enable(ctx, install.ID)
		if enableErr != nil {
			_, _ = store.SetExtensionInstallState(ctx, install.ID, false, domain.ExtensionStateError, enableErr.Error())
			continue
		}
		_, _ = store.SetExtensionInstallState(ctx, install.ID, true, status.State, status.Error)
	}
	cleanupManagedExtensionGenerations(store, installs)
}

func extensionInstallPreview(loaded LoadedExtension) domain.ExtensionInstallPreview {
	return domain.ExtensionInstallPreview{
		Path: loaded.Root, ManifestPath: loaded.ManifestPath, Integrity: loaded.Integrity,
		Summary: domain.ExtensionSummary(loaded.Manifest),
	}
}

func ensureManagedExtensionInstall(ctx context.Context, store extensionInstallStore, install domain.ExtensionInstall) (domain.ExtensionInstall, error) {
	if install.InstallMode != domain.ExtensionInstallModeManaged {
		source, err := LoadExtensionManifest(install.RootPath)
		if err != nil || source.Integrity != install.Integrity || source.Manifest.ID != install.ID {
			return install, errors.New("legacy linked extension cannot be verified; select its source folder and install it again")
		}
		managed, _, err := installManagedExtensionPackage(store, source)
		if err != nil {
			return install, fmt.Errorf("migrate extension into Aivo managed storage: %w", err)
		}
		install.Manifest = managed.Manifest
		install.Summary = domain.ExtensionSummary(managed.Manifest)
		install.InstallMode = domain.ExtensionInstallModeManaged
		install.RootPath = managed.Root
		install.ManifestPath = managed.ManifestPath
		install.Integrity = managed.Integrity
		install, err = store.SaveExtensionInstall(ctx, install)
		if err != nil {
			return install, err
		}
	}
	managedRoot, err := canonicalManagedExtensionRoot(store)
	if err != nil {
		return install, err
	}
	if _, pathErr := validateManagedInstallPath(managedRoot, install); pathErr != nil {
		install, err = relocateLegacyManagedExtensionInstall(ctx, store, managedRoot, install)
		if err != nil {
			return install, err
		}
	}
	loaded, err := LoadExtensionManifest(install.RootPath)
	if err != nil || loaded.Integrity != install.Integrity || loaded.Manifest.ID != install.ID {
		return install, errors.New("Aivo managed extension package is missing or was modified; install it again")
	}
	return install, nil
}

func relocateLegacyManagedExtensionInstall(ctx context.Context, store extensionInstallStore, managedRoot string, install domain.ExtensionInstall) (domain.ExtensionInstall, error) {
	legacyRoot, err := store.LegacyManagedExtensionRoot()
	if err != nil {
		return install, err
	}
	legacyRoot, err = filepath.Abs(legacyRoot)
	if err != nil {
		return install, err
	}
	legacyRoot, err = filepath.EvalSymlinks(legacyRoot)
	if err != nil {
		return install, errors.New("managed extension is outside the current Aivo application-data directory; install it again")
	}
	legacyRoot = filepath.Clean(legacyRoot)
	if legacyRoot == managedRoot {
		return install, errors.New("managed extension path does not match its installation record")
	}
	legacyIDRoot, err := validateManagedInstallPath(legacyRoot, install)
	if err != nil {
		return install, errors.New("managed extension is outside the current or former Aivo-owned directory; install it again")
	}
	source, err := LoadExtensionManifest(install.RootPath)
	if err != nil || source.Integrity != install.Integrity || source.Manifest.ID != install.ID {
		return install, errors.New("former Aivo managed extension package is missing or was modified; install it again")
	}
	managed, _, err := installManagedExtensionPackage(store, source)
	if err != nil {
		return install, fmt.Errorf("move extension into platform application data: %w", err)
	}
	install.Manifest = managed.Manifest
	install.Summary = domain.ExtensionSummary(managed.Manifest)
	install.RootPath = managed.Root
	install.ManifestPath = managed.ManifestPath
	install.Integrity = managed.Integrity
	install, err = store.SaveExtensionInstall(ctx, install)
	if err != nil {
		return install, err
	}
	if cleanupErr := removeManagedTree(legacyRoot, legacyIDRoot); cleanupErr == nil {
		_ = os.Remove(legacyRoot)
	}
	return install, nil
}

func removeManagedGeneration(store extensionInstallStore, loaded LoadedExtension) error {
	managedRoot, err := canonicalManagedExtensionRoot(store)
	if err != nil {
		return err
	}
	target := filepath.Clean(loaded.Root)
	if filepath.Dir(target) != filepath.Join(managedRoot, loaded.Manifest.ID) || filepath.Base(target) != loaded.Integrity {
		return errors.New("managed extension generation path is invalid")
	}
	return removeManagedTree(managedRoot, target)
}

func cleanupManagedExtensionGenerations(store extensionInstallStore, installs []domain.ExtensionInstall) {
	managedRoot, err := canonicalManagedExtensionRoot(store)
	if err != nil {
		return
	}
	for _, install := range installs {
		if install.InstallMode != domain.ExtensionInstallModeManaged {
			continue
		}
		idRoot, pathErr := validateManagedInstallPath(managedRoot, install)
		if pathErr != nil {
			continue
		}
		entries, readErr := os.ReadDir(idRoot)
		if readErr != nil {
			continue
		}
		for _, entry := range entries {
			if !entry.IsDir() || entry.Name() == install.Integrity || strings.HasPrefix(entry.Name(), ".") {
				continue
			}
			_ = removeManagedTree(managedRoot, filepath.Join(idRoot, entry.Name()))
		}
	}
}
