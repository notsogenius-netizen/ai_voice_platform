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

// Client forwards transcripts to ai-orchestrator.
type Client interface {
	SendTurn(ctx context.Context, turn Turn) (Reply, error)
}
