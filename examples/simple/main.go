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
	ctx := context.Background()

	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		fmt.Fprintln(os.Stderr, "Error: GEMINI_API_KEY environment variable is not set.")
		os.Exit(1)
	}

	// Initialize the client.
	client, err := search.NewClient(ctx, apiKey, search.WithModelName("gemini-2.5-flash-preview-04-17")) //
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating client:\n%+v\n", err)
		os.Exit(1)
	}

	// Optionally, list available models (commented out for brevity).
	// Uncomment the following lines to list available models.
	// models, err := client.ListAvailableModels(ctx)
	// if err != nil {
	// 	fmt.Fprintf(os.Stderr, "Error listing available models:\n%+v\n", err)
	// 	os.Exit(1)
	// }
	// fmt.Println("Available models:")
	// for _, model := range models {
	// 	fmt.Printf("- %s\n", model)
	// }

	// Create a context for the request, e.g., with a 30-second timeout.
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Define the search query (prompt).
	query := "What is the State-of-the-Art LLM model for Coding tasks at this time?"
	if len(os.Args) > 1 {
		query = os.Args[1] // Allow query override from command-line argument.
	}
	log.Printf("Querying with: \"%s\"\n", query)

	// Request grounded content generation.
	response, err := client.GenerateGroundedContent(ctx, query)
	if err != nil {
		// Example error handling; specific helper functions (e.g., IsAPIError) will be defined in errors.go.
		if apiErr, ok := search.GetAPIError(err); ok {
			// For APIError, we might want to show the specific API message and code,
			// but also the full wrapped error for more context if needed.
			fmt.Fprintf(os.Stderr, "API Error: %s (StatusCode: %d)\nFull error details:\n%+v\n", apiErr.Message, apiErr.StatusCode, err)
		} else if search.IsContentBlockedError(err) {
			fmt.Fprintf(os.Stderr, "Content generation blocked:\n%+v\n", err)
		} else {
			fmt.Fprintf(os.Stderr, "Error generating grounded content:\n%+v\n", err)
		}
		os.Exit(1)
	}

	// Display the generated text.
	// response.GeneratedText is part of the response struct to be defined in types.go.
	fmt.Printf("\n--- Generated Text ---\n%s\n\n", response.GeneratedText)

	// Display grounding attributions (sources).
	// response.GroundingAttributions is part of the response struct to be defined in types.go.
	if len(response.GroundingAttributions) > 0 {
		fmt.Println("--- Sources (Grounding Attributions) ---")
		for i, attr := range response.GroundingAttributions {
			fmt.Printf("%d. Title: %s\n", i+1, attr.Title) // Assumes attr.Title from types.go
			fmt.Printf("   URL: %s\n", attr.URL)           // Assumes attr.URL from types.go
			if attr.Segments != nil {
				fmt.Println("   Segments:")
				for _, segment := range attr.Segments {
					fmt.Printf("   - Start: %d, End: %d, Text: \"%s\"\n", segment.StartIndex, segment.EndIndex, segment.Text)
				}
			} else {
				fmt.Println("   No segments available.")
			}
		}
	} else {
		fmt.Println("No grounding attributions found for this response.")
	}

	// Optionally, display other information from the response, like search suggestions.
	if len(response.SearchSuggestions) > 0 { // Assumes response.SearchSuggestions from types.go
		fmt.Println("\n--- Search Suggestions ---")
		for _, suggestion := range response.SearchSuggestions {
			fmt.Printf("- %s\n", suggestion)
		}
	}
}
