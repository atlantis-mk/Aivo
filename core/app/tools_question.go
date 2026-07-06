package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"aivo/core/domain"
)

func questionToolSpec(name string, description string) domain.ToolSpec {
	if strings.TrimSpace(description) == "" {
		description = "Ask the user one or more clarifying questions during execution and wait for their answer. Use this only when the answer materially changes what you should do. If you recommend a choice, put it first and suffix the label with \"(Recommended)\". Do not include an \"Other\" option because users can always type a custom answer."
	}
	return domain.ToolSpec{
		Name:        name,
		Description: description,
		Kind:        domain.ToolKindJSON,
		InputSchema: map[string]any{
			"type":                 "object",
			"additionalProperties": false,
			"properties": map[string]any{
				"questions": map[string]any{
					"type":        "array",
					"description": "Questions to ask.",
					"minItems":    1,
					"items": map[string]any{
						"type":                 "object",
						"additionalProperties": false,
						"properties": map[string]any{
							"id":       map[string]any{"type": "string", "description": "Optional stable question id."},
							"header":   map[string]any{"type": "string", "description": "Very short label, max 30 characters."},
							"question": map[string]any{"type": "string", "description": "Complete question to show the user."},
							"options": map[string]any{
								"type":        "array",
								"description": "Available choices.",
								"items": map[string]any{
									"type":                 "object",
									"additionalProperties": false,
									"properties": map[string]any{
										"label":       map[string]any{"type": "string", "description": "Display text, concise."},
										"description": map[string]any{"type": "string", "description": "Explanation of this choice."},
									},
									"required": []string{"label"},
								},
							},
							"multiple": map[string]any{"type": "boolean", "description": "Allow selecting multiple choices."},
						},
						"required": []string{"question", "options"},
					},
				},
			},
			"required": []string{"questions"},
		},
		Capability: "user.question",
		RiskLevel:  "low",
		Category:   "interaction",
		Toolsets:   []string{"safe", "personal", "coding"},
	}
}

func (s *Service) askUserTool(ctx context.Context, args json.RawMessage, execCtx domain.ToolExecutionContext) domain.ToolResult {
	return s.questionToolNamed(ctx, args, execCtx, "ask_user")
}

func (s *Service) questionToolNamed(ctx context.Context, args json.RawMessage, execCtx domain.ToolExecutionContext, toolName string) domain.ToolResult {
	var input struct {
		Questions []domain.QuestionPrompt `json:"questions"`
	}
	if err := json.Unmarshal(args, &input); err != nil {
		return errorToolResult(toolName, err)
	}
	questions, err := normalizeQuestionPrompts(input.Questions)
	if err != nil {
		return errorToolResult(toolName, err)
	}
	request, err := s.AskQuestion(ctx, domain.QuestionRequest{
		SessionID:  execCtx.SessionID,
		TurnID:     execCtx.TurnID,
		ToolCallID: execCtx.ToolCallID,
		ToolName:   toolName,
		Questions:  questions,
		Arguments:  rawMessageToAnyMap(args),
	})
	if err != nil {
		return errorToolResult(toolName, err)
	}
	content := formatQuestionAnswers(questions, request.Answers)
	return domain.ToolResult{
		Name:         toolName,
		OK:           true,
		Content:      content,
		ModelContent: content,
		Structured: map[string]any{
			"requestId": request.ID,
			"answers":   request.Answers,
		},
	}
}

func (s *Service) AskQuestion(ctx context.Context, request domain.QuestionRequest) (domain.QuestionRequest, error) {
	if s == nil || s.store == nil {
		return domain.QuestionRequest{}, errors.New("store is not configured")
	}
	questions, err := normalizeQuestionPrompts(request.Questions)
	if err != nil {
		return domain.QuestionRequest{}, err
	}
	request.Questions = questions
	if request.ToolName == "" {
		request.ToolName = "ask_user"
	}
	created, err := s.store.CreateQuestionRequest(ctx, request)
	if err != nil {
		return domain.QuestionRequest{}, err
	}
	watched := s.questionNotifier.watch(created.ID)
	defer s.questionNotifier.forget(created.ID, watched)
	if created.SessionID != "" && created.ToolCallID != "" {
		_, _ = s.SaveToolCall(context.Background(), domain.CreateToolCallRequest{
			ID:            created.ToolCallID,
			SessionID:     created.SessionID,
			TurnID:        created.TurnID,
			Name:          created.ToolName,
			Arguments:     created.Arguments,
			Status:        domain.ToolCallStatusPending,
			ResultSummary: "Waiting for user answer",
			Result: map[string]any{
				"ok":                false,
				"call_id":           created.ToolCallID,
				"name":              created.ToolName,
				"pendingQuestionId": created.ID,
			},
		})
	}
	if s.onQuestionRequested != nil {
		s.onQuestionRequested(created)
	}
	if s.onSessionUpdated != nil && created.SessionID != "" {
		s.onSessionUpdated(created.SessionID, nil)
	}
	for {
		select {
		case <-ctx.Done():
			return domain.QuestionRequest{}, ctx.Err()
		case <-watched:
			current, err := s.store.GetQuestionRequest(ctx, created.ID)
			if err != nil {
				return domain.QuestionRequest{}, err
			}
			switch current.Status {
			case domain.QuestionRequestStatusAnswered:
				return current, nil
			case domain.QuestionRequestStatusRejected:
				return domain.QuestionRequest{}, errors.New(firstNonEmpty(current.Reason, "question rejected by user"))
			}
		}
	}
}

func (s *Service) ListQuestionRequests(ctx context.Context, sessionID string, status string) ([]domain.QuestionRequest, error) {
	if s == nil || s.store == nil {
		return nil, errors.New("store is not configured")
	}
	return s.store.ListQuestionRequests(ctx, sessionID, status)
}

func (s *Service) ReplyQuestionRequest(ctx context.Context, input domain.ReplyQuestionRequestInput) (domain.QuestionRequest, error) {
	if strings.TrimSpace(input.RequestID) == "" {
		return domain.QuestionRequest{}, errors.New("requestId is required")
	}
	current, err := s.store.GetQuestionRequest(ctx, input.RequestID)
	if err != nil {
		return domain.QuestionRequest{}, err
	}
	if current.Status != domain.QuestionRequestStatusPending {
		return current, nil
	}
	answers := normalizeQuestionAnswers(input.Answers, len(current.Questions))
	request, err := s.store.UpdateQuestionRequest(ctx, input.RequestID, domain.QuestionRequestStatusAnswered, answers, "")
	if err != nil {
		return domain.QuestionRequest{}, err
	}
	s.resolveQuestionRequest(request)
	return request, nil
}

func (s *Service) RejectQuestionRequest(ctx context.Context, input domain.RejectQuestionRequestInput) (domain.QuestionRequest, error) {
	if strings.TrimSpace(input.RequestID) == "" {
		return domain.QuestionRequest{}, errors.New("requestId is required")
	}
	current, err := s.store.GetQuestionRequest(ctx, input.RequestID)
	if err != nil {
		return domain.QuestionRequest{}, err
	}
	if current.Status != domain.QuestionRequestStatusPending {
		return current, nil
	}
	request, err := s.store.UpdateQuestionRequest(ctx, input.RequestID, domain.QuestionRequestStatusRejected, nil, firstNonEmpty(input.Reason, "Dismissed by user"))
	if err != nil {
		return domain.QuestionRequest{}, err
	}
	s.resolveQuestionRequest(request)
	return request, nil
}

func (s *Service) resolveQuestionRequest(request domain.QuestionRequest) {
	if s.questionNotifier != nil {
		s.questionNotifier.resolve(request.ID)
	}
	if s.onQuestionResolved != nil {
		s.onQuestionResolved(request)
	}
	if s.onSessionUpdated != nil && request.SessionID != "" {
		s.onSessionUpdated(request.SessionID, nil)
	}
}

func normalizeQuestionPrompts(input []domain.QuestionPrompt) ([]domain.QuestionPrompt, error) {
	if len(input) == 0 {
		return nil, errors.New("questions must contain at least one question")
	}
	out := make([]domain.QuestionPrompt, 0, len(input))
	for index, question := range input {
		text := strings.TrimSpace(question.Question)
		if text == "" {
			return nil, fmt.Errorf("questions[%d].question is required", index)
		}
		options := make([]domain.QuestionOption, 0, len(question.Options))
		for _, option := range question.Options {
			label := strings.TrimSpace(option.Label)
			if label == "" {
				continue
			}
			options = append(options, domain.QuestionOption{Label: label, Description: strings.TrimSpace(option.Description)})
		}
		if len(options) == 0 {
			return nil, fmt.Errorf("questions[%d].options must contain at least one option", index)
		}
		if strings.TrimSpace(question.ID) == "" {
			question.ID = uuid.NewString()
		}
		question.Header = strings.TrimSpace(question.Header)
		question.Question = text
		question.Options = options
		out = append(out, question)
	}
	return out, nil
}

func normalizeQuestionAnswers(input [][]string, count int) [][]string {
	out := make([][]string, count)
	for i := 0; i < count && i < len(input); i++ {
		seen := map[string]struct{}{}
		for _, raw := range input[i] {
			answer := strings.TrimSpace(raw)
			if answer == "" {
				continue
			}
			if _, ok := seen[answer]; ok {
				continue
			}
			seen[answer] = struct{}{}
			out[i] = append(out[i], answer)
		}
	}
	return out
}

func formatQuestionAnswers(questions []domain.QuestionPrompt, answers [][]string) string {
	parts := make([]string, 0, len(questions))
	for index, question := range questions {
		answer := "Unanswered"
		if index < len(answers) && len(answers[index]) > 0 {
			answer = strings.Join(answers[index], ", ")
		}
		parts = append(parts, fmt.Sprintf("%q=%q", question.Question, answer))
	}
	return "User has answered your questions: " + strings.Join(parts, ", ") + ". You can now continue with the user's answers in mind."
}

func rawMessageToAnyMap(raw json.RawMessage) map[string]any {
	var out map[string]any
	_ = json.Unmarshal(raw, &out)
	return out
}
