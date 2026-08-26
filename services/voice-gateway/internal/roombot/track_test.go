package roombot

import (
	"context"
	"testing"

	"github.com/sourabh/ai-voice-platform/services/voice-gateway/internal/stt"
)

type fakeSTTStream struct {
	transcripts chan stt.Transcript
}

func (f *fakeSTTStream) WritePCM([]byte) error              { return nil }
func (f *fakeSTTStream) Transcripts() <-chan stt.Transcript { return f.transcripts }
func (f *fakeSTTStream) Close() error {
	close(f.transcripts)
	return nil
}

type fakeSTTClient struct {
	opened chan stt.Session
}

func (c *fakeSTTClient) Open(_ context.Context, sess stt.Session) (stt.Stream, error) {
	if c.opened != nil {
		c.opened <- sess
	}
	return &fakeSTTStream{transcripts: make(chan stt.Transcript, 1)}, nil
}

func TestLogTranscriptFinal(t *testing.T) {
	tr := stt.Transcript{
		Session: stt.Session{Room: "room-1", Participant: "browser-1", TrackID: "TR_1"},
		Text:    "hello",
		IsFinal: true,
	}
	logTranscript(tr)
}

func TestOpenSTTStreamWithoutClient(t *testing.T) {
	b := Bot{}
	stream, onPCM := b.openSTTStream(t.Context(), stt.Session{}, "label")
	if stream != nil || onPCM != nil {
		t.Fatalf("expected nil stream and handler without STT client")
	}
}

func TestOpenSTTStreamWithFakeClient(t *testing.T) {
	opened := make(chan stt.Session, 1)
	b := Bot{STT: &fakeSTTClient{opened: opened}}

	sess := stt.Session{Room: "room-1", Participant: "browser-1", TrackID: "TR_1"}
	stream, onPCM := b.openSTTStream(t.Context(), sess, "label")
	if stream == nil || onPCM == nil {
		t.Fatal("expected STT stream and PCM handler")
	}
	defer stream.Close()

	got := <-opened
	if got != sess {
		t.Fatalf("session = %+v, want %+v", got, sess)
	}
	onPCM([]byte{0, 1})
}

func TestTrackSetCloseAll(t *testing.T) {
	set := newTrackSet()
	cancelled := false
	set.add("TR_1", &audioTrack{
		cancel: func() { cancelled = true },
	})
	set.closeAll()
	if !cancelled {
		t.Fatal("expected track cancel on closeAll")
	}
}
