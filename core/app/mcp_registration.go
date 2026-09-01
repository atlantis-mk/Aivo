package app

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"aivo/core/domain"
)

const (
	mcpRegistrationProposalTTL = 10 * time.Minute
	mcpRegistrationMaxPending  = 64
)

var (
	mcpRegistrationIDPattern  = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)
	mcpRegistrationEnvPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
	mcpExecutableNamePattern  = regexp.MustCompile(`^[A-Za-z0-9_.+-]+$`)
)

type mcpRegistrationError struct {
	code string
	err  error
}

func (e *mcpRegistrationError) Error() string {
	if e == nil || e.err == nil {
		return "MCP registration failed"
	}
	return e.err.Error()
}

func (e *mcpRegistrationError) Unwrap() error { return e.err }

func newMCPRegistrationError(code, message string) error {
	return &mcpRegistrationError{code: code, err: errors.New(message)}
}

func mcpRegistrationErrorCode(err error) string {
	var registrationErr *mcpRegistrationError
	if errors.As(err, &registrationErr) && registrationErr.code != "" {
		return registrationErr.code
	}
	if errors.Is(err, context.Canceled) {
		return "cancelled"
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return "timeout"
	}
	return "mcp_registration_failed"
}

type mcpRegistrationProposal struct {
	ID         string
	SessionID  string
	TurnID     string
	ToolCallID string
	Input      domain.MCPRegistrationProposalInput
	Hash       string
	CreatedAt  time.Time
	ExpiresAt  time.Time
}

type mcpRegistrationProposalStore struct {
	mu       sync.Mutex
	commitMu sync.Mutex
	byCall   map[string]mcpRegistrationProposal
	now      func() time.Time
	ttl      time.Duration
	limit    int
}

func newMCPRegistrationProposalStore() *mcpRegistrationProposalStore {
	return &mcpRegistrationProposalStore{byCall: map[string]mcpRegistrationProposal{}, now: time.Now, ttl: mcpRegistrationProposalTTL, limit: mcpRegistrationMaxPending}
}

func (s *mcpRegistrationProposalStore) prepare(input domain.MCPRegistrationProposalInput, execCtx domain.ToolExecutionContext) (mcpRegistrationProposal, error) {
	if s == nil || strings.TrimSpace(execCtx.SessionID) == "" || strings.TrimSpace(execCtx.TurnID) == "" || strings.TrimSpace(execCtx.ToolCallID) == "" {
		return mcpRegistrationProposal{}, newMCPRegistrationError("invalid_proposal_owner", "registration proposal requires an owning session, turn, and tool call")
	}
	now := s.now()
	hash := mcpRegistrationInputHash(input)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cleanupLocked(now)
	if current, ok := s.byCall[execCtx.ToolCallID]; ok {
		if current.SessionID == execCtx.SessionID && current.TurnID == execCtx.TurnID && current.Hash == hash {
			return current, nil
		}
		return mcpRegistrationProposal{}, newMCPRegistrationError("proposal_conflict", "tool call already owns a different registration proposal")
	}
	if len(s.byCall) >= s.limit {
		s.evictOldestLocked()
	}
	proposal := mcpRegistrationProposal{
		ID: uuid.NewString(), SessionID: execCtx.SessionID, TurnID: execCtx.TurnID, ToolCallID: execCtx.ToolCallID,
		Input: input, Hash: hash, CreatedAt: now, ExpiresAt: now.Add(s.ttl),
	}
	s.byCall[proposal.ToolCallID] = proposal
	return proposal, nil
}

func (s *mcpRegistrationProposalStore) consume(input domain.MCPRegistrationProposalInput, execCtx domain.ToolExecutionContext) (mcpRegistrationProposal, error) {
	if s == nil {
		return mcpRegistrationProposal{}, newMCPRegistrationError("proposal_unavailable", "registration proposal store is unavailable")
	}
	now := s.now()
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cleanupLocked(now)
	proposal, ok := s.byCall[strings.TrimSpace(execCtx.ToolCallID)]
	if !ok {
		return mcpRegistrationProposal{}, newMCPRegistrationError("proposal_expired", "registration proposal is missing, expired, cancelled, or already consumed")
	}
	if proposal.SessionID != strings.TrimSpace(execCtx.SessionID) || proposal.TurnID != strings.TrimSpace(execCtx.TurnID) || proposal.Hash != mcpRegistrationInputHash(input) {
		return mcpRegistrationProposal{}, newMCPRegistrationError("proposal_mismatch", "registration confirmation does not match the exact approved proposal")
	}
	delete(s.byCall, proposal.ToolCallID)
	return proposal, nil
}

func (s *mcpRegistrationProposalStore) discard(toolCallID string) {
	if s == nil || strings.TrimSpace(toolCallID) == "" {
		return
	}
	s.mu.Lock()
	delete(s.byCall, strings.TrimSpace(toolCallID))
	s.mu.Unlock()
}

func (s *mcpRegistrationProposalStore) clear() {
	if s == nil {
		return
	}
	s.mu.Lock()
	s.byCall = map[string]mcpRegistrationProposal{}
	s.mu.Unlock()
}

func (s *mcpRegistrationProposalStore) cleanupLocked(now time.Time) {
	for key, proposal := range s.byCall {
		if !proposal.ExpiresAt.After(now) {
			delete(s.byCall, key)
		}
	}
}

func (s *mcpRegistrationProposalStore) evictOldestLocked() {
	oldestKey := ""
	var oldest time.Time
	for key, proposal := range s.byCall {
		if oldestKey == "" || proposal.CreatedAt.Before(oldest) {
			oldestKey, oldest = key, proposal.CreatedAt
		}
	}
	delete(s.byCall, oldestKey)
}

func mcpRegistrationInputHash(input domain.MCPRegistrationProposalInput) string {
	raw, _ := json.Marshal(input)
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func (s *Service) prepareToolRegistrationPermission(ctx context.Context, name string, args json.RawMessage, execCtx domain.ToolExecutionContext) ([]string, map[string]any, bool, error) {
	switch name {
	case toolRegistrationMCPName:
		return s.prepareMCPRegistrationPermission(ctx, name, args, execCtx)
	case toolRegistrationResourceName:
		return s.prepareResourceRegistrationPermission(ctx, name, args, execCtx)
	default:
		return nil, nil, false, newMCPRegistrationError("invalid_registration_tool", "unknown registration tool")
	}
}

func (s *Service) prepareMCPRegistrationPermission(ctx context.Context, name string, args json.RawMessage, execCtx domain.ToolExecutionContext) ([]string, map[string]any, bool, error) {
	if name != toolRegistrationMCPName {
		return nil, nil, false, newMCPRegistrationError("invalid_registration_tool", "unknown registration tool")
	}
	var input domain.MCPRegistrationProposalInput
	if err := decodeStrictToolArgs(args, &input); err != nil {
		return nil, nil, false, newMCPRegistrationError("invalid_arguments", err.Error())
	}
	normalized, err := normalizeMCPRegistrationInput(input)
	if err != nil {
		return nil, nil, false, err
	}
	if err := s.ensureMCPRegistrationIdentityAvailable(ctx, normalized); err != nil {
		return nil, nil, false, err
	}
	if s.mcpRegistrationProposals == nil {
		s.mcpRegistrationProposals = newMCPRegistrationProposalStore()
	}
	proposal, err := s.mcpRegistrationProposals.prepare(normalized, execCtx)
	if err != nil {
		return nil, nil, false, err
	}
	metadata := map[string]any{
		"registrationProposalId": proposal.ID,
		"registrationKind":       "mcp",
		"registrationServerId":   normalized.ID,
		"registrationName":       normalized.DisplayName,
		"registrationTransport":  normalized.Transport,
		"registrationTarget":     mcpRegistrationTarget(normalized),
		"registrationCwd":        normalized.CWD,
		"registrationRoots":      normalized.Roots,
		"registrationAuth":       normalized.AuthType,
		"registrationGlobal":     true,
		"registrationExpiresAt":  domain.NowString(proposal.ExpiresAt),
		"riskLevel":              "high",
		"category":               "external_tool_registration",
		"rememberScope":          "never",
	}
	if normalized.BearerTokenEnv != "" {
		metadata["registrationBearerTokenEnv"] = normalized.BearerTokenEnv
	}
	if normalized.Transport == domain.MCPTransportStdio {
		metadata["networkHint"] = "depends_on_registered_process"
	} else {
		metadata["networkHint"] = "required"
	}
	return nil, metadata, false, nil
}

func (s *Service) commitMCPRegistrationProposal(ctx context.Context, input domain.MCPRegistrationProposalInput, execCtx domain.ToolExecutionContext) (domain.MCPRegistrationResult, error) {
	normalized, err := normalizeMCPRegistrationInput(input)
	if err != nil {
		return domain.MCPRegistrationResult{}, err
	}
	if s.mcpRegistrationProposals == nil {
		return domain.MCPRegistrationResult{}, newMCPRegistrationError("proposal_unavailable", "registration proposal store is unavailable")
	}
	if _, err := s.mcpRegistrationProposals.consume(normalized, execCtx); err != nil {
		return domain.MCPRegistrationResult{}, err
	}
	s.mcpRegistrationProposals.commitMu.Lock()
	defer s.mcpRegistrationProposals.commitMu.Unlock()
	if err := s.ensureMCPRegistrationIdentityAvailable(ctx, normalized); err != nil {
		return domain.MCPRegistrationResult{}, err
	}
	if s.mcpManager == nil {
		s.mcpManager = NewMCPManager(s.store, s.secrets)
	}
	server := mcpServerFromRegistrationInput(normalized)
	saved, err := s.mcpManager.Save(ctx, domain.SaveMCPServerInput{Server: server})
	if err != nil {
		return domain.MCPRegistrationResult{}, newMCPRegistrationError("registration_persist_failed", sanitizeMCPError(err.Error()))
	}
	probe, probeErr := s.mcpManager.Probe(ctx, domain.MCPProbeInput{ServerID: saved.ID})
	if probeErr != nil || !probe.OK {
		message := "MCP capability discovery failed"
		if probeErr != nil {
			message = sanitizeMCPError(probeErr.Error())
		} else if strings.TrimSpace(probe.Error) != "" {
			message = sanitizeMCPError(probe.Error)
		}
		s.markMCPRegistrationFailed(ctx, saved.ID, message)
		return domain.MCPRegistrationResult{}, newMCPRegistrationError("mcp_probe_failed", message)
	}
	toolNames, err := s.validateMCPRegistrationNames(ctx, saved, probe.Tools)
	if err != nil {
		s.markMCPRegistrationFailed(ctx, saved.ID, err.Error())
		return domain.MCPRegistrationResult{}, err
	}
	ready, err := s.mcpManager.store.SetMCPServerEnabled(ctx, saved.ID, true, domain.MCPServerStatusReady, "")
	if err != nil {
		s.markMCPRegistrationFailed(ctx, saved.ID, "failed to mark MCP source ready")
		return domain.MCPRegistrationResult{}, newMCPRegistrationError("registration_enable_failed", sanitizeMCPError(err.Error()))
	}
	s.refreshProviderExtensions("")
	return domain.MCPRegistrationResult{ID: ready.ID, DisplayName: firstNonEmptyApp(ready.DisplayName, ready.Name, ready.ID), Transport: ready.Transport, Status: ready.Status, ToolNames: toolNames}, nil
}

func normalizeMCPRegistrationInput(input domain.MCPRegistrationProposalInput) (domain.MCPRegistrationProposalInput, error) {
	input.ID = strings.TrimSpace(input.ID)
	input.DisplayName = strings.TrimSpace(input.DisplayName)
	input.Description = strings.TrimSpace(input.Description)
	input.Transport = strings.TrimSpace(input.Transport)
	input.Command = strings.TrimSpace(input.Command)
	input.CWD = strings.TrimSpace(input.CWD)
	input.URL = strings.TrimSpace(input.URL)
	input.AuthType = strings.TrimSpace(input.AuthType)
	input.BearerTokenEnv = strings.TrimSpace(input.BearerTokenEnv)
	if input.AuthType == "" {
		input.AuthType = domain.MCPAuthNone
	}
	if input.TimeoutSeconds == 0 {
		input.TimeoutSeconds = 30
	}
	if input.ConnectTimeoutSeconds == 0 {
		input.ConnectTimeoutSeconds = 30
	}
	if input.ID == "" || len(input.ID) > 48 || !mcpRegistrationIDPattern.MatchString(input.ID) {
		return input, newMCPRegistrationError("invalid_source_id", "id must match ^[A-Za-z0-9_-]+$ and be at most 48 bytes")
	}
	if input.DisplayName == "" || len(input.DisplayName) > 100 {
		return input, newMCPRegistrationError("invalid_display_name", "displayName is required and must be at most 100 bytes")
	}
	if len(input.Description) > 500 {
		return input, newMCPRegistrationError("invalid_description", "description must be at most 500 bytes")
	}
	if input.TimeoutSeconds < 1 || input.TimeoutSeconds > 120 || input.ConnectTimeoutSeconds < 1 || input.ConnectTimeoutSeconds > 60 {
		return input, newMCPRegistrationError("invalid_timeout", "timeouts are outside the supported bounds")
	}
	if len(input.Args) > 64 || len(input.Roots) > 16 {
		return input, newMCPRegistrationError("proposal_too_large", "registration proposal exceeds argument or root bounds")
	}
	for index, arg := range input.Args {
		if len(arg) > 2000 {
			return input, newMCPRegistrationError("invalid_argument", fmt.Sprintf("argument %d exceeds 2000 bytes", index+1))
		}
		if looksLikeRawSecretArgument(arg) {
			return input, newMCPRegistrationError("raw_secret_refused", fmt.Sprintf("argument %d looks like a raw secret; use a Host environment reference instead", index+1))
		}
	}
	roots := make([]string, 0, len(input.Roots))
	seenRoots := map[string]bool{}
	for _, root := range input.Roots {
		root = filepath.Clean(strings.TrimSpace(root))
		if root == "." || !filepath.IsAbs(root) {
			return input, newMCPRegistrationError("invalid_root", "MCP roots must be existing absolute local directories")
		}
		info, err := os.Stat(root)
		if err != nil || !info.IsDir() {
			return input, newMCPRegistrationError("invalid_root", "MCP roots must be existing absolute local directories")
		}
		if !seenRoots[root] {
			seenRoots[root] = true
			roots = append(roots, root)
		}
	}
	input.Roots = roots
	switch input.Transport {
	case domain.MCPTransportStdio:
		if input.Command == "" || !safeMCPExecutable(input.Command) {
			return input, newMCPRegistrationError("invalid_command", "stdio command must be an absolute executable path or a simple executable name without shell syntax")
		}
		if input.URL != "" || input.AuthType != domain.MCPAuthNone || input.BearerTokenEnv != "" {
			return input, newMCPRegistrationError("invalid_stdio_configuration", "stdio registration cannot include a remote URL or HTTP authentication")
		}
		if input.CWD != "" {
			if !filepath.IsAbs(input.CWD) {
				return input, newMCPRegistrationError("invalid_cwd", "stdio cwd must be an existing absolute directory")
			}
			info, err := os.Stat(input.CWD)
			if err != nil || !info.IsDir() {
				return input, newMCPRegistrationError("invalid_cwd", "stdio cwd must be an existing absolute directory")
			}
			input.CWD = filepath.Clean(input.CWD)
		}
	case domain.MCPTransportStreamableHTTP, domain.MCPTransportSSE:
		if input.Command != "" || len(input.Args) > 0 || input.CWD != "" {
			return input, newMCPRegistrationError("invalid_remote_configuration", "remote MCP registration cannot include a command, arguments, or cwd")
		}
		normalizedURL, err := normalizeMCPRegistrationURL(input.URL)
		if err != nil {
			return input, err
		}
		input.URL = normalizedURL
		if input.AuthType != domain.MCPAuthNone && input.AuthType != domain.MCPAuthBearer {
			return input, newMCPRegistrationError("unsupported_auth", "conversational registration supports none or bearer authentication")
		}
		if input.AuthType == domain.MCPAuthBearer {
			if !mcpRegistrationEnvPattern.MatchString(input.BearerTokenEnv) {
				return input, newMCPRegistrationError("invalid_credential_reference", "bearer authentication requires a valid bearerTokenEnv name")
			}
		} else if input.BearerTokenEnv != "" {
			return input, newMCPRegistrationError("invalid_credential_reference", "bearerTokenEnv requires bearer authentication")
		}
	default:
		return input, newMCPRegistrationError("unsupported_transport", "transport must be stdio, streamable_http, or sse")
	}
	return input, nil
}

func safeMCPExecutable(command string) bool {
	if filepath.IsAbs(command) {
		return true
	}
	if len(command) >= 3 && ((command[0] >= 'A' && command[0] <= 'Z') || (command[0] >= 'a' && command[0] <= 'z')) && command[1] == ':' && (command[2] == '\\' || command[2] == '/') {
		return true
	}
	return mcpExecutableNamePattern.MatchString(command)
}

func normalizeMCPRegistrationURL(raw string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", newMCPRegistrationError("invalid_url", "remote MCP URL must have a host and cannot include user info, query, or fragment")
	}
	host := strings.TrimSpace(parsed.Hostname())
	loopback := strings.EqualFold(host, "localhost")
	if ip := net.ParseIP(host); ip != nil && ip.IsLoopback() {
		loopback = true
	}
	if parsed.Scheme != "https" && !(parsed.Scheme == "http" && loopback) {
		return "", newMCPRegistrationError("insecure_url", "remote MCP URL must use HTTPS; HTTP is allowed only for loopback development endpoints")
	}
	parsed.Scheme = strings.ToLower(parsed.Scheme)
	parsed.Host = strings.ToLower(parsed.Host)
	return parsed.String(), nil
}

func looksLikeRawSecretArgument(arg string) bool {
	lower := strings.ToLower(strings.TrimSpace(arg))
	for _, marker := range []string{"authorization:", "bearer ", "api_key=", "apikey=", "access_token=", "token=", "secret=", "password="} {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}

func mcpRegistrationTarget(input domain.MCPRegistrationProposalInput) string {
	if input.Transport != domain.MCPTransportStdio {
		return input.URL
	}
	parts := []string{strconv.Quote(input.Command)}
	for _, arg := range input.Args {
		parts = append(parts, strconv.Quote(arg))
	}
	return strings.Join(parts, " ")
}

func mcpServerFromRegistrationInput(input domain.MCPRegistrationProposalInput) domain.MCPServerConfig {
	return domain.MCPServerConfig{
		ID: input.ID, Name: input.ID, DisplayName: input.DisplayName, Description: input.Description,
		Transport: input.Transport, Command: input.Command, Args: append([]string(nil), input.Args...), CWD: input.CWD,
		URL: input.URL, AuthType: input.AuthType, BearerTokenEnv: input.BearerTokenEnv, Roots: append([]string(nil), input.Roots...),
		TimeoutSeconds: input.TimeoutSeconds, ConnectTimeoutSeconds: input.ConnectTimeoutSeconds,
		Enabled: false, Status: domain.MCPServerStatusDisabled,
	}
}

func (s *Service) ensureMCPRegistrationIdentityAvailable(ctx context.Context, input domain.MCPRegistrationProposalInput) error {
	items, err := s.ListMCPServers(ctx, domain.MCPServerListInput{IncludeDisabled: true})
	if err != nil {
		return newMCPRegistrationError("registration_catalog_unavailable", sanitizeMCPError(err.Error()))
	}
	candidate := mcpServerFromRegistrationInput(input)
	candidatePrefix := mcpServerToolPrefix(candidate)
	for _, item := range items {
		if item.Server.ID == input.ID {
			return newMCPRegistrationError("source_conflict", "an MCP source with this id already exists")
		}
		if mcpServerToolPrefix(item.Server) == candidatePrefix {
			return newMCPRegistrationError("source_name_collision", "MCP source id collides with an existing canonical tool namespace")
		}
	}
	return nil
}

func (s *Service) validateMCPRegistrationNames(ctx context.Context, server domain.MCPServerConfig, tools []domain.MCPToolRecord) ([]string, error) {
	existing := map[string]bool{}
	if registry := s.globalToolCatalogRegistry(ctx); registry != nil {
		for _, entry := range registry.CatalogEntries() {
			existing[entry.Name] = true
		}
	}
	proposed := make([]string, 0, len(tools)+3)
	for _, tool := range tools {
		proposed = append(proposed, mcpToolName(server, tool))
	}
	base := generatedToolName("mcp", "host", firstNonEmptyApp(server.ID, server.Name))
	proposed = append(proposed,
		generatedToolName(base, "list_resources"),
		generatedToolName(base, "list_resource_templates"),
		generatedToolName(base, "read_resource"),
	)
	seen := map[string]bool{}
	for _, name := range proposed {
		if err := validateCanonicalToolName(name); err != nil {
			return nil, newMCPRegistrationError("invalid_tool_identity", err.Error())
		}
		if seen[name] || existing[name] {
			return nil, newMCPRegistrationError("tool_name_collision", "MCP source contributes a canonical tool name that already exists: "+name)
		}
		seen[name] = true
	}
	sort.Strings(proposed)
	return proposed, nil
}

func (s *Service) markMCPRegistrationFailed(ctx context.Context, serverID, message string) {
	if s == nil || s.mcpManager == nil || s.mcpManager.store == nil {
		return
	}
	_, _ = s.mcpManager.store.SetMCPServerEnabled(ctx, serverID, false, domain.MCPServerStatusError, sanitizeMCPError(message))
	if s.mcpManager.connections != nil {
		s.mcpManager.connections.drop(serverID)
	}
}
