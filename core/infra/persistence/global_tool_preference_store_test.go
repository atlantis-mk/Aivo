package persistence

import (
	"context"
	"path/filepath"
	"reflect"
	"testing"
)

func TestGlobalToolPreferencesPersistAndDefaultToEnabled(t *testing.T) {
	path := filepath.Join(t.TempDir(), "aivo.db")
	store, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if names, err := store.ListGloballyDisabledToolNames(ctx); err != nil || len(names) != 0 {
		t.Fatalf("initial disabled names = %#v, err = %v", names, err)
	}
	if err := store.SetGlobalToolEnabled(ctx, "write", false); err != nil {
		t.Fatal(err)
	}
	if err := store.SetGlobalToolEnabled(ctx, "read", false); err != nil {
		t.Fatal(err)
	}
	if err := store.SetGlobalToolEnabled(ctx, "write", false); err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}

	reopened, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	if names, err := reopened.ListGloballyDisabledToolNames(ctx); err != nil || !reflect.DeepEqual(names, []string{"read", "write"}) {
		t.Fatalf("persisted disabled names = %#v, err = %v", names, err)
	}
	if err := reopened.SetGlobalToolEnabled(ctx, "read", true); err != nil {
		t.Fatal(err)
	}
	if names, err := reopened.ListGloballyDisabledToolNames(ctx); err != nil || !reflect.DeepEqual(names, []string{"write"}) {
		t.Fatalf("re-enabled names = %#v, err = %v", names, err)
	}
}
