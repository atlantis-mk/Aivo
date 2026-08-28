package app

import (
	"context"
	"os"
	"path/filepath"
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
	for _, name := range []string{"imagegen", "openai-docs", "plugin-creator", "review-agent", "skill-creator", "skill-installer"} {
		skill, ok := byName[name]
		if !ok || skill.Source != domain.SkillSourceCodexSystem || skill.Metadata["aivo.provider"] != "openai-codex-oauth" {
			t.Fatalf("system skill %q = %+v", name, skill)
		}
		if _, err := os.Stat(skill.SkillPath); err != nil {
			t.Fatalf("system skill %q was not materialized: %v", name, err)
		}
	}
	imagegen := byName["imagegen"]
	if imagegen.Metadata["aivo.tool"] != "image_gen.imagegen" {
		t.Fatalf("imagegen metadata = %#v", imagegen.Metadata)
	}
	if _, err := service.GetManagedSkillForEdit(ctx, imagegen.ID); err == nil {
		t.Fatal("Codex system skill unexpectedly editable")
	}
	if err := service.DeleteManagedSkill(ctx, imagegen.ID); err == nil {
		t.Fatal("Codex system skill unexpectedly deletable")
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
	if len(result.Entries) != 1 || result.Entries[0].Name != "upstream-skill" || result.Entries[0].Source != domain.SkillSourceCodexSystem {
		t.Fatalf("entries = %#v", result.Entries)
	}
	if result.Entries[0].RootPath != root || result.Entries[0].Metadata["upstream"] != "true" {
		t.Fatalf("entry = %+v", result.Entries[0])
	}
	firstHash := result.Entries[0].ContentHash
	updated := marshalSkillMarkdown("upstream-skill", "Updated by Codex", map[string]string{"upstream": "true"}, "Use the refreshed upstream instructions.\n")
	if err := os.WriteFile(filepath.Join(root, "SKILL.md"), updated, 0o600); err != nil {
		t.Fatal(err)
	}
	result, err = service.ListSkills(ctx, domain.SkillListInput{})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Entries) != 1 || result.Entries[0].ContentHash == firstHash || result.Entries[0].Description != "Updated by Codex" {
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
	if len(result.Entries) != 0 {
		t.Fatalf("disconnected entries = %#v", result.Entries)
	}
}

func TestAPIKeyOpenAIRouteDoesNotInstallCodexAccountSystemSkills(t *testing.T) {
	ctx := context.Background()
	home := t.TempDir()
	t.Setenv("HOME", home)
	service, cleanup := newSkillTestService(t)
	defer cleanup()
	if err := service.store.SaveProviderAuth(ctx, domain.ProviderAuthRecord{ProviderID: "openai", Method: "api-key", APIKey: "key"}); err != nil {
		t.Fatal(err)
	}

	result, err := service.ListSkills(ctx, domain.SkillListInput{})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Entries) != 0 {
		t.Fatalf("API-key skills = %#v, want no Codex account system skills", result.Entries)
	}
}
