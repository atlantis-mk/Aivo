package app

import (
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"unicode/utf8"

	"gopkg.in/yaml.v3"

	"aivo/core/domain"
)

const (
	promptSchema          = "aivo.prompt/v1"
	promptFileMaxBytes    = 32 * 1024
	promptCatalogMaxFiles = 256
	promptCatalogMaxBytes = 2 * 1024 * 1024
)

//go:embed prompts/builtin/*/*.md
var builtinPromptFS embed.FS

type promptContract struct {
	Category          string
	Required          bool
	Disableable       bool
	Variables         []string
	RequiredVariables []string
	MaxLength         int
}

type promptMarkdownFrontmatter struct {
	Schema   string `yaml:"schema"`
	ID       string `yaml:"id"`
	Category string `yaml:"category"`
	Title    string `yaml:"title"`
	Enabled  *bool  `yaml:"enabled"`
}

type promptEntry struct {
	ID       string
	Category string
	Title    string
	Body     string
	Enabled  bool
	Revision string
	Origin   string
}

type promptManifest struct {
	Version int               `json:"version"`
	Active  map[string]string `json:"active"`
}

type PromptSnapshot struct {
	entries map[string]promptEntry
}

func (s PromptSnapshot) Render(id string, values map[string]string) (string, error) {
	entry, ok := s.entries[id]
	if !ok || !entry.Enabled {
		return "", fmt.Errorf("prompt %s is unavailable", id)
	}
	return renderPromptTemplate(entry.Body, values)
}

func (s PromptSnapshot) Body(id string) string {
	entry, ok := s.entries[id]
	if !ok || !entry.Enabled {
		return ""
	}
	return entry.Body
}

type PromptRegistry struct {
	mu          sync.RWMutex
	root        string
	contracts   map[string]promptContract
	builtins    map[string]promptEntry
	working     map[string]promptEntry
	active      map[string]promptEntry
	diagnostics map[string][]domain.PromptDiagnostic
}

var promptIDPattern = regexp.MustCompile(`^[a-z][a-z0-9_.-]{0,127}$`)
var promptVariablePattern = regexp.MustCompile(`\{\{([a-z][a-z0-9_]*)\}\}`)
var anyPromptVariablePattern = regexp.MustCompile(`\{\{([^{}]+)\}\}`)

var retiredPromptIDs = map[string]bool{
	"protocol.subagents": true,
}

func defaultPromptContracts() map[string]promptContract {
	contracts := map[string]promptContract{}
	for _, id := range []string{"assistant", "summary", "title", "scheduler_worker"} {
		contracts["agent."+id] = promptContract{Category: domain.PromptCategoryAgent, Required: true, MaxLength: promptFileMaxBytes}
	}
	add := func(id, category string, required, disableable bool, vars, requiredVars []string) {
		contracts[id] = promptContract{Category: category, Required: required, Disableable: disableable, Variables: vars, RequiredVariables: requiredVars, MaxLength: promptFileMaxBytes}
	}
	add("protocol.tool", domain.PromptCategoryProtocol, true, false, nil, nil)
	add("dynamic.context_policy", domain.PromptCategoryDynamicContext, true, false, nil, nil)
	add("dynamic.live_terminals", domain.PromptCategoryDynamicContext, true, false, []string{"terminals"}, []string{"terminals"})
	add("dynamic.file_snapshots", domain.PromptCategoryDynamicContext, true, false, []string{"snapshots"}, []string{"snapshots"})
	add("dynamic.older_recap", domain.PromptCategoryDynamicContext, true, false, []string{"messages"}, []string{"messages"})
	add("dynamic.skill_content_footer", domain.PromptCategoryDynamicContext, true, false, []string{"directory"}, []string{"directory"})
	add("auxiliary.title.system", domain.PromptCategoryAuxiliary, true, false, nil, nil)
	add("auxiliary.title.user", domain.PromptCategoryAuxiliary, true, false, []string{"content"}, []string{"content"})
	add("auxiliary.summary.system", domain.PromptCategoryAuxiliary, true, false, nil, nil)
	add("auxiliary.summary.user", domain.PromptCategoryAuxiliary, true, false, []string{"content"}, []string{"content"})
	add("auxiliary.project_description.system", domain.PromptCategoryAuxiliary, false, true, nil, nil)
	add("auxiliary.project_description.user", domain.PromptCategoryAuxiliary, false, true, []string{"project_path", "content"}, []string{"project_path", "content"})
	add("auxiliary.mcp_description.system", domain.PromptCategoryAuxiliary, false, true, nil, nil)
	add("auxiliary.mcp_description.user", domain.PromptCategoryAuxiliary, false, true, []string{"catalog"}, []string{"catalog"})
	add("auxiliary.host_resource_groups.system", domain.PromptCategoryAuxiliary, true, false, nil, nil)
	add("auxiliary.host_resource_groups.user", domain.PromptCategoryAuxiliary, true, false, []string{"intent", "candidates"}, []string{"intent", "candidates"})
	add("auxiliary.host_resources.system", domain.PromptCategoryAuxiliary, true, false, nil, nil)
	add("task.init", domain.PromptCategoryTask, false, true, nil, nil)
	add("task.review", domain.PromptCategoryTask, false, true, []string{"target"}, []string{"target"})
	add("task.terminal_resume", domain.PromptCategoryTask, true, false, []string{"process_ref", "cursor", "mode"}, []string{"process_ref", "cursor", "mode"})
	for _, id := range []string{"organize_files", "analyze_code", "run_task", "search_research"} {
		add("quick."+id, domain.PromptCategoryQuickPrompt, false, true, nil, nil)
	}
	return contracts
}

func defaultPromptRoot() (string, error) {
	if configured := strings.TrimSpace(os.Getenv("AIVO_PROMPTS_DIR")); configured != "" {
		return filepath.Abs(configured)
	}
	root, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, "Aivo", "Default", "Prompts"), nil
}

func NewPromptRegistry(root string) (*PromptRegistry, error) {
	if strings.TrimSpace(root) == "" {
		var err error
		root, err = defaultPromptRoot()
		if err != nil {
			return nil, err
		}
	}
	root, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	registry, err := NewBuiltinPromptRegistry()
	if err != nil {
		return nil, err
	}
	registry.root = root
	if err := registry.ensureRoot(); err != nil {
		return nil, err
	}
	if err := registry.Reload(); err != nil {
		return nil, err
	}
	return registry, nil
}

func NewBuiltinPromptRegistry() (*PromptRegistry, error) {
	registry := &PromptRegistry{contracts: defaultPromptContracts(), builtins: map[string]promptEntry{}, working: map[string]promptEntry{}, active: map[string]promptEntry{}, diagnostics: map[string][]domain.PromptDiagnostic{}}
	if err := registry.loadBuiltins(); err != nil {
		return nil, err
	}
	for id, entry := range registry.builtins {
		registry.active[id] = entry
	}
	return registry, nil
}

var builtinPromptsOnce sync.Once
var builtinPrompts map[string]promptEntry
var builtinPromptsErr error

func builtinPromptBody(id string) string {
	builtinPromptsOnce.Do(func() {
		registry, err := NewBuiltinPromptRegistry()
		builtinPromptsErr = err
		if registry != nil {
			builtinPrompts = registry.builtins
		}
	})
	if builtinPromptsErr != nil {
		panic(builtinPromptsErr)
	}
	return builtinPrompts[id].Body
}

func (r *PromptRegistry) ensureRoot() error {
	for _, path := range []string{r.root, filepath.Join(r.root, "overrides"), filepath.Join(r.root, ".state", "validated")} {
		if err := os.MkdirAll(path, 0o700); err != nil {
			return err
		}
		_ = os.Chmod(path, 0o700)
	}
	return nil
}

func (r *PromptRegistry) loadBuiltins() error {
	return fs.WalkDir(builtinPromptFS, "prompts/builtin", func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() || filepath.Ext(path) != ".md" {
			return nil
		}
		raw, err := builtinPromptFS.ReadFile(path)
		if err != nil {
			return err
		}
		entry, diagnostics := r.parse(raw, "builtin")
		if len(diagnostics) > 0 {
			return fmt.Errorf("invalid builtin prompt %s: %s", path, diagnostics[0].Message)
		}
		if _, exists := r.builtins[entry.ID]; exists {
			return fmt.Errorf("duplicate builtin prompt %s", entry.ID)
		}
		r.builtins[entry.ID] = entry
		return nil
	})
}

func (r *PromptRegistry) Snapshot() PromptSnapshot {
	r.mu.RLock()
	defer r.mu.RUnlock()
	entries := make(map[string]promptEntry, len(r.active))
	for id, entry := range r.active {
		entries[id] = entry
	}
	return PromptSnapshot{entries: entries}
}

func (r *PromptRegistry) Render(id string, values map[string]string) (string, error) {
	return r.Snapshot().Render(id, values)
}

func (s *Service) renderManagedPrompt(id string, values map[string]string) (string, error) {
	if s != nil && s.prompts != nil {
		return s.prompts.Render(id, values)
	}
	return renderPromptTemplate(builtinPromptBody(id), values)
}

func (s *Service) currentPromptSnapshot() PromptSnapshot {
	if s != nil && s.prompts != nil {
		return s.prompts.Snapshot()
	}
	return PromptSnapshot{}
}

func (r *PromptRegistry) Root() string { return r.root }

func (r *PromptRegistry) Reload() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	working, diagnostics, err := r.loadWorkingFiles()
	if err != nil {
		return err
	}
	validated := r.loadValidatedManifest()
	active := make(map[string]promptEntry, len(r.builtins)+len(working))
	for id, builtin := range r.builtins {
		active[id] = builtin
	}
	for id, entry := range working {
		if len(diagnostics[id]) > 0 {
			if previous, ok := validated[id]; ok {
				active[id] = previous
			}
			continue
		}
		active[id] = entry
		if err := r.publishValidated(entry); err != nil {
			diagnostics[id] = append(diagnostics[id], domain.PromptDiagnostic{Code: "publish_failed", Message: err.Error()})
			if previous, ok := validated[id]; ok {
				active[id] = previous
			} else if builtin, ok := r.builtins[id]; ok {
				active[id] = builtin
			} else {
				delete(active, id)
			}
		}
	}
	for id, contract := range r.contracts {
		entry, ok := active[id]
		if contract.Required && (!ok || !entry.Enabled) {
			diagnostics[id] = append(diagnostics[id], domain.PromptDiagnostic{Code: "required_unavailable", Message: "required prompt has no active enabled revision"})
			if builtin, exists := r.builtins[id]; exists {
				active[id] = builtin
			}
		}
	}
	r.working, r.active, r.diagnostics = working, active, diagnostics
	return r.writeManifest(active)
}

func (r *PromptRegistry) loadWorkingFiles() (map[string]promptEntry, map[string][]domain.PromptDiagnostic, error) {
	entries := map[string]promptEntry{}
	diagnostics := map[string][]domain.PromptDiagnostic{}
	root := filepath.Join(r.root, "overrides")
	files, total := 0, 0
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			relative, err := filepath.Rel(root, path)
			if err != nil || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
				return errors.New("prompt catalog path escapes the managed root")
			}
			return nil
		}
		if d.Type()&os.ModeSymlink != 0 {
			return errors.New("prompt catalog contains a symbolic link")
		}
		if filepath.Ext(path) != ".md" {
			return nil
		}
		files++
		if files > promptCatalogMaxFiles {
			return errors.New("prompt file limit exceeded")
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return errors.New("prompt catalog contains a non-regular file")
		}
		total += int(info.Size())
		if info.Size() > promptFileMaxBytes {
			diagnostics[filepath.Base(path)] = []domain.PromptDiagnostic{{Code: "file_too_large", Message: "prompt file exceeds 32768 bytes"}}
			return nil
		}
		if total > promptCatalogMaxBytes {
			return errors.New("prompt catalog size limit exceeded")
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		entry, parsedDiagnostics := r.parse(raw, "override")
		if retiredPromptIDs[entry.ID] {
			return nil
		}
		key := entry.ID
		if key == "" {
			key = filepath.Base(path)
		}
		if _, exists := entries[entry.ID]; exists && entry.ID != "" {
			parsedDiagnostics = append(parsedDiagnostics, domain.PromptDiagnostic{Code: "duplicate_id", Message: "prompt id is duplicated"})
		}
		if entry.ID != "" {
			entries[entry.ID] = entry
		}
		if len(parsedDiagnostics) > 0 {
			diagnostics[key] = parsedDiagnostics
		}
		return nil
	})
	return entries, diagnostics, err
}

func (r *PromptRegistry) parse(raw []byte, origin string) (promptEntry, []domain.PromptDiagnostic) {
	if len(raw) > promptFileMaxBytes {
		return promptEntry{}, []domain.PromptDiagnostic{{Code: "file_too_large", Message: "prompt file exceeds 32768 bytes"}}
	}
	if !utf8.Valid(raw) {
		return promptEntry{}, []domain.PromptDiagnostic{{Code: "invalid_utf8", Message: "prompt file must be UTF-8"}}
	}
	text := strings.ReplaceAll(string(raw), "\r\n", "\n")
	if !strings.HasPrefix(text, "---\n") {
		return promptEntry{}, []domain.PromptDiagnostic{{Code: "frontmatter_required", Message: "prompt Markdown must start with YAML frontmatter", Line: 1, Column: 1}}
	}
	end := strings.Index(text[4:], "\n---\n")
	if end < 0 {
		return promptEntry{}, []domain.PromptDiagnostic{{Code: "frontmatter_unclosed", Message: "prompt YAML frontmatter is not closed", Line: 1, Column: 1}}
	}
	end += 4
	frontmatterRaw := text[4:end]
	body := strings.TrimSpace(text[end+5:])
	var frontmatter promptMarkdownFrontmatter
	decoder := yaml.NewDecoder(strings.NewReader(frontmatterRaw))
	decoder.KnownFields(true)
	if err := decoder.Decode(&frontmatter); err != nil {
		return promptEntry{}, []domain.PromptDiagnostic{{Code: "frontmatter_invalid", Message: err.Error()}}
	}
	entry := promptEntry{ID: strings.TrimSpace(frontmatter.ID), Category: strings.TrimSpace(frontmatter.Category), Title: strings.TrimSpace(frontmatter.Title), Body: body, Enabled: true, Origin: origin}
	if frontmatter.Enabled != nil {
		entry.Enabled = *frontmatter.Enabled
	}
	diagnostics := r.validateEntry(entry, frontmatter.Schema)
	entry.Revision = promptRevision(entry)
	return entry, diagnostics
}

func (r *PromptRegistry) validateEntry(entry promptEntry, schema string) []domain.PromptDiagnostic {
	var diagnostics []domain.PromptDiagnostic
	if schema != promptSchema {
		diagnostics = append(diagnostics, domain.PromptDiagnostic{Code: "schema_invalid", Message: "schema must be aivo.prompt/v1"})
	}
	if !promptIDPattern.MatchString(entry.ID) {
		diagnostics = append(diagnostics, domain.PromptDiagnostic{Code: "id_invalid", Message: "id must be a stable lowercase prompt identifier"})
	}
	if entry.Title == "" || len(entry.Title) > 80 {
		diagnostics = append(diagnostics, domain.PromptDiagnostic{Code: "title_invalid", Message: "title is required and must not exceed 80 bytes"})
	}
	if entry.Body == "" {
		diagnostics = append(diagnostics, domain.PromptDiagnostic{Code: "body_required", Message: "prompt body is required"})
	}
	contract, builtin := r.contracts[entry.ID]
	if !builtin {
		if entry.Category != domain.PromptCategoryAgent && entry.Category != domain.PromptCategoryQuickPrompt {
			diagnostics = append(diagnostics, domain.PromptDiagnostic{Code: "custom_category_invalid", Message: "custom prompts must be agent or quick_prompt"})
		}
		contract = promptContract{Category: entry.Category, Required: entry.Category == domain.PromptCategoryAgent, Disableable: entry.Category == domain.PromptCategoryQuickPrompt, MaxLength: promptFileMaxBytes}
	}
	if entry.Category != contract.Category {
		diagnostics = append(diagnostics, domain.PromptDiagnostic{Code: "category_mismatch", Message: "category does not match the registered prompt contract"})
	}
	if contract.Required && !entry.Enabled {
		diagnostics = append(diagnostics, domain.PromptDiagnostic{Code: "required_disabled", Message: "required prompts cannot be disabled"})
	}
	allowed := map[string]bool{}
	for _, name := range contract.Variables {
		allowed[name] = true
	}
	found := map[string]bool{}
	for _, match := range anyPromptVariablePattern.FindAllStringSubmatch(entry.Body, -1) {
		if !promptVariablePattern.MatchString(match[0]) {
			diagnostics = append(diagnostics, domain.PromptDiagnostic{Code: "variable_syntax_invalid", Message: "template variables must use snake_case names"})
		}
	}
	for _, match := range promptVariablePattern.FindAllStringSubmatch(entry.Body, -1) {
		found[match[1]] = true
		if !allowed[match[1]] {
			diagnostics = append(diagnostics, domain.PromptDiagnostic{Code: "variable_unknown", Message: "unknown template variable {{" + match[1] + "}}"})
		}
	}
	for _, name := range contract.RequiredVariables {
		if !found[name] {
			diagnostics = append(diagnostics, domain.PromptDiagnostic{Code: "variable_missing", Message: "required template variable {{" + name + "}} is missing"})
		}
	}
	return diagnostics
}

func promptRevision(entry promptEntry) string {
	hash := sha256.Sum256([]byte(entry.ID + "\x00" + entry.Category + "\x00" + entry.Title + "\x00" + fmt.Sprint(entry.Enabled) + "\x00" + entry.Body))
	return hex.EncodeToString(hash[:])
}

func renderPromptTemplate(body string, values map[string]string) (string, error) {
	missing := ""
	rendered := promptVariablePattern.ReplaceAllStringFunc(body, func(marker string) string {
		match := promptVariablePattern.FindStringSubmatch(marker)
		value, ok := values[match[1]]
		if !ok {
			missing = match[1]
			return marker
		}
		return value
	})
	if missing != "" || anyPromptVariablePattern.MatchString(rendered) {
		return "", fmt.Errorf("prompt template variable %s is unresolved", missing)
	}
	return strings.TrimSpace(rendered), nil
}

func (r *PromptRegistry) documentForID(id string) (domain.PromptDocument, bool) {
	active, hasActive := r.active[id]
	working, hasWorking := r.working[id]
	builtin, hasBuiltin := r.builtins[id]
	contract, registered := r.contracts[id]
	entry := active
	if hasWorking {
		entry = working
	} else if !hasActive && hasBuiltin {
		entry = builtin
	}
	if entry.ID == "" {
		return domain.PromptDocument{}, false
	}
	if !registered {
		contract = promptContract{Category: entry.Category, Required: entry.Category == domain.PromptCategoryAgent, Disableable: entry.Category == domain.PromptCategoryQuickPrompt, MaxLength: promptFileMaxBytes}
	}
	origin := entry.Origin
	if hasBuiltin && hasWorking {
		origin = "override"
	}
	status := "valid"
	if !entry.Enabled {
		status = "disabled"
	}
	if len(r.diagnostics[id]) > 0 {
		status = "invalid"
	}
	return domain.PromptDocument{
		ID: id, Category: entry.Category, Title: entry.Title, Body: entry.Body, Enabled: entry.Enabled, Origin: origin,
		Required: contract.Required, Disableable: contract.Disableable, Deletable: !hasBuiltin && !registered,
		Variables: append([]string(nil), contract.Variables...), RequiredVariables: append([]string(nil), contract.RequiredVariables...),
		MaxLength:       contract.MaxLength,
		WorkingRevision: entry.Revision, ActiveRevision: active.Revision, Status: status, Fallback: hasWorking && active.Revision != working.Revision,
		Diagnostics: append([]domain.PromptDiagnostic(nil), r.diagnostics[id]...),
	}, true
}

func (r *PromptRegistry) List() []domain.PromptDocument {
	r.mu.RLock()
	defer r.mu.RUnlock()
	ids := map[string]bool{}
	for id := range r.builtins {
		ids[id] = true
	}
	for id := range r.working {
		ids[id] = true
	}
	ordered := make([]string, 0, len(ids))
	for id := range ids {
		ordered = append(ordered, id)
	}
	sort.Strings(ordered)
	documents := make([]domain.PromptDocument, 0, len(ordered))
	for _, id := range ordered {
		if document, ok := r.documentForID(id); ok {
			documents = append(documents, document)
		}
	}
	return documents
}

func (r *PromptRegistry) Get(id string) (domain.PromptDocument, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if document, ok := r.documentForID(strings.TrimSpace(id)); ok {
		return document, nil
	}
	return domain.PromptDocument{}, errors.New("prompt document not found")
}

func (r *PromptRegistry) Validate(input domain.PromptDocumentInput) domain.PromptValidationResult {
	entry := promptEntry{ID: strings.TrimSpace(input.ID), Category: strings.TrimSpace(input.Category), Title: strings.TrimSpace(input.Title), Body: strings.TrimSpace(input.Body), Enabled: input.Enabled, Origin: "override"}
	diagnostics := r.validateEntry(entry, promptSchema)
	entry.Revision = promptRevision(entry)
	return domain.PromptValidationResult{Valid: len(diagnostics) == 0, Revision: entry.Revision, Diagnostics: diagnostics}
}

func (r *PromptRegistry) Save(input domain.PromptDocumentInput) (domain.PromptDocument, error) {
	input.ID, input.Category, input.Title, input.Body = strings.TrimSpace(input.ID), strings.TrimSpace(input.Category), strings.TrimSpace(input.Title), strings.TrimSpace(input.Body)
	if !promptIDPattern.MatchString(input.ID) {
		return domain.PromptDocument{}, errors.New("invalid prompt id")
	}
	if retiredPromptIDs[input.ID] {
		return domain.PromptDocument{}, errors.New("prompt id is retired")
	}
	if !containsPromptCategory(input.Category) {
		return domain.PromptDocument{}, errors.New("invalid prompt category")
	}
	raw := marshalPromptMarkdown(input)
	path := r.overridePath(input.Category, input.ID)
	if err := atomicWritePrompt(path, raw); err != nil {
		return domain.PromptDocument{}, err
	}
	if err := r.Reload(); err != nil {
		return domain.PromptDocument{}, err
	}
	return r.Get(input.ID)
}

func (r *PromptRegistry) Reset(id string) (domain.PromptDocument, error) {
	id = strings.TrimSpace(id)
	if _, builtin := r.builtins[id]; !builtin {
		return domain.PromptDocument{}, errors.New("custom prompts cannot be reset")
	}
	for _, category := range promptCategories() {
		_ = os.Remove(r.overridePath(category, id))
	}
	if err := r.Reload(); err != nil {
		return domain.PromptDocument{}, err
	}
	return r.Get(id)
}

func (r *PromptRegistry) Delete(id string) error {
	document, err := r.Get(id)
	if err != nil {
		return err
	}
	if !document.Deletable {
		return errors.New("prompt document cannot be deleted")
	}
	if err := os.Remove(r.overridePath(document.Category, document.ID)); err != nil && !os.IsNotExist(err) {
		return err
	}
	return r.Reload()
}

func (r *PromptRegistry) SetEnabled(id string, enabled bool) (domain.PromptDocument, error) {
	document, err := r.Get(id)
	if err != nil {
		return domain.PromptDocument{}, err
	}
	if !enabled && !document.Disableable {
		return domain.PromptDocument{}, errors.New("prompt document cannot be disabled")
	}
	document.Enabled = enabled
	return r.Save(domain.PromptDocumentInput{ID: document.ID, Category: document.Category, Title: document.Title, Body: document.Body, Enabled: enabled})
}

func (r *PromptRegistry) overridePath(category, id string) string {
	return filepath.Join(r.root, "overrides", category, id+".md")
}

func promptCategories() []string {
	return []string{domain.PromptCategoryAgent, domain.PromptCategoryProtocol, domain.PromptCategoryAuxiliary, domain.PromptCategoryTask, domain.PromptCategoryDynamicContext, domain.PromptCategoryQuickPrompt}
}

func containsPromptCategory(category string) bool {
	for _, candidate := range promptCategories() {
		if category == candidate {
			return true
		}
	}
	return false
}

func marshalPromptMarkdown(input domain.PromptDocumentInput) []byte {
	title, _ := json.Marshal(input.Title)
	return []byte(fmt.Sprintf("---\nschema: %s\nid: %s\ncategory: %s\ntitle: %s\nenabled: %t\n---\n\n%s\n", promptSchema, input.ID, input.Category, title, input.Enabled, input.Body))
}

func atomicWritePrompt(path string, raw []byte) error {
	if len(raw) > promptFileMaxBytes {
		return errors.New("prompt file exceeds 32768 bytes")
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	temp, err := os.CreateTemp(dir, ".prompt-*.tmp")
	if err != nil {
		return err
	}
	tempPath := temp.Name()
	defer os.Remove(tempPath)
	if err := temp.Chmod(0o600); err != nil {
		_ = temp.Close()
		return err
	}
	if _, err := temp.Write(raw); err != nil {
		_ = temp.Close()
		return err
	}
	if err := temp.Sync(); err != nil {
		_ = temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tempPath, path); err != nil {
		return err
	}
	directory, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer directory.Close()
	return directory.Sync()
}

func (r *PromptRegistry) publishValidated(entry promptEntry) error {
	raw := marshalPromptMarkdown(domain.PromptDocumentInput{ID: entry.ID, Category: entry.Category, Title: entry.Title, Body: entry.Body, Enabled: entry.Enabled})
	return atomicWritePrompt(filepath.Join(r.root, ".state", "validated", entry.Revision+".md"), raw)
}

func (r *PromptRegistry) writeManifest(active map[string]promptEntry) error {
	manifest := promptManifest{Version: 1, Active: map[string]string{}}
	for id, entry := range active {
		if entry.Origin != "builtin" {
			manifest.Active[id] = entry.Revision
		}
	}
	raw, _ := json.MarshalIndent(manifest, "", "  ")
	return atomicWritePrompt(filepath.Join(r.root, ".state", "active.json"), raw)
}

func (r *PromptRegistry) loadValidatedManifest() map[string]promptEntry {
	out := map[string]promptEntry{}
	raw, err := os.ReadFile(filepath.Join(r.root, ".state", "active.json"))
	if err != nil {
		return out
	}
	var manifest promptManifest
	if json.Unmarshal(raw, &manifest) != nil || manifest.Version != 1 {
		return out
	}
	for id, revision := range manifest.Active {
		if !promptIDPattern.MatchString(id) || len(revision) != 64 {
			continue
		}
		validatedRaw, readErr := os.ReadFile(filepath.Join(r.root, ".state", "validated", revision+".md"))
		if readErr != nil {
			continue
		}
		entry, diagnostics := r.parse(validatedRaw, "override")
		if len(diagnostics) == 0 && entry.ID == id && entry.Revision == revision {
			out[id] = entry
		}
	}
	return out
}
