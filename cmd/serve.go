/*
Copyright © 2025 Semen Adamenko <semaadamenko1@gmail.com>
*/
package cmd

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/paniccaaa/tsunami/cmd/server"
	"github.com/paniccaaa/tsunami/frontend"
	"github.com/spf13/cobra"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start HTTP server for web interface",
	Long: `Start an HTTP server that provides a REST API and WebSocket endpoint
for controlling load tests via a web interface.

The server exposes the following endpoints:
  POST /api/attack/start     - Start a new load test
  POST /api/attack/stop      - Stop the current test
  GET  /api/attack/status    - Get current test status and metrics
  GET  /api/attack/results   - Get final results (after test completes)
  GET  /api/attack/results/download - Download results as JSON file
  WS   /ws/metrics           - WebSocket for real-time metrics streaming

Examples:
  # Start server on default port 8080
  tsunami serve

  # Start server on custom port
  tsunami serve --port 3000`,
	Run: func(cmd *cobra.Command, args []string) {
		port, _ := cmd.Flags().GetInt("port")

		// Create session manager and WebSocket hub
		sessionManager := server.NewSessionManager()
		wsHub := server.NewHub()

		// Start WebSocket hub
		go wsHub.Run()

		// Create handlers
		handlers := server.NewHandlers(sessionManager, wsHub)

		// Setup routes
		mux := http.NewServeMux()

		// API routes
		mux.HandleFunc("/api/attack/start", handlers.HandleStartAttack)
		mux.HandleFunc("/api/attack/stop", handlers.HandleStopAttack)
		mux.HandleFunc("/api/attack/status", handlers.HandleStatus)
		mux.HandleFunc("/api/attack/results", handlers.HandleResults)
		mux.HandleFunc("/api/attack/results/download", handlers.HandleDownload)

		// WebSocket route
		mux.HandleFunc("/ws/metrics", server.HandleWebSocket(wsHub))

		// Health check
		mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
		})

		// Determine the main handler based on whether frontend is embedded
		var mainHandler http.Handler

		if frontend.IsEmbedded() {
			// Serve embedded frontend
			staticFS, err := frontend.GetFS()
			if err != nil {
				log.Fatalf("Failed to load embedded frontend: %v", err)
			}
			spaHandler := server.NewSPAHandler(staticFS)

			// Create main handler that routes API/WS to mux and everything else to SPA
			mainHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				path := r.URL.Path
				if strings.HasPrefix(path, "/api/") || strings.HasPrefix(path, "/ws/") || path == "/health" {
					mux.ServeHTTP(w, r)
					return
				}
				spaHandler.ServeHTTP(w, r)
			})
		} else {
			// No embedded frontend - API only mode
			log.Println("Note: Web UI not available (built without embedded frontend)")
			log.Println("Use 'brew install paniccaaa/tap/tsunami' for full version with web UI")
			mainHandler = mux
		}

		// Apply CORS middleware
		handler := server.CORSMiddleware(mainHandler)

		// Create server
		srv := &http.Server{
			Addr:         fmt.Sprintf(":%d", port),
			Handler:      handler,
			ReadTimeout:  15 * time.Second,
			WriteTimeout: 15 * time.Second,
			IdleTimeout:  60 * time.Second,
		}

		// Start server in goroutine
		go func() {
			log.Printf("Starting Tsunami HTTP server on http://localhost:%d", port)
			log.Printf("WebSocket endpoint: ws://localhost:%d/ws/metrics", port)
			log.Println("Press Ctrl+C to stop")

			if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Fatalf("Server error: %v", err)
			}
		}()

		// Wait for interrupt signal
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
		<-sigCh

		log.Println("\nShutting down server...")

		// Graceful shutdown with timeout
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := srv.Shutdown(ctx); err != nil {
			log.Printf("Server shutdown error: %v", err)
		}

		log.Println("Server stopped")
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)
	serveCmd.Flags().IntP("port", "p", 8080, "Port to run the HTTP server on")
}
