package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"aivo/core/domain"
	"aivo/core/infra/persistence"
)

func newSessionTestService(t *testing.T) (*Service, func()) {
	t.Helper()
	store, err := persistence.Open(filepath.Join(t.TempDir(), "aivo.db"))
	if err != nil {
		t.Fatal(err)
	}
	service := NewService(store)
	return service, func() { _ = store.Close() }
}

func sessionEventContains(events []domain.SessionEvent, eventType string, parts ...string) bool {
	for _, event := range events {
		if event.Type != eventType {
			continue
		}
		matches := true
		for _, part := range parts {
			if !strings.Contains(event.Content, part) {
				matches = false
				break
			}
		}
		if matches {
			return true
		}
	}
	return false
}

func countSessionEventsContaining(events []domain.SessionEvent, eventType string, part string) int {
	count := 0
	for _, event := range events {
		if event.Type == eventType && strings.Contains(event.Content, part) {
			count++
		}
	}
	return count
}

func contains(value string, needle string) bool {
	return strings.Contains(value, needle)
}

func writeSessionProjectFile(t *testing.T, root string, rel string, content string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}
