package app

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"aivo/core/domain"
)

func installManagedExtensionPackage(store extensionInstallStore, source LoadedExtension) (LoadedExtension, bool, error) {
	managedRoot, err := canonicalManagedExtensionRoot(store)
	if err != nil {
		return LoadedExtension{}, false, err
	}
	if pathWithin(managedRoot, source.Root) || pathWithin(source.Root, managedRoot) {
		return LoadedExtension{}, false, errors.New("extension source must be outside Aivo managed extension storage")
	}

	staging, err := os.MkdirTemp(managedRoot, ".install-")
	if err != nil {
		return LoadedExtension{}, false, fmt.Errorf("prepare managed extension staging: %w", err)
	}
	defer removeManagedTree(managedRoot, staging)
	stagedRoot := filepath.Join(staging, "package")
	if err := copyExtensionPackage(source.Root, stagedRoot); err != nil {
		return LoadedExtension{}, false, fmt.Errorf("copy extension package: %w", err)
	}
	stagedSource, err := LoadExtensionManifest(stagedRoot)
	if err != nil {
		return LoadedExtension{}, false, fmt.Errorf("validate copied extension package: %w", err)
	}
	if stagedSource.Manifest.ID != source.Manifest.ID || stagedSource.Integrity != source.Integrity {
		return LoadedExtension{}, false, errors.New("extension package changed while it was being copied; review it again")
	}
	if err := hardenManagedExtensionPackage(stagedRoot); err != nil {
		return LoadedExtension{}, false, fmt.Errorf("protect managed extension package: %w", err)
	}
	managed, err := LoadExtensionManifest(stagedRoot)
	if err != nil {
		return LoadedExtension{}, false, fmt.Errorf("validate protected extension package: %w", err)
	}

	idRoot := filepath.Join(managedRoot, managed.Manifest.ID)
	if err := os.MkdirAll(idRoot, 0o700); err != nil {
		return LoadedExtension{}, false, fmt.Errorf("prepare managed extension directory: %w", err)
	}
	if err := os.Chmod(idRoot, 0o700); err != nil {
		return LoadedExtension{}, false, fmt.Errorf("secure managed extension directory: %w", err)
	}
	finalRoot := filepath.Join(idRoot, managed.Integrity)
	if existing, loadErr := LoadExtensionManifest(finalRoot); loadErr == nil {
		if existing.Manifest.ID == managed.Manifest.ID && existing.Integrity == managed.Integrity {
			return existing, false, nil
		}
	}

	var quarantined string
	if _, statErr := os.Lstat(finalRoot); statErr == nil {
		quarantined, err = reserveManagedRemovalPath(managedRoot)
		if err != nil {
			return LoadedExtension{}, false, err
		}
		if err = os.Chmod(finalRoot, 0o700); err != nil {
			return LoadedExtension{}, false, fmt.Errorf("unlock damaged managed extension root: %w", err)
		}
		if err = os.Rename(finalRoot, quarantined); err != nil {
			_ = os.Chmod(finalRoot, 0o500)
			return LoadedExtension{}, false, fmt.Errorf("quarantine damaged managed extension: %w", err)
		}
	}
	if err = os.Chmod(stagedRoot, 0o700); err != nil {
		return LoadedExtension{}, false, fmt.Errorf("prepare protected extension for publish: %w", err)
	}
	if err = os.Rename(stagedRoot, finalRoot); err != nil {
		if quarantined != "" {
			_ = os.Rename(quarantined, finalRoot)
			_ = os.Chmod(finalRoot, 0o500)
		}
		return LoadedExtension{}, false, fmt.Errorf("publish managed extension package: %w", err)
	}
	if err = os.Chmod(finalRoot, 0o500); err != nil {
		_ = removeManagedTree(managedRoot, finalRoot)
		return LoadedExtension{}, false, fmt.Errorf("protect published extension root: %w", err)
	}
	if quarantined != "" {
		_ = removeManagedTree(managedRoot, quarantined)
	}
	published, err := LoadExtensionManifest(finalRoot)
	if err != nil || published.Integrity != managed.Integrity {
		_ = removeManagedTree(managedRoot, finalRoot)
		if err == nil {
			err = errors.New("published extension integrity does not match staged package")
		}
		return LoadedExtension{}, false, err
	}
	return published, true, nil
}

func canonicalManagedExtensionRoot(store extensionInstallStore) (string, error) {
	root, err := store.ManagedExtensionRoot()
	if err != nil {
		return "", err
	}
	canonical, err := filepath.EvalSymlinks(root)
	if err != nil {
		return "", err
	}
	return filepath.Clean(canonical), nil
}

func copyExtensionPackage(sourceRoot string, targetRoot string) error {
	return filepath.WalkDir(sourceRoot, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(sourceRoot, path)
		if err != nil {
			return err
		}
		target := filepath.Join(targetRoot, relative)
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("extension package cannot contain symbolic link %s", filepath.ToSlash(relative))
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if info.IsDir() {
			return os.Mkdir(target, 0o700)
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("extension package contains unsupported file %s", filepath.ToSlash(relative))
		}
		source, err := os.Open(path)
		if err != nil {
			return err
		}
		targetFile, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, info.Mode().Perm())
		if err != nil {
			_ = source.Close()
			return err
		}
		_, copyErr := io.Copy(targetFile, source)
		syncErr := targetFile.Sync()
		closeTargetErr := targetFile.Close()
		closeSourceErr := source.Close()
		for _, candidate := range []error{copyErr, syncErr, closeTargetErr, closeSourceErr} {
			if candidate != nil {
				return candidate
			}
		}
		return nil
	})
}

func hardenManagedExtensionPackage(root string) error {
	var directories []string
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return errors.New("managed extension package cannot contain symbolic links")
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if info.IsDir() {
			directories = append(directories, path)
			return nil
		}
		mode := os.FileMode(0o400)
		if info.Mode().Perm()&0o111 != 0 {
			mode = 0o500
		}
		return os.Chmod(path, mode)
	})
	if err != nil {
		return err
	}
	for index := len(directories) - 1; index >= 0; index-- {
		if err := os.Chmod(directories[index], 0o500); err != nil {
			return err
		}
	}
	return nil
}

func cleanupManagedExtensionStaging(store extensionInstallStore) {
	root, err := canonicalManagedExtensionRoot(store)
	if err != nil {
		return
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		return
	}
	for _, entry := range entries {
		if !entry.IsDir() || (!strings.HasPrefix(entry.Name(), ".install-") && !strings.HasPrefix(entry.Name(), ".remove-")) {
			continue
		}
		_ = removeManagedTree(root, filepath.Join(root, entry.Name()))
	}
}

func reserveManagedRemovalPath(root string) (string, error) {
	path, err := os.MkdirTemp(root, ".remove-")
	if err != nil {
		return "", err
	}
	if err := os.Remove(path); err != nil {
		return "", err
	}
	return path, nil
}

func removeManagedTree(managedRoot string, target string) error {
	managedRoot = filepath.Clean(managedRoot)
	target = filepath.Clean(target)
	if target == managedRoot || !pathWithin(managedRoot, target) {
		return errors.New("refusing to remove a path outside managed extension storage")
	}
	_ = filepath.WalkDir(target, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		if entry.IsDir() {
			_ = os.Chmod(path, 0o700)
		} else {
			_ = os.Chmod(path, 0o600)
		}
		return nil
	})
	if err := os.RemoveAll(target); err != nil {
		return err
	}
	return nil
}

func validateManagedInstallPath(managedRoot string, install domain.ExtensionInstall) (string, error) {
	if install.InstallMode != domain.ExtensionInstallModeManaged || !extensionIDPattern.MatchString(install.ID) {
		return "", errors.New("extension is not installed in Aivo managed storage")
	}
	idRoot := filepath.Join(managedRoot, install.ID)
	installedRoot := filepath.Clean(install.RootPath)
	if filepath.Dir(installedRoot) != idRoot || filepath.Base(installedRoot) != install.Integrity || !pathWithin(managedRoot, installedRoot) {
		return "", errors.New("managed extension path does not match its installation record")
	}
	return idRoot, nil
}
