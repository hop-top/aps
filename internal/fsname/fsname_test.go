package fsname

import (
	"strings"
	"testing"
)

func TestRoundTrip(t *testing.T) {
	for _, id := range []string{"plain", "aps:global", "a/b\\c", "100%", "k?*|<>\"", "", "tab\there", "ünï"} {
		enc := Encode(id)
		if strings.ContainsAny(enc, `<>:"/\|?*`) {
			t.Errorf("Encode(%q) = %q has reserved char", id, enc)
		}
		got, err := Decode(enc)
		if err != nil || got != id {
			t.Errorf("Decode(Encode(%q)) = %q, %v", id, got, err)
		}
	}
}

func TestPlainUnchanged(t *testing.T) {
	if got := Encode("key-2.v_3"); got != "key-2.v_3" {
		t.Errorf("got %q", got)
	}
}

func TestDecodeMalformed(t *testing.T) {
	for _, s := range []string{"%", "%4", "%zz"} {
		if _, err := Decode(s); err == nil {
			t.Errorf("Decode(%q) expected error", s)
		}
	}
}
