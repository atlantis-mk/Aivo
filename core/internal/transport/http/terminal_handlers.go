package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/websocket"

	coreapp "aivo/core/app"
)

func (api *API) handleTerminalConnectToken(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Aivo-Terminal-CSRF") == "" {
		writeError(w, http.StatusForbidden, errors.New("missing terminal CSRF header"))
		return
	}
	terminalID := strings.TrimSpace(r.PathValue("id"))
	workspaceRoot := strings.TrimSpace(r.URL.Query().Get("workspaceRoot"))
	var input struct {
		WorkspaceRoot string `json:"workspaceRoot"`
		SessionID     string `json:"sessionId"`
	}
	_ = json.NewDecoder(r.Body).Decode(&input)
	if workspaceRoot == "" {
		workspaceRoot = strings.TrimSpace(input.WorkspaceRoot)
	}
	if sessionID := strings.TrimSpace(input.SessionID); sessionID != "" {
		attach := coreapp.AgentTerminalAttachInput{WorkspaceRoot: workspaceRoot, SessionID: sessionID, ProcessRef: terminalID}
		if err := api.service.ValidateAgentTerminalOwner(attach); err != nil {
			writeError(w, http.StatusNotFound, err)
			return
		}
		ticket, err := api.terminalTickets.createAgent(workspaceRoot, sessionID, terminalID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ticket": ticket.Token, "expiresAt": ticket.ExpiresAt})
		return
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
	if strings.TrimSpace(r.URL.Query().Get("sessionId")) != "" {
		r.SetPathValue("ref", strings.TrimSpace(r.PathValue("id")))
		api.handleAgentTerminalConnect(w, r)
		return
	}
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
