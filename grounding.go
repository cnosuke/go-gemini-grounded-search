package search

import (
	"google.golang.org/genai"
)

// newGoogleSearchRetrieverTool creates a new genai.Tool configured for Google Search Retrieval.
// This helper function centralizes the creation of the search tool.
// It uses the GoogleSearchRetrieval field for grounding with public web data.
func newGoogleSearchRetrieverTool() *genai.Tool {
	return &genai.Tool{
		// For grounding using public web data, GoogleSearchRetrieval is typically used.
		// Refer to the SDK documentation for genai.Tool and genai.GoogleSearchRetrieval
		// for more configuration options if needed.
		GoogleSearchRetrieval: &genai.GoogleSearchRetrieval{},
	}
}

// extractGroundingAttributions transforms a slice of *genai.Citation (from the SDK)
// into the application's []GroundingAttribution format.
// The sdkCitations are typically obtained from a candidate's CitationMetadata.Citations field.
func extractGroundingAttributions(sdkCitations []*genai.Citation) ([]GroundingAttribution, error) {
	// If there are no SDK citations, return an empty slice.
	if len(sdkCitations) == 0 {
		return []GroundingAttribution{}, nil
	}

	// Pre-allocate slice with a capacity based on the number of SDK citations.
	appAttributions := make([]GroundingAttribution, 0, len(sdkCitations))

	for _, sdkCitation := range sdkCitations {
		if sdkCitation == nil {
			continue // Skip nil SDK citations, though this should be rare.
		}

		// Initialize our application's GroundingAttribution struct.
		appAttr := GroundingAttribution{
			// Store the raw SDK citation for users who might need deeper access.
			// This corresponds to the RawSource *genai.Citation field in your types.go.
			RawSource: sdkCitation,

			// Extract Title and URL directly from the SDK's Citation object.
			Title: sdkCitation.Title, // Assuming genai.Citation has a Title field
			URL:   sdkCitation.URI,   // Assuming genai.Citation has a URI field

			// Snippet: The genai.Citation type (from google.golang.org/genai/types.go you provided)
			// does not directly include a 'Snippet' field that corresponds to a pre-extracted text segment.
			// The old logic iterated over `sdkAttr.Content.Parts`; this is no longer applicable
			// as genai.Citation does not have such a field.
			//
			// Populating this 'Snippet' field would require custom logic:
			// 1. If another part of the SDK's response (e.g., within genai.Candidate.Content.Parts)
			//    contains text segments that are directly linked to these citations (e.g., via StartIndex/EndIndex
			//    from genai.Citation), you would extract them there.
			// 2. Alternatively, fetching and summarizing content from sdkCitation.URI could be an option,
			//    but that's an external operation.
			//
			// For this function, 'Snippet' will remain empty.
			// Consider if and how your application needs to populate this.
			Snippet: "",
		}

		appAttributions = append(appAttributions, appAttr)
	}

	return appAttributions, nil
}
