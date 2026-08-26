package token_test

import (
	"testing"
	"time"

	"github.com/livekit/protocol/auth"

	"github.com/sourabh/ai-voice-platform/services/voice-gateway/internal/token"
)

func TestMintRequiresIdentityAndRoom(t *testing.T) {
	m := token.Minter{APIKey: "devkey", APISecret: "devsecret_ai_voice_platform_local_only"}
	if _, err := m.Mint("", "room", token.BrowserGrants()); err == nil {
		t.Fatal("expected identity error")
	}
	if _, err := m.Mint("user", "", token.BrowserGrants()); err == nil {
		t.Fatal("expected room error")
	}
}

func TestMintClaims(t *testing.T) {
	m := token.Minter{
		APIKey:    "devkey",
		APISecret: "devsecret_ai_voice_platform_local_only",
		ValidFor:  time.Hour,
	}
	raw, err := m.Mint("browser-user", "sess_test", token.BrowserGrants())
	if err != nil {
		t.Fatalf("Mint: %v", err)
	}

	verifier, err := auth.ParseAPIToken(raw)
	if err != nil {
		t.Fatalf("ParseAPIToken: %v", err)
	}
	_, grants, err := verifier.Verify(m.APISecret)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if grants.Identity != "browser-user" {
		t.Fatalf("identity = %q", grants.Identity)
	}
	if grants.Video == nil || grants.Video.Room != "sess_test" {
		t.Fatalf("video grant room = %#v", grants.Video)
	}
	if !grants.Video.RoomJoin {
		t.Fatal("expected RoomJoin")
	}
}
