package search

import (
	"google.golang.org/genai"
)

// newGoogleSearchRetrieverTool creates a new genai.Tool configured for Google Search Retrieval.
// This helper function centralizes the creation of the search tool.
// It uses the GoogleSearch field for grounding with public web data.
func newGoogleSearchRetrieverTool() *genai.Tool {
	return &genai.Tool{
		// For grounding using public web data, GoogleSearch is typically used.
		// Refer to the SDK documentation for genai.Tool and genai.GoogleSearch
		// for more configuration options if needed.
		GoogleSearch: &genai.GoogleSearch{},
	}
}

// extractGroundingMetadata transforms grounding metadata from the SDK (*genai.GroundingMetadata)
// into a slice of GroundingAttribution.
func extractGroundingMetadata(metadata *genai.GroundingMetadata) ([]GroundingAttribution, error) {
	if metadata == nil || len(metadata.GroundingChunks) == 0 {
		// No chunks, so no attributions to create based on chunks.
		// If there are GroundingSupports without chunks, they would be orphaned based on current logic.
		// Depending on requirements, might still process supports if they don't rely on chunk linkage.
		return []GroundingAttribution{}, nil
	}

	// Initialize a slice for our application-specific GroundingAttribution.
	// The size is based on the number of chunks, as each chunk will form the basis of one GroundingAttribution.
	numChunks := len(metadata.GroundingChunks)
	appAttributions := make([]GroundingAttribution, numChunks)

	for i, c := range metadata.GroundingChunks {
		if c == nil {
			// Initialize with empty data or handle error if a nil chunk is unexpected.
			appAttributions[i] = GroundingAttribution{
				Segments: []GroundingAttributionSegment{},
			}
			continue
		}

		var title, domain, url string
		if c.Web != nil {
			title = c.Web.Title
			domain = c.Web.Domain
			url = c.Web.URI
		} else if c.RetrievedContext != nil {
			title = c.RetrievedContext.Title
			// Domain might not be applicable or available for RetrievedContext
			url = c.RetrievedContext.URI
		}

		appAttributions[i] = GroundingAttribution{
			Title:    title,
			Domain:   domain,
			URL:      url,
			Segments: []GroundingAttributionSegment{},
		}
	}

	// Now, process the GroundingSupports and link their segments to the appropriate GroundingAttribution.
	for _, s := range metadata.GroundingSupports {
		if s == nil || s.Segment == nil {
			continue
		}

		segment := s.Segment
		confidenceScore := float32(0.0)

		// If ConfidenceScores are available, use the first one for this segment.
		// Adjust this logic if a more specific mapping is required.
		if len(s.ConfidenceScores) > 0 {
			confidenceScore = s.ConfidenceScores[0]
		}

		appSegment := GroundingAttributionSegment{
			StartIndex:      int(segment.StartIndex),
			PartIndex:       int(segment.PartIndex),
			EndIndex:        int(segment.EndIndex),
			Text:            segment.Text,
			ConfidenceScore: confidenceScore,
		}

		// Link this segment to all chunks referenced by this support.
		for _, chunkIndex32 := range s.GroundingChunkIndices {
			chunkIndex := int(chunkIndex32)
			if chunkIndex >= 0 && chunkIndex < numChunks {
				appAttributions[chunkIndex].Segments = append(appAttributions[chunkIndex].Segments, appSegment)
			} else {
				// Handle or log invalid chunk index if necessary
				// return nil, fmt.Errorf("invalid chunk index %d in GroundingSupport", chunkIndex)
			}
		}
	}

	return appAttributions, nil
}
