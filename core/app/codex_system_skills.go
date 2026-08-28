package app

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"

	"aivo/core/domain"
)

const codexSystemSkillVersion = "codex-main-2026-08-28"

type builtinCodexSkill struct {
	Name        string
	Description string
	Content     string
	Tool        string
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
	{Name: "skill-creator", Description: "Create or update Codex Agent Skills with scoped instructions and supporting resources.", Content: `# Codex skill creator

Create a focused Agent Skill with a valid SKILL.md frontmatter name and a precise trigger-oriented description.

- Keep the main instructions concise and move detailed reusable material into references or scripts.
- Use lowercase kebab-case names and avoid overlapping unrelated workflows.
- Treat skill text as instructions, never as new execution authority.
- Validate the skill directory, referenced resources, and examples before completion.
`},
	{Name: "skill-installer", Description: "Discover and install Codex Agent Skills from approved curated or repository sources.", Content: `# Codex skill installer

Install only the skill source explicitly selected by the user.

- Inspect the source, license, target directory, and existing-name conflicts before installation.
- Never copy credentials, repository secrets, or unrelated files.
- Preserve supporting references and scripts needed by the skill.
- Validate the installed SKILL.md and report where it was installed; do not silently replace a user-modified skill.
`},
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
		enabled := true
		if id == "" {
			id = uuid.NewString()
			created = now
		} else {
			enabled = existing.Enabled
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
			RootPath: parsed.RootPath, SkillPath: parsed.SkillPath, ContentHash: parsed.ContentHash, Enabled: enabled,
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

func (m *SkillManager) materializeFallbackCodexSystemSkills() ([]string, error) {
	root := filepath.Join(m.home, ".aivo", "system-skills", "codex", codexSystemSkillVersion)
	if err := os.MkdirAll(root, 0o700); err != nil {
		return nil, err
	}
	dirs := make([]string, 0, len(fallbackCodexSystemSkills))
	for _, skill := range fallbackCodexSystemSkills {
		dir := filepath.Join(root, skill.Name)
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return nil, err
		}
		metadata := map[string]string{"aivo.system": "codex", "aivo.provider": "openai-codex-oauth", "aivo.version": codexSystemSkillVersion}
		if skill.Tool != "" {
			metadata["aivo.tool"] = skill.Tool
		}
		raw := marshalSkillMarkdown(skill.Name, skill.Description, metadata, strings.TrimSpace(skill.Content)+"\n")
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
