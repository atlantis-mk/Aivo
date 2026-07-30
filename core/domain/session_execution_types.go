package domain

type SessionExecutionState struct {
	ID              string         `json:"id"`
	SessionID       string         `json:"sessionId"`
	TurnID          string         `json:"turnId,omitempty"`
	Status          string         `json:"status"`
	Reason          string         `json:"reason,omitempty"`
	LastEventID     string         `json:"lastEventId,omitempty"`
	PendingInputIDs []string       `json:"pendingInputIds,omitempty"`
	Metadata        map[string]any `json:"metadata,omitempty"`
	TimeCreated     string         `json:"timeCreated"`
	TimeUpdated     string         `json:"timeUpdated"`
}

type PendingSessionInput struct {
	ID             string `json:"id"`
	SessionID      string `json:"sessionId"`
	TurnID         string `json:"turnId,omitempty"`
	Text           string `json:"text"`
	Delivery       string `json:"delivery"`
	Status         string `json:"status"`
	PromotedTurnID string `json:"promotedTurnId,omitempty"`
	TimeCreated    string `json:"timeCreated"`
	TimeUpdated    string `json:"timeUpdated"`
}

type InterruptSessionExecutionInput struct {
	SessionID string `json:"sessionId"`
	Reason    string `json:"reason,omitempty"`
}

type ResumeSessionExecutionInput struct {
	SessionID string `json:"sessionId"`
}

type CompactSessionContextInput struct {
	SessionID       string `json:"sessionId"`
	CharacterBudget int    `json:"characterBudget,omitempty"`
	Automatic       bool   `json:"automatic,omitempty"`
}

type CompactSessionContextResult struct {
	State            SessionExecutionState     `json:"state"`
	Summary          SessionSummary            `json:"summary"`
	Context          BuildSessionContextResult `json:"context"`
	CompactedEventID string                    `json:"compactedEventId,omitempty"`
}

type ListSessionEventsAfterCursorInput struct {
	SessionID        string `json:"sessionId"`
	Cursor           string `json:"cursor,omitempty"`
	IncludeNonNormal bool   `json:"includeNonNormal,omitempty"`
	Limit            int    `json:"limit,omitempty"`
}

type ListSessionEventsAfterCursorResult struct {
	Events     []SessionEvent `json:"events"`
	NextCursor string         `json:"nextCursor"`
}
