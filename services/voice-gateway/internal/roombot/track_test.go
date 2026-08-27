package roombot

import (
	"context"
	"errors"
	"testing"

	"github.com/sourabh/ai-voice-platform/services/voice-gateway/internal/stt"
)

type fakeSTTStream struct {
	transcripts chan stt.Transcript
	writeErr    error
	closed      bool
}

func (f *fakeSTTStream) WritePCM([]byte) error {
	if f.writeErr != nil {
		return f.writeErr
	}
	return nil
}

func (f *fakeSTTStream) Transcripts() <-chan stt.Transcript {
	return f.transcripts
}

func (f *fakeSTTStream) Close() error {
	if !f.closed {
		close(f.transcripts)
		f.closed = true
	}
	return nil
}

type fakeSTTClient struct {
	opened  chan stt.Session
	openErr error
}

func (c *fakeSTTClient) Open(_ context.Context, sess stt.Session) (stt.Stream, error) {
	if c.openErr != nil {
		return nil, c.openErr
	}
	if c.opened != nil {
		c.opened <- sess
	}
	return &fakeSTTStream{transcripts: make(chan stt.Transcript, 1)}, nil
}

type failingSTTClient struct{}

func (failingSTTClient) Open(context.Context, stt.Session) (stt.Stream, error) {
	return nil, errors.New("dial failed")
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
	if pipe := b.openSTTStream(t.Context(), stt.Session{}, "label", nil); pipe != nil {
		t.Fatal("expected nil pipe without STT client")
	}
}

func TestOpenSTTStreamOpenFailure(t *testing.T) {
	b := Bot{STT: failingSTTClient{}}
	if pipe := b.openSTTStream(t.Context(), stt.Session{}, "label", nil); pipe != nil {
		t.Fatal("expected nil pipe when STT open fails")
	}
}

func TestOpenSTTStreamWithFakeClient(t *testing.T) {
	opened := make(chan stt.Session, 1)
	b := Bot{STT: &fakeSTTClient{opened: opened}}

	sess := stt.Session{Room: "room-1", Participant: "browser-1", TrackID: "TR_1"}
	pipe := b.openSTTStream(t.Context(), sess, "label", nil)
	if pipe == nil {
		t.Fatal("expected STT pipe")
	}
	defer pipe.closeQuietly()

	got := <-opened
	if got != sess {
		t.Fatalf("session = %+v, want %+v", got, sess)
	}
	pipe.write([]byte{0, 1})
}

func TestSTTPipeWriteErrorStopsFurtherWrites(t *testing.T) {
	stream := &fakeSTTStream{
		transcripts: make(chan stt.Transcript),
		writeErr:    errors.New("write failed"),
	}
	pipe := newSTTPipe(stream, "label")

	pipe.write([]byte{1})
	if !pipe.dead.Load() {
		t.Fatal("expected pipe to stop after write error")
	}

	stream.writeErr = nil
	pipe.write([]byte{2})
}

func TestSTTPipeEndClosesStreamOnce(t *testing.T) {
	stream := &fakeSTTStream{transcripts: make(chan stt.Transcript, 1)}
	pipe := newSTTPipe(stream, "label")

	pipe.end(errors.New("boom"))
	pipe.end(errors.New("again"))
	if !stream.closed {
		t.Fatal("expected stream closed")
	}
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
