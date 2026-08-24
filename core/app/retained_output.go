package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"aivo/core/domain"
)

const retainedOutputLifetime = 24 * time.Hour

const (
	defaultRetainedOutputReadLimit = 16000
	maxRetainedOutputReadLimit     = 100000
)

func (s *Service) ReadRetainedOutput(ctx context.Context, input domain.RetainedOutputReadInput) (domain.RetainedOutputReadResult, error) {
	select {
	case <-ctx.Done():
		return domain.RetainedOutputReadResult{}, ctx.Err()
	default:
	}
	ref := strings.TrimSpace(input.Ref)
	if ref == "" {
		return domain.RetainedOutputReadResult{}, fmt.Errorf("retained output ref is required")
	}
	path, err := validateRetainedOutputRef(ref)
	if err != nil {
		return domain.RetainedOutputReadResult{}, err
	}
	info, err := os.Stat(path)
	if err != nil {
		return domain.RetainedOutputReadResult{}, err
	}
	if info.IsDir() {
		return domain.RetainedOutputReadResult{}, fmt.Errorf("retained output ref is a directory")
	}
	size := int(info.Size())
	offset := input.Offset
	if offset < 0 {
		return domain.RetainedOutputReadResult{}, fmt.Errorf("offset must be non-negative")
	}
	if offset > size {
		offset = size
	}
	limit := input.Limit
	if limit <= 0 {
		limit = defaultRetainedOutputReadLimit
	}
	if limit > maxRetainedOutputReadLimit {
		limit = maxRetainedOutputReadLimit
	}
	file, err := os.Open(path)
	if err != nil {
		return domain.RetainedOutputReadResult{}, err
	}
	defer file.Close()
	buffer := make([]byte, limit)
	n, err := file.ReadAt(buffer, int64(offset))
	if err != nil && n == 0 && offset < size {
		return domain.RetainedOutputReadResult{}, err
	}
	nextOffset := offset + n
	return domain.RetainedOutputReadResult{
		Ref:        path,
		Content:    string(buffer[:n]),
		Offset:     offset,
		NextOffset: nextOffset,
		Size:       size,
		Truncated:  nextOffset < size,
	}, nil
}

func validateRetainedOutputRef(ref string) (string, error) {
	path, err := filepath.Abs(ref)
	if err != nil {
		return "", err
	}
	if !pathWithinRetainedOutputRoots(path) {
		return "", fmt.Errorf("retained output ref is outside the Aivo artifact store")
	}
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return "", err
	}
	if !pathWithinRetainedOutputRoots(resolved) {
		return "", fmt.Errorf("retained output ref resolves outside the Aivo artifact store")
	}
	return resolved, nil
}

func pathWithinRetainedOutputRoots(path string) bool {
	for _, root := range retainedOutputRoots() {
		if pathHasPrefix(path, root) {
			return true
		}
	}
	return false
}

func retainedOutputRoots() []string {
	roots := make([]string, 0, 2)
	if cacheDir, err := os.UserCacheDir(); err == nil && strings.TrimSpace(cacheDir) != "" {
		roots = append(roots, filepath.Join(cacheDir, "aivo", "command-artifacts"))
	}
	roots = append(roots, filepath.Join(os.TempDir(), "aivo", "command-artifacts"))
	return roots
}

func cleanupRetainedOutputSession(sessionID string) {
	sessionID = safeArtifactPart(sessionID)
	if sessionID == "" || sessionID == "artifact" {
		return
	}
	for _, root := range retainedOutputRoots() {
		_ = os.RemoveAll(filepath.Join(root, sessionID))
	}
}

func reclaimStaleRetainedOutput(now time.Time) {
	for _, root := range retainedOutputRoots() {
		entries, err := os.ReadDir(root)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			info, err := entry.Info()
			if err == nil && now.Sub(info.ModTime()) >= retainedOutputLifetime {
				_ = os.RemoveAll(filepath.Join(root, entry.Name()))
			}
		}
	}
}
