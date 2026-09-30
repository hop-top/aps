package core

import (
	"errors"
	"fmt"
	"strings"
)

// ErrInvalidProfileID is returned (wrapped) when a profile id cannot be
// used as a portable directory name. Match with errors.Is.
var ErrInvalidProfileID = errors.New("invalid profile id")

// windowsReservedNames are device names Windows refuses as a file or
// directory name, regardless of case or extension ("nul", "Con.txt").
// COM/LPT include the superscript-digit variants Windows also reserves.
var windowsReservedNames = func() map[string]struct{} {
	m := map[string]struct{}{"CON": {}, "PRN": {}, "AUX": {}, "NUL": {}}
	for _, d := range []string{"1", "2", "3", "4", "5", "6", "7", "8", "9", "¹", "²", "³"} {
		m["COM"+d] = struct{}{}
		m["LPT"+d] = struct{}{}
	}
	return m
}()

// windowsInvalidChars are characters Windows rejects in file names.
const windowsInvalidChars = `<>:"/\|?*`

// ValidateProfileID reports whether id is usable as a profile directory
// name on every supported platform. Profiles travel between machines
// via bundle export/import, so the Windows rules apply everywhere:
//
//   - not empty, "." or ".."
//   - no reserved device name (CON, PRN, AUX, NUL, COM1-9, LPT1-9),
//     case-insensitive, with or without an extension
//   - none of < > : " / \ | ? * and no control characters (0x00-0x1F)
//   - no trailing dot or space
//
// Validation applies when a profile is created; loading and listing
// existing profiles never calls it.
func ValidateProfileID(id string) error {
	if id == "" {
		return fmt.Errorf("%w: id must not be empty", ErrInvalidProfileID)
	}
	if id == "." || id == ".." {
		return fmt.Errorf("%w %q: %q is a reserved path name", ErrInvalidProfileID, id, id)
	}
	for _, r := range id {
		if r < 0x20 {
			return fmt.Errorf("%w %q: control character %U not allowed", ErrInvalidProfileID, id, r)
		}
		if strings.ContainsRune(windowsInvalidChars, r) {
			return fmt.Errorf("%w %q: invalid character %q (not allowed: %s)", ErrInvalidProfileID, id, r, windowsInvalidChars)
		}
	}
	if last := id[len(id)-1]; last == '.' || last == ' ' {
		return fmt.Errorf("%w %q: trailing dot or space not allowed", ErrInvalidProfileID, id)
	}
	stem, _, _ := strings.Cut(id, ".")
	stem = strings.TrimRight(stem, " ") // Windows ignores spaces before the extension
	if _, ok := windowsReservedNames[strings.ToUpper(stem)]; ok {
		return fmt.Errorf("%w %q: %q is a reserved device name on Windows", ErrInvalidProfileID, id, strings.ToUpper(stem))
	}
	return nil
}
