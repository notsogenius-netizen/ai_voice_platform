// Package tts defines the text-to-speech boundary for voice-gateway.
package tts

import "context"

// Request is text to synthesize into speech.
type Request struct {
	Text string
}

// Audio is synthesized speech ready for LiveKit publish.
// Phase 4 uses Ogg Opus (matches verification-tone publish path; no CGO).
type Audio struct {
	// Ogg is an Ogg Opus container suitable for lksdk.NewLocalReaderTrack.
	Ogg []byte
}

// Client synthesizes speech from text.
type Client interface {
	Synthesize(ctx context.Context, req Request) (Audio, error)
}
