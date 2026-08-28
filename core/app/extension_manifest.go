package app

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"hash"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"unicode/utf8"

	"aivo/core/domain"
)

var extensionIDPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]*$`)
var extensionToolNamePattern = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

const (
	extensionPackageMaxFiles = 4096
	extensionPackageMaxBytes = 64 << 20
)

type LoadedExtension struct {
	Root         string
	ManifestPath string
	Manifest     domain.ExtensionManifest
	ToolSchemas  map[string]map[string]any
	Integrity    string
}

func LoadBuiltinExtensionManifest(raw []byte) (LoadedExtension, error) {
	var manifest domain.ExtensionManifest
	if err := json.Unmarshal(raw, &manifest); err != nil {
		return LoadedExtension{}, fmt.Errorf("invalid built-in extension manifest: %w", err)
	}
	if manifest.Runtime.Type != domain.ExtensionRuntimeBuiltin {
		return LoadedExtension{}, errors.New("embedded extension manifest must use the built-in runtime")
	}
	if err := validateExtensionManifest("", manifest); err != nil {
		return LoadedExtension{}, err
	}
	hasher := sha256.New()
	canonicalManifest, _ := json.Marshal(manifest)
	_, _ = hasher.Write(canonicalManifest)
	schemas := make(map[string]map[string]any, len(manifest.Contributes.Tools))
	for _, contribution := range manifest.Contributes.Tools {
		if _, externalSchema := contribution.Schema.(string); externalSchema {
			return LoadedExtension{}, fmt.Errorf("built-in tool %s schema must be embedded inline", contribution.Name)
		}
		schema, schemaRaw, err := loadExtensionToolSchema("", contribution.Schema)
		if err != nil {
			return LoadedExtension{}, fmt.Errorf("tool %s: %w", contribution.Name, err)
		}
		schemas[contribution.Name] = schema
		_, _ = hasher.Write([]byte(contribution.Name))
		var normalizedSchema any
		if json.Unmarshal(schemaRaw, &normalizedSchema) == nil {
			schemaRaw, _ = json.Marshal(normalizedSchema)
		}
		_, _ = hasher.Write(schemaRaw)
	}
	return LoadedExtension{
		ManifestPath: "embedded:" + manifest.ID,
		Manifest:     manifest,
		ToolSchemas:  schemas,
		Integrity:    hex.EncodeToString(hasher.Sum(nil)),
	}, nil
}

func LoadExtensionManifest(path string) (LoadedExtension, error) {
	root := strings.TrimSpace(path)
	if root == "" {
		return LoadedExtension{}, errors.New("extension path is required")
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return LoadedExtension{}, err
	}
	canonical, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return LoadedExtension{}, err
	}
	abs = filepath.Clean(canonical)
	info, err := os.Stat(abs)
	if err != nil {
		return LoadedExtension{}, err
	}
	manifestPath := abs
	if info.IsDir() {
		root = abs
		manifestPath = ""
		for _, candidate := range []string{filepath.Join(root, ".aivo-extension", "extension.json"), filepath.Join(root, "aivo.extension.json")} {
			if item, statErr := os.Stat(candidate); statErr == nil && item.Mode().IsRegular() {
				manifestPath = candidate
				break
			}
		}
		if manifestPath == "" {
			return LoadedExtension{}, errors.New("extension manifest not found")
		}
	} else {
		root = filepath.Dir(abs)
	}
	if !pathWithin(root, manifestPath) {
		return LoadedExtension{}, errors.New("extension manifest escapes package root")
	}
	raw, err := os.ReadFile(manifestPath)
	if err != nil {
		return LoadedExtension{}, err
	}
	var manifest domain.ExtensionManifest
	if err := json.Unmarshal(raw, &manifest); err != nil {
		return LoadedExtension{}, fmt.Errorf("invalid extension manifest: %w", err)
	}
	if err := validateExtensionManifest(root, manifest); err != nil {
		return LoadedExtension{}, err
	}
	schemas := map[string]map[string]any{}
	hasher := sha256.New()
	canonicalManifest, _ := json.Marshal(manifest)
	_, _ = hasher.Write(canonicalManifest)
	for _, contribution := range manifest.Contributes.Tools {
		schema, schemaRaw, err := loadExtensionToolSchema(root, contribution.Schema)
		if err != nil {
			return LoadedExtension{}, fmt.Errorf("tool %s: %w", contribution.Name, err)
		}
		schemas[contribution.Name] = schema
		_, _ = hasher.Write([]byte(contribution.Name))
		var normalizedSchema any
		if json.Unmarshal(schemaRaw, &normalizedSchema) == nil {
			schemaRaw, _ = json.Marshal(normalizedSchema)
		}
		_, _ = hasher.Write(schemaRaw)
	}
	for _, contribution := range manifest.Contributes.Contexts {
		asset, err := resolveExtensionPackagePath(root, contribution.Path)
		if err != nil {
			return LoadedExtension{}, err
		}
		rawAsset, err := os.ReadFile(asset)
		if err != nil {
			return LoadedExtension{}, err
		}
		_, _ = hasher.Write([]byte(contribution.ID))
		_, _ = hasher.Write(rawAsset)
	}
	if (manifest.Runtime.Type == domain.ExtensionRuntimeProcess || manifest.Runtime.Type == domain.ExtensionRuntimeService) && strings.ContainsAny(manifest.Runtime.Command, `/\`) {
		commandPath, err := resolveExtensionPackagePath(root, manifest.Runtime.Command)
		if err != nil {
			return LoadedExtension{}, err
		}
		commandRaw, err := os.ReadFile(commandPath)
		if err != nil {
			return LoadedExtension{}, err
		}
		_, _ = hasher.Write(commandRaw)
	}
	if err := hashExtensionPackage(root, hasher); err != nil {
		return LoadedExtension{}, err
	}
	return LoadedExtension{Root: root, ManifestPath: manifestPath, Manifest: manifest, ToolSchemas: schemas, Integrity: hex.EncodeToString(hasher.Sum(nil))}, nil
}

func hashExtensionPackage(root string, hasher hash.Hash) error {
	files := 0
	var totalBytes int64
	return filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if relative == "." {
			return nil
		}
		relative = filepath.ToSlash(relative)
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("extension package cannot contain symbolic link %s", relative)
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if info.IsDir() {
			_, _ = hasher.Write([]byte("directory\x00" + relative + "\x00"))
			return nil
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("extension package contains unsupported file %s", relative)
		}
		files++
		if files > extensionPackageMaxFiles {
			return fmt.Errorf("extension package exceeds %d files", extensionPackageMaxFiles)
		}
		_, _ = hasher.Write([]byte(fmt.Sprintf("file\x00%s\x00%o\x00", relative, info.Mode().Perm())))
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		remaining := int64(extensionPackageMaxBytes) - totalBytes
		written, copyErr := io.Copy(hasher, io.LimitReader(file, remaining+1))
		closeErr := file.Close()
		totalBytes += written
		if copyErr != nil {
			return copyErr
		}
		if closeErr != nil {
			return closeErr
		}
		if totalBytes > extensionPackageMaxBytes {
			return fmt.Errorf("extension package exceeds %d bytes", extensionPackageMaxBytes)
		}
		_, _ = hasher.Write([]byte("\x00"))
		return nil
	})
}

func validateExtensionManifest(root string, manifest domain.ExtensionManifest) error {
	apiVersion := strings.TrimSpace(manifest.APIVersion)
	if manifest.SchemaVersion != 2 || apiVersion != "2" {
		return errors.New("extension manifest and api versions must use the supported 2/2 pair")
	}
	if !extensionIDPattern.MatchString(manifest.ID) || strings.TrimSpace(manifest.Name) == "" || strings.TrimSpace(manifest.Version) == "" {
		return errors.New("extension id, name, and version are required and id must be namespaced")
	}
	switch manifest.Runtime.Type {
	case domain.ExtensionRuntimeBuiltin:
	case domain.ExtensionRuntimeProcess:
		if strings.TrimSpace(manifest.Runtime.Command) == "" || firstNonEmpty(manifest.Runtime.Transport, "stdio") != "stdio" {
			return errors.New("process extensions require a stdio command")
		}
		if strings.ContainsAny(manifest.Runtime.Command, `/\`) {
			if _, err := resolveExtensionPackagePath(root, manifest.Runtime.Command); err != nil {
				return err
			}
		}
	case domain.ExtensionRuntimeService:
		if strings.TrimSpace(manifest.Runtime.Command) == "" {
			return errors.New("local service extensions require a supervised command")
		}
		switch firstNonEmpty(manifest.Runtime.Transport, "http") {
		case "http":
			if strings.TrimSpace(manifest.Runtime.URL) == "" {
				return errors.New("fixed local service extensions require a loopback URL")
			}
			parsed, err := url.Parse(manifest.Runtime.URL)
			if err != nil || parsed.Scheme != "http" || !extensionLoopbackHost(parsed.Hostname()) {
				return errors.New("local service extension URL must use loopback HTTP")
			}
		case "dynamic-http":
			if strings.TrimSpace(manifest.Runtime.URL) != "" {
				return errors.New("dynamic local service extensions must omit runtime.url")
			}
		default:
			return errors.New("local service extension transport must be http or dynamic-http")
		}
	case domain.ExtensionRuntimeExternal:
		if !strings.HasPrefix(strings.ToLower(strings.TrimSpace(manifest.Runtime.URL)), "https://") {
			return errors.New("external extension services require HTTPS")
		}
		if len(manifest.Requirements.Credentials) == 0 {
			return errors.New("external extension services require an explicitly bound credential slot")
		}
	case domain.ExtensionRuntimeStatic:
		if len(manifest.Contributes.Tools) > 0 {
			return errors.New("static extensions cannot contribute executable tools")
		}
	default:
		return fmt.Errorf("unsupported extension runtime %q", manifest.Runtime.Type)
	}
	seenPermissions := map[string]bool{}
	for _, permission := range manifest.Permissions {
		if permission != "runtime.messaging" || seenPermissions[permission] {
			return fmt.Errorf("invalid or duplicate extension permission %q", permission)
		}
		if manifest.Runtime.Type != domain.ExtensionRuntimeService && manifest.Runtime.Type != domain.ExtensionRuntimeExternal {
			return errors.New("runtime.messaging requires a service or external runtime")
		}
		seenPermissions[permission] = true
	}
	seenTools := map[string]bool{}
	for _, tool := range manifest.Contributes.Tools {
		if !extensionToolNamePattern.MatchString(tool.Name) || len(tool.Name) > toolNameMaxLength || (!strings.Contains(tool.Name, "_") && !strings.Contains(tool.Name, "-")) {
			return fmt.Errorf("extension tool %q must use a Provider-safe `_`/`-` namespace and be at most %d bytes", tool.Name, toolNameMaxLength)
		}
		if isReservedCoreToolName(tool.Name) || isBridgeToolName(tool.Name) {
			return fmt.Errorf("extension tool %q uses a reserved name", tool.Name)
		}
		if seenTools[tool.Name] {
			return fmt.Errorf("duplicate extension tool %q", tool.Name)
		}
		seenTools[tool.Name] = true
		switch firstNonEmpty(tool.Activation, "auto") {
		case "auto", "manual", "default":
		default:
			return fmt.Errorf("tool %s has invalid activation", tool.Name)
		}
	}
	seenToolGroups := map[string]bool{}
	groupedTools := map[string]bool{}
	for _, group := range manifest.Contributes.ToolGroups {
		groupID := strings.TrimSpace(group.ID)
		groupName := strings.TrimSpace(group.Name)
		groupDescription := strings.TrimSpace(group.Description)
		if !extensionIDPattern.MatchString(groupID) || seenToolGroups[groupID] {
			return fmt.Errorf("invalid or duplicate extension tool group %q", group.ID)
		}
		if groupName == "" || utf8.RuneCountInString(groupName) > 100 || utf8.RuneCountInString(groupDescription) > 500 {
			return fmt.Errorf("extension tool group %q requires a bounded name and description", group.ID)
		}
		if len(group.Tools) == 0 {
			return fmt.Errorf("extension tool group %q must contain at least one tool", group.ID)
		}
		seenToolGroups[groupID] = true
		seenMembers := map[string]bool{}
		for _, toolName := range group.Tools {
			toolName = strings.TrimSpace(toolName)
			if !seenTools[toolName] {
				return fmt.Errorf("extension tool group %q references undeclared tool %q", group.ID, toolName)
			}
			if seenMembers[toolName] || groupedTools[toolName] {
				return fmt.Errorf("extension tool %q belongs to more than one tool group", toolName)
			}
			seenMembers[toolName] = true
			groupedTools[toolName] = true
		}
	}
	allowedSurfaces := map[string]bool{"page": true, "dialog": true, "tool-detail": true, "settings": true, "notification": true}
	seenViews := map[string]bool{}
	for _, view := range manifest.Contributes.Views {
		if view.ID == "" || seenViews[view.ID] || view.Type != "web" || !strings.HasPrefix(view.Route, "/") || strings.Contains(view.Route, "..") {
			return fmt.Errorf("invalid extension view %q", view.ID)
		}
		seenViews[view.ID] = true
		for _, surface := range view.Surfaces {
			if !allowedSurfaces[surface] {
				return fmt.Errorf("view %s uses unsupported surface %s", view.ID, surface)
			}
		}
		seenActions := map[string]bool{}
		for _, action := range view.Actions {
			if !extensionIDPattern.MatchString(action) || seenActions[action] {
				return fmt.Errorf("view %s has an invalid or duplicate action %q", view.ID, action)
			}
			seenActions[action] = true
		}
		for _, toolName := range view.Tools {
			if !seenTools[toolName] {
				return fmt.Errorf("view %s references undeclared tool %s", view.ID, toolName)
			}
		}
	}
	seenContexts := map[string]bool{}
	for _, contribution := range manifest.Contributes.Contexts {
		if !extensionIDPattern.MatchString(contribution.ID) || seenContexts[contribution.ID] || strings.TrimSpace(contribution.Kind) == "" {
			return fmt.Errorf("invalid extension context %q", contribution.ID)
		}
		seenContexts[contribution.ID] = true
	}
	seenPolicies := map[string]bool{}
	for _, policy := range manifest.Contributes.Policies {
		if manifest.Runtime.Type == domain.ExtensionRuntimeStatic {
			return errors.New("static extensions cannot contribute executable policies")
		}
		if !extensionIDPattern.MatchString(policy) || !strings.Contains(policy, ".") || seenPolicies[policy] {
			return fmt.Errorf("invalid or duplicate extension policy %q", policy)
		}
		seenPolicies[policy] = true
	}
	if environment := manifest.Contributes.Environment; environment != nil {
		if manifest.Runtime.Type == domain.ExtensionRuntimeStatic || !extensionIDPattern.MatchString(environment.ID) || !strings.Contains(environment.ID, ".") {
			return fmt.Errorf("invalid executable extension environment %q", environment.ID)
		}
	}
	if len(manifest.Requirements.Platforms) > 0 {
		platformAllowed := false
		for _, platform := range manifest.Requirements.Platforms {
			if platform == runtime.GOOS {
				platformAllowed = true
			}
		}
		if !platformAllowed {
			return fmt.Errorf("extension does not support platform %s", runtime.GOOS)
		}
	}
	credentials := append([]string(nil), manifest.Requirements.Credentials...)
	sort.Strings(credentials)
	for index, name := range credentials {
		if strings.TrimSpace(name) == "" || (index > 0 && name == credentials[index-1]) {
			return errors.New("credential slot names must be unique and non-empty")
		}
	}
	return nil
}

func extensionToolSelectionGroups(manifest domain.ExtensionManifest) map[string]domain.ToolSelectionGroup {
	groups := make(map[string]domain.ToolSelectionGroup)
	for _, contribution := range manifest.Contributes.ToolGroups {
		group := domain.ToolSelectionGroup{
			ID:          generatedToolName("extension", manifest.ID, contribution.ID),
			Name:        strings.TrimSpace(contribution.Name),
			Description: strings.TrimSpace(contribution.Description),
		}
		for _, toolName := range contribution.Tools {
			groups[strings.TrimSpace(toolName)] = group
		}
	}
	return groups
}

func extensionLoopbackHost(host string) bool {
	return host == "127.0.0.1" || host == "localhost" || host == "::1"
}

func loadExtensionToolSchema(root string, value any) (map[string]any, []byte, error) {
	switch schema := value.(type) {
	case string:
		path, err := resolveExtensionPackagePath(root, schema)
		if err != nil {
			return nil, nil, err
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return nil, nil, err
		}
		var decoded map[string]any
		if err := json.Unmarshal(raw, &decoded); err != nil {
			return nil, nil, err
		}
		if decoded["type"] != "object" {
			return nil, nil, errors.New("tool schema must describe an object")
		}
		return decoded, raw, nil
	case map[string]any:
		raw, _ := json.Marshal(schema)
		if schema["type"] != "object" {
			return nil, nil, errors.New("tool schema must describe an object")
		}
		return domain.CloneRawMap(schema), raw, nil
	default:
		return nil, nil, errors.New("tool schema must be an object or package-relative JSON path")
	}
}

func resolveExtensionPackagePath(root, value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" || filepath.IsAbs(value) {
		return "", errors.New("extension asset path must be package-relative")
	}
	path := filepath.Join(root, filepath.Clean(value))
	if !pathWithin(root, path) {
		return "", errors.New("extension asset escapes package root")
	}
	canonical, err := filepath.EvalSymlinks(path)
	if err != nil {
		return "", err
	}
	if !pathWithin(root, canonical) {
		return "", errors.New("extension asset escapes package root through symbolic link")
	}
	return canonical, nil
}
