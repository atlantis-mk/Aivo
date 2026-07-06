package app

import (
	"encoding/json"
	"strings"

	"aivo/core/domain"
)

func applyPatchDraftFiles(args json.RawMessage, workspaceRoot string) (string, []domain.ToolResultFile) {
	patchText := patchTextFromToolArguments(args)
	if strings.TrimSpace(patchText) == "" {
		return "", nil
	}
	return patchText, fillToolResultFileFullPaths(parseApplyPatchDraftFiles(patchText), workspaceRoot)
}

func patchTextFromToolArguments(args json.RawMessage) string {
	raw := strings.TrimSpace(string(args))
	if raw == "" {
		return ""
	}
	if strings.HasPrefix(raw, "*** Begin Patch") {
		return raw
	}
	var parsed struct {
		PatchText string `json:"patchText"`
	}
	if err := json.Unmarshal(args, &parsed); err == nil && parsed.PatchText != "" {
		return parsed.PatchText
	}
	return partialJSONStringField(raw, "patchText")
}

func partialJSONStringField(raw string, field string) string {
	key := `"` + field + `"`
	keyIndex := strings.Index(raw, key)
	if keyIndex < 0 {
		return ""
	}
	rest := raw[keyIndex+len(key):]
	colonIndex := strings.IndexByte(rest, ':')
	if colonIndex < 0 {
		return ""
	}
	rest = strings.TrimLeft(rest[colonIndex+1:], " \t\r\n")
	if rest == "" || rest[0] != '"' {
		return ""
	}
	return partialJSONStringValue(rest[1:])
}

func partialJSONStringValue(raw string) string {
	var out strings.Builder
	escaped := false
	for i := 0; i < len(raw); i++ {
		ch := raw[i]
		if escaped {
			switch ch {
			case '"', '\\', '/':
				out.WriteByte(ch)
			case 'n':
				out.WriteByte('\n')
			case 'r':
				out.WriteByte('\r')
			case 't':
				out.WriteByte('\t')
			case 'b':
				out.WriteByte('\b')
			case 'f':
				out.WriteByte('\f')
			case 'u':
				if i+4 < len(raw) {
					out.WriteString(`\u`)
					out.WriteString(raw[i+1 : i+5])
					i += 4
				}
			default:
				out.WriteByte(ch)
			}
			escaped = false
			continue
		}
		switch ch {
		case '\\':
			escaped = true
		case '"':
			return out.String()
		default:
			out.WriteByte(ch)
		}
	}
	return out.String()
}

func parseApplyPatchDraftFiles(patchText string) []domain.ToolResultFile {
	lines := strings.Split(strings.ReplaceAll(strings.ReplaceAll(patchText, "\r\n", "\n"), "\r", "\n"), "\n")
	files := make([]domain.ToolResultFile, 0)
	current := -1
	for _, line := range lines {
		switch {
		case strings.HasPrefix(line, "*** Add File: "):
			files = append(files, domain.ToolResultFile{Path: cleanPatchPath(strings.TrimPrefix(line, "*** Add File: ")), Type: "add"})
			current = len(files) - 1
		case strings.HasPrefix(line, "*** Update File: "):
			files = append(files, domain.ToolResultFile{Path: cleanPatchPath(strings.TrimPrefix(line, "*** Update File: ")), Type: "update"})
			current = len(files) - 1
		case strings.HasPrefix(line, "*** Delete File: "):
			files = append(files, domain.ToolResultFile{Path: cleanPatchPath(strings.TrimPrefix(line, "*** Delete File: ")), Type: "delete"})
			current = len(files) - 1
		case strings.HasPrefix(line, "*** Move to: "):
			if current >= 0 {
				files[current].Type = "move"
				files[current].MovePath = cleanPatchPath(strings.TrimPrefix(line, "*** Move to: "))
			}
		default:
			if current < 0 {
				continue
			}
			if strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++") {
				files[current].Additions++
			}
			if strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "---") {
				files[current].Deletions++
			}
		}
	}
	return files
}

func fillToolResultFileFullPaths(files []domain.ToolResultFile, workspaceRoot string) []domain.ToolResultFile {
	if strings.TrimSpace(workspaceRoot) == "" || len(files) == 0 {
		return files
	}
	out := make([]domain.ToolResultFile, len(files))
	copy(out, files)
	for i := range out {
		out[i].FullPath = fullWorkspacePath(workspaceRoot, out[i].Path)
		out[i].MoveFullPath = fullWorkspacePath(workspaceRoot, out[i].MovePath)
	}
	return out
}
