package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"google.golang.org/genai"
)

func main() {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		log.Fatal("GEMINI_API_KEY environment variable is required")
	}

	ctx := context.Background()
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	fmt.Println("Available models:")
	fmt.Println("=================")

	for model, err := range client.Models.All(ctx) {
		if err != nil {
			log.Fatalf("Error listing models: %v", err)
		}

		fmt.Printf("\nModel: %s\n", model.Name)
		if model.DisplayName != "" {
			fmt.Printf("  Display Name: %s\n", model.DisplayName)
		}
		if model.Description != "" {
			fmt.Printf("  Description: %s\n", model.Description)
		}
		if len(model.SupportedActions) > 0 {
			fmt.Printf("  Supported Actions: %v\n", model.SupportedActions)
		}
	}
}
