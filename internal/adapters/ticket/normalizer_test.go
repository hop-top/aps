package ticket

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAdapterDefinitionForTicketAdapters(t *testing.T) {
	tests := []struct {
		adapter string
		option  string
		reply   string
	}{
		{AdapterJira, "jql", "Jira comment body"},
		{AdapterLinear, "team", "Linear comment body"},
		{AdapterGitLab, "group", "GitLab note body"},
	}

	for _, tt := range tests {
		t.Run(tt.adapter, func(t *testing.T) {
			def, ok := AdapterDefinitionFor(tt.adapter)
			require.True(t, ok)
			assert.Contains(t, def.Options, tt.option)
			assert.Contains(t, def.Replies, tt.reply)
			assert.Contains(t, def.ReplyBehaviors, "comment")
		})
	}
}

func TestNormalizerNormalizeJiraComment(t *testing.T) {
	n := NewNormalizer()
	got, err := n.Normalize(AdapterJira, map[string]any{
		"webhookEvent": "comment_created",
		"user": map[string]any{
			"accountId":   "acc-1",
			"displayName": "Nia",
		},
		"issue": map[string]any{
			"id":  "10001",
			"key": "OPS-7",
			"fields": map[string]any{
				"summary":     "Deploy failed",
				"description": "Deploy logs attached",
				"labels":      []any{"deploy", "sev2"},
				"project": map[string]any{
					"key": "OPS",
				},
				"status": map[string]any{
					"name": "In Progress",
				},
			},
		},
		"comment": map[string]any{
			"id":   "500",
			"body": "Can APS triage this?",
			"author": map[string]any{
				"accountId":   "acc-2",
				"displayName": "Omar",
			},
		},
	})

	require.NoError(t, err)
	assert.Equal(t, AdapterJira, got.Adapter)
	assert.Equal(t, TicketKindComment, got.Kind)
	assert.Equal(t, "500", got.ID)
	assert.Equal(t, "OPS", got.ChannelID)
	assert.Equal(t, "OPS-7", got.ThreadID)
	assert.Equal(t, "Deploy failed", got.Title)
	assert.Equal(t, "Can APS triage this?", got.Body)
	assert.Equal(t, "Omar", got.Author.Name)
	assert.Equal(t, []string{"deploy", "sev2"}, got.Labels)
}

func TestNormalizerNormalizeLinearIssue(t *testing.T) {
	n := NewNormalizer()
	got, err := n.Normalize(AdapterLinear, map[string]any{
		"action":         "create",
		"type":           "Issue",
		"organizationId": "ws-1",
		"actor": map[string]any{
			"id":   "u-1",
			"name": "Priya",
		},
		"data": map[string]any{
			"id":          "issue-id",
			"identifier":  "ENG-42",
			"title":       "Queue retries stall",
			"description": "Retries stop after one attempt",
			"url":         "https://linear.app/acme/issue/ENG-42",
			"team": map[string]any{
				"key": "ENG",
			},
			"project": map[string]any{
				"id":   "proj-1",
				"name": "Reliability",
			},
			"state": map[string]any{
				"name": "Todo",
			},
			"labels": []any{
				map[string]any{"name": "backend"},
			},
		},
	})

	require.NoError(t, err)
	assert.Equal(t, AdapterLinear, got.Adapter)
	assert.Equal(t, TicketKindIssue, got.Kind)
	assert.Equal(t, "ENG", got.ChannelID)
	assert.Equal(t, "ENG-42", got.ThreadID)
	assert.Equal(t, "ws-1", got.WorkspaceID)
	assert.Equal(t, "Queue retries stall", got.Title)
	assert.Equal(t, []string{"backend"}, got.Labels)
}

func TestNormalizerNormalizeGitLabMergeRequestNote(t *testing.T) {
	n := NewNormalizer()
	got, err := n.Normalize(AdapterGitLab, map[string]any{
		"object_kind": "note",
		"user": map[string]any{
			"id":       float64(9),
			"username": "sam",
			"name":     "Sam",
		},
		"project": map[string]any{
			"id":                  float64(2),
			"path_with_namespace": "platform/api",
		},
		"object_attributes": map[string]any{
			"id":     float64(99),
			"note":   "Please review the migration path.",
			"action": "create",
		},
		"merge_request": map[string]any{
			"iid":   float64(12),
			"title": "Add service UX",
			"url":   "https://gitlab.example/platform/api/-/merge_requests/12",
		},
	})

	require.NoError(t, err)
	assert.Equal(t, AdapterGitLab, got.Adapter)
	assert.Equal(t, TicketKindComment, got.Kind)
	assert.Equal(t, "platform/api", got.ChannelID)
	assert.Equal(t, "12", got.ThreadID)
	assert.Equal(t, TicketKindMergeRequest, got.ThreadType)
	assert.Equal(t, "Please review the migration path.", got.Body)
	assert.Equal(t, "sam", got.Author.Handle)
}

func TestNormalizerDenormalizeReplyPayloads(t *testing.T) {
	n := NewNormalizer()
	ticket := &NormalizedTicket{ID: "OPS-7", ThreadID: "OPS-7"}
	result := &ActionResult{Status: "success", Output: "Investigated and commented."}

	jira, err := n.Denormalize(AdapterJira, result, ticket)
	require.NoError(t, err)
	assert.Equal(t, "jira_comment", jira["target"])
	assert.Equal(t, "Investigated and commented.", jira["body"])
	assert.Equal(t, "OPS-7", jira["thread_id"])

	gitlab, err := n.Denormalize(AdapterGitLab, result, ticket)
	require.NoError(t, err)
	assert.Equal(t, "gitlab_note", gitlab["target"])
}

func TestNormalizerNormalizeEmail(t *testing.T) {
	n := NewNormalizer()
	got, err := n.Normalize(AdapterEmail, map[string]any{
		"message_id":  "<abc@example.com>",
		"in_reply_to": "<root@example.com>",
		"from":        "Alice Example <Alice@Example.com>",
		"to":          "support@example.com",
		"cc":          "ops@example.com",
		"subject":     "Re: Cannot log in",
		"body":        "Still broken.",
		"date":        "2026-08-19T10:00:00Z",
		"labels":      []any{"inbox", "urgent"},
		"attachments": []any{map[string]any{"type": "file", "url": "https://files.example/1"}},
	})

	require.NoError(t, err)
	assert.Equal(t, "<abc@example.com>", got.ID)
	assert.Equal(t, AdapterEmail, got.Adapter)
	assert.Equal(t, TicketKindComment, got.Kind, "a reply is a comment on the thread")
	assert.Equal(t, "received", got.Action)
	assert.Equal(t, "support@example.com", got.ChannelID, "mailbox is the channel")
	assert.Equal(t, "<root@example.com>", got.ThreadID, "thread root is the In-Reply-To")
	assert.Equal(t, TicketKindEmail, got.ThreadType)
	assert.Equal(t, "Re: Cannot log in", got.Title)
	assert.Equal(t, "Still broken.", got.Body)
	assert.Equal(t, "alice@example.com", got.Author.ID, "author id is the lowercased address")
	assert.Equal(t, "alice@example.com", got.Author.Email)
	assert.Equal(t, "Alice Example", got.Author.Name)
	assert.Equal(t, []string{"inbox", "urgent"}, got.Labels)
	assert.Equal(t, "2026-08-19T10:00:00Z", got.CreatedAt.UTC().Format(time.RFC3339))
	assert.Equal(t, "ops@example.com", got.Metadata["cc"])
	assert.Len(t, got.Metadata["attachments"], 1)
	assert.Equal(t, "support@example.com#<root@example.com>", RouteKey(got))
}

func TestNormalizerNormalizeEmailNewThread(t *testing.T) {
	n := NewNormalizer()
	got, err := n.Normalize(AdapterEmail, map[string]any{
		"from":    "bob@example.com",
		"to":      "support@example.com",
		"subject": "Hello",
		"body":    "Hi",
	})

	require.NoError(t, err)
	assert.Equal(t, TicketKindIssue, got.Kind, "a fresh email opens a thread")
	assert.NotEmpty(t, got.ID, "missing Message-ID is synthesized")
	assert.Equal(t, got.ID, got.ThreadID, "thread root is the message itself")
	assert.Equal(t, "Hello", got.Title)
}

func TestNormalizerNormalizeEmailRequiresSenderAndMailbox(t *testing.T) {
	n := NewNormalizer()
	_, err := n.Normalize(AdapterEmail, map[string]any{"to": "support@example.com", "body": "x"})
	require.Error(t, err)
	_, err = n.Normalize(AdapterEmail, map[string]any{"from": "bob@example.com", "body": "x"})
	require.Error(t, err)
}

func TestNormalizerDenormalizeEmail(t *testing.T) {
	n := NewNormalizer()
	out, err := n.Denormalize(AdapterEmail, &ActionResult{Status: "success", Output: "Thanks, on it."}, &NormalizedTicket{ID: "<abc@example.com>", ThreadID: "<abc@example.com>"})
	require.NoError(t, err)
	assert.Equal(t, "email_reply", out["target"])
	assert.Equal(t, "Thanks, on it.", out["body"])
	assert.Equal(t, "<abc@example.com>", out["ticket_id"])
}
