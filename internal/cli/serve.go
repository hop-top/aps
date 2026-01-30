package cli

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"oss-aps-cli/internal/adapters"
	"oss-aps-cli/internal/core/protocol"

	"github.com/spf13/cobra"
)

var (
	serveAddr      string
	serveAuthToken string
	serveLogLevel  string

	TestServeAddr      = &serveAddr
	TestServeAuthToken = &serveAuthToken
	TestServeLogLevel  = &serveLogLevel
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start protocol server",
	Long:  `Start Agent Protocol server to expose APS functionality via HTTP API.`,
	Run:   runServe,
}

func init() {
	rootCmd.AddCommand(serveCmd)

	serveAddr = "127.0.0.1:8080"
	serveAuthToken = ""
	serveLogLevel = "info"

	serveCmd.Flags().StringVar(&serveAddr, "addr", "127.0.0.1:8080", "Address to listen on")
	serveCmd.Flags().StringVar(&serveAuthToken, "auth-token", "", "Bearer token for authentication (optional)")
	serveCmd.Flags().StringVar(&serveLogLevel, "log-level", "info", "Log level (debug, info, warn, error)")
}

func runServe(cmd *cobra.Command, args []string) {
	if err := adapters.RegisterDefaults(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to register adapters: %v\n", err)
		os.Exit(1)
	}

	core, err := protocol.NewAPSAdapter()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create core adapter: %v\n", err)
		os.Exit(1)
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status":"healthy"}`)
	})

	if err := adapters.GetRegistry().RegisterAll(mux, core); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to register adapter routes: %v\n", err)
		os.Exit(1)
	}

	if serveAuthToken != "" {
		mux.HandleFunc("/", authMiddleware(mux, serveAuthToken))
	}

	listener, err := net.Listen("tcp", serveAddr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to listen on %s: %v\n", serveAddr, err)
		os.Exit(1)
	}

	log.Printf("Protocol server starting on %s", serveAddr)
	log.Printf("Health check: http://%s/health", serveAddr)
	log.Printf("Press Ctrl+C to stop")

	server := &http.Server{
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			log.Printf("Server error: %v", err)
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Error during shutdown: %v", err)
	}

	log.Println("Server stopped")
}

func authMiddleware(mux *http.ServeMux, token string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			mux.ServeHTTP(w, r)
			return
		}

		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			w.Header().Set("WWW-Authenticate", `Bearer realm="APS Protocol Server"`)
			http.Error(w, "Unauthorized: Missing authorization header", http.StatusUnauthorized)
			return
		}

		const bearerPrefix = "Bearer "
		if len(authHeader) < len(bearerPrefix) || authHeader[:len(bearerPrefix)] != bearerPrefix {
			http.Error(w, "Unauthorized: Invalid authorization header format", http.StatusUnauthorized)
			return
		}

		providedToken := authHeader[len(bearerPrefix):]
		if providedToken != token {
			http.Error(w, "Unauthorized: Invalid token", http.StatusUnauthorized)
			return
		}

		mux.ServeHTTP(w, r)
	}
}
