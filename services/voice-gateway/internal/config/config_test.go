package config_test

import (
	"testing"

	"github.com/sourabh/ai-voice-platform/services/voice-gateway/internal/config"
)

func TestLoadDefaults(t *testing.T) {
	t.Setenv("VOICE_GATEWAY_ADDR", "")
	t.Setenv("VOICE_GATEWAY_CORS_ORIGIN", "")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Addr != ":8080" {
		t.Fatalf("Addr = %q, want :8080", cfg.Addr)
	}
	if cfg.CORSOrigin != "http://127.0.0.1:5173" {
		t.Fatalf("CORSOrigin = %q", cfg.CORSOrigin)
	}
}

func TestLoadFromEnv(t *testing.T) {
	t.Setenv("VOICE_GATEWAY_ADDR", ":9090")
	t.Setenv("VOICE_GATEWAY_CORS_ORIGIN", "http://localhost:3000")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Addr != ":9090" {
		t.Fatalf("Addr = %q, want :9090", cfg.Addr)
	}
	if cfg.CORSOrigin != "http://localhost:3000" {
		t.Fatalf("CORSOrigin = %q", cfg.CORSOrigin)
	}
}
