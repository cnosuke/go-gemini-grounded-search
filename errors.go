package search

import (
	"errors"
	"fmt"

	"google.golang.org/api/iterator" // For checking if an error means "iterator done"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Sentinel errors for common client-side issues.
var (
	// ErrMissingAPIKey is returned when the API key is not provided during client initialization.
	ErrMissingAPIKey = errors.New("gemini: API key is missing")

	// ErrInvalidModelName is returned if an invalid or empty model name is provided.
	ErrInvalidModelName = errors.New("gemini: model name is invalid or empty")

	// ErrInvalidParameter is returned for invalid input parameters to library functions or options.
	ErrInvalidParameter = errors.New("gemini: invalid parameter provided")

	// ErrNoContentGenerated is returned when the API call is successful but the model generates no content.
	ErrNoContentGenerated = errors.New("gemini: model generated no content")

	// ErrContentBlocked is returned when content generation is blocked due to safety filters or other policy reasons.
	ErrContentBlocked = errors.New("gemini: content generation was blocked")

	// ErrUnsupportedFunctionality is returned when a requested feature or operation is not supported.
	ErrUnsupportedFunctionality = errors.New("gemini: unsupported functionality")
)

// APIError represents an error returned from the Gemini API.
// It wraps the underlying error and provides additional context like status codes.
type APIError struct {
	// StatusCode is the gRPC status code from the API response.
	// For non-gRPC errors or pre-request errors, this might be 0 or a mapped HTTP-like status.
	StatusCode codes.Code

	// Message is a human-readable message describing the error.
	Message string

	// Details may contain more specific information or structured error details from the API.
	Details []interface{} // Matches what genai.GenerateContentResponse.PromptFeedback.BlockReasonMessage can be

	// Err is the underlying error, if any.
	Err error
}

// Error implements the error interface for APIError.
func (e *APIError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("gemini: API error (status: %s): %s - %v", e.StatusCode.String(), e.Message, e.Err)
	}
	return fmt.Sprintf("gemini: API error (status: %s): %s", e.StatusCode.String(), e.Message)
}

// Unwrap returns the underlying wrapped error, allowing for errors.Is and errors.As.
func (e *APIError) Unwrap() error {
	return e.Err
}

// newAPIError creates a new APIError.
// This is an internal helper. Users should typically rely on error checking functions.
func newAPIError(code codes.Code, message string, originalError error, details ...interface{}) *APIError {
	return &APIError{
		StatusCode: code,
		Message:    message,
		Err:        originalError,
		Details:    details,
	}
}

// --- Error Type Checking Helper Functions ---

// IsAPIError checks if the given error is an *APIError.
func IsAPIError(err error) bool {
	var apiErr *APIError
	return errors.As(err, &apiErr)
}

// GetAPIError attempts to retrieve an *APIError from the given error.
// Returns the *APIError and true if successful, otherwise nil and false.
func GetAPIError(err error) (*APIError, bool) {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr, true
	}
	return nil, false
}

// IsAuthenticationError checks if an error is due to authentication issues (e.g., invalid API key).
// These typically correspond to gRPC codes Unauthenticated or PermissionDenied.
func IsAuthenticationError(err error) bool {
	if s, ok := status.FromError(err); ok {
		return s.Code() == codes.Unauthenticated || s.Code() == codes.PermissionDenied
	}
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.StatusCode == codes.Unauthenticated || apiErr.StatusCode == codes.PermissionDenied
	}
	return errors.Is(err, ErrMissingAPIKey) // Also consider client-side missing key
}

// IsQuotaError checks if an error is due to quota exhaustion or rate limiting.
// This typically corresponds to gRPC code ResourceExhausted.
func IsQuotaError(err error) bool {
	if s, ok := status.FromError(err); ok {
		return s.Code() == codes.ResourceExhausted
	}
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.StatusCode == codes.ResourceExhausted
	}
	return false
}

// IsInvalidRequestError checks if an error is due to an invalid request (e.g., malformed parameters).
// This typically corresponds to gRPC code InvalidArgument.
func IsInvalidRequestError(err error) bool {
	if s, ok := status.FromError(err); ok {
		return s.Code() == codes.InvalidArgument
	}
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.StatusCode == codes.InvalidArgument
	}
	return errors.Is(err, ErrInvalidParameter) || errors.Is(err, ErrInvalidModelName)
}

// IsContentBlockedError checks if the error indicates that content generation was blocked,
// often due to safety filters.
func IsContentBlockedError(err error) bool {
	// This error might be explicitly returned by our client logic based on API response fields
	// (e.g., PromptFeedback.BlockReason or Candidate.FinishReason == SAFETY).
	return errors.Is(err, ErrContentBlocked)
}

// IsServerError checks if an error is a server-side error from the Gemini API.
// These typically correspond to gRPC codes Internal, Unavailable, or Unknown.
func IsServerError(err error) bool {
	if s, ok := status.FromError(err); ok {
		return s.Code() == codes.Internal || s.Code() == codes.Unavailable || s.Code() == codes.Unknown
	}
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.StatusCode == codes.Internal || apiErr.StatusCode == codes.Unavailable || apiErr.StatusCode == codes.Unknown
	}
	return false
}

// IsIteratorDone is a helper to check for the specific error returned by the genai SDK
// when a streaming iterator (like for GenerateContentStream) is finished.
// While our library might not expose streaming directly initially, this can be useful.
func IsIteratorDone(err error) bool {
	return errors.Is(err, iterator.Done)
}
