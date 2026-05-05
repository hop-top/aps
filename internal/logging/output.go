// Package logging — stdout/stderr formatter helpers with redaction.
//
// These mirror fmt.Print*/Fprint* but route their formatted string
// through Apply before writing. Migrate HIGH-severity stdout sites
// (env, run, adapter exec, session inspect, a2a get-task) to use
// these helpers so the redact filter is applied at the choke point
// instead of at every callsite.
//
// LOW-severity sites that print curated values (skill metadata,
// version, identity DID/badge) keep using bare fmt.* — adding the
// indirection without a leak class to defend against just adds noise.

package logging

import (
	"fmt"
	"io"
	"os"
)

// Print is a redacting fmt.Print: writes to os.Stdout.
func Print(a ...any) (int, error) {
	return Fprint(os.Stdout, a...)
}

// Println is a redacting fmt.Println: writes to os.Stdout.
func Println(a ...any) (int, error) {
	return Fprintln(os.Stdout, a...)
}

// Printf is a redacting fmt.Printf: writes to os.Stdout.
func Printf(format string, a ...any) (int, error) {
	return Fprintf(os.Stdout, format, a...)
}

// Fprint is a redacting fmt.Fprint: writes to w.
func Fprint(w io.Writer, a ...any) (int, error) {
	s := fmt.Sprint(a...)
	if Enabled() {
		s = Redactor().Apply(s)
	}
	return io.WriteString(w, s)
}

// Fprintln is a redacting fmt.Fprintln: writes to w.
func Fprintln(w io.Writer, a ...any) (int, error) {
	s := fmt.Sprintln(a...)
	if Enabled() {
		s = Redactor().Apply(s)
	}
	return io.WriteString(w, s)
}

// Fprintf is a redacting fmt.Fprintf: writes to w.
func Fprintf(w io.Writer, format string, a ...any) (int, error) {
	s := fmt.Sprintf(format, a...)
	if Enabled() {
		s = Redactor().Apply(s)
	}
	return io.WriteString(w, s)
}
