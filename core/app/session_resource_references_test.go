package app

import (
	"context"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"aivo/core/domain"
)

func TestNormalizeSessionResourceReferencesDeduplicatesAndRejectsInvalidInput(t *testing.T) {
	references, err := normalizeSessionResourceReferences([]domain.SessionResourceReference{
		{Kind: " skill ", ID: "skill-1"},
		{Kind: "skill", ID: "skill-1"},
		{Kind: "tool", ID: "read"},
	})
	if err != nil || len(references) != 2 || references[0].Kind != domain.SessionResourceSkill {
		t.Fatalf("references = %#v, err = %v", references, err)
	}
	if _, err := normalizeSessionResourceReferences([]domain.SessionResourceReference{{Kind: "unknown", ID: "x"}}); err == nil {
		t.Fatal("unknown reference kind was accepted")
	}
	if _, err := normalizeSessionResourceReferences([]domain.SessionResourceReference{
		{Kind: "project", ID: "one", RootPath: "/one"},
		{Kind: "project", ID: "two", RootPath: "/two"},
	}); err == nil {
		t.Fatal("multiple project references were accepted")
	}
	overLimit := make([]domain.SessionResourceReference, sessionResourceReferenceLimit+1)
	for index := range overLimit {
		overLimit[index] = domain.SessionResourceReference{Kind: "tool", ID: "tool"}
	}
	if _, err := normalizeSessionResourceReferences(overLimit); err == nil {
		t.Fatal("over-limit references were accepted")
	}
	if _, err := normalizeSessionResourceReferences([]domain.SessionResourceReference{{
		Kind: "tool", ID: strings.Repeat("x", sessionResourceReferenceIDLimit+1),
	}}); err == nil {
		t.Fatal("over-limit resource id was accepted")
	}
}

func TestSubmitSessionMessageBindsExactMentionedProjectAndRedactsPathFromEvent(t *testing.T) {
	service, cleanup := newSessionTestService(t)
	defer cleanup()
	ctx := context.Background()
	initial := t.TempDir()
	if _, err := service.CompleteInitialization(ctx, domain.CompleteInitializationInput{InitialWorkspacePath: initial}); err != nil {
		t.Fatal(err)
	}
	project, err := service.RegisterAgentProject(ctx, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	session, err := service.CreateRuntimeSession(ctx, domain.CreateSessionRequest{Type: domain.SessionTypeCoding, Title: "mention project"})
	if err != nil {
		t.Fatal(err)
	}
	run, err := service.SubmitSessionMessage(ctx, domain.SubmitSessionMessageRequest{
		SessionID: session.ID,
		Text:      "Use the selected project",
		ResourceReferences: []domain.SessionResourceReference{{
			Kind: domain.SessionResourceProject, ID: project.Project.ID, RootPath: project.Project.RootPath,
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	bound, err := service.GetRuntimeSession(ctx, session.ID)
	if err != nil || bound.ProjectPath != project.Project.RootPath {
		t.Fatalf("bound session = %#v, err = %v", bound, err)
	}
	if strings.Contains(string(mustJSONRaw(t, run.UserEvent.Payload)), project.Project.RootPath) {
		t.Fatalf("user event leaked project path: %#v", run.UserEvent.Payload)
	}
}

func TestSubmitSessionMessageActivatesMentionedSkillAndRejectsStaleReference(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	service, cleanup := newSkillTestService(t)
	defer cleanup()
	ctx := context.Background()
	writeSkill(t, filepath.Join(home, ".claude", "skills", "mention-skill"), "mention-skill", "Mention test", "Follow the mentioned workflow.")
	scan, err := service.ScanGlobalSkills(ctx)
	if err != nil || len(scan.Candidates) != 1 {
		t.Fatalf("scan = %#v, err = %v", scan, err)
	}
	skill, err := service.ImportSkill(ctx, domain.SkillImportInput{CandidateID: scan.Candidates[0].ID, TargetScope: domain.SkillScopeGlobal})
	if err != nil {
		t.Fatal(err)
	}
	session, err := service.CreateRuntimeSession(ctx, domain.CreateSessionRequest{Type: domain.SessionTypeCoding, ProjectPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.SubmitSessionMessage(ctx, domain.SubmitSessionMessageRequest{
		SessionID: session.ID,
		Text:      "Use the selected skill",
		ResourceReferences: []domain.SessionResourceReference{{
			Kind: domain.SessionResourceSkill, ID: skill.ID,
		}},
	}); err != nil {
		t.Fatal(err)
	}
	active, err := service.GetSessionActiveSkills(ctx, session.ID)
	if err != nil || len(active.SkillIDs) != 1 || active.SkillIDs[0] != skill.ID {
		t.Fatalf("active skills = %#v, err = %v", active, err)
	}

	staleSession, err := service.CreateRuntimeSession(ctx, domain.CreateSessionRequest{Type: domain.SessionTypeCoding, ProjectPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.SubmitSessionMessage(ctx, domain.SubmitSessionMessageRequest{
		SessionID: staleSession.ID,
		Text:      "Do not persist this",
		ResourceReferences: []domain.SessionResourceReference{{
			Kind: domain.SessionResourceSkill, ID: "missing-skill",
		}},
	}); err == nil {
		t.Fatal("stale skill reference was accepted")
	}
	events, err := service.ListEvents(ctx, staleSession.ID, false, 20)
	if err != nil {
		t.Fatal(err)
	}
	for _, event := range events {
		if event.Type == domain.EventTypeUserMessage {
			t.Fatalf("validation failure persisted a user event: %#v", events)
		}
	}
}

func TestSubmitSessionMessageActivatesMentionedSkillGroup(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	service, cleanup := newSkillTestService(t)
	defer cleanup()
	ctx := context.Background()
	writeSkill(t, filepath.Join(home, ".codex", "skills", "hyperframes"), "hyperframes", "Mandatory entry point for HyperFrames video work", "Read routing instructions first.")
	writeSkill(t, filepath.Join(home, ".codex", "skills", "hyperframes-animation"), "hyperframes-animation", "HyperFrames animation knowledge", "Follow animation instructions.")
	writeSkill(t, filepath.Join(home, ".codex", "skills", "hyperframes-audio"), "hyperframes-audio", "HyperFrames audio mixing", "Follow audio instructions.")
	scan, err := service.ScanGlobalSkills(ctx)
	if err != nil || len(scan.Candidates) != 3 {
		t.Fatalf("scan = %#v, err = %v", scan, err)
	}
	for _, candidate := range scan.Candidates {
		if _, err := service.ImportSkill(ctx, domain.SkillImportInput{CandidateID: candidate.ID, TargetScope: domain.SkillScopeGlobal}); err != nil {
			t.Fatal(err)
		}
	}
	session, err := service.CreateRuntimeSession(ctx, domain.CreateSessionRequest{Type: domain.SessionTypeCoding, ProjectPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	run, err := service.SubmitSessionMessage(ctx, domain.SubmitSessionMessageRequest{
		SessionID: session.ID,
		Text:      "Use the selected HyperFrames skills",
		ResourceReferences: []domain.SessionResourceReference{{
			Kind: domain.SessionResourceSkill, ID: "skill-group:hyperframes",
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	active, err := service.GetSessionActiveSkills(ctx, session.ID)
	if err != nil || len(active.SkillIDs) != 3 {
		t.Fatalf("active grouped skills = %#v, err = %v", active, err)
	}
	names := make([]string, 0, len(active.Skills))
	for _, skill := range active.Skills {
		names = append(names, skill.Name)
	}
	sort.Strings(names)
	if strings.Join(names, ",") != "hyperframes,hyperframes-animation,hyperframes-audio" {
		t.Fatalf("active grouped skill names = %#v", names)
	}
	payload := string(mustJSONRaw(t, run.UserEvent.Payload))
	if !strings.Contains(payload, `"id":"skill-group:hyperframes"`) || !strings.Contains(payload, `"name":"HyperFrames"`) {
		t.Fatalf("user event did not retain canonical group summary: %s", payload)
	}
}

func TestSubmitSessionMessageRejectsGloballyDisabledAndQueuedReferences(t *testing.T) {
	service, cleanup := newSessionTestService(t)
	defer cleanup()
	ctx := context.Background()
	workspace := t.TempDir()
	if _, err := service.SetGlobalToolEnabled(ctx, domain.GlobalToolEnabledInput{Name: "grep", Enabled: false, WorkspaceRoot: workspace}); err != nil {
		t.Fatal(err)
	}
	session, err := service.CreateRuntimeSession(ctx, domain.CreateSessionRequest{Type: domain.SessionTypeCoding, ProjectPath: workspace})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.SubmitSessionMessage(ctx, domain.SubmitSessionMessageRequest{
		SessionID: session.ID,
		Text:      "Use grep",
		ResourceReferences: []domain.SessionResourceReference{{
			Kind: domain.SessionResourceTool, ID: "grep",
		}},
	}); err == nil {
		t.Fatal("globally disabled tool reference was accepted")
	}
	if _, err := service.SubmitSessionMessage(ctx, domain.SubmitSessionMessageRequest{
		SessionID: session.ID,
		Text:      "Queue this",
		Delivery:  domain.InputDeliverySteer,
		ResourceReferences: []domain.SessionResourceReference{{
			Kind: domain.SessionResourceTool, ID: "read",
		}},
	}); err == nil || !strings.Contains(err.Error(), "immediate") {
		t.Fatalf("queued resource reference error = %v", err)
	}
	events, err := service.ListEvents(ctx, session.ID, false, 20)
	if err != nil {
		t.Fatal(err)
	}
	for _, event := range events {
		if event.Type == domain.EventTypeUserMessage {
			t.Fatalf("rejected references persisted a user event: %#v", events)
		}
	}
}

func TestSessionResourceSourceToolNamesKeepsOneSourceAndStableOrder(t *testing.T) {
	entries := map[string]domain.ToolCatalogEntry{
		"mcp_two": {Name: "mcp_two", Source: domain.ToolSourceMCP, SourceID: "server"},
		"ext_one": {Name: "ext_one", Source: domain.ToolSourceExtension, SourceID: "extension"},
		"mcp_one": {Name: "mcp_one", Source: domain.ToolSourceMCP, SourceID: "server"},
		"other":   {Name: "other", Source: domain.ToolSourceMCP, SourceID: "other"},
	}
	names := sessionResourceSourceToolNames(entries, domain.ToolSourceMCP, "server")
	if strings.Join(names, ",") != "mcp_one,mcp_two" {
		t.Fatalf("names = %#v", names)
	}
}

func TestRenderSessionResourceReferenceContextUsesCanonicalSafeMetadata(t *testing.T) {
	context := renderSessionResourceReferenceContext([]domain.SessionResourceReference{{
		Kind:     domain.SessionResourceExtension,
		ID:       `extension<&`,
		Name:     `Example "Extension"`,
		RootPath: "/private/path/that/must/not/appear",
	}})
	if !strings.Contains(context, `id="extension&lt;&amp;"`) || !strings.Contains(context, `name="Example &quot;Extension&quot;"`) {
		t.Fatalf("context was not escaped: %s", context)
	}
	if strings.Contains(context, "/private/path") {
		t.Fatalf("context leaked a local path: %s", context)
	}
}
