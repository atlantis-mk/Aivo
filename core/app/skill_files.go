package app

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"strings"

	"aivo/core/domain"
)

func copyDirectory(src string, dest string) error {
	if err := os.RemoveAll(dest); err != nil {
		return err
	}
	return filepath.WalkDir(src, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.Type()&os.ModeSymlink != 0 {
			return &os.PathError{Op: "copy", Path: path, Err: os.ErrInvalid}
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dest, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		if d.Type() != 0 {
			return &os.PathError{Op: "copy", Path: path, Err: os.ErrInvalid}
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		in, err := os.Open(path)
		if err != nil {
			return err
		}
		defer in.Close()
		out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
		if err != nil {
			return err
		}
		defer out.Close()
		_, err = io.Copy(out, in)
		return err
	})
}

func normalizeSkillName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

func normalizeSkillScope(scope string) string {
	switch strings.TrimSpace(scope) {
	case domain.SkillScopeProject:
		return domain.SkillScopeProject
	default:
		return domain.SkillScopeGlobal
	}
}

func candidateIDForPath(path string) string {
	sum := sha256.Sum256([]byte(filepath.Clean(path)))
	return "skill-candidate-" + hex.EncodeToString(sum[:12])
}

func sourceIDForPath(skillID string, path string) string {
	sum := sha256.Sum256([]byte(skillID + "\x00" + filepath.Clean(path)))
	return "skill-source-" + hex.EncodeToString(sum[:12])
}

func samePath(a string, b string) bool {
	aa, errA := filepath.Abs(a)
	bb, errB := filepath.Abs(b)
	if errA == nil {
		a = aa
	}
	if errB == nil {
		b = bb
	}
	return filepath.Clean(a) == filepath.Clean(b)
}

func skillPathWithin(path string, root string) bool {
	absPath, errPath := filepath.Abs(path)
	absRoot, errRoot := filepath.Abs(root)
	if errPath != nil || errRoot != nil {
		return false
	}
	rel, err := filepath.Rel(absRoot, absPath)
	return err == nil && rel != "." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && rel != ".."
}
