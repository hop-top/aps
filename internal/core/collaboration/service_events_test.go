// Tests covering the T-1290 domain.Service[T] adoption:
// SessionManager and WorkspaceContext now route CRUD through
// kit/runtime/domain.Service so each mutation publishes
// pre_validated + pre_persisted (synchronous, veto-able) plus a
// post-event (created / updated / deleted, best-effort).
//
// Topic strategy: kit.runtime.entity.* are the authoritative pre/post
// events. SessionManager / WorkspaceContext additionally fan out
// aps.runtime.session.* and aps.runtime.context_variable.* aliases on
// success — see session_service.go / context_service.go.
package collaboration_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"hop.top/aps/internal/core/collaboration"
	"hop.top/kit/go/runtime/domain"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// recordingPublisher captures every publish call. Optionally vetoes
// publishes whose topic matches vetoTopic (returns vetoErr).
type recordingPublisher struct {
	mu        sync.Mutex
	events    []recordedEvent
	vetoTopic string
	vetoErr   error
}

type recordedEvent struct {
	topic   string
	source  string
	payload any
}

func (p *recordingPublisher) Publish(_ context.Context, topic, source string, payload any) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.events = append(p.events, recordedEvent{topic: topic, source: source, payload: payload})
	if p.vetoTopic != "" && topic == p.vetoTopic {
		return p.vetoErr
	}
	return nil
}

func (p *recordingPublisher) topics() []string {
	p.mu.Lock()
	defer p.mu.Unlock()
	out := make([]string, len(p.events))
	for i, e := range p.events {
		out[i] = e.topic
	}
	return out
}

const (
	// kit canonical pre/post topics — must match domain.DefaultTopics.
	kitEntityPreValidated = "kit.runtime.entity.pre_validated"
	kitEntityPrePersisted = "kit.runtime.entity.pre_persisted"
	kitEntityCreated      = "kit.runtime.entity.created"
	kitEntityUpdated      = "kit.runtime.entity.updated"
	kitEntityDeleted      = "kit.runtime.entity.deleted"
)

// --- SessionManager event coverage ----------------------------------

func TestSessionManager_PublishesCreateLifecycle(t *testing.T) {
	pub := &recordingPublisher{}
	sm := collaboration.NewSessionManager(collaboration.WithSessionPublisher(pub))

	_, err := sm.CreateSession("ws-1", "agent-1", 30*time.Second)
	require.NoError(t, err)

	got := pub.topics()
	assert.Equal(t, []string{
		kitEntityPreValidated,
		kitEntityPrePersisted,
		kitEntityCreated,
		collaboration.TopicSessionCreated, // aps alias on success
	}, got, "create must fire pre_validated, pre_persisted, kit-created, then aps alias")
}

func TestSessionManager_PublishesUpdateLifecycle(t *testing.T) {
	pub := &recordingPublisher{}
	sm := collaboration.NewSessionManager(collaboration.WithSessionPublisher(pub))

	s, err := sm.CreateSession("ws-1", "agent-1", 30*time.Second)
	require.NoError(t, err)

	// Heartbeat exercises Update through the service.
	require.NoError(t, sm.Heartbeat(s.ID, 30*time.Second))

	got := pub.topics()
	require.GreaterOrEqual(t, len(got), 8)
	assert.Equal(t, []string{
		kitEntityPreValidated,
		kitEntityPrePersisted,
		kitEntityUpdated,
		collaboration.TopicSessionUpdated,
	}, got[4:8])
}

func TestSessionManager_PublishesDeleteLifecycle(t *testing.T) {
	pub := &recordingPublisher{}
	sm := collaboration.NewSessionManager(collaboration.WithSessionPublisher(pub))

	s, err := sm.CreateSession("ws-1", "agent-1", 30*time.Second)
	require.NoError(t, err)

	require.NoError(t, sm.DeleteSession(s.ID))

	got := pub.topics()
	require.GreaterOrEqual(t, len(got), 8)
	assert.Equal(t, []string{
		kitEntityPreValidated,
		kitEntityPrePersisted,
		kitEntityDeleted,
		collaboration.TopicSessionDeleted,
	}, got[4:8])
}

func TestSessionManager_PreValidatedVetoBlocksPersistence(t *testing.T) {
	vetoErr := errors.New("policy: not allowed")
	pub := &recordingPublisher{
		vetoTopic: kitEntityPreValidated,
		vetoErr:   vetoErr,
	}
	sm := collaboration.NewSessionManager(collaboration.WithSessionPublisher(pub))

	_, err := sm.CreateSession("ws-1", "agent-1", 30*time.Second)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "pre-validated veto")

	// Veto must short-circuit before validate / persist / post-event /
	// alias.
	got := pub.topics()
	assert.Equal(t, []string{kitEntityPreValidated}, got)

	// Aps-side alias must not have fired either.
	for _, topic := range got {
		assert.NotEqual(t, collaboration.TopicSessionCreated, topic)
	}
}

func TestSessionManager_PrePersistedVetoBlocksWrite(t *testing.T) {
	vetoErr := errors.New("policy: not allowed")
	pub := &recordingPublisher{
		vetoTopic: kitEntityPrePersisted,
		vetoErr:   vetoErr,
	}
	sm := collaboration.NewSessionManager(collaboration.WithSessionPublisher(pub))

	_, err := sm.CreateSession("ws-1", "agent-1", 30*time.Second)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "pre-persisted veto")

	got := pub.topics()
	// pre_validated fired, validation succeeded, pre_persisted vetoed,
	// repo write skipped, post-event skipped, alias skipped.
	assert.Equal(t, []string{
		kitEntityPreValidated,
		kitEntityPrePersisted,
	}, got)
}

func TestSessionManager_PreEventCarriesOpAndPhase(t *testing.T) {
	pub := &recordingPublisher{}
	sm := collaboration.NewSessionManager(collaboration.WithSessionPublisher(pub))

	_, err := sm.CreateSession("ws-1", "agent-1", 30*time.Second)
	require.NoError(t, err)

	pub.mu.Lock()
	defer pub.mu.Unlock()
	require.GreaterOrEqual(t, len(pub.events), 3)

	pre, ok := pub.events[0].payload.(domain.PreEntityPayload)
	require.True(t, ok, "pre_validated payload must be PreEntityPayload")
	assert.Equal(t, domain.OpCreate, pre.Op)
	assert.Equal(t, domain.PhasePreValidated, pre.Phase)

	pp, ok := pub.events[1].payload.(domain.PreEntityPayload)
	require.True(t, ok, "pre_persisted payload must be PreEntityPayload")
	assert.Equal(t, domain.OpCreate, pp.Op)
	assert.Equal(t, domain.PhasePrePersisted, pp.Phase)
}

// --- WorkspaceContext event coverage --------------------------------

func TestWorkspaceContext_PublishesSetCreate(t *testing.T) {
	pub := &recordingPublisher{}
	wc := collaboration.NewWorkspaceContext(collaboration.WithContextPublisher(pub))

	_, err := wc.Set("k", "v", "agent-1", collaboration.RoleOwner)
	require.NoError(t, err)

	got := pub.topics()
	assert.Equal(t, []string{
		kitEntityPreValidated,
		kitEntityPrePersisted,
		kitEntityCreated,
		collaboration.TopicContextVariableCreated,
	}, got)
}

func TestWorkspaceContext_PublishesSetUpdate(t *testing.T) {
	pub := &recordingPublisher{}
	wc := collaboration.NewWorkspaceContext(collaboration.WithContextPublisher(pub))

	_, err := wc.Set("k", "v1", "agent-1", collaboration.RoleOwner)
	require.NoError(t, err)
	_, err = wc.Set("k", "v2", "agent-1", collaboration.RoleOwner)
	require.NoError(t, err)

	got := pub.topics()
	require.Len(t, got, 8)
	assert.Equal(t, []string{
		kitEntityPreValidated,
		kitEntityPrePersisted,
		kitEntityUpdated,
		collaboration.TopicContextVariableUpdated,
	}, got[4:8])
}

func TestWorkspaceContext_PublishesDelete(t *testing.T) {
	pub := &recordingPublisher{}
	wc := collaboration.NewWorkspaceContext(collaboration.WithContextPublisher(pub))

	_, err := wc.Set("k", "v", "agent-1", collaboration.RoleOwner)
	require.NoError(t, err)
	require.NoError(t, wc.Delete("k", "agent-1", collaboration.RoleOwner))

	got := pub.topics()
	require.Len(t, got, 8)
	assert.Equal(t, []string{
		kitEntityPreValidated,
		kitEntityPrePersisted,
		kitEntityDeleted,
		collaboration.TopicContextVariableDeleted,
	}, got[4:8])
}

func TestWorkspaceContext_PreValidatedVetoBlocksPersistence(t *testing.T) {
	vetoErr := errors.New("policy: not allowed")
	pub := &recordingPublisher{
		vetoTopic: kitEntityPreValidated,
		vetoErr:   vetoErr,
	}
	wc := collaboration.NewWorkspaceContext(collaboration.WithContextPublisher(pub))

	_, err := wc.Set("k", "v", "agent-1", collaboration.RoleOwner)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "pre-validated veto")

	// Variable must NOT have been persisted.
	_, ok := wc.Get("k")
	assert.False(t, ok)
}

func TestWorkspaceContext_ACLDenialShortCircuitsBeforePublish(t *testing.T) {
	pub := &recordingPublisher{}
	wc := collaboration.NewWorkspaceContext(collaboration.WithContextPublisher(pub))

	// Observers cannot write per DefaultACL.
	_, err := wc.Set("k", "v", "obs-1", collaboration.RoleObserver)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "permission denied")

	// ACL denial must short-circuit BEFORE the service publishes
	// anything: callers wouldn't expect a denied operation to surface
	// in the audit subscriber's log.
	assert.Empty(t, pub.topics())
}
