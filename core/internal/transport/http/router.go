package http

import "net/http"

func NewRouter() http.Handler {
	api, err := newAPI()
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", handleHealth)
	if err != nil {
		mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
			writeError(w, http.StatusInternalServerError, err)
		})
		return withCORS(mux)
	}
	mux.HandleFunc("POST /api/rpc/{method}", api.handleRPC)
	mux.HandleFunc("GET /api/events", api.handleEvents)
	mux.HandleFunc("POST /api/terminals/{id}/connect-token", api.handleTerminalConnectToken)
	mux.HandleFunc("GET /api/terminals/{id}/connect", api.handleTerminalConnect)
	return &routerWithShutdown{handler: withCORS(mux), api: api}
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Aivo-Terminal-CSRF")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
