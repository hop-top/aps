package adapter

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"oss-aps-cli/internal/core"
	"oss-aps-cli/internal/core/adapter/mobile"
	"oss-aps-cli/internal/styles"

	"github.com/spf13/cobra"
)

func newPairCmd() *cobra.Command {
	var (
		profileID    string
		expires      string
		qrExpires    string
		capabilities []string
		port         int
		bindAddr     string
		noQR         bool
		codeOnly     bool
		qrOutput     string
		jsonOutput   bool
		quiet        bool
	)

	cmd := &cobra.Command{
		Use:   "pair",
		Short: "Generate QR code to pair a mobile device",
		Long: `Start a device server and display a QR code for mobile device pairing.

The QR code contains connection details that the mobile APS app uses to
establish a secure WebSocket connection to this profile.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPair(cmd.Context(), profileID, expires, qrExpires, capabilities,
				port, bindAddr, noQR, codeOnly, qrOutput, jsonOutput, quiet)
		},
	}

	cmd.Flags().StringVarP(&profileID, "profile", "p", "", "Profile to pair with (required)")
	cmd.MarkFlagRequired("profile")
	cmd.Flags().StringVar(&expires, "expires", "14d", "Device token expiry (e.g., 14d, 30d)")
	cmd.Flags().StringVar(&qrExpires, "qr-expires", "15m", "QR code expiry (e.g., 15m, 30m)")
	cmd.Flags().StringSliceVar(&capabilities, "capabilities", nil, "Device capabilities (default: run:stateless,run:streaming,monitor:sessions)")
	cmd.Flags().IntVar(&port, "port", mobile.DefaultPort, "Server port")
	cmd.Flags().StringVar(&bindAddr, "bind-addr", "", "Bind address (auto-detected if not set)")
	cmd.Flags().BoolVar(&noQR, "no-qr", false, "Skip QR code display (accessibility, screen readers)")
	cmd.Flags().BoolVar(&codeOnly, "code-only", false, "Show pairing code only, no QR")
	cmd.Flags().StringVar(&qrOutput, "qr-output", "", "Save QR code as PNG to file")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "JSON output")
	cmd.Flags().BoolVar(&quiet, "quiet", false, "Minimal output (pairing code only)")

	return cmd
}

func runPair(ctx context.Context, profileID, expires, qrExpires string,
	capabilities []string, port int, bindAddr string,
	noQR, codeOnly bool, qrOutput string, jsonOut, quiet bool) error {

	// Load and validate profile
	profile, err := core.LoadProfile(profileID)
	if err != nil {
		return fmt.Errorf("failed to load profile: %w", err)
	}

	// Check mobile config
	if profile.Mobile == nil || !profile.Mobile.Enabled {
		fmt.Fprintf(os.Stderr, "%s\n\n", errorStyle.Render(
			fmt.Sprintf("Error: Mobile device linking not enabled for profile '%s'", profileID)))
		fmt.Fprintf(os.Stderr, "  Enable it:\n")
		fmt.Fprintf(os.Stderr, "    aps profile edit %s\n", profileID)
		fmt.Fprintf(os.Stderr, "    # Add: mobile.enabled = true\n\n")
		return fmt.Errorf("mobile not enabled")
	}

	// Apply profile config defaults
	if profile.Mobile.Port > 0 && port == mobile.DefaultPort {
		port = profile.Mobile.Port
	}
	maxDevices := 10
	if profile.Mobile.MaxDevices > 0 {
		maxDevices = profile.Mobile.MaxDevices
	}

	// Parse capabilities
	if len(capabilities) == 0 {
		caps := mobile.DefaultCapabilities()
		for _, c := range caps {
			capabilities = append(capabilities, string(c))
		}
	} else {
		for _, cap := range capabilities {
			if !mobile.IsValidCapability(cap) {
				return fmt.Errorf("invalid capability '%s'", cap)
			}
		}
	}

	// Parse durations
	tokenExpiry, err := parseDuration(expires)
	if err != nil {
		return fmt.Errorf("invalid --expires: %w", err)
	}
	qrExpiryDuration, err := parseDuration(qrExpires)
	if err != nil {
		return fmt.Errorf("invalid --qr-expires: %w", err)
	}

	// Setup key directory and token manager
	profileDir, err := core.GetProfileDir(profileID)
	if err != nil {
		return err
	}
	keyDir := profileDir + "/device-keys"

	tokenMgr, err := mobile.NewTokenManager(profileID, keyDir)
	if err != nil {
		return fmt.Errorf("failed to initialize token manager: %w", err)
	}

	// Setup registry
	home, _ := os.UserHomeDir()
	registryDir := home + "/.aps/devices"
	registry, err := mobile.NewRegistry(registryDir)
	if err != nil {
		return fmt.Errorf("failed to initialize registry: %w", err)
	}

	// Check max devices
	activeCount, err := registry.CountActive(profileID)
	if err != nil {
		return err
	}
	if activeCount >= maxDevices {
		fmt.Fprintf(os.Stderr, "%s\n\n", errorStyle.Render(
			fmt.Sprintf("Error: Maximum devices reached for profile '%s' (%d/%d)", profileID, maxDevices, maxDevices)))
		fmt.Fprintf(os.Stderr, "  Revoke a device first:\n")
		fmt.Fprintf(os.Stderr, "    aps device revoke <device-id> --profile=%s\n\n", profileID)
		return fmt.Errorf("max devices reached")
	}

	// Detect bind address
	if bindAddr == "" {
		bindAddr, err = mobile.DetectLANAddress()
		if err != nil {
			bindAddr = "127.0.0.1"
		}
	}

	// Generate pairing code
	pairingCode, err := mobile.DefaultPairingCode()
	if err != nil {
		return fmt.Errorf("failed to generate pairing code: %w", err)
	}

	// Get cert fingerprint
	certFingerprint, err := tokenMgr.CertFingerprint()
	if err != nil {
		certFingerprint = ""
	}

	// Build endpoint
	endpoint := fmt.Sprintf("https://%s:%d/aps/device/%s", bindAddr, port, profileID)

	// Create QR payload
	payload := mobile.NewQRPayload(profileID, endpoint, pairingCode, certFingerprint, capabilities, qrExpiryDuration)

	// Encode payload
	encoded, err := mobile.EncodePairingPayload(payload)
	if err != nil {
		return fmt.Errorf("failed to encode QR payload: %w", err)
	}

	// Start device server
	server := mobile.NewDeviceServer(profileID, registry, tokenMgr,
		mobile.WithApprovalRequired(profile.Mobile.ApprovalRequired),
		mobile.WithMaxDevices(maxDevices),
	)

	serverCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	listenAddr := fmt.Sprintf(":%d", port)
	if err := server.Start(serverCtx, listenAddr); err != nil {
		return err
	}
	defer server.Stop()

	// Register pairing code
	server.RegisterPairingCode(pairingCode, capabilities, qrExpiryDuration)

	// JSON output
	if jsonOut {
		return renderPairJSON(profileID, pairingCode, endpoint, capabilities, tokenExpiry, qrExpiryDuration)
	}

	// Quiet output
	if quiet {
		fmt.Println(pairingCode)
		return waitForPairing(serverCtx, cancel, server, registry, profileID, qrExpiryDuration)
	}

	// Save QR as PNG if requested
	if qrOutput != "" {
		if err := mobile.GenerateQRPNG(encoded, qrOutput, 256); err != nil {
			return fmt.Errorf("failed to save QR code: %w", err)
		}
		fmt.Printf("  QR code saved to: %s\n\n", qrOutput)
	}

	// Display QR code
	if !noQR && !codeOnly {
		// Check terminal size
		moduleCount, _ := mobile.QRModuleCount(encoded)
		fits, width, _ := mobile.TerminalFits(moduleCount)

		if !fits && width > 0 {
			fmt.Printf("\n  %s\n\n", warnStyle.Render(
				fmt.Sprintf("Terminal too narrow for QR display (need %d cols, have %d).", moduleCount+8, width)))
			fmt.Printf("  Alternatives:\n")
			fmt.Printf("    aps device pair --qr-output=qr.png   Save as image\n")
			fmt.Printf("    aps device pair --code-only           Show pairing code only\n\n")
		} else {
			qrTerminal, err := mobile.GenerateQRTerminal(encoded)
			if err == nil {
				fmt.Println()
				fmt.Printf("  %s\n\n", headerStyle.Render("Scan with APS mobile app"))
				fmt.Println(qrTerminal)
			}
		}
	}

	// Display pairing info
	fmt.Printf("  %s  %s\n", dimStyle.Render("Pairing code:"), styles.Bold.Render(pairingCode))
	fmt.Printf("  %s  %s\n", dimStyle.Render("Profile:"), profileID)
	fmt.Printf("  %s  %s\n", dimStyle.Render("Endpoint:"), endpoint)
	fmt.Printf("  %s  in %s\n", dimStyle.Render("QR expires:"), qrExpiryDuration)
	fmt.Printf("  %s  %s after pairing\n", dimStyle.Render("Device access:"), tokenExpiry)

	// Show network info
	ifaces, _ := mobile.ListNetworkInterfaces()
	if len(ifaces) > 1 {
		fmt.Printf("\n  %s\n", dimStyle.Render("Network interfaces:"))
		for _, iface := range ifaces {
			marker := " "
			if iface.IP == bindAddr {
				marker = "*"
			}
			fmt.Printf("    %s %-8s (%s)  %s\n", marker, iface.IP, iface.Name, dimStyle.Render(iface.Type))
		}
		fmt.Printf("\n  Using: %s\n", bindAddr)
		fmt.Printf("  Override with: --bind-addr=<ip>\n")
	}

	fmt.Printf("\n  Ensure your mobile device is on the same network.\n")
	fmt.Println()

	return waitForPairing(serverCtx, cancel, server, registry, profileID, qrExpiryDuration)
}

func waitForPairing(ctx context.Context, cancel context.CancelFunc, server *mobile.DeviceServer,
	registry *mobile.Registry, profileID string, qrExpiry time.Duration) error {

	// Signal handling
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	timer := time.NewTimer(qrExpiry)
	ticker := time.NewTicker(1 * time.Second)
	defer timer.Stop()
	defer ticker.Stop()

	fmt.Printf("  %s  (Ctrl+C to cancel)\n\n", dimStyle.Render("Waiting for device..."))

	for {
		select {
		case <-sigCh:
			fmt.Printf("\n  Cancelled.\n")
			cancel()
			return nil
		case <-timer.C:
			fmt.Printf("\n  %s\n\n", warnStyle.Render("QR code expired."))
			fmt.Printf("  No device scanned within %s.\n\n", qrExpiry)
			fmt.Printf("  To generate a new QR code:\n")
			fmt.Printf("    aps device pair --profile=%s\n\n", profileID)
			cancel()
			return nil
		case <-ticker.C:
			if server.ActiveConnections() > 0 {
				fmt.Printf("  %s\n\n", successStyle.Render("Device connected!"))
				// Keep server running until Ctrl+C
				<-sigCh
				fmt.Printf("\n  Shutting down...\n")
				cancel()
				return nil
			}
		case <-ctx.Done():
			return nil
		}
	}
}

func renderPairJSON(profileID, pairingCode, endpoint string, capabilities []string,
	tokenExpiry, qrExpiry time.Duration) error {
	data := map[string]any{
		"profile":      profileID,
		"pairing_code": pairingCode,
		"endpoint":     endpoint,
		"capabilities": capabilities,
		"token_expiry":  tokenExpiry.String(),
		"qr_expiry":    qrExpiry.String(),
	}
	out, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(out))
	return nil
}

func parseDuration(s string) (time.Duration, error) {
	// Support day notation: "14d" -> 14 * 24h
	if strings.HasSuffix(s, "d") {
		var days int
		if _, err := fmt.Sscanf(s, "%dd", &days); err == nil {
			return time.Duration(days) * 24 * time.Hour, nil
		}
	}
	return time.ParseDuration(s)
}
