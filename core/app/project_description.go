package app

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"aivo/core/domain"
)

const projectDescriptionSystemPrompt = `You describe software projects for a project picker. Output exactly one factual sentence, in the same language as the supplied project material. State what the project does, not implementation details. Do not use markdown, labels, quotes, or more than 160 characters.`

func (s *Service) refineProjectDescription(projectPath string) {
	models := s.configuredAuxiliaryModels(context.Background())
	if len(models) == 0 {
		return
	}
	contextText := projectDescriptionContext(projectPath)
	if contextText == "" {
		return
	}
	for _, model := range models {
		description, _, err := s.GenerateChatReply(context.Background(), []domain.ChatMessage{
			{Role: "system", Text: projectDescriptionSystemPrompt},
			{Role: "user", Text: "Project path: " + projectPath + "\n\nProject material:\n" + contextText},
		}, &model, "low", "default")
		description = cleanProjectDescription(description)
		if err == nil && description != "" {
			_, _ = s.store.UpdateProjectDescription(context.Background(), projectPath, description)
			return
		}
	}
}

func fallbackProjectDescription(projectPath string) string {
	name := filepath.Base(projectPath)
	if contextText := projectDescriptionContext(projectPath); contextText != "" {
		for index, line := range strings.Split(contextText, "\n") {
			line = strings.TrimSpace(line)
			if index == 0 || line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "{") {
				continue
			}
			if len(line) >= 20 {
				return cleanProjectDescription(line)
			}
		}
	}
	return "Project " + name + "."
}

func projectDescriptionContext(projectPath string) string {
	for _, name := range []string{"README.md", "README", "package.json", "pyproject.toml", "go.mod", "Cargo.toml"} {
		content, err := os.ReadFile(filepath.Join(projectPath, name))
		if err != nil || len(content) == 0 {
			continue
		}
		text := strings.TrimSpace(string(content))
		if len(text) > 4000 {
			text = text[:4000]
		}
		return name + "\n" + text
	}
	return ""
}

func cleanProjectDescription(value string) string {
	value = strings.TrimSpace(stripThinkBlocks(value))
	value = strings.Trim(value, "`\"' ")
	value = strings.Join(strings.Fields(value), " ")
	if len(value) > 160 {
		value = strings.TrimSpace(value[:160])
	}
	return value
}
