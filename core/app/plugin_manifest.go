package app

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"aivo/core/domain"
)

func LoadPluginManifest(path string) (string, string, domain.PluginManifest, error) {
	root := strings.TrimSpace(path)
	if root == "" {
		return "", "", domain.PluginManifest{}, errors.New("plugin path is required")
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return "", "", domain.PluginManifest{}, err
	}
	info, err := os.Stat(absRoot)
	if err != nil {
		return "", "", domain.PluginManifest{}, err
	}
	manifestPath := absRoot
	if info.IsDir() {
		candidates := []string{filepath.Join(absRoot, ".aivo-plugin", "plugin.json"), filepath.Join(absRoot, "aivo.plugin.json")}
		manifestPath = ""
		for _, candidate := range candidates {
			if st, err := os.Stat(candidate); err == nil && !st.IsDir() {
				manifestPath = candidate
				break
			}
		}
		if manifestPath == "" {
			return "", "", domain.PluginManifest{}, errors.New("plugin manifest not found")
		}
	} else {
		absRoot = filepath.Dir(absRoot)
	}
	if !pathWithin(absRoot, manifestPath) {
		return "", "", domain.PluginManifest{}, errors.New("plugin manifest escapes plugin root")
	}
	raw, err := os.ReadFile(manifestPath)
	if err != nil {
		return "", "", domain.PluginManifest{}, err
	}
	var manifest domain.PluginManifest
	if err := json.Unmarshal(raw, &manifest); err != nil {
		return "", "", domain.PluginManifest{}, err
	}
	if manifest.ID == "" {
		manifest.ID = firstNonEmptyApp(manifest.Name, filepath.Base(absRoot))
	}
	if manifest.Name == "" {
		manifest.Name = manifest.ID
	}
	if err := validatePluginManifestPaths(absRoot, manifest); err != nil {
		return "", "", domain.PluginManifest{}, err
	}
	return absRoot, manifestPath, manifest, nil
}

func validatePluginManifestPaths(root string, manifest domain.PluginManifest) error {
	if manifest.Entrypoint.Command != "" && strings.Contains(manifest.Entrypoint.Command, string(os.PathSeparator)) && !filepath.IsAbs(manifest.Entrypoint.Command) {
		if !pathWithin(root, filepath.Join(root, manifest.Entrypoint.Command)) {
			return errors.New("entrypoint command escapes plugin root")
		}
	}
	if manifest.Entrypoint.CWD != "" && !pathWithin(root, filepath.Join(root, manifest.Entrypoint.CWD)) {
		return errors.New("entrypoint cwd escapes plugin root")
	}
	return nil
}

func pathWithin(root string, target string) bool {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return false
	}
	absTarget, err := filepath.Abs(target)
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(absRoot, absTarget)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator)) && !filepath.IsAbs(rel)
}

func firstNonEmptyApp(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func pluginContainsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
