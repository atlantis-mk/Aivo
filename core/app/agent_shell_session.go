package app

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

type AgentShellRegistry struct {
	mu       sync.Mutex
	sessions map[string]*agentShellSession
}

type agentShellSession struct {
	mu            sync.Mutex
	key           string
	ref           string
	workspaceRoot string
	sessionID     string
	cwd           string
	shell         string
	cmd           *exec.Cmd
	stdin         io.WriteCloser
	active        *agentShellCommand
	exited        bool
	exitCode      int
	waitErr       error
}

type agentShellCommand struct {
	token           string
	request         SandboxRequest
	emitter         *shellOutputEmitter
	stdout          bytes.Buffer
	stderr          bytes.Buffer
	stdoutPending   string
	stdoutRetainLen int
	done            chan agentShellCommandDone
	start           time.Time
}

type agentShellCommandDone struct {
	exitCode int
	cwd      string
	err      error
}

func NewAgentShellRegistry() *AgentShellRegistry {
	return &AgentShellRegistry{sessions: map[string]*agentShellSession{}}
}

var defaultAgentShellRegistry = NewAgentShellRegistry()

func (r *AgentShellRegistry) CurrentCWD(sessionID string, workspaceRoot string) string {
	if r == nil || strings.TrimSpace(sessionID) == "" {
		return ""
	}
	key, err := agentShellKey(sessionID, workspaceRoot)
	if err != nil {
		return ""
	}
	r.mu.Lock()
	session := r.sessions[key]
	r.mu.Unlock()
	if session == nil {
		return ""
	}
	session.mu.Lock()
	defer session.mu.Unlock()
	if session.exited {
		return ""
	}
	return session.cwd
}

func (r *AgentShellRegistry) Run(ctx context.Context, request SandboxRequest) (SandboxResult, error) {
	if r == nil {
		return SandboxResult{}, errors.New("agent shell registry is not configured")
	}
	session, err := r.session(request)
	if err != nil {
		return SandboxResult{}, err
	}
	return session.run(ctx, request)
}

func (r *AgentShellRegistry) CleanupSession(sessionID string) {
	if r == nil || strings.TrimSpace(sessionID) == "" {
		return
	}
	r.mu.Lock()
	sessions := make([]*agentShellSession, 0)
	for key, session := range r.sessions {
		if session.sessionID == sessionID {
			delete(r.sessions, key)
			sessions = append(sessions, session)
		}
	}
	r.mu.Unlock()
	for _, session := range sessions {
		session.kill()
	}
}

func (r *AgentShellRegistry) Shutdown() {
	if r == nil {
		return
	}
	r.mu.Lock()
	sessions := make([]*agentShellSession, 0, len(r.sessions))
	for key, session := range r.sessions {
		delete(r.sessions, key)
		sessions = append(sessions, session)
	}
	r.mu.Unlock()
	for _, session := range sessions {
		session.kill()
	}
}

func (r *AgentShellRegistry) session(request SandboxRequest) (*agentShellSession, error) {
	workspaceRoot, cwd, err := normalizeSandboxCWD(request.WorkspaceRoot, request.CWD, request.AllowExternalCWD)
	if err != nil {
		return nil, &SandboxError{Code: SandboxErrorPolicyDenied, Message: err.Error(), Err: err}
	}
	key, err := agentShellKey(request.SessionID, workspaceRoot)
	if err != nil {
		return nil, err
	}
	r.mu.Lock()
	session := r.sessions[key]
	if session == nil || session.isExited() {
		if session != nil {
			delete(r.sessions, key)
		}
		session, err = startAgentShellSession(key, request.SessionID, workspaceRoot, cwd, request)
		if err != nil {
			r.mu.Unlock()
			return nil, err
		}
		r.sessions[key] = session
	}
	r.mu.Unlock()
	return session, nil
}

func agentShellKey(sessionID string, workspaceRoot string) (string, error) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return "", errors.New("sessionId is required for persistent agent shell")
	}
	root, err := filepath.Abs(strings.TrimSpace(workspaceRoot))
	if err != nil {
		return "", err
	}
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		return "", err
	}
	return sessionID + "\x00" + root, nil
}

func startAgentShellSession(key string, sessionID string, workspaceRoot string, cwd string, request SandboxRequest) (*agentShellSession, error) {
	shell := firstNonEmpty(strings.TrimSpace(request.Shell), defaultShell())
	cmd := exec.Command(shell)
	cmd.Dir = cwd
	cmd.Env = SanitizedEnvironment(workspaceRoot, request.EnvAllowlist, request.Env, nil)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		_ = stdin.Close()
		return nil, err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		_ = stdin.Close()
		return nil, err
	}
	setProcessGroup(cmd)
	if err := cmd.Start(); err != nil {
		_ = stdin.Close()
		return nil, err
	}
	session := &agentShellSession{
		key: key, ref: "agent-shell:" + safeArtifactPart(sessionID), workspaceRoot: workspaceRoot, sessionID: sessionID, cwd: cwd,
		shell: shell, cmd: cmd, stdin: stdin, exitCode: -1,
	}
	go session.readLoop(stdout, "stdout")
	go session.readLoop(stderr, "stderr")
	go session.wait()
	return session, nil
}

func (s *agentShellSession) isExited() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.exited
}

func (s *agentShellSession) run(ctx context.Context, request SandboxRequest) (SandboxResult, error) {
	timeout := clampCommandTimeout(request.Timeout)
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	command := strings.TrimSpace(request.Command)
	if command == "" {
		return SandboxResult{}, &SandboxError{Code: SandboxErrorPolicyDenied, Message: "command is required"}
	}
	token := strings.ReplaceAll(uuid.NewString(), "-", "")
	cmdState := &agentShellCommand{
		token: token, request: request, start: time.Now(), done: make(chan agentShellCommandDone, 1),
		emitter: newShellOutputEmitter(request, s.ref),
	}
	cmdState.stdoutRetainLen = len(agentShellDonePrefix(token)) - 1

	s.mu.Lock()
	if s.exited {
		err := firstNonNilError(s.waitErr, errors.New("agent shell has exited"))
		s.mu.Unlock()
		return SandboxResult{Command: command, Mode: "foreground", CWD: request.CWD, ExitCode: s.exitCode, Backend: "local"}, err
	}
	if s.active != nil {
		s.mu.Unlock()
		return SandboxResult{}, errors.New("agent shell already has a running command")
	}
	startCWD := s.cwd
	s.active = cmdState
	script := agentShellCommandScript(token, command, startCWD, request.CWD)
	_, writeErr := io.WriteString(s.stdin, script)
	s.mu.Unlock()
	if writeErr != nil {
		s.finishActive(agentShellCommandDone{exitCode: -1, cwd: startCWD, err: writeErr})
		return SandboxResult{Command: command, Mode: "foreground", CWD: startCWD, ExitCode: -1, Backend: "local"}, writeErr
	}

	var done agentShellCommandDone
	select {
	case <-runCtx.Done():
		s.kill()
		done = agentShellCommandDone{exitCode: -1, cwd: startCWD, err: runCtx.Err()}
	case done = <-cmdState.done:
	}
	duration := time.Since(cmdState.start)
	stdoutText, stderrText, truncated, originalSize, stdoutRef, stderrRef, retainErr := finalizeAgentShellOutput(request, cmdState)
	if retainErr != nil {
		return SandboxResult{}, retainErr
	}
	result := SandboxResult{
		Command: command, Argv: []string{s.shell, "-c", command}, Mode: "foreground",
		CWD: firstNonEmpty(done.cwd, request.CWD, startCWD), ExitCode: done.exitCode,
		Stdout: stdoutText, Stderr: stderrText, Backend: "local",
		NetworkPolicy: firstNonEmpty(request.NetworkPolicy, "deny"),
		ProcessID:     s.pid(), ProcessRef: s.ref, Duration: duration,
		Truncated: truncated, OriginalSize: originalSize, StdoutRef: stdoutRef, StderrRef: stderrRef,
	}
	if errors.Is(done.err, context.DeadlineExceeded) {
		result.TimedOut = true
		done.err = &SandboxError{Code: SandboxErrorCommandTimeout, Message: "command timed out", Err: done.err}
	} else if errors.Is(done.err, context.Canceled) {
		result.Cancelled = true
		done.err = &SandboxError{Code: SandboxErrorCommandCancelled, Message: "command cancelled", Err: done.err}
	}
	return result, done.err
}

func (s *agentShellSession) pid() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cmd == nil || s.cmd.Process == nil {
		return 0
	}
	return s.cmd.Process.Pid
}

func (s *agentShellSession) readLoop(reader io.Reader, stream string) {
	buf := make([]byte, 4096)
	for {
		n, err := reader.Read(buf)
		if n > 0 {
			s.appendOutput(stream, string(buf[:n]))
		}
		if err != nil {
			return
		}
	}
}

func (s *agentShellSession) appendOutput(stream string, chunk string) {
	s.mu.Lock()
	active := s.active
	if active == nil {
		s.mu.Unlock()
		return
	}
	if stream == "stderr" {
		active.stderr.WriteString(chunk)
		active.emit("stderr", chunk)
		s.mu.Unlock()
		return
	}
	done, finished := active.appendStdout(chunk)
	if finished {
		s.cwd = done.cwd
		s.active = nil
	}
	s.mu.Unlock()
	if finished {
		active.done <- done
	}
}

func (s *agentShellSession) wait() {
	err := s.cmd.Wait()
	exitCode := 0
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = -1
		}
	}
	s.finishActive(agentShellCommandDone{exitCode: exitCode, cwd: s.cwd, err: err})
	s.mu.Lock()
	s.exited = true
	s.exitCode = exitCode
	s.waitErr = err
	s.mu.Unlock()
}

func (s *agentShellSession) finishActive(done agentShellCommandDone) {
	s.mu.Lock()
	active := s.active
	if active != nil {
		active.flushStdoutPending()
		s.active = nil
	}
	s.mu.Unlock()
	if active != nil {
		active.done <- done
	}
}

func (s *agentShellSession) kill() {
	s.mu.Lock()
	cmd := s.cmd
	s.mu.Unlock()
	if cmd != nil && cmd.Process != nil {
		_ = killProcessGroup(cmd.Process)
	}
}

func (c *agentShellCommand) appendStdout(chunk string) (agentShellCommandDone, bool) {
	c.stdoutPending += chunk
	prefix := agentShellDonePrefix(c.token)
	if idx := strings.Index(c.stdoutPending, prefix); idx >= 0 {
		c.emitStdout(c.stdoutPending[:idx])
		rest := c.stdoutPending[idx+len(prefix):]
		line, _, _ := strings.Cut(rest, "\n")
		exitCodeText, cwd, _ := strings.Cut(line, ":")
		exitCode, err := strconv.Atoi(strings.TrimSpace(exitCodeText))
		if err != nil {
			exitCode = -1
		}
		c.stdoutPending = ""
		var runErr error
		if exitCode != 0 {
			runErr = fmt.Errorf("command exited with status %d", exitCode)
		}
		return agentShellCommandDone{exitCode: exitCode, cwd: strings.TrimSpace(cwd), err: runErr}, true
	}
	emitLength := len(c.stdoutPending) - c.stdoutRetainLen
	if emitLength > 0 {
		c.emitStdout(c.stdoutPending[:emitLength])
		c.stdoutPending = c.stdoutPending[emitLength:]
	}
	return agentShellCommandDone{}, false
}

func (c *agentShellCommand) flushStdoutPending() {
	if c.stdoutPending == "" {
		return
	}
	c.emitStdout(c.stdoutPending)
	c.stdoutPending = ""
}

func (c *agentShellCommand) emitStdout(chunk string) {
	if chunk == "" {
		return
	}
	c.stdout.WriteString(chunk)
	c.emit("stdout", chunk)
}

func (c *agentShellCommand) emit(stream string, chunk string) {
	if c.emitter != nil && chunk != "" {
		c.emitter.emit(stream, chunk)
	}
}

func agentShellDonePrefix(token string) string {
	return "\n__AIVO_CMD_DONE_" + token + ":"
}

func agentShellCommandScript(token string, command string, currentCWD string, requestedCWD string) string {
	var b strings.Builder
	if strings.TrimSpace(requestedCWD) != "" && requestedCWD != currentCWD {
		b.WriteString("cd ")
		b.WriteString(shellSingleQuote(requestedCWD))
		b.WriteString("\n")
	}
	b.WriteString("{\n")
	b.WriteString(command)
	b.WriteString("\n}\n")
	b.WriteString("__aivo_status=$?\n")
	b.WriteString("printf '\\n__AIVO_CMD_DONE_")
	b.WriteString(token)
	b.WriteString(":%s:%s\\n' \"$__aivo_status\" \"$(pwd -P)\"\n")
	return b.String()
}

func finalizeAgentShellOutput(request SandboxRequest, command *agentShellCommand) (string, string, bool, int, string, string, error) {
	maxChars := request.OutputPolicy.MaxChars
	if maxChars <= 0 {
		maxChars = defaultStreamMaxChars
	}
	stdoutOriginal := command.stdout.Len()
	stderrOriginal := command.stderr.Len()
	stdoutText, stdoutTruncated, stdoutRef, _, err := boundedSandboxOutput(request, "stdout", command.stdout.String(), maxChars)
	if err != nil {
		return "", "", false, 0, "", "", err
	}
	stderrText, stderrTruncated, stderrRef, _, err := boundedSandboxOutput(request, "stderr", command.stderr.String(), maxChars)
	if err != nil {
		return "", "", false, 0, "", "", err
	}
	return redactCommandOutput(stdoutText), redactCommandOutput(stderrText), stdoutTruncated || stderrTruncated, stdoutOriginal + stderrOriginal, stdoutRef, stderrRef, nil
}

func firstNonNilError(values ...error) error {
	for _, value := range values {
		if value != nil {
			return value
		}
	}
	return nil
}
