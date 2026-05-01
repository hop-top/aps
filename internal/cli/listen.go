// Package cli — `aps listen` long-running daemon.
//
// Subscribes to configured bus topic patterns for a profile and prints
// every matching event as JSONL on stdout. This is the minimal
// subscribe-and-print form (story 051); routing config (story 052) and
// per-event handler dispatch land in T-0152.
//
// Shutdown is handled by the existing drainBus pattern wired in
// root.go (defer drainBus()): SIGTERM/SIGINT cancels the listener
// context, the cobra Execute returns, drainBus closes eventBus +
// netAdapter in the order documented in bus.go (T-0176).
package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"hop.top/aps/internal/core"
	"hop.top/kit/go/runtime/bus"
)

// listenLine is the JSONL record written to stdout per delivered event.
// Field names mirror the kit/bus wire format so consumers can pipe
// directly into anything that already speaks bus events.
type listenLine struct {
	Topic     string    `json:"topic"`
	Source    string    `json:"source"`
	Timestamp time.Time `json:"timestamp"`
	Payload   any       `json:"payload"`
	Profile   string    `json:"profile"`
}

var listenCmd = &cobra.Command{
	Use:   "listen",
	Short: "Subscribe to bus topics for a profile and print events as JSONL",
	Long: `Long-running daemon that subscribes to configured bus topic patterns
for the given profile and prints every matching event as JSONL on stdout.

Honors SIGTERM/SIGINT for graceful shutdown — in-flight async forwarders
flush before exit (see drainBus in internal/cli/bus.go).

Examples:
  aps listen --profile noor
  aps listen --profile noor --topics "aps.#,tlc.task.*"
  aps listen --profile noor --exit-after-events 5`,
	RunE: runListen,
}

func init() {
	rootCmd.AddCommand(listenCmd)
	listenCmd.Flags().String("topics", "aps.#,tlc.#",
		"Comma-separated topic patterns to subscribe to (kit/bus glob: * = one segment, # = trailing)")
	listenCmd.Flags().Int("exit-after-events", 0,
		"Exit cleanly after N events received (0 = run until signal)")
}

func runListen(cmd *cobra.Command, _ []string) error {
	profileID := root.Viper.GetString("profile")
	if profileID == "" {
		return fmt.Errorf("--profile is required")
	}

	if _, err := core.LoadProfile(profileID); err != nil {
		return fmt.Errorf("loading profile %q: %w", profileID, err)
	}

	if eventBus == nil {
		return fmt.Errorf("bus disabled (set BUS_TOKEN or APS_BUS_TOKEN to enable subscribe)")
	}

	topicsCSV, _ := cmd.Flags().GetString("topics")
	patterns := splitTopics(topicsCSV)
	if len(patterns) == 0 {
		return fmt.Errorf("--topics produced no patterns from %q", topicsCSV)
	}

	exitAfter, _ := cmd.Flags().GetInt("exit-after-events")
	if exitAfter < 0 {
		return fmt.Errorf("--exit-after-events must be >= 0, got %d", exitAfter)
	}

	ctx, cancel := signal.NotifyContext(cmd.Context(), syscall.SIGTERM, syscall.SIGINT)
	defer cancel()

	return listen(ctx, cmd.OutOrStdout(), profileID, patterns, exitAfter)
}

// listen wires subscribers and blocks until ctx.Done or the configured
// event budget is exhausted. Stdout is the JSONL sink; stderr carries
// startup/shutdown breadcrumbs so callers can tee one stream without
// the other interleaving.
//
// The async path (SubscribeAsync) is intentional: the listener prints
// from a goroutine pool the bus already manages, so a slow stdout
// consumer cannot stall publish on the local bus or starve other
// subscribers.
func listen(
	ctx context.Context,
	stdout interface{ Write([]byte) (int, error) },
	profileID string,
	patterns []string,
	exitAfter int,
) error {
	enc := json.NewEncoder(stdout)

	var (
		writeMu sync.Mutex
		count   atomic.Int64
		done    = make(chan struct{}, 1)
	)

	signalDone := func() {
		select {
		case done <- struct{}{}:
		default:
		}
	}

	handler := func(_ context.Context, e bus.Event) {
		line := listenLine{
			Topic:     string(e.Topic),
			Source:    e.Source,
			Timestamp: e.Timestamp,
			Payload:   e.Payload,
			Profile:   profileID,
		}
		writeMu.Lock()
		err := enc.Encode(line)
		writeMu.Unlock()
		if err != nil {
			fmt.Fprintf(os.Stderr, "warn: listen encode (%s): %v\n", line.Topic, err)
			return
		}
		n := count.Add(1)
		if exitAfter > 0 && n >= int64(exitAfter) {
			signalDone()
		}
	}

	unsubs := make([]bus.Unsubscribe, 0, len(patterns))
	for _, p := range patterns {
		unsubs = append(unsubs, eventBus.SubscribeAsync(p, handler))
	}
	defer func() {
		for _, u := range unsubs {
			u()
		}
	}()

	fmt.Fprintf(os.Stderr, "aps listen: profile=%s topics=%s\n",
		profileID, strings.Join(patterns, ","))

	select {
	case <-ctx.Done():
		fmt.Fprintln(os.Stderr, "aps listen: shutdown signal received")
		return nil
	case <-done:
		fmt.Fprintf(os.Stderr, "aps listen: exit-after-events=%d reached\n", exitAfter)
		return nil
	}
}

// splitTopics parses a comma-separated topic list, trimming whitespace
// and dropping empty entries. Empty input returns nil so callers can
// treat "no patterns" as a hard error rather than subscribing to "".
func splitTopics(csv string) []string {
	parts := strings.Split(csv, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}
