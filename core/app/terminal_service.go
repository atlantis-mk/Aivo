package app

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	TerminalStatusRunning = AgentPTYStatusRunning
	TerminalStatusExited  = AgentPTYStatusExited
	TerminalStatusRemoved = "removed"
	userTerminalSessionID = "__workspace_terminal__"
)

type TerminalService interface {
	List(context.Context, string) ([]TerminalInfo, error)
	Create(context.Context, TerminalCreateInput) (TerminalInfo, error)
	Get(context.Context, string, string) (TerminalInfo, error)
	Update(context.Context, TerminalUpdateInput) (TerminalInfo, error)
	Remove(context.Context, string, string) error
	Attach(context.Context, TerminalAttachInput) (TerminalAttachment, error)
}

type TerminalInfo struct {
	ID            string    `json:"id"`
	WorkspaceRoot string    `json:"workspaceRoot"`
	SessionID     string    `json:"sessionId,omitempty"`
	Origin        string    `json:"origin"`
	Title         string    `json:"title"`
	Command       string    `json:"command"`
	Args          []string  `json:"args,omitempty"`
	CWD           string    `json:"cwd"`
	Status        string    `json:"status"`
	Attention     string    `json:"attention"`
	InputOwner    string    `json:"inputOwner"`
	LeaseMode     string    `json:"leaseMode"`
	LeaseVersion  int64     `json:"leaseVersion"`
	PID           int       `json:"pid"`
	ExitCode      *int      `json:"exitCode,omitempty"`
	Rows          int       `json:"rows"`
	Cols          int       `json:"cols"`
	Cursor        int64     `json:"cursor"`
	Truncated     bool      `json:"truncated,omitempty"`
	TimeCreated   time.Time `json:"timeCreated"`
	TimeUpdated   time.Time `json:"timeUpdated"`
}

type TerminalCreateInput struct {
	WorkspaceRoot string            `json:"workspaceRoot"`
	SessionID     string            `json:"sessionId,omitempty"`
	CWD           string            `json:"cwd,omitempty"`
	Title         string            `json:"title,omitempty"`
	Shell         string            `json:"shell,omitempty"`
	Env           map[string]string `json:"env,omitempty"`
	Rows          int               `json:"rows,omitempty"`
	Cols          int               `json:"cols,omitempty"`
}

type TerminalUpdateInput struct {
	WorkspaceRoot string `json:"workspaceRoot"`
	TerminalID    string `json:"terminalId"`
	Title         string `json:"title,omitempty"`
	Rows          int    `json:"rows,omitempty"`
	Cols          int    `json:"cols,omitempty"`
}

type TerminalAttachInput struct {
	WorkspaceRoot string `json:"workspaceRoot"`
	TerminalID    string `json:"terminalId"`
	Cursor        int64  `json:"cursor"`
}

type TerminalAttachment interface {
	Replay() []byte
	Cursor() int64
	Write([]byte) error
	Resize(rows int, cols int) error
	Data() <-chan []byte
	Detach()
}

type DefaultTerminalService struct {
	registry *AgentPTYRegistry
	mu       sync.Mutex
	onEvent  func(string, TerminalInfo)
}

func NewTerminalService() *DefaultTerminalService {
	return NewTerminalServiceWithRegistry(NewAgentPTYRegistry())
}

func NewTerminalServiceWithRegistry(registry *AgentPTYRegistry) *DefaultTerminalService {
	if registry == nil {
		registry = NewAgentPTYRegistry()
	}
	return &DefaultTerminalService{registry: registry}
}

func (s *DefaultTerminalService) SetEventHook(hook func(string, TerminalInfo)) {
	s.mu.Lock()
	s.onEvent = hook
	s.mu.Unlock()
}

func (s *DefaultTerminalService) List(_ context.Context, workspaceRoot string) ([]TerminalInfo, error) {
	results, err := s.registry.List(workspaceRoot, userTerminalSessionID)
	if err != nil {
		return nil, err
	}
	infos := make([]TerminalInfo, 0, len(results))
	for _, result := range results {
		infos = append(infos, terminalInfoFromPTY(workspaceRoot, result))
	}
	return infos, nil
}

func (s *DefaultTerminalService) Create(ctx context.Context, input TerminalCreateInput) (TerminalInfo, error) {
	root, cwd, err := normalizeTerminalCWD(input.WorkspaceRoot, input.CWD)
	if err != nil {
		return TerminalInfo{}, err
	}
	shell := firstNonEmpty(strings.TrimSpace(input.Shell), defaultShell())
	command := "exec " + shellSingleQuote(shell)
	result, err := s.registry.Start(ctx, SandboxRequest{
		WorkspaceRoot: root, CWD: cwd, SessionID: userTerminalSessionID, Command: command,
		Shell: shell, EnvAllowlist: append(defaultEnvAllowlist(), "AIVO_TERMINAL"), EnvOverrides: input.Env,
	}, input.Rows, input.Cols, 100*time.Millisecond, agentPTYDefaultMaxChunk)
	if err != nil && result.ProcessRef == "" {
		return TerminalInfo{}, err
	}
	session, ownErr := s.registry.owned(root, userTerminalSessionID, result.ProcessRef)
	if ownErr != nil {
		return TerminalInfo{}, ownErr
	}
	session.mu.Lock()
	session.origin = "user"
	session.command = shell
	session.title = firstNonEmpty(strings.TrimSpace(input.Title), filepath.Base(shell))
	result = session.snapshotLocked(-1, agentPTYDefaultMaxChunk)
	session.mu.Unlock()
	info := terminalInfoFromPTY(root, result)
	s.emit("terminal.created", info)
	go func() {
		<-session.done
		exited := terminalInfoFromPTY(root, session.snapshot(-1, 0))
		s.emit("terminal.exited", exited)
	}()
	return info, nil
}

func (s *DefaultTerminalService) Get(_ context.Context, workspaceRoot, terminalID string) (TerminalInfo, error) {
	session, err := s.registry.owned(workspaceRoot, userTerminalSessionID, terminalID)
	if err != nil {
		return TerminalInfo{}, err
	}
	return terminalInfoFromPTY(workspaceRoot, session.snapshot(-1, 0)), nil
}

func (s *DefaultTerminalService) Update(_ context.Context, input TerminalUpdateInput) (TerminalInfo, error) {
	session, err := s.registry.owned(input.WorkspaceRoot, userTerminalSessionID, input.TerminalID)
	if err != nil {
		return TerminalInfo{}, err
	}
	if input.Rows > 0 || input.Cols > 0 {
		if input.Rows <= 0 || input.Cols <= 0 {
			return TerminalInfo{}, errors.New("rows and cols must be provided together")
		}
		if err := session.resize(input.Rows, input.Cols); err != nil {
			return TerminalInfo{}, err
		}
	}
	session.mu.Lock()
	if title := strings.TrimSpace(input.Title); title != "" {
		session.title = title
	}
	session.updatedAt = time.Now()
	result := session.snapshotLocked(-1, 0)
	session.mu.Unlock()
	info := terminalInfoFromPTY(input.WorkspaceRoot, result)
	s.emit("terminal.updated", info)
	return info, nil
}

func (s *DefaultTerminalService) Remove(_ context.Context, workspaceRoot, terminalID string) error {
	session, err := s.registry.owned(workspaceRoot, userTerminalSessionID, terminalID)
	if err != nil {
		return err
	}
	if err := s.registry.Remove(workspaceRoot, userTerminalSessionID, terminalID); err != nil {
		return err
	}
	info := terminalInfoFromPTY(workspaceRoot, session.snapshot(-1, 0))
	info.Status = TerminalStatusRemoved
	s.emit("terminal.removed", info)
	return nil
}

func (s *DefaultTerminalService) Attach(_ context.Context, input TerminalAttachInput) (TerminalAttachment, error) {
	attachment, err := s.registry.Attach(input.WorkspaceRoot, userTerminalSessionID, input.TerminalID, input.Cursor)
	if err != nil {
		return nil, err
	}
	data := make(chan []byte, 64)
	done := make(chan struct{})
	go func() {
		defer close(data)
		for {
			select {
			case <-done:
				return
			case event, ok := <-attachment.Events():
				if !ok {
					return
				}
				if event.Type == "output" {
					select {
					case data <- event.Data:
					default:
					}
				}
			}
		}
	}()
	return &unifiedTerminalAttachment{service: s, input: input, attachment: attachment, replay: []byte(attachment.Snapshot.Output), cursor: attachment.Snapshot.ProcessCursor, data: data, done: done}, nil
}

func (s *DefaultTerminalService) Shutdown() {
	if s != nil && s.registry != nil {
		s.registry.CleanupSession(userTerminalSessionID)
	}
}

func (s *DefaultTerminalService) emit(name string, info TerminalInfo) {
	s.mu.Lock()
	hook := s.onEvent
	s.mu.Unlock()
	if hook != nil {
		hook(name, info)
	}
}

type unifiedTerminalAttachment struct {
	service    *DefaultTerminalService
	input      TerminalAttachInput
	attachment *AgentPTYAttachment
	replay     []byte
	cursor     int64
	data       <-chan []byte
	done       chan struct{}
	once       sync.Once
}

func (a *unifiedTerminalAttachment) Replay() []byte      { return append([]byte(nil), a.replay...) }
func (a *unifiedTerminalAttachment) Cursor() int64       { return a.cursor }
func (a *unifiedTerminalAttachment) Data() <-chan []byte { return a.data }
func (a *unifiedTerminalAttachment) Write(data []byte) error {
	_, err := a.service.registry.WriteUserNow(AgentPTYWriteInput{WorkspaceRoot: a.input.WorkspaceRoot, SessionID: userTerminalSessionID, ProcessRef: a.input.TerminalID, Chars: string(data), Cursor: a.cursor})
	return err
}
func (a *unifiedTerminalAttachment) Resize(rows, cols int) error {
	session, err := a.service.registry.owned(a.input.WorkspaceRoot, userTerminalSessionID, a.input.TerminalID)
	if err != nil {
		return err
	}
	return session.resize(rows, cols)
}
func (a *unifiedTerminalAttachment) Detach() {
	a.once.Do(func() { close(a.done); a.attachment.Detach() })
}

func terminalInfoFromPTY(workspaceRoot string, result AgentPTYResult) TerminalInfo {
	created, _ := time.Parse(time.RFC3339Nano, result.CreatedAt)
	updated, _ := time.Parse(time.RFC3339Nano, result.UpdatedAt)
	return TerminalInfo{
		ID: result.ProcessRef, WorkspaceRoot: workspaceRoot, SessionID: result.SessionID, Origin: result.Origin,
		Title: result.Title, Command: result.Command, CWD: result.CWD, Status: result.Status, Attention: result.Attention,
		InputOwner: result.InputOwner, LeaseMode: result.LeaseMode, LeaseVersion: result.LeaseVersion,
		PID: result.PID, ExitCode: result.ExitCode, Rows: result.Rows, Cols: result.Cols, Cursor: result.ProcessCursor,
		Truncated: result.OutputTruncated || result.BaseCursor > 0, TimeCreated: created, TimeUpdated: updated,
	}
}

var _ TerminalService = (*DefaultTerminalService)(nil)
var _ TerminalAttachment = (*unifiedTerminalAttachment)(nil)

func (s *DefaultTerminalService) validate() error {
	if s == nil || s.registry == nil {
		return fmt.Errorf("terminal service is not configured")
	}
	return nil
}
