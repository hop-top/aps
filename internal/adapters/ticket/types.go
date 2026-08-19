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
	AdapterEmail  = "email"

	TicketKindIssue        = "issue"
	TicketKindComment      = "comment"
	TicketKindMergeRequest = "merge_request"
	// TicketKindEmail is the thread type of email tickets: the mailbox is the
	// channel, the thread root Message-ID is the thread.
	TicketKindEmail = "email"

	// ServiceType is the persisted service type ticket adapters serve.
	ServiceType = "ticket"

	// StatusSuccess and StatusFailed are the ActionResult.Status values.
	StatusSuccess = "success"
	StatusFailed  = "failed"

	optionDefaultAction = "default-action"
	maturityComponent   = "component"

	// MetadataServiceID is the NormalizedTicket.Metadata key carrying the
	// ticket service that received the event.
	MetadataServiceID = "service_id"
	// MetadataRouting is the NormalizedTicket.Metadata key carrying the route
	// table decision when the service dispatches by sender.
	MetadataRouting = "routing"
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
// GitLab, and email before routing to a profile action. It is also the JSON
// document a routed action reads on stdin (see ActionPayload).
type NormalizedTicket struct {
	ID          string         `json:"id"`
	Adapter     string         `json:"adapter"`
	Kind        string         `json:"kind"`
	Action      string         `json:"action,omitempty"`
	WorkspaceID string         `json:"workspace_id,omitempty"`
	ProjectID   string         `json:"project_id,omitempty"`
	ChannelID   string         `json:"channel_id"`
	ThreadID    string         `json:"thread_id,omitempty"`
	ThreadType  string         `json:"thread_type,omitempty"`
	Title       string         `json:"title,omitempty"`
	Body        string         `json:"body,omitempty"`
	URL         string         `json:"url,omitempty"`
	State       string         `json:"state,omitempty"`
	Author      Actor          `json:"author"`
	Labels      []string       `json:"labels,omitempty"`
	CreatedAt   time.Time      `json:"created_at,omitempty"`
	UpdatedAt   time.Time      `json:"updated_at,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

type Actor struct {
	ID     string `json:"id"`
	Name   string `json:"name,omitempty"`
	Handle string `json:"handle,omitempty"`
	Email  string `json:"email,omitempty"`
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

// replyBehaviors are the reply modes every ticket adapter understands.
var replyBehaviors = []string{"comment", "status", "auto", "none"} //nolint:goconst // option catalogue; literals are the data

//nolint:goconst // option catalogue; literals are the data
var adapterDefinitions = map[string]AdapterDefinition{
	AdapterEmail: {
		Name:           AdapterEmail,
		Options:        []string{optionDefaultAction, "reply", "allowed_senders", "auth_token_env", "signature_secret_env"},
		Receives:       "inbound email events posted by a mail relay or poller",
		Executes:       "routed profile action with normalized email payload",
		Replies:        "reply body when reply=comment or auto; status metadata when reply=status",
		RouteKeys:      []string{"mailbox", "thread"},
		ReplyBehaviors: replyBehaviors,
		Maturity:       maturityComponent,
	},
	AdapterJira: {
		Name:           AdapterJira,
		Options:        []string{"env:JIRA_TOKEN", "site", "project", "jql", optionDefaultAction, "reply"},
		Receives:       "Jira issue and comment webhooks or queried issues",
		Executes:       "routed profile action with normalized issue/comment payload",
		Replies:        "Jira comment body when reply=comment or auto; status metadata when reply=status",
		RouteKeys:      []string{"project", "issue"},
		ReplyBehaviors: replyBehaviors,
		Maturity:       maturityComponent,
	},
	AdapterLinear: {
		Name:           AdapterLinear,
		Options:        []string{"env:LINEAR_API_KEY", "workspace", "team", "project", optionDefaultAction, "reply"},
		Receives:       "Linear issue and comment webhooks",
		Executes:       "routed profile action with normalized issue/comment payload",
		Replies:        "Linear comment body when reply=comment or auto; status metadata when reply=status",
		RouteKeys:      []string{"team", "project", "issue"},
		ReplyBehaviors: replyBehaviors,
		Maturity:       maturityComponent,
	},
	AdapterGitLab: {
		Name:           AdapterGitLab,
		Options:        []string{"env:GITLAB_TOKEN", "project", "group", "events", optionDefaultAction, "reply"},
		Receives:       "GitLab issue, merge request, and note webhooks",
		Executes:       "routed profile action with normalized issue/MR/comment payload",
		Replies:        "GitLab note body when reply=comment or auto; status metadata when reply=status",
		RouteKeys:      []string{"project", "group", "issue", "merge_request"},
		ReplyBehaviors: replyBehaviors,
		Maturity:       maturityComponent,
	},
}
