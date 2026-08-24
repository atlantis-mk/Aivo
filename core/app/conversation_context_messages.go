package app

import (
	"strings"

	"aivo/core/domain"
)

func chatMessagesFromEvents(events []domain.SessionEvent, turns []domain.Turn) []domain.ChatMessage {
	cancelledTurnIDs := make(map[string]struct{})
	cancelledEventIDs := make(map[string]struct{})
	for _, turn := range turns {
		if turn.Status != domain.TurnStatusCancelled {
			continue
		}
		if turn.ID != "" {
			cancelledTurnIDs[turn.ID] = struct{}{}
		}
		if turn.UserEventID != "" {
			cancelledEventIDs[turn.UserEventID] = struct{}{}
		}
	}
	messages := make([]domain.ChatMessage, 0, len(events))
	for _, event := range events {
		if event.Type != domain.EventTypeUserMessage && event.Type != domain.EventTypeAssistantMessage {
			continue
		}
		if _, cancelled := cancelledEventIDs[event.ID]; cancelled {
			continue
		}
		if _, cancelled := cancelledTurnIDs[event.TurnID]; cancelled {
			continue
		}
		role := strings.TrimSpace(event.Role)
		if role == "" {
			if event.Type == domain.EventTypeUserMessage {
				role = domain.EventRoleUser
			} else {
				role = domain.EventRoleAssistant
			}
		}
		messages = append(messages, domain.ChatMessage{Role: role, Text: event.Content})
	}
	return messages
}

func selectRecentChatTail(messages []domain.ChatMessage, limit int, charBudget int) []domain.ChatMessage {
	if len(messages) == 0 {
		return nil
	}
	if limit <= 0 || limit > len(messages) {
		limit = len(messages)
	}
	if charBudget <= 0 {
		return messages[len(messages)-limit:]
	}
	start := len(messages)
	used := 0
	for start > 0 && len(messages)-start < limit {
		nextLen := chatMessageCharLen(messages[start-1])
		if used > 0 && used+nextLen > charBudget {
			break
		}
		used += nextLen
		start--
	}
	if start == len(messages) {
		start = len(messages) - 1
	}
	return messages[start:]
}

func chatMessageCharLen(message domain.ChatMessage) int {
	total := len(message.Role) + len(message.Text)
	for _, call := range message.ToolCalls {
		total += len(call.ID) + len(call.Name) + len(call.Arguments)
	}
	total += len(message.ToolCallID) + len(message.Name)
	return total
}

func chatMessagesCharLen(messages []domain.ChatMessage) int {
	total := 0
	for _, message := range messages {
		total += chatMessageCharLen(message)
	}
	return total
}
