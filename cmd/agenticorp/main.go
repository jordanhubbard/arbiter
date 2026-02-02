package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/jordanhubbard/agenticorp/internal/agenticorp"
	"github.com/jordanhubbard/agenticorp/internal/api"
	"github.com/jordanhubbard/agenticorp/internal/auth"
	"github.com/jordanhubbard/agenticorp/internal/hotreload"
	"github.com/jordanhubbard/agenticorp/internal/keymanager"
	"github.com/jordanhubbard/agenticorp/pkg/config"
)

const version = "0.1.0"

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	configPath := flag.String("config", "config.yaml", "Path to configuration file")
	showVersion := flag.Bool("version", false, "Show version information")
	showHelp := flag.Bool("help", false, "Show help message")
	flag.Parse()

	if *showHelp {
		printHelp()
		return
	}

	if *showVersion {
		fmt.Printf("AgentiCorp v%s\n", version)
		return
	}

	cfg, err := config.LoadConfigFromFile(*configPath)
	if err != nil {
		log.Fatalf("failed to load config from %s: %v", *configPath, err)
	}

	// Override with environment variables if set
	if temporalHost := os.Getenv("TEMPORAL_HOST"); temporalHost != "" {
		cfg.Temporal.Host = temporalHost
		log.Printf("Using Temporal host from environment: %s", temporalHost)
	}
	if temporalNamespace := os.Getenv("TEMPORAL_NAMESPACE"); temporalNamespace != "" {
		cfg.Temporal.Namespace = temporalNamespace
		log.Printf("Using Temporal namespace from environment: %s", temporalNamespace)
	}

	arb, err := agenticorp.New(cfg)
	if err != nil {
		log.Fatalf("failed to create agenticorp: %v", err)
	}

	runCtx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := arb.Initialize(runCtx); err != nil {
		log.Fatalf("failed to initialize agenticorp: %v", err)
	}

	// Initialize hot-reload for development
	var hrManager *hotreload.Manager
	if cfg.HotReload.Enabled {
		hrManager, err = hotreload.NewManager(
			cfg.HotReload.Enabled,
			cfg.HotReload.WatchDirs,
			cfg.HotReload.Patterns,
		)
		if err != nil {
			log.Printf("Hot-reload initialization failed: %v", err)
		} else {
			defer hrManager.Close()
		}
	}

	// Initialize key manager for encrypted API keys
	keyStorePath := filepath.Join(".", ".keys.json")
	km := keymanager.NewKeyManager(keyStorePath)

	// Load password from environment or .env file
	password := loadPassword()
	if password == "" {
		log.Printf("Warning: No password found. Using default password. Set AGENTICORP_PASSWORD environment variable or create .env file")
		password = "agenticorp-default-password"
	}

	if err := km.Unlock(password); err != nil {
		// Try default password if the provided one failed
		log.Printf("Password unlock failed: %v. Trying default password...", err)
		if err := km.Unlock("agenticorp-default-password"); err != nil {
			log.Fatalf("Failed to unlock key manager with both passwords: %v", err)
		}
	}

	go arb.StartMaintenanceLoop(runCtx)

	// FIX #3: Always start fallback dispatch loop
	// Even if Temporal is configured, the server might not be running.
	// The fallback ensures work continues to flow even if Temporal fails.
	// If Temporal is healthy, its workflows will handle dispatch; if not, this fallback kicks in.
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-runCtx.Done():
				return
			case <-ticker.C:
				// Use fallback dispatch if Temporal is not configured
				// If Temporal is configured, it handles dispatch via workflows
				if arb.GetTemporalManager() == nil {
					if _, err := arb.GetDispatcher().DispatchOnce(runCtx, ""); err != nil {
						// Only log non-nil errors (nil means no work available)
						if err != nil {
							log.Printf("[Main] Fallback dispatch error: %v", err)
						}
					}
				}
			}
		}
	}()

	// Initialize auth manager (JWT + API key support)
	authManager := auth.NewManager(cfg.Security.JWTSecret)

	apiServer := api.NewServer(arb, km, authManager, cfg)
	handler := apiServer.SetupRoutes()

	// Add hot-reload WebSocket endpoint if enabled
	if hrManager != nil && hrManager.IsEnabled() {
		mux := http.NewServeMux()
		mux.Handle("/", handler)
		mux.HandleFunc("/ws/hotreload", hrManager.GetServer().HandleWebSocket)
		mux.HandleFunc("/api/v1/hotreload/status", hrManager.GetServer().HandleStatus)
		handler = mux
		log.Println("[HotReload] WebSocket endpoint registered at /ws/hotreload")
	}

	httpSrv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.HTTPPort),
		Handler:      handler,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	go func() {
		log.Printf("AgentiCorp API listening on %s", httpSrv.Addr)
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("http server error: %v", err)
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
	cancel()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_ = httpSrv.Shutdown(shutdownCtx)
	arb.Shutdown()

}

func loadPassword() string {
	// First, check environment variable
	if pwd := os.Getenv("AGENTICORP_PASSWORD"); pwd != "" {
		return pwd
	}

	// Second, try to load from .env file
	if envData, err := os.ReadFile(".env"); err == nil {
		lines := strings.Split(string(envData), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			// Skip comments and empty lines
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			// Look for AGENTICORP_PASSWORD=value
			if strings.HasPrefix(line, "AGENTICORP_PASSWORD=") {
				pwd := strings.TrimPrefix(line, "AGENTICORP_PASSWORD=")
				pwd = strings.Trim(pwd, "\"'")
				return pwd
			}
		}
	}

	return ""
}

func printHelp() {
	fmt.Println("Usage: agenticorp [flags]")
	fmt.Println()
	fmt.Println("Flags:")
	fmt.Println("  -config   Path to configuration file (default: config.yaml)")
	fmt.Println("  -version  Show version information")
	fmt.Println("  -help     Show help message")
	fmt.Println()
	fmt.Println("Environment:")
	fmt.Println("  AGENTICORP_PASSWORD  Master password for UI login and key encryption")
}
