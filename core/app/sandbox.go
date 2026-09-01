package app

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"aivo/core/domain"
)

const (
	SandboxErrorUnavailable           = "sandbox_unavailable"
	SandboxErrorPolicyDenied          = "sandbox_policy_denied"
	SandboxErrorCommandTimeout        = "command_timeout"
	SandboxErrorCommandCancelled      = "command_cancelled"
	SandboxErrorProcessCleanupFailed  = "process_cleanup_failed"
	SandboxErrorOutputRetentionFailed = "output_retention_failed"

	defaultCommandTimeout = 30 * time.Second
	maxCommandTimeout     = 5 * time.Minute
	defaultStreamMaxChars = 12000
)

type SandboxRunner interface {
	Run(ctx context.Context, request SandboxRequest) (SandboxResult, error)
}

type SandboxRequest struct {
	WorkspaceRoot    string
	CWD              string
	Command          string
	Argv             []string
	Mode             string
	Timeout          time.Duration
	Stdin            string
	Env              map[string]string
	EnvOverrides     map[string]string
	EnvAllowlist     []string
	NetworkPolicy    string
	Backend          string
	Shell            string
	LoginShell       bool
	OutputPolicy     domain.OutputPolicy
	SessionID        string
	TurnID           string
	ToolCallID       string
	ToolName         string
	ApprovalKey      string
	AllowExternalCWD bool
	OutputSink       ShellOutputSink
}

type SandboxResult struct {
	Command       string
	Argv          []string
	Mode          string
	CWD           string
	ExitCode      int
	Stdout        string
	Stderr        string
	TimedOut      bool
	Cancelled     bool
	Truncated     bool
	OriginalSize  int
	StdoutRef     string
	StderrRef     string
	Duration      time.Duration
	Backend       string
	NetworkPolicy string
	ProcessID     int
	ProcessRef    string
}

type SandboxError struct {
	Code    string
	Message string
	Err     error
}

type ShellOutputEvent struct {
	SessionID   string `json:"sessionId,omitempty"`
	TurnID      string `json:"turnId,omitempty"`
	ToolCallID  string `json:"toolCallId,omitempty"`
	ProcessRef  string `json:"processRef,omitempty"`
	Stream      string `json:"stream"`
	Chunk       string `json:"chunk"`
	Sequence    int64  `json:"sequence"`
	Cursor      int64  `json:"cursor,omitempty"`
	Status      string `json:"status,omitempty"`
	Truncated   bool   `json:"truncated,omitempty"`
	TimeCreated string `json:"timeCreated"`
}

type ShellOutputSink func(ShellOutputEvent)

type shellOutputEmitter struct {
	mu         sync.Mutex
	nextSeq    int64
	processRef string
	request    SandboxRequest
}

type shellOutputWriter struct {
	target  *bytes.Buffer
	emitter *shellOutputEmitter
	stream  string
}

func (e *SandboxError) Error() string {
	if e == nil {
		return ""
	}
	if e.Message != "" {
		return e.Message
	}
	return e.Code
}

func (e *SandboxError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

type LocalSandboxRunner struct{}

func NewLocalSandboxRunner() *LocalSandboxRunner {
	return &LocalSandboxRunner{}
}

func (r *LocalSandboxRunner) Run(ctx context.Context, request SandboxRequest) (SandboxResult, error) {
	mode := normalizeSandboxMode(request.Mode)
	if mode != "foreground" {
		return SandboxResult{Command: request.Command, Mode: mode, ExitCode: -1, Backend: "local"}, &SandboxError{Code: SandboxErrorPolicyDenied, Message: "local sandbox Run only executes foreground commands; use process registry for background or PTY mode"}
	}
	start := time.Now()
	workspaceRoot, cwd, err := normalizeSandboxCWD(request.WorkspaceRoot, request.CWD, request.AllowExternalCWD)
	if err != nil {
		return SandboxResult{}, &SandboxError{Code: SandboxErrorPolicyDenied, Message: err.Error(), Err: err}
	}
	timeout := clampCommandTimeout(request.Timeout)
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	shell := strings.TrimSpace(request.Shell)
	if shell == "" {
		shell = defaultShell()
	}
	command := strings.TrimSpace(request.Command)
	if command == "" {
		return SandboxResult{}, &SandboxError{Code: SandboxErrorPolicyDenied, Message: "command is required"}
	}

	argv := shellCommandArgs(shell, command, request.LoginShell)
	cmd := exec.CommandContext(runCtx, shell, argv...)
	cmd.Dir = cwd
	cmd.Env = SanitizedEnvironment(workspaceRoot, request.EnvAllowlist, request.Env, request.EnvOverrides)
	if request.Stdin != "" {
		cmd.Stdin = strings.NewReader(request.Stdin)
	} else {
		cmd.Stdin = nil
	}
	setProcessGroup(cmd)
	cmd.Cancel = func() error {
		if cmd.Process == nil {
			return os.ErrProcessDone
		}
		if err := killProcessGroup(cmd.Process); err != nil {
			_ = cmd.Process.Kill()
		}
		return nil
	}
	cmd.WaitDelay = 250 * time.Millisecond

	emitter := newShellOutputEmitter(request, "")
	maxChars := request.OutputPolicy.MaxChars
	if maxChars <= 0 {
		maxChars = defaultStreamMaxChars
	}
	stdout, err := newBoundedOutputCapture(request, "stdout", maxChars, emitter)
	if err != nil {
		return SandboxResult{}, &SandboxError{Code: SandboxErrorOutputRetentionFailed, Message: err.Error(), Err: err}
	}
	stderr, err := newBoundedOutputCapture(request, "stderr", maxChars, emitter)
	if err != nil {
		stdout.Abort()
		return SandboxResult{}, &SandboxError{Code: SandboxErrorOutputRetentionFailed, Message: err.Error(), Err: err}
	}
	defer stdout.Abort()
	defer stderr.Abort()
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	result := SandboxResult{
		Command:       command,
		Argv:          append([]string{shell}, argv...),
		Mode:          mode,
		CWD:           cwd,
		ExitCode:      0,
		Backend:       "local",
		NetworkPolicy: firstNonEmpty(request.NetworkPolicy, "deny"),
	}
	if err := cmd.Start(); err != nil {
		result.Duration = time.Since(start)
		result.ExitCode = -1
		return result, err
	}
	result.ProcessID = cmd.Process.Pid

	waitErr := cmd.Wait()
	result.Duration = time.Since(start)
	if runCtx.Err() != nil {
		if cleanupErr := killProcessGroup(cmd.Process); cleanupErr != nil && !errors.Is(cleanupErr, os.ErrProcessDone) {
			return result, &SandboxError{Code: SandboxErrorProcessCleanupFailed, Message: cleanupErr.Error(), Err: cleanupErr}
		}
		if errors.Is(ctx.Err(), context.Canceled) {
			result.Cancelled = true
			result.ExitCode = -1
			waitErr = &SandboxError{Code: SandboxErrorCommandCancelled, Message: "command cancelled", Err: ctx.Err()}
		} else {
			result.TimedOut = true
			result.ExitCode = -1
			waitErr = &SandboxError{Code: SandboxErrorCommandTimeout, Message: "command timed out", Err: runCtx.Err()}
		}
	}
	if waitErr != nil && result.ExitCode == 0 {
		var exitErr *exec.ExitError
		if errors.As(waitErr, &exitErr) {
			result.ExitCode = exitErr.ExitCode()
		} else {
			result.ExitCode = -1
		}
	}

	stdoutText, stdoutTruncated, stdoutRef, stdoutOriginal, retainErr := stdout.Finish()
	if retainErr != nil {
		return result, retainErr
	}
	stderrText, stderrTruncated, stderrRef, stderrOriginal, retainErr := stderr.Finish()
	if retainErr != nil {
		return result, retainErr
	}
	result.Stdout = redactCommandOutput(stdoutText)
	result.Stderr = redactCommandOutput(stderrText)
	result.StdoutRef = stdoutRef
	result.StderrRef = stderrRef
	result.Truncated = stdoutTruncated || stderrTruncated
	result.OriginalSize = stdoutOriginal + stderrOriginal
	return result, waitErr
}

type DockerSandboxRunner struct{}

func NewDockerSandboxRunner() *DockerSandboxRunner {
	return &DockerSandboxRunner{}
}

func (r *DockerSandboxRunner) Run(context.Context, SandboxRequest) (SandboxResult, error) {
	return SandboxResult{Backend: "docker", ExitCode: -1}, &SandboxError{
		Code:    SandboxErrorUnavailable,
		Message: "docker sandbox backend is not enabled",
	}
}
