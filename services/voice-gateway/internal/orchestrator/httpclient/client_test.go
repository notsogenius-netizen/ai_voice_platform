package httpclient_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sourabh/ai-voice-platform/services/voice-gateway/internal/orchestrator"
	"github.com/sourabh/ai-voice-platform/services/voice-gateway/internal/orchestrator/httpclient"
)

func TestNewClientRequiresBaseURL(t *testing.T) {
	_, err := httpclient.NewClient(httpclient.Config{})
	if err == nil {
		t.Fatal("expected error without base URL")
	}
}

func TestSendTurnStreamsReply(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/sessions/room-1/turn" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, strings.Join([]string{
			`data: {"text":"Hello"}`,
			``,
			`data: {"text":" there"}`,
			``,
			`data: [DONE]`,
			``,
		}, "\n"))
	}))
	defer srv.Close()

	client := mustClient(t, srv.URL)
	reply, err := client.SendTurn(context.Background(), orchestrator.Turn{
		SessionID: "room-1",
		Text:      "hi",
		IsFinal:   true,
	})
	if err != nil {
		t.Fatalf("SendTurn: %v", err)
	}
	if reply.Text != "Hello there" {
		t.Fatalf("reply = %q", reply.Text)
	}
}

func TestStreamTurnInvokesChunks(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, strings.Join([]string{
			`data: {"text":"Hello"}`,
			``,
			`data: {"text":" there."}`,
			``,
			`data: [DONE]`,
			``,
		}, "\n"))
	}))
	defer srv.Close()

	client := mustClient(t, srv.URL)
	var chunks []string
	reply, err := client.StreamTurn(
		context.Background(),
		orchestrator.Turn{SessionID: "room-1", Text: "hi", IsFinal: true},
		func(text string) error {
			chunks = append(chunks, text)
			return nil
		},
	)
	if err != nil {
		t.Fatalf("StreamTurn: %v", err)
	}
	if reply.Text != "Hello there." {
		t.Fatalf("reply = %q", reply.Text)
	}
	if len(chunks) != 2 || chunks[0] != "Hello" || chunks[1] != " there." {
		t.Fatalf("chunks = %#v", chunks)
	}
}

func TestSendTurnIgnoredPartial(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_, _ = io.WriteString(w, `{"status":"ignored","reason":"partial transcript"}`)
	}))
	defer srv.Close()

	client := mustClient(t, srv.URL)
	reply, err := client.SendTurn(context.Background(), orchestrator.Turn{
		SessionID: "room-1",
		Text:      "hel",
		IsFinal:   false,
	})
	if err != nil {
		t.Fatalf("SendTurn: %v", err)
	}
	if !reply.Ignored {
		t.Fatal("expected ignored reply")
	}
}

func TestSendTurnHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = io.WriteString(w, `{"error":"llm not configured"}`)
	}))
	defer srv.Close()

	client := mustClient(t, srv.URL)
	_, err := client.SendTurn(context.Background(), orchestrator.Turn{
		SessionID: "room-1",
		Text:      "hi",
		IsFinal:   true,
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func mustClient(t *testing.T, baseURL string) *httpclient.Client {
	t.Helper()
	client, err := httpclient.NewClient(httpclient.Config{BaseURL: baseURL})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return client
}
