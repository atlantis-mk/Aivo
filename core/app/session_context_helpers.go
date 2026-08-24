package app

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"aivo/core/domain"
)

func (s *Service) fallbackSummary(ctx context.Context, input domain.CreateSummaryRequest) string {
	events, previous, err := s.summaryEvents(ctx, input)
	if err != nil || len(events) == 0 {
		return "No visible events have been recorded yet."
	}
	if len(events) > 5 {
		events = events[len(events)-5:]
	}
	parts := make([]string, 0, len(events))
	if previous != nil && strings.TrimSpace(previous.Summary) != "" {
		parts = append(parts, bounded(previous.Summary, 1000))
	}
	for _, event := range events {
		if event.Content != "" {
			parts = append(parts, bounded(event.Content, 160))
		}
	}
	if len(parts) == 0 {
		return "Visible activity was recorded without message content."
	}
	return strings.Join(parts, "\n")
}

func gitOutput(ctx context.Context, dir string, args ...string) string {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func detectLanguageStack(dir string) []string {
	var stack []string
	if fileExists(filepath.Join(dir, "go.mod")) {
		stack = append(stack, "go")
	}
	if fileExists(filepath.Join(dir, "package.json")) {
		stack = append(stack, "typescript", "node")
	}
	return stack
}

func detectPackageManager(dir string) string {
	for _, item := range []struct{ file, name string }{{"pnpm-lock.yaml", "pnpm"}, {"yarn.lock", "yarn"}, {"package-lock.json", "npm"}, {"go.mod", "go"}} {
		if fileExists(filepath.Join(dir, item.file)) {
			return item.name
		}
	}
	return ""
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func lines(raw string) []string {
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, "\n")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if clean := strings.TrimSpace(part); clean != "" {
			out = append(out, clean)
		}
	}
	return out
}

func renderEvents(events []domain.SessionEvent) string {
	var out []string
	for _, event := range events {
		out = append(out, event.Type+": "+event.Content)
	}
	return strings.Join(out, "\n")
}

func renderTools(tools []domain.ToolCall) string {
	var out []string
	for _, tool := range tools {
		if tool.ResultSummary != "" {
			out = append(out, tool.Name+": "+tool.ResultSummary)
		}
	}
	return strings.Join(out, "\n")
}

func applyContextBudget(sessionID string, sections []domain.ContextSection, charBudget int, maxTokens int) domain.BuildSessionContextResult {
	if charBudget <= 0 && maxTokens > 0 {
		charBudget = maxTokens * 4
	}
	if charBudget <= 0 {
		charBudget = 12000
	}
	used := 0
	var truncated []string
	for i := range sections {
		content := sections[i].Content
		remaining := charBudget - used
		if remaining <= 0 {
			if content != "" {
				truncated = append(truncated, sections[i].Name)
			}
			sections[i].Content = ""
			sections[i].Truncated = true
			continue
		}
		if len(content) > remaining {
			sections[i].Content = content[:remaining]
			sections[i].Truncated = true
			truncated = append(truncated, sections[i].Name)
		}
		used += len(sections[i].Content)
	}
	return domain.BuildSessionContextResult{SessionID: sessionID, Sections: sections, EstimatedTokens: used / 4, CharacterBudget: charBudget, TruncatedSections: truncated}
}

func bounded(value string, limit int) string {
	value = strings.TrimSpace(value)
	if limit <= 0 || len(value) <= limit {
		return value
	}
	return value[:limit]
}
