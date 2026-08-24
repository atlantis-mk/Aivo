package http

import (
	"encoding/json"
	"net/http"
)

type healthResponse struct {
	Status string `json:"status"`
	Name   string `json:"name"`
}

type rpcError struct {
	Error string `json:"error"`
}

func handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, healthResponse{
		Status: "ok",
		Name:   "aivo-core",
	})
}

func writeJSON(w http.ResponseWriter, status int, value interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, rpcError{Error: err.Error()})
}
