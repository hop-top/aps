package cli

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"hop.top/upgrade"
)

func TestNewChecker_ReturnsChecker(t *testing.T) {
	c := newChecker()
	if c == nil {
		t.Fatal("newChecker returned nil")
	}
}

func TestUpgradeRunCLI_NoUpdate(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{ //nolint:errcheck
			"version": "0.0.0",
			"url":     "http://example.com/aps",
		})
	}))
	defer srv.Close()

	c := upgrade.New(
		upgrade.WithBinary("aps", "99.0.0"),
		upgrade.WithReleaseURL(srv.URL),
		upgrade.WithStateDir(t.TempDir()),
	)

	var out strings.Builder
	if err := upgrade.RunCLI(context.Background(), c, upgrade.CLIOptions{Quiet: true, Out: &out}); err != nil {
		t.Fatal(err)
	}
}

func TestUpgrade_UpdateAvail(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{ //nolint:errcheck
			"version": "99.0.0",
			"url":     "http://example.com/aps",
			"notes":   "Many improvements",
		})
	}))
	defer srv.Close()

	c := upgrade.New(
		upgrade.WithBinary("aps", "1.0.0"),
		upgrade.WithReleaseURL(srv.URL),
		upgrade.WithStateDir(t.TempDir()),
	)

	r := c.Check(context.Background())
	if r.Err != nil {
		t.Fatal(r.Err)
	}
	if !r.UpdateAvail {
		t.Error("expected update available")
	}
}
