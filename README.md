# go-llm-providers

Unified Go interface for multiple LLM providers with proxy support.

**Features:**
- Single interface for Claude, OpenAI, and custom providers
- First-class proxy support (z.ai, custom gateways)
- Environment-based configuration
- Streaming and non-streaming completions
- Tool/function calling support
- Drop-in for any Go project

## Installation

```bash
go get github.com/birddigital/go-llm-providers
```

## Quick Start

### Using Claude (with z.ai proxy)

```go
package main

import (
    "context"
    "fmt"
    "log"

    claude "github.com/birddigital/go-llm-providers/pkg/claude"
    "github.com/birddigital/go-llm-providers/config"
    "github.com/birddigital/go-llm-providers/pkg/providers"
)

func main() {
    // Load from environment (reads ANTHROPIC_API_KEY, ANTHROPIC_BASE_URL)
    cfg, err := config.LoadFromEnv(providers.ProviderClaude)
    if err != nil {
        log.Fatal(err)
    }

    // Create client
    client, err := claude.New(cfg.APIKey,
        claude.WithBaseURL(cfg.BaseURL),
        claude.WithTimeout(cfg.Timeout),
    )
    if err != nil {
        log.Fatal(err)
    }

    // Generate completion
    req := &providers.CompletionRequest{
        Model: cfg.Model,
        Messages: []providers.Message{
            {Role: "user", Content: "Explain quantum computing"},
        },
        MaxTokens: 1024,
    }

    resp, err := client.Complete(context.Background(), req)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println(resp.Content)
}
```

### Environment Configuration

```bash
# Claude (Anthropic) - Direct
export ANTHROPIC_API_KEY=sk-ant-xxx
export ANTHROPIC_MODEL=claude-3-5-sonnet-20241022

# Claude (Anthropic) - Via z.ai proxy
export ANTHROPIC_API_KEY=your-zai-key
export ANTHROPIC_BASE_URL=https://api.z.ai/api/anthropic
export ANTHROPIC_MODEL=claude-3-5-sonnet-20241022

# OpenAI
export OPENAI_API_KEY=sk-xxx
export OPENAI_MODEL=gpt-4o
```

### Streaming

```go
chunkChan, err := client.CompleteStream(ctx, req)
if err != nil {
    log.Fatal(err)
}

for chunk := range chunkChan {
    if chunk.Error != nil {
        log.Printf("Error: %v", chunk.Error)
        break
    }
    fmt.Print(chunk.Content)
    if chunk.Done {
        break
    }
}
```

## Supported Providers

| Provider | Status | Notes |
|----------|--------|-------|
| Claude (Anthropic) | ✅ | Full support including streaming |
| OpenAI | 🚧 | Planned |
| Custom | ✅ | Via BaseURL configuration |

## Proxy Configuration

### z.ai Proxy

The package has built-in support for z.ai proxy:

```bash
export ANTHROPIC_API_KEY=your-zai-token
export ANTHROPIC_BASE_URL=https://api.z.ai/api/anthropic
```

### Custom Proxies

```go
client, err := claude.New(apiKey,
    claude.WithBaseURL("https://your-proxy.com/v1"),
    claude.WithHeaders(map[string]string{
        "X-Custom-Header": "value",
    }),
)
```

## Project Integration

### learning-desktop

The learning-desktop project uses this package for AI tutoring:

```go
// internal/ai/tutor.go
package ai

import (
    "context"
    "github.com/birddigital/go-llm-providers/pkg/claude"
    "github.com/birddigital/go-llm-providers/pkg/providers"
)

type Tutor struct {
    client *claude.Client
}

func NewTutor(apiKey, baseURL string) (*Tutor, error) {
    client, err := claude.New(apiKey, claude.WithBaseURL(baseURL))
    if err != nil {
        return nil, err
    }
    return &Tutor{client: client}, nil
}

func (t *Tutor) Respond(ctx context.Context, question string) (string, error) {
    req := &providers.CompletionRequest{
        Model: "claude-3-5-sonnet-20241022",
        Messages: []providers.Message{
            {Role: "user", Content: question},
        },
        MaxTokens: 2048,
    }

    resp, err := t.client.Complete(ctx, req)
    if err != nil {
        return "", err
    }

    return resp.Content, nil
}
```

## License

MIT
