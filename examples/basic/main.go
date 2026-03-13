// Package main demonstrates basic usage of go-llm-providers.
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	claude "github.com/birddigital/go-llm-providers/pkg/claude"
	"github.com/birddigital/go-llm-providers/pkg/providers"
)

func main() {
	// Get API key from environment
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		log.Fatal("ANTHROPIC_API_KEY environment variable is required")
	}

	// Get base URL (for proxy support)
	baseURL := os.Getenv("ANTHROPIC_BASE_URL")
	if baseURL == "" {
		baseURL = "https://api.anthropic.com" // Default
	}

	// Create client with proxy support
	opts := []claude.Option{
		claude.WithBaseURL(baseURL),
		claude.WithTimeout(60 * 1000000000), // 60 seconds
	}

	client, err := claude.New(apiKey, opts...)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	// Create completion request
	req := &providers.CompletionRequest{
		Model: "claude-3-5-sonnet-20241022",
		Messages: []providers.Message{
			{
				Role:    "user",
				Content: "Explain what a learning management system is in 3 sentences.",
			},
		},
		MaxTokens:   500,
		Temperature: 0.7,
	}

	// Generate completion
	fmt.Println("Sending request to Claude...")
	fmt.Printf("Via: %s\n\n", baseURL)

	resp, err := client.Complete(context.Background(), req)
	if err != nil {
		log.Fatalf("Completion failed: %v", err)
	}

	// Print response
	fmt.Println("Response:")
	fmt.Println(resp.Content)
	fmt.Printf("\nTokens: %d input, %d output (total: %d)\n",
		resp.Usage.InputTokens,
		resp.Usage.OutputTokens,
		resp.Usage.TotalTokens,
	)
	fmt.Printf("Model: %s\n", resp.ModelID)
	fmt.Printf("Request ID: %s\n", resp.RequestID)
	fmt.Printf("Latency: %v\n", resp.Latency)
}
