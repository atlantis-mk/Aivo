package app

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestTerminalServiceCreateAttachReplayResizeAndRemove(t *testing.T) {
	service := NewTerminalService()
	defer service.Shutdown()
	ctx := context.Background()
	root := t.TempDir()
	info, err := service.Create(ctx, TerminalCreateInput{
		WorkspaceRoot: root,
		Shell:         "/bin/sh",
		Rows:          12,
		Cols:          40,
	})
	if err != nil {
		t.Skipf("pty unavailable: %v", err)
	}
	if info.Status != TerminalStatusRunning || info.PID == 0 {
		t.Fatalf("info = %#v, want running terminal", info)
	}
	attachment, err := service.Attach(ctx, TerminalAttachInput{WorkspaceRoot: root, TerminalID: info.ID, Cursor: 0})
	if err != nil {
		t.Fatal(err)
	}
	defer attachment.Detach()
	if err := attachment.Resize(20, 80); err != nil {
		t.Fatal(err)
	}
	if err := attachment.Write([]byte("printf hello\n")); err != nil {
		t.Fatal(err)
	}
	var output strings.Builder
	deadline := time.After(2 * time.Second)
	for !strings.Contains(output.String(), "hello") {
		select {
		case chunk := <-attachment.Data():
			output.Write(chunk)
		case <-deadline:
			t.Fatalf("terminal output = %q, want hello", output.String())
		}
	}
	info, err = service.Update(ctx, TerminalUpdateInput{WorkspaceRoot: root, TerminalID: info.ID, Title: "Renamed", Rows: 18, Cols: 90})
	if err != nil {
		t.Fatal(err)
	}
	if info.Title != "Renamed" || info.Rows != 18 || info.Cols != 90 {
		t.Fatalf("updated info = %#v", info)
	}
	later, err := service.Attach(ctx, TerminalAttachInput{WorkspaceRoot: root, TerminalID: info.ID, Cursor: 0})
	if err != nil {
		t.Fatal(err)
	}
	if replay := string(later.Replay()); !strings.Contains(replay, "hello") {
		t.Fatalf("replay = %q, want hello", replay)
	}
	later.Detach()
	if err := service.Remove(ctx, root, info.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Get(ctx, root, info.ID); err == nil {
		t.Fatal("removed terminal should not be returned")
	}
}

func TestShellSingleQuote(t *testing.T) {
	got := shellSingleQuote("/tmp/Aivo Workspaces/it's ok")
	want := "'/tmp/Aivo Workspaces/it'\\''s ok'"
	if got != want {
		t.Fatalf("shellSingleQuote = %q, want %q", got, want)
	}
}

func TestTerminalOutlivesCreateContext(t *testing.T) {
	service := NewTerminalService()
	defer service.Shutdown()
	root := t.TempDir()
	ctx, cancel := context.WithCancel(context.Background())
	info, err := service.Create(ctx, TerminalCreateInput{
		WorkspaceRoot: root,
		Shell:         "/bin/sh",
		Rows:          12,
		Cols:          40,
	})
	if err != nil {
		t.Skipf("pty unavailable: %v", err)
	}
	cancel()
	time.Sleep(50 * time.Millisecond)
	attachment, err := service.Attach(context.Background(), TerminalAttachInput{
		WorkspaceRoot: root,
		TerminalID:    info.ID,
		Cursor:        0,
	})
	if err != nil {
		t.Fatalf("terminal should outlive create context: %v", err)
	}
	defer attachment.Detach()
	if err := attachment.Write([]byte("printf alive\n")); err != nil {
		t.Fatal(err)
	}
	var output strings.Builder
	deadline := time.After(2 * time.Second)
	for !strings.Contains(output.String(), "alive") {
		select {
		case chunk := <-attachment.Data():
			output.Write(chunk)
		case <-deadline:
			t.Fatalf("terminal output = %q, want alive", output.String())
		}
	}
}

func TestTerminalServiceAllowsEmptyWorkspaceRoot(t *testing.T) {
	service := NewTerminalService()
	defer service.Shutdown()
	ctx := context.Background()
	info, err := service.Create(ctx, TerminalCreateInput{
		Shell: "/bin/sh",
		Rows:  12,
		Cols:  40,
	})
	if err != nil {
		t.Skipf("pty unavailable: %v", err)
	}
	if info.WorkspaceRoot != "" {
		t.Fatalf("WorkspaceRoot = %q, want empty", info.WorkspaceRoot)
	}
	if strings.TrimSpace(info.CWD) == "" {
		t.Fatal("CWD should be populated for empty workspace terminal")
	}
	list, err := service.List(ctx, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].ID != info.ID {
		t.Fatalf("list = %#v, want created terminal", list)
	}
	attachment, err := service.Attach(ctx, TerminalAttachInput{
		TerminalID: info.ID,
		Cursor:     0,
	})
	if err != nil {
		t.Fatal(err)
	}
	attachment.Detach()
	if err := service.Remove(ctx, "", info.ID); err != nil {
		t.Fatal(err)
	}
}
