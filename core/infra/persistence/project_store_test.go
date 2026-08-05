package persistence

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"aivo/core/domain"
)

func TestProjectRegistrationQueryPaginationAndRestore(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "aivo.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	ctx := context.Background()

	alphaPath := filepath.Join(t.TempDir(), "AlphaProject")
	betaPath := filepath.Join(t.TempDir(), "beta-project")
	gammaPath := filepath.Join(t.TempDir(), "gamma-project")
	for _, root := range []string{alphaPath, betaPath, gammaPath} {
		if err := mkdirProjectFixture(root); err != nil {
			t.Fatal(err)
		}
	}

	alpha, err := store.RegisterProject(ctx, alphaPath)
	if err != nil || alpha.Status != domain.ProjectRegistrationCreated {
		t.Fatalf("alpha registration = %#v, err = %v", alpha, err)
	}
	repeated, err := store.RegisterProject(ctx, alphaPath)
	if err != nil || repeated.Status != domain.ProjectRegistrationExisting || repeated.Project.ID != alpha.Project.ID {
		t.Fatalf("repeated registration = %#v, err = %v", repeated, err)
	}
	beta, err := store.RegisterProject(ctx, betaPath)
	if err != nil {
		t.Fatal(err)
	}
	gamma, err := store.RegisterProject(ctx, gammaPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.UpdateProjectDescription(ctx, betaPath, "Local API Client"); err != nil {
		t.Fatal(err)
	}
	if err := store.db.WithContext(ctx).Model(&projectRow{}).Where("id = ?", alpha.Project.ID).Update("time_updated", "2026-08-04T00:00:01Z").Error; err != nil {
		t.Fatal(err)
	}
	if err := store.db.WithContext(ctx).Model(&projectRow{}).Where("id = ?", beta.Project.ID).Update("time_updated", "2026-08-04T00:00:02Z").Error; err != nil {
		t.Fatal(err)
	}
	if err := store.db.WithContext(ctx).Model(&projectRow{}).Where("id = ?", gamma.Project.ID).Update("time_updated", "2026-08-04T00:00:03Z").Error; err != nil {
		t.Fatal(err)
	}

	search, err := store.QueryProjects(ctx, domain.ProjectQueryInput{Query: "api CLIENT", Limit: 20})
	if err != nil || len(search.Projects) != 1 || search.Projects[0].ID != beta.Project.ID {
		t.Fatalf("description search = %#v, err = %v", search, err)
	}
	page1, err := store.QueryProjects(ctx, domain.ProjectQueryInput{Limit: 2})
	if err != nil || len(page1.Projects) != 2 || page1.NextCursor == "" {
		t.Fatalf("first page = %#v, err = %v", page1, err)
	}
	page2, err := store.QueryProjects(ctx, domain.ProjectQueryInput{Limit: 2, Cursor: page1.NextCursor})
	if err != nil || len(page2.Projects) != 1 || page2.NextCursor != "" {
		t.Fatalf("second page = %#v, err = %v", page2, err)
	}
	seen := map[string]bool{}
	for _, project := range append(page1.Projects, page2.Projects...) {
		if seen[project.ID] {
			t.Fatalf("project %q repeated across cursor pages", project.ID)
		}
		seen[project.ID] = true
	}
	if _, err := store.QueryProjects(ctx, domain.ProjectQueryInput{Cursor: "not-a-cursor"}); err == nil {
		t.Fatal("invalid cursor was accepted")
	}

	if _, err := store.SetProjectSidebarHidden(ctx, alphaPath, true); err != nil {
		t.Fatal(err)
	}
	visible, err := store.QueryProjects(ctx, domain.ProjectQueryInput{Limit: 20})
	if err != nil {
		t.Fatal(err)
	}
	for _, project := range visible.Projects {
		if project.ID == alpha.Project.ID {
			t.Fatal("hidden project appeared in ordinary query")
		}
	}
	restored, err := store.RegisterProject(ctx, alphaPath)
	if err != nil || restored.Status != domain.ProjectRegistrationRestored || restored.Project.SidebarHidden {
		t.Fatalf("restored registration = %#v, err = %v", restored, err)
	}
}

func TestBindSessionProjectIsAtomicAndConcurrent(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "aivo.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	ctx := context.Background()
	firstPath := t.TempDir()
	secondPath := t.TempDir()
	first, err := store.RegisterProject(ctx, firstPath)
	if err != nil {
		t.Fatal(err)
	}
	second, err := store.RegisterProject(ctx, secondPath)
	if err != nil {
		t.Fatal(err)
	}
	session, err := store.CreateRuntimeSession(ctx, domain.CreateSessionRequest{Type: domain.SessionTypeCoding, Title: "unscoped"})
	if err != nil {
		t.Fatal(err)
	}

	type outcome struct {
		project domain.AssistantProject
		result  domain.SessionProjectBindingResult
		err     error
	}
	start := make(chan struct{})
	results := make(chan outcome, 2)
	var wg sync.WaitGroup
	for _, project := range []domain.AssistantProject{first.Project, second.Project} {
		project := project
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			result, bindErr := store.BindSessionProject(ctx, session.ID, project.ID, domain.CodingContext{
				SessionID: session.ID, ProjectPath: project.RootPath, CWD: project.RootPath,
			})
			results <- outcome{project: project, result: result, err: bindErr}
		}()
	}
	close(start)
	wg.Wait()
	close(results)

	changed := 0
	conflicts := 0
	var winner domain.AssistantProject
	for result := range results {
		if result.err != nil {
			t.Fatal(result.err)
		}
		if result.result.Changed {
			changed++
			winner = result.project
		}
		if result.result.Conflict {
			conflicts++
		}
	}
	if changed != 1 || conflicts != 1 {
		t.Fatalf("concurrent results changed=%d conflicts=%d", changed, conflicts)
	}
	savedSession, err := store.GetRuntimeSession(ctx, session.ID)
	if err != nil {
		t.Fatal(err)
	}
	savedContext, err := store.GetCodingContext(ctx, session.ID)
	if err != nil {
		t.Fatal(err)
	}
	if savedSession.ProjectPath != winner.RootPath || savedContext.ProjectPath != winner.RootPath || savedContext.CWD != winner.RootPath {
		t.Fatalf("session/context did not commit one winner: session=%#v context=%#v winner=%#v", savedSession, savedContext, winner)
	}
	retry, err := store.BindSessionProject(ctx, session.ID, winner.ID, savedContext)
	if err != nil || retry.Changed || retry.Conflict {
		t.Fatalf("same-project retry = %#v, err = %v", retry, err)
	}
}

func TestBindSessionProjectRollsBackWhenCodingContextWriteFails(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "aivo.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	ctx := context.Background()
	project, err := store.RegisterProject(ctx, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	session, err := store.CreateRuntimeSession(ctx, domain.CreateSessionRequest{Type: domain.SessionTypeCoding, Title: "rollback"})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.db.Exec(`CREATE TRIGGER reject_project_context BEFORE INSERT ON coding_contexts BEGIN SELECT RAISE(ABORT, 'forced coding context failure'); END`).Error; err != nil {
		t.Fatal(err)
	}
	_, err = store.BindSessionProject(ctx, session.ID, project.Project.ID, domain.CodingContext{
		SessionID: session.ID, ProjectPath: project.Project.RootPath, CWD: project.Project.RootPath,
	})
	if err == nil {
		t.Fatal("injected coding-context failure did not fail binding")
	}
	saved, err := store.GetRuntimeSession(ctx, session.ID)
	if err != nil {
		t.Fatal(err)
	}
	if saved.ProjectPath != "" {
		t.Fatalf("session project escaped rolled-back transaction: %#v", saved)
	}
	if _, err := store.GetCodingContext(ctx, session.ID); err == nil {
		t.Fatal("coding context unexpectedly persisted after rollback")
	}
}

func mkdirProjectFixture(path string) error {
	return os.MkdirAll(path, 0o755)
}
