package persistence

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"aivo/core/domain"
)

func TestAgentModeDefinitionStoreOmitsToolsets(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "aivo.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	definition := domain.AgentModeDefinition{
		ID: "research", DisplayName: "Research", Prompt: "Investigate carefully.",
		Toolsets: []string{"coding", "web"}, Mode: "all",
	}
	if err := store.SaveAgentModeDefinition(context.Background(), definition); err != nil {
		t.Fatal(err)
	}
	var raw string
	if err := store.db.Model(&agentModeDefinitionRow{}).Where("id = ?", definition.ID).Select("definition").Scan(&raw).Error; err != nil {
		t.Fatal(err)
	}
	if strings.Contains(raw, "toolsets") || strings.Contains(raw, "coding") || strings.Contains(raw, "web") {
		t.Fatalf("persisted managed definition retained toolsets: %s", raw)
	}
	stored, err := store.ListAgentModeDefinitions(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(stored) != 1 || len(stored[0].Toolsets) != 0 {
		t.Fatalf("stored definitions = %#v", stored)
	}
}
