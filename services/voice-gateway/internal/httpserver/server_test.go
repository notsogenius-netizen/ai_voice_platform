package httpserver_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/sourabh/ai-voice-platform/services/voice-gateway/internal/config"
	"github.com/sourabh/ai-voice-platform/services/voice-gateway/internal/httpserver"
	"github.com/sourabh/ai-voice-platform/services/voice-gateway/internal/session"
	"github.com/sourabh/ai-voice-platform/services/voice-gateway/internal/token"
)

func testDeps() (config.Config, httpserver.Deps) {
	cfg := config.Config{
		Addr:             ":8080",
		CORSOrigin:       "http://127.0.0.1:5173",
		LiveKitURL:       "ws://127.0.0.1:7880",
		LiveKitAPIKey:    "devkey",
		LiveKitAPISecret: "devsecret_ai_voice_platform_local_only",
	}
	deps := httpserver.Deps{
		Sessions: session.Service{
			LiveKitURL: cfg.LiveKitURL,
			Minter: token.Minter{
				APIKey:    cfg.LiveKitAPIKey,
				APISecret: cfg.LiveKitAPISecret,
				ValidFor:  time.Hour,
			},
		},
	}
	return cfg, deps
}

func TestHealthz(t *testing.T) {
	cfg, deps := testDeps()
	handler := httpserver.NewMux(cfg, deps)

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

func TestCreateSession(t *testing.T) {
	cfg, deps := testDeps()
	handler := httpserver.NewMux(cfg, deps)

	req := httptest.NewRequest(
		http.MethodPost,
		"/v1/sessions",
		strings.NewReader(`{"identity":"tester"}`),
	)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}

	var res session.Response
	if err := json.NewDecoder(rec.Body).Decode(&res); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if res.Identity != "tester" || res.Token == "" || res.Room == "" {
		t.Fatalf("unexpected response: %+v", res)
	}
}

func TestCORSPreflight(t *testing.T) {
	cfg, deps := testDeps()
	handler := httpserver.NewMux(cfg, deps)

	req := httptest.NewRequest(http.MethodOptions, "/v1/sessions", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", rec.Code)
	}
}
