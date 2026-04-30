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

	"hop.top/aps/internal/adapters"
	"hop.top/aps/internal/core/protocol"
	kitapi "hop.top/kit/go/transport/api"

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
	RunE:  runServe,
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

// buildServerHandler wires the kit Router with the health endpoint and
// mounts adapter routes. The authToken parameter is reserved for the
// auth middleware wired in T-0349.
func buildServerHandler(core protocol.APSCore, _ string) (http.Handler, error) {
	router := kitapi.NewRouter()

	router.Handle(http.MethodGet, "/health", func(w http.ResponseWriter, r *http.Request) {
		kitapi.JSON(w, http.StatusOK, map[string]string{"status": "healthy"})
	})

	// Adapter routes still register against an http.ServeMux. Mount it
	// under "/" so the kit Router delegates non-/health paths to it.
	adapterMux := http.NewServeMux()
	if err := adapters.GetRegistry().RegisterAll(adapterMux, core); err != nil {
		return nil, fmt.Errorf("registering adapter routes: %w", err)
	}
	router.Mount("/", adapterMux)

	return router, nil
}

func runServe(cmd *cobra.Command, args []string) error {
	if err := adapters.RegisterDefaults(); err != nil {
		return fmt.Errorf("registering adapters: %w", err)
	}

	core, err := protocol.NewAPSAdapter()
	if err != nil {
		return fmt.Errorf("creating core adapter: %w", err)
	}

	handler, err := buildServerHandler(core, serveAuthToken)
	if err != nil {
		return err
	}

	if serveAuthToken != "" {
		handler = authMiddleware(handler, serveAuthToken)
	}

	listener, err := net.Listen("tcp", serveAddr)
	if err != nil {
		return fmt.Errorf("listening on %s: %w", serveAddr, err)
	}

	log.Printf("Protocol server starting on %s", serveAddr)
	log.Printf("Health check: http://%s/health", serveAddr)
	log.Printf("Press Ctrl+C to stop")

	server := &http.Server{
		Handler:      handler,
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
	return nil
}

func authMiddleware(next http.Handler, token string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			next.ServeHTTP(w, r)
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

		next.ServeHTTP(w, r)
	}
}
