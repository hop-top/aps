package adapter_e2e

import (
	"strings"
	"testing"
)

// Coverage for adapters/calendar/backends/{gcalcli,gam}/*.sh.
//
// Both calendar backends already build argv with bash arrays, so the
// word-splitting class that affected email and contacts does not apply
// here. What does apply is their much richer conditional logic:
// optional fields accumulated into an ARGS array, comma-separated
// attendee lists split and trimmed, the "primary" calendar resolved
// differently per backend, and pre-check gates that must refuse rather
// than act on an ambiguous target.
//
// The two backends deliberately differ (gcalcli is per-user and
// title-matching; gam is admin-scoped and id-precise), so they are
// asserted separately rather than through a shared table.

func gcalcliScript(action string) backendScript {
	return backendScript{Adapter: "calendar", Backend: "gcalcli", Action: action}
}

func gamScript(action string) backendScript {
	return backendScript{Adapter: "calendar", Backend: "gam", Action: action}
}

// baseEventEnv is the minimum for a create-event call.
func baseEventEnv() map[string]string {
	return map[string]string{
		"CAL_SUMMARY": "Standup",
		"CAL_START":   "2026-08-10T09:00:00Z",
		"CAL_END":     "2026-08-10T09:15:00Z",
	}
}

// --- gcalcli ---------------------------------------------------------

// TestGcalcliCreate_Minimal covers the baseline argv: no --calendar
// flag when the calendar is "primary", since gcalcli defaults there.
func TestGcalcliCreate_Minimal(t *testing.T) {
	requireBash(t)
	t.Parallel()

	invs, out, err := runBackendScriptMulti(t, gcalcliScript("create-event"),
		baseEventEnv(), stubOptions{})
	if err != nil {
		t.Fatalf("create-event.sh failed: %v\noutput: %s", err, out)
	}

	assertArgv(t, invs[0],
		"add", "--title", "Standup",
		"--when", "2026-08-10T09:00:00Z",
		"--duration_end", "2026-08-10T09:15:00Z")
}

// TestGcalcliCreate_NamedCalendar covers the --calendar flag appearing
// for a non-primary calendar, ahead of the verb.
func TestGcalcliCreate_NamedCalendar(t *testing.T) {
	requireBash(t)
	t.Parallel()

	env := baseEventEnv()
	env["CAL_CALENDAR"] = "Team Planning"

	invs, out, err := runBackendScriptMulti(t, gcalcliScript("create-event"),
		env, stubOptions{})
	if err != nil {
		t.Fatalf("create-event.sh failed: %v\noutput: %s", err, out)
	}

	if !argvHasPair(invs[0], "--calendar", "Team Planning") {
		t.Errorf("calendar name with a space was not passed as one argument: %s",
			invs[0].argvString())
	}
}

// TestGcalcliCreate_OptionalFields covers every optional field being
// appended, and the flag-valued ones staying paired with their values.
func TestGcalcliCreate_OptionalFields(t *testing.T) {
	requireBash(t)
	t.Parallel()

	env := baseEventEnv()
	env["CAL_LOCATION"] = "Room 3B"
	env["CAL_DESCRIPTION"] = "Weekly sync, bring notes"
	env["CAL_RECURRENCE"] = "RRULE:FREQ=WEEKLY;BYDAY=MO"

	invs, out, err := runBackendScriptMulti(t, gcalcliScript("create-event"),
		env, stubOptions{})
	if err != nil {
		t.Fatalf("create-event.sh failed: %v\noutput: %s", err, out)
	}

	for _, pair := range [][2]string{
		{"--where", "Room 3B"},
		{"--description", "Weekly sync, bring notes"},
		{"--rrule", "RRULE:FREQ=WEEKLY;BYDAY=MO"},
	} {
		if !argvHasPair(invs[0], pair[0], pair[1]) {
			t.Errorf("missing %s %q: %s", pair[0], pair[1], invs[0].argvString())
		}
	}
}

// TestGcalcliCreate_AllDayAndTransparency covers the two boolean-ish
// flags, which are emitted only on an exact string match.
func TestGcalcliCreate_AllDayAndTransparency(t *testing.T) {
	requireBash(t)
	t.Parallel()

	env := baseEventEnv()
	env["CAL_ALL_DAY"] = "true"
	env["CAL_TRANSPARENCY"] = "transparent"

	invs, out, err := runBackendScriptMulti(t, gcalcliScript("create-event"),
		env, stubOptions{})
	if err != nil {
		t.Fatalf("create-event.sh failed: %v\noutput: %s", err, out)
	}

	if !argvContains(invs[0], "--allday") {
		t.Errorf("CAL_ALL_DAY=true did not emit --allday: %s", invs[0].argvString())
	}
	if !argvHasPair(invs[0], "--transparency", "transparent") {
		t.Errorf("CAL_TRANSPARENCY=transparent not passed: %s", invs[0].argvString())
	}
}

// TestGcalcliCreate_AllDayFalseOmitsFlag pins that only the literal
// "true" enables all-day. A stray value must not silently create an
// all-day event.
func TestGcalcliCreate_AllDayFalseOmitsFlag(t *testing.T) {
	requireBash(t)
	t.Parallel()

	for _, v := range []string{"false", "TRUE", "1", "yes", ""} {
		t.Run("value="+v, func(t *testing.T) {
			t.Parallel()

			env := baseEventEnv()
			env["CAL_ALL_DAY"] = v

			invs, out, err := runBackendScriptMulti(t, gcalcliScript("create-event"),
				env, stubOptions{})
			if err != nil {
				t.Fatalf("create-event.sh failed: %v\noutput: %s", err, out)
			}
			if argvContains(invs[0], "--allday") {
				t.Errorf("CAL_ALL_DAY=%q emitted --allday: %s", v, invs[0].argvString())
			}
		})
	}
}

// TestGcalcliCreate_AttendeeSplitting covers the comma-separated
// attendee list: each becomes its own --email flag, display-name form
// is reduced to the bracketed address, and surrounding whitespace is
// trimmed.
func TestGcalcliCreate_AttendeeSplitting(t *testing.T) {
	requireBash(t)
	t.Parallel()

	env := baseEventEnv()
	env["CAL_ATTENDEES"] = "ada@example.com, Bob Smith <bob@example.com> ,carol@example.com"

	invs, out, err := runBackendScriptMulti(t, gcalcliScript("create-event"),
		env, stubOptions{})
	if err != nil {
		t.Fatalf("create-event.sh failed: %v\noutput: %s", err, out)
	}

	for _, want := range []string{
		"ada@example.com",
		"bob@example.com",
		"carol@example.com",
	} {
		if !argvHasPair(invs[0], "--email", want) {
			t.Errorf("attendee %q not passed as --email: %s", want, invs[0].argvString())
		}
	}
	// The display name must not survive as its own attendee.
	if argvContains(invs[0], "Bob Smith <bob@example.com>") {
		t.Errorf("display-name form was not reduced to the address: %s",
			invs[0].argvString())
	}
}

// TestGcalcliListEvents covers the agenda argv, including --details
// all which downstream parsing depends on.
func TestGcalcliListEvents(t *testing.T) {
	requireBash(t)
	t.Parallel()

	invs, out, err := runBackendScriptMulti(t, gcalcliScript("list-events"), map[string]string{
		"CAL_START": "2026-08-01",
		"CAL_END":   "2026-08-31",
	}, stubOptions{})
	if err != nil {
		t.Fatalf("list-events.sh failed: %v\noutput: %s", err, out)
	}

	assertArgv(t, invs[0], "agenda", "2026-08-01", "2026-08-31", "--details", "all")
}

// TestGcalcliDelete_RefusesAmbiguous covers the pre-query gate: when
// the title search matches more than one event, the delete must be
// refused rather than removing whichever matched first.
func TestGcalcliDelete_RefusesAmbiguous(t *testing.T) {
	requireBash(t)
	t.Parallel()

	// Two weekday-leading lines = two matches.
	searchOut := "Mon Aug 10  Standup\nTue Aug 11  Standup\n"

	invs, out, err := runBackendScriptMulti(t, gcalcliScript("delete-event"),
		map[string]string{"CAL_EVENT_ID": "Standup"},
		stubOptions{Responses: []stubResponse{
			{Match: []string{"search"}, Stdout: searchOut},
		}})
	if err == nil {
		t.Fatalf("expected refusal on ambiguous match, got success:\n%s",
			invocationsString(invs))
	}
	if !strings.Contains(out, "refusing ambiguous delete") {
		t.Errorf("diagnostic does not explain the refusal:\n%s", out)
	}
	// Only the search should have run — never the delete.
	for _, inv := range invs {
		if argvContains(inv, "delete") {
			t.Errorf("delete was issued despite an ambiguous match: %s", inv.argvString())
		}
	}
}

// TestGcalcliDelete_RefusesNoMatch covers the zero-match branch, which
// must exit 65 (EX_DATAERR) rather than deleting anything.
func TestGcalcliDelete_RefusesNoMatch(t *testing.T) {
	requireBash(t)
	t.Parallel()

	invs, out, err := runBackendScriptMulti(t, gcalcliScript("delete-event"),
		map[string]string{"CAL_EVENT_ID": "Nonexistent"},
		stubOptions{Responses: []stubResponse{
			{Match: []string{"search"}, Stdout: "No Events Found\n"},
		}})
	if err == nil {
		t.Fatalf("expected refusal on zero matches, got success:\n%s",
			invocationsString(invs))
	}
	if !strings.Contains(out, "no event matches") {
		t.Errorf("diagnostic does not explain the refusal:\n%s", out)
	}
	for _, inv := range invs {
		if argvContains(inv, "delete") {
			t.Errorf("delete was issued despite zero matches: %s", inv.argvString())
		}
	}
}

// TestGcalcliDelete_SingleMatchProceeds covers the happy path: exactly
// one match, so the delete runs.
func TestGcalcliDelete_SingleMatchProceeds(t *testing.T) {
	requireBash(t)
	t.Parallel()

	invs, out, err := runBackendScriptMulti(t, gcalcliScript("delete-event"),
		map[string]string{"CAL_EVENT_ID": "Standup"},
		stubOptions{Responses: []stubResponse{
			{Match: []string{"search"}, Stdout: "Mon Aug 10  Standup\n"},
		}})
	if err != nil {
		t.Fatalf("delete-event.sh failed on a single match: %v\noutput: %s", err, out)
	}
	if len(invs) != 2 {
		t.Fatalf("expected search + delete, got %d:\n%s", len(invs), invocationsString(invs))
	}

	assertArgv(t, invs[1], "delete", "Standup")
}

// TestGcalcliUnsupportedActions pins the two actions gcalcli cannot
// perform. Both must exit 2 (backend cannot do this) and say which
// backend to switch to — silently succeeding would be far worse.
func TestGcalcliUnsupportedActions(t *testing.T) {
	requireBash(t)
	t.Parallel()

	cases := []struct {
		action string
		env    map[string]string
	}{
		{
			action: "update-event",
			env:    map[string]string{"CAL_EVENT_ID": "evt1"},
		},
		{
			action: "respond-event",
			env: map[string]string{
				"CAL_EVENT_ID": "evt1",
				"CAL_RESPONSE": "accepted",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.action, func(t *testing.T) {
			t.Parallel()

			invs, out, err := runBackendScriptMulti(t, gcalcliScript(tc.action),
				tc.env, stubOptions{})
			if err == nil {
				t.Fatalf("%s: expected refusal, got success", tc.action)
			}
			if len(invs) != 0 {
				t.Errorf("%s: backend was invoked despite being unsupported:\n%s",
					tc.action, invocationsString(invs))
			}
			if !strings.Contains(out, "not supported by gcalcli") {
				t.Errorf("%s: diagnostic does not name the limitation:\n%s", tc.action, out)
			}
			if !strings.Contains(out, "gam") {
				t.Errorf("%s: diagnostic does not point at the gam backend:\n%s",
					tc.action, out)
			}
		})
	}
}

// TestGcalcliRespond_RejectsInvalidResponse covers the enum gate,
// which must reject before the unsupported-action message so a typo is
// reported as a usage error (64) rather than a backend limitation.
func TestGcalcliRespond_RejectsInvalidResponse(t *testing.T) {
	requireBash(t)
	t.Parallel()

	invs, out, err := runBackendScriptMulti(t, gcalcliScript("respond-event"), map[string]string{
		"CAL_EVENT_ID": "evt1",
		"CAL_RESPONSE": "maybe",
	}, stubOptions{})
	if err == nil {
		t.Fatalf("expected rejection of an invalid response:\n%s", invocationsString(invs))
	}
	if !strings.Contains(out, "invalid response") {
		t.Errorf("diagnostic does not name the invalid value:\n%s", out)
	}
}

// TestGcalcliFreeBusy_PerAttendee covers the loop: one agenda call per
// attendee, each with the trimmed address as the calendar.
func TestGcalcliFreeBusy_PerAttendee(t *testing.T) {
	requireBash(t)
	t.Parallel()

	invs, out, err := runBackendScriptMulti(t, gcalcliScript("free-busy"), map[string]string{
		"CAL_EMAILS": "ada@example.com, Bob <bob@example.com>",
		"CAL_START":  "2026-08-01",
		"CAL_END":    "2026-08-02",
	}, stubOptions{})
	if err != nil {
		t.Fatalf("free-busy.sh failed: %v\noutput: %s", err, out)
	}
	if len(invs) != 2 {
		t.Fatalf("expected one call per attendee, got %d:\n%s",
			len(invs), invocationsString(invs))
	}

	if !argvHasPair(invs[0], "--calendar", "ada@example.com") {
		t.Errorf("first attendee not queried: %s", invs[0].argvString())
	}
	if !argvHasPair(invs[1], "--calendar", "bob@example.com") {
		t.Errorf("display-name attendee not reduced to address: %s", invs[1].argvString())
	}
}

// --- gam -------------------------------------------------------------

// baseGamEnv adds the profile user gam requires on top of the event
// fields.
func baseGamEnv() map[string]string {
	env := baseEventEnv()
	env["APS_EMAIL_FROM"] = "ops@example.com"
	return env
}

// TestGamCreate_PrimaryResolvesToUser covers gam's difference from
// gcalcli: "primary" is rewritten to the profile user's address,
// because gam addresses calendars by id.
func TestGamCreate_PrimaryResolvesToUser(t *testing.T) {
	requireBash(t)
	t.Parallel()

	invs, out, err := runBackendScriptMulti(t, gamScript("create-event"),
		baseGamEnv(), stubOptions{})
	if err != nil {
		t.Fatalf("create-event.sh failed: %v\noutput: %s", err, out)
	}

	assertArgv(t, invs[0],
		"calendar", "ops@example.com", "addevent", "summary", "Standup",
		"start", "time", "2026-08-10T09:00:00Z",
		"end", "time", "2026-08-10T09:15:00Z")
}

// TestGamCreate_AllDayUsesAlldayKeywords covers the branch selecting
// gam's allday start/end form over the timed form.
func TestGamCreate_AllDayUsesAlldayKeywords(t *testing.T) {
	requireBash(t)
	t.Parallel()

	env := baseGamEnv()
	env["CAL_ALL_DAY"] = "true"

	invs, out, err := runBackendScriptMulti(t, gamScript("create-event"),
		env, stubOptions{})
	if err != nil {
		t.Fatalf("create-event.sh failed: %v\noutput: %s", err, out)
	}

	joined := strings.Join(invs[0].Argv, " ")
	if !strings.Contains(joined, "start allday") || !strings.Contains(joined, "end allday") {
		t.Errorf("all-day event did not use gam's allday keywords: %s",
			invs[0].argvString())
	}
	if strings.Contains(joined, "start time") {
		t.Errorf("all-day event also emitted the timed form: %s", invs[0].argvString())
	}
}

// TestGamCreate_NamedCalendarPassedThrough covers a non-primary
// calendar id being used verbatim rather than resolved.
func TestGamCreate_NamedCalendarPassedThrough(t *testing.T) {
	requireBash(t)
	t.Parallel()

	env := baseGamEnv()
	env["CAL_CALENDAR"] = "team-x@group.calendar.google.com"

	invs, out, err := runBackendScriptMulti(t, gamScript("create-event"),
		env, stubOptions{})
	if err != nil {
		t.Fatalf("create-event.sh failed: %v\noutput: %s", err, out)
	}

	if !argvHasPair(invs[0], "calendar", "team-x@group.calendar.google.com") {
		t.Errorf("named calendar id not passed through: %s", invs[0].argvString())
	}
}

// TestGamCreate_AttendeeSplitting mirrors the gcalcli attendee case
// with gam's `attendee <addr>` keyword form.
func TestGamCreate_AttendeeSplitting(t *testing.T) {
	requireBash(t)
	t.Parallel()

	env := baseGamEnv()
	env["CAL_ATTENDEES"] = "ada@example.com, Bob Smith <bob@example.com>"

	invs, out, err := runBackendScriptMulti(t, gamScript("create-event"),
		env, stubOptions{})
	if err != nil {
		t.Fatalf("create-event.sh failed: %v\noutput: %s", err, out)
	}

	for _, want := range []string{"ada@example.com", "bob@example.com"} {
		if !argvHasPair(invs[0], "attendee", want) {
			t.Errorf("attendee %q not passed: %s", want, invs[0].argvString())
		}
	}
}

// TestGamDelete_IdPrecise covers gam's delete: by exact id, with the
// `doit` confirmation keyword gam requires for non-interactive runs.
func TestGamDelete_IdPrecise(t *testing.T) {
	requireBash(t)
	t.Parallel()

	invs, out, err := runBackendScriptMulti(t, gamScript("delete-event"), map[string]string{
		"APS_EMAIL_FROM": "ops@example.com",
		"CAL_EVENT_ID":   "evt-abc123",
	}, stubOptions{})
	if err != nil {
		t.Fatalf("delete-event.sh failed: %v\noutput: %s", err, out)
	}

	assertArgv(t, invs[0],
		"calendar", "ops@example.com", "deleteevent", "evt-abc123", "doit")
}

// TestGamDelete_SuppressNotifications covers the opt-out flag, which
// must appear only when explicitly set to "false".
func TestGamDelete_SuppressNotifications(t *testing.T) {
	requireBash(t)
	t.Parallel()

	invs, out, err := runBackendScriptMulti(t, gamScript("delete-event"), map[string]string{
		"APS_EMAIL_FROM":         "ops@example.com",
		"CAL_EVENT_ID":           "evt-abc123",
		"CAL_SEND_NOTIFICATIONS": "false",
	}, stubOptions{})
	if err != nil {
		t.Fatalf("delete-event.sh failed: %v\noutput: %s", err, out)
	}

	if !argvHasPair(invs[0], "notifyattendees", "false") {
		t.Errorf("notification opt-out not passed: %s", invs[0].argvString())
	}
}

// TestGamUpdate_OnlyPassedFields covers the patch semantics: fields
// the caller did not set must not appear, so an update cannot blank a
// field it never mentioned.
func TestGamUpdate_OnlyPassedFields(t *testing.T) {
	requireBash(t)
	t.Parallel()

	invs, out, err := runBackendScriptMulti(t, gamScript("update-event"), map[string]string{
		"APS_EMAIL_FROM": "ops@example.com",
		"CAL_EVENT_ID":   "evt-abc123",
		"CAL_SUMMARY":    "Renamed standup",
	}, stubOptions{})
	if err != nil {
		t.Fatalf("update-event.sh failed: %v\noutput: %s", err, out)
	}

	if !argvHasPair(invs[0], "summary", "Renamed standup") {
		t.Errorf("summary not patched: %s", invs[0].argvString())
	}
	for _, absent := range []string{"location", "description", "rrule", "transparency"} {
		if argvContains(invs[0], absent) {
			t.Errorf("unset field %q leaked into the patch: %s",
				absent, invs[0].argvString())
		}
	}
}

// TestGamUpdate_StartRequiresEnd pins that a half-specified time range
// is dropped rather than sent. gam would otherwise receive a start
// with no matching end.
func TestGamUpdate_StartRequiresEnd(t *testing.T) {
	requireBash(t)
	t.Parallel()

	invs, out, err := runBackendScriptMulti(t, gamScript("update-event"), map[string]string{
		"APS_EMAIL_FROM": "ops@example.com",
		"CAL_EVENT_ID":   "evt-abc123",
		"CAL_START":      "2026-08-10T09:00:00Z",
	}, stubOptions{})
	if err != nil {
		t.Fatalf("update-event.sh failed: %v\noutput: %s", err, out)
	}

	if argvContains(invs[0], "start") {
		t.Errorf("start was sent without a matching end: %s", invs[0].argvString())
	}
}

// TestGamRespond_RequiresAttendeeMatch covers the pre-check: gam's
// updateevent would silently ADD the user as a new attendee if they
// were not already on the event, which is not what "respond" means.
func TestGamRespond_RequiresAttendeeMatch(t *testing.T) {
	requireBash(t)
	t.Parallel()

	// Event info listing someone else entirely.
	info := "Event: evt1\nAttendees:\n  email: someone@example.com\n"

	invs, out, err := runBackendScriptMulti(t, gamScript("respond-event"), map[string]string{
		"APS_EMAIL_FROM": "ops@example.com",
		"CAL_EVENT_ID":   "evt1",
		"CAL_RESPONSE":   "accepted",
	}, stubOptions{Responses: []stubResponse{
		{Match: []string{"calendar"}, Stdout: info},
	}})
	if err == nil {
		t.Fatalf("expected refusal when user is not an attendee:\n%s",
			invocationsString(invs))
	}
	if !strings.Contains(out, "is not an attendee") {
		t.Errorf("diagnostic does not explain the refusal:\n%s", out)
	}
	for _, inv := range invs {
		if argvContains(inv, "updateevent") {
			t.Errorf("updateevent ran despite the user not being an attendee: %s",
				inv.argvString())
		}
	}
}

// TestGamRespond_ProceedsWhenAttendee covers the happy path: the user
// is listed, so the response status is patched.
func TestGamRespond_ProceedsWhenAttendee(t *testing.T) {
	requireBash(t)
	t.Parallel()

	info := "Event: evt1\nAttendees:\n  email: ops@example.com\n"

	invs, out, err := runBackendScriptMulti(t, gamScript("respond-event"), map[string]string{
		"APS_EMAIL_FROM": "ops@example.com",
		"CAL_EVENT_ID":   "evt1",
		"CAL_RESPONSE":   "accepted",
	}, stubOptions{Responses: []stubResponse{
		{Match: []string{"calendar"}, Stdout: info},
	}})
	if err != nil {
		t.Fatalf("respond-event.sh failed: %v\noutput: %s", err, out)
	}
	if len(invs) != 2 {
		t.Fatalf("expected info + updateevent, got %d:\n%s",
			len(invs), invocationsString(invs))
	}

	if !argvHasPair(invs[1], "responsestatus", "accepted") {
		t.Errorf("response status not patched: %s", invs[1].argvString())
	}
	if !argvHasPair(invs[1], "attendee", "ops@example.com") {
		t.Errorf("attendee not identified in the patch: %s", invs[1].argvString())
	}
}

// TestGamRespond_RejectsInvalidResponse covers the enum gate, which
// must fire before any backend call.
func TestGamRespond_RejectsInvalidResponse(t *testing.T) {
	requireBash(t)
	t.Parallel()

	invs, out, err := runBackendScriptMulti(t, gamScript("respond-event"), map[string]string{
		"APS_EMAIL_FROM": "ops@example.com",
		"CAL_EVENT_ID":   "evt1",
		"CAL_RESPONSE":   "maybe",
	}, stubOptions{})
	if err == nil {
		t.Fatalf("expected rejection of an invalid response:\n%s", invocationsString(invs))
	}
	if len(invs) != 0 {
		t.Errorf("backend invoked despite an invalid response value:\n%s",
			invocationsString(invs))
	}
	if !strings.Contains(out, "invalid response") {
		t.Errorf("diagnostic does not name the invalid value:\n%s", out)
	}
}

// TestGamListCalendars_RequestsJSON pins formatjson. gam's default
// human-formatted output would produce garbage downstream.
func TestGamListCalendars_RequestsJSON(t *testing.T) {
	requireBash(t)
	t.Parallel()

	invs, out, err := runBackendScriptMulti(t, gamScript("list-calendars"), map[string]string{
		"APS_EMAIL_FROM": "ops@example.com",
	}, stubOptions{})
	if err != nil {
		t.Fatalf("list-calendars.sh failed: %v\noutput: %s", err, out)
	}

	assertArgv(t, invs[0], "user", "ops@example.com", "show", "calendars", "formatjson")
}

// TestGamListEvents_RequestsJSON covers the same for showevents, plus
// the time-range keywords.
func TestGamListEvents_RequestsJSON(t *testing.T) {
	requireBash(t)
	t.Parallel()

	invs, out, err := runBackendScriptMulti(t, gamScript("list-events"), map[string]string{
		"APS_EMAIL_FROM": "ops@example.com",
		"CAL_START":      "2026-08-01",
		"CAL_END":        "2026-08-31",
	}, stubOptions{})
	if err != nil {
		t.Fatalf("list-events.sh failed: %v\noutput: %s", err, out)
	}

	assertArgv(t, invs[0],
		"calendar", "ops@example.com", "showevents",
		"timemin", "2026-08-01", "timemax", "2026-08-31", "formatjson")
}

// TestCalendarMissingRequiredVars covers the ${VAR:?} guards across
// both backends: no backend call may happen with a required field
// absent.
func TestCalendarMissingRequiredVars(t *testing.T) {
	requireBash(t)
	t.Parallel()

	cases := []struct {
		name    string
		script  backendScript
		env     map[string]string
		missing string
	}{
		{
			name:    "gcalcli/create-event/summary",
			script:  gcalcliScript("create-event"),
			env:     map[string]string{"CAL_START": "s", "CAL_END": "e"},
			missing: "CAL_SUMMARY",
		},
		{
			name:    "gcalcli/list-events/start",
			script:  gcalcliScript("list-events"),
			env:     map[string]string{"CAL_END": "e"},
			missing: "CAL_START",
		},
		{
			name:    "gcalcli/free-busy/emails",
			script:  gcalcliScript("free-busy"),
			env:     map[string]string{"CAL_START": "s", "CAL_END": "e"},
			missing: "CAL_EMAILS",
		},
		{
			name:    "gam/create-event/user",
			script:  gamScript("create-event"),
			env:     map[string]string{"CAL_SUMMARY": "x", "CAL_START": "s", "CAL_END": "e"},
			missing: "APS_EMAIL_FROM",
		},
		{
			name:    "gam/delete-event/event-id",
			script:  gamScript("delete-event"),
			env:     map[string]string{"APS_EMAIL_FROM": "ops@example.com"},
			missing: "CAL_EVENT_ID",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			invs, out, err := runBackendScriptMulti(t, tc.script, tc.env, stubOptions{})
			if err == nil {
				t.Fatalf("expected failure with %s unset:\n%s",
					tc.missing, invocationsString(invs))
			}
			if len(invs) != 0 {
				t.Errorf("backend invoked despite missing %s:\n%s",
					tc.missing, invocationsString(invs))
			}
			if !strings.Contains(out, tc.missing) {
				t.Errorf("diagnostic does not name %s:\n%s", tc.missing, out)
			}
		})
	}
}
