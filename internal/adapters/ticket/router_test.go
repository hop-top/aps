package ticket

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestRouterHandleTicketReplySemanticsAreStatusOnlyPlaceholder(t *testing.T) {
	router := NewRouter(StaticRouteResolver{
		"gitlab:platform/api": "maintainer=review",
	})

	result, err := router.HandleTicket(context.Background(), &NormalizedTicket{
		ID:        "99",
		Adapter:   AdapterGitLab,
		Kind:      TicketKindComment,
		ChannelID: "platform/api",
		ThreadID:  "12",
		Author:    Actor{ID: "9"},
	})

	require.NoError(t, err)
	assert.Equal(t, "success", result.Status)
	assert.Contains(t, result.Output, `action "review" dispatched to profile "maintainer"`)
	assert.Contains(t, result.Output, "adapter gitlab")
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
