package httpserver_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sourabh/ai-voice-platform/services/ai-orchestrator/internal/config"
	"github.com/sourabh/ai-voice-platform/services/ai-orchestrator/internal/conversation"
	"github.com/sourabh/ai-voice-platform/services/ai-orchestrator/internal/httpserver"
	"github.com/sourabh/ai-voice-platform/services/ai-orchestrator/internal/llm"
)

type stubLLM struct {
	reply string
}

func (s *stubLLM) Stream(_ context.Context, _ []llm.Message) (<-chan llm.Chunk, error) {
	ch := make(chan llm.Chunk, 2)
	ch <- llm.Chunk{Text: s.reply}
	close(ch)
	return ch, nil
}

func testHandler(t *testing.T, llmClient llm.Client) http.Handler {
	t.Helper()
	cfg := config.Config{Addr: ":8081"}
	svc := conversation.NewService(llmClient, "test-system")
	return httpserver.NewMux(cfg, httpserver.Deps{Conversations: svc})
}

func TestHealthz(t *testing.T) {
	handler := testHandler(t, &stubLLM{reply: "ok"})

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body, _ := io.ReadAll(rec.Body)
	if string(body) != "ok\n" {
		t.Fatalf("body = %q, want ok\\n", body)
	}
}

func TestTurnIgnoresPartialTranscript(t *testing.T) {
	handler := testHandler(t, &stubLLM{reply: "unused"})

	req := httptest.NewRequest(
		http.MethodPost,
		"/v1/sessions/room-1/turn",
		strings.NewReader(`{"text":"hel","is_final":false}`),
	)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}

	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["status"] != "ignored" {
		t.Fatalf("status = %q", body["status"])
	}
}

func TestTurnStreamsFinalReply(t *testing.T) {
	handler := testHandler(t, &stubLLM{reply: "Hello there"})

	req := httptest.NewRequest(
		http.MethodPost,
		"/v1/sessions/room-1/turn",
		strings.NewReader(`{"text":"hi","is_final":true}`),
	)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Header().Get("Content-Type"), "text/event-stream") {
		t.Fatalf("content-type = %q", rec.Header().Get("Content-Type"))
	}

	body := rec.Body.String()
	if !strings.Contains(body, `"text":"Hello there"`) {
		t.Fatalf("body = %q", body)
	}
	if !strings.Contains(body, "data: [DONE]") {
		t.Fatalf("missing done marker: %q", body)
	}
}

func TestTurnRequiresLLM(t *testing.T) {
	cfg := config.Config{Addr: ":8081"}
	svc := conversation.NewService(nil, "test-system")
	handler := httpserver.NewMux(cfg, httpserver.Deps{Conversations: svc})

	req := httptest.NewRequest(
		http.MethodPost,
		"/v1/sessions/room-1/turn",
		strings.NewReader(`{"text":"hi","is_final":true}`),
	)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestTurnRejectsInvalidJSON(t *testing.T) {
	handler := testHandler(t, &stubLLM{reply: "ok"})

	req := httptest.NewRequest(
		http.MethodPost,
		"/v1/sessions/room-1/turn",
		strings.NewReader(`{not-json`),
	)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
}
