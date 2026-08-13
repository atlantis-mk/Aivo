package app

import (
	"encoding/base64"
	"fmt"
	"io"
	"strings"

	"aivo/core/domain"
)

const maxSessionMessageAttachmentBytes = 50 * 1024 * 1024

func validateSessionMessageAttachments(attachments []domain.MessageAttachment) error {
	for index, attachment := range attachments {
		name := strings.TrimSpace(attachment.Name)
		if name == "" {
			name = fmt.Sprintf("attachment %d", index+1)
		}
		kind := strings.TrimSpace(attachment.Kind)
		if kind != "" && kind != "file" && kind != "image" {
			return fmt.Errorf("%s has unsupported attachment kind %q", name, kind)
		}
		if attachment.Size < 0 {
			return fmt.Errorf("%s has an invalid attachment size", name)
		}
		if attachment.Size > maxSessionMessageAttachmentBytes {
			return fmt.Errorf("%s exceeds the 50 MB attachment limit", name)
		}
		data := strings.TrimSpace(attachment.Data)
		textPresent := strings.TrimSpace(attachment.Text) != ""
		if data != "" && textPresent {
			return fmt.Errorf("%s cannot contain both binary data and text", name)
		}
		if data == "" {
			if !textPresent {
				return fmt.Errorf("%s has no readable attachment content", name)
			}
			if len([]byte(attachment.Text)) > maxSessionMessageAttachmentBytes {
				return fmt.Errorf("%s exceeds the 50 MB attachment limit", name)
			}
			continue
		}
		encoded := data
		if strings.HasPrefix(encoded, "data:") {
			comma := strings.IndexByte(encoded, ',')
			if comma < 0 || !strings.Contains(strings.ToLower(encoded[:comma]), ";base64") {
				return fmt.Errorf("%s has invalid attachment data", name)
			}
			encoded = encoded[comma+1:]
		}
		decoder := base64.NewDecoder(base64.StdEncoding, strings.NewReader(encoded))
		decodedBytes, err := io.Copy(io.Discard, io.LimitReader(decoder, maxSessionMessageAttachmentBytes+1))
		if err != nil {
			return fmt.Errorf("%s has invalid base64 attachment data", name)
		}
		if decodedBytes > maxSessionMessageAttachmentBytes {
			return fmt.Errorf("%s exceeds the 50 MB attachment limit", name)
		}
	}
	return nil
}

func normalizeSessionMessageAttachments(attachments []domain.MessageAttachment) []domain.MessageAttachment {
	out := make([]domain.MessageAttachment, 0, len(attachments))
	for _, attachment := range attachments {
		data := strings.TrimSpace(attachment.Data)
		text := attachment.Text
		if data == "" && strings.TrimSpace(text) == "" {
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
