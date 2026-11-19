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
		hi, lo := unhex(name[i+1]), unhex(name[i+2])
		if hi < 0 || lo < 0 {
			return "", fmt.Errorf("fsname: bad escape in %q", name)
		}
		b.WriteByte(byte(hi<<4 | lo))
		i += 2
	}
	return b.String(), nil
}

func unhex(c byte) int {
	switch {
	case c >= '0' && c <= '9':
		return int(c - '0')
	case c >= 'A' && c <= 'F':
		return int(c-'A') + 10
	case c >= 'a' && c <= 'f':
		return int(c-'a') + 10
	}
	return -1
}
