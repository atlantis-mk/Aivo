package app

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/creack/pty"
	"github.com/google/uuid"

	"aivo/core/domain"
)

const (
	agentPTYBufferCap       = 256 * 1024
	agentPTYDefaultYield    = 10 * time.Second
	agentPTYMaxYield        = 30 * time.Second
	agentPTYIdleBoundary    = 300 * time.Millisecond
	agentPTYAttentionDelay  = 1500 * time.Millisecond
	agentPTYDefaultMaxChunk = 16 * 1024
	agentPTYControlMaxLine  = 16 * 1024
)

const (
	AgentPTYStatusRunning      = "running"
	AgentPTYStatusWaitingInput = "waiting_input"
	AgentPTYStatusExited       = "exited"

	AgentPTYAttentionNone            = "none"
	AgentPTYAttentionPossiblyWaiting = "possibly_waiting"
	AgentPTYAttentionInteractive     = "interactive"

	AgentPTYOwnerNone  = "none"
	AgentPTYOwnerUser  = "user"
	AgentPTYOwnerAgent = "agent"

	AgentPTYLeaseNone   = "none"
	AgentPTYLeaseOnce   = "once"
	AgentPTYLeaseAlways = "always"

	AgentPTYInputAsk         = "ask"
	AgentPTYInputAgentOnce   = "agent_once"
	AgentPTYInputUserOnce    = "user_once"
	AgentPTYInputAgentAlways = "agent_always"
)

type AgentPTYInputRequest struct {
	ID        string `json:"id"`
	Cursor    int64  `json:"cursor"`
	Mode      string `json:"mode"`
	Resolved  bool   `json:"resolved"`
	CreatedAt string `json:"createdAt"`
	Prompt    string `json:"prompt,omitempty"`
	Secret    bool   `json:"secret,omitempty"`
}

type AgentPTYResult struct {
	ProcessRef      string                `json:"processRef"`
	Status          string                `json:"status"`
	PID             int                   `json:"pid,omitempty"`
	ExitCode        *int                  `json:"exitCode,omitempty"`
	CWD             string                `json:"cwd,omitempty"`
	Rows            int                   `json:"rows"`
	Cols            int                   `json:"cols"`
	Cursor          int64                 `json:"cursor"`
	ProcessCursor   int64                 `json:"processCursor"`
	BaseCursor      int64                 `json:"baseCursor"`
	Output          string                `json:"output,omitempty"`
	OutputTruncated bool                  `json:"outputTruncated,omitempty"`
	YieldReason     string                `json:"yieldReason"`
	InputMode       string                `json:"inputMode"`
	InputRequest    *AgentPTYInputRequest `json:"inputRequest,omitempty"`
	Attention       string                `json:"attention"`
	InputOwner      string                `json:"inputOwner"`
	LeaseMode       string                `json:"leaseMode"`
	LeaseVersion    int64                 `json:"leaseVersion"`
	SessionID       string                `json:"sessionId,omitempty"`
	Origin          string                `json:"origin"`
	Title           string                `json:"title,omitempty"`
	Command         string                `json:"command,omitempty"`
	CreatedAt       string                `json:"createdAt"`
	UpdatedAt       string                `json:"updatedAt"`
}

type AgentPTYDecisionRequiredError struct {
	ProcessRef string
	RequestID  string
	Cursor     int64
	InputMode  string
}

func (e *AgentPTYDecisionRequiredError) Error() string {
	return "terminal input ownership decision is required"
}

type AgentPTYEvent struct {
	Type     string         `json:"type"`
	Data     []byte         `json:"-"`
	Snapshot AgentPTYResult `json:"snapshot,omitempty"`
}

type AgentPTYAttachment struct {
	Snapshot AgentPTYResult
	events   <-chan AgentPTYEvent
	detach   func()
}

func (a *AgentPTYAttachment) Events() <-chan AgentPTYEvent { return a.events }
func (a *AgentPTYAttachment) Detach() {
	if a != nil && a.detach != nil {
		a.detach()
	}
}

type AgentPTYWriteInput struct {
	WorkspaceRoot       string
	SessionID           string
	ProcessRef          string
	Chars               string
	Cursor              int64
	YieldTime           time.Duration
	MaxOutput           int
	Rows                int
	Cols                int
	Terminate           bool
	LeaseVersion        int64
	EnforceLeaseVersion bool
}

type AgentPTYResolveInput struct {
	WorkspaceRoot string
	SessionID     string
	ProcessRef    string
	RequestID     string
	Mode          string
}

type PTYManager struct {
	mu              sync.Mutex
	sessions        map[string]*agentPTYSession
	shutdown        bool
	globalBufferCap int
}

// AgentPTYRegistry is retained as a source-compatible name while all terminal
// entry points migrate to the unified PTY manager.
type AgentPTYRegistry = PTYManager

type agentPTYSession struct {
	mu               sync.Mutex
	operationMu      sync.Mutex
	ref              string
	workspaceRoot    string
	sessionID        string
	cwd              string
	command          string
	title            string
	origin           string
	createdAt        time.Time
	updatedAt        time.Time
	rows             int
	cols             int
	cmd              *exec.Cmd
	ptyFile          *os.File
	controlFile      *os.File
	buffer           []byte
	baseCursor       int64
	nextCursor       int64
	status           string
	exitCode         *int
	notify           chan struct{}
	done             chan struct{}
	doneOnce         sync.Once
	emitter          *shellOutputEmitter
	inputMode        string
	attention        string
	inputOwner       string
	leaseMode        string
	leaseVersion     int64
	inputRequest     *AgentPTYInputRequest
	lastPromptCursor int64
	subscribers      map[chan AgentPTYEvent]struct{}
	attentionTimer   *time.Timer
	lastViewedAt     time.Time
	registry         *AgentPTYRegistry
}

func NewPTYManager() *PTYManager {
	return &PTYManager{sessions: map[string]*agentPTYSession{}, globalBufferCap: 64 * 1024 * 1024}
}

func NewAgentPTYRegistry() *AgentPTYRegistry { return NewPTYManager() }

var defaultAgentPTYRegistry = NewAgentPTYRegistry()

func (r *AgentPTYRegistry) Start(ctx context.Context, request SandboxRequest, rows int, cols int, yield time.Duration, maxOutput int) (AgentPTYResult, error) {
	if r == nil {
		return AgentPTYResult{}, errors.New("agent PTY registry is not configured")
	}
	if strings.TrimSpace(request.SessionID) == "" {
		return AgentPTYResult{}, errors.New("sessionId is required for interactive process")
	}
	root, cwd, err := normalizeSandboxCWD(request.WorkspaceRoot, request.CWD, request.AllowExternalCWD)
	if err != nil {
		return AgentPTYResult{}, &SandboxError{Code: SandboxErrorPolicyDenied, Message: err.Error(), Err: err}
	}
	command := strings.TrimSpace(request.Command)
	if command == "" {
		return AgentPTYResult{}, errors.New("command is required")
	}
	rows, cols = normalizeTerminalSize(rows, cols)
	shell := firstNonEmpty(strings.TrimSpace(request.Shell), defaultShell())
	cmd := exec.Command(shell, shellCommandArgs(shell, command, request.LoginShell)...)
	cmd.Dir = cwd
	controlRead, controlWrite, err := os.Pipe()
	if err != nil {
		return AgentPTYResult{}, fmt.Errorf("create terminal control pipe: %w", err)
	}
	cmd.ExtraFiles = []*os.File{controlWrite}
	cmd.Env = append(
		SanitizedEnvironment(root, request.EnvAllowlist, request.Env, request.EnvOverrides),
		"AIVO_TERMINAL=agent", "TERM=xterm-256color", "AIVO_CONTROL_FD=3", "AIVO_CONTROL_PROTOCOL=1",
	)
	ptmx, err := pty.StartWithSize(cmd, &pty.Winsize{Rows: uint16(rows), Cols: uint16(cols)})
	_ = controlWrite.Close()
	if err != nil {
		_ = controlRead.Close()
		return AgentPTYResult{}, fmt.Errorf("start interactive command: %w", err)
	}
	session := &agentPTYSession{
		ref: "agent-pty:" + uuid.NewString(), workspaceRoot: root, sessionID: strings.TrimSpace(request.SessionID),
		cwd: cwd, command: command, title: terminalCommandTitle(command), origin: "agent", createdAt: time.Now(), updatedAt: time.Now(), rows: rows, cols: cols, cmd: cmd, ptyFile: ptmx, controlFile: controlRead, status: AgentPTYStatusRunning,
		notify: make(chan struct{}, 1), done: make(chan struct{}), emitter: newShellOutputEmitter(request, ""),
		inputMode: AgentPTYInputAsk, attention: AgentPTYAttentionNone, inputOwner: AgentPTYOwnerNone,
		leaseMode: AgentPTYLeaseNone, lastPromptCursor: -1, subscribers: map[chan AgentPTYEvent]struct{}{}, lastViewedAt: time.Now(), registry: r,
	}
	if session.emitter != nil {
		session.emitter.processRef = session.ref
	}
	r.mu.Lock()
	if r.shutdown {
		r.mu.Unlock()
		_ = killTerminalProcess(cmd.Process)
		_ = ptmx.Close()
		return AgentPTYResult{}, errors.New("agent PTY registry is shutting down")
	}
	r.sessions[session.ref] = session
	r.mu.Unlock()
	readDone := make(chan struct{})
	go session.readLoop(readDone)
	go session.controlLoop()
	go session.waitLoop(readDone)
	session.mu.Lock()
	session.attentionTimer = time.AfterFunc(agentPTYAttentionDelay, func() { session.detectAttention(0) })
	session.mu.Unlock()
	result, waitErr := session.waitForBoundary(ctx, -1, yield, maxOutput)
	if waitErr != nil {
		return result, waitErr
	}
	return result, nil
}

func (r *AgentPTYRegistry) Write(ctx context.Context, input AgentPTYWriteInput) (AgentPTYResult, error) {
	return r.write(ctx, input, "agent")
}

func (r *AgentPTYRegistry) WriteUser(ctx context.Context, input AgentPTYWriteInput) (AgentPTYResult, error) {
	return r.write(ctx, input, "user")
}

func (r *AgentPTYRegistry) WriteUserNow(input AgentPTYWriteInput) (AgentPTYResult, error) {
	session, err := r.owned(input.WorkspaceRoot, input.SessionID, input.ProcessRef)
	if err != nil {
		return AgentPTYResult{}, err
	}
	session.operationMu.Lock()
	defer session.operationMu.Unlock()
	if input.Chars != "" {
		if err := session.writeAs("user", []byte(input.Chars), input.LeaseVersion, input.EnforceLeaseVersion); err != nil {
			return AgentPTYResult{}, err
		}
	}
	return session.snapshot(input.Cursor, agentPTYDefaultMaxChunk), nil
}

func (r *AgentPTYRegistry) write(ctx context.Context, input AgentPTYWriteInput, actor string) (AgentPTYResult, error) {
	session, err := r.owned(input.WorkspaceRoot, input.SessionID, input.ProcessRef)
	if err != nil {
		return AgentPTYResult{}, err
	}
	session.operationMu.Lock()
	defer session.operationMu.Unlock()
	if input.Rows > 0 || input.Cols > 0 {
		if input.Rows <= 0 || input.Cols <= 0 {
			return AgentPTYResult{}, errors.New("rows and cols must be provided together")
		}
		if err := session.resize(input.Rows, input.Cols); err != nil {
			return AgentPTYResult{}, err
		}
	}
	if input.Terminate {
		session.terminate()
	} else if input.Chars != "" {
		if err := session.writeAs(actor, []byte(input.Chars), input.LeaseVersion, input.EnforceLeaseVersion); err != nil {
			return AgentPTYResult{}, err
		}
	}
	result, waitErr := session.waitForBoundary(ctx, input.Cursor, input.YieldTime, input.MaxOutput)
	return result, waitErr
}

func (r *AgentPTYRegistry) ResolveInput(input AgentPTYResolveInput) (AgentPTYResult, error) {
	session, err := r.owned(input.WorkspaceRoot, input.SessionID, input.ProcessRef)
	if err != nil {
		return AgentPTYResult{}, err
	}
	return session.resolveInput(strings.TrimSpace(input.RequestID), strings.TrimSpace(input.Mode))
}

func (r *AgentPTYRegistry) ReleaseInput(workspaceRoot, sessionID, processRef string, leaseVersion int64) (AgentPTYResult, error) {
	session, err := r.owned(workspaceRoot, sessionID, processRef)
	if err != nil {
		return AgentPTYResult{}, err
	}
	session.mu.Lock()
	defer session.mu.Unlock()
	if leaseVersion > 0 && leaseVersion != session.leaseVersion {
		return session.snapshotLocked(session.nextCursor, 0), errors.New("terminal input lease is stale")
	}
	session.inputRequest = nil
	session.releaseLeaseLocked()
	result := session.snapshotLocked(session.nextCursor, 0)
	session.broadcastLocked(AgentPTYEvent{Type: "status", Snapshot: result})
	return result, nil
}

func (r *AgentPTYRegistry) UpdateTitle(workspaceRoot, sessionID, processRef, title string) (AgentPTYResult, error) {
	session, err := r.owned(workspaceRoot, sessionID, processRef)
	if err != nil {
		return AgentPTYResult{}, err
	}
	session.mu.Lock()
	if title = strings.TrimSpace(title); title == "" {
		session.mu.Unlock()
		return AgentPTYResult{}, errors.New("terminal title is required")
	}
	session.title = title
	session.updatedAt = time.Now()
	result := session.snapshotLocked(session.nextCursor, 0)
	session.broadcastLocked(AgentPTYEvent{Type: "status", Snapshot: result})
	session.mu.Unlock()
	return result, nil
}

func (r *AgentPTYRegistry) Remove(workspaceRoot, sessionID, processRef string) error {
	session, err := r.owned(workspaceRoot, sessionID, processRef)
	if err != nil {
		return err
	}
	session.terminate()
	r.mu.Lock()
	delete(r.sessions, processRef)
	r.mu.Unlock()
	return nil
}

func (r *AgentPTYRegistry) Attach(workspaceRoot, sessionID, processRef string, cursor int64) (*AgentPTYAttachment, error) {
	session, err := r.owned(workspaceRoot, sessionID, processRef)
	if err != nil {
		return nil, err
	}
	ch := make(chan AgentPTYEvent, 64)
	session.mu.Lock()
	session.lastViewedAt = time.Now()
	snapshot := session.snapshotLocked(cursor, agentPTYBufferCap)
	exited := session.status == AgentPTYStatusExited
	if !exited {
		session.subscribers[ch] = struct{}{}
	}
	session.mu.Unlock()
	if exited {
		close(ch)
	}
	var once sync.Once
	return &AgentPTYAttachment{Snapshot: snapshot, events: ch, detach: func() {
		once.Do(func() {
			session.mu.Lock()
			if _, ok := session.subscribers[ch]; ok {
				delete(session.subscribers, ch)
				close(ch)
			}
			session.mu.Unlock()
		})
	}}, nil
}

func (r *AgentPTYRegistry) ValidateOwner(workspaceRoot string, sessionID string, processRef string) error {
	_, err := r.owned(workspaceRoot, sessionID, processRef)
	return err
}

func (r *AgentPTYRegistry) List(workspaceRoot, sessionID string) ([]AgentPTYResult, error) {
	root, err := terminalWorkspaceRoot(workspaceRoot)
	if err != nil {
		return nil, err
	}
	r.mu.Lock()
	sessions := make([]*agentPTYSession, 0)
	for _, session := range r.sessions {
		if session.workspaceRoot == root && session.sessionID == strings.TrimSpace(sessionID) {
			sessions = append(sessions, session)
		}
	}
	r.mu.Unlock()
	sort.Slice(sessions, func(i, j int) bool { return sessions[i].createdAt.Before(sessions[j].createdAt) })
	results := make([]AgentPTYResult, 0, len(sessions))
	for _, session := range sessions {
		results = append(results, session.snapshot(-1, 0))
	}
	return results, nil
}

func (r *AgentPTYRegistry) CleanupSession(sessionID string) {
	if r == nil || strings.TrimSpace(sessionID) == "" {
		return
	}
	r.mu.Lock()
	sessions := make([]*agentPTYSession, 0)
	for ref, session := range r.sessions {
		if session.sessionID == strings.TrimSpace(sessionID) {
			delete(r.sessions, ref)
			sessions = append(sessions, session)
		}
	}
	r.mu.Unlock()
	terminatePTYSessions(sessions)
}

func (r *AgentPTYRegistry) Shutdown() {
	if r == nil {
		return
	}
	r.mu.Lock()
	r.shutdown = true
	sessions := make([]*agentPTYSession, 0, len(r.sessions))
	for ref, session := range r.sessions {
		delete(r.sessions, ref)
		sessions = append(sessions, session)
	}
	r.mu.Unlock()
	terminatePTYSessions(sessions)
}

func terminatePTYSessions(sessions []*agentPTYSession) {
	var wait sync.WaitGroup
	wait.Add(len(sessions))
	for _, session := range sessions {
		go func() {
			defer wait.Done()
			session.terminate()
		}()
	}
	wait.Wait()
}

func (r *AgentPTYRegistry) owned(workspaceRoot string, sessionID string, processRef string) (*agentPTYSession, error) {
	if r == nil {
		return nil, errors.New("agent PTY registry is not configured")
	}
	root, err := terminalWorkspaceRoot(workspaceRoot)
	if err != nil {
		return nil, err
	}
	r.mu.Lock()
	session := r.sessions[strings.TrimSpace(processRef)]
	r.mu.Unlock()
	if session == nil || session.workspaceRoot != root || session.sessionID != strings.TrimSpace(sessionID) {
		return nil, errors.New("interactive process not found for this workspace and session")
	}
	return session, nil
}

func (s *agentPTYSession) readLoop(done chan<- struct{}) {
	defer close(done)
	buf := make([]byte, 4096)
	for {
		n, err := s.ptyFile.Read(buf)
		if n > 0 {
			s.appendOutput(buf[:n])
		}
		if err != nil {
			if !errors.Is(err, io.EOF) && !strings.Contains(err.Error(), "input/output error") {
				s.appendOutput([]byte("\r\n[aivo interactive terminal read error]\r\n"))
			}
			return
		}
	}
}

func (s *agentPTYSession) waitLoop(readDone <-chan struct{}) {
	err := s.cmd.Wait()
	<-readDone
	exitCode := 0
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = -1
		}
	}
	s.mu.Lock()
	s.status = AgentPTYStatusExited
	s.updatedAt = time.Now()
	s.exitCode = &exitCode
	cursor := s.nextCursor
	s.inputRequest = nil
	s.broadcastLocked(AgentPTYEvent{Type: "status", Snapshot: s.snapshotLocked(s.nextCursor, 0)})
	for subscriber := range s.subscribers {
		close(subscriber)
		delete(s.subscribers, subscriber)
	}
	s.mu.Unlock()
	if s.registry != nil {
		s.registry.enforceGlobalBufferCap()
	}
	if s.emitter != nil {
		s.emitter.emitWithCursor("stdout", "", cursor, AgentPTYStatusExited, false)
	}
	s.signal()
	s.doneOnce.Do(func() { close(s.done) })
	if s.controlFile != nil {
		_ = s.controlFile.Close()
	}
}

func (r *AgentPTYRegistry) enforceGlobalBufferCap() {
	r.mu.Lock()
	sessions := make([]*agentPTYSession, 0, len(r.sessions))
	for _, session := range r.sessions {
		sessions = append(sessions, session)
	}
	r.mu.Unlock()
	total := 0
	for _, session := range sessions {
		session.mu.Lock()
		total += len(session.buffer)
		session.mu.Unlock()
	}
	cap := r.globalBufferCap
	if cap <= 0 {
		cap = 64 * 1024 * 1024
	}
	if total <= cap {
		return
	}
	sort.Slice(sessions, func(i, j int) bool {
		sessions[i].mu.Lock()
		iExited, iViewed := sessions[i].status == AgentPTYStatusExited, sessions[i].lastViewedAt
		sessions[i].mu.Unlock()
		sessions[j].mu.Lock()
		jExited, jViewed := sessions[j].status == AgentPTYStatusExited, sessions[j].lastViewedAt
		sessions[j].mu.Unlock()
		if iExited != jExited {
			return iExited
		}
		return iViewed.Before(jViewed)
	})
	excess := total - cap
	for _, session := range sessions {
		if excess <= 0 {
			break
		}
		session.mu.Lock()
		drop := len(session.buffer)
		if drop > excess {
			drop = excess
		}
		if drop > 0 {
			session.buffer = append([]byte(nil), session.buffer[drop:]...)
			session.baseCursor += int64(drop)
			excess -= drop
		}
		session.mu.Unlock()
	}
}

func (s *agentPTYSession) appendOutput(data []byte) {
	chunk := append([]byte(nil), data...)
	s.mu.Lock()
	s.buffer = append(s.buffer, chunk...)
	s.nextCursor += int64(len(chunk))
	s.updatedAt = time.Now()
	if len(s.buffer) > agentPTYBufferCap {
		drop := len(s.buffer) - agentPTYBufferCap
		s.buffer = append([]byte(nil), s.buffer[drop:]...)
		s.baseCursor += int64(drop)
	}
	cursor := s.nextCursor
	s.broadcastLocked(AgentPTYEvent{Type: "output", Data: chunk})
	s.attention = AgentPTYAttentionNone
	if s.attentionTimer != nil {
		s.attentionTimer.Stop()
	}
	s.attentionTimer = time.AfterFunc(agentPTYAttentionDelay, func() { s.detectAttention(cursor) })
	s.mu.Unlock()
	if s.registry != nil {
		s.registry.enforceGlobalBufferCap()
	}
	if s.emitter != nil {
		s.emitter.emitWithCursor("stdout", string(chunk), cursor, AgentPTYStatusRunning, false)
	}
	s.signal()
}

func (s *agentPTYSession) signal() {
	select {
	case s.notify <- struct{}{}:
	default:
	}
}

func (s *agentPTYSession) writeAs(actor string, data []byte, expectedLeaseVersion int64, enforceLeaseVersion bool) error {
	s.mu.Lock()
	file := s.ptyFile
	running := s.status != AgentPTYStatusExited
	mode := s.inputMode
	request := cloneAgentPTYInputRequest(s.inputRequest)
	if (enforceLeaseVersion || expectedLeaseVersion > 0) && expectedLeaseVersion != s.leaseVersion {
		current := s.leaseVersion
		s.mu.Unlock()
		return fmt.Errorf("terminal input lease is stale: got %d, current %d", expectedLeaseVersion, current)
	}
	if running && s.inputRequest == nil && s.inputOwner == AgentPTYOwnerNone {
		s.inputOwner = actor
		s.leaseMode = AgentPTYLeaseOnce
		s.leaseVersion++
		if actor == AgentPTYOwnerAgent {
			s.inputMode = AgentPTYInputAgentOnce
		} else {
			s.inputMode = AgentPTYInputUserOnce
		}
		mode = s.inputMode
	}
	authorized := (actor == "agent" && (mode == AgentPTYInputAgentOnce || mode == AgentPTYInputAgentAlways)) ||
		(actor == "user" && mode == AgentPTYInputUserOnce)
	if running && !authorized {
		s.mu.Unlock()
		err := &AgentPTYDecisionRequiredError{ProcessRef: s.ref, Cursor: s.nextCursor, InputMode: mode}
		if request != nil {
			err.RequestID = request.ID
		}
		return err
	}
	s.mu.Unlock()
	if file == nil || !running {
		return errors.New("interactive process is not running")
	}
	if _, err := file.Write(data); err != nil {
		return err
	}
	s.mu.Lock()
	s.attention = AgentPTYAttentionNone
	if actor == AgentPTYOwnerAgent && mode == AgentPTYInputAgentOnce {
		s.clearInputRequestLocked(request)
		s.releaseLeaseLocked()
	} else if actor == AgentPTYOwnerUser && mode == AgentPTYInputUserOnce && bytesContainEnter(data) {
		s.clearInputRequestLocked(request)
		s.releaseLeaseLocked()
	}
	s.broadcastLocked(AgentPTYEvent{Type: "status", Snapshot: s.snapshotLocked(s.nextCursor, 0)})
	s.mu.Unlock()
	return nil
}

func (s *agentPTYSession) clearInputRequestLocked(resolved *AgentPTYInputRequest) {
	if resolved != nil && s.inputRequest != nil && s.inputRequest.ID == resolved.ID {
		s.inputRequest = nil
	}
}

func (s *agentPTYSession) releaseLeaseLocked() {
	s.inputOwner = AgentPTYOwnerNone
	s.leaseMode = AgentPTYLeaseNone
	s.leaseVersion++
	if s.inputMode != AgentPTYInputAgentAlways {
		s.inputMode = AgentPTYInputAsk
	}
	if s.inputRequest == nil {
		s.status = AgentPTYStatusRunning
	}
}

func (s *agentPTYSession) resolveInput(requestID, mode string) (AgentPTYResult, error) {
	if mode != AgentPTYInputAgentOnce && mode != AgentPTYInputUserOnce && mode != AgentPTYInputAgentAlways {
		return AgentPTYResult{}, errors.New("input mode must be agent_once, user_once, or agent_always")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.status == AgentPTYStatusExited {
		return s.snapshotLocked(s.nextCursor, 0), errors.New("interactive process is not running")
	}
	if requestID == "" {
		s.inputMode = mode
		s.status = AgentPTYStatusRunning
		s.inputRequest = nil
		s.leaseVersion++
		s.inputOwner = AgentPTYOwnerAgent
		s.leaseMode = AgentPTYLeaseOnce
		if mode == AgentPTYInputUserOnce {
			s.inputOwner = AgentPTYOwnerUser
		}
		if mode == AgentPTYInputAgentAlways {
			s.leaseMode = AgentPTYLeaseAlways
		}
		result := s.snapshotLocked(s.nextCursor, 0)
		s.broadcastLocked(AgentPTYEvent{Type: "status", Snapshot: result})
		return result, nil
	}
	if s.inputRequest == nil || s.inputRequest.ID != requestID || s.inputRequest.Resolved {
		return s.snapshotLocked(s.nextCursor, 0), errors.New("terminal input request is no longer pending")
	}
	s.inputMode = mode
	s.leaseVersion++
	s.inputOwner = AgentPTYOwnerAgent
	s.leaseMode = AgentPTYLeaseOnce
	if mode == AgentPTYInputUserOnce {
		s.inputOwner = AgentPTYOwnerUser
	}
	if mode == AgentPTYInputAgentAlways {
		s.leaseMode = AgentPTYLeaseAlways
	}
	s.inputRequest.Mode = mode
	s.inputRequest.Resolved = true
	result := s.snapshotLocked(s.nextCursor, 0)
	s.broadcastLocked(AgentPTYEvent{Type: "status", Snapshot: result})
	return result, nil
}

func (s *agentPTYSession) resize(rows int, cols int) error {
	rows, cols = normalizeTerminalSize(rows, cols)
	s.mu.Lock()
	file := s.ptyFile
	running := s.status != AgentPTYStatusExited
	if running {
		s.rows, s.cols = rows, cols
	}
	s.mu.Unlock()
	if file == nil || !running {
		return errors.New("interactive process is not running")
	}
	return pty.Setsize(file, &pty.Winsize{Rows: uint16(rows), Cols: uint16(cols)})
}

func (s *agentPTYSession) terminate() {
	s.mu.Lock()
	process := s.cmd.Process
	file := s.ptyFile
	if s.attentionTimer != nil {
		s.attentionTimer.Stop()
	}
	s.mu.Unlock()
	_ = terminateProcessGroup(process)
	select {
	case <-s.done:
	case <-time.After(2 * time.Second):
		_ = killTerminalProcess(process)
	}
	if file != nil {
		_ = file.Close()
	}
}

func (s *agentPTYSession) waitForBoundary(ctx context.Context, cursor int64, yield time.Duration, maxOutput int) (AgentPTYResult, error) {
	yield = normalizeAgentPTYYield(yield)
	if maxOutput <= 0 {
		maxOutput = agentPTYDefaultMaxChunk
	}
	if maxOutput > agentPTYBufferCap {
		maxOutput = agentPTYBufferCap
	}
	deadline := time.NewTimer(yield)
	defer deadline.Stop()
	var idle *time.Timer
	var idleC <-chan time.Time
	lastProcessCursor := int64(-1)
	pendingInputBoundary := false
	resetIdle := func() {
		if idle == nil {
			idle = time.NewTimer(agentPTYIdleBoundary)
		} else {
			if !idle.Stop() {
				select {
				case <-idle.C:
				default:
				}
			}
			idle.Reset(agentPTYIdleBoundary)
		}
		idleC = idle.C
	}
	defer func() {
		if idle != nil {
			idle.Stop()
		}
	}()
	for {
		result := s.snapshot(cursor, maxOutput)
		if result.Status == AgentPTYStatusExited {
			result.YieldReason = "exited"
			return result, nil
		}
		if result.Status == AgentPTYStatusWaitingInput && result.InputRequest != nil && !result.InputRequest.Resolved {
			if strings.TrimSpace(result.InputRequest.Prompt) != "" || result.ProcessCursor > result.InputRequest.Cursor {
				result.YieldReason = "input_request"
				return result, nil
			}
			if !pendingInputBoundary {
				resetIdle()
				pendingInputBoundary = true
			}
		} else {
			pendingInputBoundary = false
		}
		if result.OutputTruncated && len(result.Output) >= maxOutput {
			result.YieldReason = "output_limit"
			return result, nil
		}
		if result.ProcessCursor != lastProcessCursor {
			if lastProcessCursor >= 0 || result.Output != "" {
				resetIdle()
			}
			lastProcessCursor = result.ProcessCursor
		}
		select {
		case <-ctx.Done():
			result.YieldReason = "cancelled"
			return result, ctx.Err()
		case <-deadline.C:
			result = s.snapshot(cursor, maxOutput)
			result.YieldReason = "timeout"
			return result, nil
		case <-idleC:
			result = s.snapshot(cursor, maxOutput)
			if result.Status == AgentPTYStatusWaitingInput && result.InputRequest != nil && !result.InputRequest.Resolved {
				result.YieldReason = "input_request"
			} else {
				result.YieldReason = "output_idle"
			}
			return result, nil
		case <-s.notify:
		}
	}
}

func (s *agentPTYSession) snapshot(cursor int64, maxOutput int) AgentPTYResult {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.snapshotLocked(cursor, maxOutput)
}

func (s *agentPTYSession) snapshotLocked(cursor int64, maxOutput int) AgentPTYResult {
	if maxOutput <= 0 {
		maxOutput = agentPTYBufferCap
	}
	start := cursor
	truncated := false
	if start < 0 {
		start = s.baseCursor
	}
	if start < s.baseCursor {
		start = s.baseCursor
		truncated = true
	}
	if start > s.nextCursor {
		start = s.nextCursor
	}
	offset := int(start - s.baseCursor)
	available := s.buffer[offset:]
	if len(available) > maxOutput {
		available = available[:maxOutput]
		truncated = true
	}
	output := redactCommandOutput(string(available))
	result := AgentPTYResult{
		ProcessRef: s.ref, Status: s.status, CWD: filepath.ToSlash(s.cwd), Rows: s.rows, Cols: s.cols,
		Cursor: start + int64(len(available)), ProcessCursor: s.nextCursor, BaseCursor: s.baseCursor,
		Output: output, OutputTruncated: truncated, ExitCode: s.exitCode,
		InputMode: s.inputMode, InputRequest: cloneAgentPTYInputRequest(s.inputRequest),
		Attention: s.attention, InputOwner: s.inputOwner, LeaseMode: s.leaseMode, LeaseVersion: s.leaseVersion,
		SessionID: s.sessionID, Origin: s.origin, Title: s.title, Command: s.command,
		CreatedAt: domain.NowString(s.createdAt), UpdatedAt: domain.NowString(s.updatedAt),
	}
	if s.cmd != nil && s.cmd.Process != nil {
		result.PID = s.cmd.Process.Pid
	}
	return result
}

func terminalCommandTitle(command string) string {
	command = strings.Join(strings.Fields(command), " ")
	if len(command) > 80 {
		command = command[:77] + "..."
	}
	if command == "" {
		return "Terminal"
	}
	return command
}

func (s *agentPTYSession) markInputBoundary(cursor int64) AgentPTYResult {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.markInputBoundaryLocked(cursor)
	return s.snapshotLocked(-1, agentPTYDefaultMaxChunk)
}

func (s *agentPTYSession) markInputBoundaryLocked(cursor int64) {
	if s.inputMode == AgentPTYInputAsk && s.lastPromptCursor != cursor {
		s.lastPromptCursor = cursor
		s.status = AgentPTYStatusWaitingInput
		s.inputRequest = &AgentPTYInputRequest{ID: "input:" + uuid.NewString(), Cursor: cursor, Mode: AgentPTYInputAsk, CreatedAt: domain.NowString(time.Now())}
		s.broadcastLocked(AgentPTYEvent{Type: "input_request", Snapshot: s.snapshotLocked(cursor, 0)})
	}
}

func (s *agentPTYSession) detectAttention(expectedCursor int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.status == AgentPTYStatusExited || s.nextCursor != expectedCursor {
		return
	}
	if terminalInteractiveMode(s.ptyFile) {
		s.attention = AgentPTYAttentionInteractive
	} else {
		s.attention = AgentPTYAttentionPossiblyWaiting
	}
	s.broadcastLocked(AgentPTYEvent{Type: "attention", Snapshot: s.snapshotLocked(expectedCursor, 0)})
	s.signal()
}

func (s *agentPTYSession) broadcastLocked(event AgentPTYEvent) {
	for subscriber := range s.subscribers {
		select {
		case subscriber <- event:
		default:
		}
	}
}

func cloneAgentPTYInputRequest(value *AgentPTYInputRequest) *AgentPTYInputRequest {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

func bytesContainEnter(data []byte) bool {
	for _, value := range data {
		if value == '\r' || value == '\n' {
			return true
		}
	}
	return false
}

func normalizeAgentPTYYield(value time.Duration) time.Duration {
	if value <= 0 {
		return agentPTYDefaultYield
	}
	if value < 100*time.Millisecond {
		return 100 * time.Millisecond
	}
	if value > agentPTYMaxYield {
		return agentPTYMaxYield
	}
	return value
}

type agentPTYControlMessage struct {
	Version int    `json:"v"`
	Type    string `json:"type"`
	ID      string `json:"id"`
	Prompt  string `json:"prompt"`
	Secret  bool   `json:"secret"`
}

func (s *agentPTYSession) controlLoop() {
	if s.controlFile == nil {
		return
	}
	scanner := bufio.NewScanner(s.controlFile)
	scanner.Buffer(make([]byte, 1024), agentPTYControlMaxLine)
	for scanner.Scan() {
		var message agentPTYControlMessage
		if json.Unmarshal(scanner.Bytes(), &message) != nil || message.Version != 1 {
			continue
		}
		s.mu.Lock()
		switch message.Type {
		case "input_request":
			if strings.TrimSpace(message.ID) != "" && s.status != AgentPTYStatusExited {
				s.status = AgentPTYStatusWaitingInput
				s.inputOwner = AgentPTYOwnerNone
				s.leaseMode = AgentPTYLeaseNone
				s.inputMode = AgentPTYInputAsk
				s.leaseVersion++
				s.inputRequest = &AgentPTYInputRequest{ID: strings.TrimSpace(message.ID), Cursor: s.nextCursor, Mode: AgentPTYInputAsk, CreatedAt: domain.NowString(time.Now()), Prompt: message.Prompt, Secret: message.Secret}
				s.broadcastLocked(AgentPTYEvent{Type: "input_request", Snapshot: s.snapshotLocked(s.nextCursor, 0)})
			}
		case "input_clear":
			if s.inputRequest != nil && (message.ID == "" || message.ID == s.inputRequest.ID) {
				s.inputRequest = nil
				s.status = AgentPTYStatusRunning
				s.releaseLeaseLocked()
				s.broadcastLocked(AgentPTYEvent{Type: "status", Snapshot: s.snapshotLocked(s.nextCursor, 0)})
			}
		}
		s.mu.Unlock()
		s.signal()
	}
}
