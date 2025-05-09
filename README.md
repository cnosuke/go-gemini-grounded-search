# go-gemini-grounded-search

[![Go Reference](https://pkg.go.dev/badge/github.com/cnosuke/go-gemini-grounded-search.svg)](https://pkg.go.dev/github.com/cnosuke/go-gemini-grounded-search)
[![Go Report Card](https://goreportcard.com/badge/github.com/cnosuke/go-gemini-grounded-search)](https://goreportcard.com/report/github.com/cnosuke/go-gemini-grounded-search)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

A Go client library for Google's Gemini API, focusing on leveraging its Google Search Tool capabilities for grounded generation. This library provides a simple and idiomatic Go interface to interact with Gemini, making it easy to get answers based on up-to-date information from the web.

## Features

- Simple, idiomatic Go API
- Support for Gemini API's Google Search Tool (Grounding) for fact-based responses
- Configurable via functional options pattern (e.g., model selection, temperature)
- Clear error handling for API interactions
- Typed request and response structures, including access to grounding metadata

## Prerequisites

- Go 1.20 or later
- A Google Gemini API Key. You can obtain one from [Google AI Studio](https://aistudio.google.com/app/apikey).

## Installation

```bash
go get github.com/cnosuke/go-gemini-grounded-search
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

	// Create a new client with your API key
	// By default, it will use a model like "gemini-2.0-flash" and enable Google Search Tool
	client, err := search.NewClient(apiKey)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close() // Important to close the client when done

	// Create a context
	ctx := context.Background()

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
			fmt.Printf("%d. [%s](%s)\n", i+1, attr.Title, attr.URL)
			if attr.Snippet != "" {
				fmt.Printf("   Snippet: %s\n", attr.Snippet)
			}
		}
	} else {
		fmt.Println("No grounding attributions found for this response.")
	}
}
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

	search "github.com/cnosuke/go-gemini-grounded-search"
)

func main() {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		log.Fatal("Please set the GEMINI_API_KEY environment variable.")
	}

	// Create a client with options
	client, err := search.NewClient(
		apiKey,
		search.WithModel("gemini-2.0-flash"), // Use a different model
		search.WithTemperature(0.5),                // Adjust temperature
		// Add other options as they are developed, e.g.,
		// search.WithTimeout(60*time.Second),
		// search.WithGroundingDisabled(), // If you want to allow disabling grounding
	)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()
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

For detailed API documentation, see the [Go Reference](https://www.google.com/url?sa=E&source=gmail&q=https://pkg.go.dev/github.com/cnosuke/go-gemini-grounded-search). (Link will be active once the library is published)

### Client

The `Client` is the main entry point for using the library:

```go
// Create a new client
client, err := search.NewClient("your-api-key")

// Create a client with options
client, err := search.NewClient(
    "your-api-key",
    search.WithModel("gemini-2.0-flash"),
    search.WithTemperature(0.2),
)
```

### Generating Grounded Content

```go
// Simple grounded content generation
response, err := client.GenerateGroundedContent(ctx, "your query string")

// You might also want to provide a more structured way to pass parameters
// if the Gemini API supports more granular control over the search/grounding.
// For example:
// params := &search.GenerationParams{
//     Prompt: "your query string",
//     // other parameters like desired source count, etc.
// }
// response, err := client.GenerateGroundedContentWithParams(ctx, params)
```

## Error Handling

The library provides detailed error information. Errors can be inspected to handle specific API issues:

```go
if err != nil {
    if search.IsAPIError(err) {
        apiErr := search.GetAPIError(err) // Hypothetical function to get typed error
        // Handle specific API errors, e.g., apiErr.StatusCode, apiErr.Message
        log.Printf("API Error: Status %d, Message: %s", apiErr.StatusCode, apiErr.Message)
    } else if search.IsContentBlockedError(err) {
        // Handle content blocked due to safety settings
        log.Println("Content was blocked due to safety settings.")
    } else if search.IsQuotaError(err) {
        // Handle quota exhausted error
        log.Println("API Quota exhausted.")
    } else {
        // Handle other types of errors (network, etc.)
        log.Fatalf("An unexpected error occurred: %v", err)
    }
}
```

_(Specific error handling functions like `IsAPIError`, `IsContentBlockedError`, `IsQuotaError` will need to be defined in `errors.go` based on the errors returned by the official Go SDK for Gemini.)_

## Configuration

The library supports several configuration options through the functional options pattern passed to `NewClient`:

- `WithModel(modelName string)`: Specifies which Gemini model to use (e.g., "gemini-2.0-flash").
- `WithTemperature(temp float32)`: Sets the generation temperature (0.0 for more factual, higher for more creative).
- _(More options will be added as the library develops, e.g., `WithTimeout`, `WithMaxRetries` if not handled by the underlying SDK, specific grounding parameters, etc.)_

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
