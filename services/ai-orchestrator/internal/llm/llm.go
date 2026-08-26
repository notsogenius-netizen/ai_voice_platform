// Package llm defines the streaming language-model boundary for ai-orchestrator.
package llm

import "context"

// Role is a chat message author.
type Role string

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
)

// Message is one turn in a conversation.
type Message struct {
	Role    Role
	Content string
}

// Chunk is a streamed piece of model output.
type Chunk struct {
	Text string
	Err  error
}

// Client streams chat completions from a configured provider.
type Client interface {
	Stream(ctx context.Context, messages []Message) (<-chan Chunk, error)
}
