package search

import (
	"net/http"
	"time"

	"google.golang.org/genai"
)

// ClientConfig holds the configuration for the Gemini API client.
type ClientConfig struct {
	// APIKey is the Google AI API key for authenticating requests.
	// This field is mandatory.
	APIKey string

	// ModelName is the default Gemini model to be used for requests (e.g., "gemini-3-flash-preview").
	// Can be overridden per request via GenerationParams.
	ModelName string

	// DefaultTemperature is the default sampling temperature for generation.
	// Values typically range from 0.0 to 1.0. For grounded, factual responses, 0.0 is often preferred.
	// Use a pointer to distinguish between not set (nil) and explicitly set to 0.0.
	// If nil, the underlying SDK/API default will be used.
	DefaultTemperature *float32

	// DefaultMaxOutputTokens is the default maximum number of tokens to generate.
	// If nil, the underlying SDK/API default will be used.
	DefaultMaxOutputTokens *int32

	// DefaultTopK is the default TopK sampling parameter.
	// If nil, the underlying SDK/API default will be used.
	DefaultTopK *int32

	// DefaultTopP is the default TopP (nucleus) sampling parameter.
	// If nil, the underlying SDK/API default will be used.
	DefaultTopP *float32

	// DefaultSafetySettings is a list of default safety settings to apply to requests.
	// These can be overridden per request via GenerationParams.
	// If nil or empty, the underlying SDK/API defaults will apply.
	DefaultSafetySettings []*SafetySetting

	// DefaultThinkingConfig controls the Gemini model's thinking behavior.
	// If nil, the model's default thinking behavior is used.
	DefaultThinkingConfig *genai.ThinkingConfig

	// HTTPClient allows providing a custom *http.Client for making API requests.
	// If nil, the underlying genai SDK will use its default HTTP client.
	HTTPClient *http.Client

	// RequestTimeout is the default timeout duration for API requests made by the client.
	// If zero, no specific timeout is set at this library's client level, relying on
	// context deadlines or underlying SDK/HTTP client timeouts.
	RequestTimeout time.Duration

	// DisableGroundingToolGlobally, if true, makes the client not automatically enable
	// the Google Search Tool, even if the method implies its use.
	// Grounding can then be explicitly enabled via GenerationParams or specific methods.
	// Given the library name, this would typically be false.
	DisableGoogleSearchToolGlobally bool

	// NoRedirection, if true, instructs the client to resolve the original URL
	// from any redirected URL returned by the grounding service.
	NoRedirection bool
}

// newDefaultClientConfig creates a ClientConfig with sensible default values.
// These defaults will be defined in constants.go.
func newDefaultClientConfig(apiKey string) (*ClientConfig, error) {
	if apiKey == "" {
		return nil, ErrMissingAPIKey
	}
	defaultTemp := DefaultTemperature

	return &ClientConfig{
		APIKey:             apiKey,
		ModelName:          DefaultModelName,
		DefaultTemperature: &defaultTemp,
		// DefaultMaxOutputTokens, DefaultTopK, DefaultTopP can be left nil to use SDK/API defaults
		// DefaultSafetySettings can be initialized with common safe defaults or left nil
		DefaultSafetySettings:           nil,   // Or a predefined safe default set from constants.go
		DisableGoogleSearchToolGlobally: false, // Enable grounding by default for this library
		RequestTimeout:                  DefaultRequestTimeout,
		NoRedirection:                   false, // Default to following redirects
	}, nil
}

// validate checks if the essential parts of the ClientConfig are valid.
// Currently, it only checks for the APIKey.
func (c *ClientConfig) validate() error {
	if c.APIKey == "" {
		// This error (e.g., ErrMissingAPIKey) will be defined in errors.go
		return ErrMissingAPIKey
	}
	// Add other validations as necessary, e.g., for ModelName format, etc.
	return nil
}
