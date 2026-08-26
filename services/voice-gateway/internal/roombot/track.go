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
	stt    stt.Stream
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
		if err := t.stt.Close(); err != nil {
			log.Printf("roombot: stt close: %v", err)
		}
	}
}

func (b Bot) startAudioTrack(
	ctx context.Context,
	roomName string,
	track *webrtc.TrackRemote,
	rp *lksdk.RemoteParticipant,
	tracks *trackSet,
) {
	label := fmt.Sprintf(
		"room=%s identity=%s track=%s",
		roomName,
		rp.Identity(),
		track.ID(),
	)

	trackCtx, cancel := context.WithCancel(ctx)
	sess := stt.Session{
		Room:        roomName,
		Participant: rp.Identity(),
		TrackID:     track.ID(),
	}

	sttStream, onPCM := b.openSTTStream(trackCtx, sess, label)
	pcmTrack, err := pcm.StartRemoteTrack(trackCtx, track, b.STTSampleRate, label, onPCM)
	if err != nil {
		log.Printf("roombot: pcm start %s: %v", label, err)
		cancel()
		if sttStream != nil {
			_ = sttStream.Close()
		}
		return
	}

	tracks.add(track.ID(), &audioTrack{
		cancel: cancel,
		pcm:    pcmTrack,
		stt:    sttStream,
	})
}

func (b Bot) openSTTStream(
	ctx context.Context,
	sess stt.Session,
	label string,
) (stt.Stream, func([]byte)) {
	if b.STT == nil {
		return nil, nil
	}

	stream, err := b.STT.Open(ctx, sess)
	if err != nil {
		log.Printf("roombot: stt open %s: %v", label, err)
		return nil, nil
	}

	go readTranscripts(ctx, stream)
	onPCM := func(pcmBytes []byte) {
		if err := stream.WritePCM(pcmBytes); err != nil {
			log.Printf("roombot: stt write %s: %v", label, err)
		}
	}
	log.Printf("roombot: stt stream opened %s", label)
	return stream, onPCM
}

func readTranscripts(ctx context.Context, stream stt.Stream) {
	for {
		select {
		case <-ctx.Done():
			return
		case tr, ok := <-stream.Transcripts():
			if !ok {
				return
			}
			logTranscript(tr)
		}
	}
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
