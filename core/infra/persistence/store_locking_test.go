package persistence

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"aivo/core/domain"
)

func TestStoreWaitsForConcurrentSQLiteWriter(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "aivo.db")
	first, err := Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer first.Close()
	second, err := Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer second.Close()

	transaction := first.db.Begin()
	if transaction.Error != nil {
		t.Fatal(transaction.Error)
	}
	if err := transaction.Create(&providerRow{
		ID:        "writer-holding-lock",
		Type:      "openai",
		Status:    "ready",
		UpdatedAt: domain.NowString(time.Now()),
	}).Error; err != nil {
		_ = transaction.Rollback().Error
		t.Fatal(err)
	}

	finished := make(chan error, 1)
	go func() {
		finished <- second.SaveProvider(ctx, domain.ProviderConfig{
			ID:    "writer-waiting-for-lock",
			Type:  "openai",
			Model: "gpt-5.5",
		})
	}()

	select {
	case err := <-finished:
		t.Fatalf("concurrent write returned before the lock was released: %v", err)
	case <-time.After(100 * time.Millisecond):
	}
	if err := transaction.Commit().Error; err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-finished:
		if err != nil {
			t.Fatalf("concurrent write after lock release: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("concurrent write did not resume after the lock was released")
	}
}
