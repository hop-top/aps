package directory

import (
	"errors"
	"fmt"
	"testing"

	"github.com/spf13/cobra"

	"hop.top/aps/internal/core"
)

// newRootWithDiscover builds a synthetic root with --instance + --offline
// persistent flags (mirrors what kit/cli installs in production) and the
// real discover subcommand attached. Tests parse args through this root.
func newRootWithDiscover() (*cobra.Command, *cobra.Command) {
	root := &cobra.Command{Use: "aps"}
	root.PersistentFlags().String("instance", "", "backend instance")
	root.PersistentFlags().Bool("offline", false, "disable network")
	disc := NewDiscoverCmd()
	root.AddCommand(disc)
	return root, disc
}

// stubResolver swaps core.Resolve for the duration of a test.
func stubResolver(t *testing.T, fn func(string) (*core.Instance, error)) {
	t.Helper()
	prev := resolveInstance
	resolveInstance = fn
	t.Cleanup(func() { resolveInstance = prev })
}

// captureClientFactory swaps discovery.NewClient... not feasible without
// touching the discovery package. Instead the wiring test captures the
// resolved endpoint by hooking the resolver itself: any call to Resolve
// is observed, and the test asserts the right name was looked up.

func TestDiscover_NoInstance_NoLookup(t *testing.T) {
	called := false
	stubResolver(t, func(name string) (*core.Instance, error) {
		called = true
		return nil, fmt.Errorf("should not be called")
	})

	root, _ := newRootWithDiscover()
	root.SetArgs([]string{"discover", "--capability", "x"})
	if err := root.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if called {
		t.Error("resolveInstance called when --instance was not set")
	}
}

func TestDiscover_ExplicitEndpoint_BypassesInstance(t *testing.T) {
	called := false
	stubResolver(t, func(name string) (*core.Instance, error) {
		called = true
		return &core.Instance{Name: name, DirectoryEndpoint: "https://from-instance"}, nil
	})

	root, _ := newRootWithDiscover()
	root.SetArgs([]string{
		"--instance", "prod",
		"discover", "--capability", "x",
		"--endpoint", "https://explicit.example.com",
	})
	if err := root.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if called {
		t.Error("resolveInstance called when --endpoint was explicit; --endpoint should win")
	}
}

func TestDiscover_InstanceResolved(t *testing.T) {
	got := ""
	stubResolver(t, func(name string) (*core.Instance, error) {
		got = name
		return &core.Instance{Name: name, DirectoryEndpoint: "https://dir.prod.example.com"}, nil
	})

	root, _ := newRootWithDiscover()
	root.SetArgs([]string{
		"--instance", "prod",
		"discover", "--capability", "x",
	})
	if err := root.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if got != "prod" {
		t.Errorf("resolveInstance got %q, want %q", got, "prod")
	}
}

func TestDiscover_InstanceResolveError(t *testing.T) {
	wantErr := errors.New("not found")
	stubResolver(t, func(name string) (*core.Instance, error) {
		return nil, wantErr
	})

	root, _ := newRootWithDiscover()
	root.SetArgs([]string{
		"--instance", "missing",
		"discover", "--capability", "x",
	})
	err := root.Execute()
	if err == nil {
		t.Fatal("Execute err = nil, want resolve failure to propagate")
	}
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want errors.Is(%v)", err, wantErr)
	}
}
