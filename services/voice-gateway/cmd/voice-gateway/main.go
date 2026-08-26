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

	"github.com/sourabh/ai-voice-platform/services/voice-gateway/internal/config"
	"github.com/sourabh/ai-voice-platform/services/voice-gateway/internal/httpserver"
	"github.com/sourabh/ai-voice-platform/services/voice-gateway/internal/roombot"
	"github.com/sourabh/ai-voice-platform/services/voice-gateway/internal/session"
	"github.com/sourabh/ai-voice-platform/services/voice-gateway/internal/stt/deepgram"
	"github.com/sourabh/ai-voice-platform/services/voice-gateway/internal/token"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("voice-gateway: %v", err)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	rootCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	logSTTStatus(cfg)
	if err := validateSTTClient(cfg); err != nil {
		return err
	}

	srv, errCh := startServer(cfg, rootCtx)
	return waitForShutdown(rootCtx, srv, errCh)
}

func logSTTStatus(cfg config.Config) {
	if cfg.STTEnabled() {
		log.Printf(
			"stt: enabled provider=deepgram sample_rate=%d listen_url=%s",
			cfg.STTSampleRate,
			cfg.DeepgramListenURL,
		)
		return
	}
	log.Printf("stt: disabled (set DEEPGRAM_API_KEY to enable)")
}

func validateSTTClient(cfg config.Config) error {
	if !cfg.STTEnabled() {
		return nil
	}
	_, err := deepgram.NewClient(deepgram.Config{
		APIKey:         cfg.DeepgramAPIKey,
		ListenURL:      cfg.DeepgramListenURL,
		SampleRate:     cfg.STTSampleRate,
		ConnectTimeout: 10 * time.Second,
	})
	if err != nil {
		return err
	}
	return nil
}

func startServer(cfg config.Config, rootCtx context.Context) (*http.Server, <-chan error) {
	minter := token.Minter{
		APIKey:    cfg.LiveKitAPIKey,
		APISecret: cfg.LiveKitAPISecret,
		ValidFor:  time.Hour,
	}

	deps := httpserver.Deps{
		Sessions: session.Service{
			LiveKitURL: cfg.LiveKitURL,
			Minter:     minter,
			RootCtx:    rootCtx,
			Bot: roombot.Bot{
				LiveKitURL:    cfg.LiveKitURL,
				Minter:        minter,
				STTSampleRate: cfg.STTSampleRate,
			},
		},
	}

	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           httpserver.NewMux(cfg, deps),
		ReadHeaderTimeout: 5 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		log.Printf("voice-gateway listening on %s", cfg.Addr)
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
