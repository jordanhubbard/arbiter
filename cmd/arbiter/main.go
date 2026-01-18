package main

import (
"context"
"flag"
"fmt"
"log"
"net/http"
"os"
"os/signal"
"syscall"
"time"

"github.com/jordanhubbard/arbiter/internal/api"
"github.com/jordanhubbard/arbiter/internal/arbiter"
"github.com/jordanhubbard/arbiter/pkg/config"
	"fmt"
	"log"
	"os"

	"github.com/jordanhubbard/arbiter/internal/config"
	"github.com/jordanhubbard/arbiter/internal/database"
	"github.com/jordanhubbard/arbiter/internal/keymanager"
	"github.com/jordanhubbard/arbiter/internal/models"
)

const version = "0.1.0"

func main() {
log.SetFlags(log.LstdFlags | log.Lshortfile)

// Parse command line flags
configPath := flag.String("config", "config.yaml", "Path to configuration file")
showVersion := flag.Bool("version", false, "Show version information")
showHelp := flag.Bool("help", false, "Show help message")
flag.Parse()

if *showVersion {
fmt.Printf("Arbiter v%s\n", version)
return
}

if *showHelp {
fmt.Printf("Arbiter v%s - Agentic Coding Orchestrator\n\n", version)
fmt.Println("Usage: arbiter [options]")
fmt.Println("\nOptions:")
flag.PrintDefaults()
return
// Arbiter is the main orchestrator that manages agents and providers
type Arbiter struct {
	db         *database.Database
	keyManager *keymanager.KeyManager
	config     *config.Config
}

// NewArbiter creates a new arbiter instance
func NewArbiter(cfg *config.Config) (*Arbiter, error) {
	// Initialize database
	db, err := database.New(cfg.DatabasePath)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	// Initialize key manager
	km := keymanager.NewKeyManager(cfg.KeyStorePath)

	// Get password and unlock key store
	password, err := config.GetPassword()
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to get password: %w", err)
	}

	if err := km.Unlock(password); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to unlock key store: %w", err)
	}

	log.Println("Arbiter initialized successfully")
	log.Printf("Database: %s", cfg.DatabasePath)
	log.Printf("Key Store: %s", cfg.KeyStorePath)

	return &Arbiter{
		db:         db,
		keyManager: km,
		config:     cfg,
	}, nil
}

// Close cleans up arbiter resources
func (a *Arbiter) Close() error {
	// Lock the key manager to clear sensitive data
	a.keyManager.Lock()

	// Close database
	if err := a.db.Close(); err != nil {
		return fmt.Errorf("failed to close database: %w", err)
	}

	return nil
}

// CreateProvider creates a new provider with optional credentials
func (a *Arbiter) CreateProvider(provider *models.Provider, apiKey string) error {
	// If provider requires a key and one is provided, store it
	if provider.RequiresKey && apiKey != "" {
		keyID := fmt.Sprintf("key_%s", provider.ID)
		if err := a.keyManager.StoreKey(keyID, provider.Name, "API Key for "+provider.Name, apiKey); err != nil {
			return fmt.Errorf("failed to store provider key: %w", err)
		}
		provider.KeyID = keyID
	}

	// Create provider in database
	if err := a.db.CreateProvider(provider); err != nil {
		return fmt.Errorf("failed to create provider: %w", err)
	}

	log.Printf("Created provider: %s (%s)", provider.Name, provider.ID)
	return nil
}

fmt.Printf("Arbiter v%s - Agentic Coding Orchestrator\n", version)
fmt.Println("An agentic based coding orchestrator for both on-prem and off-prem development")
fmt.Println()

// Load configuration
cfg, err := config.LoadConfig(*configPath)
if err != nil {
log.Fatalf("Failed to load configuration: %v", err)
}

// Create arbiter instance
arb := arbiter.New(cfg)

// Initialize arbiter
ctx := context.Background()
if err := arb.Initialize(ctx); err != nil {
log.Fatalf("Failed to initialize arbiter: %v", err)
}

log.Println("Arbiter initialized successfully")

// Start maintenance loop in background
go arb.StartMaintenanceLoop(ctx)

// Create API server
apiServer := api.NewServer(arb, cfg)
handler := apiServer.SetupRoutes()

// Start HTTP server
httpAddr := fmt.Sprintf(":%d", cfg.Server.HTTPPort)
server := &http.Server{
Addr:         httpAddr,
Handler:      handler,
ReadTimeout:  cfg.Server.ReadTimeout,
WriteTimeout: cfg.Server.WriteTimeout,
IdleTimeout:  cfg.Server.IdleTimeout,
}

// Start server in goroutine
go func() {
log.Printf("Starting HTTP server on %s", httpAddr)
if cfg.WebUI.Enabled {
log.Printf("Web UI available at http://localhost%s", httpAddr)
}
if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
log.Fatalf("HTTP server failed: %v", err)
}
}()

// Setup graceful shutdown
stop := make(chan os.Signal, 1)
signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

<-stop
log.Println("Shutting down server...")

shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()

if err := server.Shutdown(shutdownCtx); err != nil {
log.Printf("Server shutdown error: %v", err)
}

log.Println("Server stopped")
func main() {
	// Get default configuration
	cfg, err := config.Default()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Create arbiter instance
	arbiter, err := NewArbiter(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize arbiter: %v", err)
	}
	defer arbiter.Close()

	// Example: Create a sample provider
	provider := &models.Provider{
		ID:          "openai-gpt4",
		Name:        "OpenAI GPT-4",
		Type:        "openai",
		Endpoint:    "https://api.openai.com/v1",
		Description: "OpenAI GPT-4 API",
		RequiresKey: true,
		Status:      "active",
	}

	// Note: In real usage, the API key would be provided by the user
	// For this example, we skip creating the provider if no key is provided
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey != "" {
		if err := arbiter.CreateProvider(provider, apiKey); err != nil {
			log.Printf("Note: Could not create example provider: %v", err)
		}
	}

	// List all providers
	providers, err := arbiter.ListProviders()
	if err != nil {
		log.Fatalf("Failed to list providers: %v", err)
	}

	log.Printf("Total providers: %d", len(providers))
	for _, p := range providers {
		log.Printf("  - %s (%s): %s", p.Name, p.Type, p.Status)
	}

	log.Println("Arbiter is ready to orchestrate agents and providers")
func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	fmt.Printf("Arbiter v%s - Agentic Coding Orchestrator\n", version)
	fmt.Println("An agentic based coding orchestrator for both on-prem and off-prem development")
	fmt.Println()

	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "version", "--version", "-v":
			fmt.Printf("Version: %s\n", version)
		case "help", "--help", "-h":
			printHelp()
		default:
			fmt.Printf("Unknown command: %s\n", os.Args[1])
			printHelp()
			os.Exit(1)
		}
	} else {
		fmt.Println("Starting arbiter service...")
		fmt.Println("Ready to orchestrate coding tasks")
		// Main service loop would go here
	}
}

func printHelp() {
	fmt.Println("Usage: arbiter [command]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  version    Display version information")
	fmt.Println("  help       Display this help message")
	fmt.Println()
	fmt.Println("When run without commands, arbiter starts in service mode")
}
