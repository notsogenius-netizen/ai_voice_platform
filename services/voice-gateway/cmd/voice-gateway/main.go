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
	"github.com/sourabh/ai-voice-platform/services/voice-gateway/internal/stt"
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
	sttClient, err := newSTTClient(cfg)
	if err != nil {
		return err
	}

	srv, errCh := startServer(cfg, rootCtx, sttClient)
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

func newSTTClient(cfg config.Config) (stt.Client, error) {
	if !cfg.STTEnabled() {
		return nil, nil
	}
	return deepgram.NewClient(deepgram.Config{
		APIKey:         cfg.DeepgramAPIKey,
		ListenURL:      cfg.DeepgramListenURL,
		SampleRate:     cfg.STTSampleRate,
		ConnectTimeout: 10 * time.Second,
	})
}

func startServer(cfg config.Config, rootCtx context.Context, sttClient stt.Client) (*http.Server, <-chan error) {
	srv := newHTTPServer(cfg, rootCtx, sttClient)

	errCh := make(chan error, 1)
	go func() {
		log.Printf("voice-gateway listening on %s", cfg.Addr)
		errCh <- srv.ListenAndServe()
	}()
	return srv, errCh
}

func newHTTPServer(cfg config.Config, rootCtx context.Context, sttClient stt.Client) *http.Server {
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
				STT:           sttClient,
			},
		},
	}

	return &http.Server{
		Addr:              cfg.Addr,
		Handler:           httpserver.NewMux(cfg, deps),
		ReadHeaderTimeout: 5 * time.Second,
	}
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
