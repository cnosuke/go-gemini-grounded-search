package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	search "github.com/cnosuke/go-gemini-grounded-search"
	"github.com/urfave/cli/v3"
)

const defaultModel = "gemini-3-flash-preview"

func parseThinkingLevel(s string) (search.ThinkingLevel, error) {
	switch strings.ToUpper(s) {
	case "MINIMAL":
		return search.ThinkingLevelMinimal, nil
	case "LOW":
		return search.ThinkingLevelLow, nil
	case "MEDIUM":
		return search.ThinkingLevelMedium, nil
	case "HIGH":
		return search.ThinkingLevelHigh, nil
	default:
		return "", fmt.Errorf("invalid thinking level %q: must be one of minimal, low, medium, high", s)
	}
}

func main() {
	cmd := &cli.Command{
		Name:  "gemini-search",
		Usage: "A CLI tool to perform a grounded search using the Gemini API.",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "api-key",
				Aliases: []string{"k"},
				Usage:   "Google AI API key. Can also be set with the GEMINI_API_KEY environment variable.",
			},
			&cli.StringFlag{
				Name:    "model",
				Aliases: []string{"m"},
				Usage:   "Gemini model to use. Can also be set with the GEMINI_MODEL_ID environment variable.",
			},
			&cli.StringFlag{
				Name:    "thinking-level",
				Aliases: []string{"t"},
				Usage:   "Thinking level for the model (minimal, low, medium, high). Only for Gemini 3 series models.",
			},
			&cli.BoolFlag{
				Name:    "verbose",
				Aliases: []string{"v"},
				Usage:   "Enable verbose output for debugging.",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			apiKey := cmd.String("api-key")
			if apiKey == "" {
				apiKey = os.Getenv("GEMINI_API_KEY")
			}
			if apiKey == "" {
				return cli.Exit("API key is required. Set it with --api-key or the GEMINI_API_KEY environment variable.", 1)
			}

			model := cmd.String("model")
			if model == "" {
				model = os.Getenv("GEMINI_MODEL_ID")
			}
			if model == "" {
				model = defaultModel
			}

			query := cmd.Args().First()
			if query == "" {
				return cli.Exit("Search query argument is required.", 1)
			}

			var clientOpts []search.ClientOption
			clientOpts = append(clientOpts, search.WithNoRedirection())
			if model != "" {
				clientOpts = append(clientOpts, search.WithModelName(model))
			}

			if tl := cmd.String("thinking-level"); tl != "" {
				level, err := parseThinkingLevel(tl)
				if err != nil {
					return cli.Exit(err.Error(), 1)
				}
				clientOpts = append(clientOpts, search.WithDefaultThinkingConfig(&search.ThinkingConfig{
					ThinkingLevel: level,
				}))
			}

			client, err := search.NewClient(ctx, apiKey, clientOpts...)
			if err != nil {
				return cli.Exit(fmt.Sprintf("Failed to create client: %v", err), 1)
			}

			startNow := time.Now()
			if cmd.Bool("verbose") {
				log.Printf("API Key: %s****%s", apiKey[:4], apiKey[len(apiKey)-4:])
				log.Printf("Using model: %s", model)
				log.Printf("Search query: %s", query)
				if tl := cmd.String("thinking-level"); tl != "" {
					log.Printf("Thinking level: %s", strings.ToUpper(tl))
				}
			}

			resp, err := client.GenerateGroundedContent(ctx, query)
			if err != nil {
				return cli.Exit(fmt.Sprintf("Search failed: %v", err), 1)
			}

			finishNow := time.Now()

			fmt.Println(resp.GeneratedText)
			if len(resp.GroundingAttributions) > 0 {
				fmt.Println("\n---\nSources:")
				for _, attr := range resp.GroundingAttributions {
					fmt.Printf("- %s (%s)\n", attr.Title, attr.URL)
				}
			}

			if cmd.Bool("verbose") {
				log.Printf("\n=========\nSearch completed in %s\n", finishNow.Sub(startNow))
			}

			return nil
		},
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
