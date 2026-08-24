package app

import (
	"fmt"
	"regexp"
	"strings"
)

type mcpHTTPAuthChallengeError struct {
	StatusCode          int
	Challenge           string
	Body                string
	ErrorCode           string
	ErrorDescription    string
	ResourceMetadataURL string
	Resource            string
	Scope               string
}

func (e *mcpHTTPAuthChallengeError) Error() string {
	if e == nil {
		return ""
	}
	message := fmt.Sprintf("mcp http request requires authentication (status %d)", e.StatusCode)
	if e.ErrorCode != "" {
		message += "; error: " + e.ErrorCode
	}
	if e.ErrorDescription != "" {
		message += "; description: " + e.ErrorDescription
	}
	if e.ResourceMetadataURL != "" {
		message += "; oauth resource metadata: " + e.ResourceMetadataURL
	}
	if e.Scope != "" {
		message += "; required scope: " + e.Scope
	}
	if e.Body != "" {
		message += "; response: " + bounded(e.Body, 500)
	}
	return message
}

func mcpHTTPAuthError(statusCode int, challenge string, body []byte) error {
	params := mcpWWWAuthenticateParams(challenge)
	metadataURL := firstNonEmptyApp(params["resource_metadata"], params["resource"])
	return &mcpHTTPAuthChallengeError{
		StatusCode:          statusCode,
		Challenge:           challenge,
		Body:                strings.TrimSpace(string(body)),
		ErrorCode:           params["error"],
		ErrorDescription:    params["error_description"],
		ResourceMetadataURL: metadataURL,
		Resource:            params["resource"],
		Scope:               params["scope"],
	}
}

func mcpWWWAuthenticateParam(challenge string, name string) string {
	return mcpWWWAuthenticateParams(challenge)[strings.ToLower(strings.TrimSpace(name))]
}

func mcpWWWAuthenticateParams(challenge string) map[string]string {
	out := map[string]string{}
	text := strings.TrimSpace(challenge)
	if text == "" {
		return out
	}
	if idx := strings.IndexAny(text, " \t"); idx >= 0 {
		scheme := strings.ToLower(strings.TrimSpace(text[:idx]))
		if scheme == "bearer" || scheme == "basic" || scheme == "digest" {
			text = strings.TrimSpace(text[idx+1:])
		}
	}
	for i := 0; i < len(text); {
		for i < len(text) && (text[i] == ',' || text[i] == ' ' || text[i] == '\t') {
			i++
		}
		start := i
		for i < len(text) && text[i] != '=' && text[i] != ',' {
			i++
		}
		if i >= len(text) || text[i] != '=' {
			for i < len(text) && text[i] != ',' {
				i++
			}
			continue
		}
		key := strings.ToLower(strings.TrimSpace(text[start:i]))
		i++
		value := ""
		if i < len(text) && text[i] == '"' {
			i++
			var builder strings.Builder
			for i < len(text) {
				if text[i] == '\\' && i+1 < len(text) {
					builder.WriteByte(text[i+1])
					i += 2
					continue
				}
				if text[i] == '"' {
					i++
					break
				}
				builder.WriteByte(text[i])
				i++
			}
			value = builder.String()
		} else {
			startValue := i
			for i < len(text) && text[i] != ',' {
				i++
			}
			value = strings.TrimSpace(text[startValue:i])
		}
		if key != "" {
			out[key] = strings.TrimSpace(value)
		}
		for i < len(text) && text[i] != ',' {
			i++
		}
	}
	return out
}

var mcpCredentialPattern = regexp.MustCompile(`(?i)(Bearer\s+[A-Za-z0-9._~+/=-]+|sk-[A-Za-z0-9._-]{8,}|ghp_[A-Za-z0-9_]{8,}|(?:token|key|api_key|password|secret)=["']?[^&\s,"']+)`)

func sanitizeMCPError(message string) string {
	message = strings.TrimSpace(message)
	if message == "" {
		return ""
	}
	return mcpCredentialPattern.ReplaceAllString(message, "[redacted]")
}
