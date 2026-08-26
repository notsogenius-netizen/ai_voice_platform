package roombot

import (
	"embed"
	"log"
	"os"
	"path/filepath"

	lksdk "github.com/livekit/server-sdk-go/v2"
)

//go:embed tones/verification.ogg
var verificationToneFS embed.FS

// publishVerificationTone publishes a short embedded Ogg Opus tone once.
func publishVerificationTone(room *lksdk.Room) {
	if room == nil || room.LocalParticipant == nil {
		return
	}

	tonePath, cleanup, err := materializeVerificationTone()
	if err != nil {
		log.Printf("roombot: prepare verification tone: %v", err)
		return
	}
	defer cleanup()

	track, err := lksdk.NewLocalFileTrack(tonePath, lksdk.ReaderTrackWithOnWriteComplete(func() {
		log.Printf("roombot: verification tone finished room=%s", room.Name())
	}))
	if err != nil {
		log.Printf("roombot: create tone track: %v", err)
		return
	}

	if _, err := room.LocalParticipant.PublishTrack(track, &lksdk.TrackPublicationOptions{
		Name: "verification-tone",
	}); err != nil {
		log.Printf("roombot: publish tone track: %v", err)
		return
	}
	log.Printf("roombot: publishing verification tone room=%s", room.Name())
}

func materializeVerificationTone() (string, func(), error) {
	raw, err := verificationToneFS.ReadFile("tones/verification.ogg")
	if err != nil {
		return "", nil, err
	}
	dir, err := os.MkdirTemp("", "voice-gateway-tone-*")
	if err != nil {
		return "", nil, err
	}
	path := filepath.Join(dir, "verification.ogg")
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		_ = os.RemoveAll(dir)
		return "", nil, err
	}
	return path, func() { _ = os.RemoveAll(dir) }, nil
}
