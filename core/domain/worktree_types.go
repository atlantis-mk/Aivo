package domain

const (
	WorktreeStatusReady   = "ready"
	WorktreeStatusMissing = "missing"
	WorktreeStatusRemoved = "removed"
	WorktreeStatusError   = "error"
)

type GitWorktree struct {
	ID             string   `json:"id"`
	RepositoryRoot string   `json:"repositoryRoot"`
	Path           string   `json:"path"`
	Branch         string   `json:"branch,omitempty"`
	BaseRef        string   `json:"baseRef,omitempty"`
	Head           string   `json:"head,omitempty"`
	Status         string   `json:"status"`
	Managed        bool     `json:"managed"`
	OwnsBranch     bool     `json:"ownsBranch,omitempty"`
	Detached       bool     `json:"detached,omitempty"`
	Dirty          bool     `json:"dirty,omitempty"`
	ActiveSessions []string `json:"activeSessions,omitempty"`
	Error          string   `json:"error,omitempty"`
	TimeCreated    string   `json:"timeCreated"`
	TimeUpdated    string   `json:"timeUpdated"`
}

type CreateGitWorktreeInput struct {
	RepositoryPath   string `json:"repositoryPath"`
	Path             string `json:"path,omitempty"`
	ApprovedRoot     string `json:"approvedRoot,omitempty"`
	Name             string `json:"name,omitempty"`
	Branch           string `json:"branch,omitempty"`
	BaseRef          string `json:"baseRef,omitempty"`
	Detached         bool   `json:"detached,omitempty"`
	StartupCommand   string `json:"startupCommand,omitempty"`
	StartupConfirmed bool   `json:"startupConfirmed,omitempty"`
	SessionID        string `json:"sessionId,omitempty"`
}

type ListGitWorktreesInput struct {
	RepositoryPath string `json:"repositoryPath,omitempty"`
	IncludeRemoved bool   `json:"includeRemoved,omitempty"`
}

type ResetGitWorktreeInput struct {
	WorktreeID string `json:"worktreeId"`
	TargetRef  string `json:"targetRef,omitempty"`
	Clean      bool   `json:"clean,omitempty"`
	Confirmed  bool   `json:"confirmed"`
}

type RemoveGitWorktreeInput struct {
	WorktreeID   string `json:"worktreeId"`
	Force        bool   `json:"force,omitempty"`
	DeleteBranch *bool  `json:"deleteBranch,omitempty"`
	Confirmed    bool   `json:"confirmed"`
}

type BindSessionGitWorktreeInput struct {
	SessionID  string `json:"sessionId"`
	WorktreeID string `json:"worktreeId"`
}
