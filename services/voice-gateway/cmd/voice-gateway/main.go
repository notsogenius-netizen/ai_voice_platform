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
	"github.com/sourabh/ai-voice-platform/services/voice-gateway/internal/orchestrator"
	orchhttp "github.com/sourabh/ai-voice-platform/services/voice-gateway/internal/orchestrator/httpclient"
	"github.com/sourabh/ai-voice-platform/services/voice-gateway/internal/roombot"
	"github.com/sourabh/ai-voice-platform/services/voice-gateway/internal/session"
	"github.com/sourabh/ai-voice-platform/services/voice-gateway/internal/stt"
	sttdeepgram "github.com/sourabh/ai-voice-platform/services/voice-gateway/internal/stt/deepgram"
	"github.com/sourabh/ai-voice-platform/services/voice-gateway/internal/token"
	"github.com/sourabh/ai-voice-platform/services/voice-gateway/internal/tts"
	ttsdeepgram "github.com/sourabh/ai-voice-platform/services/voice-gateway/internal/tts/deepgram"
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
	logDependencyStatus(cfg)

	clients, err := newRuntimeClients(cfg)
	if err != nil {
		return err
	}
	srv, errCh := startServer(cfg, rootCtx, clients)
	return waitForShutdown(rootCtx, srv, errCh)
}

func logDependencyStatus(cfg config.Config) {
	logSTTStatus(cfg)
	logTTSStatus(cfg)
	logOrchestratorStatus(cfg)
}

type runtimeClients struct {
	stt  stt.Client
	tts  tts.Client
	orch orchestrator.Client
}

func newRuntimeClients(cfg config.Config) (runtimeClients, error) {
	sttClient, err := newSTTClient(cfg)
	if err != nil {
		return runtimeClients{}, err
	}
	ttsClient, err := newTTSClient(cfg)
	if err != nil {
		return runtimeClients{}, err
	}
	orchClient, err := newOrchestratorClient(cfg)
	if err != nil {
		return runtimeClients{}, err
	}
	return runtimeClients{stt: sttClient, tts: ttsClient, orch: orchClient}, nil
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

func logTTSStatus(cfg config.Config) {
	if cfg.TTSEnabled() {
		log.Printf(
			"tts: enabled provider=deepgram model=%s speak_url=%s",
			cfg.TTSModel,
			cfg.DeepgramSpeakURL,
		)
		return
	}
	log.Printf("tts: disabled (set DEEPGRAM_API_KEY to enable)")
}

func newSTTClient(cfg config.Config) (stt.Client, error) {
	if !cfg.STTEnabled() {
		return nil, nil
	}
	return sttdeepgram.NewClient(sttdeepgram.Config{
		APIKey:         cfg.DeepgramAPIKey,
		ListenURL:      cfg.DeepgramListenURL,
		SampleRate:     cfg.STTSampleRate,
		ConnectTimeout: 10 * time.Second,
	})
}

func newTTSClient(cfg config.Config) (tts.Client, error) {
	if !cfg.TTSEnabled() {
		return nil, nil
	}
	return ttsdeepgram.NewClient(ttsdeepgram.Config{
		APIKey:   cfg.DeepgramAPIKey,
		SpeakURL: cfg.DeepgramSpeakURL,
		Model:    cfg.TTSModel,
	})
}

func logOrchestratorStatus(cfg config.Config) {
	if cfg.OrchestratorEnabled() {
		log.Printf("orchestrator: enabled url=%s", cfg.OrchestratorURL)
		return
	}
	log.Printf("orchestrator: disabled (set AI_ORCHESTRATOR_URL to enable)")
}

func newOrchestratorClient(cfg config.Config) (orchestrator.Client, error) {
	if !cfg.OrchestratorEnabled() {
		return nil, nil
	}
	return orchhttp.NewClient(orchhttp.Config{
		BaseURL:        cfg.OrchestratorURL,
		RequestTimeout: 60 * time.Second,
	})
}

func startServer(
	cfg config.Config,
	rootCtx context.Context,
	clients runtimeClients,
) (*http.Server, <-chan error) {
	srv := newHTTPServer(cfg, rootCtx, clients)
	errCh := make(chan error, 1)
	go func() {
		log.Printf("voice-gateway listening on %s", cfg.Addr)
		errCh <- srv.ListenAndServe()
	}()
	return srv, errCh
}

func newHTTPServer(cfg config.Config, rootCtx context.Context, clients runtimeClients) *http.Server {
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
			Bot:        newRoomBot(cfg, minter, clients),
		},
	}
	return &http.Server{
		Addr:              cfg.Addr,
		Handler:           httpserver.NewMux(cfg, deps),
		ReadHeaderTimeout: 5 * time.Second,
	}
}

func newRoomBot(cfg config.Config, minter token.Minter, clients runtimeClients) roombot.Bot {
	return roombot.Bot{
		LiveKitURL:    cfg.LiveKitURL,
		Minter:        minter,
		STTSampleRate: cfg.STTSampleRate,
		STT:           clients.stt,
		TTS:           clients.tts,
		Orchestrator:  clients.orch,
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
