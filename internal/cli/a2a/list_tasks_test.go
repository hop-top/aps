package a2a

import (
	"testing"
	"time"

	a2a "github.com/a2aproject/a2a-go/a2a"

	"hop.top/aps/internal/cli/listing"
)

// TestTaskToSummaryRow_HappyPath asserts ID, Status, Profile,
// Recipient (from Metadata), Messages, UpdatedAt all populate.
func TestTaskToSummaryRow_HappyPath(t *testing.T) {
	ts := time.Date(2026, 5, 4, 15, 30, 0, 0, time.UTC)
	task := &a2a.Task{
		ID: a2a.TaskID("task-abc"),
		Status: a2a.TaskStatus{
			State:     a2a.TaskState("working"),
			Timestamp: &ts,
		},
		History: []*a2a.Message{
			{ID: "m1", Role: a2a.MessageRoleUser},
			{ID: "m2", Role: a2a.MessageRoleAgent},
		},
		Metadata: map[string]any{"recipient": "worker-bee"},
	}

	row := taskToSummaryRow(task, "noor")

	if row.ID != "task-abc" {
		t.Errorf("ID = %q, want task-abc", row.ID)
	}
	if row.Status != "working" {
		t.Errorf("Status = %q, want working", row.Status)
	}
	if row.Profile != "noor" {
		t.Errorf("Profile = %q, want noor", row.Profile)
	}
	if row.Recipient != "worker-bee" {
		t.Errorf("Recipient = %q, want worker-bee", row.Recipient)
	}
	if row.Messages != 2 {
		t.Errorf("Messages = %d, want 2", row.Messages)
	}
	if row.UpdatedAt != "2026-05-04 15:30:00" {
		t.Errorf("UpdatedAt = %q, want 2026-05-04 15:30:00", row.UpdatedAt)
	}
}

// TestTaskToSummaryRow_RecipientMissing keeps the column empty when
// the sender did not record it in Metadata.
func TestTaskToSummaryRow_RecipientMissing(t *testing.T) {
	row := taskToSummaryRow(&a2a.Task{
		ID:     a2a.TaskID("task-no-meta"),
		Status: a2a.TaskStatus{State: a2a.TaskState("submitted")},
	}, "noor")

	if row.Recipient != "" {
		t.Errorf("Recipient = %q, want empty", row.Recipient)
	}
	if row.UpdatedAt != "" {
		t.Errorf("UpdatedAt = %q, want empty", row.UpdatedAt)
	}
}

// TestStatusFilter_Match exercises the listing.MatchString predicate on
// the same shape the RunE wires up.
func TestStatusFilter_Match(t *testing.T) {
	rows := []a2aTaskSummaryRow{
		{ID: "1", Status: "working"},
		{ID: "2", Status: "completed"},
		{ID: "3", Status: "working"},
	}
	pred := listing.MatchString(func(r a2aTaskSummaryRow) string { return r.Status }, "working")
	got := listing.Filter(rows, pred)
	if len(got) != 2 {
		t.Fatalf("filtered len = %d, want 2", len(got))
	}
	for _, r := range got {
		if r.Status != "working" {
			t.Errorf("unexpected row %+v", r)
		}
	}
}

// TestStatusFilter_Empty — unset --status matches every row.
func TestStatusFilter_Empty(t *testing.T) {
	rows := []a2aTaskSummaryRow{
		{ID: "1", Status: "working"},
		{ID: "2", Status: "completed"},
	}
	pred := listing.All(
		listing.MatchString(func(r a2aTaskSummaryRow) string { return r.Status }, ""),
	)
	got := listing.Filter(rows, pred)
	if len(got) != 2 {
		t.Fatalf("filtered len = %d, want 2 (empty status = match-all)", len(got))
	}
}
