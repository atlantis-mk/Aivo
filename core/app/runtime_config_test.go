package app

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEffectiveRuntimeConfigMergesGlobalAndProjectWithProvenance(t *testing.T) {
	home := t.TempDir()
	root := t.TempDir()
	t.Setenv("HOME", home)
	writeRuntimeConfigTestFile(t, filepath.Join(home, ".config", "aivo", "config.json"), `{
  "instructions": ["global.md"],
  "commands": {"review": {"template": "Review {{path}}"}},
  "compaction": {"thresholdPercent": 70},
  "maxParallelChildren": 2
}`)
	writeRuntimeConfigTestFile(t, filepath.Join(root, ".aivo", "config.json"), `{
  "instructions": ["project.md"],
  "commands": {"review": {"template": "Project review {{path}}"}},
  "compaction": {"reserveTokens": 2048}
}`)

	result, err := (&Service{}).EffectiveRuntimeConfig(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Sources) != 2 || len(result.Diagnostics) != 0 {
		t.Fatalf("sources = %#v diagnostics = %#v", result.Sources, result.Diagnostics)
	}
	if len(result.Config.Instructions) != 2 || result.Config.Commands["review"].Template != "Project review {{path}}" {
		t.Fatalf("config = %#v", result.Config)
	}
	if result.Config.Compaction.ThresholdPercent != 70 || result.Config.Compaction.ReserveTokens != 2048 || result.Config.MaxParallelChildren != 2 {
		t.Fatalf("merged runtime limits = %#v", result.Config)
	}
}

func TestEffectiveRuntimeConfigRejectsUnknownAndSecretValues(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	root := t.TempDir()
	writeRuntimeConfigTestFile(t, filepath.Join(root, "aivo.json"), `{"unknown": true}`)
	writeRuntimeConfigTestFile(t, filepath.Join(root, ".aivo", "config.json"), `{
  "providerExtensions": {"unsafe": {"protocol": "openai", "headers": {"Authorization": "secret"}}}
}`)
	result := loadEffectiveRuntimeConfig(root)
	if len(result.Sources) != 0 || len(result.Diagnostics) != 2 {
		t.Fatalf("result = %#v", result)
	}
}

func TestEffectiveRuntimeConfigLoadsJSONCAndNestedMarkdownEntries(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	root := t.TempDir()
	writeRuntimeConfigTestFile(t, filepath.Join(root, "aivo.jsonc"), `{
  // Project defaults may use JSONC comments and trailing commas.
  "defaultAgent": "release/manager",
  "maxParallelChildren": 6,
}`)
	writeRuntimeConfigTestFile(t, filepath.Join(root, ".aivo", "agents", "release", "manager.md"), `---
name: Release Manager
description: Coordinates release work
mode: all
model: openai/gpt-5
temperature: 0.25
top_p: 0.8
toolsets:
  - safe
  - git
options:
  reasoning_effort: high
---
Coordinate the release carefully.`)
	writeRuntimeConfigTestFile(t, filepath.Join(root, ".aivo", "commands", "quality", "audit.md"), `---
description: Run a delegated quality audit
agent: release/manager
subtask: true
toolsets: [safe, git]
---
Audit $ARGUMENTS and report findings.`)

	result := loadEffectiveRuntimeConfig(root)
	if len(result.Diagnostics) != 0 {
		t.Fatalf("diagnostics = %#v", result.Diagnostics)
	}
	if len(result.Sources) != 3 || result.Config.DefaultAgent != "release/manager" || result.Config.MaxParallelChildren != 6 {
		t.Fatalf("effective config = %#v sources = %#v", result.Config, result.Sources)
	}
	agent := result.Config.Agents["release/manager"]
	if agent.DisplayName != "Release Manager" || agent.Mode != "all" || agent.Model == nil || agent.Model.ProviderID != "openai" || agent.Model.ModelID != "gpt-5" {
		t.Fatalf("agent = %#v", agent)
	}
	if agent.TopP == nil || *agent.TopP != 0.8 || agent.Temperature == nil || *agent.Temperature != 0.25 || len(agent.Toolsets) != 2 || agent.Options["reasoning_effort"] != "high" {
		t.Fatalf("agent generation/tool settings = %#v", agent)
	}
	command := result.Config.Commands["quality/audit"]
	if !command.Subtask || command.Agent != "release/manager" || !strings.Contains(command.Template, "$ARGUMENTS") || len(command.Toolsets) != 2 {
		t.Fatalf("command = %#v", command)
	}
	definition, err := NewAgentCatalogWithRuntime(result.Config).Get("release/manager")
	if err != nil || definition.Mode != "all" || definition.TopP == nil || definition.Options["reasoning_effort"] != "high" {
		t.Fatalf("catalog definition = %#v err = %v", definition, err)
	}
}

func TestEffectiveRuntimeConfigRejectsInvalidEffectiveDefaultAgent(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	root := t.TempDir()
	writeRuntimeConfigTestFile(t, filepath.Join(root, "aivo.json"), `{"defaultAgent":"worker","agents":{"worker":{"mode":"subagent"}}}`)
	result := loadEffectiveRuntimeConfig(root)
	if len(result.Diagnostics) != 1 || !strings.Contains(result.Diagnostics[0].Message, "defaultAgent") {
		t.Fatalf("diagnostics = %#v", result.Diagnostics)
	}
}

func writeRuntimeConfigTestFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}
