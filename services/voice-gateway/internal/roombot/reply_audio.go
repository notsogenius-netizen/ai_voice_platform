package roombot

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log"
	"sync"

	lksdk "github.com/livekit/server-sdk-go/v2"
	"github.com/pion/webrtc/v4"
)

var errReplyInterrupted = errors.New("reply playback interrupted")

// replyPlayback publishes assistant Ogg Opus into a LiveKit room and supports barge-in.
type replyPlayback struct {
	mu      sync.Mutex
	cancel  context.CancelFunc
	track   *lksdk.LocalTrack
	pub     *lksdk.LocalTrackPublication
	room    *lksdk.Room
	playing bool
}

func newReplyPlayback(_ int) *replyPlayback {
	return &replyPlayback{}
}

func (p *replyPlayback) Playing() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.playing
}

// Interrupt stops current playback (barge-in). Safe when idle.
func (p *replyPlayback) Interrupt() {
	p.mu.Lock()
	cancel := p.cancel
	track := p.track
	pub := p.pub
	room := p.room
	p.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	if track != nil {
		_ = track.Close()
	}
	if room != nil && room.LocalParticipant != nil && pub != nil {
		_ = room.LocalParticipant.UnpublishTrack(pub.SID())
	}
}

// PlayOgg publishes one Ogg Opus clip and blocks until playout finishes or barge-in.
func (p *replyPlayback) PlayOgg(ctx context.Context, room *lksdk.Room, ogg []byte) error {
	if room == nil || room.LocalParticipant == nil {
		return errors.New("reply playback: room unavailable")
	}
	if len(ogg) == 0 {
		return errors.New("reply playback: empty ogg")
	}

	playCtx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})
	track, err := lksdk.NewLocalReaderTrack(
		io.NopCloser(bytes.NewReader(ogg)),
		webrtc.MimeTypeOpus,
		lksdk.ReaderTrackWithOnWriteComplete(func() {
			select {
			case <-done:
			default:
				close(done)
			}
		}),
	)
	if err != nil {
		cancel()
		return err
	}

	pub, err := room.LocalParticipant.PublishTrack(track, &lksdk.TrackPublicationOptions{
		Name: "assistant-reply",
	})
	if err != nil {
		cancel()
		_ = track.Close()
		return err
	}
	log.Printf("roombot: publishing reply audio room=%s", room.Name())

	p.mu.Lock()
	if p.cancel != nil {
		p.cancel()
	}
	if p.track != nil {
		_ = p.track.Close()
	}
	p.cancel = cancel
	p.track = track
	p.pub = pub
	p.room = room
	p.playing = true
	p.mu.Unlock()

	defer p.clearIfCurrent(track)

	select {
	case <-done:
		return nil
	case <-playCtx.Done():
		_ = track.Close()
		if pub != nil {
			_ = room.LocalParticipant.UnpublishTrack(pub.SID())
		}
		if errors.Is(ctx.Err(), context.Canceled) || errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return ctx.Err()
		}
		return errReplyInterrupted
	}
}

func (p *replyPlayback) clearIfCurrent(track *lksdk.LocalTrack) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.track != track {
		return
	}
	if p.cancel != nil {
		p.cancel()
	}
	p.cancel = nil
	p.track = nil
	p.pub = nil
	p.room = nil
	p.playing = false
}
