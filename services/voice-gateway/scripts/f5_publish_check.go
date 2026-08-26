//go:build ignore

// Manual F5 check: create a session and publish a microphone track as the browser.
// Check voice-gateway logs for roombot participant/track lines.
//
//	cd services/voice-gateway && go run ./scripts/f5_publish_check.go
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	lksdk "github.com/livekit/server-sdk-go/v2"
	"github.com/pion/webrtc/v4"
)

type sessionResp struct {
	Room       string `json:"room"`
	LiveKitURL string `json:"livekit_url"`
	Token      string `json:"token"`
	Identity   string `json:"identity"`
}

func main() {
	sess := mustCreateSession()
	fmt.Printf("session room=%s identity=%s\n", sess.Room, sess.Identity)

	room, err := lksdk.ConnectToRoomWithToken(
		sess.LiveKitURL,
		sess.Token,
		&lksdk.RoomCallback{},
		lksdk.WithAutoSubscribe(false),
	)
	if err != nil {
		panic(err)
	}
	defer room.Disconnect()

	track, err := lksdk.NewLocalTrack(webrtc.RTPCodecCapability{
		MimeType:  webrtc.MimeTypeOpus,
		ClockRate: 48000,
		Channels:  1,
	})
	if err != nil {
		panic(err)
	}
	if _, err := room.LocalParticipant.PublishTrack(track, &lksdk.TrackPublicationOptions{
		Name: "microphone",
	}); err != nil {
		panic(err)
	}
	fmt.Println("published microphone track; waiting for gateway subscribe logs")
	time.Sleep(3 * time.Second)
}

func mustCreateSession() sessionResp {
	res, err := http.Post(
		"http://127.0.0.1:8080/v1/sessions",
		"application/json",
		bytes.NewReader([]byte(`{"identity":"f5-browser"}`)),
	)
	if err != nil {
		panic(err)
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)
	if res.StatusCode != http.StatusCreated {
		panic(fmt.Sprintf("session %s: %s", res.Status, body))
	}
	var sess sessionResp
	if err := json.Unmarshal(body, &sess); err != nil {
		panic(err)
	}
	return sess
}
