package httpclient

import (
	"bufio"
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

	"github.com/sourabh/ai-voice-platform/services/voice-gateway/internal/orchestrator"
)

const defaultRequestTimeout = 60 * time.Second

type roundTripper func(*http.Request) (*http.Response, error)

// Config holds ai-orchestrator HTTP client settings.
type Config struct {
	BaseURL        string
	RequestTimeout time.Duration
	HTTPClient     *http.Client
}

// Client posts turns to ai-orchestrator.
type Client struct {
	cfg    Config
	postFn roundTripper
}

// NewClient validates config and returns an orchestrator HTTP client.
func NewClient(cfg Config) (*Client, error) {
	if strings.TrimSpace(cfg.BaseURL) == "" {
		return nil, errors.New("orchestrator: base URL is required")
	}
	if cfg.RequestTimeout <= 0 {
		cfg.RequestTimeout = defaultRequestTimeout
	}
	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &Client{cfg: cfg, postFn: httpClient.Do}, nil
}

// SendTurn forwards one transcript turn and returns the full reply text.
func (c *Client) SendTurn(ctx context.Context, turn orchestrator.Turn) (orchestrator.Reply, error) {
	return c.StreamTurn(ctx, turn, nil)
}

// StreamTurn forwards one turn and invokes onChunk for each SSE text fragment.
func (c *Client) StreamTurn(
	ctx context.Context,
	turn orchestrator.Turn,
	onChunk orchestrator.ChunkHandler,
) (orchestrator.Reply, error) {
	body, err := json.Marshal(turnRequest{
		Text:    turn.Text,
		IsFinal: turn.IsFinal,
	})
	if err != nil {
		return orchestrator.Reply{}, fmt.Errorf("orchestrator encode request: %w", err)
	}

	reqCtx, cancel := context.WithTimeout(ctx, c.cfg.RequestTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(
		reqCtx,
		http.MethodPost,
		turnURL(c.cfg.BaseURL, turn.SessionID),
		bytes.NewReader(body),
	)
	if err != nil {
		return orchestrator.Reply{}, fmt.Errorf("orchestrator build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream, application/json")

	resp, err := c.postFn(req)
	if err != nil {
		return orchestrator.Reply{}, fmt.Errorf("orchestrator request: %w", err)
	}
	defer resp.Body.Close()

	return c.readResponse(resp, onChunk)
}

func (c *Client) readResponse(
	resp *http.Response,
	onChunk orchestrator.ChunkHandler,
) (orchestrator.Reply, error) {
	if resp.StatusCode == http.StatusAccepted {
		return orchestrator.Reply{Ignored: true}, nil
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		msg := readErrorBody(resp.Body)
		return orchestrator.Reply{}, fmt.Errorf("orchestrator status %d: %s", resp.StatusCode, msg)
	}

	if strings.Contains(resp.Header.Get("Content-Type"), "text/event-stream") {
		text, err := readSSEText(resp.Body, onChunk)
		if err != nil {
			return orchestrator.Reply{}, err
		}
		return orchestrator.Reply{Text: text}, nil
	}

	var ignored ignoredResponse
	if err := json.NewDecoder(resp.Body).Decode(&ignored); err == nil && ignored.Status == "ignored" {
		return orchestrator.Reply{Ignored: true}, nil
	}
	return orchestrator.Reply{}, nil
}

func turnURL(baseURL, sessionID string) string {
	return strings.TrimRight(baseURL, "/") + "/v1/sessions/" + url.PathEscape(sessionID) + "/turn"
}

func readErrorBody(r io.Reader) string {
	b, err := io.ReadAll(io.LimitReader(r, 4096))
	if err != nil || len(b) == 0 {
		return "request failed"
	}
	return strings.TrimSpace(string(b))
}

func readSSEText(body io.Reader, onChunk orchestrator.ChunkHandler) (string, error) {
	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	var reply strings.Builder
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if payload == "[DONE]" {
			break
		}
		text, err := parseSSEPayload(payload)
		if err != nil {
			return "", err
		}
		if text == "" {
			continue
		}
		reply.WriteString(text)
		if onChunk != nil {
			if err := onChunk(text); err != nil {
				return reply.String(), err
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("orchestrator read stream: %w", err)
	}
	return reply.String(), nil
}

func parseSSEPayload(payload string) (string, error) {
	var msg ssePayload
	if err := json.Unmarshal([]byte(payload), &msg); err != nil {
		return "", fmt.Errorf("orchestrator decode chunk: %w", err)
	}
	if msg.Error != "" {
		return "", errors.New(msg.Error)
	}
	return msg.Text, nil
}

type turnRequest struct {
	Text    string `json:"text"`
	IsFinal bool   `json:"is_final"`
}

type ignoredResponse struct {
	Status string `json:"status"`
}

type ssePayload struct {
	Text  string `json:"text"`
	Error string `json:"error"`
}
