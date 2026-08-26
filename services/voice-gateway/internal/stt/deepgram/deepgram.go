// Package deepgram implements streaming STT over the Deepgram listen WebSocket API.
package deepgram

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"github.com/sourabh/ai-voice-platform/services/voice-gateway/internal/stt"
)

const (
	defaultListenURL      = "wss://api.deepgram.com/v1/listen"
	defaultSampleRate     = 16000
	defaultConnectTimeout = 10 * time.Second
)

type dialContext func(ctx context.Context, url string, header http.Header) (*websocket.Conn, *http.Response, error)

// Config holds Deepgram streaming settings.
type Config struct {
	APIKey         string
	ListenURL      string
	SampleRate     int
	ConnectTimeout time.Duration
}

// Client opens Deepgram streaming sessions.
type Client struct {
	cfg  Config
	dial dialContext
}

// NewClient validates config and returns a Deepgram STT client.
func NewClient(cfg Config) (*Client, error) {
	if cfg.APIKey == "" {
		return nil, errors.New("deepgram: API key is required")
	}
	if cfg.ListenURL == "" {
		cfg.ListenURL = defaultListenURL
	}
	if cfg.SampleRate <= 0 {
		cfg.SampleRate = defaultSampleRate
	}
	if cfg.ConnectTimeout <= 0 {
		cfg.ConnectTimeout = defaultConnectTimeout
	}
	return &Client{
		cfg:  cfg,
		dial: websocket.DefaultDialer.DialContext,
	}, nil
}

// Open starts a streaming recognition session for the given LiveKit session.
func (c *Client) Open(ctx context.Context, sess stt.Session) (stt.Stream, error) {
	wsURL, err := c.listenURL()
	if err != nil {
		return nil, err
	}

	dialCtx, cancel := context.WithTimeout(ctx, c.cfg.ConnectTimeout)
	defer cancel()

	header := http.Header{}
	header.Set("Authorization", "Token "+c.cfg.APIKey)

	conn, resp, err := c.dial(dialCtx, wsURL, header)
	if err != nil {
		if resp != nil && resp.Body != nil {
			_ = resp.Body.Close()
		}
		return nil, fmt.Errorf("deepgram dial: %w", err)
	}
	if resp != nil && resp.Body != nil {
		_ = resp.Body.Close()
	}

	stream := newStream(conn, sess)
	go stream.readLoop(ctx)
	return stream, nil
}

func (c *Client) listenURL() (string, error) {
	u, err := url.Parse(c.cfg.ListenURL)
	if err != nil {
		return "", fmt.Errorf("deepgram listen url: %w", err)
	}
	q := u.Query()
	q.Set("encoding", "linear16")
	q.Set("sample_rate", fmt.Sprintf("%d", c.cfg.SampleRate))
	q.Set("channels", "1")
	q.Set("interim_results", "true")
	q.Set("punctuate", "true")
	u.RawQuery = q.Encode()
	return u.String(), nil
}

type stream struct {
	conn   *websocket.Conn
	sess   stt.Session
	events chan stt.Transcript
	done   chan struct{}
	once   sync.Once
	err    error
	write  sync.Mutex
}

func newStream(conn *websocket.Conn, sess stt.Session) *stream {
	return &stream{
		conn:   conn,
		sess:   sess,
		events: make(chan stt.Transcript, 16),
		done:   make(chan struct{}),
	}
}

func (s *stream) WritePCM(pcm []byte) error {
	s.write.Lock()
	defer s.write.Unlock()

	if err := s.closedErr(); err != nil {
		return err
	}
	if len(pcm) == 0 {
		return nil
	}
	if err := s.conn.WriteMessage(websocket.BinaryMessage, pcm); err != nil {
		s.fail(err)
		return err
	}
	return nil
}

func (s *stream) Transcripts() <-chan stt.Transcript {
	return s.events
}

func (s *stream) Close() error {
	s.once.Do(s.closeConn)
	return s.err
}

func (s *stream) closedErr() error {
	select {
	case <-s.done:
		return s.err
	default:
		return nil
	}
}

func (s *stream) closeConn() {
	s.write.Lock()
	defer s.write.Unlock()

	_ = s.conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"CloseStream"}`))
	_ = s.conn.WriteControl(
		websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
		time.Now().Add(time.Second),
	)
	close(s.done)
	_ = s.conn.Close()
	close(s.events)
}

func (s *stream) fail(err error) {
	s.once.Do(func() {
		s.err = err
		close(s.done)
		_ = s.conn.Close()
		close(s.events)
	})
}

func (s *stream) readLoop(ctx context.Context) {
	defer s.Close()

	for {
		if ctx.Err() != nil {
			return
		}

		_, data, err := s.conn.ReadMessage()
		if err != nil {
			s.recordReadErr(err)
			return
		}

		text, isFinal, ok := parseResult(data)
		if !ok {
			continue
		}

		if !s.emit(stt.Transcript{Session: s.sess, Text: text, IsFinal: isFinal}) {
			return
		}
	}
}

func (s *stream) recordReadErr(err error) {
	select {
	case <-s.done:
		return
	default:
	}
	if errors.Is(err, io.EOF) || websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
		return
	}
	s.err = fmt.Errorf("deepgram read: %w", err)
}

func (s *stream) emit(tr stt.Transcript) bool {
	select {
	case <-s.done:
		return false
	default:
	}

	defer func() {
		_ = recover()
	}()

	select {
	case s.events <- tr:
		return true
	case <-s.done:
		return false
	}
}

func parseResult(data []byte) (text string, isFinal bool, ok bool) {
	var msg resultMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return "", false, false
	}
	if msg.Type != "Results" {
		return "", false, false
	}
	text = msg.bestTranscript()
	if text == "" {
		return "", false, false
	}
	return text, msg.IsFinal, true
}

type resultMessage struct {
	Type    string `json:"type"`
	IsFinal bool   `json:"is_final"`
	Channel struct {
		Alternatives []struct {
			Transcript string `json:"transcript"`
		} `json:"alternatives"`
	} `json:"channel"`
}

func (m resultMessage) bestTranscript() string {
	if len(m.Channel.Alternatives) == 0 {
		return ""
	}
	return m.Channel.Alternatives[0].Transcript
}
