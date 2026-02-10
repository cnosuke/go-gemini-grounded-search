# go-gemini-grounded-search

[![Go Reference](https://pkg.go.dev/badge/github.com/cnosuke/go-gemini-grounded-search.svg)](https://pkg.go.dev/github.com/cnosuke/go-gemini-grounded-search)
[![Go Report Card](https://goreportcard.com/badge/github.com/cnosuke/go-gemini-grounded-search)](https://goreportcard.com/report/github.com/cnosuke/go-gemini-grounded-search)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

A Go client library for Google's Gemini API, focusing on leveraging its Google Search Tool capabilities for grounded generation. This library provides a simple and idiomatic Go interface to interact with Gemini, making it easy to get answers based on up-to-date information from the web.

## Features

- Simple, idiomatic Go API
- Support for Gemini API's Google Search Tool (Grounding) for fact-based responses
- Configurable via functional options pattern (e.g., model selection, temperature)
- Clear error handling for API interactions
- Typed request and response structures, including access to grounding metadata (sources and text segments)
- Optional URL redirection resolution to get original source URLs instead of redirect URLs

## Prerequisites

- Go 1.20 or later (refer to `go.mod` in the library for precise module dependencies)
- A Google Gemini API Key. You can obtain one from [Google AI Studio](https://aistudio.google.com/app/apikey).

## Installation

```bash
go get github.com/cnosuke/go-gemini-grounded-search
```

### CLI

```bash
# via Makefile
make install        # installs gemini-search to $GOBIN

# or directly
go install github.com/cnosuke/go-gemini-grounded-search/cmd/gemini-search@latest
```

## Quick Start

```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	search "github.com/cnosuke/go-gemini-grounded-search"
)

func main() {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		log.Fatal("Please set the GEMINI_API_KEY environment variable.")
	}

	// Create a context
	ctx := context.Background()

	// Create a new client with your API key
	// By default, it will use a model like "gemini-3-flash-preview" (see constants.go)
	// and enable Google Search Tool.
	client, err := search.NewClient(ctx, apiKey)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	// Perform a search (grounded generation)
	query := "What are the latest advancements in AI?"
	response, err := client.GenerateGroundedContent(ctx, query)
	if err != nil {
		log.Fatalf("Generation failed: %v", err)
	}

	// Print the generated text
	fmt.Printf("Generated Text for '%s':\n%s\n\n", query, response.GeneratedText)

	// Print grounding attributions (sources)
	if len(response.GroundingAttributions) > 0 {
		fmt.Println("Sources (Grounding Attributions):")
		for i, attr := range response.GroundingAttributions {
			fmt.Printf("%d. Title: %s\n", i+1, attr.Title)
			fmt.Printf("   URL: %s\n", attr.URL)
		}
	} else {
		fmt.Println("No grounding attributions found for this response.")
	}
}
```

## CLI Usage

```bash
export GEMINI_API_KEY="your-api-key"
gemini-search "幼児を連れても安心のオススメ東京観光スポットを教えて"
```

## Advanced Usage

### With Options

You can customize the client's behavior using functional options:

```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	search "github.com/cnosuke/go-gemini-grounded-search"
)

func main() {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		log.Fatal("Please set the GEMINI_API_KEY environment variable.")
	}

	// Create a context
	ctx := context.Background()

	// Note: To avoid hallucinations and ensure responses are factually grounded,
	// setting the temperature to 0.0 is strongly recommended.
	// (note that 0.0 is already the default temperature value in this library)
	var temperatureValue float32 = 0.0
	var maxTokensValue int32 = 500

	client, err := search.NewClient(
		ctx,
		apiKey,
		search.WithModelName("gemini-3-flash-preview"),           // Use a specific model (see options.go)
		search.WithDefaultTemperature(temperatureValue),        // Adjust temperature (see options.go)
		search.WithDefaultMaxOutputTokens(maxTokensValue),      // Adjust max output tokens
		search.WithRequestTimeout(90*time.Second),              // Set a request timeout
		search.WithNoRedirection(),                             // Resolve original URLs instead of redirect URLs
	)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	query := "Tell me a fun fact about the Go programming language, citing sources."
	response, err := client.GenerateGroundedContent(ctx, query)
	if err != nil {
		log.Fatalf("Generation failed: %v", err)
	}

	fmt.Printf("Generated Text:\n%s\n\n", response.GeneratedText)
	if len(response.GroundingAttributions) > 0 {
		fmt.Println("Sources:")
		for _, attr := range response.GroundingAttributions {
			fmt.Printf("- %s (%s)\n", attr.Title, attr.URL)
		}
	}
}
```

## API Reference

For detailed API documentation, see the [Go Reference](https://pkg.go.dev/github.com/cnosuke/go-gemini-grounded-search).

### Client

The `Client` is the main entry point for using the library:

```go
// Create a new client
client, err := search.NewClient(ctx, "your-api-key")

// Create a client with options
var temp float32 = 0.2
client, err := search.NewClient(
    ctx,
    "your-api-key",
    search.WithModelName("gemini-3-flash-preview"),
    search.WithDefaultTemperature(temp),
)
```

### Generating Grounded Content

```go
// Simple grounded content generation with default client settings
response, err := client.GenerateGroundedContent(ctx, "your query string")

// For more granular control per request, use GenerateGroundedContentWithParams:
var tempValue float32 = 0.1
var topKValue int32 = 10
params := &search.GenerationParams{
     Prompt:      "your query string with custom parameters",
     Temperature: &tempValue,
     TopK:        &topKValue,
     // other parameters like ModelName, MaxOutputTokens, SafetySettings, etc.
}
response, err := client.GenerateGroundedContentWithParams(ctx, params)
```

### URL Redirection Resolution

By default, Gemini's grounding service returns redirect URLs (e.g., `https://vertexaisearch.cloud.google.com/grounding-api-redirect/...`) instead of the original source URLs. You can enable automatic resolution to get the actual source URLs:

```go
// Enable automatic URL redirection resolution
client, err := search.NewClient(
    ctx,
    apiKey,
    search.WithNoRedirection(), // This will resolve redirect URLs to original URLs
)
```

This feature is useful when you want to:

- Display the actual source domain to users
- Perform further analysis on the source URLs
- Cache or store references to the original content

## Error Handling

The library provides detailed error information. Errors can be inspected to handle specific API issues using helper functions from the `search` package (defined in `errors.go`):

```go
if err != nil {
    if apiErr, ok := search.GetAPIError(err); ok {
        // Handle specific API errors
        log.Printf("API Error: Status %s (%d), Message: %s, Details: %v", apiErr.StatusCode.String(), apiErr.StatusCode, apiErr.Message, apiErr.Details)
        // You can also use other helper functions:
        if search.IsAuthenticationError(err) {
            log.Println("Authentication failed. Check your API key.")
        } else if search.IsQuotaError(err) {
            log.Println("API Quota exhausted.")
        }
    } else if search.IsContentBlockedError(err) {
        // Handle content blocked due to safety settings or other reasons
        log.Println("Content was blocked. Original error:", err)
    } else if errors.Is(err, search.ErrNoContentGenerated) {
        log.Println("The model generated no content for the given prompt.")
    } else {
        // Handle other types of errors (network, invalid parameters before API call, etc.)
        log.Fatalf("An unexpected error occurred: %v", err)
    }
}
```

The helper functions in `errors.go` (e.g., `IsAPIError`, `IsContentBlockedError`, `IsQuotaError`, `IsInvalidRequestError`, `IsServerError`) allow for robust error checking.

## Configuration

The library supports several configuration options through the functional options pattern passed to `NewClient` (see `options.go` for all available options):

- `WithModelName(name string)`: Specifies which Gemini model to use (e.g., `"gemini-3-flash-preview"`).
- `WithDefaultTemperature(temp float32)`: Sets the default generation temperature (0.0 for more factual, higher for more creative).
- `WithDefaultMaxOutputTokens(tokens int32)`: Sets the default maximum number of tokens to generate.
- `WithDefaultTopK(k int32)`: Sets the default TopK sampling parameter.
- `WithDefaultTopP(p float32)`: Sets the default TopP (nucleus) sampling parameter.
- `WithDefaultSafetySettings(settings []*SafetySetting)`: Sets default safety settings.
- `WithDefaultThinkingConfig(tc *ThinkingConfig)`: Controls the model's thinking behavior. Some models (e.g., `gemini-3-flash-preview`) have thinking enabled by default, which adds latency. Set `ThinkingBudget` to `0` to disable thinking.
- `WithHTTPClient(client *http.Client)`: Provides a custom HTTP client.
- `WithRequestTimeout(timeout time.Duration)`: Sets a default timeout for API requests.
- `WithGoogleSearchToolDisabled(disabled bool)`: Allows disabling the Google Search Tool globally for the client.
- `WithNoRedirection()`: Resolves original URLs from redirect URLs returned by the grounding service.

## Development Status

This library is currently in **early development**. While the core functionality is being actively built, the API surface is subject to change. Feedback and contributions are highly welcome\!

## License

This project is licensed under the MIT License - see the [LICENSE](https://www.google.com/search?q=LICENSE) file for details.

## Contributing

Contributions are welcome\! Please feel free to submit a Pull Request or open an issue.

## Acknowledgments

- This library utilizes the official [Google Generative AI Go SDK](https://github.com/google/generative-ai-go).
- Thanks to Google for providing the Gemini API and its powerful grounding capabilities.

## Author

cnosuke ( x.com/cnosuke )
