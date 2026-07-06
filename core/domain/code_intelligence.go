package domain

import "context"

const (
	CodeIntelligenceStatusReady       = "ready"
	CodeIntelligenceStatusUnavailable = "unavailable"
	CodeIntelligenceStatusFallback    = "fallback"

	DiagnosticSeverityError       = "error"
	DiagnosticSeverityWarning     = "warning"
	DiagnosticSeverityInformation = "information"
	DiagnosticSeverityHint        = "hint"
)

type SourcePosition struct {
	Line      int `json:"line"`
	Character int `json:"character,omitempty"`
}

type SourceRange struct {
	Start SourcePosition `json:"start"`
	End   SourcePosition `json:"end"`
}

type CodeIntelligenceStatus struct {
	WorkspaceRoot string `json:"workspaceRoot,omitempty"`
	Language      string `json:"language,omitempty"`
	Status        string `json:"status"`
	Source        string `json:"source,omitempty"`
	Message       string `json:"message,omitempty"`
	TimeUpdated   string `json:"timeUpdated,omitempty"`
}

type CodeDiagnostic struct {
	Path     string      `json:"path"`
	Range    SourceRange `json:"range"`
	Severity string      `json:"severity"`
	Message  string      `json:"message"`
	Source   string      `json:"source,omitempty"`
	Code     string      `json:"code,omitempty"`
}

type CodeSymbol struct {
	Name      string      `json:"name"`
	Kind      string      `json:"kind"`
	Path      string      `json:"path"`
	Range     SourceRange `json:"range"`
	Language  string      `json:"language,omitempty"`
	Signature string      `json:"signature,omitempty"`
	Source    string      `json:"source,omitempty"`
}

type CodeLocation struct {
	Path     string      `json:"path"`
	Range    SourceRange `json:"range"`
	Language string      `json:"language,omitempty"`
	Preview  string      `json:"preview,omitempty"`
	Source   string      `json:"source,omitempty"`
}

type CodeIntelligenceService interface {
	Status(ctx context.Context, workspaceRoot string) (CodeIntelligenceStatus, error)
	Diagnostics(ctx context.Context, workspaceRoot string, path string) ([]CodeDiagnostic, CodeIntelligenceStatus, error)
	Symbols(ctx context.Context, workspaceRoot string, query string, path string, kind string, limit int) ([]CodeSymbol, CodeIntelligenceStatus, error)
	Definition(ctx context.Context, workspaceRoot string, path string, position SourcePosition, limit int) ([]CodeLocation, CodeIntelligenceStatus, error)
	References(ctx context.Context, workspaceRoot string, path string, position SourcePosition, limit int) ([]CodeLocation, CodeIntelligenceStatus, error)
}
