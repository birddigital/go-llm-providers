// Package config provides configuration management for LLM providers.
// Supports environment variables, config files, and direct configuration.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

import "github.com/birddigital/go-llm-providers/pkg/providers"

// ProviderConfig holds configuration for a specific LLM provider.
type ProviderConfig struct {
	Type     providers.ProviderType `json:"type"`
	APIKey   string                 `json:"api_key"`
	BaseURL  string                 `json:"base_url"`
	Model    string                 `json:"model"`
	Timeout  time.Duration          `json:"timeout"`
	MaxRetries int                  `json:"max_retries"`
	Headers  map[string]string      `json:"headers,omitempty"`
}

// LoadFromEnv loads provider configuration from environment variables.
func LoadFromEnv(provider providers.ProviderType) (*ProviderConfig, error) {
	cfg := &ProviderConfig{
		Type:     provider,
		Timeout:  60 * time.Second,
		MaxRetries: 3,
		Headers:  make(map[string]string),
	}

	// Get environment prefix for this provider
	prefix := envPrefix(provider)

	// Load API key
	apiKey := os.Getenv(prefix + "_API_KEY")
	if apiKey == "" {
		// Try alternate naming
		apiKey = os.Getenv(prefix + "_KEY")
	}
	if apiKey == "" {
		return nil, fmt.Errorf("%s API key not found in environment", provider)
	}
	cfg.APIKey = apiKey

	// Load base URL (optional, for proxies)
	if baseURL := os.Getenv(prefix + "_BASE_URL"); baseURL != "" {
		cfg.BaseURL = baseURL
	}

	// Load default model (optional)
	if model := os.Getenv(prefix + "_MODEL"); model != "" {
		cfg.Model = model
	}

	// Load timeout (optional)
	if timeoutSec := os.Getenv(prefix + "_TIMEOUT"); timeoutSec != "" {
		if sec, err := strconv.Atoi(timeoutSec); err == nil {
			cfg.Timeout = time.Duration(sec) * time.Second
		}
	}

	// Load max retries (optional)
	if retries := os.Getenv(prefix + "_MAX_RETRIES"); retries != "" {
		if r, err := strconv.Atoi(retries); err == nil {
			cfg.MaxRetries = r
		}
	}

	// Load custom headers (optional)
	// Format: HEADER_KEY_1=value1,HEADER_KEY_2=value2
	if headers := os.Getenv(prefix + "_HEADERS"); headers != "" {
		cfg.Headers = parseHeaders(headers)
	}

	// Apply proxy-specific defaults
	if cfg.BaseURL != "" {
		cfg = applyProxyDefaults(cfg)
	}

	return cfg, nil
}

// LoadFromEnvWithDefaults loads configuration with fallback values.
func LoadFromEnvWithDefaults(provider providers.ProviderType, defaults *ProviderConfig) (*ProviderConfig, error) {
	cfg, err := LoadFromEnv(provider)
	if err != nil {
		// Use defaults if env not found
		if defaults != nil {
			return defaults, nil
		}
		return nil, err
	}

	// Apply defaults for any missing values
	if defaults != nil {
		if cfg.BaseURL == "" && defaults.BaseURL != "" {
			cfg.BaseURL = defaults.BaseURL
		}
		if cfg.Model == "" && defaults.Model != "" {
			cfg.Model = defaults.Model
		}
		if cfg.Timeout == 0 && defaults.Timeout > 0 {
			cfg.Timeout = defaults.Timeout
		}
		if cfg.MaxRetries == 0 && defaults.MaxRetries > 0 {
			cfg.MaxRetries = defaults.MaxRetries
		}
	}

	return cfg, nil
}

// ToProvidersConfig converts to the internal providers.Config format.
func (c *ProviderConfig) ToProvidersConfig() *providers.Config {
	return &providers.Config{
		APIKey:     c.APIKey,
		BaseURL:    c.BaseURL,
		Timeout:    c.Timeout,
		MaxRetries: c.MaxRetries,
		Headers:    c.Headers,
	}
}

// envPrefix returns the environment variable prefix for a provider.
func envPrefix(provider providers.ProviderType) string {
	switch provider {
	case providers.ProviderClaude:
		return "ANTHROPIC"
	case providers.ProviderOpenAI:
		return "OPENAI"
	default:
		return strings.ToUpper(string(provider))
	}
}

// parseHeaders parses a comma-separated list of headers.
func parseHeaders(input string) map[string]string {
	headers := make(map[string]string)
	pairs := strings.Split(input, ",")
	for _, pair := range pairs {
		parts := strings.SplitN(pair, "=", 2)
		if len(parts) == 2 {
			headers[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}
	}
	return headers
}

// applyProxyDefaults applies known proxy configurations.
func applyProxyDefaults(cfg *ProviderConfig) *ProviderConfig {
	// z.ai proxy configuration
	if strings.Contains(cfg.BaseURL, "z.ai") {
		if cfg.Model == "" {
			cfg.Model = "claude-3-5-sonnet-20241022"
		}
		// Add any z.ai specific headers if needed
		if cfg.Headers == nil {
			cfg.Headers = make(map[string]string)
		}
	}
	return cfg
}

// GetModelForProvider returns the default model for a provider.
func GetModelForProvider(provider providers.ProviderType) string {
	switch provider {
	case providers.ProviderClaude:
		return "claude-3-5-sonnet-20241022"
	case providers.ProviderOpenAI:
		return "gpt-4o"
	default:
		return ""
	}
}

// IsProxyConfigured returns true if a proxy base URL is configured.
func IsProxyConfigured(provider providers.ProviderType) bool {
	prefix := envPrefix(provider)
	baseURL := os.Getenv(prefix + "_BASE_URL")
	return baseURL != "" && baseURL != defaultBaseURL(provider)
}

// defaultBaseURL returns the default base URL for a provider.
func defaultBaseURL(provider providers.ProviderType) string {
	switch provider {
	case providers.ProviderClaude:
		return "https://api.anthropic.com"
	case providers.ProviderOpenAI:
		return "https://api.openai.com"
	default:
		return ""
	}
}
