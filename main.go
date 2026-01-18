package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/jordanhubbard/arbiter/pkg/config"
	"github.com/jordanhubbard/arbiter/pkg/server"
)

func main() {
	fmt.Println("Welcome to Arbiter - AI Coding Agent Orchestrator")
	fmt.Println("==================================================")

	// Load or create configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Printf("Creating new configuration: %v", err)
		cfg = config.NewConfig()
	}

	// If no providers configured, run interactive setup
	if len(cfg.Providers) == 0 {
		fmt.Println("\nNo providers configured. Let's set up your AI providers.")
		if err := setupProviders(cfg); err != nil {
			log.Fatalf("Failed to setup providers: %v", err)
		}
	}

	// Save configuration
	if err := config.SaveConfig(cfg); err != nil {
		log.Fatalf("Failed to save configuration: %v", err)
	}

	// Start the server
	fmt.Println("\nStarting Arbiter server...")
	srv := server.NewServer(cfg)
	if err := srv.Start(); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func setupProviders(cfg *config.Config) error {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("\nWhich AI providers do you have access to?")
	fmt.Println("Enter provider names separated by commas (e.g., claude, openai, cursor):")
	fmt.Print("> ")

	input, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read input: %w", err)
	}

	providerNames := strings.Split(strings.TrimSpace(input), ",")
	for _, name := range providerNames {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}

		fmt.Printf("\nConfiguring provider: %s\n", name)

		// Ask if they want to specify endpoint or look it up
		fmt.Println("Do you have a specific API endpoint URL? (y/n)")
		fmt.Print("> ")
		hasEndpoint, _ := reader.ReadString('\n')
		hasEndpoint = strings.TrimSpace(strings.ToLower(hasEndpoint))

		var endpoint string
		if hasEndpoint == "y" || hasEndpoint == "yes" {
			fmt.Println("Enter the API endpoint URL:")
			fmt.Print("> ")
			endpoint, err = reader.ReadString('\n')
			if err != nil {
				return fmt.Errorf("failed to read endpoint: %w", err)
			}
			endpoint = strings.TrimSpace(endpoint)
		} else {
			// Look up endpoint using Google API
			fmt.Printf("Looking up endpoint for '%s' via Google API...\n", name)
			endpoint, err = config.LookupProviderEndpoint(name)
			if err != nil {
				log.Printf("Warning: Could not auto-lookup endpoint for %s: %v", name, err)
				fmt.Println("Please enter the API endpoint URL manually:")
				fmt.Print("> ")
				endpoint, err = reader.ReadString('\n')
				if err != nil {
					return fmt.Errorf("failed to read endpoint: %w", err)
				}
				endpoint = strings.TrimSpace(endpoint)
			} else {
				fmt.Printf("Found endpoint: %s\n", endpoint)
			}
		}

		// Ask for API key
		fmt.Printf("Enter your API key for %s:\n", name)
		fmt.Print("> ")
		apiKey, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read API key: %w", err)
		}
		apiKey = strings.TrimSpace(apiKey)

		// Add provider to configuration
		provider := config.Provider{
			Name:     name,
			Endpoint: endpoint,
		}
		cfg.Providers = append(cfg.Providers, provider)

		// Store API key securely
		if err := cfg.SecretStore.Set(name, apiKey); err != nil {
			return fmt.Errorf("failed to store API key for %s: %w", name, err)
		}

		fmt.Printf("✓ Provider %s configured successfully\n", name)
	}

	return nil
}
