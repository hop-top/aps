package ticket

import (
	"fmt"
	"strings"
	"time"
)

const (
	AdapterJira   = "jira"
	AdapterLinear = "linear"
	AdapterGitLab = "gitlab"

	TicketKindIssue        = "issue"
	TicketKindComment      = "comment"
	TicketKindMergeRequest = "merge_request"
)

// AdapterDefinition describes a ticket adapter's user-facing service shape.
type AdapterDefinition struct {
	Name           string
	Options        []string
	Receives       string
	Executes       string
	Replies        string
	RouteKeys      []string
	ReplyBehaviors []string
	Maturity       string
}

// NormalizedTicket is the common ticket/work-item shape used by Jira, Linear,
// and GitLab before routing to a profile action.
type NormalizedTicket struct {
	ID          string
	Adapter     string
	Kind        string
	Action      string
	WorkspaceID string
	ProjectID   string
	ChannelID   string
	ThreadID    string
	ThreadType  string
	Title       string
	Body        string
	URL         string
	State       string
	Author      Actor
	Labels      []string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	Metadata    map[string]any
}

type Actor struct {
	ID     string
	Name   string
	Handle string
	Email  string
}

type TargetAction struct {
	ProfileID  string
	ActionName string
}

type RoutingResult struct {
	TicketID   string
	ProfileID  string
	ActionName string
	Route      string
	Status     string
	Error      error
}

type RouteResolver interface {
	ResolveTicketRoute(adapter, routeKey string) (string, error)
}

type StaticRouteResolver map[string]string

func (r StaticRouteResolver) ResolveTicketRoute(adapter, routeKey string) (string, error) {
	if target, ok := r[adapter+":"+routeKey]; ok {
		return target, nil
	}
	if target, ok := r[routeKey]; ok {
		return target, nil
	}
	return "", fmt.Errorf("no route for %s:%s", adapter, routeKey)
}

type ActionResult struct {
	Status        string
	Output        string
	OutputData    any
	ExecutionTime time.Duration
	Error         error
}

func (t *NormalizedTicket) Validate() error {
	if t.ID == "" {
		return fmt.Errorf("ticket ID is required")
	}
	if t.Adapter == "" {
		return fmt.Errorf("ticket adapter is required")
	}
	if t.Kind == "" {
		return fmt.Errorf("ticket kind is required")
	}
	if t.ChannelID == "" {
		return fmt.Errorf("ticket channel is required")
	}
	if t.Author.ID == "" {
		return fmt.Errorf("ticket author is required")
	}
	return nil
}

func AdapterDefinitionFor(name string) (AdapterDefinition, bool) {
	def, ok := adapterDefinitions[strings.ToLower(strings.TrimSpace(name))]
	return def, ok
}

func ParseTargetAction(mapping string) (TargetAction, error) {
	sep := "="
	if !strings.Contains(mapping, "=") && strings.Contains(mapping, ":") {
		sep = ":"
	}
	profileID, actionName, ok := strings.Cut(mapping, sep)
	if !ok || profileID == "" || actionName == "" {
		return TargetAction{}, fmt.Errorf("invalid route target %q: expected profile=action", mapping)
	}
	return TargetAction{ProfileID: profileID, ActionName: actionName}, nil
}

func (t TargetAction) String() string {
	return t.ProfileID + "=" + t.ActionName
}

var adapterDefinitions = map[string]AdapterDefinition{
	AdapterJira: {
		Name:           AdapterJira,
		Options:        []string{"env:JIRA_TOKEN", "site", "project", "jql", "default-action", "reply"},
		Receives:       "Jira issue and comment webhooks or queried issues",
		Executes:       "routed profile action with normalized issue/comment payload",
		Replies:        "Jira comment body when reply=comment or auto; status metadata when reply=status",
		RouteKeys:      []string{"project", "issue"},
		ReplyBehaviors: []string{"comment", "status", "auto", "none"},
		Maturity:       "component",
	},
	AdapterLinear: {
		Name:           AdapterLinear,
		Options:        []string{"env:LINEAR_API_KEY", "workspace", "team", "project", "default-action", "reply"},
		Receives:       "Linear issue and comment webhooks",
		Executes:       "routed profile action with normalized issue/comment payload",
		Replies:        "Linear comment body when reply=comment or auto; status metadata when reply=status",
		RouteKeys:      []string{"team", "project", "issue"},
		ReplyBehaviors: []string{"comment", "status", "auto", "none"},
		Maturity:       "component",
	},
	AdapterGitLab: {
		Name:           AdapterGitLab,
		Options:        []string{"env:GITLAB_TOKEN", "project", "group", "events", "default-action", "reply"},
		Receives:       "GitLab issue, merge request, and note webhooks",
		Executes:       "routed profile action with normalized issue/MR/comment payload",
		Replies:        "GitLab note body when reply=comment or auto; status metadata when reply=status",
		RouteKeys:      []string{"project", "group", "issue", "merge_request"},
		ReplyBehaviors: []string{"comment", "status", "auto", "none"},
		Maturity:       "component",
	},
}
