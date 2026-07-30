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
