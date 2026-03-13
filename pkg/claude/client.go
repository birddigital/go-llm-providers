// Package claude provides an Anthropic Claude API client with proxy support.
package claude

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

import "github.com/birddigital/go-llm-providers/pkg/providers"

const (
	// DefaultBaseURL is the official Anthropic API endpoint.
	DefaultBaseURL = "https://api.anthropic.com"

	// DefaultVersion is the API version header.
	DefaultVersion = "2023-06-01"

	// DefaultModel is the default Claude model.
	DefaultModel = "claude-3-5-sonnet-20241022"
)

// Client implements the Claude API client.
type Client struct {
	config     *providers.Config
	httpClient *http.Client
	version    string
}

// New creates a new Claude client.
func New(apiKey string, opts ...Option) (*Client, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("API key is required")
	}

	config := &providers.Config{
		APIKey:     apiKey,
		BaseURL:    DefaultBaseURL,
		Timeout:    60 * time.Second,
		MaxRetries: 3,
	}

	for _, opt := range opts {
		opt(config)
	}

	return &Client{
		config:     config,
		httpClient: &http.Client{Timeout: config.Timeout},
		version:    DefaultVersion,
	}, nil
}

// Option configures the Claude client.
type Option func(*providers.Config)

// WithBaseURL sets a custom base URL (for proxies).
func WithBaseURL(baseURL string) Option {
	return func(c *providers.Config) {
		c.BaseURL = baseURL
	}
}

// WithTimeout sets the request timeout.
func WithTimeout(timeout time.Duration) Option {
	return func(c *providers.Config) {
		c.Timeout = timeout
	}
}

// WithMaxRetries sets the maximum retry attempts.
func WithMaxRetries(retries int) Option {
	return func(c *providers.Config) {
		c.MaxRetries = retries
	}
}

// WithHeaders sets additional headers.
func WithHeaders(headers map[string]string) Option {
	return func(c *providers.Config) {
		c.Headers = headers
	}
}

// Complete generates a non-streaming completion.
func (c *Client) Complete(ctx context.Context, req *providers.CompletionRequest) (*providers.CompletionResponse, error) {
	start := time.Now()

	// Build Claude API request
	claudeReq := c.buildRequest(req)

	body, err := json.Marshal(claudeReq)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	// Create HTTP request
	url := c.config.BaseURL + "/v1/messages"
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	// Set headers
	c.setHeaders(httpReq)

	// Execute request
	httpResp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}
	defer httpResp.Body.Close()

	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if httpResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error: status %d: %s", httpResp.StatusCode, string(respBody))
	}

	// Parse response
	var claudeResp completionResponse
	if err := json.Unmarshal(respBody, &claudeResp); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	// Convert to unified response
	response := c.toCompletionResponse(&claudeResp, time.Since(start))

	return response, nil
}

// CompleteStream generates a streaming completion.
func (c *Client) CompleteStream(ctx context.Context, req *providers.CompletionRequest) (<-chan providers.CompletionChunk, error) {
	// Build Claude API request with streaming enabled
	claudeReq := c.buildRequest(req)
	claudeReq.Stream = true

	body, err := json.Marshal(claudeReq)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	// Create HTTP request
	url := c.config.BaseURL + "/v1/messages"
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	// Set headers
	c.setHeaders(httpReq)

	// Execute request
	httpResp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}

	if httpResp.StatusCode != http.StatusOK {
		defer httpResp.Body.Close()
		respBody, _ := io.ReadAll(httpResp.Body)
		return nil, fmt.Errorf("API error: status %d: %s", httpResp.StatusCode, string(respBody))
	}

	// Create streaming channel
	chunkChan := make(chan providers.CompletionChunk, 16)

	go c.streamResponse(httpResp.Body, chunkChan)

	return chunkChan, nil
}

// buildRequest converts unified request to Claude API format.
func (c *Client) buildRequest(req *providers.CompletionRequest) *completionRequest {
	claudeReq := &completionRequest{
		Model:     req.Model,
		MaxTokens: c.maxTokens(req),
		Stream:    false,
		Messages:  make([]claudeMessage, 0),
	}

	if req.SystemPrompt != "" {
		claudeReq.System = req.SystemPrompt
	}

	if req.Temperature > 0 {
		claudeReq.Temperature = &req.Temperature
	}

	if req.TopP > 0 {
		claudeReq.TopP = &req.TopP
	}

	if len(req.StopSequences) > 0 {
		claudeReq.StopSequences = req.StopSequences
	}

	// Convert messages
	for _, msg := range req.Messages {
		claudeMsg := claudeMessage{
			Role: msg.Role,
		}

		// Handle content blocks for multimodal
		if len(msg.ContentBlocks) > 0 {
			for _, block := range msg.ContentBlocks {
				claudeMsg.Content = append(claudeMsg.Content, map[string]interface{}{
					"type": block.Type,
					"text": block.Text,
				})
			}
		} else {
			claudeMsg.Content = []map[string]interface{}{
				{"type": "text", "text": msg.Content},
			}
		}

		claudeReq.Messages = append(claudeReq.Messages, claudeMsg)
	}

	return claudeReq
}

// maxTokens returns the max tokens or a default value.
func (c *Client) maxTokens(req *providers.CompletionRequest) int {
	if req.MaxTokens > 0 {
		return req.MaxTokens
	}
	return 4096 // Default max tokens
}

// setHeaders sets the required headers for Claude API.
func (c *Client) setHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.config.APIKey)
	req.Header.Set("anthropic-version", c.version)

	// Set any additional headers from config
	for k, v := range c.config.Headers {
		req.Header.Set(k, v)
	}
}

// toCompletionResponse converts Claude response to unified format.
func (c *Client) toCompletionResponse(resp *completionResponse, latency time.Duration) *providers.CompletionResponse {
	content := ""
	if len(resp.Content) > 0 {
		content = resp.Content[0].Text
	}

	usage := &providers.Usage{
		InputTokens:  resp.Usage.InputTokens,
		OutputTokens: resp.Usage.OutputTokens,
		TotalTokens:  resp.Usage.InputTokens + resp.Usage.OutputTokens,
	}

	return &providers.CompletionResponse{
		Content:    content,
		StopReason: resp.StopReason,
		Usage:      usage,
		ModelID:    resp.Model,
		RequestID:  resp.ID,
		Latency:    latency,
	}
}

// streamResponse handles streaming response from Claude API.
func (c *Client) streamResponse(body io.ReadCloser, chunkChan chan<- providers.CompletionChunk) {
	defer close(chunkChan)
	defer body.Close()

	decoder := json.NewDecoder(body)

	for {
		var event struct {
			Type string `json:"type"`
			Data json.RawMessage `json:"data,omitempty"`
			Delta json.RawMessage `json:"delta,omitempty"`
			Index int `json:"index,omitempty"`
		}

		if err := decoder.Decode(&event); err != nil {
			if err == io.EOF {
				return
			}
			chunkChan <- providers.CompletionChunk{Error: err}
			return
		}

		chunk := providers.CompletionChunk{}

		switch event.Type {
		case "content_block_delta":
			var delta struct {
				Delta struct {
					Type string `json:"type"`
					Text string `json:"text"`
				} `json:"delta"`
			}
			if err := json.Unmarshal(event.Data, &delta); err == nil {
				chunk.Content = delta.Delta.Text
				chunk.Delta = delta.Delta.Text
			}

		case "message_stop":
			chunk.Done = true

		case "error":
			chunk.Error = fmt.Errorf("stream error")
			chunk.Done = true
		}

		chunkChan <- chunk
	}
}

// ============================================================================
// Claude API Types
// ============================================================================

type completionRequest struct {
	Model         string          `json:"model"`
	MaxTokens     int             `json:"max_tokens"`
	Messages      []claudeMessage `json:"messages"`
	System        string          `json:"system,omitempty"`
	Temperature   *float64        `json:"temperature,omitempty"`
	TopP          *float64        `json:"top_p,omitempty"`
	StopSequences []string        `json:"stop_sequences,omitempty"`
	Stream        bool            `json:"stream,omitempty"`
	Tools         []claudeTool    `json:"tools,omitempty"`
}

type claudeMessage struct {
	Role    string                 `json:"role"`
	Content []map[string]interface{} `json:"content"`
}

type claudeTool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"input_schema"`
}

type completionResponse struct {
	ID           string        `json:"id"`
	Type         string        `json:"type"`
	Role         string        `json:"role"`
	Content      []contentBlock `json:"content"`
	Model        string        `json:"model"`
	StopReason   string        `json:"stop_reason"`
	Usage        usage         `json:"usage"`
}

type contentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type usage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}
