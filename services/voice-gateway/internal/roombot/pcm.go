package roombot

import (
	"context"
	"fmt"
	"log"

	lksdk "github.com/livekit/server-sdk-go/v2"
	"github.com/pion/webrtc/v4"

	"github.com/sourabh/ai-voice-platform/services/voice-gateway/internal/audio/pcm"
)

func (b Bot) startPCMTrack(
	ctx context.Context,
	roomName string,
	track *webrtc.TrackRemote,
	rp *lksdk.RemoteParticipant,
	pipelines *pcm.TrackSet,
) {
	label := fmt.Sprintf(
		"room=%s identity=%s track=%s",
		roomName,
		rp.Identity(),
		track.ID(),
	)

	pcmTrack, err := pcm.StartRemoteTrack(ctx, track, b.STTSampleRate, label, nil)
	if err != nil {
		log.Printf("roombot: pcm start %s: %v", label, err)
		return
	}
	pipelines.Add(track.ID(), pcmTrack)
}

func (b Bot) stopPCMTrack(trackID string, pipelines *pcm.TrackSet) {
	pipelines.Remove(trackID)
}
