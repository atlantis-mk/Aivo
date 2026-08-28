package app

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"aivo/core/domain"
)

func TestCompleteInitializationCreatesAndPersistsInitialWorkspace(t *testing.T) {
	service, cleanup := newSessionTestService(t)
	defer cleanup()
	ctx := context.Background()
	workspace := filepath.Join(t.TempDir(), "nested", "workspace")

	appName := "  小艾  "
	cfg, err := service.CompleteInitialization(ctx, domain.CompleteInitializationInput{
		AppName:              &appName,
		InitialWorkspacePath: workspace,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.Initialized || cfg.AppName != "小艾" || cfg.InitialWorkspacePath != workspace {
		t.Fatalf("config = %#v", cfg)
	}
	if info, err := os.Stat(workspace); err != nil || !info.IsDir() {
		t.Fatalf("workspace was not created: %v", err)
	}

	loaded, err := service.AppConfig(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.AppName != "小艾" || loaded.InitialWorkspacePath != workspace {
		t.Fatalf("persisted config = %#v", loaded)
	}
}

func TestCompleteInitializationRejectsInvalidAppNameBeforeCreatingWorkspace(t *testing.T) {
	service, cleanup := newSessionTestService(t)
	defer cleanup()
	workspace := filepath.Join(t.TempDir(), "workspace")
	appName := "   "

	_, err := service.CompleteInitialization(context.Background(), domain.CompleteInitializationInput{
		AppName:              &appName,
		InitialWorkspacePath: workspace,
	})
	if err == nil || !strings.Contains(err.Error(), "app name is required") {
		t.Fatalf("error = %v", err)
	}
	if _, err := os.Stat(workspace); !os.IsNotExist(err) {
		t.Fatalf("workspace was created before name validation: %v", err)
	}
}

func TestCompleteInitializationUsesDefaultInitialWorkspace(t *testing.T) {
	service, cleanup := newSessionTestService(t)
	defer cleanup()
	ctx := context.Background()
	workspace := filepath.Join(t.TempDir(), "default-workspace")
	t.Setenv(managedWorkspaceRootEnv, workspace)

	before, err := service.AppConfig(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if before.InitialWorkspacePath != "" || before.DefaultInitialWorkspacePath != workspace {
		t.Fatalf("config before initialization = %#v", before)
	}
	if _, err := os.Stat(workspace); !os.IsNotExist(err) {
		t.Fatalf("default workspace was created before confirmation: %v", err)
	}

	after, err := service.CompleteInitialization(ctx, domain.CompleteInitializationInput{})
	if err != nil {
		t.Fatal(err)
	}
	if !after.Initialized || after.AppName != domain.DefaultAppName || after.InitialWorkspacePath != workspace {
		t.Fatalf("config after initialization = %#v", after)
	}
	if info, err := os.Stat(workspace); err != nil || !info.IsDir() {
		t.Fatalf("default workspace was not created: %v", err)
	}
}

func TestManagedWorkspaceRootUsesHyphenatedDirectoryName(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv(managedWorkspaceRootEnv, "")

	workspace, err := managedWorkspaceRoot()
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(home, "Documents", "Aivo-Workspaces")
	if workspace != want {
		t.Fatalf("managed workspace root = %q, want %q", workspace, want)
	}
	if strings.Contains(filepath.Base(workspace), " ") {
		t.Fatalf("managed workspace directory contains spaces: %q", workspace)
	}
}

func TestIsManagedWorkspaceRecognizesCurrentAndLegacyRoots(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv(managedWorkspaceRootEnv, "")

	for _, workspace := range []string{
		filepath.Join(home, "Documents", "Aivo-Workspaces", "current"),
		filepath.Join(home, "Documents", "Aivo Workspaces", "legacy"),
	} {
		if !isManagedWorkspace(workspace) {
			t.Fatalf("managed workspace was not recognized: %q", workspace)
		}
	}
	if isManagedWorkspace(filepath.Join(home, "Documents", "Aivo Project")) {
		t.Fatal("ordinary project was recognized as a managed workspace")
	}
}

func TestCompleteInitializationRejectsNonDirectoryWorkspace(t *testing.T) {
	service, cleanup := newSessionTestService(t)
	defer cleanup()
	path := filepath.Join(t.TempDir(), "workspace-file")
	if err := os.WriteFile(path, []byte("not a directory"), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := service.CompleteInitialization(context.Background(), domain.CompleteInitializationInput{
		InitialWorkspacePath: path,
	})
	if err == nil || !strings.Contains(err.Error(), "not a directory") {
		t.Fatalf("error = %v", err)
	}
}

func TestUnscopedCodingSessionsShareInitialWorkspaceWithoutChildDirectories(t *testing.T) {
	service, cleanup := newSessionTestService(t)
	defer cleanup()
	ctx := context.Background()
	workspace := t.TempDir()
	if _, err := service.CompleteInitialization(ctx, domain.CompleteInitializationInput{InitialWorkspacePath: workspace}); err != nil {
		t.Fatal(err)
	}

	for index := 0; index < 2; index++ {
		session, err := service.CreateRuntimeSession(ctx, domain.CreateSessionRequest{Type: domain.SessionTypeCoding})
		if err != nil {
			t.Fatal(err)
		}
		if session.ProjectPath != "" {
			t.Fatalf("unscoped session project path = %q", session.ProjectPath)
		}
		context, err := service.GetCodingContext(ctx, session.ID)
		if err != nil {
			t.Fatal(err)
		}
		if context.ProjectPath != workspace {
			t.Fatalf("context workspace = %q, want %q", context.ProjectPath, workspace)
		}
	}
	entries, err := os.ReadDir(workspace)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("unexpected per-session directories: %#v", entries)
	}
}

func TestUnscopedCodingSessionRequiresInitialWorkspace(t *testing.T) {
	service, cleanup := newSessionTestService(t)
	defer cleanup()
	_, err := service.CreateRuntimeSession(context.Background(), domain.CreateSessionRequest{Type: domain.SessionTypeCoding})
	if err == nil || !strings.Contains(err.Error(), "complete setup") {
		t.Fatalf("error = %v", err)
	}
}

func TestExplicitProjectTakesPrecedenceOverInitialWorkspace(t *testing.T) {
	service, cleanup := newSessionTestService(t)
	defer cleanup()
	ctx := context.Background()
	workspace := t.TempDir()
	project := t.TempDir()
	if _, err := service.CompleteInitialization(ctx, domain.CompleteInitializationInput{InitialWorkspacePath: workspace}); err != nil {
		t.Fatal(err)
	}
	session, err := service.CreateRuntimeSession(ctx, domain.CreateSessionRequest{Type: domain.SessionTypeCoding, ProjectPath: project})
	if err != nil {
		t.Fatal(err)
	}
	if session.ProjectPath != project {
		t.Fatalf("session project = %q, want %q", session.ProjectPath, project)
	}
}
