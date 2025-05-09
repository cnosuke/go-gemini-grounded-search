/*
Package search provides a Go client library for Google's Gemini API,
focusing on leveraging its Google Search Tool capabilities for grounded generation.
This library offers a simplified interface to obtain answers based on up-to-date
information from the web, along with their cited sources.

Basic Usage:

	apiKey := os.Getenv("GEMINI_API_KEY")
	client, err := search.NewClient(apiKey)
	if err != nil {
	  log.Fatal(err)
	}
	defer client.Close()

	ctx := context.Background()
	response, err := client.GenerateGroundedContent(ctx, "What are the recent developments in quantum computing?")
	if err != nil {
	  log.Fatal(err)
	}

	fmt.Println("Generated Text:", response.GeneratedText)
	for _, attr := range response.GroundingAttributions {
	  fmt.Printf("Source: %s (%s)\n", attr.Title, attr.URL)
	}

See the examples directory and README.md for more detailed usage patterns.
*/
package search

import (
	"context"
	"fmt"
	"strings" // Added for strings.Builder

	// "google.golang.org/api/option" // REMOVED: No longer using this for ClientOptions

	"google.golang.org/genai" // This is the new SDK client package
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Client is the main client for interacting with the Gemini API,
// configured for grounded search capabilities.
type Client struct {
	config                  ClientConfig                 // Resolved configuration after applying options
	genaiClient             *genai.Client                // Underlying client from the official Google AI Go SDK
	defaultModel            string                       // Default model name (e.g., "gemini-2.5-flash")
	defaultGenContentConfig *genai.GenerateContentConfig // Default generation configuration
	userAgent               string                       // Combined user-agent string
}

// NewClient creates and initializes a new Gemini API client.
// apiKey is your Google AI API key.
// opts are functional options to customize the client's behavior.
func NewClient(ctx context.Context, apiKey string, opts ...ClientOption) (*Client, error) {
	cfg, err := newDefaultClientConfig(apiKey) // Assuming this function initializes ClientConfig
	if err != nil {
		return nil, err
	}

	if err := applyClientOptions(cfg, opts...); err != nil { // Assuming this function applies custom options
		return nil, err
	}

	if err := cfg.validate(); err != nil { // Assuming this validates ClientConfig
		return nil, err
	}

	// Use ClientOptions from the google.golang.org/genai package directly
	sdkConfig := &genai.ClientConfig{
		APIKey: cfg.APIKey,
	}

	if cfg.HTTPClient != nil {
		sdkConfig.HTTPClient = cfg.HTTPClient
	}

	gClient, err := genai.NewClient(ctx, sdkConfig) // Assuming this is the correct way to create a new genai client
	if err != nil {
		if s, ok := status.FromError(err); ok {
			return nil, newAPIError(s.Code(), s.Message(), err, s.Details()...) // Assuming newAPIError is defined
		}
		return nil, newAPIError(codes.Internal, "failed to create genai client", err)
	}

	var gConf genai.GenerateContentConfig

	if cfg.DefaultTemperature != nil {
		gConf.Temperature = cfg.DefaultTemperature
	}
	if cfg.DefaultTopK != nil { // ClientConfig.DefaultTopK is *int32, SDK's GenerationConfig.TopK is *float32
		topKVal := float32(*cfg.DefaultTopK)
		gConf.TopK = &topKVal
	}
	if cfg.DefaultTopP != nil {
		gConf.TopP = cfg.DefaultTopP
	}
	if cfg.DefaultMaxOutputTokens != nil {
		gConf.MaxOutputTokens = *cfg.DefaultMaxOutputTokens
	}
	if cfg.DefaultSafetySettings != nil && len(cfg.DefaultSafetySettings) > 0 {
		sdkSafetySettings := make([]*genai.SafetySetting, len(cfg.DefaultSafetySettings))
		for i, s := range cfg.DefaultSafetySettings {
			// Based on user-provided SDK's types.go, HarmCategory and HarmBlockThreshold are string types.
			sdkSafetySettings[i] = &genai.SafetySetting{
				Category:  genai.HarmCategory(s.Category),        // Assumes s.Category (app type) is string
				Threshold: genai.HarmBlockThreshold(s.Threshold), // Assumes s.Threshold (app type) is string
			}
		}
		gConf.SafetySettings = sdkSafetySettings
	}
	if cfg.DisableGoogleSearchToolGlobally {
		gConf.Tools = nil
	} else {
		gConf.Tools = []*genai.Tool{newGoogleSearchRetrieverTool()} // Use from grounding.go
	}

	client := &Client{
		config:                  *cfg,
		genaiClient:             gClient,
		defaultModel:            cfg.ModelName,
		defaultGenContentConfig: &gConf,
	}
	return client, nil
}

// processGenaiResponse is a helper function to handle the response from genai.GenerateContent.
func (c *Client) processGenaiResponse(genaiResp *genai.GenerateContentResponse, callErr error) (*Response, error) {
	if callErr != nil {
		s, ok := status.FromError(callErr)
		if ok {
			if s.Code() == codes.InvalidArgument && containsSafetyBlockDetails(s.Details()) {
				return nil, newAPIError(s.Code(), s.Message(), ErrContentBlocked, s.Details()...) // Assuming ErrContentBlocked defined
			}
			return nil, newAPIError(s.Code(), s.Message(), callErr, s.Details()...)
		}
		return nil, newAPIError(codes.Unknown, "genai API call failed", callErr)
	}

	if genaiResp == nil {
		return nil, newAPIError(codes.Internal, "received nil response from API without explicit error", ErrNoContentGenerated) // Assuming ErrNoContentGenerated
	}

	// Based on user-provided SDK's types.go, PromptFeedback.BlockReason is a string.
	if genaiResp.PromptFeedback != nil && genaiResp.PromptFeedback.BlockReason != genai.BlockedReasonUnspecified { // genai.BlockedReasonUnspecified is a string const from SDK
		return nil, newAPIError(codes.InvalidArgument,
			fmt.Sprintf("prompt blocked due to %s: %s", genaiResp.PromptFeedback.BlockReason, genaiResp.PromptFeedback.BlockReasonMessage),
			ErrContentBlocked) // Assuming ErrContentBlocked
	}

	if len(genaiResp.Candidates) == 0 {
		return nil, ErrNoContentGenerated // Assuming ErrNoContentGenerated
	}

	candidate := genaiResp.Candidates[0]
	// Based on user-provided SDK's types.go, FinishReason is a string.
	if candidate.FinishReason == genai.FinishReasonSafety { // genai.FinishReasonSafety is a string const from SDK
		var safetyDetails string
		if len(candidate.SafetyRatings) > 0 {
			safetyDetails = fmt.Sprintf(" (Ratings: %v)", candidate.SafetyRatings)
		}
		return nil, newAPIError(codes.FailedPrecondition,
			fmt.Sprintf("content generation stopped due to safety filters%s", safetyDetails),
			ErrContentBlocked) // Assuming ErrContentBlocked
	}

	if candidate.Content == nil || len(candidate.Content.Parts) == 0 {
		return nil, ErrNoContentGenerated // Assuming ErrNoContentGenerated
	}

	var generatedTextBuilder strings.Builder
	for _, part := range candidate.Content.Parts {
		// New SDK's genai.Part struct has a 'Text string' field.
		if part.Text != "" { // No need for type assertion like part.(genai.Text)
			generatedTextBuilder.WriteString(part.Text)
		}
	}

	var sdkCitations []*genai.Citation
	if candidate.CitationMetadata != nil {
		sdkCitations = candidate.CitationMetadata.Citations
	}
	extractedAttributions, err := extractGroundingAttributions(sdkCitations) // From grounding.go
	if err != nil {
		return nil, fmt.Errorf("failed to process grounding attributions: %w", err)
	}

	// Your application's Response struct (from your types.go)
	libResponse := &Response{
		GeneratedText:         generatedTextBuilder.String(),
		GroundingAttributions: extractedAttributions,
		SearchSuggestions:     []string{}, // TODO: Populate if new SDK provides similar info
		PromptFeedback:        genaiResp.PromptFeedback,
		Candidates:            genaiResp.Candidates,
		RawResponse:           genaiResp,
	}

	if libResponse.GeneratedText == "" && len(libResponse.GroundingAttributions) == 0 {
		return nil, ErrNoContentGenerated // Assuming ErrNoContentGenerated
	}

	return libResponse, nil
}

// containsSafetyBlockDetails checks if error details indicate a safety block.
// Details type is []any as per status.Details().
func containsSafetyBlockDetails(details []any) bool {
	for _, detail := range details {
		// This check is a placeholder and needs to be adapted based on
		// how the SDK actually structures safety-related error details.
		// It might involve checking for specific proto messages or string patterns.
		if detailStr, ok := detail.(string); ok {
			if strings.Contains(strings.ToUpper(detailStr), "SAFETY") {
				return true
			}
		}
		// Example for structured error (if SDK uses something like this):
		// if _, ok := detail.(*errdetails.ErrorInfo); ok {
		//   // check specific fields of ErrorInfo
		// }
	}
	return false
}

// GenerateGroundedContent sends a query to the Gemini API using client's default model settings.
func (c *Client) GenerateGroundedContent(ctx context.Context, query string) (*Response, error) {
	if query == "" {
		return nil, fmt.Errorf("%w: query cannot be empty", ErrInvalidParameter) // Assuming ErrInvalidParameter
	}

	params := &GenerationParams{
		Prompt: query,
	}

	return c.GenerateGroundedContentWithParams(ctx, params)
}

// GenerateGroundedContentWithParams sends a query to the Gemini API with per-request parameters.
func (c *Client) GenerateGroundedContentWithParams(ctx context.Context, params *GenerationParams) (*Response, error) {
	if params == nil {
		return nil, fmt.Errorf("%w: generation parameters cannot be nil", ErrInvalidParameter) // Assuming ErrInvalidParameter
	}
	if params.Prompt == "" {
		return nil, fmt.Errorf("%w: prompt within generation parameters cannot be empty", ErrInvalidParameter) // Assuming ErrInvalidParameter
	}

	modelName := c.config.ModelName
	if params.ModelName != "" {
		modelName = params.ModelName
	}
	if modelName == "" {
		return nil, newAPIError(codes.InvalidArgument, "model name is not configured", ErrInvalidModelName) // Assuming ErrInvalidModelName
	}

	model := c.defaultModel
	if params.ModelName != "" {
		model = params.ModelName
	}

	// Apply generation parameters by modifying a copy of the model's GenerationConfig
	currentConfig := *c.defaultGenContentConfig // Copy the default config to avoid modifying the original

	if params.Temperature != nil {
		currentConfig.Temperature = params.Temperature
	}

	if params.TopK != nil { // params.TopK is *int32, SDK's GenerationConfig.TopK is *float32
		topKVal := float32(*params.TopK)
		currentConfig.TopK = &topKVal
	}
	if params.TopP != nil {
		currentConfig.TopP = params.TopP
	}

	if params.MaxOutputTokens != nil {
		currentConfig.MaxOutputTokens = *params.MaxOutputTokens
	}

	if params.CandidateCount != nil {
		currentConfig.CandidateCount = *params.CandidateCount
	}

	if params.StopSequences != nil && len(params.StopSequences) > 0 {
		currentConfig.StopSequences = params.StopSequences
	}

	// Apply safety settings (directly on the model struct)
	if params.SafetySettings != nil && len(params.SafetySettings) > 0 {
		sdkSafetySettings := make([]*genai.SafetySetting, len(params.SafetySettings))
		for i, s := range params.SafetySettings {
			sdkSafetySettings[i] = &genai.SafetySetting{
				Category:  genai.HarmCategory(s.Category),
				Threshold: genai.HarmBlockThreshold(s.Threshold),
			}
		}
		currentConfig.SafetySettings = sdkSafetySettings
	}

	contents := []*genai.Content{
		genai.NewContentFromText(params.Prompt, genai.RoleUser),
	}

	var cancelFunc context.CancelFunc = func() {}
	if c.config.RequestTimeout > 0 {
		_, deadlineSet := ctx.Deadline()
		if !deadlineSet {
			ctx, cancelFunc = context.WithTimeout(ctx, c.config.RequestTimeout)
		}
	}
	defer cancelFunc()

	r, err := c.genaiClient.Models.GenerateContent(ctx, model, contents, &currentConfig)

	return c.processGenaiResponse(r, err)
}
