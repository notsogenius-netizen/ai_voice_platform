package session_test

import (
	"strings"
	"testing"
	"time"

	"github.com/sourabh/ai-voice-platform/services/voice-gateway/internal/session"
	"github.com/sourabh/ai-voice-platform/services/voice-gateway/internal/token"
)

func TestCreateMintsToken(t *testing.T) {
	svc := session.Service{
		LiveKitURL: "ws://127.0.0.1:7880",
		Minter: token.Minter{
			APIKey:    "devkey",
			APISecret: "devsecret_ai_voice_platform_local_only",
			ValidFor:  time.Hour,
		},
	}

	res, err := svc.Create(session.Request{Identity: "alice"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if res.Identity != "alice" {
		t.Fatalf("identity = %q", res.Identity)
	}
	if !strings.HasPrefix(res.Room, "sess_") {
		t.Fatalf("room = %q", res.Room)
	}
	if res.LiveKitURL != "ws://127.0.0.1:7880" {
		t.Fatalf("url = %q", res.LiveKitURL)
	}
	if res.Token == "" {
		t.Fatal("expected token")
	}
}
