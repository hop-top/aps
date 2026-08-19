package version

import "testing"

func TestNormalize(t *testing.T) {
	cases := []struct {
		name, in, want string
	}{
		{"bare semver", "0.6.0", "0.6.0"},
		{"v-prefixed", "v0.6.0", "0.6.0"},
		{"monorepo tag prefix + describe", "aps/v0.6.0-alpha.0-70-g7a18b94", "0.6.0-alpha.0-70-g7a18b94"},
		{"monorepo tag prefix, bare", "aps/0.6.0", "0.6.0"},
		{"dirty describe", "aps/v0.6.0-dirty", "0.6.0-dirty"},
		{"dev", "dev", "dev"},
		{"empty", "", "dev"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := Normalize(tc.in); got != tc.want {
				t.Errorf("Normalize(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestInfoString(t *testing.T) {
	cases := []struct {
		name, version, commit, want string
	}{
		{"bare semver", "0.6.0", "abc1234", "aps v0.6.0 (abc1234)"},
		{"v-prefixed", "v0.6.0", "abc1234", "aps v0.6.0 (abc1234)"},
		{"tag-prefixed describe", "aps/v0.6.0-alpha.0-70-g7a18b94", "7a18b94deadbeef", "aps v0.6.0-alpha.0-70-g7a18b94 (7a18b94)"},
		{"dev", "dev", "none", "aps dev (none)"},
		{"empty", "", "none", "aps dev (none)"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := Info{Version: tc.version, Commit: tc.commit}.String()
			if got != tc.want {
				t.Errorf("Info{%q,%q}.String() = %q, want %q", tc.version, tc.commit, got, tc.want)
			}
		})
	}
}

func TestGetAndShortNormalize(t *testing.T) {
	orig := Version
	t.Cleanup(func() { Version = orig })

	Version = "aps/v0.6.0-alpha.0-70-g7a18b94"
	const want = "0.6.0-alpha.0-70-g7a18b94"
	if got := Short(); got != want {
		t.Errorf("Short() = %q, want %q", got, want)
	}
	if got := Get().Version; got != want {
		t.Errorf("Get().Version = %q, want %q", got, want)
	}
}
