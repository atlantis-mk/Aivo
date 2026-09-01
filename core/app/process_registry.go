package app

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"aivo/core/domain"
)

const backgroundOutputMaxChars = 64 * 1024

type ShellProcessInfo struct {
	ID            string `json:"id"`
	SessionID     string `json:"sessionId,omitempty"`
	TurnID        string `json:"turnId,omitempty"`
	ToolCallID    string `json:"toolCallId,omitempty"`
	WorkspaceRoot string `json:"workspaceRoot"`
	Command       string `json:"command"`
	CWD           string `json:"cwd"`
	Backend       string `json:"backend"`
	PID           int    `json:"pid"`
	Status        string `json:"status"`
	ExitCode      *int   `json:"exitCode,omitempty"`
	Stdout        string `json:"stdout,omitempty"`
	Stderr        string `json:"stderr,omitempty"`
	StdoutRef     string `json:"stdoutRef,omitempty"`
	StderrRef     string `json:"stderrRef,omitempty"`
	TimeCreated   string `json:"timeCreated"`
	TimeUpdated   string `json:"timeUpdated"`
}

type shellProcessRecord struct {
	mu      sync.Mutex
	info    ShellProcessInfo
	cmd     *exec.Cmd
	stdout  bytes.Buffer
	stderr  bytes.Buffer
	done    chan struct{}
	waitErr error
}

type ShellProcessRegistry struct {
	mu        sync.Mutex
	processes map[string]*shellProcessRecord
}

func NewShellProcessRegistry() *ShellProcessRegistry {
	return &ShellProcessRegistry{processes: map[string]*shellProcessRecord{}}
}

var defaultShellProcessRegistry = NewShellProcessRegistry()

func (r *ShellProcessRegistry) Start(ctx context.Context, request SandboxRequest) (SandboxResult, error) {
	if r == nil {
		return SandboxResult{}, errors.New("process registry is not configured")
	}
	workspaceRoot, cwd, err := normalizeSandboxCWD(request.WorkspaceRoot, request.CWD, request.AllowExternalCWD)
	if err != nil {
		return SandboxResult{}, &SandboxError{Code: SandboxErrorPolicyDenied, Message: err.Error(), Err: err}
	}
	command := strings.TrimSpace(request.Command)
	if command == "" {
		return SandboxResult{}, &SandboxError{Code: SandboxErrorPolicyDenied, Message: "command is required"}
	}
	shell := firstNonEmpty(request.Shell, defaultShell())
	argv := shellCommandArgs(shell, command)
	cmd := exec.CommandContext(context.Background(), shell, argv...)
	cmd.Dir = cwd
	cmd.Env = SanitizedEnvironment(workspaceRoot, request.EnvAllowlist, request.Env, request.EnvOverrides)
	if request.Stdin != "" {
		cmd.Stdin = bytes.NewBufferString(request.Stdin)
	}
	setProcessGroup(cmd)
	if err := ctx.Err(); err != nil {
		return SandboxResult{}, err
	}
	now := time.Now()
	id := uuid.NewString()
	record := &shellProcessRecord{
		cmd:  cmd,
		done: make(chan struct{}),
		info: ShellProcessInfo{
			ID: id, SessionID: request.SessionID, TurnID: request.TurnID, ToolCallID: request.ToolCallID,
			WorkspaceRoot: workspaceRoot, Command: command, CWD: cwd, Backend: "local",
			Status: "running", TimeCreated: domain.NowString(now), TimeUpdated: domain.NowString(now),
		},
	}
	emitter := newShellOutputEmitter(request, id)
	cmd.Stdout = &boundedBufferWriter{record: record, target: &record.stdout, limit: backgroundOutputMaxChars, emitter: emitter, stream: "stdout"}
	cmd.Stderr = &boundedBufferWriter{record: record, target: &record.stderr, limit: backgroundOutputMaxChars, emitter: emitter, stream: "stderr"}
	if err := cmd.Start(); err != nil {
		return SandboxResult{}, err
	}
	record.info.PID = cmd.Process.Pid
	r.mu.Lock()
	r.processes[id] = record
	r.mu.Unlock()
	go r.wait(id, record, request)
	return SandboxResult{
		Command: command, Argv: append([]string{shell}, argv...), Mode: "background",
		CWD: cwd, ExitCode: 0, Stdout: "", Stderr: "", Backend: "local", NetworkPolicy: request.NetworkPolicy,
		ProcessID: cmd.Process.Pid, ProcessRef: id,
	}, nil
}

type boundedBufferWriter struct {
	record  *shellProcessRecord
	target  *bytes.Buffer
	limit   int
	emitter *shellOutputEmitter
	stream  string
}

func (w *boundedBufferWriter) Write(p []byte) (int, error) {
	w.record.mu.Lock()
	defer w.record.mu.Unlock()
	remaining := w.limit - w.target.Len()
	if remaining > 0 {
		if len(p) > remaining {
			_, _ = w.target.Write(p[:remaining])
		} else {
			_, _ = w.target.Write(p)
		}
	}
	if len(p) > 0 && w.emitter != nil {
		w.emitter.emit(w.stream, string(p))
	}
	return len(p), nil
}

func (r *ShellProcessRegistry) wait(id string, record *shellProcessRecord, request SandboxRequest) {
	err := record.cmd.Wait()
	record.mu.Lock()
	record.waitErr = err
	exitCode := 0
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = -1
		}
	}
	record.info.ExitCode = &exitCode
	record.info.Status = "exited"
	record.info.Stdout = redactCommandOutput(record.stdout.String())
	record.info.Stderr = redactCommandOutput(record.stderr.String())
	record.info.TimeUpdated = domain.NowString(time.Now())
	record.mu.Unlock()
	close(record.done)
}

func (r *ShellProcessRegistry) Get(id string) (ShellProcessInfo, error) {
	record, err := r.record(id)
	if err != nil {
		return ShellProcessInfo{}, err
	}
	record.mu.Lock()
	defer record.mu.Unlock()
	info := record.info
	info.Stdout = redactCommandOutput(record.stdout.String())
	info.Stderr = redactCommandOutput(record.stderr.String())
	return info, nil
}

func (r *ShellProcessRegistry) Poll(id string) (ShellProcessInfo, error) {
	return r.Get(id)
}

func (r *ShellProcessRegistry) Wait(ctx context.Context, id string) (ShellProcessInfo, error) {
	record, err := r.record(id)
	if err != nil {
		return ShellProcessInfo{}, err
	}
	select {
	case <-ctx.Done():
		return r.Get(id)
	case <-record.done:
		return r.Get(id)
	}
}

func (r *ShellProcessRegistry) Kill(id string) (ShellProcessInfo, error) {
	record, err := r.record(id)
	if err != nil {
		return ShellProcessInfo{}, err
	}
	if record.cmd != nil && record.cmd.Process != nil {
		_ = killProcessGroup(record.cmd.Process)
	}
	return r.Wait(context.Background(), id)
}

func (r *ShellProcessRegistry) ReadOutput(id string) (ShellProcessInfo, error) {
	return r.Get(id)
}

func (r *ShellProcessRegistry) CleanupSession(sessionID string) {
	r.mu.Lock()
	records := make([]*shellProcessRecord, 0)
	for _, record := range r.processes {
		record.mu.Lock()
		matches := record.info.SessionID == sessionID && record.info.Status == "running"
		record.mu.Unlock()
		if matches {
			records = append(records, record)
		}
	}
	r.mu.Unlock()
	for _, record := range records {
		if record.cmd != nil && record.cmd.Process != nil {
			_ = killProcessGroup(record.cmd.Process)
		}
	}
}

func (r *ShellProcessRegistry) Shutdown() {
	r.mu.Lock()
	records := make([]*shellProcessRecord, 0, len(r.processes))
	for _, record := range r.processes {
		records = append(records, record)
	}
	r.mu.Unlock()
	for _, record := range records {
		record.mu.Lock()
		running := record.info.Status == "running"
		record.mu.Unlock()
		if running && record.cmd != nil && record.cmd.Process != nil {
			_ = killProcessGroup(record.cmd.Process)
		}
	}
}

func (r *ShellProcessRegistry) record(id string) (*shellProcessRecord, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	record := r.processes[id]
	if record == nil {
		return nil, fmt.Errorf("unknown process ref %q", id)
	}
	return record, nil
}

func retainProcessOutput(request SandboxRequest, info ShellProcessInfo) ShellProcessInfo {
	if info.Stdout != "" {
		if ref, err := retainSandboxOutput(request, "stdout", info.Stdout); err == nil {
			info.StdoutRef = ref
		}
	}
	if info.Stderr != "" {
		if ref, err := retainSandboxOutput(request, "stderr", info.Stderr); err == nil {
			info.StderrRef = ref
		}
	}
	return info
}

func processInfoToResult(info ShellProcessInfo) SandboxResult {
	exitCode := 0
	if info.ExitCode != nil {
		exitCode = *info.ExitCode
	}
	return SandboxResult{
		Command: info.Command, Mode: "background", CWD: info.CWD, ExitCode: exitCode,
		Stdout: info.Stdout, Stderr: info.Stderr, Backend: info.Backend,
		ProcessID: info.PID, ProcessRef: info.ID, StdoutRef: info.StdoutRef, StderrRef: info.StderrRef,
	}
}
