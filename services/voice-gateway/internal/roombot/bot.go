// Package roombot joins LiveKit rooms as the voice-gateway participant.
package roombot

import (
	"context"
	"fmt"
	"log"

	lksdk "github.com/livekit/server-sdk-go/v2"
	"github.com/pion/webrtc/v4"

	"github.com/sourabh/ai-voice-platform/services/voice-gateway/internal/token"
)

// Bot connects to LiveKit rooms and observes remote audio tracks.
type Bot struct {
	LiveKitURL string
	Minter     token.Minter
}

// Join connects as voice-gateway, logs participants/tracks, and blocks until ctx ends.
func (b Bot) Join(ctx context.Context, roomName string) error {
	jwt, err := b.Minter.Mint("voice-gateway", roomName, token.BotGrants())
	if err != nil {
		return fmt.Errorf("mint bot token: %w", err)
	}

	room, err := lksdk.ConnectToRoomWithToken(
		b.LiveKitURL,
		jwt,
		newCallbacks(roomName),
		lksdk.WithAutoSubscribe(true),
	)
	if err != nil {
		return fmt.Errorf("connect bot to room %s: %w", roomName, err)
	}
	defer room.Disconnect()

	log.Printf("roombot: joined room=%s as voice-gateway", roomName)
	logExistingRemotes(room)

	<-ctx.Done()
	log.Printf("roombot: leaving room=%s", roomName)
	return nil
}

func newCallbacks(roomName string) *lksdk.RoomCallback {
	return &lksdk.RoomCallback{
		OnParticipantConnected: func(rp *lksdk.RemoteParticipant) {
			log.Printf("roombot: participant connected room=%s identity=%s", roomName, rp.Identity())
		},
		OnParticipantDisconnected: func(rp *lksdk.RemoteParticipant) {
			log.Printf("roombot: participant disconnected room=%s identity=%s", roomName, rp.Identity())
		},
		ParticipantCallback: lksdk.ParticipantCallback{
			OnTrackPublished: func(pub *lksdk.RemoteTrackPublication, rp *lksdk.RemoteParticipant) {
				log.Printf(
					"roombot: track published room=%s identity=%s kind=%s name=%s",
					roomName,
					rp.Identity(),
					pub.Kind().String(),
					pub.Name(),
				)
			},
			OnTrackSubscribed: onTrackSubscribed(roomName),
		},
	}
}

func onTrackSubscribed(roomName string) func(*webrtc.TrackRemote, *lksdk.RemoteTrackPublication, *lksdk.RemoteParticipant) {
	return func(track *webrtc.TrackRemote, pub *lksdk.RemoteTrackPublication, rp *lksdk.RemoteParticipant) {
		log.Printf(
			"roombot: track subscribed room=%s identity=%s kind=%s name=%s sid=%s",
			roomName,
			rp.Identity(),
			track.Kind().String(),
			pub.Name(),
			track.ID(),
		)
	}
}

func logExistingRemotes(room *lksdk.Room) {
	for _, rp := range room.GetRemoteParticipants() {
		log.Printf("roombot: existing participant room=%s identity=%s", room.Name(), rp.Identity())
	}
}
