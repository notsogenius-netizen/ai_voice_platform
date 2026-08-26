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

func TestLoadWithoutSTT(t *testing.T) {
	setBaseEnv(t)
	t.Setenv("DEEPGRAM_API_KEY", "")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.STTEnabled() {
		t.Fatal("STT should be disabled without DEEPGRAM_API_KEY")
	}
	if cfg.STTSampleRate != 16000 {
		t.Fatalf("STTSampleRate = %d, want 16000", cfg.STTSampleRate)
	}
	if cfg.DeepgramListenURL != "wss://api.deepgram.com/v1/listen" {
		t.Fatalf("DeepgramListenURL = %q", cfg.DeepgramListenURL)
	}
}

func TestLoadSTTFromEnv(t *testing.T) {
	setBaseEnv(t)
	t.Setenv("DEEPGRAM_API_KEY", "dg-test-key")
	t.Setenv("DEEPGRAM_LISTEN_URL", "wss://example.test/listen")
	t.Setenv("STT_SAMPLE_RATE", "48000")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !cfg.STTEnabled() {
		t.Fatal("STT should be enabled when DEEPGRAM_API_KEY is set")
	}
	if cfg.DeepgramAPIKey != "dg-test-key" {
		t.Fatalf("DeepgramAPIKey = %q", cfg.DeepgramAPIKey)
	}
	if cfg.DeepgramListenURL != "wss://example.test/listen" {
		t.Fatalf("DeepgramListenURL = %q", cfg.DeepgramListenURL)
	}
	if cfg.STTSampleRate != 48000 {
		t.Fatalf("STTSampleRate = %d, want 48000", cfg.STTSampleRate)
	}
}

func TestLoadRejectsInvalidSTTSampleRate(t *testing.T) {
	setBaseEnv(t)
	t.Setenv("STT_SAMPLE_RATE", "not-a-number")

	if _, err := config.Load(); err == nil {
		t.Fatal("expected error for invalid STT_SAMPLE_RATE")
	}
}

func TestLoadRejectsNonPositiveSTTSampleRate(t *testing.T) {
	setBaseEnv(t)
	t.Setenv("STT_SAMPLE_RATE", "0")

	if _, err := config.Load(); err == nil {
		t.Fatal("expected error for non-positive STT_SAMPLE_RATE")
	}
}

func setBaseEnv(t *testing.T) {
	t.Helper()
	t.Setenv("VOICE_GATEWAY_ADDR", ":8080")
	t.Setenv("VOICE_GATEWAY_CORS_ORIGIN", "http://127.0.0.1:5173")
	t.Setenv("LIVEKIT_URL", "ws://127.0.0.1:7880")
	t.Setenv("LIVEKIT_API_KEY", "devkey")
	t.Setenv("LIVEKIT_API_SECRET", "devsecret_ai_voice_platform_local_only")
}
