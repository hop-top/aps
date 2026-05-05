package core

import (
	"strings"
	"testing"
)

func TestGenerateProfileColor_Deterministic(t *testing.T) {
	a := GenerateProfileColor("noor")
	b := GenerateProfileColor("noor")
	if a != b {
		t.Fatalf("color not deterministic: %s vs %s", a, b)
	}
	if !strings.HasPrefix(a, "#") || len(a) != 7 {
		t.Fatalf("expected #RRGGBB hex, got %q", a)
	}
}

func TestGenerateProfileColor_DifferentIDs(t *testing.T) {
	// We can't guarantee distinct colors with a small palette, but two
	// well-known ids should at least both map into the palette.
	for _, id := range []string{"noor", "sami", "rami", "kai", ""} {
		c := GenerateProfileColor(id)
		found := false
		for _, p := range avatarPalette {
			if p == c {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("id %q produced %q, not in palette", id, c)
		}
	}
}

func TestGenerateProfileAvatar(t *testing.T) {
	url := GenerateProfileAvatar("noor", ProfileAvatarConfig{})
	if !strings.Contains(url, "shapes") {
		t.Errorf("default style not applied: %s", url)
	}
	if !strings.Contains(url, "seed=noor") {
		t.Errorf("seed not applied: %s", url)
	}

	custom := GenerateProfileAvatar("noor", ProfileAvatarConfig{Style: "bottts", Size: 256})
	if !strings.Contains(custom, "bottts") {
		t.Errorf("custom style not applied: %s", custom)
	}
	if !strings.Contains(custom, "size=256") {
		t.Errorf("size not applied: %s", custom)
	}
}

func TestAutoMode_ShouldAutoAssign(t *testing.T) {
	cases := []struct {
		mode        AutoMode
		interactive bool
		want        bool
	}{
		{AutoModeTrue, false, true},
		{AutoModeTrue, true, true},
		{AutoModeFalse, false, false},
		{AutoModeFalse, true, false},
		{AutoModeAuto, false, true},
		{AutoModeAuto, true, false},
		{"", false, false}, // unset == false
	}
	for _, c := range cases {
		got := c.mode.ShouldAutoAssign(c.interactive)
		if got != c.want {
			t.Errorf("AutoMode(%q).ShouldAutoAssign(%v) = %v, want %v", c.mode, c.interactive, got, c.want)
		}
	}
}
