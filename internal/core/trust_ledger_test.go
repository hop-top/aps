package core

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHasRole(t *testing.T) {
	t.Parallel()
	p := &Profile{ID: "test", Roles: []string{"owner", "evaluator"}}

	assert.True(t, p.HasRole("owner"))
	assert.True(t, p.HasRole("evaluator"))
	assert.False(t, p.HasRole("auditor"))
	assert.False(t, p.HasRole(""))
}

func TestHasRole_Empty(t *testing.T) {
	t.Parallel()
	p := &Profile{ID: "test"}
	assert.False(t, p.HasRole("owner"))
}

func TestAddRole_Dedup(t *testing.T) {
	t.Parallel()
	p := &Profile{ID: "test"}

	p.AddRole("owner")
	p.AddRole("owner")
	assert.Equal(t, []string{"owner"}, p.Roles)
}

func TestRemoveRole(t *testing.T) {
	t.Parallel()
	p := &Profile{ID: "test", Roles: []string{"owner", "auditor"}}

	p.RemoveRole("owner")
	assert.Equal(t, []string{"auditor"}, p.Roles)
}

func TestRemoveRole_NotPresent(t *testing.T) {
	t.Parallel()
	p := &Profile{ID: "test", Roles: []string{"owner"}}

	p.RemoveRole("evaluator")
	assert.Equal(t, []string{"owner"}, p.Roles)
}

func TestIsValidRole(t *testing.T) {
	t.Parallel()
	assert.True(t, IsValidRole("owner"))
	assert.True(t, IsValidRole("assignee"))
	assert.True(t, IsValidRole("evaluator"))
	assert.True(t, IsValidRole("auditor"))
	assert.False(t, IsValidRole("admin"))
	assert.False(t, IsValidRole(""))
}

func TestIsValidTrustDomain(t *testing.T) {
	t.Parallel()
	assert.True(t, IsValidTrustDomain("hooks"))
	assert.True(t, IsValidTrustDomain("general"))
	assert.False(t, IsValidTrustDomain("unknown"))
}

func TestEnsureTrustLedger(t *testing.T) {
	t.Parallel()
	p := &Profile{ID: "test"}
	require.Nil(t, p.TrustLedger)

	p.EnsureTrustLedger()
	require.NotNil(t, p.TrustLedger)
	require.NotNil(t, p.TrustLedger.Scores)

	// idempotent
	p.EnsureTrustLedger()
	require.NotNil(t, p.TrustLedger.Scores)
}

func TestRecordTrust(t *testing.T) {
	t.Parallel()
	p := &Profile{ID: "test"}

	now := time.Date(2026, 4, 24, 12, 0, 0, 0, time.UTC)
	p.RecordTrust(TrustEntry{
		TaskRef:    "T-0001",
		Domain:     "hooks",
		Difficulty: "M",
		Timestamp:  now,
		Delta:      1.5,
		Breakdown: []TrustBreakdown{
			{Label: "correctness", Value: 1.0},
			{Label: "timeliness", Value: 0.5},
		},
	})

	assert.Equal(t, 1.5, p.TrustScore("hooks"))
	assert.Len(t, p.TrustHistory(""), 1)
	assert.Equal(t, "T-0001", p.TrustHistory("")[0].TaskRef)
}

func TestRecordTrust_Multiple(t *testing.T) {
	t.Parallel()
	p := &Profile{ID: "test"}

	p.RecordTrust(TrustEntry{
		TaskRef: "T-0001", Domain: "hooks", Delta: 2.0,
	})
	p.RecordTrust(TrustEntry{
		TaskRef: "T-0002", Domain: "hooks", Delta: -0.5,
	})
	p.RecordTrust(TrustEntry{
		TaskRef: "T-0003", Domain: "skills", Delta: 3.0,
	})

	assert.Equal(t, 1.5, p.TrustScore("hooks"))
	assert.Equal(t, 3.0, p.TrustScore("skills"))
	assert.Len(t, p.TrustHistory(""), 3)
	assert.Len(t, p.TrustHistory("hooks"), 2)
	assert.Len(t, p.TrustHistory("skills"), 1)
}

func TestRecordTrust_DefaultTimestamp(t *testing.T) {
	t.Parallel()
	p := &Profile{ID: "test"}

	before := time.Now().UTC()
	p.RecordTrust(TrustEntry{
		TaskRef: "T-0001", Domain: "general", Delta: 1.0,
	})

	entry := p.TrustHistory("")[0]
	assert.False(t, entry.Timestamp.IsZero())
	assert.True(t, entry.Timestamp.After(before) ||
		entry.Timestamp.Equal(before))
}

func TestTrustScore_NilLedger(t *testing.T) {
	t.Parallel()
	p := &Profile{ID: "test"}
	assert.Equal(t, 0.0, p.TrustScore("hooks"))
}

func TestTrustHistory_NilLedger(t *testing.T) {
	t.Parallel()
	p := &Profile{ID: "test"}
	assert.Nil(t, p.TrustHistory(""))
	assert.Nil(t, p.TrustHistory("hooks"))
}

func TestRolesRoundTrip(t *testing.T) {
	t.Setenv("APS_DATA_PATH", t.TempDir())

	p := Profile{
		ID:          "roles-rt",
		DisplayName: "Roles RT",
		Roles:       []string{"evaluator", "auditor"},
	}
	require.NoError(t, SaveProfile(&p))

	loaded, err := LoadProfile("roles-rt")
	require.NoError(t, err)
	assert.True(t, loaded.HasRole("evaluator"))
	assert.True(t, loaded.HasRole("auditor"))
	assert.False(t, loaded.HasRole("owner"))
}

func TestTrustLedgerRoundTrip(t *testing.T) {
	t.Setenv("APS_DATA_PATH", t.TempDir())

	p := Profile{
		ID:          "ledger-rt",
		DisplayName: "Ledger RT",
	}
	ts := time.Date(2026, 4, 24, 12, 0, 0, 0, time.UTC)
	p.RecordTrust(TrustEntry{
		TaskRef:    "T-0300",
		Domain:     "skills",
		Difficulty: "L",
		Timestamp:  ts,
		Delta:      2.0,
	})
	require.NoError(t, SaveProfile(&p))

	loaded, err := LoadProfile("ledger-rt")
	require.NoError(t, err)
	require.NotNil(t, loaded.TrustLedger)
	assert.Equal(t, 2.0, loaded.TrustScore("skills"))
	require.Len(t, loaded.TrustLedger.History, 1)
	assert.Equal(t, "T-0300", loaded.TrustLedger.History[0].TaskRef)
}
