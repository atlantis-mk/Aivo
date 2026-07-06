package app

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

type fileSnapshot struct {
	ID        string `json:"snapshotId"`
	Path      string `json:"path"`
	SHA256    string `json:"sha256"`
	MTime     string `json:"mtime"`
	Size      int64  `json:"size"`
	LineRange string `json:"lineRange,omitempty"`
	Truncated bool   `json:"truncated,omitempty"`
}

type staleFileError struct {
	Path         string
	ExpectedHash string
	CurrentHash  string
}

func (err staleFileError) Error() string {
	return fmt.Sprintf("file changed after it was read or approved: %s (expected %s, current %s). Read it again before editing.", err.Path, err.ExpectedHash, err.CurrentHash)
}

var fileMutationLocks sync.Map

func lockForFile(path string) *sync.Mutex {
	clean := filepath.Clean(path)
	value, _ := fileMutationLocks.LoadOrStore(clean, &sync.Mutex{})
	return value.(*sync.Mutex)
}

func snapshotForBytes(relPath string, absPath string, raw []byte, lineRange string, truncated bool) (fileSnapshot, error) {
	info, err := os.Stat(absPath)
	if err != nil {
		return fileSnapshot{}, err
	}
	hash := sha256.Sum256(raw)
	encoded := hex.EncodeToString(hash[:])
	clean := cleanPatchPath(relPath)
	return fileSnapshot{
		ID:        clean + "@" + encoded[:12],
		Path:      clean,
		SHA256:    encoded,
		MTime:     info.ModTime().UTC().Format(time.RFC3339Nano),
		Size:      info.Size(),
		LineRange: lineRange,
		Truncated: truncated,
	}, nil
}

func readFileSnapshot(relPath string, absPath string, lineRange string, truncated bool) (fileSnapshot, []byte, error) {
	raw, err := os.ReadFile(absPath)
	if err != nil {
		return fileSnapshot{}, nil, err
	}
	snapshot, err := snapshotForBytes(relPath, absPath, raw, lineRange, truncated)
	if err != nil {
		return fileSnapshot{}, nil, err
	}
	return snapshot, raw, nil
}

func fileHashIfExists(absPath string) (string, bool, error) {
	raw, err := os.ReadFile(absPath)
	if errors.Is(err, os.ErrNotExist) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	hash := sha256.Sum256(raw)
	return hex.EncodeToString(hash[:]), true, nil
}

func writeFileIfUnchanged(absPath string, relPath string, expectedHash string, content []byte, perm os.FileMode) error {
	lock := lockForFile(absPath)
	lock.Lock()
	defer lock.Unlock()
	expectedHash = strings.TrimSpace(expectedHash)
	if expectedHash != "" {
		currentHash, exists, err := fileHashIfExists(absPath)
		if err != nil {
			return err
		}
		if !exists {
			currentHash = "<missing>"
		}
		if expectedHash == "<missing>" && !exists {
			currentHash = "<missing>"
		}
		if currentHash != expectedHash {
			return staleFileError{Path: cleanPatchPath(relPath), ExpectedHash: expectedHash, CurrentHash: currentHash}
		}
	}
	if err := os.MkdirAll(filepath.Dir(absPath), 0o700); err != nil {
		return err
	}
	return os.WriteFile(absPath, content, perm)
}

func removeFileIfUnchanged(absPath string, relPath string, expectedHash string) error {
	lock := lockForFile(absPath)
	lock.Lock()
	defer lock.Unlock()
	currentHash := ""
	exists := true
	if strings.TrimSpace(expectedHash) != "" {
		var err error
		currentHash, exists, err = fileHashIfExists(absPath)
		if err != nil {
			return err
		}
		if !exists {
			currentHash = "<missing>"
		}
		if currentHash != expectedHash {
			return staleFileError{Path: cleanPatchPath(relPath), ExpectedHash: expectedHash, CurrentHash: currentHash}
		}
	}
	if !exists {
		return nil
	}
	return os.Remove(absPath)
}

func lineRangeString(offset *int, limit *int, truncated bool, next int) string {
	if offset == nil && limit == nil {
		return "all"
	}
	start := 1
	if offset != nil {
		start = *offset
	}
	count := readFileDefaultLineLimit
	if limit != nil {
		count = *limit
	}
	end := start + count - 1
	if truncated && next > 0 {
		end = next - 1
	}
	return strconv.Itoa(start) + "-" + strconv.Itoa(end)
}
