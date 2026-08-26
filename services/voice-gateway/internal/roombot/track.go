package roombot

import (
	"context"
	"fmt"
	"log"
	"sync"

	lksdk "github.com/livekit/server-sdk-go/v2"
	"github.com/pion/webrtc/v4"

	"github.com/sourabh/ai-voice-platform/services/voice-gateway/internal/audio/pcm"
	"github.com/sourabh/ai-voice-platform/services/voice-gateway/internal/stt"
)

type audioTrack struct {
	cancel context.CancelFunc
	pcm    *pcm.RemoteTrack
	stt    *sttPipe
}

type trackSet struct {
	mu      sync.Mutex
	entries map[string]*audioTrack
}

func newTrackSet() *trackSet {
	return &trackSet{entries: make(map[string]*audioTrack)}
}

func (s *trackSet) add(trackID string, track *audioTrack) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.entries[trackID] = track
}

func (s *trackSet) remove(trackID string) {
	s.mu.Lock()
	track := s.entries[trackID]
	delete(s.entries, trackID)
	s.mu.Unlock()
	if track != nil {
		track.close()
	}
}

func (s *trackSet) closeAll() {
	s.mu.Lock()
	entries := s.entries
	s.entries = make(map[string]*audioTrack)
	s.mu.Unlock()

	for id, track := range entries {
		if track != nil {
			track.close()
		}
		log.Printf("roombot: closed audio track=%s", id)
	}
}

func (t *audioTrack) close() {
	if t == nil {
		return
	}
	if t.cancel != nil {
		t.cancel()
	}
	if t.pcm != nil {
		t.pcm.Close()
	}
	if t.stt != nil {
		t.stt.closeQuietly()
	}
}

func (b Bot) startAudioTrack(
	ctx context.Context,
	roomName string,
	track *webrtc.TrackRemote,
	rp *lksdk.RemoteParticipant,
	tracks *trackSet,
) {
	label, sess := trackSTTSession(roomName, rp, track)
	trackCtx, cancel := context.WithCancel(ctx)
	pipe := b.openSTTStream(trackCtx, sess, label)

	pcmTrack, err := pcm.StartRemoteTrack(trackCtx, track, b.STTSampleRate, label, pipe.writeFn())
	if err != nil {
		abortAudioTrack(cancel, pipe, label, err)
		return
	}

	tracks.add(track.ID(), &audioTrack{
		cancel: cancel,
		pcm:    pcmTrack,
		stt:    pipe,
	})
}

func trackSTTSession(
	roomName string,
	rp *lksdk.RemoteParticipant,
	track *webrtc.TrackRemote,
) (string, stt.Session) {
	label := fmt.Sprintf(
		"room=%s identity=%s track=%s",
		roomName,
		rp.Identity(),
		track.ID(),
	)
	return label, stt.Session{
		Room:        roomName,
		Participant: rp.Identity(),
		TrackID:     track.ID(),
	}
}

func abortAudioTrack(cancel context.CancelFunc, pipe *sttPipe, label string, err error) {
	log.Printf("roombot: pcm start %s: %v", label, err)
	cancel()
	if pipe != nil {
		pipe.closeQuietly()
	}
}

func (b Bot) openSTTStream(
	ctx context.Context,
	sess stt.Session,
	label string,
) *sttPipe {
	if b.STT == nil {
		return nil
	}

	stream, err := b.STT.Open(ctx, sess)
	if err != nil {
		log.Printf("roombot: stt open %s: %v", label, err)
		return nil
	}

	pipe := newSTTPipe(stream, label)
	go readTranscripts(ctx, pipe)
	log.Printf("roombot: stt stream opened %s", label)
	return pipe
}

func logTranscript(tr stt.Transcript) {
	kind := "partial"
	if tr.IsFinal {
		kind = "final"
	}
	log.Printf(
		"stt %s room=%s identity=%s track=%s text=%q",
		kind,
		tr.Session.Room,
		tr.Session.Participant,
		tr.Session.TrackID,
		tr.Text,
	)
}

func (b Bot) stopAudioTrack(trackID string, tracks *trackSet) {
	tracks.remove(trackID)
}
