package http

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

func (api *API) handleEvents(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, errors.New("streaming is unavailable"))
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	ch, unsubscribe := api.events.subscribe()
	defer unsubscribe()
	for {
		select {
		case <-r.Context().Done():
			return
		case event := <-ch:
			data, _ := json.Marshal(event.Data)
			fmt.Fprintf(w, "id: %d\n", event.ID)
			fmt.Fprintf(w, "event: %s\n", event.Name)
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
		}
	}
}
