package conversation_test

import (
	"context"
	"testing"

	"github.com/sourabh/ai-voice-platform/services/ai-orchestrator/internal/conversation"
	"github.com/sourabh/ai-voice-platform/services/ai-orchestrator/internal/llm"
)

type stubLLM struct {
	lastMessages []llm.Message
	reply        string
}

func (s *stubLLM) Stream(_ context.Context, messages []llm.Message) (<-chan llm.Chunk, error) {
	s.lastMessages = append([]llm.Message(nil), messages...)
	ch := make(chan llm.Chunk, 2)
	ch <- llm.Chunk{Text: s.reply}
	close(ch)
	return ch, nil
}

func TestHandleTurnIgnoresPartialTranscripts(t *testing.T) {
	llmStub := &stubLLM{reply: "should not run"}
	svc := conversation.NewService(llmStub, "system")

	ch, err := svc.HandleTurn(context.Background(), "room-1", conversation.TurnRequest{
		Text:    "hel",
		IsFinal: false,
	})
	if err != nil {
		t.Fatalf("HandleTurn: %v", err)
	}
	if ch != nil {
		t.Fatal("expected nil channel for partial transcript")
	}
	if len(llmStub.lastMessages) != 0 {
		t.Fatal("llm should not be called for partial transcript")
	}
}

func TestHandleTurnStreamsFinalReplyAndStoresHistory(t *testing.T) {
	llmStub := &stubLLM{reply: "Hello there"}
	svc := conversation.NewService(llmStub, "system")

	ch, err := svc.HandleTurn(context.Background(), "room-1", conversation.TurnRequest{
		Text:    "hi",
		IsFinal: true,
	})
	if err != nil {
		t.Fatalf("HandleTurn: %v", err)
	}

	got := readText(t, ch)
	if got != "Hello there" {
		t.Fatalf("reply = %q, want Hello there", got)
	}

	history := svc.History("room-1")
	if len(history) != 3 {
		t.Fatalf("history len = %d, want 3", len(history))
	}
	if history[0].Role != llm.RoleSystem || history[0].Content != "system" {
		t.Fatalf("system message = %+v", history[0])
	}
	if history[1].Role != llm.RoleUser || history[1].Content != "hi" {
		t.Fatalf("user message = %+v", history[1])
	}
	if history[2].Role != llm.RoleAssistant || history[2].Content != "Hello there" {
		t.Fatalf("assistant message = %+v", history[2])
	}
}

func TestHandleTurnRequiresLLM(t *testing.T) {
	svc := conversation.NewService(nil, "system")
	_, err := svc.HandleTurn(context.Background(), "room-1", conversation.TurnRequest{
		Text:    "hi",
		IsFinal: true,
	})
	if err == nil {
		t.Fatal("expected error when llm is nil")
	}
}

func readText(t *testing.T, ch <-chan llm.Chunk) string {
	t.Helper()
	var got string
	for chunk := range ch {
		if chunk.Err != nil {
			t.Fatalf("chunk error: %v", chunk.Err)
		}
		got += chunk.Text
	}
	return got
}
