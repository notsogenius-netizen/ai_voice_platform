// Package config loads voice-gateway settings from the environment.
package config

import (
	"fmt"
	"os"
	"strings"
)

// Config holds runtime settings for the voice-gateway process.
type Config struct {
	Addr       string
	CORSOrigin string
}

// Load reads configuration from environment variables.
func Load() (Config, error) {
	cfg := Config{
		Addr:       envOr("VOICE_GATEWAY_ADDR", ":8080"),
		CORSOrigin: envOr("VOICE_GATEWAY_CORS_ORIGIN", "http://127.0.0.1:5173"),
	}
	if err := cfg.validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func (c Config) validate() error {
	if strings.TrimSpace(c.Addr) == "" {
		return fmt.Errorf("VOICE_GATEWAY_ADDR must not be empty")
	}
	if strings.TrimSpace(c.CORSOrigin) == "" {
		return fmt.Errorf("VOICE_GATEWAY_CORS_ORIGIN must not be empty")
	}
	return nil
}

func envOr(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && strings.TrimSpace(v) != "" {
		return v
	}
	return fallback
}
