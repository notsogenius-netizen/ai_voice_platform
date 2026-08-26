package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sourabh/ai-voice-platform/services/ai-orchestrator/internal/config"
	"github.com/sourabh/ai-voice-platform/services/ai-orchestrator/internal/conversation"
	"github.com/sourabh/ai-voice-platform/services/ai-orchestrator/internal/httpserver"
	"github.com/sourabh/ai-voice-platform/services/ai-orchestrator/internal/llm"
	"github.com/sourabh/ai-voice-platform/services/ai-orchestrator/internal/llm/openai"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("ai-orchestrator: %v", err)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	llmClient, err := newLLMClient(cfg)
	if err != nil {
		return err
	}
	logLLMStatus(cfg)

	conversations := conversation.NewService(llmClient, cfg.AgentSystemPrompt)

	rootCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	srv, errCh := startServer(cfg, conversations)
	return waitForShutdown(rootCtx, srv, errCh)
}

func newLLMClient(cfg config.Config) (llm.Client, error) {
	if !cfg.LLMEnabled() {
		return nil, nil
	}
	return openai.NewClient(openai.Config{
		APIKey:  cfg.LLMAPIKey,
		BaseURL: cfg.LLMBaseURL,
		Model:   cfg.LLMModel,
	})
}

func logLLMStatus(cfg config.Config) {
	if cfg.LLMEnabled() {
		log.Printf("llm: enabled model=%s base_url=%s", cfg.LLMModel, cfg.LLMBaseURL)
		return
	}
	log.Printf("llm: disabled (set LLM_API_KEY to enable)")
}

func startServer(cfg config.Config, conversations *conversation.Service) (*http.Server, <-chan error) {
	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           httpserver.NewMux(cfg, httpserver.Deps{Conversations: conversations}),
		ReadHeaderTimeout: 5 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		log.Printf("ai-orchestrator listening on %s", cfg.Addr)
		errCh <- srv.ListenAndServe()
	}()
	return srv, errCh
}

func waitForShutdown(ctx context.Context, srv *http.Server, errCh <-chan error) error {
	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}
