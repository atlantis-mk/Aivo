package http

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestHandleEventsWritesIDEventAndJSONData(t *testing.T) {
	api := &API{events: newEventBroker()}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	request := httptest.NewRequest("GET", "/api/events", nil).WithContext(ctx)
	recorder := httptest.NewRecorder()
	done := make(chan struct{})

	go func() {
		api.handleEvents(recorder, request)
		close(done)
	}()

	for {
		api.events.mu.Lock()
		clientCount := len(api.events.clients)
		api.events.mu.Unlock()
		if clientCount > 0 {
			break
		}
		time.Sleep(time.Millisecond)
	}
	api.events.emit("session.updated", map[string]string{"sessionId": "s1"})

	deadline := time.After(time.Second)
	for {
		body := recorder.Body.String()
		if strings.Contains(body, "event: session.updated") {
			if !strings.Contains(body, "id: 1\n") {
				t.Fatalf("SSE body missing id: %q", body)
			}
			if !strings.Contains(body, `data: {"sessionId":"s1"}`) {
				t.Fatalf("SSE body missing JSON data: %q", body)
			}
			cancel()
			<-done
			return
		}
		select {
		case <-deadline:
			t.Fatalf("SSE event was not written: %q", body)
		default:
			time.Sleep(time.Millisecond)
		}
	}

}
