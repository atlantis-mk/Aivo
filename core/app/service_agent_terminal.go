package app

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"aivo/core/domain"
)

type AgentTerminalAttachInput struct {
	WorkspaceRoot       string `json:"workspaceRoot"`
	SessionID           string `json:"sessionId"`
	ProcessRef          string `json:"processRef"`
	Cursor              int64  `json:"cursor"`
	LeaseVersion        int64  `json:"leaseVersion,omitempty"`
	EnforceLeaseVersion bool   `json:"-"`
}

type ResolveAgentTerminalInputRequest struct {
	WorkspaceRoot string `json:"workspaceRoot"`
	SessionID     string `json:"sessionId"`
	ProcessRef    string `json:"processRef"`
	RequestID     string `json:"requestId"`
	Mode          string `json:"mode"`
}

type ReleaseAgentTerminalInputRequest struct {
	WorkspaceRoot string `json:"workspaceRoot"`
	SessionID     string `json:"sessionId"`
	ProcessRef    string `json:"processRef"`
	LeaseVersion  int64  `json:"leaseVersion"`
}

type UpdateSessionTerminalRequest struct {
	WorkspaceRoot string `json:"workspaceRoot"`
	SessionID     string `json:"sessionId"`
	ProcessRef    string `json:"processRef"`
	Title         string `json:"title"`
}

func (s *Service) ValidateAgentTerminalOwner(input AgentTerminalAttachInput) error {
	if strings.TrimSpace(input.SessionID) == "" || strings.TrimSpace(input.ProcessRef) == "" {
		return errors.New("sessionId and processRef are required")
	}
	return s.agentPTYRegistry().ValidateOwner(input.WorkspaceRoot, input.SessionID, input.ProcessRef)
}

func (s *Service) AttachAgentTerminal(input AgentTerminalAttachInput) (*AgentPTYAttachment, error) {
	return s.agentPTYRegistry().Attach(input.WorkspaceRoot, input.SessionID, input.ProcessRef, input.Cursor)
}

func (s *Service) WriteAgentTerminalUserInput(_ context.Context, input AgentTerminalAttachInput, chars string) (AgentPTYResult, error) {
	return s.agentPTYRegistry().WriteUserNow(AgentPTYWriteInput{
		WorkspaceRoot: input.WorkspaceRoot, SessionID: input.SessionID, ProcessRef: input.ProcessRef,
		Chars: chars, Cursor: input.Cursor, YieldTime: 100 * time.Millisecond,
		LeaseVersion: input.LeaseVersion, EnforceLeaseVersion: input.EnforceLeaseVersion,
	})
}

func (s *Service) ResizeAgentTerminal(ctx context.Context, input AgentTerminalAttachInput, rows, cols int) (AgentPTYResult, error) {
	return s.agentPTYRegistry().WriteUser(ctx, AgentPTYWriteInput{
		WorkspaceRoot: input.WorkspaceRoot, SessionID: input.SessionID, ProcessRef: input.ProcessRef,
		Cursor: input.Cursor, Rows: rows, Cols: cols, YieldTime: 100 * time.Millisecond,
	})
}

func (s *Service) TerminateAgentTerminal(ctx context.Context, input AgentTerminalAttachInput) (AgentPTYResult, error) {
	return s.agentPTYRegistry().WriteUser(ctx, AgentPTYWriteInput{
		WorkspaceRoot: input.WorkspaceRoot, SessionID: input.SessionID, ProcessRef: input.ProcessRef,
		Cursor: input.Cursor, Terminate: true, YieldTime: 100 * time.Millisecond,
	})
}

func (s *Service) ResolveAgentTerminalInput(ctx context.Context, input ResolveAgentTerminalInputRequest) (AgentPTYResult, error) {
	result, err := s.agentPTYRegistry().ResolveInput(AgentPTYResolveInput{
		WorkspaceRoot: input.WorkspaceRoot, SessionID: input.SessionID, ProcessRef: input.ProcessRef,
		RequestID: input.RequestID, Mode: input.Mode,
	})
	if err != nil {
		return result, err
	}
	if input.Mode == AgentPTYInputAgentOnce || input.Mode == AgentPTYInputAgentAlways {
		go s.resumeAgentTerminalInput(input, result.Cursor)
	}
	return result, nil
}

func (s *Service) ListSessionTerminals(_ context.Context, workspaceRoot, sessionID string) ([]AgentPTYResult, error) {
	return s.agentPTYRegistry().List(workspaceRoot, sessionID)
}

func (s *Service) ReleaseAgentTerminalInput(_ context.Context, input ReleaseAgentTerminalInputRequest) (AgentPTYResult, error) {
	return s.agentPTYRegistry().ReleaseInput(input.WorkspaceRoot, input.SessionID, input.ProcessRef, input.LeaseVersion)
}

func (s *Service) TerminateSessionTerminals(_ context.Context, workspaceRoot, sessionID string) error {
	registry := s.agentPTYRegistry()
	terminals, err := registry.List(workspaceRoot, sessionID)
	if err != nil {
		return err
	}
	for _, terminal := range terminals {
		if terminal.Status != AgentPTYStatusExited {
			_, _ = registry.Write(context.Background(), AgentPTYWriteInput{WorkspaceRoot: workspaceRoot, SessionID: sessionID, ProcessRef: terminal.ProcessRef, Cursor: terminal.ProcessCursor, Terminate: true, YieldTime: 100 * time.Millisecond})
		}
	}
	return nil
}

func (s *Service) UpdateSessionTerminal(_ context.Context, input UpdateSessionTerminalRequest) (AgentPTYResult, error) {
	return s.agentPTYRegistry().UpdateTitle(input.WorkspaceRoot, input.SessionID, input.ProcessRef, input.Title)
}

func (s *Service) RemoveSessionTerminal(_ context.Context, workspaceRoot, sessionID, processRef string) error {
	return s.agentPTYRegistry().Remove(workspaceRoot, sessionID, processRef)
}

func (s *Service) agentPTYRegistry() *AgentPTYRegistry {
	if s != nil && s.ptyManager != nil {
		return s.ptyManager
	}
	return defaultAgentPTYRegistry
}

func (s *Service) resumeAgentTerminalInput(input ResolveAgentTerminalInputRequest, cursor int64) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	for {
		state, _ := s.store.GetSessionExecutionState(ctx, input.SessionID)
		if state.Status != domain.ExecutionStatusRunning && state.Status != domain.ExecutionStatusCompacting {
			break
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(100 * time.Millisecond):
		}
	}
	prompt, promptErr := s.renderManagedPrompt("task.terminal_resume", map[string]string{
		"process_ref": input.ProcessRef, "cursor": fmt.Sprint(cursor), "mode": input.Mode,
	})
	if promptErr != nil {
		return
	}
	prepared, err := s.SubmitSessionMessageStreaming(context.Background(), domain.SubmitSessionMessageRequest{SessionID: input.SessionID, Text: prompt})
	if err == nil && prepared.UserEvent.ID != "" {
		_, _ = s.store.SetSessionEventVisibility(context.Background(), prepared.UserEvent.ID, domain.EventVisibilityInternal)
	}
}
