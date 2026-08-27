package roombot

import (
	"context"
	"errors"
	"testing"

	"github.com/sourabh/ai-voice-platform/services/voice-gateway/internal/orchestrator"
	"github.com/sourabh/ai-voice-platform/services/voice-gateway/internal/stt"
	"github.com/sourabh/ai-voice-platform/services/voice-gateway/internal/tts"
)

type fakeOrchestrator struct {
	lastTurn orchestrator.Turn
	chunks   []string
	reply    orchestrator.Reply
	err      error
}

func (f *fakeOrchestrator) SendTurn(ctx context.Context, turn orchestrator.Turn) (orchestrator.Reply, error) {
	return f.StreamTurn(ctx, turn, nil)
}

func (f *fakeOrchestrator) StreamTurn(
	_ context.Context,
	turn orchestrator.Turn,
	onChunk orchestrator.ChunkHandler,
) (orchestrator.Reply, error) {
	f.lastTurn = turn
	if f.err != nil {
		return orchestrator.Reply{}, f.err
	}
	for _, chunk := range f.chunks {
		if onChunk != nil {
			if err := onChunk(chunk); err != nil {
				return orchestrator.Reply{}, err
			}
		}
	}
	return f.reply, nil
}

type fakeTTS struct {
	texts []string
	err   error
}

func (f *fakeTTS) Synthesize(_ context.Context, req tts.Request) (tts.Audio, error) {
	f.texts = append(f.texts, req.Text)
	if f.err != nil {
		return tts.Audio{}, f.err
	}
	return tts.Audio{Ogg: []byte("OggS-fake")}, nil
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

func TestHandleFinalSpeaksStreamedSentences(t *testing.T) {
	orch := &fakeOrchestrator{
		chunks: []string{"Hello there. ", "How are you?"},
		reply:  orchestrator.Reply{Text: "Hello there. How are you?"},
	}
	synth := &fakeTTS{}
	pipeline := &turnPipeline{orch: orch, tts: synth}

	pipeline.handleFinal(context.Background(), stt.Transcript{
		Session: stt.Session{Room: "room-1", Participant: "browser-1", TrackID: "TR_1"},
		Text:    "hi",
		IsFinal: true,
	})

	if len(synth.texts) != 2 {
		t.Fatalf("texts = %#v", synth.texts)
	}
	if synth.texts[0] != "Hello there." || synth.texts[1] != "How are you?" {
		t.Fatalf("texts = %#v", synth.texts)
	}
}

func TestHandleFinalSpeaksFullReplyWithoutChunks(t *testing.T) {
	orch := &fakeOrchestrator{reply: orchestrator.Reply{Text: "One shot reply."}}
	synth := &fakeTTS{}
	pipeline := &turnPipeline{orch: orch, tts: synth}

	pipeline.handleFinal(context.Background(), stt.Transcript{
		Session: stt.Session{Room: "room-1"},
		Text:    "hi",
		IsFinal: true,
	})

	if len(synth.texts) != 1 || synth.texts[0] != "One shot reply." {
		t.Fatalf("texts = %#v", synth.texts)
	}
}
