package app

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"aivo/core/domain"
)

func TestCodexOAuthAccountAutomaticallySynchronizesReadOnlySystemSkills(t *testing.T) {
	ctx := context.Background()
	home := t.TempDir()
	t.Setenv("HOME", home)
	service, cleanup := newSkillTestService(t)
	defer cleanup()
	if err := service.store.SaveProviderAuth(ctx, domain.ProviderAuthRecord{ProviderID: "openai", Method: "oauth-browser", AccessToken: "token"}); err != nil {
		t.Fatal(err)
	}

	result, err := service.ListSkills(ctx, domain.SkillListInput{})
	if err != nil {
		t.Fatal(err)
	}
	byName := map[string]domain.SkillEntry{}
	for _, skill := range result.Entries {
		byName[skill.Name] = skill
	}
	expectedInstructions := map[string]string{
		"skill-creator":   "Use an independent subagent pass when a skill is sufficiently complex or risky",
		"skill-installer": "Install skills with the helper scripts.",
	}
	for _, name := range []string{"imagegen", "openai-docs", "plugin-creator", "review-agent", "skill-creator", "skill-installer"} {
		skill, ok := byName[name]
		expectedSource := domain.SkillSourceCodexSystem
		if name == "skill-creator" || name == "skill-installer" {
			expectedSource = domain.SkillSourceAivoSystem
		}
		if !ok || skill.Source != expectedSource {
			t.Fatalf("system skill %q = %+v", name, skill)
		}
		if expectedSource == domain.SkillSourceCodexSystem && skill.Metadata["aivo.provider"] != "openai-codex-oauth" {
			t.Fatalf("Codex system skill %q metadata = %+v", name, skill.Metadata)
		}
		if _, err := os.Stat(skill.SkillPath); err != nil {
			t.Fatalf("system skill %q was not materialized: %v", name, err)
		}
		if expected, ok := expectedInstructions[name]; ok {
			content, err := service.ensureSkillManager().ReadContent(skill)
			if err != nil || !strings.Contains(content, expected) {
				t.Fatalf("system skill %q does not contain its embedded instructions: %v, %q", name, err, content)
			}
		}
	}
	imagegen := byName["imagegen"]
	if imagegen.Metadata["aivo.tool"] != "image_gen.imagegen" {
		t.Fatalf("imagegen metadata = %#v", imagegen.Metadata)
	}
	if _, err := service.GetManagedSkillForEdit(ctx, imagegen.ID); err == nil {
		t.Fatal("Codex system skill unexpectedly editable")
	}
	if _, err := service.SetSkillEnabled(ctx, domain.SkillEnabledInput{SkillID: imagegen.ID, Enabled: false}); err == nil {
		t.Fatal("Codex system skill unexpectedly toggleable")
	}
	if err := service.DeleteManagedSkill(ctx, imagegen.ID); err == nil {
		t.Fatal("Codex system skill unexpectedly deletable")
	}
	if !skillHasAction(imagegen, domain.SkillActionActivate) || skillHasAction(imagegen, domain.SkillActionSetEnabled) || skillHasAction(imagegen, domain.SkillActionEdit) || skillHasAction(imagegen, domain.SkillActionDelete) {
		t.Fatalf("imagegen actions = %#v, want activate only", imagegen.Actions)
	}
	catalog, err := service.Catalog(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, provider := range catalog.Providers {
		if provider.ID == "openai" {
			if provider.NativeCapabilities == nil || !provider.NativeCapabilities.NamespaceTools || !provider.NativeCapabilities.ImageGeneration || !provider.NativeCapabilities.WebSearch {
				t.Fatalf("native capabilities = %#v", provider.NativeCapabilities)
			}
			return
		}
	}
	t.Fatal("OpenAI provider missing from catalog")
}

func TestCodexSystemSkillSyncUsesInstalledCodexSystemAssets(t *testing.T) {
	ctx := context.Background()
	home := t.TempDir()
	t.Setenv("HOME", home)
	service, cleanup := newSkillTestService(t)
	defer cleanup()
	if err := service.store.SaveProviderAuth(ctx, domain.ProviderAuthRecord{ProviderID: "openai", Method: "oauth-browser", AccessToken: "token"}); err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(home, ".codex", "skills", ".system", "upstream-skill")
	if err := os.MkdirAll(root, 0o700); err != nil {
		t.Fatal(err)
	}
	raw := marshalSkillMarkdown("upstream-skill", "Installed by Codex", map[string]string{"upstream": "true"}, "Use the installed upstream instructions.\n")
	if err := os.WriteFile(filepath.Join(root, "SKILL.md"), raw, 0o600); err != nil {
		t.Fatal(err)
	}

	result, err := service.ListSkills(ctx, domain.SkillListInput{})
	if err != nil {
		t.Fatal(err)
	}
	var upstream domain.SkillEntry
	for _, skill := range result.Entries {
		if skill.Name == "upstream-skill" {
			upstream = skill
			break
		}
	}
	if upstream.ID == "" || upstream.Source != domain.SkillSourceCodexSystem {
		t.Fatalf("entries = %#v", result.Entries)
	}
	if upstream.RootPath != root || upstream.Metadata["upstream"] != "true" {
		t.Fatalf("entry = %+v", upstream)
	}
	firstHash := upstream.ContentHash
	updated := marshalSkillMarkdown("upstream-skill", "Updated by Codex", map[string]string{"upstream": "true"}, "Use the refreshed upstream instructions.\n")
	if err := os.WriteFile(filepath.Join(root, "SKILL.md"), updated, 0o600); err != nil {
		t.Fatal(err)
	}
	result, err = service.ListSkills(ctx, domain.SkillListInput{})
	if err != nil {
		t.Fatal(err)
	}
	upstream = domain.SkillEntry{}
	for _, skill := range result.Entries {
		if skill.Name == "upstream-skill" {
			upstream = skill
			break
		}
	}
	if upstream.ID == "" || upstream.ContentHash == firstHash || upstream.Description != "Updated by Codex" {
		t.Fatalf("refreshed entries = %#v", result.Entries)
	}
}

func TestCodexSystemSkillsAreHiddenAfterOAuthAccountDisconnects(t *testing.T) {
	ctx := context.Background()
	t.Setenv("HOME", t.TempDir())
	service, cleanup := newSkillTestService(t)
	defer cleanup()
	if err := service.store.SaveProviderAuth(ctx, domain.ProviderAuthRecord{ProviderID: "openai", Method: "oauth-browser", AccessToken: "token"}); err != nil {
		t.Fatal(err)
	}
	result, err := service.ListSkills(ctx, domain.SkillListInput{})
	if err != nil || len(result.Entries) == 0 {
		t.Fatalf("connected entries = %#v, err = %v", result.Entries, err)
	}
	auth, err := service.store.LoadProviderAuth(ctx, "openai")
	if err != nil || auth == nil {
		t.Fatalf("auth = %#v, err = %v", auth, err)
	}
	if err := service.store.DeleteProviderAuth(ctx, auth.ID); err != nil {
		t.Fatal(err)
	}
	result, err = service.ListSkills(ctx, domain.SkillListInput{IncludeDisabled: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Entries) != 2 {
		t.Fatalf("disconnected entries = %#v, want Aivo base skills only", result.Entries)
	}
	for _, skill := range result.Entries {
		if skill.Source != domain.SkillSourceAivoSystem {
			t.Fatalf("disconnected entry = %#v, want Aivo base skill", skill)
		}
	}
}

func TestAivoBaseSystemSkillsDoNotRequireCodexOAuth(t *testing.T) {
	ctx := context.Background()
	home := t.TempDir()
	t.Setenv("HOME", home)
	service, cleanup := newSkillTestService(t)
	defer cleanup()
	materialized, err := service.ensureSkillManager().List(ctx, domain.SkillListInput{})
	if err != nil || len(materialized.Entries) != 2 {
		t.Fatalf("startup Aivo base skills = %#v, err = %v", materialized.Entries, err)
	}
	if err := service.store.SaveProviderAuth(ctx, domain.ProviderAuthRecord{ProviderID: "openai", Method: "api-key", APIKey: "key"}); err != nil {
		t.Fatal(err)
	}

	result, err := service.ListSkills(ctx, domain.SkillListInput{})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Entries) != 2 {
		t.Fatalf("API-key skills = %#v, want Aivo base skills only", result.Entries)
	}
	var creator domain.SkillEntry
	for _, skill := range result.Entries {
		if skill.Source != domain.SkillSourceAivoSystem || skill.Metadata["aivo.system"] != "aivo" {
			t.Fatalf("base skill = %#v", skill)
		}
		if skill.Name == "skill-creator" {
			creator = skill
		}
	}
	if creator.ID == "" {
		t.Fatal("skill-creator is missing from the base skills")
	}
	if _, err := service.GetManagedSkillForEdit(ctx, creator.ID); err == nil {
		t.Fatal("Aivo base system skill unexpectedly editable")
	}
	if _, err := service.SetSkillEnabled(ctx, domain.SkillEnabledInput{SkillID: creator.ID, Enabled: false}); err == nil {
		t.Fatal("Aivo base system skill unexpectedly toggleable")
	}
	if !skillHasAction(creator, domain.SkillActionActivate) || skillHasAction(creator, domain.SkillActionSetEnabled) || skillHasAction(creator, domain.SkillActionEdit) || skillHasAction(creator, domain.SkillActionDelete) {
		t.Fatalf("creator actions = %#v, want activate only", creator.Actions)
	}
	if _, err := service.ensureSkillManager().store.SetSkillEnabled(ctx, creator.ID, false); err != nil {
		t.Fatal(err)
	}
	result, err = service.ListSkills(ctx, domain.SkillListInput{})
	if err != nil {
		t.Fatal(err)
	}
	creator, ok := skillEntryByName(result.Entries, "skill-creator")
	if !ok || !creator.Enabled {
		t.Fatalf("Aivo base skill was not repaired by Host sync: %#v", creator)
	}
	if err := service.DeleteManagedSkill(ctx, creator.ID); err == nil {
		t.Fatal("Aivo base skill unexpectedly deletable")
	}
	session, err := service.CreateRuntimeSession(ctx, domain.CreateSessionRequest{Type: domain.SessionTypeCoding, ProjectPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.LoadSkillIntoSession(ctx, domain.LoadSkillIntoSessionInput{SessionID: session.ID, SkillID: creator.ID}); err != nil {
		t.Fatalf("load Aivo base skill: %v", err)
	}
}

func TestToolDependentCodexSystemSkillsHideWhenBackingToolIsDisabled(t *testing.T) {
	ctx := context.Background()
	t.Setenv("HOME", t.TempDir())
	service, cleanup := newSkillTestService(t)
	defer cleanup()
	if err := service.store.SaveProviderAuth(ctx, domain.ProviderAuthRecord{ProviderID: "openai", Method: "oauth-browser", AccessToken: "token"}); err != nil {
		t.Fatal(err)
	}
	cfg, err := service.AppConfig(ctx)
	if err != nil {
		t.Fatal(err)
	}
	cfg.NativeTools.Disabled = []string{CodexImagegenToolName}
	if err := service.store.SaveConfig(ctx, cfg); err != nil {
		t.Fatal(err)
	}

	result, err := service.ListSkills(ctx, domain.SkillListInput{})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := skillEntryByName(result.Entries, "imagegen"); ok {
		t.Fatalf("imagegen skill should be hidden while %s is disabled: %#v", CodexImagegenToolName, result.Entries)
	}
	if _, ok := skillEntryByName(result.Entries, "openai-docs"); !ok {
		t.Fatalf("non-tool Codex system skill should remain visible: %#v", result.Entries)
	}
}

func skillHasAction(skill domain.SkillEntry, action string) bool {
	for _, item := range skill.Actions {
		if item == action {
			return true
		}
	}
	return false
}
