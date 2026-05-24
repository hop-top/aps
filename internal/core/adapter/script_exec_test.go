package adapter

import (
	"strings"
	"testing"
)

func TestBuildScriptEnv_EnvPrefix(t *testing.T) {
	tests := []struct {
		name           string
		manifestPrefix string
		devicePrefix   string
		inputs         map[string]string
		wantPrefix     string
		wantKey        string
	}{
		{
			name:           "no prefix falls back to ADAPTER",
			manifestPrefix: "",
			devicePrefix:   "",
			inputs:         map[string]string{"foo": "bar"},
			wantPrefix:     "ADAPTER",
			wantKey:        "ADAPTER_FOO=bar",
		},
		{
			name:           "manifest prefix CAL",
			manifestPrefix: "CAL",
			inputs:         map[string]string{"event-id": "42"},
			wantPrefix:     "CAL",
			wantKey:        "CAL_EVENT_ID=42",
		},
		{
			name:           "manifest prefix EMAIL regression guard",
			manifestPrefix: "EMAIL",
			inputs:         map[string]string{"to": "u@example.com"},
			wantPrefix:     "EMAIL",
			wantKey:        "EMAIL_TO=u@example.com",
		},
		{
			name:           "manifest prefix CONTACT regression guard",
			manifestPrefix: "CONTACT",
			inputs:         map[string]string{"id": "abc"},
			wantPrefix:     "CONTACT",
			wantKey:        "CONTACT_ID=abc",
		},
		{
			name:           "lowercase manifest prefix uppercases",
			manifestPrefix: "cal",
			inputs:         map[string]string{"x": "y"},
			wantPrefix:     "CAL",
			wantKey:        "CAL_X=y",
		},
		{
			name:         "device prefix used when manifest empty",
			devicePrefix: "DEV",
			inputs:       map[string]string{"a": "b"},
			wantPrefix:   "DEV",
			wantKey:      "DEV_A=b",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			device := &Adapter{
				Name:      "test",
				EnvPrefix: tt.devicePrefix,
				Config:    map[string]any{},
			}
			manifest := &AdapterManifest{
				Name:      "test",
				EnvPrefix: tt.manifestPrefix,
			}

			env := buildScriptEnv(device, manifest, "", tt.inputs)

			var found bool
			for _, e := range env {
				if e == tt.wantKey {
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("env missing %q; got: %v", tt.wantKey, env)
			}

			// Negative: no input var should use a stale literal prefix
			// other than the wanted one.
			for _, e := range env {
				if !strings.Contains(e, "=") {
					continue
				}
				key := e[:strings.Index(e, "=")]
				if strings.HasPrefix(key, "APS_") {
					continue
				}
				if !strings.HasPrefix(key, tt.wantPrefix+"_") {
					t.Fatalf("env key %q does not match wanted prefix %q", key, tt.wantPrefix)
				}
			}
		})
	}
}

func TestResolveEnvPrefix_Precedence(t *testing.T) {
	device := &Adapter{EnvPrefix: "DEV"}
	manifest := &AdapterManifest{EnvPrefix: "MAN"}

	if got := resolveEnvPrefix(device, manifest); got != "MAN" {
		t.Fatalf("manifest should win; got %q", got)
	}
	if got := resolveEnvPrefix(device, &AdapterManifest{}); got != "DEV" {
		t.Fatalf("device fallback failed; got %q", got)
	}
	if got := resolveEnvPrefix(&Adapter{}, &AdapterManifest{}); got != DefaultEnvPrefix {
		t.Fatalf("default fallback failed; got %q", got)
	}
	if got := resolveEnvPrefix(nil, nil); got != DefaultEnvPrefix {
		t.Fatalf("nil-safe fallback failed; got %q", got)
	}
}
