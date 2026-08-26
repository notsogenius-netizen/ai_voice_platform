package llm_test

import (
	"context"
	"testing"

	"github.com/sourabh/ai-voice-platform/services/ai-orchestrator/internal/llm"
)

type fakeClient struct {
	chunks []llm.Chunk
}

func (f *fakeClient) Stream(_ context.Context, _ []llm.Message) (<-chan llm.Chunk, error) {
	ch := make(chan llm.Chunk, len(f.chunks))
	for _, chunk := range f.chunks {
		ch <- chunk
	}
	close(ch)
	return ch, nil
}

func TestFakeClientStreamsChunks(t *testing.T) {
	client := &fakeClient{chunks: []llm.Chunk{{Text: "hi"}, {Text: " there"}}}
	ch, err := client.Stream(context.Background(), nil)
	if err != nil {
		t.Fatalf("Stream: %v", err)
	}

	var got string
	for chunk := range ch {
		if chunk.Err != nil {
			t.Fatalf("chunk error: %v", chunk.Err)
		}
		got += chunk.Text
	}
	if got != "hi there" {
		t.Fatalf("got %q, want %q", got, "hi there")
	}
}
