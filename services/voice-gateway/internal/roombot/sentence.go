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
	out, start := splitSentences(raw, flush)
	b.replaceRemainder(raw[start:], flush)
	return out
}

func splitSentences(raw string, flush bool) ([]string, int) {
	var out []string
	start := 0
	for i := 0; i < len(raw); i++ {
		end, ok := sentenceBoundary(raw, i, flush)
		if !ok {
			continue
		}
		if sentence := strings.TrimSpace(raw[start:end]); sentence != "" {
			out = append(out, sentence)
		}
		start = skipSpaces(raw, end)
		i = start - 1
	}
	if flush {
		if rest := strings.TrimSpace(raw[start:]); rest != "" {
			out = append(out, rest)
		}
		return out, len(raw)
	}
	return out, start
}

func sentenceBoundary(raw string, i int, flush bool) (int, bool) {
	if !isSentenceEnd(raw[i]) {
		return 0, false
	}
	j := skipClosers(raw, i+1)
	if j < len(raw) && !unicode.IsSpace(rune(raw[j])) && !flush {
		return 0, false
	}
	return j, true
}

func skipClosers(raw string, j int) int {
	for j < len(raw) && (raw[j] == '"' || raw[j] == '\'' || raw[j] == ')') {
		j++
	}
	return j
}

func skipSpaces(raw string, j int) int {
	for j < len(raw) && unicode.IsSpace(rune(raw[j])) {
		j++
	}
	return j
}

func (b *sentenceBuffer) replaceRemainder(rest string, flush bool) {
	b.buf.Reset()
	if flush {
		return
	}
	b.buf.WriteString(rest)
}

func isSentenceEnd(c byte) bool {
	return c == '.' || c == '!' || c == '?' || c == '\n'
}
