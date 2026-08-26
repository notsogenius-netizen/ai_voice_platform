// Package roombot joins LiveKit rooms as the voice-gateway participant.
package roombot

import (
	"context"
	"fmt"
	"log"
	"sync"
	"sync/atomic"

	lksdk "github.com/livekit/server-sdk-go/v2"
	"github.com/pion/webrtc/v4"

	"github.com/sourabh/ai-voice-platform/services/voice-gateway/internal/orchestrator"
	"github.com/sourabh/ai-voice-platform/services/voice-gateway/internal/stt"
	"github.com/sourabh/ai-voice-platform/services/voice-gateway/internal/token"
)

// Bot connects to LiveKit rooms and observes remote audio tracks.
type Bot struct {
	LiveKitURL     string
	Minter         token.Minter
	STTSampleRate  int
	STT            stt.Client
	Orchestrator   orchestrator.Client
}

// Join connects as voice-gateway, logs participants/tracks, and blocks until ctx ends.
func (b Bot) Join(ctx context.Context, roomName string) error {
	jwt, err := b.Minter.Mint("voice-gateway", roomName, token.BotGrants())
	if err != nil {
		return fmt.Errorf("mint bot token: %w", err)
	}

	state := &joinState{
		ctx:    ctx,
		bot:    b,
		tracks: newTrackSet(),
	}
	room, err := lksdk.ConnectToRoomWithToken(
		b.LiveKitURL,
		jwt,
		state.callbacks(roomName),
		lksdk.WithAutoSubscribe(true),
	)
	if err != nil {
		return fmt.Errorf("connect bot to room %s: %w", roomName, err)
	}
	defer room.Disconnect()
	defer state.tracks.closeAll()
	state.room.Store(room)

	log.Printf("roombot: joined room=%s as voice-gateway", roomName)
	logExistingRemotes(room)

	<-ctx.Done()
	log.Printf("roombot: leaving room=%s", roomName)
	return nil
}

type joinState struct {
	room     atomic.Pointer[lksdk.Room]
	toneOnce sync.Once
	ctx      context.Context
	bot      Bot
	tracks   *trackSet
}

func (s *joinState) callbacks(roomName string) *lksdk.RoomCallback {
	return &lksdk.RoomCallback{
		OnParticipantConnected: func(rp *lksdk.RemoteParticipant) {
			log.Printf("roombot: participant connected room=%s identity=%s", roomName, rp.Identity())
		},
		OnParticipantDisconnected: func(rp *lksdk.RemoteParticipant) {
			log.Printf("roombot: participant disconnected room=%s identity=%s", roomName, rp.Identity())
		},
		ParticipantCallback: lksdk.ParticipantCallback{
			OnTrackPublished:    s.onTrackPublished(roomName),
			OnTrackSubscribed:   s.onTrackSubscribed(roomName),
			OnTrackUnsubscribed: s.onTrackUnsubscribed(roomName),
		},
	}
}

func (s *joinState) onTrackPublished(
	roomName string,
) func(*lksdk.RemoteTrackPublication, *lksdk.RemoteParticipant) {
	return func(pub *lksdk.RemoteTrackPublication, rp *lksdk.RemoteParticipant) {
		log.Printf(
			"roombot: track published room=%s identity=%s kind=%s name=%s",
			roomName,
			rp.Identity(),
			pub.Kind().String(),
			pub.Name(),
		)
	}
}

func (s *joinState) onTrackUnsubscribed(
	roomName string,
) func(*webrtc.TrackRemote, *lksdk.RemoteTrackPublication, *lksdk.RemoteParticipant) {
	return func(track *webrtc.TrackRemote, _ *lksdk.RemoteTrackPublication, rp *lksdk.RemoteParticipant) {
		if track.Kind() != webrtc.RTPCodecTypeAudio {
			return
		}
		log.Printf(
			"roombot: track unsubscribed room=%s identity=%s track=%s",
			roomName,
			rp.Identity(),
			track.ID(),
		)
		s.bot.stopAudioTrack(track.ID(), s.tracks)
	}
}

func (s *joinState) onTrackSubscribed(
	roomName string,
) func(*webrtc.TrackRemote, *lksdk.RemoteTrackPublication, *lksdk.RemoteParticipant) {
	return func(track *webrtc.TrackRemote, pub *lksdk.RemoteTrackPublication, rp *lksdk.RemoteParticipant) {
		log.Printf(
			"roombot: track subscribed room=%s identity=%s kind=%s name=%s sid=%s",
			roomName,
			rp.Identity(),
			track.Kind().String(),
			pub.Name(),
			track.ID(),
		)
		if track.Kind() != webrtc.RTPCodecTypeAudio {
			return
		}
		s.toneOnce.Do(func() {
			go s.publishToneWhenReady()
		})
		s.bot.startAudioTrack(s.ctx, roomName, track, rp, s.tracks)
	}
}

func (s *joinState) publishToneWhenReady() {
	for range 50 {
		if room := s.room.Load(); room != nil {
			publishVerificationTone(room)
			return
		}
		waitBriefly()
	}
	log.Printf("roombot: skipped verification tone; room pointer unavailable")
}

func logExistingRemotes(room *lksdk.Room) {
	for _, rp := range room.GetRemoteParticipants() {
		log.Printf("roombot: existing participant room=%s identity=%s", room.Name(), rp.Identity())
	}
}
