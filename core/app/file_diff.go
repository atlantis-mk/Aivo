package app

import (
	"path/filepath"
	"strings"
)

func countLineDelta(oldText string, newText string) (int, int) {
	oldLines := splitComparableLines(oldText)
	newLines := splitComparableLines(newText)
	oldCounts := map[string]int{}
	for _, line := range oldLines {
		oldCounts[line]++
	}
	additions := 0
	for _, line := range newLines {
		if oldCounts[line] > 0 {
			oldCounts[line]--
		} else {
			additions++
		}
	}
	newCounts := map[string]int{}
	for _, line := range newLines {
		newCounts[line]++
	}
	deletions := 0
	for _, line := range oldLines {
		if newCounts[line] > 0 {
			newCounts[line]--
		} else {
			deletions++
		}
	}
	return additions, deletions
}

func splitComparableLines(text string) []string {
	text = strings.TrimSuffix(text, "\n")
	if text == "" {
		return nil
	}
	return strings.Split(text, "\n")
}

func simpleFileDiff(oldPath string, newPath string, oldText string, newText string) string {
	var b strings.Builder
	b.WriteString("--- " + oldPath + "\n")
	b.WriteString("+++ " + newPath + "\n")
	for _, line := range splitComparableLines(oldText) {
		b.WriteString("-" + line + "\n")
	}
	for _, line := range splitComparableLines(newText) {
		b.WriteString("+" + line + "\n")
	}
	return b.String()
}

func cleanPatchPath(path string) string {
	return filepath.ToSlash(filepath.Clean(strings.TrimSpace(path)))
}
