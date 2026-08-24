package app

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"aivo/core/domain"
)

func TestAskUserToolWaitsForReply(t *testing.T) {
	service, cleanup := newSessionTestService(t)
	defer cleanup()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	args := json.RawMessage(`{"questions":[{"header":"Scope","question":"How should I proceed?","options":[{"label":"Continue (Recommended)","description":"Use the default path."},{"label":"Stop","description":"Do not continue."}]}]}`)
	resultCh := make(chan domain.ToolResult, 1)
	go func() {
		resultCh <- service.askUserTool(ctx, args, domain.ToolExecutionContext{
			SessionID:  "session-question",
			TurnID:     "turn-question",
			ToolCallID: "call-question",
		})
	}()

	var request domain.QuestionRequest
	for deadline := time.Now().Add(time.Second); time.Now().Before(deadline); {
		requests, err := service.ListQuestionRequests(ctx, "session-question", domain.QuestionRequestStatusPending)
		if err != nil {
			t.Fatal(err)
		}
		if len(requests) > 0 {
			request = requests[0]
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if request.ID == "" {
		t.Fatal("question request was not created")
	}
	if request.ToolCallID != "call-question" || len(request.Questions) != 1 {
		t.Fatalf("request = %#v", request)
	}
	if request.ToolName != "ask_user" {
		t.Fatalf("request.ToolName = %q, want ask_user", request.ToolName)
	}
	if _, err := service.ReplyQuestionRequest(ctx, domain.ReplyQuestionRequestInput{
		RequestID: request.ID,
		Answers:   [][]string{{"Continue (Recommended)"}},
	}); err != nil {
		t.Fatal(err)
	}

	select {
	case result := <-resultCh:
		if !result.OK {
			t.Fatalf("result failed: %#v", result)
		}
		if result.Name != "ask_user" {
			t.Fatalf("result.Name = %q, want ask_user", result.Name)
		}
		if !strings.Contains(result.ModelContent, "Continue (Recommended)") {
			t.Fatalf("model content = %q", result.ModelContent)
		}
	case <-ctx.Done():
		t.Fatal("question tool did not return after reply")
	}
}
