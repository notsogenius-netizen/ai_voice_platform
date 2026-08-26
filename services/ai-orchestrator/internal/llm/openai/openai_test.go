package openai_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/sourabh/ai-voice-platform/services/ai-orchestrator/internal/llm"
	"github.com/sourabh/ai-voice-platform/services/ai-orchestrator/internal/llm/openai"
)

func TestNewClientRequiresAPIKey(t *testing.T) {
	_, err := openai.NewClient(openai.Config{
		BaseURL: "https://example.test/v1",
		Model:   "test-model",
	})
	if err == nil {
		t.Fatal("expected error without API key")
	}
}

func TestStreamEmitsChunks(t *testing.T) {
	srv := newStreamServer(t, strings.Join([]string{
		`data: {"choices":[{"delta":{"content":"Hello"}}]}`,
		``,
		`data: {"choices":[{"delta":{"content":" world"}}]}`,
		``,
		`data: [DONE]`,
		``,
	}, "\n"))
	defer srv.Close()

	client := mustClient(t, srv.URL)
	ch, err := client.Stream(context.Background(), []llm.Message{
		{Role: llm.RoleUser, Content: "hi"},
	})
	if err != nil {
		t.Fatalf("Stream: %v", err)
	}

	got := collectText(t, ch)
	if got != "Hello world" {
		t.Fatalf("got %q, want %q", got, "Hello world")
	}
}

func TestStreamSkipsEmptyDeltas(t *testing.T) {
	srv := newStreamServer(t, strings.Join([]string{
		`data: {"choices":[{"delta":{"role":"assistant"}}]}`,
		``,
		`data: {"choices":[{"delta":{"content":"ok"}}]}`,
		``,
		`data: [DONE]`,
		``,
	}, "\n"))
	defer srv.Close()

	client := mustClient(t, srv.URL)
	ch, err := client.Stream(context.Background(), []llm.Message{
		{Role: llm.RoleUser, Content: "hi"},
	})
	if err != nil {
		t.Fatalf("Stream: %v", err)
	}

	got := collectText(t, ch)
	if got != "ok" {
		t.Fatalf("got %q, want ok", got)
	}
}

func TestStreamHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"nope"}`))
	}))
	defer srv.Close()

	client := mustClient(t, srv.URL)
	_, err := client.Stream(context.Background(), []llm.Message{
		{Role: llm.RoleUser, Content: "hi"},
	})
	if err == nil {
		t.Fatal("expected error for non-2xx response")
	}
}

func TestStreamUsesChatCompletionsPath(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, "data: [DONE]\n\n")
	}))
	defer srv.Close()

	client := mustClient(t, srv.URL+"/v1beta/openai")
	ch, err := client.Stream(context.Background(), []llm.Message{
		{Role: llm.RoleUser, Content: "hi"},
	})
	if err != nil {
		t.Fatalf("Stream: %v", err)
	}
	for range ch {
	}
	if gotPath != "/v1beta/openai/chat/completions" {
		t.Fatalf("path = %q, want /v1beta/openai/chat/completions", gotPath)
	}
}

func TestStreamRespectsContextCancel(t *testing.T) {
	srv := newStreamServer(t, strings.Repeat("data: {\"choices\":[{\"delta\":{\"content\":\"x\"}}]}\n\n", 100))
	defer srv.Close()

	client := mustClient(t, srv.URL)
	ctx, cancel := context.WithCancel(context.Background())

	ch, err := client.Stream(ctx, []llm.Message{
		{Role: llm.RoleUser, Content: "hi"},
	})
	if err != nil {
		t.Fatalf("Stream: %v", err)
	}

	if _, ok := <-ch; !ok {
		t.Fatal("expected at least one chunk before cancel")
	}
	cancel()

	for chunk := range ch {
		if chunk.Err != nil {
			return
		}
	}
	t.Fatal("expected context error during stream")
}

func newStreamServer(t *testing.T, body string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, body)
	}))
}

func mustClient(t *testing.T, baseURL string) *openai.Client {
	t.Helper()
	client, err := openai.NewClient(openai.Config{
		APIKey:         "test-key",
		BaseURL:        baseURL,
		Model:          "test-model",
		RequestTimeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return client
}

func collectText(t *testing.T, ch <-chan llm.Chunk) string {
	t.Helper()
	var got strings.Builder
	for chunk := range ch {
		if chunk.Err != nil {
			t.Fatalf("chunk error: %v", chunk.Err)
		}
		got.WriteString(chunk.Text)
	}
	return got.String()
}
