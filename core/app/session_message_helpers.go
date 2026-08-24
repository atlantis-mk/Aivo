package app

import (
	"encoding/base64"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"unicode/utf8"

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
			if !utf8.ValidString(attachment.Text) {
				return fmt.Errorf("%s is not valid UTF-8 text", name)
			}
			if kind == "image" {
				return fmt.Errorf("%s has text content with image attachment kind", name)
			}
			mimeType := effectiveTextAttachmentMIME(attachment.MIMEType, attachment.Name)
			if !isTextAttachmentMIME(mimeType) {
				return fmt.Errorf("%s has unsupported text attachment MIME type %q", name, attachment.MIMEType)
			}
			continue
		}
		mimeType := normalizeAttachmentMIME(attachment.MIMEType)
		if !isSupportedBinaryAttachmentMIME(mimeType) {
			return fmt.Errorf("%s has unsupported binary attachment MIME type %q", name, attachment.MIMEType)
		}
		if kind == "image" && !isImageAttachmentMIME(mimeType) {
			return fmt.Errorf("%s has image attachment kind with non-image MIME type %q", name, attachment.MIMEType)
		}
		if kind == "file" && isImageAttachmentMIME(mimeType) {
			return fmt.Errorf("%s has file attachment kind with image MIME type %q", name, attachment.MIMEType)
		}
		encoded, embeddedMIME, err := attachmentBase64Payload(data)
		if err != nil {
			return fmt.Errorf("%s has invalid attachment data", name)
		}
		if embeddedMIME != "" && embeddedMIME != mimeType {
			return fmt.Errorf("%s has attachment data MIME type %q that does not match %q", name, embeddedMIME, mimeType)
		}
		decoder := base64.NewDecoder(base64.StdEncoding, strings.NewReader(encoded))
		header := make([]byte, 1024)
		headerBytes, readErr := io.ReadFull(decoder, header)
		if readErr != nil && readErr != io.EOF && readErr != io.ErrUnexpectedEOF {
			return fmt.Errorf("%s has invalid base64 attachment data", name)
		}
		decodedBytes, err := io.Copy(io.Discard, io.LimitReader(decoder, maxSessionMessageAttachmentBytes+1-int64(headerBytes)))
		if err != nil {
			return fmt.Errorf("%s has invalid base64 attachment data", name)
		}
		decodedBytes += int64(headerBytes)
		if decodedBytes > maxSessionMessageAttachmentBytes {
			return fmt.Errorf("%s exceeds the 50 MB attachment limit", name)
		}
		if !attachmentHeaderMatchesMIME(mimeType, header[:headerBytes]) {
			return fmt.Errorf("%s content does not match declared attachment MIME type %q", name, mimeType)
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
		mimeType := normalizeAttachmentMIME(attachment.MIMEType)
		if data == "" {
			mimeType = effectiveTextAttachmentMIME(mimeType, name)
		} else if payload, _, err := attachmentBase64Payload(data); err == nil {
			data = payload
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

func normalizeAttachmentMIME(mimeType string) string {
	mimeType = strings.ToLower(strings.TrimSpace(strings.SplitN(mimeType, ";", 2)[0]))
	if mimeType == "image/jpg" {
		return "image/jpeg"
	}
	return mimeType
}

func effectiveTextAttachmentMIME(mimeType string, name string) string {
	mimeType = normalizeAttachmentMIME(mimeType)
	if mimeType == "" || mimeType == "application/octet-stream" {
		if isTextAttachmentName(name) || mimeType == "" {
			return "text/plain"
		}
	}
	return mimeType
}

func isTextAttachmentMIME(mimeType string) bool {
	mimeType = normalizeAttachmentMIME(mimeType)
	if strings.HasPrefix(mimeType, "text/") {
		return true
	}
	switch mimeType {
	case "application/json", "application/ld+json", "application/toml", "application/xml", "application/x-httpd-php", "application/x-sh", "application/x-yaml", "application/yaml":
		return true
	default:
		return false
	}
}

func isTextAttachmentName(name string) bool {
	switch strings.ToLower(filepath.Ext(strings.TrimSpace(name))) {
	case ".c", ".cc", ".conf", ".cpp", ".cs", ".css", ".env", ".go", ".h", ".hpp", ".htm", ".html", ".ini", ".java", ".js", ".jsx", ".json", ".jsonl", ".kt", ".kts", ".log", ".lua", ".md", ".mjs", ".php", ".pl", ".properties", ".py", ".rb", ".rs", ".sh", ".sql", ".swift", ".toml", ".ts", ".tsx", ".txt", ".vue", ".xml", ".yaml", ".yml":
		return true
	default:
		return false
	}
}

func isSupportedBinaryAttachmentMIME(mimeType string) bool {
	switch normalizeAttachmentMIME(mimeType) {
	case "image/png", "image/jpeg", "image/gif", "image/webp", "application/pdf", "application/vnd.openxmlformats-officedocument.wordprocessingml.document", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", "application/vnd.openxmlformats-officedocument.presentationml.presentation":
		return true
	default:
		return false
	}
}

func isImageAttachmentMIME(mimeType string) bool {
	switch normalizeAttachmentMIME(mimeType) {
	case "image/png", "image/jpeg", "image/gif", "image/webp":
		return true
	default:
		return false
	}
}

func attachmentHeaderMatchesMIME(mimeType string, header []byte) bool {
	hasPrefix := func(prefix ...byte) bool {
		if len(header) < len(prefix) {
			return false
		}
		for index, value := range prefix {
			if header[index] != value {
				return false
			}
		}
		return true
	}
	switch normalizeAttachmentMIME(mimeType) {
	case "image/png":
		return hasPrefix(0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a)
	case "image/jpeg":
		return hasPrefix(0xff, 0xd8, 0xff)
	case "image/gif":
		return string(header)[:min(len(header), 6)] == "GIF87a" || string(header)[:min(len(header), 6)] == "GIF89a"
	case "image/webp":
		return len(header) >= 12 && string(header[:4]) == "RIFF" && string(header[8:12]) == "WEBP"
	case "application/pdf":
		return strings.Contains(string(header), "%PDF-")
	case "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		"application/vnd.openxmlformats-officedocument.presentationml.presentation":
		return hasPrefix('P', 'K', 0x03, 0x04) || hasPrefix('P', 'K', 0x05, 0x06) || hasPrefix('P', 'K', 0x07, 0x08)
	default:
		return false
	}
}

func attachmentBase64Payload(data string) (string, string, error) {
	data = strings.TrimSpace(data)
	if !strings.HasPrefix(strings.ToLower(data), "data:") {
		return data, "", nil
	}
	comma := strings.IndexByte(data, ',')
	if comma < 0 {
		return "", "", fmt.Errorf("missing data URL separator")
	}
	header := data[5:comma]
	segments := strings.Split(header, ";")
	if len(segments) < 2 || !strings.EqualFold(strings.TrimSpace(segments[len(segments)-1]), "base64") {
		return "", "", fmt.Errorf("attachment data URL is not base64")
	}
	mimeType := normalizeAttachmentMIME(segments[0])
	if mimeType == "" {
		return "", "", fmt.Errorf("attachment data URL has no MIME type")
	}
	return data[comma+1:], mimeType, nil
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
