// Package config loads voice-gateway settings from the environment.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

const (
	defaultDeepgramListenURL = "wss://api.deepgram.com/v1/listen"
	defaultSTTSampleRate     = 16000
)

// Config holds runtime settings for the voice-gateway process.
type Config struct {
	Addr               string
	CORSOrigin         string
	LiveKitURL         string
	LiveKitAPIKey      string
	LiveKitAPISecret   string
	DeepgramAPIKey     string
	DeepgramListenURL  string
	STTSampleRate      int
	OrchestratorURL    string
}

// Load reads configuration from environment variables.
func Load() (Config, error) {
	sampleRate, err := envIntOr("STT_SAMPLE_RATE", defaultSTTSampleRate)
	if err != nil {
		return Config{}, fmt.Errorf("STT_SAMPLE_RATE: %w", err)
	}

	cfg := Config{
		Addr:              envOr("VOICE_GATEWAY_ADDR", ":8080"),
		CORSOrigin:        envOr("VOICE_GATEWAY_CORS_ORIGIN", "http://127.0.0.1:5173"),
		LiveKitURL:        envOr("LIVEKIT_URL", "ws://127.0.0.1:7880"),
		LiveKitAPIKey:     envOr("LIVEKIT_API_KEY", ""),
		LiveKitAPISecret:  envOr("LIVEKIT_API_SECRET", ""),
		DeepgramAPIKey:    strings.TrimSpace(os.Getenv("DEEPGRAM_API_KEY")),
		DeepgramListenURL: envOr("DEEPGRAM_LISTEN_URL", defaultDeepgramListenURL),
		STTSampleRate:     sampleRate,
		OrchestratorURL:   strings.TrimSpace(os.Getenv("AI_ORCHESTRATOR_URL")),
	}
	if err := cfg.validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// STTEnabled reports whether streaming STT is configured.
func (c Config) STTEnabled() bool {
	return c.DeepgramAPIKey != ""
}

// OrchestratorEnabled reports whether transcript forwarding is configured.
func (c Config) OrchestratorEnabled() bool {
	return c.OrchestratorURL != ""
}

func (c Config) validate() error {
	checks := []struct {
		name  string
		value string
	}{
		{"VOICE_GATEWAY_ADDR", c.Addr},
		{"VOICE_GATEWAY_CORS_ORIGIN", c.CORSOrigin},
		{"LIVEKIT_URL", c.LiveKitURL},
		{"LIVEKIT_API_KEY", c.LiveKitAPIKey},
		{"LIVEKIT_API_SECRET", c.LiveKitAPISecret},
	}
	for _, check := range checks {
		if strings.TrimSpace(check.value) == "" {
			return fmt.Errorf("%s must not be empty", check.name)
		}
	}
	if strings.TrimSpace(c.DeepgramListenURL) == "" {
		return fmt.Errorf("DEEPGRAM_LISTEN_URL must not be empty")
	}
	if c.STTSampleRate <= 0 {
		return fmt.Errorf("STT_SAMPLE_RATE must be positive")
	}
	return nil
}

func envIntOr(key string, fallback int) (int, error) {
	raw, ok := os.LookupEnv(key)
	if !ok || strings.TrimSpace(raw) == "" {
		return fallback, nil
	}
	n, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil {
		return 0, fmt.Errorf("invalid integer %q", raw)
	}
	return n, nil
}

func envOr(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && strings.TrimSpace(v) != "" {
		return v
	}
	return fallback
}
