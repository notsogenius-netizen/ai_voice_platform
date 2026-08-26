package config_test

import (
	"testing"

	"github.com/sourabh/ai-voice-platform/services/ai-orchestrator/internal/config"
)

func TestLoadDefaults(t *testing.T) {
	t.Setenv("AI_ORCHESTRATOR_ADDR", "")
	t.Setenv("LLM_API_KEY", "")
	t.Setenv("LLM_BASE_URL", "")
	t.Setenv("LLM_MODEL", "")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Addr != ":8081" {
		t.Fatalf("Addr = %q, want :8081", cfg.Addr)
	}
	if cfg.LLMBaseURL != "https://generativelanguage.googleapis.com/v1beta/openai/" {
		t.Fatalf("LLMBaseURL = %q", cfg.LLMBaseURL)
	}
	if cfg.LLMModel != "gemini-3.6-flash" {
		t.Fatalf("LLMModel = %q", cfg.LLMModel)
	}
	if cfg.LLMEnabled() {
		t.Fatal("LLM should be disabled without LLM_API_KEY")
	}
}

func TestLoadFromEnv(t *testing.T) {
	t.Setenv("AI_ORCHESTRATOR_ADDR", ":9091")
	t.Setenv("LLM_API_KEY", "gemini-test-key")
	t.Setenv("LLM_BASE_URL", "https://example.test/v1")
	t.Setenv("LLM_MODEL", "gpt-4o-mini")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Addr != ":9091" {
		t.Fatalf("Addr = %q, want :9091", cfg.Addr)
	}
	if !cfg.LLMEnabled() {
		t.Fatal("LLM should be enabled when LLM_API_KEY is set")
	}
	if cfg.LLMAPIKey != "gemini-test-key" {
		t.Fatalf("LLMAPIKey = %q", cfg.LLMAPIKey)
	}
	if cfg.LLMBaseURL != "https://example.test/v1" {
		t.Fatalf("LLMBaseURL = %q", cfg.LLMBaseURL)
	}
	if cfg.LLMModel != "gpt-4o-mini" {
		t.Fatalf("LLMModel = %q", cfg.LLMModel)
	}
}
