// Package fsname maps arbitrary identifiers to file-system-safe names.
//
// Characters reserved on Windows (<>:"/\|?*, control bytes) and the escape
// character itself are percent-encoded, so the mapping is reversible and
// identical on every OS. ':' matters most: on NTFS "a:b" addresses an
// alternate data stream rather than a file named "a:b".
package fsname

import (
	"fmt"
	"strings"
)

const hexDigits = "0123456789ABCDEF"

func reserved(c byte) bool {
	return c < 0x20 || c == 0x7f || strings.IndexByte(`<>:"/\|?*%`, c) >= 0
}

// Encode returns id with reserved bytes percent-encoded.
func Encode(id string) string {
	var b strings.Builder
	for i := 0; i < len(id); i++ {
		c := id[i]
		if reserved(c) {
			b.WriteByte('%')
			b.WriteByte(hexDigits[c>>4])
			b.WriteByte(hexDigits[c&0xf])
			continue
		}
		b.WriteByte(c)
	}
	return b.String()
}

// Decode reverses Encode. Malformed escapes are an error.
func Decode(name string) (string, error) {
	var b strings.Builder
	for i := 0; i < len(name); i++ {
		c := name[i]
		if c != '%' {
			b.WriteByte(c)
			continue
		}
		if i+2 >= len(name) {
			return "", fmt.Errorf("fsname: truncated escape in %q", name)
		}
		hi, okHi := unhex(name[i+1])
		lo, okLo := unhex(name[i+2])
		if !okHi || !okLo {
			return "", fmt.Errorf("fsname: bad escape in %q", name)
		}
		b.WriteByte(hi<<4 | lo)
		i += 2
	}
	return b.String(), nil
}

func unhex(c byte) (byte, bool) {
	switch {
	case c >= '0' && c <= '9':
		return c - '0', true
	case c >= 'A' && c <= 'F':
		return c - 'A' + 10, true
	case c >= 'a' && c <= 'f':
		return c - 'a' + 10, true
	}
	return 0, false
}
