// Package httpserver provides the voice-gateway HTTP surface.
package httpserver

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"

	"github.com/sourabh/ai-voice-platform/services/voice-gateway/internal/config"
	"github.com/sourabh/ai-voice-platform/services/voice-gateway/internal/session"
)

// Deps are collaborators required by HTTP handlers.
type Deps struct {
	Sessions session.Service
}

// NewMux builds the HTTP handler tree for voice-gateway.
func NewMux(cfg config.Config, deps Deps) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", handleHealthz)
	mux.HandleFunc("POST /v1/sessions", handleCreateSession(deps.Sessions))
	return withCORS(cfg.CORSOrigin, mux)
}

func handleHealthz(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok\n"))
}

func handleCreateSession(svc session.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		req, ok := decodeSessionRequest(w, r)
		if !ok {
			return
		}

		res, err := svc.Create(req)
		if err != nil {
			log.Printf("create session: %v", err)
			writeJSONError(w, http.StatusInternalServerError, "failed to create session")
			return
		}
		writeJSON(w, http.StatusCreated, res)
	}
}

func decodeSessionRequest(w http.ResponseWriter, r *http.Request) (session.Request, bool) {
	var req session.Request
	if r.Body == nil {
		return req, true
	}
	err := json.NewDecoder(r.Body).Decode(&req)
	if errors.Is(err, io.EOF) {
		return req, true
	}
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid JSON body")
		return session.Request{}, false
	}
	return req, true
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		log.Printf("encode response: %v", err)
	}
}

func writeJSONError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func withCORS(origin string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
