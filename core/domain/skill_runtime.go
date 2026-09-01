package domain

const (
	SkillScopeGlobal  = "global"
	SkillScopeProject = "project"

	SkillSourceAivo        = "aivo"
	SkillSourceClaude      = "claude"
	SkillSourceAgents      = "agents"
	SkillSourceAivoSystem  = "aivo-system"
	SkillSourceCodex       = "codex"
	SkillSourceCodexSystem = "codex-system"
	SkillSourceOpenCode    = "opencode"

	SkillCandidateStatusPending  = "pending"
	SkillCandidateStatusImported = "imported"
	SkillCandidateStatusIgnored  = "ignored"
	SkillCandidateStatusConflict = "conflict"

	SkillActionActivate   = "activate"
	SkillActionSetEnabled = "set_enabled"
	SkillActionEdit       = "edit"
	SkillActionDelete     = "delete"
)

type SkillEntry struct {
	ID             string              `json:"id"`
	Name           string              `json:"name"`
	Description    string              `json:"description"`
	Scope          string              `json:"scope"`
	Source         string              `json:"source"`
	RootPath       string              `json:"rootPath"`
	SkillPath      string              `json:"skillPath"`
	ContentHash    string              `json:"contentHash"`
	Enabled        bool                `json:"enabled"`
	Metadata       map[string]string   `json:"metadata,omitempty"`
	Actions        []string            `json:"actions,omitempty"`
	SelectionGroup *ToolSelectionGroup `json:"selectionGroup,omitempty"`
	TimeCreated    string              `json:"timeCreated"`
	TimeUpdated    string              `json:"timeUpdated"`
}

type SkillSource struct {
	ID          string `json:"id"`
	SkillID     string `json:"skillId"`
	Source      string `json:"source"`
	Scope       string `json:"scope"`
	RootPath    string `json:"rootPath"`
	SkillPath   string `json:"skillPath"`
	ContentHash string `json:"contentHash"`
	LastSeenAt  string `json:"lastSeenAt"`
}

type SkillImportCandidate struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Scope       string `json:"scope"`
	Source      string `json:"source"`
	RootPath    string `json:"rootPath"`
	SkillPath   string `json:"skillPath"`
	ContentHash string `json:"contentHash"`
	Status      string `json:"status"`
	ConflictID  string `json:"conflictId,omitempty"`
	Error       string `json:"error,omitempty"`
	LastSeenAt  string `json:"lastSeenAt"`
}

type SkillScanInput struct {
	WorkspaceRoot string `json:"workspaceRoot,omitempty"`
}

type SkillScanResult struct {
	Entries    []SkillEntry           `json:"entries,omitempty"`
	Candidates []SkillImportCandidate `json:"candidates,omitempty"`
	Scanned    int                    `json:"scanned"`
	Imported   int                    `json:"imported"`
	Conflicts  int                    `json:"conflicts"`
	Errors     []string               `json:"errors,omitempty"`
}

type SkillListInput struct {
	WorkspaceRoot     string `json:"workspaceRoot,omitempty"`
	IncludeCandidates bool   `json:"includeCandidates,omitempty"`
	IncludeDisabled   bool   `json:"includeDisabled,omitempty"`
	IncludeIgnored    bool   `json:"includeIgnored,omitempty"`
}

type SkillListResult struct {
	Entries    []SkillEntry           `json:"entries"`
	Candidates []SkillImportCandidate `json:"candidates,omitempty"`
}

type SkillImportInput struct {
	CandidateID string `json:"candidateId"`
	TargetScope string `json:"targetScope,omitempty"`
}

type SkillIgnoreCandidatesInput struct {
	Name string `json:"name"`
}

type SkillEnabledInput struct {
	SkillID string `json:"skillId"`
	Enabled bool   `json:"enabled"`
}

type SkillEditResult struct {
	Skill   SkillEntry `json:"skill"`
	Content string     `json:"content"`
}

type SkillUpdateInput struct {
	SkillID             string `json:"skillId"`
	Description         string `json:"description"`
	Content             string `json:"content"`
	ExpectedContentHash string `json:"expectedContentHash"`
}

type LoadSkillIntoSessionInput struct {
	SessionID string `json:"sessionId"`
	SkillID   string `json:"skillId,omitempty"`
	Name      string `json:"name,omitempty"`
	Scope     string `json:"scope,omitempty"`
	Reason    string `json:"reason,omitempty"`
	Reload    bool   `json:"reload,omitempty"`
}

type SessionActiveSkillsInput struct {
	SessionID string   `json:"sessionId"`
	SkillIDs  []string `json:"skillIds,omitempty"`
}

type SessionActiveSkillsResult struct {
	SessionID string       `json:"sessionId"`
	SkillIDs  []string     `json:"skillIds"`
	Skills    []SkillEntry `json:"skills,omitempty"`
}
