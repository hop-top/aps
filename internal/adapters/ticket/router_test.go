package ticket

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"hop.top/aps/internal/core/protocol"
)

func TestRouterRoutesSpecificThreadBeforeChannel(t *testing.T) {
	router := NewRouter(StaticRouteResolver{
		"jira:OPS#OPS-7": "triage=deep-dive",
		"jira:OPS":       "triage=inbox",
	})

	result, err := router.Route(context.Background(), &NormalizedTicket{
		ID:        "OPS-7",
		Adapter:   AdapterJira,
		Kind:      TicketKindIssue,
		ChannelID: "OPS",
		ThreadID:  "OPS-7",
		Author:    Actor{ID: "acc-1"},
	})

	require.NoError(t, err)
	assert.Equal(t, "routed", result.Status)
	assert.Equal(t, "triage", result.ProfileID)
	assert.Equal(t, "deep-dive", result.ActionName)
	assert.Equal(t, "triage=deep-dive", result.Route)
}

func TestRouterFallsBackToChannelRoute(t *testing.T) {
	router := NewRouter(StaticRouteResolver{
		"linear:ENG": "worker=triage",
	})

	result, err := router.Route(context.Background(), &NormalizedTicket{
		ID:        "ENG-42",
		Adapter:   AdapterLinear,
		Kind:      TicketKindIssue,
		ChannelID: "ENG",
		ThreadID:  "ENG-42",
		Author:    Actor{ID: "u-1"},
	})

	require.NoError(t, err)
	assert.Equal(t, "routed", result.Status)
	assert.Equal(t, "worker", result.ProfileID)
	assert.Equal(t, "triage", result.ActionName)
}

type recordingExecutor struct {
	inputs []protocol.RunInput
	status protocol.RunStatus
	output string
	err    error
}

func (e *recordingExecutor) ExecuteRun(_ context.Context, input protocol.RunInput, _ protocol.StreamWriter) (*protocol.RunState, error) {
	e.inputs = append(e.inputs, input)
	if e.err != nil {
		return nil, e.err
	}
	status := e.status
	if status == "" {
		status = protocol.RunStatusCompleted
	}
	return &protocol.RunState{ProfileID: input.ProfileID, ActionID: input.ActionID, Status: status, Output: e.output}, nil
}

func TestRouterHandleTicketExecutesRoutedAction(t *testing.T) {
	executor := &recordingExecutor{output: "reviewed"}
	router := NewRouterWithExecutor(StaticRouteResolver{
		"gitlab:platform/api": "maintainer=review",
	}, executor)

	ticket := &NormalizedTicket{
		ID:        "99",
		Adapter:   AdapterGitLab,
		Kind:      TicketKindComment,
		ChannelID: "platform/api",
		ThreadID:  "12",
		Title:     "Flaky pipeline",
		Body:      "Retry does not help.",
		Author:    Actor{ID: "9", Handle: "nia"},
	}
	result, err := router.HandleTicket(context.Background(), ticket)

	require.NoError(t, err)
	assert.Equal(t, "success", result.Status)
	assert.Equal(t, "reviewed", result.Output)
	require.Len(t, executor.inputs, 1)
	input := executor.inputs[0]
	assert.Equal(t, "maintainer", input.ProfileID)
	assert.Equal(t, "review", input.ActionID)
	assert.Equal(t, "platform/api#12", input.ThreadID)

	var payload map[string]any
	require.NoError(t, json.Unmarshal(input.Payload, &payload))
	assert.Equal(t, "99", payload["id"])
	assert.Equal(t, "gitlab", payload["adapter"])
	assert.Equal(t, "comment", payload["kind"])
	assert.Equal(t, "Flaky pipeline", payload["title"])
	assert.Equal(t, "Retry does not help.", payload["body"])
	assert.Equal(t, "platform/api#12", payload["route_key"])
	author, _ := payload["author"].(map[string]any)
	assert.Equal(t, "nia", author["handle"])
	state, ok := result.OutputData.(*protocol.RunState)
	require.True(t, ok, "OutputData should carry the run state")
	assert.Equal(t, "review", state.ActionID)
}

func TestRouterExecuteActionReportsFailedRun(t *testing.T) {
	executor := &recordingExecutor{status: protocol.RunStatusFailed, output: "exit status 1"}
	router := NewRouterWithExecutor(StaticRouteResolver{"jira:OPS": "triage=inbox"}, executor)

	result, err := router.HandleTicket(context.Background(), &NormalizedTicket{
		ID: "OPS-7", Adapter: AdapterJira, Kind: TicketKindIssue, ChannelID: "OPS", Author: Actor{ID: "acc-1"},
	})

	require.NoError(t, err)
	assert.Equal(t, "failed", result.Status)
	assert.Equal(t, "exit status 1", result.Output)
}

func TestRouterExecuteActionPropagatesExecutorError(t *testing.T) {
	executor := &recordingExecutor{err: errors.New("profile not found")}
	router := NewRouterWithExecutor(StaticRouteResolver{"jira:OPS": "triage=inbox"}, executor)

	result, err := router.HandleTicket(context.Background(), &NormalizedTicket{
		ID: "OPS-7", Adapter: AdapterJira, Kind: TicketKindIssue, ChannelID: "OPS", Author: Actor{ID: "acc-1"},
	})

	require.Error(t, err)
	assert.Equal(t, "failed", result.Status)
	assert.Contains(t, result.Output, "profile not found")
}

func TestRouterWithoutExecutorRefusesToExecute(t *testing.T) {
	router := NewRouterWithExecutor(StaticRouteResolver{"jira:OPS": "triage=inbox"}, nil)

	_, err := router.ExecuteAction(context.Background(), "triage", "inbox", &NormalizedTicket{
		ID: "OPS-7", Adapter: AdapterJira, Kind: TicketKindIssue, ChannelID: "OPS", Author: Actor{ID: "acc-1"},
	})

	require.Error(t, err)
}

func TestRouterHandleTicketDoesNotExecuteUnrouted(t *testing.T) {
	executor := &recordingExecutor{}
	router := NewRouterWithExecutor(StaticRouteResolver{}, executor)

	result, err := router.HandleTicket(context.Background(), &NormalizedTicket{
		ID: "OPS-7", Adapter: AdapterJira, Kind: TicketKindIssue, ChannelID: "OPS", Author: Actor{ID: "acc-1"},
	})

	require.NoError(t, err)
	assert.Equal(t, "failed", result.Status)
	assert.True(t, IsUnrouted(result.Error), "result.Error should be the unrouted sentinel")
	assert.Empty(t, executor.inputs)
}

func TestRouterReturnsUnroutedResult(t *testing.T) {
	router := NewRouter(StaticRouteResolver{})

	result, err := router.Route(context.Background(), &NormalizedTicket{
		ID:        "OPS-7",
		Adapter:   AdapterJira,
		Kind:      TicketKindIssue,
		ChannelID: "OPS",
		Author:    Actor{ID: "acc-1"},
	})

	require.NoError(t, err)
	assert.Equal(t, "unrouted", result.Status)
	assert.Error(t, result.Error)
}
