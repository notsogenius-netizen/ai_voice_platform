package openai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/sourabh/ai-voice-platform/services/ai-orchestrator/internal/llm"
)

const defaultRequestTimeout = 30 * time.Second

type roundTripper func(*http.Request) (*http.Response, error)

// Config holds OpenAI-compatible streaming settings.
type Config struct {
	APIKey         string
	BaseURL        string
	Model          string
	RequestTimeout time.Duration
	HTTPClient     *http.Client
}

// Client streams chat completions from an OpenAI-compatible endpoint.
type Client struct {
	cfg    Config
	postFn roundTripper
}

// NewClient validates config and returns a streaming LLM client.
func NewClient(cfg Config) (*Client, error) {
	if strings.TrimSpace(cfg.APIKey) == "" {
		return nil, errors.New("openai: API key is required")
	}
	if strings.TrimSpace(cfg.BaseURL) == "" {
		return nil, errors.New("openai: base URL is required")
	}
	if strings.TrimSpace(cfg.Model) == "" {
		return nil, errors.New("openai: model is required")
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

// Stream starts a streaming chat completion for the given messages.
func (c *Client) Stream(ctx context.Context, messages []llm.Message) (<-chan llm.Chunk, error) {
	body, err := json.Marshal(chatRequest{
		Model:    c.cfg.Model,
		Messages: toAPI(messages),
		Stream:   true,
	})
	if err != nil {
		return nil, fmt.Errorf("openai encode request: %w", err)
	}

	reqCtx, cancel := context.WithTimeout(ctx, c.cfg.RequestTimeout)
	req, err := http.NewRequestWithContext(
		reqCtx,
		http.MethodPost,
		chatCompletionsURL(c.cfg.BaseURL),
		bytes.NewReader(body),
	)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("openai build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.cfg.APIKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")

	resp, err := c.postFn(req)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("openai request: %w", err)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		msg := readErrorBody(resp.Body)
		_ = resp.Body.Close()
		cancel()
		return nil, fmt.Errorf("openai status %d: %s", resp.StatusCode, msg)
	}

	ch := make(chan llm.Chunk, 16)
	go c.pumpStream(ctx, cancel, resp.Body, ch)
	return ch, nil
}

func (c *Client) pumpStream(ctx context.Context, cancel context.CancelFunc, body io.ReadCloser, out chan<- llm.Chunk) {
	defer cancel()
	defer close(out)
	defer body.Close()

	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for scanner.Scan() {
		if err := ctx.Err(); err != nil {
			out <- llm.Chunk{Err: err}
			return
		}

		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, ":") {
			continue
		}
		if !strings.HasPrefix(line, "data:") {
			continue
		}

		payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if payload == "[DONE]" {
			return
		}

		text, err := parseChunkPayload(payload)
		if err != nil {
			out <- llm.Chunk{Err: err}
			return
		}
		if text == "" {
			continue
		}
		select {
		case <-ctx.Done():
			out <- llm.Chunk{Err: ctx.Err()}
			return
		case out <- llm.Chunk{Text: text}:
		}
	}

	if err := scanner.Err(); err != nil {
		out <- llm.Chunk{Err: fmt.Errorf("openai read stream: %w", err)}
	}
}

func parseChunkPayload(payload string) (string, error) {
	var msg chunkMessage
	if err := json.Unmarshal([]byte(payload), &msg); err != nil {
		return "", fmt.Errorf("openai decode chunk: %w", err)
	}
	if len(msg.Choices) == 0 {
		return "", nil
	}
	return msg.Choices[0].Delta.Content, nil
}

func chatCompletionsURL(baseURL string) string {
	return strings.TrimRight(baseURL, "/") + "/chat/completions"
}

func readErrorBody(r io.Reader) string {
	const limit = 4096
	b, err := io.ReadAll(io.LimitReader(r, limit))
	if err != nil || len(b) == 0 {
		return "request failed"
	}
	return strings.TrimSpace(string(b))
}

func toAPI(messages []llm.Message) []chatMessage {
	out := make([]chatMessage, len(messages))
	for i, msg := range messages {
		out[i] = chatMessage{Role: string(msg.Role), Content: msg.Content}
	}
	return out
}

type chatRequest struct {
	Model    string        `json:"model"`
	Messages []chatMessage `json:"messages"`
	Stream   bool          `json:"stream"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chunkMessage struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
	} `json:"choices"`
}
