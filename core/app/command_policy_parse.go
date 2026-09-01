package app

import (
	"errors"
	"strings"
)

func normalizeCommandText(value string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
}

func shellTokenize(command string) ([]string, error) {
	var tokens []string
	var current strings.Builder
	var quote rune
	escaped := false
	for _, r := range command {
		if escaped {
			current.WriteRune(r)
			escaped = false
			continue
		}
		if r == '\\' {
			escaped = true
			continue
		}
		if quote != 0 {
			if r == quote {
				quote = 0
				continue
			}
			current.WriteRune(r)
			continue
		}
		if r == '\'' || r == '"' {
			quote = r
			continue
		}
		if r == ' ' || r == '\t' || r == '\n' || r == '\r' {
			if current.Len() > 0 {
				tokens = append(tokens, current.String())
				current.Reset()
			}
			continue
		}
		current.WriteRune(r)
	}
	if escaped {
		current.WriteRune('\\')
	}
	if quote != 0 {
		return nil, errors.New("unterminated shell quote")
	}
	if current.Len() > 0 {
		tokens = append(tokens, current.String())
	}
	return tokens, nil
}

func commandHasMetacharacters(command string) bool {
	for _, marker := range []string{"|", ">", "<", "$(", "`", "&&", "||", ";", "&"} {
		if strings.Contains(command, marker) {
			return true
		}
	}
	return false
}

func shellHeredocTokenIndex(tokens []string) int {
	for index, token := range tokens {
		token = strings.TrimSpace(token)
		if strings.HasPrefix(token, "<<") {
			return index
		}
	}
	return -1
}
