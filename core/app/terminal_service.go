package app

import (
	"context"
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
)

const (
	TerminalStatusRunning = "running"
	TerminalStatusExited  = "exited"
	TerminalStatusRemoved = "removed"
	terminalBufferCap     = 256 * 1024
)

type TerminalService interface {
	List(ctx context.Context, workspaceRoot string) ([]TerminalInfo, error)
	Create(ctx context.Context, input TerminalCreateInput) (TerminalInfo, error)
	Get(ctx context.Context, workspaceRoot string, terminalID string) (TerminalInfo, error)
	Update(ctx context.Context, input TerminalUpdateInput) (TerminalInfo, error)
	Remove(ctx context.Context, workspaceRoot string, terminalID string) error
	Attach(ctx context.Context, input TerminalAttachInput) (TerminalAttachment, error)
}

type TerminalInfo struct {
	ID            string    `json:"id"`
	WorkspaceRoot string    `json:"workspaceRoot"`
	Title         string    `json:"title"`
	Command       string    `json:"command"`
	Args          []string  `json:"args,omitempty"`
	CWD           string    `json:"cwd"`
	Status        string    `json:"status"`
	PID           int       `json:"pid"`
	ExitCode      *int      `json:"exitCode,omitempty"`
	Rows          int       `json:"rows"`
	Cols          int       `json:"cols"`
	Cursor        int64     `json:"cursor"`
	TimeCreated   time.Time `json:"timeCreated"`
	TimeUpdated   time.Time `json:"timeUpdated"`
}

type TerminalCreateInput struct {
	WorkspaceRoot string            `json:"workspaceRoot"`
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

type terminalRecord struct {
	mu          sync.Mutex
	info        TerminalInfo
	cmd         *exec.Cmd
	ptyFile     *os.File
	buffer      []byte
	baseCursor  int64
	nextCursor  int64
	subscribers map[chan []byte]struct{}
	removed     bool
}

type DefaultTerminalService struct {
	mu       sync.Mutex
	records  map[string]*terminalRecord
	onEvent  func(string, TerminalInfo)
	shutdown bool
}

func NewTerminalService() *DefaultTerminalService {
	return &DefaultTerminalService{records: map[string]*terminalRecord{}}
}

func (s *DefaultTerminalService) SetEventHook(hook func(string, TerminalInfo)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.onEvent = hook
}

func (s *DefaultTerminalService) List(_ context.Context, workspaceRoot string) ([]TerminalInfo, error) {
	root, err := terminalWorkspaceRoot(workspaceRoot)
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	records := make([]*terminalRecord, 0)
	for _, record := range s.records {
		record.mu.Lock()
		matches := record.info.WorkspaceRoot == root && !record.removed
		record.mu.Unlock()
		if matches {
			records = append(records, record)
		}
	}
	s.mu.Unlock()
	out := make([]TerminalInfo, 0, len(records))
	for _, record := range records {
		out = append(out, record.snapshot())
	}
	sort.Slice(out, func(i, j int) bool { return out[i].TimeCreated.Before(out[j].TimeCreated) })
	return out, nil
}

func (s *DefaultTerminalService) Create(_ context.Context, input TerminalCreateInput) (TerminalInfo, error) {
	root, cwd, err := normalizeTerminalCWD(input.WorkspaceRoot, input.CWD)
	if err != nil {
		return TerminalInfo{}, err
	}
	rows, cols := normalizeTerminalSize(input.Rows, input.Cols)
	shell := firstNonEmpty(strings.TrimSpace(input.Shell), defaultShell())
	title := strings.TrimSpace(input.Title)
	if title == "" {
		title = filepath.Base(shell)
	}
	cmd, ptmx, shell, err := startTerminalPTY(shell, cwd, terminalEnvironment(root, input.Env), rows, cols)
	if err != nil {
		return TerminalInfo{}, err
	}
	now := time.Now()
	record := &terminalRecord{
		info: TerminalInfo{
			ID: uuid.NewString(), WorkspaceRoot: root, Title: title, Command: shell, CWD: cwd,
			Status: TerminalStatusRunning, PID: cmd.Process.Pid, Rows: rows, Cols: cols,
			TimeCreated: now, TimeUpdated: now,
		},
		cmd:         cmd,
		ptyFile:     ptmx,
		subscribers: map[chan []byte]struct{}{},
	}
	s.mu.Lock()
	if s.shutdown {
		s.mu.Unlock()
		_ = killTerminalProcess(cmd.Process)
		_ = ptmx.Close()
		return TerminalInfo{}, errors.New("terminal service is shutting down")
	}
	s.records[record.info.ID] = record
	s.mu.Unlock()
	go s.readPTY(record)
	go s.waitTerminal(record)
	info := record.snapshot()
	s.emit("terminal.created", info)
	return info, nil
}

func (s *DefaultTerminalService) Get(_ context.Context, workspaceRoot string, terminalID string) (TerminalInfo, error) {
	record, err := s.record(workspaceRoot, terminalID)
	if err != nil {
		return TerminalInfo{}, err
	}
	return record.snapshot(), nil
}

func (s *DefaultTerminalService) Update(_ context.Context, input TerminalUpdateInput) (TerminalInfo, error) {
	record, err := s.record(input.WorkspaceRoot, input.TerminalID)
	if err != nil {
		return TerminalInfo{}, err
	}
	record.mu.Lock()
	if strings.TrimSpace(input.Title) != "" {
		record.info.Title = strings.TrimSpace(input.Title)
	}
	if input.Rows > 0 && input.Cols > 0 {
		record.info.Rows, record.info.Cols = normalizeTerminalSize(input.Rows, input.Cols)
		if record.ptyFile != nil && record.info.Status == TerminalStatusRunning {
			_ = pty.Setsize(record.ptyFile, &pty.Winsize{Rows: uint16(record.info.Rows), Cols: uint16(record.info.Cols)})
		}
	}
	record.info.TimeUpdated = time.Now()
	info := record.info
	record.mu.Unlock()
	s.emit("terminal.updated", info)
	return info, nil
}

func (s *DefaultTerminalService) Remove(_ context.Context, workspaceRoot string, terminalID string) error {
	record, err := s.record(workspaceRoot, terminalID)
	if err != nil {
		return err
	}
	record.mu.Lock()
	record.removed = true
	record.info.Status = TerminalStatusRemoved
	record.info.TimeUpdated = time.Now()
	info := record.info
	if record.cmd != nil && record.cmd.Process != nil {
		_ = killTerminalProcess(record.cmd.Process)
	}
	if record.ptyFile != nil {
		_ = record.ptyFile.Close()
	}
	for ch := range record.subscribers {
		close(ch)
		delete(record.subscribers, ch)
	}
	record.mu.Unlock()
	s.emit("terminal.removed", info)
	return nil
}

func (s *DefaultTerminalService) Attach(_ context.Context, input TerminalAttachInput) (TerminalAttachment, error) {
	record, err := s.record(input.WorkspaceRoot, input.TerminalID)
	if err != nil {
		return nil, err
	}
	record.mu.Lock()
	if record.info.Status != TerminalStatusRunning {
		record.mu.Unlock()
		return nil, fmt.Errorf("terminal %s is not running", input.TerminalID)
	}
	ch := make(chan []byte, 64)
	record.subscribers[ch] = struct{}{}
	replay := record.replayLocked(input.Cursor)
	cursor := record.nextCursor
	record.mu.Unlock()
	return &terminalAttachment{record: record, replay: replay, cursor: cursor, data: ch}, nil
}

func (s *DefaultTerminalService) Shutdown() {
	s.mu.Lock()
	s.shutdown = true
	records := make([]*terminalRecord, 0, len(s.records))
	for _, record := range s.records {
		records = append(records, record)
	}
	s.mu.Unlock()
	for _, record := range records {
		record.mu.Lock()
		if record.info.Status == TerminalStatusRunning && record.cmd != nil && record.cmd.Process != nil {
			_ = killTerminalProcess(record.cmd.Process)
		}
		if record.ptyFile != nil {
			_ = record.ptyFile.Close()
		}
		record.mu.Unlock()
	}
}

func (s *DefaultTerminalService) readPTY(record *terminalRecord) {
	buf := make([]byte, 4096)
	for {
		n, err := record.ptyFile.Read(buf)
		if n > 0 {
			chunk := append([]byte(nil), buf[:n]...)
			record.append(chunk)
		}
		if err != nil {
			if !errors.Is(err, io.EOF) && !strings.Contains(err.Error(), "input/output error") {
				record.append([]byte("\r\n[aivo terminal read error: " + err.Error() + "]\r\n"))
			}
			return
		}
	}
}

func (s *DefaultTerminalService) waitTerminal(record *terminalRecord) {
	err := record.cmd.Wait()
	exitCode := 0
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = -1
		}
	}
	record.mu.Lock()
	if !record.removed {
		record.info.Status = TerminalStatusExited
		record.info.ExitCode = &exitCode
		record.info.TimeUpdated = time.Now()
	}
	info := record.info
	for ch := range record.subscribers {
		close(ch)
		delete(record.subscribers, ch)
	}
	record.mu.Unlock()
	if !record.removed {
		s.emit("terminal.exited", info)
	}
}

func (s *DefaultTerminalService) record(workspaceRoot string, terminalID string) (*terminalRecord, error) {
	root, err := terminalWorkspaceRoot(workspaceRoot)
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	record := s.records[strings.TrimSpace(terminalID)]
	s.mu.Unlock()
	if record == nil {
		return nil, fmt.Errorf("terminal %s not found", terminalID)
	}
	record.mu.Lock()
	ok := record.info.WorkspaceRoot == root && !record.removed
	record.mu.Unlock()
	if !ok {
		return nil, fmt.Errorf("terminal %s not found", terminalID)
	}
	return record, nil
}

func (s *DefaultTerminalService) emit(name string, info TerminalInfo) {
	s.mu.Lock()
	hook := s.onEvent
	s.mu.Unlock()
	if hook != nil {
		hook(name, info)
	}
}

func (record *terminalRecord) append(chunk []byte) {
	record.mu.Lock()
	record.buffer = append(record.buffer, chunk...)
	record.nextCursor += int64(len(chunk))
	if len(record.buffer) > terminalBufferCap {
		drop := len(record.buffer) - terminalBufferCap
		record.buffer = append([]byte(nil), record.buffer[drop:]...)
		record.baseCursor += int64(drop)
	}
	record.info.Cursor = record.nextCursor
	record.info.TimeUpdated = time.Now()
	subscribers := make([]chan []byte, 0, len(record.subscribers))
	for ch := range record.subscribers {
		subscribers = append(subscribers, ch)
	}
	record.mu.Unlock()
	for _, ch := range subscribers {
		select {
		case ch <- chunk:
		default:
		}
	}
}

func (record *terminalRecord) snapshot() TerminalInfo {
	record.mu.Lock()
	defer record.mu.Unlock()
	info := record.info
	info.Cursor = record.nextCursor
	return info
}

func (record *terminalRecord) replayLocked(cursor int64) []byte {
	if cursor < 0 || cursor > record.nextCursor {
		return nil
	}
	if cursor < record.baseCursor {
		cursor = record.baseCursor
	}
	offset := int(cursor - record.baseCursor)
	if offset < 0 || offset > len(record.buffer) {
		return nil
	}
	return append([]byte(nil), record.buffer[offset:]...)
}

type terminalAttachment struct {
	record *terminalRecord
	replay []byte
	cursor int64
	data   chan []byte
	once   sync.Once
}

func (a *terminalAttachment) Replay() []byte {
	return append([]byte(nil), a.replay...)
}

func (a *terminalAttachment) Cursor() int64 {
	return a.cursor
}

func (a *terminalAttachment) Write(data []byte) error {
	a.record.mu.Lock()
	file := a.record.ptyFile
	running := a.record.info.Status == TerminalStatusRunning
	a.record.mu.Unlock()
	if file == nil || !running {
		return errors.New("terminal is not running")
	}
	_, err := file.Write(data)
	return err
}

func (a *terminalAttachment) Resize(rows int, cols int) error {
	rows, cols = normalizeTerminalSize(rows, cols)
	a.record.mu.Lock()
	file := a.record.ptyFile
	a.record.info.Rows = rows
	a.record.info.Cols = cols
	a.record.info.TimeUpdated = time.Now()
	a.record.mu.Unlock()
	if file == nil {
		return errors.New("terminal is not running")
	}
	return pty.Setsize(file, &pty.Winsize{Rows: uint16(rows), Cols: uint16(cols)})
}

func (a *terminalAttachment) Data() <-chan []byte {
	return a.data
}

func (a *terminalAttachment) Detach() {
	a.once.Do(func() {
		a.record.mu.Lock()
		_, ok := a.record.subscribers[a.data]
		if ok {
			delete(a.record.subscribers, a.data)
		}
		a.record.mu.Unlock()
		if ok {
			close(a.data)
		}
	})
}

func normalizeTerminalSize(rows int, cols int) (int, int) {
	if rows <= 0 {
		rows = 24
	}
	if cols <= 0 {
		cols = 80
	}
	if rows < 4 {
		rows = 4
	}
	if rows > 200 {
		rows = 200
	}
	if cols < 20 {
		cols = 20
	}
	if cols > 400 {
		cols = 400
	}
	return rows, cols
}

func terminalWorkspaceRoot(workspaceRoot string) (string, error) {
	root := strings.TrimSpace(workspaceRoot)
	if root == "" {
		return "", nil
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	realRoot, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return "", err
	}
	return realRoot, nil
}

func normalizeTerminalCWD(workspaceRoot string, cwd string) (string, string, error) {
	root := strings.TrimSpace(workspaceRoot)
	if root != "" {
		return normalizeSandboxCWD(root, cwd, false)
	}
	base, err := defaultTerminalCWD()
	if err != nil {
		return "", "", err
	}
	cleanCWD := strings.TrimSpace(cwd)
	if cleanCWD == "" || cleanCWD == "." {
		return "", base, nil
	}
	target := cleanCWD
	if !filepath.IsAbs(target) {
		target = filepath.Join(base, filepath.Clean(target))
	}
	absTarget, err := filepath.Abs(target)
	if err != nil {
		return "", "", err
	}
	realTarget, err := filepath.EvalSymlinks(absTarget)
	if err != nil {
		return "", "", err
	}
	return "", realTarget, nil
}

func defaultTerminalCWD() (string, error) {
	if home, err := os.UserHomeDir(); err == nil && strings.TrimSpace(home) != "" {
		realHome, evalErr := filepath.EvalSymlinks(home)
		if evalErr == nil {
			return realHome, nil
		}
		return filepath.Abs(home)
	}
	return os.Getwd()
}

func terminalEnvironment(workspaceRoot string, env map[string]string) []string {
	safe := SanitizedEnvironment(workspaceRoot, append(defaultEnvAllowlist(), "AIVO_TERMINAL"), nil, env)
	safe = append(safe, "AIVO_TERMINAL=1", "TERM=xterm-256color")
	return safe
}

func startTerminalPTY(preferredShell string, cwd string, env []string, rows int, cols int) (*exec.Cmd, *os.File, string, error) {
	shells := []string{strings.TrimSpace(preferredShell)}
	if runtimeShell := strings.TrimSpace(os.Getenv("SHELL")); runtimeShell != "" {
		shells = append(shells, runtimeShell)
	}
	shells = append(shells, "/bin/zsh", "/bin/sh")
	shells = uniqueNonEmptyStrings(shells)
	var failures []string
	cmd, ptmx, shell, err := startTerminalPTYInCWD(shells, cwd, env, rows, cols, &failures)
	if err == nil {
		return cmd, ptmx, shell, nil
	}
	if terminalStartMayBeCWDPermission(failures) {
		if home, homeErr := os.UserHomeDir(); homeErr == nil && home != "" && home != cwd {
			cmd, ptmx, shell, err = startTerminalPTYInCWD(shells, home, env, rows, cols, &failures)
			if err == nil {
				_, _ = ptmx.Write([]byte("cd " + shellSingleQuote(cwd) + "\n"))
				return cmd, ptmx, shell, nil
			}
		}
	}
	if len(failures) == 0 {
		failures = append(failures, "no terminal shell candidates")
	}
	return nil, nil, "", fmt.Errorf("start terminal shell failed; attempted %s", strings.Join(failures, "; "))
}

func startTerminalPTYInCWD(shells []string, cwd string, env []string, rows int, cols int, failures *[]string) (*exec.Cmd, *os.File, string, error) {
	var lastErr error
	for _, shell := range shells {
		if err := executableShellAvailable(shell); err != nil {
			*failures = append(*failures, fmt.Sprintf("%s in %s: %v", shell, cwd, err))
			lastErr = err
			continue
		}
		cmd := exec.Command(shell)
		cmd.Dir = cwd
		cmd.Env = env
		ptmx, err := pty.StartWithSize(cmd, &pty.Winsize{Rows: uint16(rows), Cols: uint16(cols)})
		if err == nil {
			return cmd, ptmx, shell, nil
		}
		lastErr = err
		*failures = append(*failures, fmt.Sprintf("%s in %s: %v", shell, cwd, err))
	}
	if lastErr == nil {
		lastErr = errors.New("no terminal shell candidates")
	}
	return nil, nil, "", lastErr
}

func terminalStartMayBeCWDPermission(failures []string) bool {
	for _, failure := range failures {
		if strings.Contains(strings.ToLower(failure), "operation not permitted") {
			return true
		}
	}
	return false
}

func executableShellAvailable(shell string) error {
	info, err := os.Stat(shell)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return fmt.Errorf("is a directory")
	}
	if info.Mode().Perm()&0o111 == 0 {
		return fmt.Errorf("is not executable")
	}
	return nil
}

func shellSingleQuote(value string) string {
	if value == "" {
		return "''"
	}
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}

func killTerminalProcess(process *os.Process) error {
	if process == nil {
		return nil
	}
	groupErr := killProcessGroup(process)
	directErr := process.Kill()
	if groupErr != nil {
		return groupErr
	}
	if directErr != nil && !errors.Is(directErr, os.ErrProcessDone) {
		return directErr
	}
	return nil
}

func uniqueNonEmptyStrings(values []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}

var _ TerminalService = (*DefaultTerminalService)(nil)
