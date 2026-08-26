package httpserver_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sourabh/ai-voice-platform/services/voice-gateway/internal/config"
	"github.com/sourabh/ai-voice-platform/services/voice-gateway/internal/httpserver"
)

func TestHealthz(t *testing.T) {
	handler := httpserver.NewMux(config.Config{
		Addr:       ":8080",
		CORSOrigin: "http://127.0.0.1:5173",
	})

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
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "http://127.0.0.1:5173" {
		t.Fatalf("CORS origin = %q", got)
	}
}

func TestCORSPreflight(t *testing.T) {
	handler := httpserver.NewMux(config.Config{
		Addr:       ":8080",
		CORSOrigin: "http://127.0.0.1:5173",
	})

	req := httptest.NewRequest(http.MethodOptions, "/healthz", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", rec.Code)
	}
}
