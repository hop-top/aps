// Package docsync pins judgment-laden doc prose to the code it
// describes. exitcodes_test.go pins the hand-written exit-code table
// in docs/cli/reference.md to internal/cli/exit.Code — the single
// mapping authority — and the kit constants it reuses.
//
// Two invariants per table row: the documented numeric code is what
// exit.Code returns for a representative error of the documented
// source, and the documented source identifier still exists in the
// code (registry entries reference the real Go identifiers, so a
// rename breaks this file's compile; identifiers that exit.go names
// directly are additionally grep-pinned to that file).
package docsync

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"hop.top/aps/internal/cli/exit"
	kitcli "hop.top/kit/go/console/cli"
	"hop.top/kit/go/console/output"
	"hop.top/kit/go/runtime/domain"
	"hop.top/kit/go/runtime/policy"
)

const (
	referenceDoc = "docs/cli/reference.md"
	exitSource   = "internal/cli/exit/exit.go"
)

// sourcePin binds one documented source label to the real code it
// names. Exactly one of rep / direct is used: rep is fed through
// exit.Code; direct pins a constant the classifier never returns
// itself (cobra usage handling exits before RunE).
type sourcePin struct {
	rep      error // representative error; exit.Code(rep) must equal the row's code
	repValid bool  // distinguishes rep=nil (success row) from direct pins
	direct   int   // constant value pin for rows outside exit.Code
	inExit   bool  // identifier must appear verbatim in exit.go
}

// sourcePins is keyed by the EXACT source labels the table uses
// (backticks stripped). Renaming a label in the doc misses the
// lookup; renaming the underlying identifier in code breaks the
// compile of this file. Both are loud.
var sourcePins = map[string]sourcePin{
	"—":             {rep: nil, repValid: true},
	"Generic error": {rep: errors.New("unmapped"), repValid: true},
	"kitcli.ExitUsage": {
		direct: int(kitcli.ExitUsage),
	},
	"domain.ErrNotFound": {
		rep:      fmt.Errorf("profile %q: %w", "ghost", domain.ErrNotFound),
		repValid: true,
		inExit:   true,
	},
	"domain.ErrConflict": {
		rep:      fmt.Errorf("profile already exists: %w", domain.ErrConflict),
		repValid: true,
		inExit:   true,
	},
	"policy.PolicyDeniedError": {
		rep:      &policy.PolicyDeniedError{PolicyName: "gate"},
		repValid: true,
	},
	"exit.ErrUnauthorized": {
		rep:      fmt.Errorf("credential check: %w", exit.ErrUnauthorized),
		repValid: true,
		inExit:   true,
	},
	"output.CodeRateLimited": {
		rep:      output.RateLimitedError("max-ops budget exceeded"),
		repValid: true,
	},
}

// undocumentedCodes lists convention exit codes exit.Code can emit
// that the table intentionally omits. Shrink-only: documenting one
// of these codes must remove it here, and every convention code must
// be either documented or listed — so the list can only get smaller.
var undocumentedCodes = []int{
	int(kitcli.ExitPermission), // 6
	int(kitcli.ExitTimeout),    // 7
	int(kitcli.ExitCancelled),  // 8
}

// conventionCodes is every classified exit code the convention
// declares (kit §8.1 constants plus the Factor-10 rate-limit code).
// Passthrough codes from exec.ExitError / arbitrary output.Error
// envelopes are unbounded and out of scope.
var conventionCodes = []int{
	int(kitcli.ExitOK),
	int(kitcli.ExitError),
	int(kitcli.ExitUsage),
	int(kitcli.ExitNotFound),
	int(kitcli.ExitConflict),
	int(kitcli.ExitAuth),
	int(kitcli.ExitPermission),
	int(kitcli.ExitTimeout),
	int(kitcli.ExitCancelled),
	output.ExitRateLimited,
}

type docRow struct {
	code    int
	sources []string
	line    string
}

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Join(filepath.Dir(file), "..", "..")
}

func readRepoFile(t *testing.T, rel string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(repoRoot(t), rel))
	if err != nil {
		t.Fatalf("read %s: %v", rel, err)
	}
	return string(b)
}

var backticks = regexp.MustCompile("`([^`]*)`")

// exitCodeRows extracts the rows of the "## Exit codes" table.
func exitCodeRows(t *testing.T) []docRow {
	t.Helper()
	doc := readRepoFile(t, referenceDoc)

	_, section, found := strings.Cut(doc, "\n## Exit codes\n")
	if !found {
		t.Fatalf("%s: no '## Exit codes' section", referenceDoc)
	}
	if next := strings.Index(section, "\n## "); next >= 0 {
		section = section[:next]
	}

	var rows []docRow
	for _, line := range strings.Split(section, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "|") {
			continue
		}
		cells := strings.Split(strings.Trim(line, "|"), "|")
		if len(cells) != 3 {
			t.Fatalf("%s: table row does not have 3 cells: %q", referenceDoc, line)
		}
		codeCell := strings.TrimSpace(cells[0])
		if codeCell == "Code" || strings.HasPrefix(codeCell, "---") {
			continue // header / separator
		}
		code, err := strconv.Atoi(codeCell)
		if err != nil {
			t.Fatalf("%s: non-numeric code cell %q in row %q", referenceDoc, codeCell, line)
		}
		srcCell := strings.TrimSpace(cells[1])
		var sources []string
		if m := backticks.FindAllStringSubmatch(srcCell, -1); m != nil {
			for _, g := range m {
				sources = append(sources, g[1])
			}
		} else {
			sources = []string{srcCell} // prose labels: "—", "Generic error"
		}
		rows = append(rows, docRow{code: code, sources: sources, line: line})
	}
	if len(rows) == 0 {
		t.Fatalf("%s: exit-code table has no data rows", referenceDoc)
	}
	return rows
}

// TestExitCodeTableMatchesCode asserts every documented row's code is
// what exit.Code returns for its documented source, and that each
// source identifier still exists.
func TestExitCodeTableMatchesCode(t *testing.T) {
	rows := exitCodeRows(t)
	exitSrc := readRepoFile(t, exitSource)

	prev := -1
	for _, row := range rows {
		if row.code <= prev {
			t.Errorf("rows not in ascending code order at %q", row.line)
		}
		prev = row.code

		for _, src := range row.sources {
			pin, ok := sourcePins[src]
			if !ok {
				t.Errorf("code %d: source %q is not pinned — the doc names an "+
					"identifier this test does not know; fix the doc or extend "+
					"sourcePins (row %q)", row.code, src, row.line)
				continue
			}
			if pin.repValid {
				if got := exit.Code(pin.rep); got != row.code {
					t.Errorf("code %d: exit.Code(<%s>) = %d — doc row disagrees "+
						"with the classifier (row %q)", row.code, src, got, row.line)
				}
			} else {
				if pin.direct != row.code {
					t.Errorf("code %d: constant %s = %d — doc row disagrees with "+
						"the constant (row %q)", row.code, src, pin.direct, row.line)
				}
			}
			if pin.inExit {
				ident := src[strings.LastIndex(src, ".")+1:]
				if !strings.Contains(exitSrc, ident) {
					t.Errorf("code %d: identifier %q no longer appears in %s — "+
						"sentinel renamed or mapping removed (row %q)",
						row.code, ident, exitSource, row.line)
				}
			}
		}
	}
}

// TestExitCodeTableCoversConvention asserts the shrink-only contract:
// every convention code is documented or explicitly allowlisted, and
// no allowlisted code is documented.
func TestExitCodeTableCoversConvention(t *testing.T) {
	documented := map[int]bool{}
	for _, row := range exitCodeRows(t) {
		if documented[row.code] {
			t.Errorf("code %d documented twice", row.code)
		}
		documented[row.code] = true
	}

	allowed := map[int]bool{}
	for _, code := range undocumentedCodes {
		if allowed[code] {
			t.Errorf("allowlist: duplicate entry %d", code)
		}
		allowed[code] = true
		if documented[code] {
			t.Errorf("code %d is documented now — remove it from "+
				"undocumentedCodes (shrink-only)", code)
		}
	}

	for _, code := range conventionCodes {
		if !documented[code] && !allowed[code] {
			t.Errorf("convention code %d is neither documented in %s nor "+
				"allowlisted in undocumentedCodes", code, referenceDoc)
		}
	}
}
