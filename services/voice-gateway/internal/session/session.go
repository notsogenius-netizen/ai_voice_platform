// Package session creates browser voice sessions (room + access token).
package session

import (
	"context"
	"fmt"
	"log"

	"github.com/google/uuid"

	"github.com/sourabh/ai-voice-platform/services/voice-gateway/internal/token"
)

// Request is an optional client hint when opening a session.
type Request struct {
	Identity string `json:"identity"`
}

// Response is returned to the browser so it can connect to LiveKit.
type Response struct {
	Room       string `json:"room"`
	LiveKitURL string `json:"livekit_url"`
	Token      string `json:"token"`
	Identity   string `json:"identity"`
}

// Joiner connects the voice-gateway bot into a LiveKit room.
type Joiner interface {
	Join(ctx context.Context, roomName string) error
}

// Service creates sessions backed by LiveKit tokens.
type Service struct {
	LiveKitURL string
	Minter     token.Minter
	RootCtx    context.Context
	Bot        Joiner
}

// Create allocates a room name, mints a browser token, and starts the bot join.
func (s Service) Create(req Request) (Response, error) {
	identity := req.Identity
	if identity == "" {
		identity = "browser-" + uuid.NewString()
	}
	room := "sess_" + uuid.NewString()

	jwt, err := s.Minter.Mint(identity, room, token.BrowserGrants())
	if err != nil {
		return Response{}, fmt.Errorf("mint token: %w", err)
	}

	s.startBot(room)

	return Response{
		Room:       room,
		LiveKitURL: s.LiveKitURL,
		Token:      jwt,
		Identity:   identity,
	}, nil
}

func (s Service) startBot(room string) {
	if s.Bot == nil {
		return
	}
	ctx := s.RootCtx
	if ctx == nil {
		ctx = context.Background()
	}
	go func() {
		if err := s.Bot.Join(ctx, room); err != nil {
			log.Printf("session bot join room=%s: %v", room, err)
		}
	}()
}
