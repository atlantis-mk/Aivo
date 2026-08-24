package domain

type SessionSummary struct {
	ID                  string   `json:"id"`
	SessionID           string   `json:"sessionId"`
	FromEventID         string   `json:"fromEventId,omitempty"`
	ToEventID           string   `json:"toEventId,omitempty"`
	Summary             string   `json:"summary"`
	Facts               []string `json:"facts,omitempty"`
	Decisions           []string `json:"decisions,omitempty"`
	OpenTasks           []string `json:"openTasks,omitempty"`
	ChangedFiles        []string `json:"changedFiles,omitempty"`
	NextSuggestedAction string   `json:"nextSuggestedAction,omitempty"`
	TimeCreated         string   `json:"timeCreated"`
}

type SessionCheckpoint struct {
	ID                  string   `json:"id"`
	SessionID           string   `json:"sessionId"`
	Branch              string   `json:"branch,omitempty"`
	CommitSHA           string   `json:"commitSha,omitempty"`
	ChangedFiles        []string `json:"changedFiles,omitempty"`
	DiffSummary         string   `json:"diffSummary,omitempty"`
	ConversationSummary string   `json:"conversationSummary,omitempty"`
	OpenTodos           []string `json:"openTodos,omitempty"`
	KnownIssues         []string `json:"knownIssues,omitempty"`
	NextSuggestedAction string   `json:"nextSuggestedAction,omitempty"`
	TimeCreated         string   `json:"timeCreated"`
}

type CodingContext struct {
	ID             string   `json:"id"`
	SessionID      string   `json:"sessionId"`
	ProjectPath    string   `json:"projectPath"`
	GitBranch      string   `json:"gitBranch,omitempty"`
	CommitSHA      string   `json:"commitSha,omitempty"`
	RepoURL        string   `json:"repoUrl,omitempty"`
	ChangedFiles   []string `json:"changedFiles,omitempty"`
	LanguageStack  []string `json:"languageStack,omitempty"`
	PackageManager string   `json:"packageManager,omitempty"`
	CWD            string   `json:"cwd,omitempty"`
	Permissions    []string `json:"permissions,omitempty"`
	LastCommand    string   `json:"lastCommand,omitempty"`
	TimeCreated    string   `json:"timeCreated"`
	TimeUpdated    string   `json:"timeUpdated"`
}

type ResumeRecap struct {
	SessionID           string             `json:"sessionId"`
	Title               string             `json:"title"`
	Goal                string             `json:"goal,omitempty"`
	LatestSummary       *SessionSummary    `json:"latestSummary,omitempty"`
	ProjectPath         string             `json:"projectPath,omitempty"`
	Branch              string             `json:"branch,omitempty"`
	ChangedFiles        []string           `json:"changedFiles,omitempty"`
	OpenTodos           []string           `json:"openTodos,omitempty"`
	LastCommand         string             `json:"lastCommand,omitempty"`
	NextSuggestedAction string             `json:"nextSuggestedAction,omitempty"`
	UpdatedTime         string             `json:"updatedTime"`
	LatestCheckpoint    *SessionCheckpoint `json:"latestCheckpoint,omitempty"`
	RecentEvents        []SessionEvent     `json:"recentEvents,omitempty"`
}

type ContextSection struct {
	Name      string `json:"name"`
	Content   string `json:"content"`
	Truncated bool   `json:"truncated,omitempty"`
}

type BuildSessionContextResult struct {
	SessionID         string           `json:"sessionId"`
	Sections          []ContextSection `json:"sections"`
	EstimatedTokens   int              `json:"estimatedTokens"`
	CharacterBudget   int              `json:"characterBudget,omitempty"`
	TruncatedSections []string         `json:"truncatedSections,omitempty"`
}
