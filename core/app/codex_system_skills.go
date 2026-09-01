package app

import (
	"context"
	_ "embed"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"

	"aivo/core/domain"
)

const (
	aivoSystemSkillVersion  = "aivo-base-2026-08-29"
	codexSystemSkillVersion = "codex-main-2026-08-29"
)

type builtinCodexSkill struct {
	Name        string
	Description string
	Content     string
	Tool        string
}

//go:embed system_skills/codex/skill-creator/SKILL.md
var builtinSkillCreatorMarkdown string

//go:embed system_skills/codex/skill-installer/SKILL.md
var builtinSkillInstallerMarkdown string

func mustBuiltinCodexSkill(markdown string, tool string) builtinCodexSkill {
	name, description, _, content, err := parseSkillMarkdown(markdown)
	if err != nil {
		panic("invalid embedded Codex system skill: " + err.Error())
	}
	return builtinCodexSkill{Name: name, Description: description, Content: content, Tool: tool}
}

var builtinAivoSystemSkills = []builtinCodexSkill{
	mustBuiltinCodexSkill(builtinSkillCreatorMarkdown, ""),
	mustBuiltinCodexSkill(builtinSkillInstallerMarkdown, ""),
}

var fallbackCodexSystemSkills = []builtinCodexSkill{
	{Name: "imagegen", Description: "Generate or edit images with the Codex account-native image generation tool.", Tool: "image_gen.imagegen", Content: `# Codex image generation

Use the account-native image generation tool for image creation and editing.

- For a new image, provide only a complete prompt.
- For edits, provide either up to five workspace-relative image paths or a bounded count of recent conversation images, never both.
- Preserve the user's requested composition, text, colors, background, and output constraints in the prompt.
- Inspect the generated artifact when visual correctness matters and iterate with another edit if needed.
- Return the saved local image path to the user.
`},
	{Name: "openai-docs", Description: "Answer questions about OpenAI and Codex products using current official documentation.", Content: `# OpenAI documentation

Use current official OpenAI sources for OpenAI, ChatGPT, API, and Codex product questions.

- Prefer locally bundled or installed Codex documentation helpers when present.
- Otherwise search only official OpenAI documentation domains unless the user asks for broader sources.
- Distinguish Codex account features from public API-key products and cite the exact supporting page.
- Treat model names, limits, pricing, capabilities, and settings as time-sensitive and verify them before answering.
`},
	{Name: "plugin-creator", Description: "Create and validate Codex plugin packages with manifests, skills, MCP servers, and app contributions.", Content: `# Codex plugin creator

Create the smallest valid Codex plugin package for the requested capability.

- Inspect the current plugin manifest schema before writing files.
- Keep the plugin identifier, directory structure, permissions, dependencies, and contributed skills or MCP servers explicit.
- Never place credentials in the plugin package.
- Validate the manifest and referenced files before reporting completion.
`},
	{Name: "review-agent", Description: "Review code changes for correctness, regressions, security, and missing tests.", Content: `# Review agent

Review the requested diff or code scope without mutating it unless the user separately asks for fixes.

- Lead with concrete actionable findings ordered by severity.
- Cite exact file and line locations.
- Focus on correctness, regressions, security, concurrency, data integrity, and missing verification.
- Avoid style-only findings unless they hide a defect.
- State clearly when no actionable findings remain and identify residual test gaps.
`},
}

func (s *Service) syncAivoSystemSkills(ctx context.Context) {
	_ = s.ensureSkillManager().SyncAivoSystemSkills(ctx)
}

func (s *Service) syncCodexSystemSkillsForAccount(ctx context.Context) {
	if !s.codexOAuthConfigured(ctx) {
		return
	}
	_ = s.ensureSkillManager().SyncCodexSystemSkills(ctx)
}

func (s *Service) codexOAuthConfigured(ctx context.Context) bool {
	if s == nil || s.store == nil {
		return false
	}
	auth, err := s.store.LoadProviderAuth(ctx, "openai")
	return err == nil && auth != nil && isOAuthMethod(auth.Method)
}

func (m *SkillManager) SyncCodexSystemSkills(ctx context.Context) error {
	if m == nil || m.store == nil {
		return errors.New("skill store is not configured")
	}
	m.codexSystemSyncMu.Lock()
	defer m.codexSystemSyncMu.Unlock()

	sourceRoot := filepath.Join(m.home, ".codex", "skills", ".system")
	dirs := discoverSkillDirectories(sourceRoot)
	if len(dirs) == 0 {
		var err error
		dirs, err = m.materializeFallbackCodexSystemSkills()
		if err != nil {
			return err
		}
	}
	parsedSkills := make([]parsedSkill, 0, len(dirs))
	var signature strings.Builder
	for _, dir := range dirs {
		parsed, err := parseSkillDirectory(dir)
		if err != nil {
			continue
		}
		parsedSkills = append(parsedSkills, parsed)
		signature.WriteString(parsed.Name)
		signature.WriteByte('\x00')
		signature.WriteString(parsed.RootPath)
		signature.WriteByte('\x00')
		signature.WriteString(parsed.ContentHash)
		signature.WriteByte('\n')
	}
	currentHash := signature.String()
	if m.codexSystemSyncHome == m.home && m.codexSystemSyncHash == currentHash {
		return nil
	}

	now := domain.NowString(time.Now())
	present := make(map[string]bool, len(parsedSkills))
	for _, parsed := range parsedSkills {
		present[parsed.Name] = true
		existing, _ := m.store.GetSkillByName(ctx, parsed.Name, domain.SkillScopeGlobal)
		if existing.ID != "" && existing.Source != domain.SkillSourceCodexSystem {
			continue
		}
		id := existing.ID
		created := existing.TimeCreated
		if id == "" {
			id = uuid.NewString()
			created = now
		}
		metadata := parsed.Metadata
		if metadata == nil {
			metadata = map[string]string{}
		}
		metadata["aivo.system"] = "codex"
		metadata["aivo.provider"] = "openai-codex-oauth"
		metadata["aivo.version"] = codexSystemSkillVersion
		if parsed.Name == "imagegen" {
			metadata["aivo.tool"] = "image_gen.imagegen"
		}
		_, _ = m.store.SaveSkill(ctx, domain.SkillEntry{
			ID: id, Name: parsed.Name, Description: parsed.Description, Scope: domain.SkillScopeGlobal, Source: domain.SkillSourceCodexSystem,
			RootPath: parsed.RootPath, SkillPath: parsed.SkillPath, ContentHash: parsed.ContentHash, Enabled: true,
			Metadata: metadata, TimeCreated: created, TimeUpdated: now,
		})
	}
	if existing, err := m.store.ListSkills(ctx, true); err == nil {
		for _, skill := range existing {
			if skill.Source == domain.SkillSourceCodexSystem && !present[skill.Name] {
				_ = m.store.DeleteSkill(ctx, skill.ID)
			}
		}
	}
	m.codexSystemSyncHome = m.home
	m.codexSystemSyncHash = currentHash
	return nil
}

func (m *SkillManager) SyncAivoSystemSkills(ctx context.Context) error {
	if m == nil || m.store == nil {
		return errors.New("skill store is not configured")
	}
	m.codexSystemSyncMu.Lock()
	defer m.codexSystemSyncMu.Unlock()

	dirs, err := m.materializeBuiltinSystemSkills("aivo", aivoSystemSkillVersion, builtinAivoSystemSkills, map[string]string{
		"aivo.system":  "aivo",
		"aivo.version": aivoSystemSkillVersion,
	})
	if err != nil {
		return err
	}
	now := domain.NowString(time.Now())
	for _, dir := range dirs {
		parsed, err := parseSkillDirectory(dir)
		if err != nil {
			continue
		}
		existing, _ := m.store.GetSkillByName(ctx, parsed.Name, domain.SkillScopeGlobal)
		if existing.ID != "" && existing.Source != domain.SkillSourceAivoSystem && existing.Source != domain.SkillSourceCodexSystem {
			continue
		}
		if existing.ID != "" && existing.Source == domain.SkillSourceAivoSystem && existing.ContentHash == parsed.ContentHash && existing.Metadata["aivo.version"] == aivoSystemSkillVersion && existing.Enabled {
			continue
		}
		id := existing.ID
		created := existing.TimeCreated
		if id == "" {
			id = uuid.NewString()
			created = now
		}
		metadata := parsed.Metadata
		if metadata == nil {
			metadata = map[string]string{}
		}
		metadata["aivo.system"] = "aivo"
		metadata["aivo.version"] = aivoSystemSkillVersion
		if _, err := m.store.SaveSkill(ctx, domain.SkillEntry{
			ID: id, Name: parsed.Name, Description: parsed.Description, Scope: domain.SkillScopeGlobal, Source: domain.SkillSourceAivoSystem,
			RootPath: parsed.RootPath, SkillPath: parsed.SkillPath, ContentHash: parsed.ContentHash, Enabled: true,
			Metadata: metadata, TimeCreated: created, TimeUpdated: now,
		}); err != nil {
			return err
		}
	}
	return nil
}

func (m *SkillManager) materializeFallbackCodexSystemSkills() ([]string, error) {
	return m.materializeBuiltinSystemSkills("codex", codexSystemSkillVersion, fallbackCodexSystemSkills, map[string]string{
		"aivo.system":   "codex",
		"aivo.provider": "openai-codex-oauth",
		"aivo.version":  codexSystemSkillVersion,
	})
}

func (m *SkillManager) materializeBuiltinSystemSkills(system string, version string, skills []builtinCodexSkill, metadata map[string]string) ([]string, error) {
	root := filepath.Join(m.home, ".aivo", "system-skills", system, version)
	if err := os.MkdirAll(root, 0o700); err != nil {
		return nil, err
	}
	dirs := make([]string, 0, len(skills))
	for _, skill := range skills {
		dir := filepath.Join(root, skill.Name)
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return nil, err
		}
		entryMetadata := make(map[string]string, len(metadata)+1)
		for key, value := range metadata {
			entryMetadata[key] = value
		}
		if skill.Tool != "" {
			entryMetadata["aivo.tool"] = skill.Tool
		}
		raw := marshalSkillMarkdown(skill.Name, skill.Description, entryMetadata, strings.TrimSpace(skill.Content)+"\n")
		path := filepath.Join(dir, "SKILL.md")
		if current, err := os.ReadFile(path); err != nil || string(current) != string(raw) {
			if err := atomicReplaceFile(path, raw, 0o600); err != nil {
				return nil, err
			}
		}
		dirs = append(dirs, dir)
	}
	sort.Strings(dirs)
	return dirs, nil
}
