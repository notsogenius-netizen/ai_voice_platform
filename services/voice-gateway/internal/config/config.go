// Package config loads voice-gateway settings from the environment.
package config

import (
	"fmt"
	"os"
	"strings"
)

// Config holds runtime settings for the voice-gateway process.
type Config struct {
	Addr             string
	CORSOrigin       string
	LiveKitURL       string
	LiveKitAPIKey    string
	LiveKitAPISecret string
}

// Load reads configuration from environment variables.
func Load() (Config, error) {
	cfg := Config{
		Addr:             envOr("VOICE_GATEWAY_ADDR", ":8080"),
		CORSOrigin:       envOr("VOICE_GATEWAY_CORS_ORIGIN", "http://127.0.0.1:5173"),
		LiveKitURL:       envOr("LIVEKIT_URL", "ws://127.0.0.1:7880"),
		LiveKitAPIKey:    envOr("LIVEKIT_API_KEY", ""),
		LiveKitAPISecret: envOr("LIVEKIT_API_SECRET", ""),
	}
	if err := cfg.validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
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
	return nil
}

func envOr(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && strings.TrimSpace(v) != "" {
		return v
	}
	return fallback
}
