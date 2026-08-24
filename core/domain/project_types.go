package domain

type AssistantProject struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Description   string `json:"description,omitempty"`
	RootPath      string `json:"rootPath"`
	GitBranch     string `json:"gitBranch,omitempty"`
	GitDirty      bool   `json:"gitDirty,omitempty"`
	GitAvailable  bool   `json:"gitAvailable"`
	SidebarHidden bool   `json:"sidebarHidden,omitempty"`
	TimeOpened    string `json:"timeOpened"`
	TimeUpdated   string `json:"timeUpdated"`
}

const (
	ProjectRegistrationCreated  = "created"
	ProjectRegistrationExisting = "existing"
	ProjectRegistrationRestored = "restored"
)

type ProjectQueryInput struct {
	ProjectID string `json:"projectId,omitempty"`
	Query     string `json:"query,omitempty"`
	Limit     int    `json:"limit,omitempty"`
	Cursor    string `json:"cursor,omitempty"`
}

type ProjectQueryResult struct {
	Projects       []AssistantProject `json:"projects"`
	NextCursor     string             `json:"nextCursor,omitempty"`
	CurrentProject *AssistantProject  `json:"currentProject,omitempty"`
}

type ProjectRegistrationResult struct {
	Project AssistantProject `json:"project"`
	Status  string           `json:"status"`
}

type SessionProjectBindingResult struct {
	Session  Session `json:"session"`
	Changed  bool    `json:"changed"`
	Conflict bool    `json:"-"`
}
