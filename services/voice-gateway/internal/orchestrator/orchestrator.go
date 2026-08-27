// Package orchestrator defines the voice-gateway to ai-orchestrator boundary.
package orchestrator

import "context"

// Turn is a transcript forwarded for conversation processing.
type Turn struct {
	SessionID string
	Text      string
	IsFinal   bool
}

// Reply is the orchestrator response to a turn.
type Reply struct {
	Text    string
	Ignored bool
}

// ChunkHandler receives streamed assistant text fragments.
// Returning a non-nil error aborts the stream read.
type ChunkHandler func(text string) error

// Client forwards transcripts to ai-orchestrator.
type Client interface {
	SendTurn(ctx context.Context, turn Turn) (Reply, error)
	StreamTurn(ctx context.Context, turn Turn, onChunk ChunkHandler) (Reply, error)
}
