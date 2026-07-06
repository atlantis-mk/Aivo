package http

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"

	coreapp "aivo/core/app"
	"aivo/core/domain"
	"aivo/core/infra/persistence"
)

type healthResponse struct {
	Status string `json:"status"`
	Name   string `json:"name"`
}

type rpcRequest struct {
	Args []json.RawMessage `json:"args"`
}

type rpcError struct {
	Error string `json:"error"`
}

type eventPayload struct {
	ID   uint64      `json:"id"`
	Name string      `json:"name"`
	Data interface{} `json:"data"`
}

type eventBroker struct {
	mu      sync.Mutex
	nextID  atomic.Uint64
	clients map[chan eventPayload]struct{}
}

func newEventBroker() *eventBroker {
	return &eventBroker{clients: map[chan eventPayload]struct{}{}}
}

func (b *eventBroker) subscribe() (chan eventPayload, func()) {
	ch := make(chan eventPayload, 32)
	b.mu.Lock()
	b.clients[ch] = struct{}{}
	b.mu.Unlock()
	return ch, func() {
		b.mu.Lock()
		delete(b.clients, ch)
		close(ch)
		b.mu.Unlock()
	}
}

func (b *eventBroker) emit(name string, data interface{}) {
	b.mu.Lock()
	defer b.mu.Unlock()
	payload := eventPayload{ID: b.nextID.Add(1), Name: name, Data: data}
	for ch := range b.clients {
		select {
		case ch <- payload:
		default:
		}
	}
}

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

type terminalTicket struct {
	Token         string
	WorkspaceRoot string
	TerminalID    string
	ExpiresAt     time.Time
}

type terminalTicketStore struct {
	mu      sync.Mutex
	tickets map[string]terminalTicket
}

func newTerminalTicketStore() *terminalTicketStore {
	return &terminalTicketStore{tickets: map[string]terminalTicket{}}
}

func (s *terminalTicketStore) create(workspaceRoot string, terminalID string) (terminalTicket, error) {
	raw := make([]byte, 24)
	if _, err := rand.Read(raw); err != nil {
		return terminalTicket{}, err
	}
	ticket := terminalTicket{Token: hex.EncodeToString(raw), WorkspaceRoot: workspaceRoot, TerminalID: terminalID, ExpiresAt: time.Now().Add(60 * time.Second)}
	s.mu.Lock()
	s.tickets[ticket.Token] = ticket
	s.mu.Unlock()
	return ticket, nil
}

func (s *terminalTicketStore) consume(token string, workspaceRoot string, terminalID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	ticket, ok := s.tickets[token]
	if !ok {
		return false
	}
	delete(s.tickets, token)
	return time.Now().Before(ticket.ExpiresAt) && ticket.WorkspaceRoot == workspaceRoot && ticket.TerminalID == terminalID
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

func NewRouter() http.Handler {
	api, err := newAPI()
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", handleHealth)
	if err != nil {
		mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
			writeError(w, http.StatusInternalServerError, err)
		})
		return withCORS(mux)
	}
	mux.HandleFunc("POST /api/rpc/{method}", api.handleRPC)
	mux.HandleFunc("GET /api/events", api.handleEvents)
	mux.HandleFunc("POST /api/terminals/{id}/connect-token", api.handleTerminalConnectToken)
	mux.HandleFunc("GET /api/terminals/{id}/connect", api.handleTerminalConnect)
	return &routerWithShutdown{handler: withCORS(mux), api: api}
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Aivo-Terminal-CSRF")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, healthResponse{
		Status: "ok",
		Name:   "aivo-core",
	})
}

func (api *API) handleEvents(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, errors.New("streaming is unavailable"))
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	ch, unsubscribe := api.events.subscribe()
	defer unsubscribe()
	for {
		select {
		case <-r.Context().Done():
			return
		case event := <-ch:
			data, _ := json.Marshal(event.Data)
			fmt.Fprintf(w, "id: %d\n", event.ID)
			fmt.Fprintf(w, "event: %s\n", event.Name)
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
		}
	}
}

func (api *API) handleRPC(w http.ResponseWriter, r *http.Request) {
	method := strings.TrimSpace(r.PathValue("method"))
	var request rpcRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	result, err := api.call(r.Context(), method, request.Args)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (api *API) handleTerminalConnectToken(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Aivo-Terminal-CSRF") == "" {
		writeError(w, http.StatusForbidden, errors.New("missing terminal CSRF header"))
		return
	}
	terminalID := strings.TrimSpace(r.PathValue("id"))
	workspaceRoot := strings.TrimSpace(r.URL.Query().Get("workspaceRoot"))
	if workspaceRoot == "" {
		var input struct {
			WorkspaceRoot string `json:"workspaceRoot"`
		}
		_ = json.NewDecoder(r.Body).Decode(&input)
		workspaceRoot = strings.TrimSpace(input.WorkspaceRoot)
	}
	if _, err := api.service.GetTerminal(r.Context(), workspaceRoot, terminalID); err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	ticket, err := api.terminalTickets.create(workspaceRoot, terminalID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ticket": ticket.Token, "expiresAt": ticket.ExpiresAt})
}

func (api *API) handleTerminalConnect(w http.ResponseWriter, r *http.Request) {
	if !validTerminalOrigin(r.Header.Get("Origin")) {
		writeError(w, http.StatusForbidden, errors.New("invalid websocket origin"))
		return
	}
	terminalID := strings.TrimSpace(r.PathValue("id"))
	workspaceRoot := strings.TrimSpace(r.URL.Query().Get("workspaceRoot"))
	ticket := strings.TrimSpace(r.URL.Query().Get("ticket"))
	cursor := int64(-1)
	if rawCursor := strings.TrimSpace(r.URL.Query().Get("cursor")); rawCursor != "" {
		if parsed, err := strconv.ParseInt(rawCursor, 10, 64); err == nil {
			cursor = parsed
		}
	}
	if !api.terminalTickets.consume(ticket, workspaceRoot, terminalID) {
		writeError(w, http.StatusForbidden, errors.New("invalid terminal ticket"))
		return
	}
	attachment, err := api.service.AttachTerminal(r.Context(), coreapp.TerminalAttachInput{WorkspaceRoot: workspaceRoot, TerminalID: terminalID, Cursor: cursor})
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	defer attachment.Detach()
	upgrader := websocket.Upgrader{
		CheckOrigin: func(req *http.Request) bool {
			return validTerminalOrigin(req.Header.Get("Origin"))
		},
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()
	if replay := attachment.Replay(); len(replay) > 0 {
		if err := writeTerminalChunks(conn, replay); err != nil {
			return
		}
	}
	if err := writeTerminalControl(conn, map[string]any{"cursor": attachment.Cursor()}); err != nil {
		return
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			messageType, payload, err := conn.ReadMessage()
			if err != nil {
				return
			}
			if len(payload) == 0 {
				continue
			}
			if payload[0] == 0 {
				var control struct {
					Type string `json:"type"`
					Rows int    `json:"rows"`
					Cols int    `json:"cols"`
				}
				if len(payload) > 1 && json.Unmarshal(payload[1:], &control) == nil && control.Type == "resize" {
					_ = attachment.Resize(control.Rows, control.Cols)
				}
				continue
			}
			if messageType == websocket.TextMessage || messageType == websocket.BinaryMessage {
				_ = attachment.Write(payload)
			}
		}
	}()
	for {
		select {
		case <-r.Context().Done():
			return
		case <-done:
			return
		case data, ok := <-attachment.Data():
			if !ok {
				_ = writeTerminalControl(conn, map[string]any{"type": "exit"})
				return
			}
			if err := writeTerminalChunks(conn, data); err != nil {
				return
			}
		}
	}
}

func validTerminalOrigin(origin string) bool {
	origin = strings.TrimSpace(origin)
	if origin == "" || origin == "null" || strings.HasPrefix(origin, "file://") {
		return true
	}
	return strings.HasPrefix(origin, "http://127.0.0.1:") || strings.HasPrefix(origin, "http://localhost:")
}

func writeTerminalChunks(conn *websocket.Conn, data []byte) error {
	const maxFrame = 16 * 1024
	for len(data) > 0 {
		n := len(data)
		if n > maxFrame {
			n = maxFrame
		}
		if err := conn.WriteMessage(websocket.BinaryMessage, data[:n]); err != nil {
			return err
		}
		data = data[n:]
	}
	return nil
}

func writeTerminalControl(conn *websocket.Conn, payload map[string]any) error {
	raw, _ := json.Marshal(payload)
	frame := append([]byte{0}, raw...)
	return conn.WriteMessage(websocket.BinaryMessage, frame)
}

func (api *API) call(ctx context.Context, method string, args []json.RawMessage) (interface{}, error) {
	switch method {
	case "GetAppConfig":
		return api.service.AppConfig(ctx)
	case "GetProviderCatalog":
		return api.service.Catalog(ctx)
	case "ConnectProvider":
		input, err := arg[domain.ProviderConnectInput](args, 0)
		if err != nil {
			return nil, err
		}
		return api.service.ConnectProvider(ctx, input)
	case "SaveProvider":
		input, err := arg[domain.ProviderConnectInput](args, 0)
		if err != nil {
			return nil, err
		}
		return api.service.SaveProvider(ctx, input)
	case "UpdateModelPreferences":
		input, err := arg[domain.ModelPreferencesInput](args, 0)
		if err != nil {
			return nil, err
		}
		cfg, err := api.service.UpdateModelPreferences(ctx, input)
		if err == nil {
			api.events.emit("config.changed", cfg)
		}
		return cfg, err
	case "RefreshProviderModels":
		input, err := arg[domain.ProviderConnectInput](args, 0)
		if err != nil {
			return nil, err
		}
		return api.service.RefreshProviderModels(ctx, input)
	case "ValidateProvider":
		input, err := arg[domain.ProviderConnectInput](args, 0)
		if err != nil {
			return nil, err
		}
		return api.service.ValidateProvider(ctx, input)
	case "CheckProviderIntegration":
		input, err := arg[domain.ProviderIntegrationCheckInput](args, 0)
		if err != nil {
			return nil, err
		}
		return api.service.CheckProviderIntegration(ctx, input)
	case "ListProviderCallEvents":
		input, err := arg[domain.ProviderCallEventsInput](args, 0)
		if err != nil {
			return nil, err
		}
		return api.service.ListProviderCallEvents(ctx, input)
	case "GetProviderUsage":
		input, err := arg[domain.ProviderUsageInput](args, 0)
		if err != nil {
			return nil, err
		}
		return api.service.GetProviderUsage(ctx, input)
	case "DeleteProvider":
		id, err := arg[string](args, 0)
		if err != nil {
			return nil, err
		}
		return api.service.DeleteProvider(ctx, id)
	case "DeleteProviderAccount":
		id, err := arg[string](args, 0)
		if err != nil {
			return nil, err
		}
		return api.service.DeleteProviderAccount(ctx, id)
	case "StartProviderAuth":
		input, err := arg[domain.ProviderAuthStartInput](args, 0)
		if err != nil {
			return nil, err
		}
		return api.service.StartProviderAuth(ctx, input)
	case "GetProviderAuthStatus":
		id, err := arg[string](args, 0)
		if err != nil {
			return nil, err
		}
		return api.service.GetProviderAuthStatus(ctx, id)
	case "CancelProviderAuth":
		id, err := arg[string](args, 0)
		if err != nil {
			return nil, err
		}
		return api.service.CancelProviderAuth(ctx, id)
	case "CompleteInitialization":
		input, err := arg[*domain.ProviderConfig](args, 0)
		if err != nil {
			return nil, err
		}
		return api.service.CompleteInitialization(ctx, input)
	case "SelectProjectDirectory":
		return api.service.SelectProjectDirectory("")
	case "UpsertProject":
		path, err := arg[string](args, 0)
		if err != nil {
			return nil, err
		}
		return api.service.UpsertProject(ctx, path)
	case "SetProjectSidebarHidden":
		path, err := arg[string](args, 0)
		if err != nil {
			return nil, err
		}
		hidden, err := arg[bool](args, 1)
		if err != nil {
			return nil, err
		}
		return api.service.SetProjectSidebarHidden(ctx, path, hidden)
	case "ListRecentProjects":
		limit, err := arg[int](args, 0)
		if err != nil {
			return nil, err
		}
		return api.service.ListProjects(ctx, limit)
	case "CreateSession":
		input, err := arg[domain.CreateSessionRequest](args, 0)
		if err != nil {
			return nil, err
		}
		return api.service.CreateRuntimeSession(ctx, input)
	case "ListSessions":
		input, err := arg[domain.ListSessionsRequest](args, 0)
		if err != nil {
			return nil, err
		}
		return api.service.ListRuntimeSessions(ctx, input)
	case "SubmitSessionMessage", "SubmitSessionMessageStreaming":
		input, err := arg[domain.SubmitSessionMessageRequest](args, 0)
		if err != nil {
			return nil, err
		}
		return api.service.SubmitSessionMessageStreaming(ctx, input)
	case "GetSession":
		id, err := arg[string](args, 0)
		if err != nil {
			return nil, err
		}
		return api.service.GetRuntimeSession(ctx, id)
	case "UpdateSession":
		input, err := arg[domain.UpdateSessionRequest](args, 0)
		if err != nil {
			return nil, err
		}
		return api.service.UpdateRuntimeSession(ctx, input)
	case "ListAgentModes":
		includeHidden, err := arg[bool](args, 0)
		if err != nil {
			return nil, err
		}
		return api.service.ListAgentModes(ctx, includeHidden)
	case "SetSessionAgentMode":
		input, err := arg[domain.SetSessionAgentModeInput](args, 0)
		if err != nil {
			return nil, err
		}
		return api.service.SetSessionAgentMode(ctx, input)
	case "ListAgentRuns":
		input, err := arg[domain.AgentRunListRequest](args, 0)
		if err != nil {
			return nil, err
		}
		return api.service.ListAgentRuns(ctx, input)
	case "CancelAgentRun":
		id, err := arg[string](args, 0)
		if err != nil {
			return nil, err
		}
		return api.service.CancelAgentRun(ctx, id)
	case "ListTodoItems":
		input, err := arg[domain.TodoListInput](args, 0)
		if err != nil {
			return nil, err
		}
		return api.service.ListTodoItems(ctx, input)
	case "ListScheduledJobs":
		input, err := arg[domain.ScheduledJobListInput](args, 0)
		if err != nil {
			return nil, err
		}
		return api.service.ListScheduledJobs(ctx, input)
	case "SaveScheduledJob":
		input, err := arg[domain.ScheduledJobInput](args, 0)
		if err != nil {
			return nil, err
		}
		return api.service.SaveScheduledJob(ctx, input)
	case "DeleteScheduledJob":
		id, err := arg[string](args, 0)
		if err != nil {
			return nil, err
		}
		return nil, api.service.DeleteScheduledJob(ctx, id)
	case "RunDueScheduledJobs":
		limit, err := arg[int](args, 0)
		if err != nil {
			return nil, err
		}
		return api.service.RunDueScheduledJobs(ctx, limit)
	case "ArchiveSession":
		id, err := arg[string](args, 0)
		if err != nil {
			return nil, err
		}
		return api.service.ArchiveRuntimeSession(ctx, id)
	case "DeleteSession":
		id, err := arg[string](args, 0)
		if err != nil {
			return nil, err
		}
		return api.service.DeleteRuntimeSession(ctx, id)
	case "GetLatestSession":
		return api.service.ContinueLastSession(ctx)
	case "GetLatestSessionByProject":
		path, err := arg[string](args, 0)
		if err != nil {
			return nil, err
		}
		return api.service.ContinueProjectSession(ctx, path)
	case "ListSessionEvents":
		sessionID, err := arg[string](args, 0)
		if err != nil {
			return nil, err
		}
		includeNonNormal, err := arg[bool](args, 1)
		if err != nil {
			return nil, err
		}
		limit, err := arg[int](args, 2)
		if err != nil {
			return nil, err
		}
		return api.service.ListEvents(ctx, sessionID, includeNonNormal, limit)
	case "AppendSessionEvent":
		input, err := arg[domain.AppendEventRequest](args, 0)
		if err != nil {
			return nil, err
		}
		return api.service.AppendEvent(ctx, input)
	case "UpdateSessionEvent":
		input, err := arg[domain.UpdateSessionEventRequest](args, 0)
		if err != nil {
			return nil, err
		}
		return api.service.UpdateSessionEvent(ctx, input)
	case "DeleteSessionEvent":
		input, err := arg[domain.DeleteSessionEventRequest](args, 0)
		if err != nil {
			return nil, err
		}
		return api.service.DeleteSessionEvent(ctx, input)
	case "StartSessionTurn":
		input, err := arg[domain.StartTurnRequest](args, 0)
		if err != nil {
			return nil, err
		}
		return api.service.StartTurn(ctx, input)
	case "CompleteSessionTurn":
		input, err := arg[domain.CompleteTurnRequest](args, 0)
		if err != nil {
			return nil, err
		}
		return api.service.CompleteTurn(ctx, input)
	case "FailSessionTurn":
		input, err := arg[domain.FailTurnRequest](args, 0)
		if err != nil {
			return nil, err
		}
		return api.service.FailTurn(ctx, input)
	case "CancelSessionTurn":
		input, err := arg[domain.CancelTurnRequest](args, 0)
		if err != nil {
			return nil, err
		}
		return api.service.CancelTurn(ctx, input)
	case "RetrySessionTurn":
		input, err := arg[domain.RetrySessionTurnRequest](args, 0)
		if err != nil {
			return nil, err
		}
		return api.service.RetrySessionTurnStreaming(ctx, input)
	case "GetSessionExecutionState":
		sessionID, err := arg[string](args, 0)
		if err != nil {
			return nil, err
		}
		return api.service.GetSessionExecutionState(ctx, sessionID)
	case "InterruptSessionExecution":
		input, err := arg[domain.InterruptSessionExecutionInput](args, 0)
		if err != nil {
			return nil, err
		}
		return api.service.InterruptSessionExecution(ctx, input)
	case "ResumeSessionExecution":
		input, err := arg[domain.ResumeSessionExecutionInput](args, 0)
		if err != nil {
			return nil, err
		}
		return api.service.ResumeSessionExecution(ctx, input)
	case "CompactSessionContext":
		input, err := arg[domain.CompactSessionContextInput](args, 0)
		if err != nil {
			return nil, err
		}
		return api.service.CompactSessionContext(ctx, input)
	case "ListSessionEventsAfterCursor":
		input, err := arg[domain.ListSessionEventsAfterCursorInput](args, 0)
		if err != nil {
			return nil, err
		}
		return api.service.ListSessionEventsAfterCursor(ctx, input)
	case "GetSessionTurnDiff":
		input, err := arg[domain.GetSessionTurnDiffRequest](args, 0)
		if err != nil {
			return nil, err
		}
		return api.service.GetSessionTurnDiff(ctx, input)
	case "ApplySessionTurnFileState":
		input, err := arg[domain.ApplySessionTurnFileStateRequest](args, 0)
		if err != nil {
			return nil, err
		}
		return api.service.ApplySessionTurnFileState(ctx, input)
	case "ForkSession":
		input, err := arg[domain.ForkSessionRequest](args, 0)
		if err != nil {
			return nil, err
		}
		return api.service.ForkSession(ctx, input)
	case "CreateSessionSummary":
		input, err := arg[domain.CreateSummaryRequest](args, 0)
		if err != nil {
			return nil, err
		}
		return api.service.CreateSummary(ctx, input)
	case "GetLatestSessionSummary":
		id, err := arg[string](args, 0)
		if err != nil {
			return nil, err
		}
		return api.service.LatestSummary(ctx, id)
	case "CreateSessionCheckpoint":
		input, err := arg[domain.CreateCheckpointRequest](args, 0)
		if err != nil {
			return nil, err
		}
		return api.service.CreateCheckpoint(ctx, input)
	case "ListSessionCheckpoints":
		sessionID, err := arg[string](args, 0)
		if err != nil {
			return nil, err
		}
		limit, err := arg[int](args, 1)
		if err != nil {
			return nil, err
		}
		return api.service.ListCheckpoints(ctx, sessionID, limit)
	case "GetLatestSessionCheckpoint":
		id, err := arg[string](args, 0)
		if err != nil {
			return nil, err
		}
		return api.service.LatestCheckpoint(ctx, id)
	case "GetCodingContext":
		id, err := arg[string](args, 0)
		if err != nil {
			return nil, err
		}
		return api.service.GetCodingContext(ctx, id)
	case "UpdateCodingContext":
		sessionID, err := arg[string](args, 0)
		if err != nil {
			return nil, err
		}
		projectPath, err := arg[string](args, 1)
		if err != nil {
			return nil, err
		}
		return api.service.CreateOrUpdateCodingContext(ctx, sessionID, projectPath)
	case "ResumeSession":
		input, err := arg[domain.ResumeSessionRequest](args, 0)
		if err != nil {
			return nil, err
		}
		return api.service.ResumeRecap(ctx, input)
	case "BuildSessionContext":
		input, err := arg[domain.BuildSessionContextRequest](args, 0)
		if err != nil {
			return nil, err
		}
		return api.service.BuildSessionContext(ctx, input)
	case "ListSessionTurns":
		sessionID, err := arg[string](args, 0)
		if err != nil {
			return nil, err
		}
		limit, err := arg[int](args, 1)
		if err != nil {
			return nil, err
		}
		return api.service.ListTurns(ctx, sessionID, limit)
	case "SaveSessionToolCall":
		input, err := arg[domain.CreateToolCallRequest](args, 0)
		if err != nil {
			return nil, err
		}
		return api.service.SaveToolCall(ctx, input)
	case "ListSessionToolCalls":
		sessionID, err := arg[string](args, 0)
		if err != nil {
			return nil, err
		}
		return api.service.ListToolCalls(ctx, sessionID)
	case "ReplaySessionToolCall":
		input, err := arg[domain.ReplaySessionToolCallRequest](args, 0)
		if err != nil {
			return nil, err
		}
		return api.service.ReplaySessionToolCall(ctx, input)
	case "ReadRetainedOutput":
		input, err := arg[domain.RetainedOutputReadInput](args, 0)
		if err != nil {
			return nil, err
		}
		return api.service.ReadRetainedOutput(ctx, input)
	case "ListPermissionRequests":
		sessionID, err := arg[string](args, 0)
		if err != nil {
			return nil, err
		}
		status, err := arg[string](args, 1)
		if err != nil {
			return nil, err
		}
		return api.service.ListPermissionRequests(ctx, sessionID, status)
	case "GetPermissionMode":
		sessionID, err := arg[string](args, 0)
		if err != nil {
			return nil, err
		}
		return api.service.GetPermissionMode(ctx, sessionID)
	case "SetPermissionMode":
		input, err := arg[domain.PermissionModeInput](args, 0)
		if err != nil {
			return nil, err
		}
		return api.service.SetPermissionMode(ctx, input)
	case "ApprovePermissionRequest":
		input, err := arg[domain.ApprovePermissionRequestInput](args, 0)
		if err != nil {
			return nil, err
		}
		return api.service.ApprovePermissionRequest(ctx, input)
	case "DenyPermissionRequest":
		input, err := arg[domain.DenyPermissionRequestInput](args, 0)
		if err != nil {
			return nil, err
		}
		return api.service.DenyPermissionRequest(ctx, input)
	case "ListQuestionRequests":
		sessionID, err := arg[string](args, 0)
		if err != nil {
			return nil, err
		}
		status, err := arg[string](args, 1)
		if err != nil {
			return nil, err
		}
		return api.service.ListQuestionRequests(ctx, sessionID, status)
	case "ReplyQuestionRequest":
		input, err := arg[domain.ReplyQuestionRequestInput](args, 0)
		if err != nil {
			return nil, err
		}
		return api.service.ReplyQuestionRequest(ctx, input)
	case "RejectQuestionRequest":
		input, err := arg[domain.RejectQuestionRequestInput](args, 0)
		if err != nil {
			return nil, err
		}
		return api.service.RejectQuestionRequest(ctx, input)
	case "ListTerminals":
		workspaceRoot, err := arg[string](args, 0)
		if err != nil {
			return nil, err
		}
		return api.service.ListTerminals(ctx, workspaceRoot)
	case "CreateTerminal":
		input, err := arg[coreapp.TerminalCreateInput](args, 0)
		if err != nil {
			return nil, err
		}
		return api.service.CreateTerminal(ctx, input)
	case "GetTerminal":
		workspaceRoot, err := arg[string](args, 0)
		if err != nil {
			return nil, err
		}
		terminalID, err := arg[string](args, 1)
		if err != nil {
			return nil, err
		}
		return api.service.GetTerminal(ctx, workspaceRoot, terminalID)
	case "UpdateTerminal":
		input, err := arg[coreapp.TerminalUpdateInput](args, 0)
		if err != nil {
			return nil, err
		}
		return api.service.UpdateTerminal(ctx, input)
	case "RemoveTerminal":
		workspaceRoot, err := arg[string](args, 0)
		if err != nil {
			return nil, err
		}
		terminalID, err := arg[string](args, 1)
		if err != nil {
			return nil, err
		}
		return nil, api.service.RemoveTerminal(ctx, workspaceRoot, terminalID)
	case "PollShellProcess":
		id, err := arg[string](args, 0)
		if err != nil {
			return nil, err
		}
		return api.service.PollShellProcess(id)
	case "WaitShellProcess":
		id, err := arg[string](args, 0)
		if err != nil {
			return nil, err
		}
		return api.service.WaitShellProcess(ctx, id)
	case "KillShellProcess":
		id, err := arg[string](args, 0)
		if err != nil {
			return nil, err
		}
		return api.service.KillShellProcess(id)
	case "ReadShellProcessOutput":
		id, err := arg[string](args, 0)
		if err != nil {
			return nil, err
		}
		return api.service.ReadShellProcessOutput(id)
	case "ListPlugins":
		input, err := arg[domain.PluginListInput](args, 0)
		if err != nil {
			return nil, err
		}
		return api.service.ListPlugins(ctx, input)
	case "InstallPluginFromPath":
		input, err := arg[domain.InstallPluginInput](args, 0)
		if err != nil {
			return nil, err
		}
		return api.service.InstallPluginFromPath(ctx, input)
	case "SetPluginEnabled":
		input, err := arg[domain.SetPluginEnabledInput](args, 0)
		if err != nil {
			return nil, err
		}
		return api.service.SetPluginEnabled(ctx, input)
	case "ReloadPlugins":
		return api.service.ReloadPlugins(ctx)
	case "ListMCPServers":
		input, err := arg[domain.MCPServerListInput](args, 0)
		if err != nil {
			return nil, err
		}
		return api.service.ListMCPServers(ctx, input)
	case "SaveMCPServer":
		input, err := arg[domain.SaveMCPServerInput](args, 0)
		if err != nil {
			return nil, err
		}
		return api.service.SaveMCPServer(ctx, input)
	case "SetMCPServerEnabled":
		input, err := arg[domain.SetMCPServerEnabledInput](args, 0)
		if err != nil {
			return nil, err
		}
		return api.service.SetMCPServerEnabled(ctx, input)
	case "ProbeMCPServer":
		input, err := arg[domain.MCPProbeInput](args, 0)
		if err != nil {
			return nil, err
		}
		return api.service.ProbeMCPServer(ctx, input)
	case "GetMCPPrompt":
		input, err := arg[domain.MCPPromptGetInput](args, 0)
		if err != nil {
			return nil, err
		}
		return api.service.GetMCPPrompt(ctx, input)
	case "ReadMCPResource":
		input, err := arg[domain.MCPResourceReadInput](args, 0)
		if err != nil {
			return nil, err
		}
		return api.service.ReadMCPResource(ctx, input)
	case "InsertMCPPromptIntoSession":
		input, err := arg[domain.InsertMCPPromptIntoSessionInput](args, 0)
		if err != nil {
			return nil, err
		}
		return api.service.InsertMCPPromptIntoSession(ctx, input)
	case "InsertMCPResourceIntoSession":
		input, err := arg[domain.InsertMCPResourceIntoSessionInput](args, 0)
		if err != nil {
			return nil, err
		}
		return api.service.InsertMCPResourceIntoSession(ctx, input)
	case "ReadMCPServerLog":
		input, err := arg[domain.MCPServerLogInput](args, 0)
		if err != nil {
			return nil, err
		}
		return api.service.ReadMCPServerLog(ctx, input)
	case "DiscoverMCPOAuth":
		input, err := arg[domain.MCPOAuthDiscoveryInput](args, 0)
		if err != nil {
			return nil, err
		}
		return api.service.DiscoverMCPOAuth(ctx, input)
	case "StartMCPOAuth":
		input, err := arg[domain.MCPOAuthStartInput](args, 0)
		if err != nil {
			return nil, err
		}
		return api.service.StartMCPOAuth(ctx, input)
	case "GetMCPOAuthStatus":
		input, err := arg[domain.MCPOAuthStatusInput](args, 0)
		if err != nil {
			return nil, err
		}
		return api.service.GetMCPOAuthStatus(ctx, input)
	case "ListToolCatalog":
		input, err := arg[domain.ToolCatalogInput](args, 0)
		if err != nil {
			return nil, err
		}
		return api.service.ListToolCatalog(ctx, input)
	case "DescribeTool":
		input, err := arg[domain.ToolDescribeInput](args, 0)
		if err != nil {
			return nil, err
		}
		return api.service.DescribeTool(ctx, input)
	case "GetSessionActiveTools":
		sessionID, err := arg[string](args, 0)
		if err != nil {
			return nil, err
		}
		return api.service.GetSessionActiveTools(ctx, sessionID)
	case "SetSessionActiveTools":
		input, err := arg[domain.SessionActiveToolsInput](args, 0)
		if err != nil {
			return nil, err
		}
		return api.service.SetSessionActiveTools(ctx, input)
	default:
		return nil, fmt.Errorf("unknown RPC method %q", method)
	}
}

func arg[T any](args []json.RawMessage, index int) (T, error) {
	var zero T
	if index >= len(args) {
		return zero, fmt.Errorf("missing argument %d", index)
	}
	if err := json.Unmarshal(args[index], &zero); err != nil {
		return zero, err
	}
	return zero, nil
}

func writeJSON(w http.ResponseWriter, status int, value interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, rpcError{Error: err.Error()})
}
