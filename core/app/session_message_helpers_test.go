package app

import (
	"strings"
	"testing"

	"aivo/core/domain"
)

func TestNormalizeSessionMessageAttachmentsPreservesTextFileContent(t *testing.T) {
	attachments := normalizeSessionMessageAttachments([]domain.MessageAttachment{{
		Name:     "indent.txt",
		MIMEType: "text/plain",
		Kind:     "file",
		Text:     "\n  first\nsecond  \n",
	}})

	if len(attachments) != 1 {
		t.Fatalf("attachments = %#v, want one attachment", attachments)
	}
	if attachments[0].Text != "\n  first\nsecond  \n" {
		t.Fatalf("text = %q, want exact file content", attachments[0].Text)
	}
}

func TestNormalizeSessionMessageAttachmentsNeverDefaultsBinaryToOctetStream(t *testing.T) {
	attachments := normalizeSessionMessageAttachments([]domain.MessageAttachment{
		{Name: "README", Kind: "file", Text: "hello"},
		{Name: "unknown.bin", Kind: "file", Data: "YQ=="},
	})

	if attachments[0].MIMEType != "text/plain" {
		t.Fatalf("text MIME type = %q, want text/plain", attachments[0].MIMEType)
	}
	if attachments[1].MIMEType != "" {
		t.Fatalf("binary MIME type = %q, want empty so validation fails closed", attachments[1].MIMEType)
	}
}

func TestNormalizeSessionMessageAttachmentsCanonicalizesDataURLPayload(t *testing.T) {
	attachments := normalizeSessionMessageAttachments([]domain.MessageAttachment{{
		Name:     "brief.pdf",
		MIMEType: "application/pdf",
		Kind:     "file",
		Data:     "data:application/pdf;base64,cGRm",
	}})

	if attachments[0].Data != "cGRm" {
		t.Fatalf("data = %q, want raw base64 payload", attachments[0].Data)
	}
}

func TestValidateSessionMessageAttachmentsRejectsUnreadablePayloads(t *testing.T) {
	tests := []struct {
		name       string
		attachment domain.MessageAttachment
		want       string
	}{
		{
			name:       "unknown kind",
			attachment: domain.MessageAttachment{Name: "pipe", Kind: "directory", Text: "content"},
			want:       "unsupported attachment kind",
		},
		{
			name:       "invalid base64",
			attachment: domain.MessageAttachment{Name: "brief.pdf", MIMEType: "application/pdf", Kind: "file", Data: "%%%"},
			want:       "invalid base64 attachment data",
		},
		{
			name:       "generic binary MIME",
			attachment: domain.MessageAttachment{Name: "archive.zip", MIMEType: "application/octet-stream", Kind: "file", Data: "UEsDBA=="},
			want:       "unsupported binary attachment MIME type",
		},
		{
			name:       "unsupported binary MIME",
			attachment: domain.MessageAttachment{Name: "archive.zip", MIMEType: "application/zip", Kind: "file", Data: "UEsDBA=="},
			want:       "unsupported binary attachment MIME type",
		},
		{
			name:       "data URL MIME mismatch",
			attachment: domain.MessageAttachment{Name: "brief.pdf", MIMEType: "application/pdf", Kind: "file", Data: "data:application/octet-stream;base64,YQ=="},
			want:       "does not match",
		},
		{
			name:       "content signature mismatch",
			attachment: domain.MessageAttachment{Name: "brief.pdf", MIMEType: "application/pdf", Kind: "file", Data: "cGRm"},
			want:       "content does not match declared attachment MIME type",
		},
		{
			name:       "image kind MIME mismatch",
			attachment: domain.MessageAttachment{Name: "brief.pdf", MIMEType: "application/pdf", Kind: "image", Data: "YQ=="},
			want:       "image attachment kind with non-image MIME type",
		},
		{
			name:       "binary MIME with text content",
			attachment: domain.MessageAttachment{Name: "brief.pdf", MIMEType: "application/pdf", Kind: "file", Text: "not a PDF"},
			want:       "unsupported text attachment MIME type",
		},
		{
			name:       "declared oversize",
			attachment: domain.MessageAttachment{Name: "large.txt", Kind: "file", Text: "small", Size: maxSessionMessageAttachmentBytes + 1},
			want:       "exceeds the 50 MB attachment limit",
		},
		{
			name:       "negative size",
			attachment: domain.MessageAttachment{Name: "negative.txt", Kind: "file", Text: "small", Size: -1},
			want:       "invalid attachment size",
		},
		{
			name:       "ambiguous content",
			attachment: domain.MessageAttachment{Name: "mixed", Kind: "file", Data: "YQ==", Text: "a"},
			want:       "cannot contain both binary data and text",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateSessionMessageAttachments([]domain.MessageAttachment{test.attachment})
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want substring %q", err, test.want)
			}
		})
	}
}

func TestValidateSessionMessageAttachmentsAcceptsMatchingSupportedDataURL(t *testing.T) {
	err := validateSessionMessageAttachments([]domain.MessageAttachment{{
		Name:     "brief.pdf",
		MIMEType: "application/pdf",
		Kind:     "file",
		Data:     "data:application/pdf;base64,JVBERi0xLjc=",
	}})
	if err != nil {
		t.Fatal(err)
	}
}

func TestNormalizeChatMessagesRejectsUnsupportedBinaryBeforeProvider(t *testing.T) {
	_, err := normalizeChatMessages([]domain.ChatMessage{{
		Role: "user",
		Attachments: []domain.MessageAttachment{{
			Name: "archive.zip", MIMEType: "application/octet-stream", Kind: "file", Data: "UEsDBA==",
		}},
	}})
	if err == nil || !strings.Contains(err.Error(), "unsupported binary attachment MIME type") {
		t.Fatalf("error = %v, want unsupported MIME refusal", err)
	}
}
