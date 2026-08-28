package app

import (
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"aivo/core/domain"
)

const (
	extensionProtocolMaxFrame            = 1 << 20
	extensionProtocolQueueSize           = 64
	extensionServiceHandshakeMaxFrame    = 16 << 10
	extensionServiceHandshakeTimeout     = 10 * time.Second
	extensionServiceHandshakeProtocol    = "aivo-extension-service/1"
	extensionServiceDynamicHTTPTransport = "dynamic-http"
)

type extensionRuntimeClient interface {
	Initialize(context.Context, domain.ExtensionManifest) error
	Execute(context.Context, string, json.RawMessage, domain.ToolExecutionContext) (domain.ToolResult, error)
	UIEvent(context.Context, string, string, any) (any, error)
	Shutdown(context.Context) error
}

type extensionLifecycleClient interface {
	Activate(context.Context) error
	Deactivate(context.Context) error
}

type extensionCatalogClient interface {
	Catalog(context.Context) (map[string]any, error)
}

type ExtensionSupervisor struct {
	mu            sync.Mutex
	items         map[string]*supervisedExtension
	retired       map[string]map[string]*supervisedExtension
	builtins      map[string]func() extensionRuntimeClient
	httpClient    *http.Client
	credentials   *HostCredentialBroker
	idleTimeout   time.Duration
	restartDelays []time.Duration
}

type supervisedExtension struct {
	loaded         LoadedExtension
	status         domain.ExtensionStatus
	client         extensionRuntimeClient
	active         int
	views          int
	cancel         context.CancelFunc
	idleTimer      *time.Timer
	pending        *LoadedExtension
	pendingTrusted bool
}

type extensionContextCandidate struct {
	Key         string
	ExtensionID string
	ContextID   string
	Name        string
	Description string
	Kind        string
}

func NewExtensionSupervisor() *ExtensionSupervisor {
	return &ExtensionSupervisor{items: map[string]*supervisedExtension{}, retired: map[string]map[string]*supervisedExtension{}, builtins: map[string]func() extensionRuntimeClient{}, httpClient: &http.Client{Timeout: 20 * time.Second}, idleTimeout: 5 * time.Minute, restartDelays: []time.Duration{time.Second, 2 * time.Second, 4 * time.Second}}
}

func (s *ExtensionSupervisor) SetCredentialBroker(broker *HostCredentialBroker) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.credentials = broker
}

func (s *ExtensionSupervisor) RegisterBuiltin(id string, factory func() extensionRuntimeClient) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.builtins[id] = factory
}

func (s *ExtensionSupervisor) InstallBuiltin(ctx context.Context, loaded LoadedExtension, factory func() extensionRuntimeClient) (domain.ExtensionStatus, error) {
	if loaded.Manifest.Runtime.Type != domain.ExtensionRuntimeBuiltin || strings.TrimSpace(loaded.Manifest.ID) == "" || factory == nil {
		return domain.ExtensionStatus{}, errors.New("valid built-in extension manifest and factory are required")
	}
	s.mu.Lock()
	if current := s.items[loaded.Manifest.ID]; current != nil {
		if current.loaded.Integrity == loaded.Integrity {
			status := current.status
			s.mu.Unlock()
			return status, nil
		}
		s.mu.Unlock()
		return domain.ExtensionStatus{}, errors.New("built-in extension is already installed with a different integrity")
	}
	s.builtins[loaded.Manifest.ID] = factory
	status := domain.ExtensionStatus{
		ID: loaded.Manifest.ID, Version: loaded.Manifest.Version, State: domain.ExtensionStateValidated,
		Trusted: true, Integrity: loaded.Integrity,
	}
	s.items[loaded.Manifest.ID] = &supervisedExtension{loaded: loaded, status: status}
	s.mu.Unlock()
	return s.Enable(ctx, loaded.Manifest.ID)
}

func (s *ExtensionSupervisor) Discover(path string) (domain.ExtensionStatus, error) {
	loaded, err := LoadExtensionManifest(path)
	if err != nil {
		return domain.ExtensionStatus{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	trusted := loaded.Manifest.Runtime.Type == domain.ExtensionRuntimeBuiltin || loaded.Manifest.Runtime.Type == domain.ExtensionRuntimeStatic
	state := domain.ExtensionStateUntrusted
	if trusted {
		state = domain.ExtensionStateValidated
	}
	status := domain.ExtensionStatus{ID: loaded.Manifest.ID, Version: loaded.Manifest.Version, State: state, Trusted: trusted, Integrity: loaded.Integrity}
	if current := s.items[loaded.Manifest.ID]; current != nil {
		if current.loaded.Integrity == loaded.Integrity {
			return current.status, nil
		}
		current.pending = &loaded
		current.pendingTrusted = trusted
		return status, nil
	}
	s.items[loaded.Manifest.ID] = &supervisedExtension{loaded: loaded, status: status}
	return status, nil
}

func (s *ExtensionSupervisor) Trust(id, integrity string) (domain.ExtensionStatus, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	item, err := s.itemLocked(id)
	if err != nil {
		return domain.ExtensionStatus{}, err
	}
	if item.pending != nil && integrity == item.pending.Integrity {
		item.pendingTrusted = true
		return domain.ExtensionStatus{ID: item.pending.Manifest.ID, Version: item.pending.Manifest.Version, State: domain.ExtensionStateValidated, Trusted: true, Integrity: item.pending.Integrity}, nil
	}
	if integrity != item.loaded.Integrity {
		return item.status, errors.New("extension integrity changed; trust was not granted")
	}
	item.status.Trusted = true
	item.status.State = domain.ExtensionStateValidated
	item.status.Error = ""
	return item.status, nil
}

func (s *ExtensionSupervisor) Enable(ctx context.Context, id string) (domain.ExtensionStatus, error) {
	s.mu.Lock()
	item, err := s.itemLocked(id)
	if err != nil {
		s.mu.Unlock()
		return domain.ExtensionStatus{}, err
	}
	if !item.status.Trusted {
		s.mu.Unlock()
		return item.status, errors.New("extension is not trusted")
	}
	if item.pending != nil {
		if !item.pendingTrusted {
			s.mu.Unlock()
			return domain.ExtensionStatus{ID: item.pending.Manifest.ID, Version: item.pending.Manifest.Version, State: domain.ExtensionStateUntrusted, Integrity: item.pending.Integrity}, errors.New("extension update is not trusted")
		}
		pending := *item.pending
		candidate := &supervisedExtension{loaded: pending, status: domain.ExtensionStatus{ID: pending.Manifest.ID, Version: pending.Manifest.Version, State: domain.ExtensionStateEnabled, Trusted: true, Enabled: true, Integrity: pending.Integrity}}
		s.mu.Unlock()
		status, startErr := s.startItem(ctx, candidate)
		if startErr != nil {
			return status, startErr
		}
		s.mu.Lock()
		current := s.items[id]
		if current == nil || current.pending == nil || current.pending.Integrity != pending.Integrity {
			s.mu.Unlock()
			_ = s.shutdownItem(context.Background(), candidate)
			return domain.ExtensionStatus{}, errors.New("extension update changed during startup")
		}
		if current.idleTimer != nil {
			current.idleTimer.Stop()
			current.idleTimer = nil
		}
		if s.retired[id] == nil {
			s.retired[id] = map[string]*supervisedExtension{}
		}
		s.retired[id][current.loaded.Integrity] = current
		s.items[id] = candidate
		s.scheduleIdleLocked(id, candidate)
		s.mu.Unlock()
		s.scheduleRetiredStop(id, current.loaded.Integrity)
		return candidate.status, nil
	}
	item.status.Enabled = true
	item.status.State = domain.ExtensionStateEnabled
	s.mu.Unlock()
	return s.start(ctx, id)
}

func (s *ExtensionSupervisor) start(ctx context.Context, id string) (domain.ExtensionStatus, error) {
	s.mu.Lock()
	item, err := s.itemLocked(id)
	if err != nil {
		s.mu.Unlock()
		return domain.ExtensionStatus{}, err
	}
	s.mu.Unlock()
	status, err := s.startItem(ctx, item)
	if err != nil {
		return status, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.items[id] == item {
		s.scheduleIdleLocked(id, item)
	}
	return item.status, nil
}

func (s *ExtensionSupervisor) startItem(ctx context.Context, item *supervisedExtension) (domain.ExtensionStatus, error) {
	s.mu.Lock()
	item.status.State = domain.ExtensionStateStarting
	client, err := s.runtimeClientLocked(ctx, item)
	if err != nil {
		item.status.State = domain.ExtensionStateError
		item.status.Error = err.Error()
		status := item.status
		s.mu.Unlock()
		return status, err
	}
	_, cancel := context.WithCancel(context.Background())
	item.client, item.cancel = client, cancel
	s.mu.Unlock()
	initCtx, initCancel := context.WithTimeout(ctx, 20*time.Second)
	defer initCancel()
	err = client.Initialize(initCtx, item.loaded.Manifest)
	if err == nil {
		if catalogClient, ok := client.(extensionCatalogClient); ok {
			var catalog map[string]any
			catalog, err = catalogClient.Catalog(initCtx)
			if err == nil {
				err = validateRuntimeCatalog(item.loaded, catalog)
			}
		}
	}
	s.mu.Lock()
	if err != nil {
		cancel()
		_ = client.Shutdown(context.Background())
		item.client = nil
		item.status.State = domain.ExtensionStateError
		item.status.Error = err.Error()
		status := item.status
		s.mu.Unlock()
		return status, err
	}
	item.status.State = domain.ExtensionStateReady
	item.status.Error = ""
	status := item.status
	s.mu.Unlock()
	return status, nil
}

func (s *ExtensionSupervisor) RegisterReadyTools(id string, registry *Registry) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	item, err := s.itemLocked(id)
	if err != nil {
		return err
	}
	if item.status.State != domain.ExtensionStateReady && item.status.State != domain.ExtensionStateActive {
		return errors.New("extension is not ready")
	}
	tools := make([]domain.Tool, 0, len(item.loaded.Manifest.Contributes.Tools))
	selectionGroups := extensionToolSelectionGroups(item.loaded.Manifest)
	for _, contribution := range item.loaded.Manifest.Contributes.Tools {
		groupDescription := strings.TrimSpace(item.loaded.Manifest.Description)
		spec := domain.ToolSpec{Name: contribution.Name, Description: contribution.Description, InputSchema: domain.CloneRawMap(item.loaded.ToolSchemas[contribution.Name]), Namespace: generatedToolName("extension", id), NamespaceDescription: groupDescription, Capability: contribution.Capability, Category: "extension", Toolsets: []string{"coding", "extension"}, ActivationPolicy: firstNonEmpty(contribution.Activation, "auto"), ImplementationHash: item.loaded.Integrity}
		if group, ok := selectionGroups[contribution.Name]; ok {
			groupCopy := group
			spec.SelectionGroup = &groupCopy
		}
		tools = append(tools, &extensionTool{supervisor: s, extensionID: id, generation: item.loaded.Integrity, spec: spec})
	}
	return registry.RegisterScopedBatch(tools, domain.ToolSourceExtension, id, item.loaded.Manifest.Version)
}

func (s *ExtensionSupervisor) RegisterAllReadyTools(registry *Registry) error {
	s.mu.Lock()
	ids := make([]string, 0, len(s.items))
	for id, item := range s.items {
		if item.status.State == domain.ExtensionStateReady || item.status.State == domain.ExtensionStateActive {
			ids = append(ids, id)
		}
	}
	s.mu.Unlock()
	for _, id := range ids {
		if err := s.RegisterReadyTools(id, registry); err != nil {
			return err
		}
	}
	return nil
}

// InvokeHook runs declared v1 policy contributions in stable extension/policy
// order. Policy failures deny no authority and are excluded from results; a
// policy must return an explicit structured block decision to deny a call.
func (s *ExtensionSupervisor) InvokeHook(ctx context.Context, hook string, payload map[string]any) []map[string]any {
	if s == nil || strings.TrimSpace(hook) == "" {
		return nil
	}
	s.mu.Lock()
	type policyCall struct{ id, generation, policy string }
	calls := make([]policyCall, 0)
	for id, item := range s.items {
		if item.status.State != domain.ExtensionStateReady && item.status.State != domain.ExtensionStateActive {
			continue
		}
		for _, policy := range item.loaded.Manifest.Contributes.Policies {
			calls = append(calls, policyCall{id: id, generation: item.loaded.Integrity, policy: policy})
		}
	}
	s.mu.Unlock()
	sort.Slice(calls, func(i, j int) bool {
		if calls[i].id == calls[j].id {
			return calls[i].policy < calls[j].policy
		}
		return calls[i].id < calls[j].id
	})
	raw, err := json.Marshal(payload)
	if err != nil || len(raw) > extensionProtocolMaxFrame {
		return nil
	}
	results := make([]map[string]any, 0, len(calls))
	for _, call := range calls {
		result := s.executeGeneration(ctx, call.id, call.generation, "policy."+call.policy+"."+hook, raw, domain.ToolExecutionContext{})
		if result.OK && result.Structured != nil {
			results = append(results, result.Structured)
		} else if hook == "pre_tool_call" && !result.OK {
			results = append(results, map[string]any{"block": true, "message": "required policy extension is unavailable"})
		}
	}
	return results
}

func (s *ExtensionSupervisor) ContextResources(id string) ([]domain.ExtensionContextResource, error) {
	s.mu.Lock()
	item, err := s.itemLocked(id)
	if err != nil {
		s.mu.Unlock()
		return nil, err
	}
	if item.status.State != domain.ExtensionStateReady && item.status.State != domain.ExtensionStateActive {
		s.mu.Unlock()
		return nil, errors.New("extension context is not ready")
	}
	loaded := item.loaded
	s.mu.Unlock()
	resources := make([]domain.ExtensionContextResource, 0, len(loaded.Manifest.Contributes.Contexts))
	total := 0
	for _, contribution := range loaded.Manifest.Contributes.Contexts {
		path, err := resolveExtensionPackagePath(loaded.Root, contribution.Path)
		if err != nil {
			return nil, err
		}
		file, err := os.Open(path)
		if err != nil {
			return nil, err
		}
		raw, readErr := io.ReadAll(io.LimitReader(file, 64*1024+1))
		closeErr := file.Close()
		if readErr != nil {
			return nil, readErr
		}
		if closeErr != nil {
			return nil, closeErr
		}
		if len(raw) > 64*1024 || total+len(raw) > 256*1024 {
			return nil, errors.New("extension context exceeds the bounded Host context limit")
		}
		total += len(raw)
		digest := sha256.Sum256(raw)
		resources = append(resources, domain.ExtensionContextResource{ExtensionID: id, ID: contribution.ID, Kind: contribution.Kind, Content: string(bytes.ToValidUTF8(raw, []byte("�"))), SHA256: hex.EncodeToString(digest[:])})
	}
	return resources, nil
}

func (s *ExtensionSupervisor) ContextCatalog() []extensionContextCandidate {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	resources := make([]extensionContextCandidate, 0)
	for extensionID, item := range s.items {
		if !item.status.Enabled || (item.status.State != domain.ExtensionStateReady && item.status.State != domain.ExtensionStateActive) {
			continue
		}
		for _, contribution := range item.loaded.Manifest.Contributes.Contexts {
			contextID := strings.TrimSpace(contribution.ID)
			if contextID == "" {
				continue
			}
			resources = append(resources, extensionContextCandidate{
				Key:         "context:" + extensionID + ":" + contextID,
				ExtensionID: extensionID,
				ContextID:   contextID,
				Name:        firstNonEmpty(item.loaded.Manifest.Name, extensionID) + " / " + contextID,
				Description: strings.TrimSpace(item.loaded.Manifest.Description + " " + contextID + " " + contribution.Kind),
				Kind:        contribution.Kind,
			})
		}
	}
	sort.Slice(resources, func(i, j int) bool { return resources[i].Key < resources[j].Key })
	return resources
}

func (s *ExtensionSupervisor) ResolveView(ctx context.Context, id, viewID string) (domain.ExtensionViewDescriptor, error) {
	s.mu.Lock()
	item, err := s.itemLocked(id)
	if err != nil {
		s.mu.Unlock()
		return domain.ExtensionViewDescriptor{}, err
	}
	if item.status.State != domain.ExtensionStateReady && item.status.State != domain.ExtensionStateActive {
		s.mu.Unlock()
		return domain.ExtensionViewDescriptor{}, errors.New("extension is not ready")
	}
	needsStart := item.client == nil && item.status.Enabled
	s.mu.Unlock()
	if needsStart {
		if _, err := s.start(ctx, id); err != nil {
			return domain.ExtensionViewDescriptor{}, err
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	item, err = s.itemLocked(id)
	if err != nil || item.client == nil {
		return domain.ExtensionViewDescriptor{}, errors.New("extension view runtime is unavailable")
	}
	for _, view := range item.loaded.Manifest.Contributes.Views {
		if view.ID != strings.TrimSpace(viewID) {
			continue
		}
		endpoint, backendToken := extensionServiceEndpoint(item.client)
		if endpoint == "" {
			return domain.ExtensionViewDescriptor{}, errors.New("extension view service URL is unavailable")
		}
		backend := strings.TrimRight(endpoint, "/") + view.Route
		return domain.ExtensionViewDescriptor{ExtensionID: id, ViewID: view.ID, Title: view.Title, LogicalURL: "aivo-extension://" + id + view.Route, BackendURL: backend, BackendToken: backendToken, Surface: append([]string(nil), view.Surfaces...), Actions: append([]string(nil), view.Actions...), Permissions: append([]string(nil), item.loaded.Manifest.Permissions...), CSP: "default-src 'none'; script-src 'self'; style-src 'self'; img-src 'self' data:; font-src 'self'; connect-src 'self'; frame-ancestors 'none'; base-uri 'none'; form-action 'none'"}, nil
	}
	return domain.ExtensionViewDescriptor{}, errors.New("extension view not found")
}

func (s *ExtensionSupervisor) ToolViewRef(id, generation, name string) *domain.ExtensionToolViewRef {
	s.mu.Lock()
	defer s.mu.Unlock()
	id = strings.TrimSpace(id)
	name = strings.TrimSpace(name)
	if id == "" || name == "" {
		return nil
	}
	item := s.generationLocked(id, strings.TrimSpace(generation))
	if item == nil {
		return nil
	}
	return extensionToolViewRef(item.loaded.Manifest, name)
}

func extensionToolViewRef(manifest domain.ExtensionManifest, name string) *domain.ExtensionToolViewRef {
	var page *domain.ExtensionToolViewRef
	for _, view := range manifest.Contributes.Views {
		associated := false
		for _, toolName := range view.Tools {
			if toolName == name {
				associated = true
				break
			}
		}
		if !associated {
			continue
		}
		for _, surface := range view.Surfaces {
			ref := &domain.ExtensionToolViewRef{ExtensionID: manifest.ID, ViewID: view.ID, Surface: surface, Title: view.Title}
			if surface == "tool-detail" {
				return ref
			}
			if surface == "page" && page == nil {
				page = ref
			}
		}
	}
	return page
}

func (s *ExtensionSupervisor) OpenView(ctx context.Context, id string) error {
	s.mu.Lock()
	item, err := s.itemLocked(id)
	if err != nil {
		s.mu.Unlock()
		return err
	}
	if item.client == nil || (item.status.State != domain.ExtensionStateReady && item.status.State != domain.ExtensionStateActive) {
		s.mu.Unlock()
		return errors.New("extension is not ready")
	}
	needActivate := item.active == 0 && item.views == 0
	item.views++
	if item.idleTimer != nil {
		item.idleTimer.Stop()
		item.idleTimer = nil
	}
	item.status.State = domain.ExtensionStateActive
	client := item.client
	s.mu.Unlock()
	if needActivate {
		if lifecycle, ok := client.(extensionLifecycleClient); ok {
			if err := lifecycle.Activate(ctx); err != nil {
				_ = s.CloseView(context.Background(), id)
				return err
			}
		}
	}
	return nil
}

func (s *ExtensionSupervisor) CloseView(ctx context.Context, id string) error {
	s.mu.Lock()
	item, err := s.itemLocked(id)
	if err != nil {
		s.mu.Unlock()
		return err
	}
	if item.views > 0 {
		item.views--
	}
	needDeactivate := item.views == 0 && item.active == 0
	client := item.client
	if needDeactivate && item.status.State == domain.ExtensionStateActive {
		item.status.State = domain.ExtensionStateReady
		s.scheduleIdleLocked(id, item)
	}
	s.mu.Unlock()
	if needDeactivate {
		if lifecycle, ok := client.(extensionLifecycleClient); ok {
			return lifecycle.Deactivate(ctx)
		}
	}
	return nil
}

func (s *ExtensionSupervisor) InvokeViewAction(ctx context.Context, id, viewID, action string, data any) (any, error) {
	s.mu.Lock()
	item, err := s.itemLocked(id)
	if err != nil {
		s.mu.Unlock()
		return nil, err
	}
	if item.client == nil || (item.status.State != domain.ExtensionStateReady && item.status.State != domain.ExtensionStateActive) {
		s.mu.Unlock()
		return nil, errors.New("extension is not ready")
	}
	declared := false
	for _, view := range item.loaded.Manifest.Contributes.Views {
		if view.ID != viewID {
			continue
		}
		for _, candidate := range view.Actions {
			if candidate == action {
				declared = true
			}
		}
	}
	if !declared {
		s.mu.Unlock()
		return nil, errors.New("extension view action is not declared")
	}
	client := item.client
	s.mu.Unlock()
	return client.UIEvent(ctx, viewID, action, data)
}

func (s *ExtensionSupervisor) Execute(ctx context.Context, id, name string, args json.RawMessage, execCtx domain.ToolExecutionContext) domain.ToolResult {
	s.mu.Lock()
	item := s.items[id]
	generation := ""
	if item != nil {
		generation = item.loaded.Integrity
	}
	s.mu.Unlock()
	return s.executeGeneration(ctx, id, generation, name, args, execCtx)
}

func (s *ExtensionSupervisor) executeGeneration(ctx context.Context, id, generation, name string, args json.RawMessage, execCtx domain.ToolExecutionContext) (result domain.ToolResult) {
	s.mu.Lock()
	item := s.generationLocked(id, generation)
	if item == nil {
		s.mu.Unlock()
		return primitiveError(name, "extension_unavailable", errors.New("extension generation is unavailable"))
	}
	viewRef := extensionToolViewRef(item.loaded.Manifest, name)
	defer func() {
		if viewRef != nil {
			result.Details = &domain.ToolResultDetails{View: viewRef}
		}
	}()
	if item.client == nil && item.status.Enabled && item.status.State == domain.ExtensionStateReady {
		s.mu.Unlock()
		if _, startErr := s.startItem(ctx, item); startErr != nil {
			return primitiveError(name, "extension_unavailable", startErr)
		}
		s.mu.Lock()
		if s.items[id] == item {
			s.scheduleIdleLocked(id, item)
		}
		s.mu.Unlock()
		return s.executeGeneration(ctx, id, generation, name, args, execCtx)
	}
	if item.client == nil || (item.status.State != domain.ExtensionStateReady && item.status.State != domain.ExtensionStateActive) {
		s.mu.Unlock()
		return primitiveError(name, "extension_unavailable", errors.New("extension is not ready"))
	}
	needActivate := item.active == 0 && item.views == 0
	item.active++
	if item.idleTimer != nil {
		item.idleTimer.Stop()
		item.idleTimer = nil
	}
	item.status.State = domain.ExtensionStateActive
	client := item.client
	s.mu.Unlock()
	if s.credentials != nil {
		defer s.credentials.Release(execCtx.ToolCallID)
	}
	if needActivate {
		if lifecycle, ok := client.(extensionLifecycleClient); ok {
			if activateErr := lifecycle.Activate(ctx); activateErr != nil {
				s.finishExtensionCall(id, generation, client)
				_ = s.restartAfterFailure(ctx, id, generation)
				return primitiveError(name, "extension_activate_failed", activateErr)
			}
		}
	}
	result, callErr := client.Execute(ctx, name, args, execCtx)
	s.finishExtensionCall(id, generation, client)
	if callErr != nil {
		if errors.Is(callErr, context.Canceled) || errors.Is(callErr, context.DeadlineExceeded) || ctx.Err() != nil {
			return primitiveError(name, "cancelled", callErr)
		}
		_ = s.restartAfterFailure(ctx, id, generation)
		return primitiveError(name, "extension_call_failed", callErr)
	}
	result.Name = name
	return result
}

func (s *ExtensionSupervisor) finishExtensionCall(id, generation string, client extensionRuntimeClient) {
	s.mu.Lock()
	item := s.generationLocked(id, generation)
	if item == nil {
		s.mu.Unlock()
		return
	}
	if item.active > 0 {
		item.active--
	}
	needDeactivate := item.active == 0 && item.views == 0
	if needDeactivate && item.status.State == domain.ExtensionStateActive {
		item.status.State = domain.ExtensionStateReady
		if s.items[id] == item {
			s.scheduleIdleLocked(id, item)
		}
	}
	s.mu.Unlock()
	if needDeactivate {
		if lifecycle, ok := client.(extensionLifecycleClient); ok {
			deactivateCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			_ = lifecycle.Deactivate(deactivateCtx)
			cancel()
		}
	}
}

func (s *ExtensionSupervisor) Stop(ctx context.Context, id string) (domain.ExtensionStatus, error) {
	status, err := s.stopRuntime(ctx, id, true)
	s.mu.Lock()
	retired := s.retired[id]
	delete(s.retired, id)
	s.mu.Unlock()
	for _, item := range retired {
		_ = s.shutdownItem(ctx, item)
	}
	return status, err
}

func (s *ExtensionSupervisor) stopRuntime(ctx context.Context, id string, disable bool) (domain.ExtensionStatus, error) {
	s.mu.Lock()
	item, err := s.itemLocked(id)
	if err != nil {
		s.mu.Unlock()
		return domain.ExtensionStatus{}, err
	}
	item.status.State = domain.ExtensionStateDraining
	if item.idleTimer != nil {
		item.idleTimer.Stop()
		item.idleTimer = nil
	}
	client, cancel := item.client, item.cancel
	s.mu.Unlock()
	deadline := time.NewTimer(5 * time.Second)
	defer deadline.Stop()
drainLoop:
	for {
		s.mu.Lock()
		active := item.active + item.views
		s.mu.Unlock()
		if active == 0 {
			break
		}
		select {
		case <-ctx.Done():
			if cancel != nil {
				cancel()
			}
			break drainLoop
		case <-deadline.C:
			if cancel != nil {
				cancel()
			}
			break drainLoop
		case <-time.After(10 * time.Millisecond):
		}
	}
	var stopErr error
	if client != nil {
		stopCtx, stopCancel := context.WithTimeout(context.Background(), 2*time.Second)
		if lifecycle, ok := client.(extensionLifecycleClient); ok {
			_ = lifecycle.Deactivate(stopCtx)
		}
		stopErr = client.Shutdown(stopCtx)
		stopCancel()
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	item.client = nil
	item.cancel = nil
	item.active = 0
	item.views = 0
	item.status.Enabled = !disable
	if disable {
		item.status.State = domain.ExtensionStateStopped
	} else {
		item.status.State = domain.ExtensionStateReady
	}
	if stopErr != nil {
		item.status.State = domain.ExtensionStateError
		item.status.Error = stopErr.Error()
	}
	return item.status, stopErr
}

func (s *ExtensionSupervisor) scheduleIdleLocked(id string, item *supervisedExtension) {
	if s.idleTimeout <= 0 || item == nil || item.active != 0 {
		return
	}
	if item.idleTimer != nil {
		item.idleTimer.Stop()
	}
	item.idleTimer = time.AfterFunc(s.idleTimeout, func() { _, _ = s.stopRuntime(context.Background(), id, false) })
}

func (s *ExtensionSupervisor) scheduleRetiredStop(id, generation string) {
	time.AfterFunc(5*time.Minute, func() {
		s.mu.Lock()
		item := s.retired[id][generation]
		if item != nil {
			delete(s.retired[id], generation)
			if len(s.retired[id]) == 0 {
				delete(s.retired, id)
			}
		}
		s.mu.Unlock()
		if item != nil {
			_ = s.shutdownItem(context.Background(), item)
		}
	})
}

func (s *ExtensionSupervisor) shutdownItem(ctx context.Context, item *supervisedExtension) error {
	if item == nil {
		return nil
	}
	s.mu.Lock()
	if item.idleTimer != nil {
		item.idleTimer.Stop()
		item.idleTimer = nil
	}
	client, cancel := item.client, item.cancel
	item.client, item.cancel = nil, nil
	item.active, item.views = 0, 0
	s.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	if client == nil {
		return nil
	}
	stopCtx, stopCancel := context.WithTimeout(ctx, 2*time.Second)
	defer stopCancel()
	if lifecycle, ok := client.(extensionLifecycleClient); ok {
		_ = lifecycle.Deactivate(stopCtx)
	}
	err := client.Shutdown(stopCtx)
	s.mu.Lock()
	item.status.State = domain.ExtensionStateStopped
	item.status.Enabled = false
	if err != nil {
		item.status.State = domain.ExtensionStateError
		item.status.Error = err.Error()
	}
	s.mu.Unlock()
	return err
}

func (s *ExtensionSupervisor) restartAfterFailure(ctx context.Context, id, generation string) error {
	s.mu.Lock()
	item := s.generationLocked(id, generation)
	if item == nil {
		s.mu.Unlock()
		return errors.New("extension generation is unavailable")
	}
	if s.items[id] != item {
		s.mu.Unlock()
		return s.shutdownItem(ctx, item)
	}
	oldClient, oldCancel := item.client, item.cancel
	item.client, item.cancel = nil, nil
	item.status.State, item.status.Error = domain.ExtensionStateError, "extension runtime failed"
	delays := append([]time.Duration(nil), s.restartDelays...)
	s.mu.Unlock()
	if oldCancel != nil {
		oldCancel()
	}
	if oldClient != nil {
		_ = oldClient.Shutdown(context.Background())
	}
	var lastErr error
	for _, delay := range delays {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}
		_, lastErr = s.start(ctx, id)
		if lastErr == nil {
			return nil
		}
	}
	return lastErr
}

func (s *ExtensionSupervisor) Status(id string) (domain.ExtensionStatus, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	item, err := s.itemLocked(id)
	if err != nil {
		return domain.ExtensionStatus{}, err
	}
	return item.status, nil
}

func (s *ExtensionSupervisor) Remove(ctx context.Context, id string) error {
	s.mu.Lock()
	item, err := s.itemLocked(id)
	if err != nil {
		s.mu.Unlock()
		return err
	}
	if item.loaded.Manifest.Runtime.Type == domain.ExtensionRuntimeBuiltin {
		s.mu.Unlock()
		return errors.New("built-in extensions cannot be removed")
	}
	s.mu.Unlock()
	_, stopErr := s.Stop(ctx, id)
	s.mu.Lock()
	retired := make([]*supervisedExtension, 0, len(s.retired[id]))
	for _, generation := range s.retired[id] {
		retired = append(retired, generation)
	}
	delete(s.items, id)
	delete(s.retired, id)
	s.mu.Unlock()
	for _, generation := range retired {
		if err := s.shutdownItem(ctx, generation); stopErr == nil && err != nil {
			stopErr = err
		}
	}
	return stopErr
}

func (s *ExtensionSupervisor) itemLocked(id string) (*supervisedExtension, error) {
	item := s.items[strings.TrimSpace(id)]
	if item == nil {
		return nil, errors.New("extension not found")
	}
	return item, nil
}

func (s *ExtensionSupervisor) generationLocked(id, generation string) *supervisedExtension {
	current := s.items[strings.TrimSpace(id)]
	if current != nil && (generation == "" || current.loaded.Integrity == generation) {
		return current
	}
	return s.retired[strings.TrimSpace(id)][generation]
}

func (s *ExtensionSupervisor) runtimeClientLocked(ctx context.Context, item *supervisedExtension) (extensionRuntimeClient, error) {
	switch item.loaded.Manifest.Runtime.Type {
	case domain.ExtensionRuntimeBuiltin:
		factory := s.builtins[item.loaded.Manifest.ID]
		if factory == nil {
			return nil, errors.New("builtin extension implementation is unavailable")
		}
		return factory(), nil
	case domain.ExtensionRuntimeProcess:
		client, err := newProcessExtensionClient(item.loaded)
		if err == nil {
			client.credentials = s.credentials
			client.extensionID = item.loaded.Manifest.ID
			client.credentialSlots = append([]string(nil), item.loaded.Manifest.Requirements.Credentials...)
		}
		return client, err
	case domain.ExtensionRuntimeService:
		return newSupervisedServiceExtensionClient(ctx, item.loaded, s.httpClient)
	case domain.ExtensionRuntimeExternal:
		if s.credentials == nil || len(item.loaded.Manifest.Requirements.Credentials) == 0 {
			return nil, errors.New("external extension credential broker is unavailable")
		}
		opID := "extension-start:" + uuid.NewString()
		credential, err := s.credentials.Request(ctx, item.loaded.Manifest.ID, item.loaded.Manifest.Requirements.Credentials, item.loaded.Manifest.Requirements.Credentials[0], opID)
		s.credentials.Release(opID)
		if err != nil {
			return nil, fmt.Errorf("external extension credential is unavailable: %w", err)
		}
		return newServiceExtensionClient(item.loaded.Manifest.Runtime.URL, credential, s.httpClient)
	case domain.ExtensionRuntimeStatic:
		return staticExtensionClient{}, nil
	default:
		return nil, errors.New("unsupported extension runtime")
	}
}

type extensionTool struct {
	supervisor  *ExtensionSupervisor
	extensionID string
	generation  string
	spec        domain.ToolSpec
}

func (t *extensionTool) Spec() domain.ToolSpec { return t.spec }
func (t *extensionTool) Execute(ctx context.Context, args json.RawMessage, execCtx domain.ToolExecutionContext) domain.ToolResult {
	return t.supervisor.executeGeneration(ctx, t.extensionID, t.generation, t.spec.Name, args, execCtx)
}

type staticExtensionClient struct{}

func (staticExtensionClient) Initialize(context.Context, domain.ExtensionManifest) error { return nil }
func (staticExtensionClient) Execute(context.Context, string, json.RawMessage, domain.ToolExecutionContext) (domain.ToolResult, error) {
	return domain.ToolResult{}, errors.New("static extension has no executable tools")
}
func (staticExtensionClient) UIEvent(context.Context, string, string, any) (any, error) {
	return nil, errors.New("static extension has no Web runtime")
}
func (staticExtensionClient) Shutdown(context.Context) error { return nil }

type processExtensionClient struct {
	cmd             *exec.Cmd
	stdin           io.WriteCloser
	stdout          *bufio.Reader
	mu              sync.Mutex
	credentials     *HostCredentialBroker
	extensionID     string
	credentialSlots []string
}

func newProcessExtensionClient(loaded LoadedExtension) (*processExtensionClient, error) {
	command := loaded.Manifest.Runtime.Command
	if strings.ContainsAny(command, `/\`) {
		path, err := resolveExtensionPackagePath(loaded.Root, command)
		if err != nil {
			return nil, err
		}
		command = path
	}
	cmd := exec.Command(command, loaded.Manifest.Runtime.Args...)
	cmd.Dir = loaded.Root
	cmd.Env = SanitizedEnvironment(loaded.Root, defaultEnvAllowlist(), nil, nil)
	cmd.Stderr = io.Discard
	setProcessGroup(cmd)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	return &processExtensionClient{cmd: cmd, stdin: stdin, stdout: bufio.NewReaderSize(stdout, extensionProtocolMaxFrame+1)}, nil
}

func (c *processExtensionClient) Initialize(ctx context.Context, manifest domain.ExtensionManifest) error {
	_, err := c.call(ctx, "extension/initialize", map[string]any{"apiVersion": manifest.APIVersion, "extensionId": manifest.ID, "extensionVersion": manifest.Version}, "")
	return err
}
func (c *processExtensionClient) Activate(ctx context.Context) error {
	_, err := c.call(ctx, "extension/activate", nil, "")
	return err
}
func (c *processExtensionClient) Catalog(ctx context.Context) (map[string]any, error) {
	return c.call(ctx, "catalog/list", nil, "")
}
func (c *processExtensionClient) Deactivate(ctx context.Context) error {
	_, err := c.call(ctx, "extension/deactivate", nil, "")
	return err
}
func (c *processExtensionClient) Execute(ctx context.Context, name string, args json.RawMessage, execCtx domain.ToolExecutionContext) (domain.ToolResult, error) {
	var arguments any
	if err := json.Unmarshal(args, &arguments); err != nil {
		return domain.ToolResult{}, err
	}
	result, err := c.call(ctx, "tool/execute", map[string]any{"name": name, "arguments": arguments, "sessionId": execCtx.SessionID, "turnId": execCtx.TurnID, "operationId": execCtx.ToolCallID, "toolSnapshot": execCtx.ToolSnapshot}, execCtx.ToolCallID)
	if err != nil {
		return domain.ToolResult{}, err
	}
	return normalizeExternalToolResult(name, result), nil
}
func (c *processExtensionClient) UIEvent(ctx context.Context, viewID, action string, data any) (any, error) {
	return c.call(ctx, "ui/event", map[string]any{"viewId": viewID, "action": action, "data": data}, "")
}
func (c *processExtensionClient) Shutdown(ctx context.Context) error {
	_, _ = c.call(ctx, "extension/shutdown", nil, "")
	if c.stdin != nil {
		_ = c.stdin.Close()
	}
	if c.cmd != nil && c.cmd.Process != nil {
		_ = killProcessGroup(c.cmd.Process)
	}
	return nil
}

func (c *processExtensionClient) call(ctx context.Context, method string, params map[string]any, operationID string) (map[string]any, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	id := uuid.NewString()
	raw, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": id, "method": method, "params": params})
	if len(raw) > extensionProtocolMaxFrame {
		return nil, errors.New("extension request frame exceeds 1 MiB")
	}
	if _, err := c.stdin.Write(append(raw, '\n')); err != nil {
		return nil, err
	}
	type readResult struct {
		line []byte
		err  error
	}
	channel := make(chan readResult, 1)
	go func() { line, err := c.stdout.ReadBytes('\n'); channel <- readResult{line: line, err: err} }()
	select {
	case <-ctx.Done():
		cancelRaw, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "method": "tool/cancel", "params": map[string]any{"requestId": id}})
		_, _ = c.stdin.Write(append(cancelRaw, '\n'))
		select {
		case <-channel:
		case <-time.After(100 * time.Millisecond):
			if c.cmd != nil && c.cmd.Process != nil {
				_ = killProcessGroup(c.cmd.Process)
			}
		}
		return nil, ctx.Err()
	case outcome := <-channel:
		progressCount, auxiliaryFrames := 0, 0
		for {
			if outcome.err != nil {
				return nil, outcome.err
			}
			if len(outcome.line) > extensionProtocolMaxFrame {
				return nil, errors.New("extension response frame exceeds 1 MiB")
			}
			var response struct {
				JSONRPC string          `json:"jsonrpc"`
				ID      json.RawMessage `json:"id"`
				Method  string          `json:"method"`
				Params  json.RawMessage `json:"params"`
				Result  map[string]any  `json:"result"`
				Error   *struct {
					Code    int    `json:"code"`
					Message string `json:"message"`
				} `json:"error"`
			}
			if err := json.Unmarshal(bytes.TrimSpace(outcome.line), &response); err != nil {
				return nil, err
			}
			if response.JSONRPC != "2.0" {
				return nil, errors.New("invalid extension JSON-RPC version")
			}
			if response.Method != "" {
				auxiliaryFrames++
				if auxiliaryFrames > extensionProtocolQueueSize {
					return nil, errors.New("extension auxiliary frame queue exceeded 64 frames")
				}
				if response.Method == "tool/progress" {
					progressCount++
					if progressCount > 16 {
						return nil, errors.New("extension progress exceeded 16 pending events")
					}
				} else if response.Method == "credential/request" {
					if err := c.handleCredentialRequest(ctx, response.ID, response.Params, operationID); err != nil {
						return nil, err
					}
				} else {
					return nil, fmt.Errorf("unsupported extension host request %q", response.Method)
				}
				next := make(chan readResult, 1)
				go func() { line, readErr := c.stdout.ReadBytes('\n'); next <- readResult{line: line, err: readErr} }()
				select {
				case <-ctx.Done():
					cancelRaw, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "method": "tool/cancel", "params": map[string]any{"requestId": id}})
					_, _ = c.stdin.Write(append(cancelRaw, '\n'))
					if c.cmd != nil && c.cmd.Process != nil {
						_ = killProcessGroup(c.cmd.Process)
					}
					return nil, ctx.Err()
				case outcome = <-next:
				}
				continue
			}
			var responseID string
			if err := json.Unmarshal(response.ID, &responseID); err != nil || responseID != id {
				return nil, errors.New("invalid extension JSON-RPC response identity")
			}
			if response.Error != nil {
				return nil, fmt.Errorf("extension error %d: %s", response.Error.Code, response.Error.Message)
			}
			return response.Result, nil
		}
	}
}

func (c *processExtensionClient) handleCredentialRequest(ctx context.Context, requestID, params json.RawMessage, allowedOperationID string) error {
	if len(requestID) == 0 || c.credentials == nil || strings.TrimSpace(allowedOperationID) == "" {
		return errors.New("extension credential request is not allowed for this operation")
	}
	var input struct {
		Slot        string `json:"slot"`
		OperationID string `json:"operationId"`
	}
	if err := json.Unmarshal(params, &input); err != nil || input.OperationID != allowedOperationID {
		return errors.New("extension credential request has an invalid operation owner")
	}
	value, err := c.credentials.Request(ctx, c.extensionID, c.credentialSlots, input.Slot, input.OperationID)
	response := map[string]any{"jsonrpc": "2.0", "id": json.RawMessage(requestID)}
	if err != nil {
		response["error"] = map[string]any{"code": -32001, "message": "credential request denied"}
	} else {
		response["result"] = map[string]any{"value": value}
	}
	raw, _ := json.Marshal(response)
	if len(raw) > extensionProtocolMaxFrame {
		return errors.New("credential response frame exceeds 1 MiB")
	}
	_, writeErr := c.stdin.Write(append(raw, '\n'))
	return writeErr
}

type serviceExtensionClient struct {
	endpoint, bearer string
	client           *http.Client
}

type supervisedServiceExtensionClient struct {
	*serviceExtensionClient
	cmd    *exec.Cmd
	bearer string
}

func newSupervisedServiceExtensionClient(ctx context.Context, loaded LoadedExtension, httpClient *http.Client) (*supervisedServiceExtensionClient, error) {
	tokenRaw := make([]byte, 32)
	if _, err := rand.Read(tokenRaw); err != nil {
		return nil, err
	}
	bearer := hex.EncodeToString(tokenRaw)
	command := loaded.Manifest.Runtime.Command
	if strings.ContainsAny(command, `/\`) {
		resolved, err := resolveExtensionPackagePath(loaded.Root, command)
		if err != nil {
			return nil, err
		}
		command = resolved
	}
	cmd := exec.Command(command, loaded.Manifest.Runtime.Args...)
	cmd.Dir = loaded.Root
	cmd.Env = append(SanitizedEnvironment(loaded.Root, defaultEnvAllowlist(), nil, nil), "AIVO_EXTENSION_BEARER_TOKEN="+bearer)
	cmd.Stderr = io.Discard
	setProcessGroup(cmd)
	dynamicEndpoint := strings.TrimSpace(loaded.Manifest.Runtime.Transport) == extensionServiceDynamicHTTPTransport
	var stdout io.ReadCloser
	var err error
	if dynamicEndpoint {
		stdout, err = cmd.StdoutPipe()
		if err != nil {
			return nil, err
		}
	} else {
		cmd.Stdout = io.Discard
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	endpoint := loaded.Manifest.Runtime.URL
	if dynamicEndpoint {
		endpoint, err = readDynamicServiceEndpoint(ctx, stdout)
		_ = stdout.Close()
		if err != nil {
			_ = killProcessGroup(cmd.Process)
			return nil, err
		}
	}
	client, err := newServiceExtensionClient(endpoint, bearer, httpClient)
	if err != nil {
		_ = killProcessGroup(cmd.Process)
		return nil, err
	}
	return &supervisedServiceExtensionClient{serviceExtensionClient: client, cmd: cmd, bearer: bearer}, nil
}

func readDynamicServiceEndpoint(ctx context.Context, stdout io.ReadCloser) (string, error) {
	return readDynamicServiceEndpointWithTimeout(ctx, stdout, extensionServiceHandshakeTimeout)
}

func readDynamicServiceEndpointWithTimeout(ctx context.Context, stdout io.ReadCloser, timeout time.Duration) (string, error) {
	if stdout == nil {
		return "", errors.New("dynamic extension service readiness stream is unavailable")
	}
	readCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	type readResult struct {
		line []byte
		err  error
	}
	resultChannel := make(chan readResult, 1)
	go func() {
		reader := bufio.NewReader(io.LimitReader(stdout, extensionServiceHandshakeMaxFrame+1))
		line, err := reader.ReadBytes('\n')
		resultChannel <- readResult{line: line, err: err}
	}()
	var result readResult
	select {
	case <-readCtx.Done():
		_ = stdout.Close()
		return "", fmt.Errorf("dynamic extension service readiness timed out: %w", readCtx.Err())
	case result = <-resultChannel:
	}
	if len(result.line) > extensionServiceHandshakeMaxFrame {
		return "", errors.New("dynamic extension service readiness exceeds 16 KiB")
	}
	if result.err != nil {
		return "", fmt.Errorf("dynamic extension service readiness ended before one frame: %w", result.err)
	}
	if len(result.line) == 0 || result.line[len(result.line)-1] != '\n' {
		return "", errors.New("dynamic extension service readiness must be newline terminated")
	}
	var handshake struct {
		Protocol string `json:"protocol"`
		URL      string `json:"url"`
	}
	if err := json.Unmarshal(bytes.TrimSpace(result.line), &handshake); err != nil {
		return "", errors.New("dynamic extension service readiness is invalid JSON")
	}
	if handshake.Protocol != extensionServiceHandshakeProtocol {
		return "", errors.New("dynamic extension service readiness protocol is unsupported")
	}
	return validateDynamicServiceEndpoint(handshake.URL)
}

func validateDynamicServiceEndpoint(endpoint string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(endpoint))
	if err != nil || parsed.Scheme != "http" || parsed.Opaque != "" || parsed.User != nil || !extensionLoopbackHost(parsed.Hostname()) {
		return "", errors.New("dynamic extension service must announce loopback HTTP")
	}
	port, err := strconv.Atoi(parsed.Port())
	if err != nil || port < 1 || port > 65535 {
		return "", errors.New("dynamic extension service must announce an explicit non-zero port")
	}
	if (parsed.Path != "" && parsed.Path != "/") || parsed.RawPath != "" || parsed.RawQuery != "" || parsed.Fragment != "" || parsed.ForceQuery {
		return "", errors.New("dynamic extension service must announce a root origin without query or fragment")
	}
	return parsed.Scheme + "://" + parsed.Host, nil
}

func extensionServiceEndpoint(client extensionRuntimeClient) (string, string) {
	switch service := client.(type) {
	case *supervisedServiceExtensionClient:
		return service.endpoint, service.bearer
	case *serviceExtensionClient:
		return service.endpoint, service.bearer
	default:
		return "", ""
	}
}

func (c *supervisedServiceExtensionClient) Initialize(ctx context.Context, manifest domain.ExtensionManifest) error {
	var lastErr error
	for attempt := 0; attempt < 20; attempt++ {
		if err := c.serviceExtensionClient.Initialize(ctx, manifest); err == nil {
			return nil
		} else {
			lastErr = err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(100 * time.Millisecond):
		}
	}
	return lastErr
}

func (c *supervisedServiceExtensionClient) Shutdown(ctx context.Context) error {
	err := c.serviceExtensionClient.Shutdown(ctx)
	if c.cmd != nil && c.cmd.Process != nil {
		_ = killProcessGroup(c.cmd.Process)
	}
	return err
}

func newServiceExtensionClient(endpoint, bearer string, client *http.Client) (*serviceExtensionClient, error) {
	parsed, err := url.Parse(endpoint)
	if err != nil {
		return nil, err
	}
	if parsed.Scheme != "https" && !(parsed.Scheme == "http" && (parsed.Hostname() == "127.0.0.1" || parsed.Hostname() == "localhost" || parsed.Hostname() == "::1")) {
		return nil, errors.New("extension service must use HTTPS or loopback HTTP")
	}
	return &serviceExtensionClient{endpoint: endpoint, bearer: bearer, client: client}, nil
}
func (c *serviceExtensionClient) Initialize(ctx context.Context, manifest domain.ExtensionManifest) error {
	_, err := c.call(ctx, "extension/initialize", map[string]any{"apiVersion": manifest.APIVersion, "extensionId": manifest.ID, "extensionVersion": manifest.Version})
	return err
}
func (c *serviceExtensionClient) Activate(ctx context.Context) error {
	_, err := c.call(ctx, "extension/activate", nil)
	return err
}
func (c *serviceExtensionClient) Catalog(ctx context.Context) (map[string]any, error) {
	return c.call(ctx, "catalog/list", nil)
}
func (c *serviceExtensionClient) Deactivate(ctx context.Context) error {
	_, err := c.call(ctx, "extension/deactivate", nil)
	return err
}
func (c *serviceExtensionClient) Execute(ctx context.Context, name string, args json.RawMessage, execCtx domain.ToolExecutionContext) (domain.ToolResult, error) {
	var arguments any
	if err := json.Unmarshal(args, &arguments); err != nil {
		return domain.ToolResult{}, err
	}
	result, err := c.call(ctx, "tool/execute", map[string]any{"name": name, "arguments": arguments, "sessionId": execCtx.SessionID, "operationId": execCtx.ToolCallID, "toolSnapshot": execCtx.ToolSnapshot})
	if err != nil {
		return domain.ToolResult{}, err
	}
	return normalizeExternalToolResult(name, result), nil
}
func (c *serviceExtensionClient) UIEvent(ctx context.Context, viewID, action string, data any) (any, error) {
	return c.call(ctx, "ui/event", map[string]any{"viewId": viewID, "action": action, "data": data})
}
func (c *serviceExtensionClient) Shutdown(ctx context.Context) error {
	_, err := c.call(ctx, "extension/shutdown", nil)
	return err
}
func (c *serviceExtensionClient) call(ctx context.Context, method string, params map[string]any) (map[string]any, error) {
	id := uuid.NewString()
	raw, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": id, "method": method, "params": params})
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	request.Header.Set("Content-Type", "application/json")
	if c.bearer != "" {
		request.Header.Set("Authorization", "Bearer "+c.bearer)
	}
	response, err := c.client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, fmt.Errorf("extension service returned HTTP %d", response.StatusCode)
	}
	limited := io.LimitReader(response.Body, extensionProtocolMaxFrame+1)
	responseRaw, err := io.ReadAll(limited)
	if err != nil {
		return nil, err
	}
	if len(responseRaw) > extensionProtocolMaxFrame {
		return nil, errors.New("extension response frame exceeds 1 MiB")
	}
	var envelope struct {
		JSONRPC string         `json:"jsonrpc"`
		ID      string         `json:"id"`
		Result  map[string]any `json:"result"`
		Error   *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(responseRaw, &envelope); err != nil {
		return nil, err
	}
	if envelope.JSONRPC != "2.0" || envelope.ID != id {
		return nil, errors.New("invalid extension JSON-RPC response identity")
	}
	if envelope.Error != nil {
		return nil, fmt.Errorf("extension error %d: %s", envelope.Error.Code, envelope.Error.Message)
	}
	return envelope.Result, nil
}

var _ = filepath.Separator
var _ = os.ErrNotExist

func validateRuntimeCatalog(loaded LoadedExtension, catalog map[string]any) error {
	rawTools, ok := catalog["tools"].([]any)
	if !ok {
		return errors.New("extension catalog/list must return a tools array")
	}
	declared := map[string]string{}
	for name, schema := range loaded.ToolSchemas {
		raw, _ := json.Marshal(schema)
		sum := sha256.Sum256(raw)
		declared[name] = hex.EncodeToString(sum[:])
	}
	seen := map[string]bool{}
	for _, rawTool := range rawTools {
		entry, ok := rawTool.(map[string]any)
		if !ok {
			return errors.New("extension catalog tool entry must be an object")
		}
		name, _ := entry["name"].(string)
		schemaHash, _ := entry["schemaHash"].(string)
		if declared[name] == "" || seen[name] {
			return fmt.Errorf("runtime catalog returned undeclared or duplicate tool %q", name)
		}
		if schemaHash != declared[name] {
			return fmt.Errorf("runtime catalog schema drift for tool %q", name)
		}
		seen[name] = true
	}
	if len(seen) != len(declared) {
		return errors.New("runtime catalog does not match declared tools")
	}
	return nil
}
