package roombot

import (
	"strings"
	"unicode"
)

// sentenceBuffer accumulates streamed LLM text and emits complete sentences.
type sentenceBuffer struct {
	buf strings.Builder
}

// Push appends chunk and returns any newly completed sentences.
func (b *sentenceBuffer) Push(chunk string) []string {
	if chunk == "" {
		return nil
	}
	b.buf.WriteString(chunk)
	return b.extract(false)
}

// Flush returns any remaining buffered text as a final sentence.
func (b *sentenceBuffer) Flush() string {
	parts := b.extract(true)
	if len(parts) == 0 {
		return ""
	}
	return parts[0]
}

func (b *sentenceBuffer) extract(flush bool) []string {
	raw := b.buf.String()
	if raw == "" {
		return nil
	}

	var out []string
	start := 0
	for i := 0; i < len(raw); i++ {
		if !isSentenceEnd(raw[i]) {
			continue
		}
		j := i + 1
		for j < len(raw) && (raw[j] == '"' || raw[j] == '\'' || raw[j] == ')') {
			j++
		}
		if j < len(raw) && !unicode.IsSpace(rune(raw[j])) && !flush {
			continue
		}
		sentence := strings.TrimSpace(raw[start:j])
		if sentence != "" {
			out = append(out, sentence)
		}
		for j < len(raw) && unicode.IsSpace(rune(raw[j])) {
			j++
		}
		start = j
		i = j - 1
	}

	if flush {
		rest := strings.TrimSpace(raw[start:])
		if rest != "" {
			out = append(out, rest)
		}
		b.buf.Reset()
		return out
	}

	b.buf.Reset()
	b.buf.WriteString(raw[start:])
	return out
}

func isSentenceEnd(c byte) bool {
	return c == '.' || c == '!' || c == '?' || c == '\n'
}
