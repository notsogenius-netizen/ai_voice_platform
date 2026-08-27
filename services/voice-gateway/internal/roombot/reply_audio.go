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
	cancel, track, pub, room := p.snapshot()
	if cancel != nil {
		cancel()
	}
	if track != nil {
		_ = track.Close()
	}
	unpublish(room, pub)
}

func (p *replyPlayback) snapshot() (
	context.CancelFunc,
	*lksdk.LocalTrack,
	*lksdk.LocalTrackPublication,
	*lksdk.Room,
) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.cancel, p.track, p.pub, p.room
}

func unpublish(room *lksdk.Room, pub *lksdk.LocalTrackPublication) {
	if room == nil || room.LocalParticipant == nil || pub == nil {
		return
	}
	_ = room.LocalParticipant.UnpublishTrack(pub.SID())
}

// PlayOgg publishes one Ogg Opus clip and blocks until playout finishes or barge-in.
func (p *replyPlayback) PlayOgg(ctx context.Context, room *lksdk.Room, ogg []byte) error {
	if err := validatePlayRequest(room, ogg); err != nil {
		return err
	}
	playCtx, cancel := context.WithCancel(ctx)
	track, done, err := newOggTrack(ogg)
	if err != nil {
		cancel()
		return err
	}
	pub, err := publishReplyTrack(room, track)
	if err != nil {
		cancel()
		_ = track.Close()
		return err
	}
	p.arm(cancel, track, pub, room)
	defer p.clearIfCurrent(track)
	return waitPlayout(playoutWait{
		ctx:     ctx,
		playCtx: playCtx,
		room:    room,
		track:   track,
		pub:     pub,
		done:    done,
	})
}

func validatePlayRequest(room *lksdk.Room, ogg []byte) error {
	if room == nil || room.LocalParticipant == nil {
		return errors.New("reply playback: room unavailable")
	}
	if len(ogg) == 0 {
		return errors.New("reply playback: empty ogg")
	}
	return nil
}

func newOggTrack(ogg []byte) (*lksdk.LocalTrack, <-chan struct{}, error) {
	done := make(chan struct{})
	track, err := lksdk.NewLocalReaderTrack(
		io.NopCloser(bytes.NewReader(ogg)),
		webrtc.MimeTypeOpus,
		lksdk.ReaderTrackWithOnWriteComplete(closeOnce(done)),
	)
	return track, done, err
}

func closeOnce(done chan struct{}) func() {
	return func() {
		select {
		case <-done:
		default:
			close(done)
		}
	}
}

func publishReplyTrack(room *lksdk.Room, track *lksdk.LocalTrack) (*lksdk.LocalTrackPublication, error) {
	pub, err := room.LocalParticipant.PublishTrack(track, &lksdk.TrackPublicationOptions{
		Name: "assistant-reply",
	})
	if err != nil {
		return nil, err
	}
	log.Printf("roombot: publishing reply audio room=%s", room.Name())
	return pub, nil
}

func (p *replyPlayback) arm(
	cancel context.CancelFunc,
	track *lksdk.LocalTrack,
	pub *lksdk.LocalTrackPublication,
	room *lksdk.Room,
) {
	p.mu.Lock()
	defer p.mu.Unlock()
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
}

type playoutWait struct {
	ctx     context.Context
	playCtx context.Context
	room    *lksdk.Room
	track   *lksdk.LocalTrack
	pub     *lksdk.LocalTrackPublication
	done    <-chan struct{}
}

func waitPlayout(w playoutWait) error {
	select {
	case <-w.done:
		return nil
	case <-w.playCtx.Done():
		_ = w.track.Close()
		unpublish(w.room, w.pub)
		return playoutStopErr(w.ctx)
	}
}

func playoutStopErr(ctx context.Context) error {
	if errors.Is(ctx.Err(), context.Canceled) || errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return ctx.Err()
	}
	return errReplyInterrupted
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
