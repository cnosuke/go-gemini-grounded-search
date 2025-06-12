package search

import (
	"net/http"
	"time"

	"github.com/cockroachdb/errors"
)

// ClientOption is a function type used to apply configuration options to a ClientConfig.
// It returns an error if an option is invalid or cannot be applied.
type ClientOption func(*ClientConfig) error

// --- Client Configuration Options ---

// WithModelName sets the default model name for the client.
// This model will be used for requests unless overridden by GenerationParams.
func WithModelName(name string) ClientOption {
	return func(cfg *ClientConfig) error {
		if name == "" {
			return ErrInvalidModelName
		}
		cfg.ModelName = name
		return nil
	}
}

// WithDefaultTemperature sets the default sampling temperature for the client.
// Valid range is typically [0.0, 2.0].
func WithDefaultTemperature(temp float32) ClientOption {
	return func(cfg *ClientConfig) error {
		if temp < 0.0 || temp > 2.0 { // Common range for Gemini, can be adjusted.
			return errors.Wrapf(ErrInvalidParameter, "temperature must be between 0.0 and 2.0, got %f", temp)
		}
		cfg.DefaultTemperature = &temp
		return nil
	}
}

// WithDefaultMaxOutputTokens sets the default maximum number of output tokens for the client.
// Must be positive if set.
func WithDefaultMaxOutputTokens(tokens int32) ClientOption {
	return func(cfg *ClientConfig) error {
		if tokens <= 0 {
			return errors.Wrapf(ErrInvalidParameter, "max output tokens must be positive, got %d", tokens)
		}
		cfg.DefaultMaxOutputTokens = &tokens
		return nil
	}
}

// WithDefaultTopK sets the default TopK sampling parameter for the client.
// Must be positive if set.
func WithDefaultTopK(k int32) ClientOption {
	return func(cfg *ClientConfig) error {
		if k <= 0 {
			return errors.Wrapf(ErrInvalidParameter, "top_k must be positive if set, got %d", k)
		}
		cfg.DefaultTopK = &k
		return nil
	}
}

// WithDefaultTopP sets the default TopP (nucleus) sampling parameter for the client.
// Valid range is typically (0.0, 1.0].
func WithDefaultTopP(p float32) ClientOption {
	return func(cfg *ClientConfig) error {
		if p <= 0.0 || p > 1.0 { // TopP is often > 0 and <= 1
			return errors.Wrapf(ErrInvalidParameter, "top_p must be between 0.0 (exclusive) and 1.0 (inclusive), got %f", p)
		}
		cfg.DefaultTopP = &p
		return nil
	}
}

// WithDefaultSafetySettings sets the default safety settings for the client.
func WithDefaultSafetySettings(settings []*SafetySetting) ClientOption {
	return func(cfg *ClientConfig) error {
		for _, s := range settings {
			if s == nil {
				return errors.Wrap(ErrInvalidParameter, "safety setting cannot be nil")
			}
			// Basic validation, can be expanded if HarmCategory/HarmBlockThreshold have exhaustive lists defined
			if s.Category == "" || s.Threshold == "" {
				return errors.Wrap(ErrInvalidParameter, "safety setting category and threshold cannot be empty")
			}
		}
		cfg.DefaultSafetySettings = settings
		return nil
	}
}

// WithHTTPClient sets a custom HTTP client to be used for API requests.
func WithHTTPClient(client *http.Client) ClientOption {
	return func(cfg *ClientConfig) error {
		if client == nil {
			return errors.Wrap(ErrInvalidParameter, "HTTP client cannot be nil if provided")
		}
		cfg.HTTPClient = client
		return nil
	}
}

// WithRequestTimeout sets the default timeout for API requests made by the client.
// Must not be negative. A value of 0 means no timeout at this level.
func WithRequestTimeout(timeout time.Duration) ClientOption {
	return func(cfg *ClientConfig) error {
		if timeout < 0 {
			return errors.Wrapf(ErrInvalidParameter, "request timeout cannot be negative, got %v", timeout)
		}
		cfg.RequestTimeout = timeout
		return nil
	}
}

// WithGoogleSearchToolDisabled allows disabling the Google Search Tool globally for the client.
func WithGoogleSearchToolDisabled(disabled bool) ClientOption {
	return func(cfg *ClientConfig) error {
		cfg.DisableGoogleSearchToolGlobally = disabled
		return nil
	}
}

// WithNoRedirection disables URL redirection and keeps the original URL.
func WithNoRedirection() ClientOption {
	return func(cfg *ClientConfig) error {
		cfg.NoRedirection = true
		return nil
	}
}

// applyClientOptions applies the given options to the ClientConfig.
// This is an unexported helper function called by NewClient.
func applyClientOptions(cfg *ClientConfig, opts ...ClientOption) error {
	for _, opt := range opts {
		if err := opt(cfg); err != nil {
			return err
		}
	}
	return nil
}
