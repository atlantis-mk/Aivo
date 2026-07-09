package http

import (
	"net/http"

	coreapp "aivo/core/app"
	"aivo/core/domain"
	"aivo/core/infra/persistence"
)

type API struct {
	store           *persistence.Store
	service         *coreapp.Service
	events          *eventBroker
	terminalTickets *terminalTicketStore
}

type routerWithShutdown struct {
	handler http.Handler
	api     *API
}

func (r *routerWithShutdown) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.handler.ServeHTTP(w, req)
}

func (r *routerWithShutdown) Shutdown() {
	if r != nil && r.api != nil && r.api.service != nil {
		r.api.service.Shutdown()
	}
}

func newAPI() (*API, error) {
	store, err := persistence.OpenDefault()
	if err != nil {
		return nil, err
	}
	api := &API{
		store:           store,
		service:         coreapp.NewService(store),
		events:          newEventBroker(),
		terminalTickets: newTerminalTicketStore(),
	}
	api.service.SetProviderAuthUpdatedHook(func(status domain.ProviderAuthStatus) {
		api.events.emit("provider_auth.updated", map[string]any{"providerId": status.ProviderID, "status": status})
	})
	api.service.SetSessionUpdatedHook(func(sessionID string, session *domain.Session) {
		payload := map[string]any{"sessionId": sessionID}
		if session != nil {
			payload["session"] = *session
		}
		api.events.emit("session.updated", payload)
	})
	api.service.SetTurnUpdatedHook(func(sessionID string, turn domain.Turn) {
		name := "turn.started"
		switch turn.Status {
		case domain.TurnStatusCompleted:
			name = "turn.completed"
		case domain.TurnStatusFailed:
			name = "turn.failed"
		case domain.TurnStatusCancelled:
			name = "turn.cancelled"
		}
		api.events.emit(name, map[string]any{"sessionId": sessionID, "turn": turn})
	})
	api.service.SetAssistantDeltaHook(func(sessionID string, turnID string, delta string) {
		api.events.emit("assistant.delta", map[string]string{"sessionId": sessionID, "turnId": turnID, "delta": delta})
	})
	api.service.SetToolCallUpdatedHook(func(sessionID string, turnID string, call domain.ToolCall, created bool) {
		name := "tool_call.updated"
		if created {
			name = "tool_call.created"
		}
		api.events.emit(name, map[string]any{"sessionId": sessionID, "turnId": turnID, "toolCall": call})
	})
	api.service.SetShellOutputHook(func(event coreapp.ShellOutputEvent) {
		api.events.emit("shell.output", event)
	})
	api.service.SetTodoItemsUpdatedHook(func(sessionID string, projectPath string, items []domain.TodoItem) {
		api.events.emit("todo_items.updated", map[string]any{"sessionId": sessionID, "projectPath": projectPath, "items": items})
	})
	api.service.SetPermissionRequestedHook(func(request domain.PermissionRequest) {
		api.events.emit("permission.requested", map[string]any{"sessionId": request.SessionID, "turnId": request.TurnID, "permission": request})
	})
	api.service.SetPermissionResolvedHook(func(request domain.PermissionRequest) {
		api.events.emit("permission.resolved", map[string]any{"sessionId": request.SessionID, "turnId": request.TurnID, "permission": request})
	})
	api.service.SetQuestionRequestedHook(func(request domain.QuestionRequest) {
		api.events.emit("question.requested", map[string]any{"sessionId": request.SessionID, "turnId": request.TurnID, "question": request})
	})
	api.service.SetQuestionResolvedHook(func(request domain.QuestionRequest) {
		api.events.emit("question.resolved", map[string]any{"sessionId": request.SessionID, "turnId": request.TurnID, "question": request})
	})
	api.service.SetTerminalEventHook(func(name string, info coreapp.TerminalInfo) {
		api.events.emit(name, map[string]any{"workspaceRoot": info.WorkspaceRoot, "terminal": info})
	})
	return api, nil
}
