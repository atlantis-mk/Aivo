package domain

import (
	"errors"
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	DefaultAppName  = "Aivo"
	MaxAppNameRunes = 40
)

func NormalizeAppName(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", errors.New("app name is required")
	}
	if utf8.RuneCountInString(value) > MaxAppNameRunes {
		return "", errors.New("app name must be 40 characters or fewer")
	}
	for _, r := range value {
		if unicode.IsControl(r) {
			return "", errors.New("app name cannot contain control characters")
		}
	}
	return value, nil
}

func ResolveInitializationAppName(value *string) (string, error) {
	if value == nil {
		return DefaultAppName, nil
	}
	return NormalizeAppName(*value)
}
