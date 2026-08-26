// Package conversation manages in-memory voice session context and LLM turns.
package conversation

import (
	"context"
	"errors"
	"strings"

	"github.com/sourabh/ai-voice-platform/services/ai-orchestrator/internal/llm"
)

var errPartialTranscript = errors.New("partial transcript")

// TurnRequest is one transcript event from the voice gateway.
type TurnRequest struct {
	Text    string
	IsFinal bool
}

// Service coordinates session history and streaming LLM replies.
type Service struct {
	LLM          llm.Client
	SystemPrompt string
	store        *store
}

// NewService returns a conversation service with the given LLM client.
func NewService(client llm.Client, systemPrompt string) *Service {
	if strings.TrimSpace(systemPrompt) == "" {
		systemPrompt = defaultSystemPrompt
	}
	return &Service{
		LLM:          client,
		SystemPrompt: systemPrompt,
		store:        newStore(),
	}
}

// HandleTurn processes a transcript. Partial transcripts are ignored.
// Final transcripts append user input, stream an assistant reply, and persist it.
func (s *Service) HandleTurn(ctx context.Context, sessionID string, req TurnRequest) (<-chan llm.Chunk, error) {
	text, err := s.prepareFinalTurn(sessionID, req)
	if err != nil {
		if errors.Is(err, errPartialTranscript) {
			return nil, nil
		}
		return nil, err
	}

	messages := s.store.appendUser(sessionID, s.SystemPrompt, text)
	upstream, err := s.LLM.Stream(ctx, messages)
	if err != nil {
		return nil, err
	}

	out := make(chan llm.Chunk, 16)
	go s.forwardTurn(sessionID, upstream, out)
	return out, nil
}

func (s *Service) prepareFinalTurn(sessionID string, req TurnRequest) (string, error) {
	if s.LLM == nil {
		return "", errors.New("llm not configured")
	}
	if strings.TrimSpace(sessionID) == "" {
		return "", errors.New("session id is required")
	}
	if !req.IsFinal {
		return "", errPartialTranscript
	}
	text := strings.TrimSpace(req.Text)
	if text == "" {
		return "", errors.New("text is required for final transcripts")
	}
	return text, nil
}

func (s *Service) forwardTurn(sessionID string, upstream <-chan llm.Chunk, out chan<- llm.Chunk) {
	defer close(out)

	var assistant strings.Builder
	for chunk := range upstream {
		if chunk.Err != nil {
			out <- chunk
			return
		}
		if chunk.Text != "" {
			assistant.WriteString(chunk.Text)
		}
		out <- chunk
	}

	if assistant.Len() > 0 {
		s.store.appendAssistant(sessionID, assistant.String())
	}
}

// History returns a copy of the stored messages for tests and debugging.
func (s *Service) History(sessionID string) []llm.Message {
	return s.store.snapshot(sessionID, s.SystemPrompt)
}
