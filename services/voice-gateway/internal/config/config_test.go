package config_test

import (
	"testing"

	"github.com/sourabh/ai-voice-platform/services/voice-gateway/internal/config"
)

func TestLoadRequiresLiveKitCredentials(t *testing.T) {
	t.Setenv("VOICE_GATEWAY_ADDR", ":8080")
	t.Setenv("VOICE_GATEWAY_CORS_ORIGIN", "http://127.0.0.1:5173")
	t.Setenv("LIVEKIT_URL", "ws://127.0.0.1:7880")
	t.Setenv("LIVEKIT_API_KEY", "")
	t.Setenv("LIVEKIT_API_SECRET", "")

	if _, err := config.Load(); err == nil {
		t.Fatal("expected error when LiveKit credentials are missing")
	}
}

func TestLoadFromEnv(t *testing.T) {
	t.Setenv("VOICE_GATEWAY_ADDR", ":9090")
	t.Setenv("VOICE_GATEWAY_CORS_ORIGIN", "http://localhost:3000")
	t.Setenv("LIVEKIT_URL", "ws://127.0.0.1:7880")
	t.Setenv("LIVEKIT_API_KEY", "devkey")
	t.Setenv("LIVEKIT_API_SECRET", "devsecret_ai_voice_platform_local_only")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Addr != ":9090" {
		t.Fatalf("Addr = %q, want :9090", cfg.Addr)
	}
	if cfg.LiveKitAPIKey != "devkey" {
		t.Fatalf("LiveKitAPIKey = %q", cfg.LiveKitAPIKey)
	}
}
