package cli

import (
	"context"
	"errors"
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

// buildServerHandler wires the kit Router with Recovery + RequestID
// middleware globally, registers /health, and mounts adapter routes.
// When authToken is non-empty, kitapi.Auth gates the adapter routes
// (but not /health, which always answers without credentials so that
// liveness probes work uniformly).
func buildServerHandler(mgr *adapters.Manager, core protocol.APSCore, authToken string) (http.Handler, error) {
	router := kitapi.NewRouter(kitapi.WithMiddleware(
		kitapi.Recovery(nil),
		kitapi.RequestID(),
	))

	router.Handle(http.MethodGet, "/health", func(w http.ResponseWriter, r *http.Request) {
		kitapi.JSON(w, http.StatusOK, map[string]string{"status": "healthy"})
	})

	adapterMux := http.NewServeMux()
	if err := mgr.RegisterRoutes(adapterMux, core); err != nil {
		return nil, fmt.Errorf("registering adapter routes: %w", err)
	}

	var protected http.Handler = adapterMux
	if authToken != "" {
		protected = kitapi.Auth(bearerTokenAuth(authToken))(adapterMux)
	}
	router.Mount("/", protected)

	return router, nil
}

// bearerTokenAuth returns a kitapi.AuthFunc that validates a static
// bearer token from the Authorization header.
func bearerTokenAuth(token string) kitapi.AuthFunc {
	const bearerPrefix = "Bearer "
	return func(r *http.Request) (any, error) {
		h := r.Header.Get("Authorization")
		if h == "" {
			return nil, errors.New("missing authorization header")
		}
		if len(h) < len(bearerPrefix) || h[:len(bearerPrefix)] != bearerPrefix {
			return nil, errors.New("invalid authorization header format")
		}
		if h[len(bearerPrefix):] != token {
			return nil, errors.New("invalid token")
		}
		return struct{}{}, nil
	}
}

func runServe(cmd *cobra.Command, args []string) error {
	mgr := adapters.DefaultManager()
	if err := mgr.InitAll(cmd.Context()); err != nil {
		return fmt.Errorf("initialising adapters: %w", err)
	}
	defer func() {
		if errs := mgr.CloseAll(); len(errs) != 0 {
			log.Printf("adapter close errors: %v", errs)
		}
	}()

	core, err := protocol.NewAPSAdapter()
	if err != nil {
		return fmt.Errorf("creating core adapter: %w", err)
	}

	handler, err := buildServerHandler(mgr, core, serveAuthToken)
	if err != nil {
		return err
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
