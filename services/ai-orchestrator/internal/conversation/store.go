package conversation

import (
	"sync"

	"github.com/sourabh/ai-voice-platform/services/ai-orchestrator/internal/llm"
)

type store struct {
	mu       sync.Mutex
	sessions map[string][]llm.Message
}

func newStore() *store {
	return &store{sessions: make(map[string][]llm.Message)}
}

func (st *store) snapshot(sessionID, systemPrompt string) []llm.Message {
	st.mu.Lock()
	defer st.mu.Unlock()

	msgs, ok := st.sessions[sessionID]
	if !ok {
		msgs = []llm.Message{{Role: llm.RoleSystem, Content: systemPrompt}}
		st.sessions[sessionID] = msgs
	}

	out := make([]llm.Message, len(msgs))
	copy(out, msgs)
	return out
}

func (st *store) appendUser(sessionID, systemPrompt, text string) []llm.Message {
	st.mu.Lock()
	defer st.mu.Unlock()

	msgs, ok := st.sessions[sessionID]
	if !ok {
		msgs = []llm.Message{{Role: llm.RoleSystem, Content: systemPrompt}}
	}
	msgs = append(msgs, llm.Message{Role: llm.RoleUser, Content: text})
	st.sessions[sessionID] = msgs

	out := make([]llm.Message, len(msgs))
	copy(out, msgs)
	return out
}

func (st *store) appendAssistant(sessionID, text string) {
	st.mu.Lock()
	defer st.mu.Unlock()

	msgs := st.sessions[sessionID]
	msgs = append(msgs, llm.Message{Role: llm.RoleAssistant, Content: text})
	st.sessions[sessionID] = msgs
}
