package server

import (
	"fmt"
	"log"
	"net/http"

	"github.com/jordanhubbard/loom/pkg/config"
)

// Server represents the Loom HTTP server
type Server struct {
	config *config.Config
}

// NewServer creates a new server instance
func NewServer(cfg *config.Config) *Server {
	return &Server{
		config: cfg,
	}
}

// Start starts the HTTP server
func (s *Server) Start() error {
	addr := fmt.Sprintf(":%d", s.config.Server.HTTPPort)

	log.Printf("Loom server starting on %s", addr)
	log.Println("Note: This is a stub server. Full server implementation pending.")
	log.Println("The worker system is available via the WorkerManager API.")

	// Simple health check endpoint
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok","message":"Loom worker system is ready"}`))
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		html := `
		<html>
		<head><title>Loom - Worker System</title></head>
		<body>
			<h1>Loom Agent Worker System</h1>
			<p>The worker system is operational.</p>
			<p>See <code>docs/WORKER_SYSTEM.md</code> for usage information.</p>
			<h2>Endpoints</h2>
			<ul>
				<li><a href="/health">/health</a> - Health check</li>
			</ul>
		</body>
		</html>
		`
		_, _ = w.Write([]byte(html))
	})

	return http.ListenAndServe(addr, nil)
}
