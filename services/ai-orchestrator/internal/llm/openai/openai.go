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
	if err := validateConfig(&cfg); err != nil {
		return nil, err
	}
	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &Client{cfg: cfg, postFn: httpClient.Do}, nil
}

func validateConfig(cfg *Config) error {
	if strings.TrimSpace(cfg.APIKey) == "" {
		return errors.New("openai: API key is required")
	}
	if strings.TrimSpace(cfg.BaseURL) == "" {
		return errors.New("openai: base URL is required")
	}
	if strings.TrimSpace(cfg.Model) == "" {
		return errors.New("openai: model is required")
	}
	if cfg.RequestTimeout <= 0 {
		cfg.RequestTimeout = defaultRequestTimeout
	}
	return nil
}

// Stream starts a streaming chat completion for the given messages.
func (c *Client) Stream(ctx context.Context, messages []llm.Message) (<-chan llm.Chunk, error) {
	req, cancel, err := c.newStreamRequest(ctx, messages)
	if err != nil {
		return nil, err
	}

	resp, err := c.postFn(req)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("openai request: %w", err)
	}
	if err := streamResponseError(resp); err != nil {
		cancel()
		_ = resp.Body.Close()
		return nil, err
	}

	ch := make(chan llm.Chunk, 16)
	go c.pumpStream(ctx, cancel, resp.Body, ch)
	return ch, nil
}

func (c *Client) newStreamRequest(ctx context.Context, messages []llm.Message) (*http.Request, context.CancelFunc, error) {
	body, err := json.Marshal(chatRequest{
		Model:    c.cfg.Model,
		Messages: toAPI(messages),
		Stream:   true,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("openai encode request: %w", err)
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
		return nil, nil, fmt.Errorf("openai build request: %w", err)
	}
	setStreamHeaders(req, c.cfg.APIKey)
	return req, cancel, nil
}

func setStreamHeaders(req *http.Request, apiKey string) {
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
}

func streamResponseError(resp *http.Response) error {
	if resp.StatusCode >= http.StatusOK && resp.StatusCode < http.StatusMultipleChoices {
		return nil
	}
	msg := readErrorBody(resp.Body)
	return fmt.Errorf("openai status %d: %s", resp.StatusCode, msg)
}

func (c *Client) pumpStream(ctx context.Context, cancel context.CancelFunc, body io.ReadCloser, out chan<- llm.Chunk) {
	defer cancel()
	defer close(out)
	defer body.Close()

	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		if done, err := processSSELine(ctx, scanner.Text(), out); done {
			if err != nil {
				out <- llm.Chunk{Err: err}
			}
			return
		}
	}
	if err := scanner.Err(); err != nil {
		out <- llm.Chunk{Err: fmt.Errorf("openai read stream: %w", err)}
	}
}

func processSSELine(ctx context.Context, raw string, out chan<- llm.Chunk) (done bool, err error) {
	if err := ctx.Err(); err != nil {
		return true, err
	}

	line := strings.TrimSpace(raw)
	if !isDataLine(line) {
		return false, nil
	}

	payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
	if payload == "[DONE]" {
		return true, nil
	}

	return emitParsedChunk(ctx, out, payload)
}

func emitParsedChunk(ctx context.Context, out chan<- llm.Chunk, payload string) (bool, error) {
	text, err := parseChunkPayload(payload)
	if err != nil || text == "" {
		return err != nil, err
	}
	return emitChunk(ctx, out, text)
}

func isDataLine(line string) bool {
	return line != "" && !strings.HasPrefix(line, ":") && strings.HasPrefix(line, "data:")
}

func emitChunk(ctx context.Context, out chan<- llm.Chunk, text string) (bool, error) {
	select {
	case <-ctx.Done():
		return true, ctx.Err()
	case out <- llm.Chunk{Text: text}:
		return false, nil
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
