package app

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

func normalizeSandboxCWD(workspaceRoot string, cwd string, allowExternal bool) (string, string, error) {
	root := strings.TrimSpace(workspaceRoot)
	if root == "" {
		return "", "", errors.New("workspace root is required")
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return "", "", err
	}
	realRoot, err := filepath.EvalSymlinks(absRoot)
	if err != nil {
		return "", "", err
	}
	cleanCWD := strings.TrimSpace(cwd)
	if cleanCWD == "" || cleanCWD == "." {
		return realRoot, realRoot, nil
	}
	var target string
	if filepath.IsAbs(cleanCWD) {
		target = cleanCWD
	} else {
		cleanRel := filepath.Clean(cleanCWD)
		if cleanRel == ".." || strings.HasPrefix(cleanRel, ".."+string(os.PathSeparator)) {
			return "", "", errors.New("cwd escapes workspace root")
		}
		target = filepath.Join(realRoot, cleanRel)
	}
	absTarget, err := filepath.Abs(target)
	if err != nil {
		return "", "", err
	}
	realTarget, err := filepath.EvalSymlinks(absTarget)
	if err != nil {
		return "", "", err
	}
	rel, err := filepath.Rel(realRoot, realTarget)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) || filepath.IsAbs(rel) {
		if allowExternal {
			return realRoot, realTarget, nil
		}
		return "", "", errors.New("cwd escapes workspace root")
	}
	return realRoot, realTarget, nil
}

func cwdIsExternal(workspaceRoot string, cwd string) (bool, string, error) {
	root, target, err := normalizeSandboxCWD(workspaceRoot, cwd, true)
	if err != nil {
		return false, "", err
	}
	rel, err := filepath.Rel(root, target)
	if err != nil {
		return true, target, nil
	}
	return rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) || filepath.IsAbs(rel), target, nil
}

func pathHasPrefix(path string, root string) bool {
	root = strings.TrimSpace(root)
	if root == "" {
		return false
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(absRoot, path)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator)) && !filepath.IsAbs(rel)
}
