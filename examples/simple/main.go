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
		log.Fatal("Error: GEMINI_API_KEY environment variable is not set.")
	}

	// Initialize the client.
	// Assumes default model (e.g., "gemini-2.5-flash") and Google Search Tool are enabled by default.
	client, err := search.NewClient(ctx, apiKey)
	if err != nil {
		log.Fatalf("Error creating client: %v", err)
	}

	// Create a context for the request, e.g., with a 30-second timeout.
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Define the search query (prompt).
	query := "What are the latest advancements in Generative AI safety, and what are some key research papers published in the last year?"
	if len(os.Args) > 1 {
		query = os.Args[1] // Allow query override from command-line argument.
	}
	log.Printf("Querying with: \"%s\"\n", query)

	// Request grounded content generation.
	response, err := client.GenerateGroundedContent(ctx, query)
	if err != nil {
		// Example error handling; specific helper functions (e.g., IsAPIError) will be defined in errors.go.
		if apiErr, ok := search.GetAPIError(err); ok {
			log.Fatalf("API Error: %v (StatusCode: %d)", apiErr.Message, apiErr.StatusCode)
		} else if search.IsContentBlockedError(err) {
			log.Fatalf("Content generation blocked: %v", err)
		}
		log.Fatalf("Error generating grounded content: %v", err)
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
			if attr.Snippet != "" {                        // Assumes attr.Snippet from types.go
				fmt.Printf("   Snippet: %s\n", attr.Snippet)
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
