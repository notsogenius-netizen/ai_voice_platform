// Package stt defines the streaming speech-to-text boundary for voice-gateway.
package stt

import "context"

// Session identifies a LiveKit audio source within a voice session.
type Session struct {
	Room        string
	Participant string
	TrackID     string
}

// Transcript is a partial or final recognition result.
type Transcript struct {
	Session Session
	Text    string
	IsFinal bool
}

// Stream accepts PCM audio and emits transcript events until closed.
type Stream interface {
	WritePCM(pcm []byte) error
	Transcripts() <-chan Transcript
	Close() error
}

// Client opens provider-specific streaming recognition sessions.
type Client interface {
	Open(ctx context.Context, sess Session) (Stream, error)
}
