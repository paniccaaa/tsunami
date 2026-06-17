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

		sessionManager := server.NewSessionManager()
		wsHub := server.NewHub()

		go wsHub.Run()

		handlers := server.NewHandlers(sessionManager, wsHub)

		mux := http.NewServeMux()

		mux.HandleFunc("/api/attack/start", handlers.HandleStartAttack)
		mux.HandleFunc("/api/attack/stop", handlers.HandleStopAttack)
		mux.HandleFunc("/api/attack/status", handlers.HandleStatus)
		mux.HandleFunc("/api/attack/results", handlers.HandleResults)
		mux.HandleFunc("/api/attack/results/download", handlers.HandleDownload)
		mux.HandleFunc("/api/proto/upload", handlers.HandleProtoUpload)

		mux.HandleFunc("/ws/metrics", server.HandleWebSocket(wsHub))

		mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
		})

		var mainHandler http.Handler

		if frontend.IsEmbedded() {
			staticFS, err := frontend.GetFS()
			if err != nil {
				log.Fatalf("Failed to load embedded frontend: %v", err)
			}
			spaHandler := server.NewSPAHandler(staticFS)

			mainHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				path := r.URL.Path
				if strings.HasPrefix(path, "/api/") || strings.HasPrefix(path, "/ws/") || path == "/health" {
					mux.ServeHTTP(w, r)
					return
				}
				spaHandler.ServeHTTP(w, r)
			})
		} else {
			log.Println("Note: Web UI not available (built without embedded frontend)")
			log.Println("Use 'brew install paniccaaa/tap/tsunami' for full version with web UI")
			mainHandler = mux
		}

		handler := server.CORSMiddleware(mainHandler)

		srv := &http.Server{
			Addr:         fmt.Sprintf(":%d", port),
			Handler:      handler,
			ReadTimeout:  15 * time.Second,
			WriteTimeout: 15 * time.Second,
			IdleTimeout:  60 * time.Second,
		}

		go func() {
			log.Printf("Starting Tsunami HTTP server on http://localhost:%d", port)
			log.Printf("WebSocket endpoint: ws://localhost:%d/ws/metrics", port)
			log.Println("Press Ctrl+C to stop")

			if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Fatalf("Server error: %v", err)
			}
		}()

		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
		<-sigCh

		log.Println("\nShutting down server...")

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
