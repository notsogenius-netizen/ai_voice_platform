package session_test

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/sourabh/ai-voice-platform/services/voice-gateway/internal/session"
	"github.com/sourabh/ai-voice-platform/services/voice-gateway/internal/token"
)

type fakeJoiner struct {
	mu    sync.Mutex
	rooms []string
	done  chan struct{}
}

func (f *fakeJoiner) Join(_ context.Context, roomName string) error {
	f.mu.Lock()
	f.rooms = append(f.rooms, roomName)
	f.mu.Unlock()
	close(f.done)
	return nil
}

func TestCreateMintsTokenAndStartsBot(t *testing.T) {
	joiner := &fakeJoiner{done: make(chan struct{})}
	svc := session.Service{
		LiveKitURL: "ws://127.0.0.1:7880",
		Minter: token.Minter{
			APIKey:    "devkey",
			APISecret: "devsecret_ai_voice_platform_local_only",
			ValidFor:  time.Hour,
		},
		RootCtx: context.Background(),
		Bot:     joiner,
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

	select {
	case <-joiner.done:
	case <-time.After(2 * time.Second):
		t.Fatal("bot join was not started")
	}

	joiner.mu.Lock()
	defer joiner.mu.Unlock()
	if len(joiner.rooms) != 1 || joiner.rooms[0] != res.Room {
		t.Fatalf("joined rooms = %#v, want [%s]", joiner.rooms, res.Room)
	}
}
