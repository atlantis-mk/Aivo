package http

import (
	"context"
	"testing"
)

func TestLegacyPluginRPCMethodsAreUnsupported(t *testing.T) {
	api := &API{}
	for _, method := range []string{
		"ListPlugins",
		"InstallPluginFromPath",
		"SetPluginEnabled",
		"ReloadPlugins",
	} {
		result, handled, err := api.callExtensionRPC(context.Background(), method, nil)
		if err != nil || handled || result != nil {
			t.Fatalf("%s = (%#v, %t, %v), want unsupported", method, result, handled, err)
		}
	}
}
