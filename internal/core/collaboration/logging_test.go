package collaboration_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"oss-aps-cli/internal/core/collaboration"
)

func TestNewEventLogger(t *testing.T) {
	el := collaboration.NewEventLogger()
	assert.NotNil(t, el)
}

func TestEventLogger_NoPanic(t *testing.T) {
	el := collaboration.NewEventLogger()

	assert.NotPanics(t, func() {
		el.WorkspaceCreated("ws-1", "my-team", "owner-1")
		el.WorkspaceArchived("ws-1")
		el.AgentJoined("ws-1", "agent-1", collaboration.RoleContributor)
		el.AgentLeft("ws-1", "agent-1")
		el.AgentRemoved("ws-1", "agent-1", "owner-1")
		el.RoleChanged("ws-1", "agent-1", collaboration.RoleObserver)
		el.TaskCreated("ws-1", "t-1", "sender", "recipient", "analyze")
		el.TaskStatusChanged("ws-1", "t-1", collaboration.TaskCompleted)
		el.ConflictDetected("ws-1", "c-1", collaboration.ConflictWrite, "deploy-key")
		el.ConflictResolved("ws-1", "c-1", collaboration.StrategyPriority)
		el.ContextSet("ws-1", "key", "agent-1", 1)
		el.ContextDeleted("ws-1", "key", "agent-1")
		el.OperationError("test-op", fmt.Errorf("test error"), "key", "value")
	})
}
