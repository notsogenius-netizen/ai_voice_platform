// Package token mints LiveKit access tokens for room participants.
package token

import (
	"fmt"
	"time"

	"github.com/livekit/protocol/auth"
)

// Grants describes what a participant may do in a room.
type Grants struct {
	CanPublish   bool
	CanSubscribe bool
}

// BrowserGrants are the default grants for a microphone client.
func BrowserGrants() Grants {
	return Grants{CanPublish: true, CanSubscribe: true}
}

// Minter creates LiveKit JWTs using an API key/secret pair.
type Minter struct {
	APIKey    string
	APISecret string
	ValidFor  time.Duration
}

// Mint returns a signed access token for identity in room.
func (m Minter) Mint(identity, room string, grants Grants) (string, error) {
	if identity == "" {
		return "", fmt.Errorf("identity is required")
	}
	if room == "" {
		return "", fmt.Errorf("room is required")
	}
	validFor := m.ValidFor
	if validFor <= 0 {
		validFor = time.Hour
	}

	at := auth.NewAccessToken(m.APIKey, m.APISecret)
	grant := &auth.VideoGrant{
		RoomJoin: true,
		Room:     room,
	}
	grant.SetCanPublish(grants.CanPublish)
	grant.SetCanSubscribe(grants.CanSubscribe)

	at.SetVideoGrant(grant).
		SetIdentity(identity).
		SetValidFor(validFor)

	return at.ToJWT()
}
