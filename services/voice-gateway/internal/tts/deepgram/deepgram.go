package deepgram

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/sourabh/ai-voice-platform/services/voice-gateway/internal/tts"
)

const (
	defaultSpeakURL    = "https://api.deepgram.com/v1/speak"
	defaultModel       = "aura-2-thalia-en"
	defaultHTTPTimeout = 30 * time.Second
)

type roundTripper func(*http.Request) (*http.Response, error)

// Config holds Deepgram Speak settings.
type Config struct {
	APIKey     string
	SpeakURL   string
	Model      string
	HTTPClient *http.Client
}

// Client synthesizes speech via Deepgram Speak.
type Client struct {
	cfg    Config
	postFn roundTripper
}

// NewClient validates config and returns a Deepgram TTS client.
func NewClient(cfg Config) (*Client, error) {
	if strings.TrimSpace(cfg.APIKey) == "" {
		return nil, errors.New("deepgram tts: API key is required")
	}
	if cfg.SpeakURL == "" {
		cfg.SpeakURL = defaultSpeakURL
	}
	if cfg.Model == "" {
		cfg.Model = defaultModel
	}
	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: defaultHTTPTimeout}
	}
	return &Client{cfg: cfg, postFn: httpClient.Do}, nil
}

// Synthesize converts text to Ogg Opus via Deepgram Speak.
func (c *Client) Synthesize(ctx context.Context, req tts.Request) (tts.Audio, error) {
	text := strings.TrimSpace(req.Text)
	if text == "" {
		return tts.Audio{}, errors.New("deepgram tts: text is required")
	}

	body, err := json.Marshal(speakRequest{Text: text})
	if err != nil {
		return tts.Audio{}, fmt.Errorf("deepgram tts encode: %w", err)
	}

	speakURL, err := c.speakURL()
	if err != nil {
		return tts.Audio{}, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, speakURL, bytes.NewReader(body))
	if err != nil {
		return tts.Audio{}, fmt.Errorf("deepgram tts build request: %w", err)
	}
	httpReq.Header.Set("Authorization", "Token "+c.cfg.APIKey)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "audio/ogg")

	resp, err := c.postFn(httpReq)
	if err != nil {
		return tts.Audio{}, fmt.Errorf("deepgram tts request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		msg := readErrorBody(resp.Body)
		return tts.Audio{}, fmt.Errorf("deepgram tts status %d: %s", resp.StatusCode, msg)
	}

	ogg, err := io.ReadAll(resp.Body)
	if err != nil {
		return tts.Audio{}, fmt.Errorf("deepgram tts read body: %w", err)
	}
	if len(ogg) == 0 {
		return tts.Audio{}, errors.New("deepgram tts: empty audio body")
	}

	return tts.Audio{Ogg: ogg}, nil
}

func (c *Client) speakURL() (string, error) {
	u, err := url.Parse(c.cfg.SpeakURL)
	if err != nil {
		return "", fmt.Errorf("deepgram tts speak url: %w", err)
	}
	q := u.Query()
	q.Set("model", c.cfg.Model)
	q.Set("encoding", "opus")
	q.Set("container", "ogg")
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func readErrorBody(r io.Reader) string {
	b, err := io.ReadAll(io.LimitReader(r, 4096))
	if err != nil || len(b) == 0 {
		return "request failed"
	}
	return strings.TrimSpace(string(b))
}

type speakRequest struct {
	Text string `json:"text"`
}
