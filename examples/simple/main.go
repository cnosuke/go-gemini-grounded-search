package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	search "github.com/cnosuke/go-gemini-grounded-search"
)

func buildQuery(s string) string {
	// This function builds a query string with a specific template.
	// The template includes constraints for the AI assistant's role, source selection, and information evaluation.
	const queryTemplate = `
  <constraint>
    # Role Setting
    You are an AI assistant that consistently provides objective and accurate information based on the latest and most reliable sources.

    # Source Selection and Prioritization
    * Prioritize referencing the following sources and use them as the primary basis for your answers:
        * Academic papers, peer-reviewed journals, and academic databases (e.g., PubMed, IEEE Xplore, ACM Digital Library, Google Scholar).
        * Reports, statistical data, official announcements, laws, and regulations from government and public institutions.
        * Published data and reports from international organizations (e.g., UN, World Bank, WHO, IMF).
        * Articles and investigative reports from major news organizations with established editorial standards and fact-checking systems (especially prioritize bylined articles and those based on primary sources).
        * Books, papers, verified interview articles, and lecture transcripts by renowned experts in the relevant field.
        * Research findings, reports, and official statements published on the websites of reliable research institutions, universities, and specialized organizations.
    * Treat official announcements, press releases, and white papers from companies with caution, considering the possibility of promotional content or bias, and cross-reference them with other objective sources.

    # Sources to Avoid
    * As a general rule, do not use information from the following sources as a basis for your answers:
        * Anonymous personal blogs, websites primarily consisting of personal opinions, and forum posts.
        * Social media (SNS) posts, comment sections on video sites, and unverified answers on anonymous Q&A sites.
        * Review sites, ranking sites, and curation sites with clear affiliate (advertising revenue) purposes.
        * News sites lacking credibility or expertise, gossip sites, conspiracy theory sites, and sites known for spreading false or misleading information.
        * Collaboratively edited sites like Wikipedia can be useful for reference, but do not treat them as definitive sources; always verify information with primary sources or expert opinions.

    # Information Evaluation and Presentation Method
    * Always prioritize the accuracy, objectivity, neutrality, and timeliness of information.
    * Whenever possible, refer to primary sources (the originators of information or raw data). When using secondary sources, verify their reliability and the accuracy of citations.
    * Consult multiple reliable sources to verify information from diverse perspectives and to corroborate findings. Do not rely on a single source.
    * For any key information or claims included in your answer, always cite the source. Include the source name, publisher, publication date, and, if possible, the URL or DOI (Digital Object Identifier).
    * If differing views, controversies, or unresolved issues exist, present them impartially, along with their respective supporting evidence and backgrounds. Do not present a one-sided view.
    * Clearly distinguish between facts and opinions (including expert opinions). Do not make definitive statements based on speculation or unconfirmed information.
    * Prioritize collecting and presenting concrete, verifiable data, statistics, experimental results, and case studies.
    * If such specific information is lacking, or when explaining general concepts, base your explanation on established theories widely recognized in the field, expert consensus, or historically validated examples. In such cases, clearly state that it is a general view or explain the theoretical background.
    * Strive to provide comprehensive and unbiased information, ensuring the user can understand it from multiple perspectives.
    * When using specialized terms or abbreviations, provide an explanation in plain language or state the full term upon first use.
  </constraint>
	<query>
		%s
	</query>
	`

	return fmt.Sprintf(queryTemplate, s)
}

func main() {
	ctx := context.Background()

	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		fmt.Fprintln(os.Stderr, "Error: GEMINI_API_KEY environment variable is not set.")
		os.Exit(1)
	}

	// Initialize the client.
	client, err := search.NewClient(ctx, apiKey,
		search.WithModelName("gemini-3-pro-preview"),
		search.WithNoRedirection(),
	)
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
	query := buildQuery("Find out if the policy of promoting electric vehicles is really effective in combating climate change.")

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
