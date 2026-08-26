// Package session creates browser voice sessions (room + access token).
package session

import (
	"fmt"

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

// Service creates sessions backed by LiveKit tokens.
type Service struct {
	LiveKitURL string
	Minter     token.Minter
}

// Create allocates a room name and mints a browser participant token.
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

	return Response{
		Room:       room,
		LiveKitURL: s.LiveKitURL,
		Token:      jwt,
		Identity:   identity,
	}, nil
}
