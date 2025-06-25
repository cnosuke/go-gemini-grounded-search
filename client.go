/*
Package search provides a Go client library for Google's Gemini API,
focusing on leveraging its Google Search Tool capabilities for grounded generation.
This library offers a simplified interface to obtain answers based on up-to-date
information from the web, along with their cited sources.

Basic Usage:

	ctx := context.Background()
	apiKey := os.Getenv("GEMINI_API_KEY")
	client, err := search.NewClient(ctx, apiKey)
	if err != nil {
	  log.Fatal(err)
	}
	defer client.Close()

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
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/cockroachdb/errors"
	"google.golang.org/genai"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Client is the main client for interacting with the Gemini API,
// configured for grounded search capabilities.
type Client struct {
	config                  ClientConfig                 // Resolved configuration after applying options
	genaiClient             *genai.Client                // Underlying client from the official Google AI Go SDK
	httpClient              *http.Client                 // HTTP client for non-API requests like redirection resolving
	defaultModel            string                       // Default model name (e.g., "gemini-2.5-flash")
	defaultGenContentConfig *genai.GenerateContentConfig // Default generation configuration
	userAgent               string                       // Combined user-agent string
}

// NewClient creates and initializes a new Gemini API client.
// apiKey is your Google AI API key.
// opts are functional options to customize the client's behavior.
func NewClient(ctx context.Context, apiKey string, opts ...ClientOption) (*Client, error) {
	cfg, err := newDefaultClientConfig(apiKey)
	if err != nil {
		return nil, err
	}

	if err := applyClientOptions(cfg, opts...); err != nil {
		return nil, err
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	sdkConfig := &genai.ClientConfig{
		APIKey: cfg.APIKey,
	}

	if cfg.HTTPClient != nil {
		sdkConfig.HTTPClient = cfg.HTTPClient
	}

	gClient, err := genai.NewClient(ctx, sdkConfig)
	if err != nil {
		if s, ok := status.FromError(err); ok {
			return nil, newAPIError(s.Code(), s.Message(), err, s.Details()...)
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
			sdkSafetySettings[i] = &genai.SafetySetting{
				Category:  genai.HarmCategory(s.Category),
				Threshold: genai.HarmBlockThreshold(s.Threshold),
			}
		}
		gConf.SafetySettings = sdkSafetySettings
	}

	if cfg.DisableGoogleSearchToolGlobally {
		gConf.Tools = nil
	} else {
		gConf.Tools = []*genai.Tool{
			newGoogleSearchRetrieverTool(),
		}
	}

	client := &Client{
		config:                  *cfg,
		genaiClient:             gClient,
		httpClient:              cfg.HTTPClient, // Use the configured client, or nil
		defaultModel:            cfg.ModelName,
		defaultGenContentConfig: &gConf,
	}
	return client, nil
}

// processGenaiResponse is a helper function to handle the response from genai.GenerateContent.
func (c *Client) processGenaiResponse(ctx context.Context, genaiResp *genai.GenerateContentResponse, callErr error) (*Response, error) {
	if callErr != nil {
		s, ok := status.FromError(callErr)
		if ok {
			if s.Code() == codes.InvalidArgument && containsSafetyBlockDetails(s.Details()) {
				return nil, newAPIError(s.Code(), s.Message(), ErrContentBlocked, s.Details()...)
			}
			return nil, newAPIError(s.Code(), s.Message(), callErr, s.Details()...)
		}
		return nil, newAPIError(codes.Unknown, "genai API call failed", callErr)
	}

	if genaiResp == nil {
		return nil, newAPIError(codes.Internal, "received nil response from API without explicit error", ErrNoContentGenerated)
	}

	// Based on user-provided SDK's types.go, PromptFeedback.BlockReason is a string.
	if genaiResp.PromptFeedback != nil && genaiResp.PromptFeedback.BlockReason != genai.BlockedReasonUnspecified { // genai.BlockedReasonUnspecified is a string const from SDK
		return nil, newAPIError(codes.InvalidArgument,
			fmt.Sprintf("prompt blocked due to %s: %s", genaiResp.PromptFeedback.BlockReason, genaiResp.PromptFeedback.BlockReasonMessage),
			ErrContentBlocked)
	}

	if len(genaiResp.Candidates) == 0 {
		return nil, ErrNoContentGenerated
	}

	candidate := genaiResp.Candidates[0]
	// Based on user-provided SDK's types.go, FinishReason is a string.
	if candidate.FinishReason == genai.FinishReasonSafety {
		var safetyDetails string
		if len(candidate.SafetyRatings) > 0 {
			safetyDetails = fmt.Sprintf(" (Ratings: %v)", candidate.SafetyRatings)
		}
		return nil, newAPIError(codes.FailedPrecondition,
			fmt.Sprintf("content generation stopped due to safety filters%s", safetyDetails),
			ErrContentBlocked)
	}

	if candidate.Content == nil || len(candidate.Content.Parts) == 0 {
		return nil, ErrNoContentGenerated
	}

	var generatedTextBuilder strings.Builder
	for _, part := range candidate.Content.Parts {
		if part.Text != "" {
			generatedTextBuilder.WriteString(part.Text)
		}
	}

	grounding, err := extractGroundingMetadata(candidate.GroundingMetadata)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to extract grounding metadata")
	}

	// If redirection is disabled, resolve the original URL.
	if c.config.NoRedirection {
		c.resolveGroundingURLs(ctx, grounding)
	}

	// Your application's Response struct (from your types.go)
	libResponse := &Response{
		GeneratedText:         generatedTextBuilder.String(),
		GroundingAttributions: grounding,
		SearchSuggestions:     []string{}, // TODO: Populate if new SDK provides similar info
		PromptFeedback:        genaiResp.PromptFeedback,
		Candidates:            genaiResp.Candidates,
		RawResponse:           genaiResp,
	}

	if libResponse.GeneratedText == "" && len(libResponse.GroundingAttributions) == 0 {
		return nil, ErrNoContentGenerated
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

// ListAvailableModels returns a list of available Gemini model names.
func (c *Client) ListAvailableModels(ctx context.Context) ([]string, error) {
	var models []string
	for m, err := range c.genaiClient.Models.All(ctx) {
		if err != nil {
			return nil, errors.Wrapf(err, "failed to list models")
		}
		if m == nil {
			continue
		}
		models = append(models, m.Name)
	}

	if len(models) == 0 {
		return nil, errors.New("no models available")
	}

	return models, nil
}

// GenerateGroundedContent sends a query to the Gemini API using client's default model settings.
func (c *Client) GenerateGroundedContent(ctx context.Context, query string) (*Response, error) {
	if query == "" {
		return nil, errors.Wrapf(ErrInvalidParameter, "query cannot be empty")
	}

	params := &GenerationParams{
		Prompt: query,
	}

	return c.GenerateGroundedContentWithParams(ctx, params)
}

// GenerateGroundedContentWithParams sends a query to the Gemini API with per-request parameters.
func (c *Client) GenerateGroundedContentWithParams(ctx context.Context, params *GenerationParams) (*Response, error) {
	if params == nil {
		return nil, errors.Wrapf(ErrInvalidParameter, "generation parameters cannot be nil")
	}
	if params.Prompt == "" {
		return nil, errors.Wrapf(ErrInvalidParameter, "prompt within generation parameters cannot be empty")
	}

	modelName := c.config.ModelName
	if params.ModelName != "" {
		modelName = params.ModelName
	}
	if modelName == "" {
		return nil, newAPIError(codes.InvalidArgument, "model name is not configured", ErrInvalidModelName)
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

	return c.processGenaiResponse(ctx, r, err)
}

// resolveOriginURL resolves one level of redirection for a given URL.
// It performs a single HEAD request to check if the URL redirects and returns
// the redirect destination, or the original URL if no redirect is found.
func resolveOriginURL(ctx context.Context, customClient *http.Client, urlStr string) (string, error) {
	// Use the provided custom client if available, otherwise create a dedicated client
	var client *http.Client
	if customClient != nil {
		// Clone the custom client but override CheckRedirect behavior
		client = &http.Client{
			Transport: customClient.Transport,
			Timeout:   3 * time.Second, // Override timeout for URL resolution
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse // Important: tells the client to return the redirect response
			},
		}
	} else {
		// Create a dedicated client for this task
		client = &http.Client{
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse // Important: tells the client to return the redirect response
			},
			Timeout: 3 * time.Second, // Fast per-request timeout to prevent hanging
		}
	}

	req, err := http.NewRequestWithContext(ctx, "HEAD", urlStr, nil)
	if err != nil {
		return "", errors.Wrapf(err, "failed to create request for %s", urlStr)
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", errors.Wrapf(err, "failed to send HEAD request to %s", urlStr)
	}
	defer resp.Body.Close()

	// If it's a redirect status code, return the redirect destination
	if resp.StatusCode >= 300 && resp.StatusCode <= 399 {
		location, err := resp.Location()
		if err != nil {
			if err == http.ErrNoLocation {
				// It's a 3xx status but no Location header. Return the original URL.
				return urlStr, nil
			}
			return "", errors.Wrapf(err, "failed to get location header from %s", urlStr)
		}
		return location.String(), nil
	}

	// Not a redirect, return the original URL
	return urlStr, nil
}

// urlResolveJob represents a job for URL resolution
type urlResolveJob struct {
	index int
	url   string
}

// urlResolveResult represents the result of URL resolution
type urlResolveResult struct {
	index int
	url   string
	err   error
}

// resolveGroundingURLs resolves redirect URLs to their original URLs using worker pattern
func (c *Client) resolveGroundingURLs(ctx context.Context, grounding []GroundingAttribution) {
	if len(grounding) == 0 {
		return
	}

	// Create context with timeout for URL resolution
	resolveCtx, cancel := c.createResolveContext(ctx)
	defer cancel()

	// Worker pattern implementation
	const numWorkers = 8
	jobs := make(chan urlResolveJob, len(grounding))
	results := make(chan urlResolveResult, len(grounding))

	// Start workers
	for w := 0; w < numWorkers; w++ {
		go c.urlResolveWorker(resolveCtx, jobs, results)
	}

	// Send jobs
	jobCount := 0
	for i := range grounding {
		if grounding[i].URL != "" {
			jobs <- urlResolveJob{index: i, url: grounding[i].URL}
			jobCount++
		}
	}
	close(jobs)

	// Collect results
	for range jobCount {
		select {
		case result := <-results:
			if result.err == nil && result.url != "" {
				grounding[result.index].URL = result.url
			} else if result.err != nil {
				// Log the error but continue; non-fatal.
				log.Printf("warning: failed to resolve origin URL for index %d: %v", result.index+1, result.err)
			}
		case <-resolveCtx.Done():
			log.Printf("warning: URL resolution timed out, some URLs may remain unresolved")
			return
		}
	}
}

// urlResolveWorker processes URL resolution jobs
func (c *Client) urlResolveWorker(ctx context.Context, jobs <-chan urlResolveJob, results chan<- urlResolveResult) {
	for job := range jobs {
		origin, err := resolveOriginURL(ctx, c.httpClient, job.url)
		results <- urlResolveResult{
			index: job.index,
			url:   origin,
			err:   err,
		}
	}
}

// createResolveContext creates a context with appropriate timeout for URL resolution
// The caller is responsible for calling the returned cancel function.
func (c *Client) createResolveContext(ctx context.Context) (context.Context, context.CancelFunc) {
	// Use remaining time from parent context, but cap at reasonable limit
	if deadline, ok := ctx.Deadline(); ok {
		remaining := time.Until(deadline)
		if remaining > 20*time.Second {
			// Cap URL resolution time to 20 seconds to leave time for main processing
			return context.WithTimeout(ctx, 20*time.Second)
		}
		return ctx, func() {} // No-op cancel function for existing context
	}
	// No deadline in parent context, set a reasonable timeout
	return context.WithTimeout(ctx, 15*time.Second)
}
