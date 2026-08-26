package deepgram_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"github.com/sourabh/ai-voice-platform/services/voice-gateway/internal/stt"
	"github.com/sourabh/ai-voice-platform/services/voice-gateway/internal/stt/deepgram"
)

func TestNewClientRequiresAPIKey(t *testing.T) {
	if _, err := deepgram.NewClient(deepgram.Config{}); err == nil {
		t.Fatal("expected error for missing API key")
	}
}

func TestNewClientDefaults(t *testing.T) {
	client, err := deepgram.NewClient(deepgram.Config{APIKey: "test-key"})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	sess := stt.Session{Room: "room-1", Participant: "browser-1", TrackID: "track-1"}
	stream, err := client.Open(context.Background(), sess)
	if err == nil {
		_ = stream.Close()
		t.Fatal("expected dial error without a server")
	}
}

func TestOpenStreamsTranscripts(t *testing.T) {
	listenURL := startListenServer(t, func(conn *websocket.Conn) {
		defer conn.Close()

		_, _, err := conn.ReadMessage()
		if err != nil {
			t.Errorf("read pcm: %v", err)
			return
		}

		sendResult(t, conn, "hello", false)
		sendResult(t, conn, "hello world", true)
	})

	client, err := deepgram.NewClient(deepgram.Config{
		APIKey:         "test-key",
		ListenURL:      listenURL,
		SampleRate:     16000,
		ConnectTimeout: time.Second,
	})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	sess := stt.Session{Room: "room-1", Participant: "browser-1", TrackID: "track-1"}
	stream, err := client.Open(context.Background(), sess)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer stream.Close()

	if err := stream.WritePCM([]byte{0, 1, 2, 3}); err != nil {
		t.Fatalf("WritePCM: %v", err)
	}

	first := readTranscript(t, stream.Transcripts())
	if first.Text != "hello" || first.IsFinal {
		t.Fatalf("first transcript = %+v", first)
	}
	if first.Session != sess {
		t.Fatalf("session = %+v, want %+v", first.Session, sess)
	}

	second := readTranscript(t, stream.Transcripts())
	if second.Text != "hello world" || !second.IsFinal {
		t.Fatalf("final transcript = %+v", second)
	}
}

func TestOpenConnectTimeout(t *testing.T) {
	slow := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		http.Error(w, "slow", http.StatusServiceUnavailable)
	}))
	t.Cleanup(slow.Close)

	client, err := deepgram.NewClient(deepgram.Config{
		APIKey:         "test-key",
		ListenURL:      wsURL(slow.URL),
		ConnectTimeout: 20 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	_, err = client.Open(context.Background(), stt.Session{Room: "room-1"})
	if err == nil {
		t.Fatal("expected connect timeout error")
	}
}

func TestOpenIgnoresMalformedAndEmptyResults(t *testing.T) {
	listenURL := startListenServer(t, func(conn *websocket.Conn) {
		defer conn.Close()

		_ = conn.WriteMessage(websocket.TextMessage, []byte(`not-json`))
		sendResult(t, conn, "", false)
		sendResult(t, conn, "ready", true)
	})

	client, err := deepgram.NewClient(deepgram.Config{
		APIKey:         "test-key",
		ListenURL:      listenURL,
		ConnectTimeout: time.Second,
	})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	stream, err := client.Open(context.Background(), stt.Session{Room: "room-1"})
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer stream.Close()

	tr := readTranscript(t, stream.Transcripts())
	if tr.Text != "ready" || !tr.IsFinal {
		t.Fatalf("transcript = %+v", tr)
	}
}

func TestWritePCMEmptyIsNoOp(t *testing.T) {
	listenURL := startListenServer(t, func(conn *websocket.Conn) {
		defer conn.Close()
		sendResult(t, conn, "done", true)
	})

	client, err := deepgram.NewClient(deepgram.Config{
		APIKey:         "test-key",
		ListenURL:      listenURL,
		ConnectTimeout: time.Second,
	})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	stream, err := client.Open(context.Background(), stt.Session{Room: "room-1"})
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer stream.Close()

	if err := stream.WritePCM(nil); err != nil {
		t.Fatalf("WritePCM nil: %v", err)
	}
	if err := stream.WritePCM([]byte{}); err != nil {
		t.Fatalf("WritePCM empty: %v", err)
	}
}

func TestWritePCMAfterCloseReturnsError(t *testing.T) {
	listenURL := startListenServer(t, func(conn *websocket.Conn) {
		defer conn.Close()
		time.Sleep(50 * time.Millisecond)
	})

	client, err := deepgram.NewClient(deepgram.Config{
		APIKey:         "test-key",
		ListenURL:      listenURL,
		ConnectTimeout: time.Second,
	})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	stream, err := client.Open(context.Background(), stt.Session{Room: "room-1"})
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if err := stream.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if err := stream.WritePCM([]byte{1, 2}); err == nil {
		t.Fatal("expected write error after close")
	}
}

func TestContextCancelCloseIsClean(t *testing.T) {
	listenURL := startListenServer(t, func(conn *websocket.Conn) {
		defer conn.Close()
		time.Sleep(200 * time.Millisecond)
	})

	client, err := deepgram.NewClient(deepgram.Config{
		APIKey:         "test-key",
		ListenURL:      listenURL,
		ConnectTimeout: time.Second,
	})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	stream, err := client.Open(ctx, stt.Session{Room: "room-1"})
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	cancel()
	time.Sleep(50 * time.Millisecond)
	if err := stream.Close(); err != nil {
		t.Fatalf("Close after cancel: %v", err)
	}
}

func TestCloseIsIdempotent(t *testing.T) {
	listenURL := startListenServer(t, func(conn *websocket.Conn) {
		defer conn.Close()
		time.Sleep(50 * time.Millisecond)
	})

	client, err := deepgram.NewClient(deepgram.Config{
		APIKey:         "test-key",
		ListenURL:      listenURL,
		ConnectTimeout: time.Second,
	})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	stream, err := client.Open(context.Background(), stt.Session{Room: "room-1"})
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	if err := stream.Close(); err != nil {
		t.Fatalf("first Close: %v", err)
	}
	if err := stream.Close(); err != nil {
		t.Fatalf("second Close: %v", err)
	}
}

func startListenServer(t *testing.T, handler func(*websocket.Conn)) string {
	t.Helper()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Token test-key" {
			t.Errorf("authorization = %q", r.Header.Get("Authorization"))
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		conn, err := websocket.Upgrade(w, r, nil, 1024, 1024)
		if err != nil {
			t.Errorf("upgrade: %v", err)
			return
		}
		handler(conn)
	}))
	t.Cleanup(srv.Close)
	return wsURL(srv.URL)
}

func wsURL(httpURL string) string {
	return "ws" + strings.TrimPrefix(httpURL, "http")
}

func sendResult(t *testing.T, conn *websocket.Conn, text string, isFinal bool) {
	t.Helper()

	msg := map[string]any{
		"type":     "Results",
		"is_final": isFinal,
		"channel": map[string]any{
			"alternatives": []map[string]string{
				{"transcript": text},
			},
		},
	}
	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("marshal result: %v", err)
	}
	if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
		t.Fatalf("write result: %v", err)
	}
}

func readTranscript(t *testing.T, ch <-chan stt.Transcript) stt.Transcript {
	t.Helper()

	select {
	case tr, ok := <-ch:
		if !ok {
			t.Fatal("transcript channel closed unexpectedly")
		}
		return tr
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for transcript")
		return stt.Transcript{}
	}
}
