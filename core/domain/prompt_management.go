package domain

const (
	PromptCategoryAgent          = "agent"
	PromptCategoryProtocol       = "protocol"
	PromptCategoryAuxiliary      = "auxiliary"
	PromptCategoryTask           = "task"
	PromptCategoryDynamicContext = "dynamic_context"
	PromptCategoryQuickPrompt    = "quick_prompt"
)

type PromptDiagnostic struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Line    int    `json:"line,omitempty"`
	Column  int    `json:"column,omitempty"`
}

type PromptDocument struct {
	ID                string             `json:"id"`
	Category          string             `json:"category"`
	Title             string             `json:"title"`
	Body              string             `json:"body"`
	Enabled           bool               `json:"enabled"`
	Origin            string             `json:"origin"`
	Required          bool               `json:"required"`
	Disableable       bool               `json:"disableable"`
	Deletable         bool               `json:"deletable"`
	Variables         []string           `json:"variables,omitempty"`
	RequiredVariables []string           `json:"requiredVariables,omitempty"`
	MaxLength         int                `json:"maxLength"`
	WorkingRevision   string             `json:"workingRevision,omitempty"`
	ActiveRevision    string             `json:"activeRevision,omitempty"`
	Status            string             `json:"status"`
	Fallback          bool               `json:"fallback,omitempty"`
	Diagnostics       []PromptDiagnostic `json:"diagnostics,omitempty"`
}

type PromptDocumentInput struct {
	ID       string `json:"id"`
	Category string `json:"category"`
	Title    string `json:"title"`
	Body     string `json:"body"`
	Enabled  bool   `json:"enabled"`
}

type PromptDocumentIDInput struct {
	ID string `json:"id"`
}

type PromptEnabledInput struct {
	ID      string `json:"id"`
	Enabled bool   `json:"enabled"`
}

type CreateAgentPromptInput struct {
	ID              string   `json:"id"`
	Title           string   `json:"title"`
	Body            string   `json:"body"`
	Description     string   `json:"description,omitempty"`
	PermissionScope string   `json:"permissionScope,omitempty"`
	Mode            string   `json:"mode,omitempty"`
	Subagents       []string `json:"subagents,omitempty"`
}

type CreateQuickPromptInput struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Body  string `json:"body"`
}

type PromptValidationResult struct {
	Valid       bool               `json:"valid"`
	Revision    string             `json:"revision,omitempty"`
	Diagnostics []PromptDiagnostic `json:"diagnostics,omitempty"`
}

type PromptToolDescription struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Category    string `json:"category,omitempty"`
	Source      string `json:"source,omitempty"`
}
