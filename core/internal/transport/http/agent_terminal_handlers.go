package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/gorilla/websocket"

	coreapp "aivo/core/app"
)

func (api *API) handleAgentTerminalConnectToken(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Aivo-Terminal-CSRF") == "" {
		writeError(w, http.StatusForbidden, errors.New("missing terminal CSRF header"))
		return
	}
	processRef := strings.TrimSpace(r.PathValue("ref"))
	var input coreapp.AgentTerminalAttachInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	input.ProcessRef = processRef
	if err := api.service.ValidateAgentTerminalOwner(input); err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	ticket, err := api.terminalTickets.createAgent(input.WorkspaceRoot, input.SessionID, processRef)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ticket": ticket.Token, "expiresAt": ticket.ExpiresAt})
}

func (api *API) handleAgentTerminalConnect(w http.ResponseWriter, r *http.Request) {
	if !validTerminalOrigin(r.Header.Get("Origin")) {
		writeError(w, http.StatusForbidden, errors.New("invalid websocket origin"))
		return
	}
	input := coreapp.AgentTerminalAttachInput{
		WorkspaceRoot: strings.TrimSpace(r.URL.Query().Get("workspaceRoot")), SessionID: strings.TrimSpace(r.URL.Query().Get("sessionId")),
		ProcessRef: strings.TrimSpace(r.PathValue("ref")), Cursor: -1,
	}
	if raw := strings.TrimSpace(r.URL.Query().Get("cursor")); raw != "" {
		if value, err := strconv.ParseInt(raw, 10, 64); err == nil {
			input.Cursor = value
		}
	}
	if !api.terminalTickets.consumeAgent(strings.TrimSpace(r.URL.Query().Get("ticket")), input.WorkspaceRoot, input.SessionID, input.ProcessRef) {
		writeError(w, http.StatusForbidden, errors.New("invalid agent terminal ticket"))
		return
	}
	attachment, err := api.service.AttachAgentTerminal(input)
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	defer attachment.Detach()
	conn, err := (&websocket.Upgrader{CheckOrigin: func(req *http.Request) bool { return validTerminalOrigin(req.Header.Get("Origin")) }}).Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()
	var writeMu sync.Mutex
	writeChunks := func(data []byte) error {
		writeMu.Lock()
		defer writeMu.Unlock()
		return writeTerminalChunks(conn, data)
	}
	writeControl := func(value map[string]any) error {
		writeMu.Lock()
		defer writeMu.Unlock()
		return writeTerminalControl(conn, value)
	}
	if attachment.Snapshot.Output != "" {
		if err := writeChunks([]byte(attachment.Snapshot.Output)); err != nil {
			return
		}
	}
	if err := writeControl(agentTerminalSnapshotControl("snapshot", attachment.Snapshot)); err != nil {
		return
	}
	var leaseVersion atomic.Int64
	leaseVersion.Store(attachment.Snapshot.LeaseVersion)
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
					Type         string `json:"type"`
					Rows         int    `json:"rows"`
					Cols         int    `json:"cols"`
					Mode         string `json:"mode"`
					RequestID    string `json:"requestId"`
					LeaseVersion int64  `json:"leaseVersion"`
				}
				if len(payload) > 1 && json.Unmarshal(payload[1:], &control) == nil {
					switch control.Type {
					case "resize":
						_, err = api.service.ResizeAgentTerminal(r.Context(), input, control.Rows, control.Cols)
					case "terminate":
						_, err = api.service.TerminateAgentTerminal(r.Context(), input)
					case "acquire_input":
						mode := control.Mode
						if mode == "" {
							mode = coreapp.AgentPTYInputUserOnce
						}
						var result coreapp.AgentPTYResult
						result, err = api.service.ResolveAgentTerminalInput(r.Context(), coreapp.ResolveAgentTerminalInputRequest{
							WorkspaceRoot: input.WorkspaceRoot, SessionID: input.SessionID, ProcessRef: input.ProcessRef,
							RequestID: control.RequestID, Mode: mode,
						})
						if err == nil {
							leaseVersion.Store(result.LeaseVersion)
							_ = writeControl(agentTerminalSnapshotControl("input_granted", result))
						}
					case "release_input":
						var result coreapp.AgentPTYResult
						result, err = api.service.ReleaseAgentTerminalInput(r.Context(), coreapp.ReleaseAgentTerminalInputRequest{
							WorkspaceRoot: input.WorkspaceRoot, SessionID: input.SessionID, ProcessRef: input.ProcessRef,
							LeaseVersion: control.LeaseVersion,
						})
						if err == nil {
							leaseVersion.Store(result.LeaseVersion)
							_ = writeControl(agentTerminalSnapshotControl("input_granted", result))
						}
					}
				}
			} else if messageType == websocket.TextMessage || messageType == websocket.BinaryMessage {
				input.LeaseVersion = leaseVersion.Load()
				input.EnforceLeaseVersion = true
				var result coreapp.AgentPTYResult
				result, err = api.service.WriteAgentTerminalUserInput(r.Context(), input, string(payload))
				if err == nil {
					leaseVersion.Store(result.LeaseVersion)
				}
			}
			if err != nil {
				_ = writeControl(map[string]any{"type": "input_rejected", "message": err.Error(), "leaseVersion": leaseVersion.Load()})
			}
		}
	}()
	for {
		select {
		case <-r.Context().Done():
			return
		case <-done:
			return
		case event, ok := <-attachment.Events():
			if !ok {
				_ = writeControl(map[string]any{"type": "exit"})
				return
			}
			if event.Type == "output" {
				if err := writeChunks(event.Data); err != nil {
					return
				}
				continue
			}
			leaseVersion.Store(event.Snapshot.LeaseVersion)
			if err := writeControl(agentTerminalSnapshotControl(event.Type, event.Snapshot)); err != nil {
				return
			}
		}
	}
}

func agentTerminalSnapshotControl(eventType string, value coreapp.AgentPTYResult) map[string]any {
	return map[string]any{"type": eventType, "processRef": value.ProcessRef, "status": value.Status, "cursor": value.ProcessCursor,
		"baseCursor": value.BaseCursor, "rows": value.Rows, "cols": value.Cols, "inputMode": value.InputMode, "inputRequest": value.InputRequest,
		"exitCode": value.ExitCode, "truncated": value.OutputTruncated, "attention": value.Attention,
		"inputOwner": value.InputOwner, "leaseMode": value.LeaseMode, "leaseVersion": value.LeaseVersion,
		"title": value.Title, "command": value.Command, "origin": value.Origin}
}
