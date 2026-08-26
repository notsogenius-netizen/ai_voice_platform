// Package config loads ai-orchestrator settings from the environment.
package config

import (
	"fmt"
	"os"
	"strings"
)

const (
	defaultLLMBaseURL = "https://generativelanguage.googleapis.com/v1beta/openai/"
	defaultLLMModel   = "gemini-3.6-flash"
)

// Config holds runtime settings for the ai-orchestrator process.
type Config struct {
	Addr              string
	LLMAPIKey         string
	LLMBaseURL        string
	LLMModel          string
	AgentSystemPrompt string
}

// Load reads configuration from environment variables.
func Load() (Config, error) {
	cfg := Config{
		Addr:              envOr("AI_ORCHESTRATOR_ADDR", ":8081"),
		LLMAPIKey:         strings.TrimSpace(os.Getenv("LLM_API_KEY")),
		LLMBaseURL:        envOr("LLM_BASE_URL", defaultLLMBaseURL),
		LLMModel:          envOr("LLM_MODEL", defaultLLMModel),
		AgentSystemPrompt: strings.TrimSpace(os.Getenv("AGENT_SYSTEM_PROMPT")),
	}
	if err := cfg.validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// LLMEnabled reports whether an LLM provider is configured.
func (c Config) LLMEnabled() bool {
	return c.LLMAPIKey != ""
}

func (c Config) validate() error {
	if strings.TrimSpace(c.Addr) == "" {
		return fmt.Errorf("AI_ORCHESTRATOR_ADDR must not be empty")
	}
	if strings.TrimSpace(c.LLMBaseURL) == "" {
		return fmt.Errorf("LLM_BASE_URL must not be empty")
	}
	if strings.TrimSpace(c.LLMModel) == "" {
		return fmt.Errorf("LLM_MODEL must not be empty")
	}
	return nil
}

func envOr(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && strings.TrimSpace(v) != "" {
		return v
	}
	return fallback
}
