package deepgram_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sourabh/ai-voice-platform/services/voice-gateway/internal/tts"
	"github.com/sourabh/ai-voice-platform/services/voice-gateway/internal/tts/deepgram"
)

func TestNewClientRequiresAPIKey(t *testing.T) {
	_, err := deepgram.NewClient(deepgram.Config{})
	if err == nil {
		t.Fatal("expected error without API key")
	}
}

func TestSynthesizeReturnsOgg(t *testing.T) {
	ogg := []byte("OggS-fake-opus")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s", r.Method)
		}
		if got := r.Header.Get("Authorization"); got != "Token test-key" {
			t.Fatalf("Authorization = %q", got)
		}
		q := r.URL.Query()
		if q.Get("model") != "aura-2-thalia-en" {
			t.Fatalf("model = %q", q.Get("model"))
		}
		if q.Get("encoding") != "opus" || q.Get("container") != "ogg" {
			t.Fatalf("query = %v", q)
		}
		body, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(body), `"Hello"`) {
			t.Fatalf("body = %s", body)
		}
		w.Header().Set("Content-Type", "audio/ogg")
		_, _ = w.Write(ogg)
	}))
	defer srv.Close()

	client := mustClient(t, srv.URL)
	audio, err := client.Synthesize(context.Background(), tts.Request{Text: "Hello"})
	if err != nil {
		t.Fatalf("Synthesize: %v", err)
	}
	if string(audio.Ogg) != string(ogg) {
		t.Fatalf("Ogg = %v", audio.Ogg)
	}
}

func TestSynthesizeRejectsEmptyText(t *testing.T) {
	client := mustClient(t, "http://example.test")
	_, err := client.Synthesize(context.Background(), tts.Request{Text: "  "})
	if err == nil {
		t.Fatal("expected error for empty text")
	}
}

func TestSynthesizeHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = io.WriteString(w, `invalid credentials`)
	}))
	defer srv.Close()

	client := mustClient(t, srv.URL)
	_, err := client.Synthesize(context.Background(), tts.Request{Text: "Hello"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "401") {
		t.Fatalf("err = %v", err)
	}
}

func mustClient(t *testing.T, speakURL string) *deepgram.Client {
	t.Helper()
	client, err := deepgram.NewClient(deepgram.Config{
		APIKey:   "test-key",
		SpeakURL: speakURL,
	})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return client
}
