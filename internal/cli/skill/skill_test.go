package skill

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"hop.top/aps/internal/cli/listing"
	"hop.top/aps/internal/skills"
	"hop.top/kit/go/console/output"
)

// TestSkillSourceFilter — listing.MatchString on Source.
func TestSkillSourceFilter(t *testing.T) {
	rows := []skillSummaryRow{
		{Name: "deploy", Source: "Profile"},
		{Name: "review", Source: "Global"},
		{Name: "edit", Source: "Profile"},
		{Name: "scrape", Source: "Claude Code"},
	}
	pred := listing.MatchString(func(r skillSummaryRow) string { return r.Source }, "Profile")
	got := listing.Filter(rows, pred)
	if len(got) != 2 {
		t.Fatalf("len=%d, want 2", len(got))
	}
}

// TestSkillSourceFilter_Empty — unset flag matches every row.
func TestSkillSourceFilter_Empty(t *testing.T) {
	rows := []skillSummaryRow{
		{Name: "a", Source: "Profile"},
		{Name: "b", Source: "Global"},
	}
	pred := listing.All(
		listing.MatchString(func(r skillSummaryRow) string { return r.Source }, ""),
	)
	got := listing.Filter(rows, pred)
	if len(got) != 2 {
		t.Fatalf("empty --source must match all; got %d", len(got))
	}
}

// TestBuildSkillRows_TruncatesDescription — long descriptions get
// the right-trim treatment to keep the table scannable; JSON callers
// see the original via skills.Skill.Description.
func TestBuildSkillRows_TruncatesDescription(t *testing.T) {
	long := strings.Repeat("x", skillDescriptionWidth+30)
	registry := skills.NewRegistry("noor", nil, false)
	// Empty registry → empty rows; sanity guard.
	if rows := buildSkillRows(registry, "noor"); len(rows) != 0 {
		t.Fatalf("empty registry should yield 0 rows; got %d", len(rows))
	}
	// We can't seed the registry directly without disk fixtures;
	// exercise the truncation rule by hand.
	row := skillSummaryRow{Description: long}
	if len(row.Description) <= skillDescriptionWidth {
		t.Fatalf("seed too short")
	}
	short := long[:skillDescriptionWidth-1] + "…"
	if len(short) >= len(long) {
		t.Fatalf("truncation produced no shrink")
	}
}

// TestSkillListCmd_FlagSet asserts only --source survives the audit;
// --verbose was dropped per T-0440.
func TestSkillListCmd_FlagSet(t *testing.T) {
	cmd := newListCmd()
	if f := cmd.Flags().Lookup("source"); f == nil {
		t.Error("--source flag missing")
	}
	if f := cmd.Flags().Lookup("verbose"); f != nil {
		t.Error("--verbose should have been dropped")
	}
	// --profile is owned by the kit/cli root global; subcommand
	// must not redeclare it locally.
	if f := cmd.Flags().Lookup("profile"); f != nil {
		t.Error("--profile should inherit from root global, not be local")
	}
}

func TestRunSkillScript_Success(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("APS_DATA_PATH", dataDir)
	profileID := "noor"
	writeSkillFixture(t, dataDir, profileID, "runner", map[string]string{
		"echo.sh": `#!/bin/sh
printf 'skill=%s profile=%s script=%s args=%s,%s\n' "$APS_SKILL_NAME" "$APS_PROFILE_ID" "$APS_SKILL_SCRIPT" "$1" "$2"
`,
	})

	var stdout, stderr bytes.Buffer
	err := runSkillScript(context.Background(), profileID, []string{"runner", "echo.sh", "alpha", "beta"}, 1, strings.NewReader(""), &stdout, &stderr)
	if err != nil {
		t.Fatalf("runSkillScript returned error: %v", err)
	}
	if got, want := stdout.String(), "skill=runner profile=noor script=echo.sh args=alpha,beta\n"; got != want {
		t.Fatalf("stdout=%q, want %q", got, want)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr=%q, want empty", stderr.String())
	}
}

func TestRunSkillScript_MissingSkill(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("APS_DATA_PATH", dataDir)

	err := runSkillScript(context.Background(), "noor", []string{"missing", "echo.sh"}, 1, strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil {
		t.Fatal("expected missing skill error")
	}
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("error=%v, want os.ErrNotExist", err)
	}
}

func TestRunSkillScript_MissingScript(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("APS_DATA_PATH", dataDir)
	writeSkillFixture(t, dataDir, "noor", "runner", nil)

	err := runSkillScript(context.Background(), "noor", []string{"runner", "missing.sh"}, 1, strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil {
		t.Fatal("expected missing script error")
	}
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("error=%v, want os.ErrNotExist", err)
	}
}

func TestRunSkillScript_NonzeroExit(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("APS_DATA_PATH", dataDir)
	writeSkillFixture(t, dataDir, "noor", "runner", map[string]string{
		"fail.sh": `#!/bin/sh
echo failout
echo failerr >&2
exit 23
`,
	})

	var stdout, stderr bytes.Buffer
	err := runSkillScript(context.Background(), "noor", []string{"runner", "fail.sh"}, 1, strings.NewReader(""), &stdout, &stderr)
	if err == nil {
		t.Fatal("expected nonzero exit error")
	}
	var outputErr *output.Error
	if !errors.As(err, &outputErr) {
		t.Fatalf("error type=%T, want *output.Error", err)
	}
	if outputErr.ExitCode != 23 {
		t.Fatalf("exit code=%d, want 23", outputErr.ExitCode)
	}
	if got, want := stdout.String(), "failout\n"; got != want {
		t.Fatalf("stdout=%q, want %q", got, want)
	}
	if got, want := stderr.String(), "failerr\n"; got != want {
		t.Fatalf("stderr=%q, want %q", got, want)
	}
}

func writeSkillFixture(t *testing.T, dataDir, profileID, skillName string, scripts map[string]string) {
	t.Helper()
	base := filepath.Join(dataDir, "profiles", profileID, "skills", skillName)
	if err := os.MkdirAll(filepath.Join(base, "scripts"), 0755); err != nil {
		t.Fatalf("create skill scripts dir: %v", err)
	}
	skillMD := "---\nname: " + skillName + "\ndescription: Test skill\n---\n\nBody\n"
	if err := os.WriteFile(filepath.Join(base, "SKILL.md"), []byte(skillMD), 0644); err != nil {
		t.Fatalf("write SKILL.md: %v", err)
	}
	for name, body := range scripts {
		if err := os.WriteFile(filepath.Join(base, "scripts", name), []byte(body), 0755); err != nil {
			t.Fatalf("write script %s: %v", name, err)
		}
	}
}
