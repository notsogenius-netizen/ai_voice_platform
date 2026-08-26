package roombot

import (
	"context"
	"errors"
	"testing"

	"github.com/sourabh/ai-voice-platform/services/voice-gateway/internal/orchestrator"
	"github.com/sourabh/ai-voice-platform/services/voice-gateway/internal/stt"
)

type fakeOrchestrator struct {
	lastTurn orchestrator.Turn
	reply    orchestrator.Reply
	err      error
}

func (f *fakeOrchestrator) SendTurn(_ context.Context, turn orchestrator.Turn) (orchestrator.Reply, error) {
	f.lastTurn = turn
	return f.reply, f.err
}

func TestForwardFinalTranscriptLogsReply(t *testing.T) {
	orch := &fakeOrchestrator{reply: orchestrator.Reply{Text: "Hello there"}}
	tr := stt.Transcript{
		Session: stt.Session{Room: "room-1", Participant: "browser-1", TrackID: "TR_1"},
		Text:    "hi",
		IsFinal: true,
	}

	forwardFinalTranscript(context.Background(), orch, tr)

	if orch.lastTurn.SessionID != "room-1" {
		t.Fatalf("session = %q", orch.lastTurn.SessionID)
	}
	if orch.lastTurn.Text != "hi" || !orch.lastTurn.IsFinal {
		t.Fatalf("turn = %+v", orch.lastTurn)
	}
}

func TestForwardFinalTranscriptLogsError(t *testing.T) {
	orch := &fakeOrchestrator{err: errors.New("down")}
	tr := stt.Transcript{
		Session: stt.Session{Room: "room-1", Participant: "browser-1", TrackID: "TR_1"},
		Text:    "hi",
		IsFinal: true,
	}
	forwardFinalTranscript(context.Background(), orch, tr)
}
