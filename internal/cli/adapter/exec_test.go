package adapter

import (
	"strings"
	"testing"
)

func TestParseInputsWellFormedPairs(t *testing.T) {
	tests := map[string]struct {
		raw  []string
		want map[string]string
	}{
		"nil": {
			raw:  nil,
			want: map[string]string{},
		},
		"pairs": {
			raw:  []string{"to=user@example.com", "subject=Hello"},
			want: map[string]string{"to": "user@example.com", "subject": "Hello"},
		},
		"split on first separator only": {
			raw:  []string{"body=k=v&x=y=z"},
			want: map[string]string{"body": "k=v&x=y=z"},
		},
		"empty value stays a present key": {
			raw:  []string{"cc="},
			want: map[string]string{"cc": ""},
		},
		"last repetition wins": {
			raw:  []string{"body=first", "body=second"},
			want: map[string]string{"body": "second"},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := parseInputs(tc.raw)
			if err != nil {
				t.Fatalf("parseInputs(%q): %v", tc.raw, err)
			}
			if len(got) != len(tc.want) {
				t.Fatalf("parseInputs(%q) = %v, want %v", tc.raw, got, tc.want)
			}
			for k, want := range tc.want {
				v, ok := got[k]
				if !ok {
					t.Fatalf("parseInputs(%q) missing key %q; got %v", tc.raw, k, got)
				}
				if v != want {
					t.Errorf("parseInputs(%q)[%q] = %q, want %q", tc.raw, k, v, want)
				}
			}
		})
	}
}

func TestParseInputsRejectsMalformedPair(t *testing.T) {
	tests := map[string]string{
		"no separator":                  "body",
		"empty key":                     "=value",
		"empty argument":                "",
		"separator missing with spaces": "subject Hello",
	}

	for name, offending := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := parseInputs([]string{"to=user@example.com", offending})
			if err == nil {
				t.Fatalf("parseInputs(%q) returned nil error, got map %v",
					offending, got)
			}
			if got != nil {
				t.Errorf("parseInputs(%q) returned map %v, want nil", offending, got)
			}
			if !strings.Contains(err.Error(), "invalid input format") {
				t.Errorf("parseInputs(%q) error = %v, want invalid input format",
					offending, err)
			}
			if !strings.Contains(err.Error(), offending) {
				t.Errorf("parseInputs(%q) error = %v, want it to name the argument",
					offending, err)
			}
		})
	}
}
