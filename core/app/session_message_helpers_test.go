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
			attachment: domain.MessageAttachment{Name: "brief.pdf", Kind: "file", Data: "%%%"},
			want:       "invalid base64 attachment data",
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
