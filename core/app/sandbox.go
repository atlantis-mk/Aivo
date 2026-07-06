package app

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"syscall"
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

func newShellOutputEmitter(request SandboxRequest, processRef string) *shellOutputEmitter {
	if request.OutputSink == nil {
		return nil
	}
	return &shellOutputEmitter{request: request, processRef: processRef}
}

func newShellOutputWriter(target *bytes.Buffer, emitter *shellOutputEmitter, stream string) io.Writer {
	if emitter == nil {
		return target
	}
	return &shellOutputWriter{target: target, emitter: emitter, stream: stream}
}

func (w *shellOutputWriter) Write(p []byte) (int, error) {
	n, err := w.target.Write(p)
	if n > 0 {
		w.emitter.emit(w.stream, string(p[:n]))
	}
	return n, err
}

func (e *shellOutputEmitter) emit(stream string, chunk string) {
	if e == nil || e.request.OutputSink == nil || chunk == "" {
		return
	}
	e.mu.Lock()
	e.nextSeq++
	sequence := e.nextSeq
	e.mu.Unlock()
	e.request.OutputSink(ShellOutputEvent{
		SessionID:   e.request.SessionID,
		TurnID:      e.request.TurnID,
		ToolCallID:  e.request.ToolCallID,
		ProcessRef:  e.processRef,
		Stream:      stream,
		Chunk:       redactCommandOutput(chunk),
		Sequence:    sequence,
		TimeCreated: domain.NowString(time.Now()),
	})
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

	cmd := exec.CommandContext(runCtx, shell, "-c", command)
	cmd.Dir = cwd
	cmd.Env = SanitizedEnvironment(workspaceRoot, request.EnvAllowlist, request.Env, request.EnvOverrides)
	if request.Stdin != "" {
		cmd.Stdin = strings.NewReader(request.Stdin)
	} else {
		cmd.Stdin = nil
	}
	setProcessGroup(cmd)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	emitter := newShellOutputEmitter(request, "")
	cmd.Stdout = newShellOutputWriter(&stdout, emitter, "stdout")
	cmd.Stderr = newShellOutputWriter(&stderr, emitter, "stderr")

	result := SandboxResult{
		Command:       command,
		Argv:          append([]string{shell, "-c"}, command),
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

	maxChars := request.OutputPolicy.MaxChars
	if maxChars <= 0 {
		maxChars = defaultStreamMaxChars
	}
	stdoutText, stdoutTruncated, stdoutRef, stdoutOriginal, retainErr := boundedSandboxOutput(request, "stdout", stdout.String(), maxChars)
	if retainErr != nil {
		return result, retainErr
	}
	stderrText, stderrTruncated, stderrRef, stderrOriginal, retainErr := boundedSandboxOutput(request, "stderr", stderr.String(), maxChars)
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

func normalizeSandboxCWD(workspaceRoot string, cwd string, allowExternal bool) (string, string, error) {
	root := strings.TrimSpace(workspaceRoot)
	if root == "" {
		return "", "", errors.New("workspace root is required")
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return "", "", err
	}
	realRoot, err := filepath.EvalSymlinks(absRoot)
	if err != nil {
		return "", "", err
	}
	cleanCWD := strings.TrimSpace(cwd)
	if cleanCWD == "" || cleanCWD == "." {
		return realRoot, realRoot, nil
	}
	var target string
	if filepath.IsAbs(cleanCWD) {
		target = cleanCWD
	} else {
		cleanRel := filepath.Clean(cleanCWD)
		if cleanRel == ".." || strings.HasPrefix(cleanRel, ".."+string(os.PathSeparator)) {
			return "", "", errors.New("cwd escapes workspace root")
		}
		target = filepath.Join(realRoot, cleanRel)
	}
	absTarget, err := filepath.Abs(target)
	if err != nil {
		return "", "", err
	}
	realTarget, err := filepath.EvalSymlinks(absTarget)
	if err != nil {
		return "", "", err
	}
	rel, err := filepath.Rel(realRoot, realTarget)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) || filepath.IsAbs(rel) {
		if allowExternal {
			return realRoot, realTarget, nil
		}
		return "", "", errors.New("cwd escapes workspace root")
	}
	return realRoot, realTarget, nil
}

func cwdIsExternal(workspaceRoot string, cwd string) (bool, string, error) {
	root, target, err := normalizeSandboxCWD(workspaceRoot, cwd, true)
	if err != nil {
		return false, "", err
	}
	rel, err := filepath.Rel(root, target)
	if err != nil {
		return true, target, nil
	}
	return rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) || filepath.IsAbs(rel), target, nil
}

func normalizeSandboxMode(mode string) string {
	switch strings.TrimSpace(mode) {
	case "", "foreground":
		return "foreground"
	case "background":
		return "background"
	case "pty":
		return "pty"
	default:
		return strings.TrimSpace(mode)
	}
}

func clampCommandTimeout(timeout time.Duration) time.Duration {
	if timeout <= 0 {
		return defaultCommandTimeout
	}
	if timeout > maxCommandTimeout {
		return maxCommandTimeout
	}
	return timeout
}

func defaultShell() string {
	if runtime.GOOS == "windows" {
		return "cmd"
	}
	if shell := strings.TrimSpace(os.Getenv("SHELL")); shell != "" && !isSecretEnvName("SHELL") {
		return shell
	}
	return "/bin/sh"
}

func SanitizedEnvironment(workspaceRoot string, allowlist []string, env map[string]string, overrides map[string]string) []string {
	allowed := map[string]bool{}
	for _, name := range defaultEnvAllowlist() {
		allowed[name] = true
	}
	for _, name := range allowlist {
		name = strings.TrimSpace(name)
		if name != "" {
			allowed[name] = true
		}
	}
	envMap := map[string]string{}
	for _, item := range os.Environ() {
		name, value, ok := strings.Cut(item, "=")
		if !ok || !envNameAllowed(name, allowed) || isSecretEnvName(name) {
			continue
		}
		if isCacheEnvName(name) && !pathUnderHomeOrWorkspace(value, workspaceRoot) {
			continue
		}
		envMap[name] = value
	}
	for name, value := range env {
		if !envNameAllowed(name, allowed) || isSecretEnvName(name) {
			continue
		}
		if isCacheEnvName(name) && !pathUnderHomeOrWorkspace(value, workspaceRoot) {
			continue
		}
		envMap[name] = value
	}
	for name, value := range overrides {
		if !envOverrideKeyAllowed(name) || isSecretEnvName(name) {
			continue
		}
		if isCacheEnvName(name) && !pathUnderHomeOrWorkspace(value, workspaceRoot) {
			continue
		}
		envMap[name] = value
	}
	envMap["AIVO_SANDBOX"] = "local"
	if _, ok := envMap["CI"]; !ok {
		envMap["CI"] = "1"
	}
	keys := make([]string, 0, len(envMap))
	for key := range envMap {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]string, 0, len(keys))
	for _, key := range keys {
		out = append(out, key+"="+envMap[key])
	}
	return out
}

func envOverrideKeyAllowed(name string) bool {
	switch strings.TrimSpace(name) {
	case "CI", "NODE_ENV", "GOFLAGS":
		return true
	case "NPM_CONFIG_CACHE", "PNPM_HOME", "YARN_CACHE_FOLDER":
		return true
	default:
		return false
	}
}

func defaultEnvAllowlist() []string {
	return []string{
		"PATH", "HOME", "USER", "LOGNAME", "SHELL", "TMPDIR", "TEMP", "TMP", "LANG", "TERM", "CI",
		"GOCACHE", "GOMODCACHE", "GOPATH", "NPM_CONFIG_CACHE", "PNPM_HOME", "YARN_CACHE_FOLDER",
	}
}

func envNameAllowed(name string, allowed map[string]bool) bool {
	if allowed[name] {
		return true
	}
	return strings.HasPrefix(name, "LC_")
}

func isSecretEnvName(name string) bool {
	upper := strings.ToUpper(strings.TrimSpace(name))
	if upper == "" {
		return false
	}
	for _, marker := range []string{"KEY", "TOKEN", "SECRET", "PASSWORD", "CREDENTIAL", "COOKIE", "SESSION", "AUTH"} {
		if strings.Contains(upper, marker) {
			return true
		}
	}
	return false
}

func isCacheEnvName(name string) bool {
	switch name {
	case "NPM_CONFIG_CACHE", "PNPM_HOME", "YARN_CACHE_FOLDER":
		return true
	default:
		return false
	}
}

func pathUnderHomeOrWorkspace(value string, workspaceRoot string) bool {
	if strings.TrimSpace(value) == "" {
		return false
	}
	target, err := filepath.Abs(value)
	if err != nil {
		return false
	}
	if home, err := os.UserHomeDir(); err == nil && pathHasPrefix(target, home) {
		return true
	}
	return pathHasPrefix(target, workspaceRoot)
}

func pathHasPrefix(path string, root string) bool {
	root = strings.TrimSpace(root)
	if root == "" {
		return false
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(absRoot, path)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator)) && !filepath.IsAbs(rel)
}

func boundedSandboxOutput(request SandboxRequest, stream string, content string, maxChars int) (string, bool, string, int, error) {
	original := len(content)
	if original <= maxChars {
		return content, false, "", original, nil
	}
	ref, err := retainSandboxOutput(request, stream, content)
	if err != nil {
		return "", false, "", original, &SandboxError{Code: SandboxErrorOutputRetentionFailed, Message: err.Error(), Err: err}
	}
	return content[:maxChars] + fmt.Sprintf("\n\n[truncated: %s exceeded %d characters; full output retained at %s]", stream, maxChars, ref), true, ref, original, nil
}

func retainSandboxOutput(request SandboxRequest, stream string, content string) (string, error) {
	base, err := os.UserCacheDir()
	if err != nil || strings.TrimSpace(base) == "" {
		base = os.TempDir()
	}
	sessionID := safeArtifactPart(firstNonEmpty(request.SessionID, "session"))
	toolCallID := safeArtifactPart(firstNonEmpty(request.ToolCallID, fmt.Sprintf("%d", time.Now().UnixNano())))
	dir := filepath.Join(base, "aivo", "command-artifacts", sessionID, toolCallID)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	path := filepath.Join(dir, safeArtifactPart(stream)+".log")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		return "", err
	}
	return path, nil
}

func safeArtifactPart(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "artifact"
	}
	var b strings.Builder
	for _, r := range value {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_' || r == '.' {
			b.WriteRune(r)
			continue
		}
		b.WriteByte('_')
	}
	return b.String()
}

func redactCommandOutput(value string) string {
	for _, marker := range []string{"OPENAI_API_KEY=", "ANTHROPIC_API_KEY=", "GITHUB_TOKEN=", "GOOGLE_API_KEY="} {
		value = redactAfterMarker(value, marker)
	}
	return value
}

func redactAfterMarker(value string, marker string) string {
	for {
		idx := strings.Index(value, marker)
		if idx < 0 {
			return value
		}
		start := idx + len(marker)
		end := start
		for end < len(value) && value[end] != '\n' && value[end] != ' ' && value[end] != '\t' {
			end++
		}
		value = value[:start] + "[redacted]" + value[end:]
	}
}

func setProcessGroup(cmd *exec.Cmd) {
	if runtime.GOOS == "windows" {
		return
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

func killProcessGroup(process *os.Process) error {
	if process == nil {
		return nil
	}
	if runtime.GOOS == "windows" {
		return process.Kill()
	}
	if process.Pid <= 0 {
		return nil
	}
	if err := syscall.Kill(-process.Pid, syscall.SIGKILL); err != nil && !errors.Is(err, syscall.ESRCH) {
		return err
	}
	return nil
}
