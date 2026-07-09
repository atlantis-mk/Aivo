package app

import (
	"fmt"
	"strings"

	"aivo/core/domain"
)

func normalizeSessionMessageAttachments(attachments []domain.MessageAttachment) []domain.MessageAttachment {
	out := make([]domain.MessageAttachment, 0, len(attachments))
	for _, attachment := range attachments {
		data := strings.TrimSpace(attachment.Data)
		text := strings.TrimSpace(attachment.Text)
		if data == "" && text == "" {
			continue
		}
		name := strings.TrimSpace(attachment.Name)
		if name == "" {
			name = "attachment"
		}
		mimeType := strings.TrimSpace(attachment.MIMEType)
		if mimeType == "" {
			mimeType = "application/octet-stream"
		}
		kind := strings.TrimSpace(attachment.Kind)
		if kind == "" {
			if strings.HasPrefix(mimeType, "image/") {
				kind = "image"
			} else {
				kind = "file"
			}
		}
		out = append(out, domain.MessageAttachment{
			ID: attachment.ID, Name: name, MIMEType: mimeType, Kind: kind,
			Data: data, Text: text, Size: attachment.Size,
		})
	}
	return out
}

func buildSessionMessageEventText(text string, attachments []domain.MessageAttachment) string {
	text = strings.TrimSpace(text)
	if len(attachments) == 0 {
		return text
	}
	lines := make([]string, 0, len(attachments))
	for _, attachment := range attachments {
		detail := strings.TrimSpace(attachment.MIMEType)
		if detail == "" {
			detail = attachment.Kind
		}
		if attachment.Size > 0 {
			detail = fmt.Sprintf("%s, %d bytes", detail, attachment.Size)
		}
		lines = append(lines, fmt.Sprintf("- %s (%s)", attachment.Name, detail))
	}
	summary := "附件:\n" + strings.Join(lines, "\n")
	if text == "" {
		return summary
	}
	return text + "\n\n" + summary
}

func buildSessionMessageEventPayload(attachments []domain.MessageAttachment) map[string]any {
	if len(attachments) == 0 {
		return nil
	}
	items := make([]map[string]any, 0, len(attachments))
	for _, attachment := range attachments {
		item := map[string]any{
			"id":       strings.TrimSpace(attachment.ID),
			"name":     strings.TrimSpace(attachment.Name),
			"mimeType": strings.TrimSpace(attachment.MIMEType),
			"kind":     strings.TrimSpace(attachment.Kind),
			"size":     attachment.Size,
		}
		if strings.EqualFold(strings.TrimSpace(attachment.Kind), "image") {
			item["data"] = attachment.Data
		}
		items = append(items, item)
	}
	return map[string]any{
		"kind":        "user_message",
		"attachments": items,
	}
}

func attachFilesToCurrentTurn(messages []domain.ChatMessage, userEventID string, eventText string, attachments []domain.MessageAttachment) {
	if len(attachments) == 0 || len(messages) == 0 {
		return
	}
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role != domain.EventRoleUser {
			continue
		}
		if strings.TrimSpace(messages[i].Text) == strings.TrimSpace(eventText) {
			messages[i].Attachments = attachments
			return
		}
	}
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == domain.EventRoleUser {
			messages[i].Attachments = attachments
			return
		}
	}
}
