package http

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

type rpcRequest struct {
	Args []json.RawMessage `json:"args"`
}

func (api *API) handleRPC(w http.ResponseWriter, r *http.Request) {
	method := strings.TrimSpace(r.PathValue("method"))
	var request rpcRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	result, err := api.call(r.Context(), method, request.Args)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func arg[T any](args []json.RawMessage, index int) (T, error) {
	var zero T
	if index >= len(args) {
		return zero, fmt.Errorf("missing argument %d", index)
	}
	if err := json.Unmarshal(args[index], &zero); err != nil {
		return zero, err
	}
	return zero, nil
}
