package domain

import (
	"strings"
	"testing"
)

func TestNormalizeAppName(t *testing.T) {
	value, err := NormalizeAppName("  小艾  ")
	if err != nil || value != "小艾" {
		t.Fatalf("NormalizeAppName() = %q, %v", value, err)
	}

	for _, invalid := range []string{"", "   ", "Aivo\nAgent", strings.Repeat("名", MaxAppNameRunes+1)} {
		if _, err := NormalizeAppName(invalid); err == nil {
			t.Fatalf("NormalizeAppName(%q) accepted invalid value", invalid)
		}
	}
}

func TestResolveInitializationAppNameDefaultsOnlyWhenOmitted(t *testing.T) {
	value, err := ResolveInitializationAppName(nil)
	if err != nil || value != DefaultAppName {
		t.Fatalf("ResolveInitializationAppName(nil) = %q, %v", value, err)
	}
	empty := ""
	if _, err := ResolveInitializationAppName(&empty); err == nil {
		t.Fatal("explicit empty app name was accepted")
	}
}
