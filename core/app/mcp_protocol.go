package app

import (
	"bytes"
	"encoding/json"
	"errors"
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
	if strings.TrimSpace(challenge) == "" {
		return out
	}
	text := strings.TrimSpace(challenge)
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

func parseMCPHTTPResponse(contentType string, body []byte) (map[string]any, error) {
	return parseMCPHTTPResponseWithNotifications(contentType, body, nil)
}

func parseMCPHTTPResponseWithNotifications(contentType string, body []byte, handle func(json.RawMessage, string) error) (map[string]any, error) {
	payload := bytes.TrimSpace(body)
	if strings.Contains(strings.ToLower(contentType), "text/event-stream") || bytes.HasPrefix(payload, []byte("event:")) || bytes.HasPrefix(payload, []byte("data:")) {
		return parseMCPSSEResponse(payload, handle)
	}
	result, _, err := parseMCPJSONRPCMessage(payload, handle)
	return result, err
}

func parseMCPSSEResponse(body []byte, handle func(json.RawMessage, string) error) (map[string]any, error) {
	payloads, err := mcpJSONPayloadsFromSSE(body)
	if err != nil {
		return nil, err
	}
	for _, payload := range payloads {
		result, response, err := parseMCPJSONRPCMessage(payload, handle)
		if err != nil {
			return nil, err
		}
		if response {
			return result, nil
		}
	}
	return map[string]any{}, nil
}

func parseMCPJSONRPCResult(payload []byte) (map[string]any, error) {
	result, _, err := parseMCPJSONRPCMessage(payload, nil)
	return result, err
}

func parseMCPJSONRPCMessage(payload []byte, handle func(json.RawMessage, string) error) (map[string]any, bool, error) {
	payload = bytes.TrimSpace(payload)
	if len(payload) == 0 {
		return map[string]any{}, false, nil
	}
	var message struct {
		ID     json.RawMessage `json:"id"`
		Method string          `json:"method"`
		Result map[string]any  `json:"result"`
		Error  any             `json:"error"`
	}
	if err := json.Unmarshal(payload, &message); err != nil {
		return nil, false, err
	}
	if strings.TrimSpace(message.Method) != "" {
		if handle != nil {
			if err := handle(message.ID, message.Method); err != nil {
				return nil, false, err
			}
		}
		return map[string]any{}, false, nil
	}
	if len(message.ID) == 0 || string(message.ID) == "null" {
		return map[string]any{}, false, nil
	}
	if message.Error != nil {
		return nil, true, mcpRPCError(message.Error)
	}
	if message.Result == nil {
		return map[string]any{}, true, nil
	}
	return message.Result, true, nil
}

func mcpJSONFromSSE(body []byte) ([]byte, error) {
	payloads, err := mcpJSONPayloadsFromSSE(body)
	if err != nil {
		return nil, err
	}
	for _, payload := range payloads {
		if len(bytes.TrimSpace(payload)) > 0 {
			return payload, nil
		}
	}
	return nil, errors.New("mcp sse response did not contain data")
}

func mcpJSONPayloadsFromSSE(body []byte) ([][]byte, error) {
	events := strings.Split(string(body), "\n\n")
	payloads := make([][]byte, 0, len(events))
	for _, event := range events {
		lines := strings.Split(event, "\n")
		data := strings.Builder{}
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "data:") {
				data.WriteString(strings.TrimSpace(strings.TrimPrefix(line, "data:")))
			}
		}
		if strings.TrimSpace(data.String()) != "" {
			payloads = append(payloads, []byte(data.String()))
		}
	}
	if len(payloads) == 0 {
		return nil, errors.New("mcp sse response did not contain data")
	}
	return payloads, nil
}

func mcpRPCError(value any) error {
	if item, ok := value.(map[string]any); ok {
		if message, _ := item["message"].(string); message != "" {
			return errors.New(sanitizeMCPError(message))
		}
	}
	return errors.New(sanitizeMCPError(fmt.Sprintf("%v", value)))
}

var mcpCredentialPattern = regexp.MustCompile(`(?i)(Bearer\s+[A-Za-z0-9._~+/=-]+|sk-[A-Za-z0-9._-]{8,}|ghp_[A-Za-z0-9_]{8,}|(?:token|key|api_key|password|secret)=["']?[^&\s,"']+)`)

func sanitizeMCPError(message string) string {
	message = strings.TrimSpace(message)
	if message == "" {
		return ""
	}
	return mcpCredentialPattern.ReplaceAllString(message, "[redacted]")
}
