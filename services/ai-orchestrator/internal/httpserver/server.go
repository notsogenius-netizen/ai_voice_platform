// Package httpserver provides the ai-orchestrator HTTP surface.
package httpserver

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/sourabh/ai-voice-platform/services/ai-orchestrator/internal/config"
	"github.com/sourabh/ai-voice-platform/services/ai-orchestrator/internal/conversation"
	"github.com/sourabh/ai-voice-platform/services/ai-orchestrator/internal/llm"
)

// Deps are collaborators required by HTTP handlers.
type Deps struct {
	Conversations *conversation.Service
}

// NewMux builds the HTTP handler tree for ai-orchestrator.
func NewMux(_ config.Config, deps Deps) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", handleHealthz)
	mux.HandleFunc("POST /v1/sessions/{id}/turn", handleTurn(deps.Conversations))
	return mux
}

func handleHealthz(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok\n"))
}

type turnRequest struct {
	Text    string `json:"text"`
	IsFinal bool   `json:"is_final"`
}

func handleTurn(svc *conversation.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if svc == nil {
			writeJSONError(w, http.StatusServiceUnavailable, "conversation service unavailable")
			return
		}

		sessionID := r.PathValue("id")
		req, ok := decodeTurnRequest(w, r)
		if !ok {
			return
		}

		if !req.IsFinal {
			writeJSON(w, http.StatusAccepted, map[string]string{
				"status": "ignored",
				"reason": "partial transcript",
			})
			return
		}

		stream, err := svc.HandleTurn(r.Context(), sessionID, conversation.TurnRequest{
			Text:    req.Text,
			IsFinal: req.IsFinal,
		})
		if err != nil {
			writeJSONError(w, turnErrorStatus(err), err.Error())
			return
		}

		streamTurn(w, stream)
	}
}

func turnErrorStatus(err error) int {
	msg := err.Error()
	switch {
	case msg == "llm not configured":
		return http.StatusServiceUnavailable
	case msg == "session id is required", msg == "text is required for final transcripts":
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}

func decodeTurnRequest(w http.ResponseWriter, r *http.Request) (turnRequest, bool) {
	var req turnRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
		writeJSONError(w, http.StatusBadRequest, "invalid JSON body")
		return turnRequest{}, false
	}
	return req, true
}

func streamTurn(w http.ResponseWriter, chunks <-chan llm.Chunk) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeJSONError(w, http.StatusInternalServerError, "streaming not supported")
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)

	for chunk := range chunks {
		if chunk.Err != nil {
			writeSSE(w, flusher, ssePayload{Error: chunk.Err.Error()})
			return
		}
		if strings.TrimSpace(chunk.Text) == "" {
			continue
		}
		writeSSE(w, flusher, ssePayload{Text: chunk.Text})
	}
	writeSSEDone(w, flusher)
}

type ssePayload struct {
	Text  string `json:"text,omitempty"`
	Error string `json:"error,omitempty"`
}

func writeSSE(w http.ResponseWriter, flusher http.Flusher, payload ssePayload) {
	body, err := json.Marshal(payload)
	if err != nil {
		log.Printf("encode sse payload: %v", err)
		return
	}
	_, _ = w.Write([]byte("data: "))
	_, _ = w.Write(body)
	_, _ = w.Write([]byte("\n\n"))
	flusher.Flush()
}

func writeSSEDone(w http.ResponseWriter, flusher http.Flusher) {
	_, _ = w.Write([]byte("data: [DONE]\n\n"))
	flusher.Flush()
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
