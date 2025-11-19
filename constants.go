package search

import (
	"time"
)

const (
	// LibraryName is the name of this library.
	LibraryName = "go-gemini-grounded-search"

	// LibraryVersion is the current version of this library.
	// Follow semantic versioning (https://semver.org/).
	LibraryVersion = "0.1.0"
)

// Default configuration values for the client.
const (
	// DefaultModelName is the default Gemini model used if not specified by the user.
	// Gemini flash serieses are chosen for its balance of speed and capability for grounded search.
	DefaultModelName = "gemini-3.0-flash"

	// DefaultTemperature for grounded search tasks.
	// 0.0f is generally recommended for factuality and to minimize hallucinations.
	DefaultTemperature float32 = 0.0

	// DefaultRequestTimeout is the default duration for API requests.
	DefaultRequestTimeout = 60 * time.Second
)

// Note: Constants for HarmCategory and HarmBlockThreshold are defined in types.go
// as they are part of the public API type definitions for SafetySetting.
// If we wanted to define a *default set* of SafetySettings, those could be here, e.g.:
/*
var DefaultSafetySettings = []*SafetySetting{
	{Category: HarmCategoryHarassment, Threshold: HarmBlockThresholdBlockMedium},
	{Category: HarmCategoryHateSpeech, Threshold: HarmBlockThresholdBlockMedium},
	{Category: HarmCategorySexuallyExplicit, Threshold: HarmBlockThresholdBlockMedium},
	{Category: HarmCategoryDangerousContent, Threshold: HarmBlockThresholdBlockMedium},
}
*/
// For now, newDefaultClientConfig in config.go sets DefaultSafetySettings to nil,
// implying reliance on the API/SDK's default safety settings unless overridden by the user.
