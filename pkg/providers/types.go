// Package providers provides unified types and interfaces for LLM providers.
package providers

import (
	"context"
	"time"
)

// Provider represents an LLM provider that can generate completions.
type Provider interface {
	// Name returns the provider name (e.g., "claude", "openai").
	Name() string

	// Complete generates a non-streaming completion for the given messages.
	Complete(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error)

	// CompleteStream generates a streaming completion.
	CompleteStream(ctx context.Context, req *CompletionRequest) (<-chan CompletionChunk, error)
}

// CompletionRequest is a unified request structure for all providers.
type CompletionRequest struct {
	// Messages is the conversation history.
	Messages []Message `json:"messages"`

	// Model specifies which model to use (e.g., "claude-3-5-sonnet-20241022").
	Model string `json:"model"`

	// MaxTokens limits the response length.
	MaxTokens int `json:"max_tokens,omitempty"`

	// Temperature controls randomness (0.0 = deterministic, 1.0 = random).
	Temperature float64 `json:"temperature,omitempty"`

	// TopP controls nucleus sampling.
	TopP float64 `json:"top_p,omitempty"`

	// StopSequences are strings that will stop generation.
	StopSequences []string `json:"stop_sequences,omitempty"`

	// Tools specifies available tools for function calling.
	Tools []Tool `json:"tools,omitempty"`

	// SystemPrompt is the system message (provider-specific handling).
	SystemPrompt string `json:"system,omitempty"`

	// Metadata for tracking and logging.
	Metadata map[string]string `json:"meta,omitempty"`
}

// CompletionResponse is the unified response structure.
type CompletionResponse struct {
	// Content is the generated text.
	Content string `json:"content"`

	// StopReason indicates why generation stopped.
	StopReason string `json:"stop_reason"`

	// Usage contains token usage information.
	Usage *Usage `json:"usage,omitempty"`

	// ToolCalls contains any tool calls made by the model.
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`

	// ModelID is the actual model that processed the request.
	ModelID string `json:"model"`

	// RequestID for tracing and debugging.
	RequestID string `json:"request_id"`

	// Latency is how long the request took.
	Latency time.Duration `json:"latency_ms"`
}

// CompletionChunk represents a streaming response chunk.
type CompletionChunk struct {
	// Content is the partial text content.
	Content string `json:"content"`

	// Delta is the incremental change.
	Delta string `json:"delta"`

	// Done indicates the stream is complete.
	Done bool `json:"done"`

	// ToolCalls contains partial tool call data.
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`

	// Error contains any stream error.
	Error error `json:"error,omitempty"`
}

// Message represents a single message in the conversation.
type Message struct {
	Role    string `json:"role"`    // "system", "user", "assistant", "tool"
	Content string `json:"content"` // Text content
	// ToolUseID and ToolCallID for tool responses
	ToolUseID    string         `json:"tool_use_id,omitempty"`
	ToolCallID   string         `json:"tool_call_id,omitempty"`
	ToolCalls    []ToolCall     `json:"tool_calls,omitempty"`
	ContentBlocks []ContentBlock `json:"content_blocks,omitempty"`
}

// ContentBlock represents structured content (for Claude multimodal).
type ContentBlock struct {
	Type        string       `json:"type"` // "text", "image", "tool_use", "tool_result"
	Text        string       `json:"text,omitempty"`
	Source      *ImageSource `json:"source,omitempty"`
	ToolUse     *ToolUse     `json:"tool_use,omitempty"`
	ToolResult  *ToolResult  `json:"tool_result,omitempty"`
}

// ImageSource describes an image input.
type ImageSource struct {
	Type      string `json:"type"`      // "base64"
	MediaType string `json:"media_type"` // "image/jpeg", "image/png"
	Data      string `json:"data"`      // base64 data
}

// ToolUse represents a tool use call from the assistant.
type ToolUse struct {
	ID    string                 `json:"id"`
	Name  string                 `json:"name"`
	Input map[string]interface{} `json:"input"`
}

// ToolResult represents the result of a tool execution.
type ToolResult struct {
	ToolUseID string `json:"tool_use_id"`
	Content   string `json:"content"`
	IsError   bool   `json:"is_error,omitempty"`
}

// Tool represents a function available for the model to call.
type Tool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"input_schema"`
}

// ToolCall represents a tool call made by the model.
type ToolCall struct {
	ID    string                 `json:"id"`
	Name  string                 `json:"name"`
	Input map[string]interface{} `json:"input"`
}

// Usage contains token usage information.
type Usage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
	TotalTokens  int `json:"total_tokens"`
}

// Config holds provider configuration.
type Config struct {
	// APIKey is the authentication key.
	APIKey string `json:"api_key"`

	// BaseURL is the API endpoint (for proxies/custom deployments).
	BaseURL string `json:"base_url"`

	// Timeout for requests.
	Timeout time.Duration `json:"timeout"`

	// MaxRetries for transient failures.
	MaxRetries int `json:"max_retries"`

	// Additional headers for requests.
	Headers map[string]string `json:"headers,omitempty"`
}

// ProviderType identifies the LLM provider.
type ProviderType string

const (
	ProviderClaude ProviderType = "claude"
	ProviderOpenAI ProviderType = "openai"
	ProviderCustom ProviderType = "custom"
)

// Role constants for messages.
const (
	RoleSystem    string = "system"
	RoleUser      string = "user"
	RoleAssistant string = "assistant"
	RoleTool      string = "tool"
)
