package roombot

import "testing"

func TestSentenceBufferPushEmitsOnBoundary(t *testing.T) {
	var buf sentenceBuffer
	parts := buf.Push("Hello there. How")
	if len(parts) != 1 || parts[0] != "Hello there." {
		t.Fatalf("parts = %#v", parts)
	}
	parts = buf.Push(" are you?")
	if len(parts) != 1 || parts[0] != "How are you?" {
		t.Fatalf("parts = %#v", parts)
	}
	if left := buf.Flush(); left != "" {
		t.Fatalf("flush = %q", left)
	}
}

func TestSentenceBufferFlushRemainder(t *testing.T) {
	var buf sentenceBuffer
	_ = buf.Push("Almost done")
	if got := buf.Flush(); got != "Almost done" {
		t.Fatalf("flush = %q", got)
	}
}

func TestSentenceBufferIgnoresDecimal(t *testing.T) {
	var buf sentenceBuffer
	parts := buf.Push("It costs 3.5 dollars today.")
	if len(parts) != 1 || parts[0] != "It costs 3.5 dollars today." {
		t.Fatalf("parts = %#v", parts)
	}
}
