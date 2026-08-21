package ticket

import (
	"fmt"
	"net/mail"
	"strconv"
	"strings"
	"time"
)

type Normalizer struct{}

func NewNormalizer() *Normalizer {
	return &Normalizer{}
}

func (n *Normalizer) Normalize(adapter string, raw map[string]any) (*NormalizedTicket, error) {
	if raw == nil {
		return nil, fmt.Errorf("raw ticket event is nil")
	}

	var (
		ticket *NormalizedTicket
		err    error
	)
	switch strings.ToLower(strings.TrimSpace(adapter)) {
	case AdapterJira:
		ticket, err = n.normalizeJira(raw)
	case AdapterLinear:
		ticket, err = n.normalizeLinear(raw)
	case AdapterGitLab:
		ticket, err = n.normalizeGitLab(raw)
	case AdapterEmail:
		ticket, err = n.normalizeEmail(raw)
	default:
		return nil, fmt.Errorf("unsupported ticket adapter %q", adapter)
	}
	if err != nil {
		return nil, err
	}
	if err := ticket.Validate(); err != nil {
		return nil, err
	}
	return ticket, nil
}

func (n *Normalizer) normalizeJira(raw map[string]any) (*NormalizedTicket, error) {
	issue := getMap(raw, "issue")
	if issue == nil {
		return nil, fmt.Errorf("missing jira issue")
	}
	fields := getMap(issue, "fields")
	if fields == nil {
		return nil, fmt.Errorf("missing jira issue fields")
	}

	project := getMap(fields, "project")
	comment := getMap(raw, "comment")
	user := getMap(raw, "user")
	if comment != nil {
		if author := getMap(comment, "author"); author != nil {
			user = author
		}
	}

	key := firstNonEmpty(getString(issue, "key"), getString(issue, "id"))
	projectKey := firstNonEmpty(getString(project, "key"), getString(project, "id"))
	if projectKey == "" {
		return nil, fmt.Errorf("missing jira project")
	}

	kind := TicketKindIssue
	body := getString(fields, "description")
	id := key
	if comment != nil {
		kind = TicketKindComment
		id = firstNonEmpty(getString(comment, "id"), key)
		body = getString(comment, "body")
	}

	status := getMap(fields, "status")
	issueType := getMap(fields, "issuetype")

	return &NormalizedTicket{
		ID:         id,
		Adapter:    AdapterJira,
		Kind:       kind,
		Action:     getString(raw, "webhookEvent"),
		ProjectID:  projectKey,
		ChannelID:  projectKey,
		ThreadID:   key,
		ThreadType: TicketKindIssue,
		Title:      getString(fields, "summary"),
		Body:       body,
		State:      getString(status, "name"),
		Author:     jiraActor(user),
		Labels:     getStringSlice(fields, "labels"),
		CreatedAt:  parseTime(getString(fields, "created")),
		UpdatedAt:  parseTime(getString(fields, "updated")),
		Metadata:   map[string]any{"raw": raw, "issue_type": getString(issueType, "name")},
	}, nil
}

func (n *Normalizer) normalizeLinear(raw map[string]any) (*NormalizedTicket, error) {
	data := getMap(raw, "data")
	if data == nil {
		return nil, fmt.Errorf("missing linear data")
	}
	actor := getMap(raw, "actor")
	kind := TicketKindIssue
	issue := data
	body := getString(data, "description")
	if strings.EqualFold(getString(raw, "type"), "Comment") {
		kind = TicketKindComment
		body = getString(data, "body")
		if linkedIssue := getMap(data, "issue"); linkedIssue != nil {
			issue = linkedIssue
		}
	}

	team := getMap(issue, "team")
	project := getMap(issue, "project")
	state := getMap(issue, "state")
	channelID := firstNonEmpty(getString(team, "key"), getString(team, "id"), getString(project, "id"))
	if channelID == "" {
		return nil, fmt.Errorf("missing linear team or project")
	}

	identifier := firstNonEmpty(getString(issue, "identifier"), getString(issue, "id"))
	id := firstNonEmpty(getString(data, "id"), identifier)

	return &NormalizedTicket{
		ID:          id,
		Adapter:     AdapterLinear,
		Kind:        kind,
		Action:      getString(raw, "action"),
		WorkspaceID: firstNonEmpty(getString(raw, "organizationId"), getString(raw, "workspaceId")),
		ProjectID:   firstNonEmpty(getString(project, "id"), getString(project, "name")),
		ChannelID:   channelID,
		ThreadID:    identifier,
		ThreadType:  TicketKindIssue,
		Title:       getString(issue, "title"),
		Body:        body,
		URL:         getString(issue, "url"),
		State:       getString(state, "name"),
		Author:      linearActor(actor),
		Labels:      linearLabels(issue),
		CreatedAt:   parseTime(getString(issue, "createdAt")),
		UpdatedAt:   parseTime(getString(issue, "updatedAt")),
		Metadata:    map[string]any{"raw": raw},
	}, nil
}

func (n *Normalizer) normalizeGitLab(raw map[string]any) (*NormalizedTicket, error) {
	project := getMap(raw, "project")
	attrs := getMap(raw, "object_attributes")
	user := getMap(raw, "user")
	if project == nil {
		return nil, fmt.Errorf("missing gitlab project")
	}
	if attrs == nil {
		return nil, fmt.Errorf("missing gitlab object attributes")
	}

	projectPath := firstNonEmpty(getString(project, "path_with_namespace"), getString(project, "web_url"), getString(project, "id"))
	if projectPath == "" {
		return nil, fmt.Errorf("missing gitlab project path")
	}

	objectKind := firstNonEmpty(getString(raw, "object_kind"), getString(raw, "event_type"))
	kind := TicketKindIssue
	threadID := firstNonEmpty(getString(attrs, "iid"), getString(attrs, "id"))
	threadType := TicketKindIssue
	title := getString(attrs, "title")
	body := getString(attrs, "description")
	url := getString(attrs, "url")

	switch objectKind {
	case "merge_request":
		kind = TicketKindMergeRequest
		threadType = TicketKindMergeRequest
	case "note":
		kind = TicketKindComment
		body = getString(attrs, "note")
		if issue := getMap(raw, "issue"); issue != nil {
			threadID = firstNonEmpty(getString(issue, "iid"), getString(issue, "id"))
			title = getString(issue, "title")
			url = getString(issue, "url")
			threadType = TicketKindIssue
		} else if mr := getMap(raw, "merge_request"); mr != nil {
			threadID = firstNonEmpty(getString(mr, "iid"), getString(mr, "id"))
			title = getString(mr, "title")
			url = getString(mr, "url")
			threadType = TicketKindMergeRequest
		}
	}

	return &NormalizedTicket{
		ID:         firstNonEmpty(getString(attrs, "id"), threadID),
		Adapter:    AdapterGitLab,
		Kind:       kind,
		Action:     firstNonEmpty(getString(attrs, "action"), objectKind),
		ProjectID:  projectPath,
		ChannelID:  projectPath,
		ThreadID:   threadID,
		ThreadType: threadType,
		Title:      title,
		Body:       body,
		URL:        url,
		State:      getString(attrs, "state"),
		Author:     gitLabActor(user),
		Labels:     gitLabLabels(attrs),
		CreatedAt:  parseTime(getString(attrs, "created_at")),
		UpdatedAt:  parseTime(getString(attrs, "updated_at")),
		Metadata:   map[string]any{"raw": raw},
	}, nil
}

// normalizeEmail maps an inbound email event to a ticket. Expected shape
// (the same flat document the messenger email normalizer accepts, so a mail
// relay or IMAP poller can post to either service type):
//
//	{"message_id": "<id>", "in_reply_to": "<root>", "references": "<root> ...",
//	 "from": "Name <addr>", "to": "mailbox@example.com", "cc": "...",
//	 "subject": "...", "body": "...", "date": RFC3339/RFC1123Z,
//	 "labels": [...], "attachments": [{"type","url","mime_type","size_bytes"}]}
//
// The recipient mailbox is the channel; the thread root (In-Reply-To, else the
// first References entry, else the message itself) is the thread. A reply is a
// comment on that thread; a fresh message opens an issue.
func (n *Normalizer) normalizeEmail(raw map[string]any) (*NormalizedTicket, error) {
	fromName, fromAddr := parseEmailAddress(firstNonEmpty(getString(raw, "from"), getString(raw, "sender")))
	if fromAddr == "" {
		return nil, fmt.Errorf("missing email sender (from)")
	}
	_, mailbox := parseEmailAddress(firstNonEmpty(getString(raw, "to"), getString(raw, "mailbox"), getString(raw, "recipient")))
	if mailbox == "" {
		return nil, fmt.Errorf("missing email recipient (to)")
	}

	id := strings.TrimSpace(firstNonEmpty(getString(raw, "message_id"), getString(raw, "message-id"), getString(raw, "id")))
	if id == "" {
		id = fmt.Sprintf("email_%d", time.Now().UnixNano())
	}
	threadID := strings.TrimSpace(firstNonEmpty(getString(raw, "thread_id"), getString(raw, "in_reply_to"), getString(raw, "in-reply-to")))
	if threadID == "" {
		if refs := strings.Fields(getString(raw, "references")); len(refs) > 0 {
			threadID = refs[0]
		}
	}
	kind := TicketKindComment
	if threadID == "" {
		kind = TicketKindIssue
		threadID = id
	}

	metadata := map[string]any{"raw": raw, "mailbox": mailbox}
	for _, key := range []string{"cc", "bcc", "reply_to", "references", "in_reply_to", "headers", "attachments", "html_body"} {
		if value, ok := raw[key]; ok && value != nil {
			metadata[key] = value
		}
	}
	if raw["to"] != nil {
		metadata["to"] = raw["to"]
	}

	received := parseEmailDate(firstNonEmpty(getString(raw, "date"), getString(raw, "received_at")))

	return &NormalizedTicket{
		ID:         id,
		Adapter:    AdapterEmail,
		Kind:       kind,
		Action:     firstNonEmpty(getString(raw, "event"), "received"),
		ChannelID:  mailbox,
		ThreadID:   threadID,
		ThreadType: TicketKindEmail,
		Title:      getString(raw, "subject"),
		Body:       firstNonEmpty(getString(raw, "body"), getString(raw, "text")),
		State:      "received",
		Author: Actor{
			ID:     fromAddr,
			Name:   fromName,
			Handle: fromAddr,
			Email:  fromAddr,
		},
		Labels:    getStringSlice(raw, "labels"),
		CreatedAt: received,
		UpdatedAt: received,
		Metadata:  metadata,
	}, nil
}

// parseEmailDate accepts RFC 3339 and RFC 5322 dates; anything else (or an
// empty value) falls back to the time of normalization.
func parseEmailDate(value string) time.Time {
	if parsed := parseTime(value); !parsed.IsZero() {
		return parsed
	}
	if parsed, err := mail.ParseDate(strings.TrimSpace(value)); err == nil {
		return parsed.UTC()
	}
	return time.Now().UTC()
}

// parseEmailAddress splits "Name <addr>" or a bare address into its display
// name and lowercased address. Malformed input yields the trimmed raw value
// as the address so routing still has a stable key.
func parseEmailAddress(value string) (name, addr string) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", ""
	}
	if parsed, err := mail.ParseAddress(value); err == nil {
		return strings.TrimSpace(parsed.Name), strings.ToLower(strings.TrimSpace(parsed.Address))
	}
	if list, err := mail.ParseAddressList(value); err == nil && len(list) > 0 {
		return strings.TrimSpace(list[0].Name), strings.ToLower(strings.TrimSpace(list[0].Address))
	}
	if first, _, ok := strings.Cut(value, ","); ok {
		value = strings.TrimSpace(first)
	}
	return "", strings.ToLower(strings.TrimPrefix(strings.TrimSpace(value), "mailto:"))
}

func (n *Normalizer) Denormalize(adapter string, result *ActionResult, ticket *NormalizedTicket) (map[string]any, error) {
	if result == nil {
		return nil, fmt.Errorf("action result is nil")
	}

	status := result.Status
	if status == "" {
		status = "success"
	}
	body := result.Output
	if body == "" && result.Error != nil {
		body = result.Error.Error()
	}

	response := map[string]any{
		"status": status,
		"body":   body,
	}
	if ticket != nil {
		response["ticket_id"] = ticket.ID
		response["thread_id"] = ticket.ThreadID
	}
	if result.OutputData != nil {
		response["data"] = result.OutputData
	}

	switch strings.ToLower(strings.TrimSpace(adapter)) {
	case AdapterJira:
		response["target"] = "jira_comment"
	case AdapterLinear:
		response["target"] = "linear_comment"
	case AdapterGitLab:
		response["target"] = "gitlab_note"
	case AdapterEmail:
		response["target"] = "email_reply"
	default:
		response["target"] = "status"
	}
	return response, nil
}

func jiraActor(user map[string]any) Actor {
	return Actor{
		ID:     firstNonEmpty(getString(user, "accountId"), getString(user, "name"), getString(user, "emailAddress")),
		Name:   firstNonEmpty(getString(user, "displayName"), getString(user, "name")),
		Handle: getString(user, "name"),
		Email:  getString(user, "emailAddress"),
	}
}

func linearActor(user map[string]any) Actor {
	return Actor{
		ID:     firstNonEmpty(getString(user, "id"), getString(user, "email")),
		Name:   getString(user, "name"),
		Handle: getString(user, "url"),
		Email:  getString(user, "email"),
	}
}

func gitLabActor(user map[string]any) Actor {
	return Actor{
		ID:     firstNonEmpty(getString(user, "id"), getString(user, "username")),
		Name:   getString(user, "name"),
		Handle: getString(user, "username"),
		Email:  getString(user, "email"),
	}
}

func linearLabels(issue map[string]any) []string {
	values, ok := issue["labels"].([]any)
	if !ok {
		return nil
	}
	labels := make([]string, 0, len(values))
	for _, value := range values {
		if label, ok := value.(map[string]any); ok {
			if name := getString(label, "name"); name != "" {
				labels = append(labels, name)
			}
		}
	}
	return labels
}

func gitLabLabels(attrs map[string]any) []string {
	if labels := getStringSlice(attrs, "labels"); len(labels) > 0 {
		return labels
	}
	if csv := getString(attrs, "labels"); csv != "" {
		parts := strings.Split(csv, ",")
		labels := make([]string, 0, len(parts))
		for _, part := range parts {
			if label := strings.TrimSpace(part); label != "" {
				labels = append(labels, label)
			}
		}
		return labels
	}
	return nil
}

func getMap(m map[string]any, key string) map[string]any {
	v, ok := m[key]
	if !ok || v == nil {
		return nil
	}
	sub, _ := v.(map[string]any)
	return sub
}

func getString(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	v, ok := m[key]
	if !ok || v == nil {
		return ""
	}
	switch s := v.(type) {
	case string:
		return s
	case float64:
		if s == float64(int64(s)) {
			return strconv.FormatInt(int64(s), 10)
		}
		return strconv.FormatFloat(s, 'f', -1, 64)
	case int:
		return strconv.Itoa(s)
	case int64:
		return strconv.FormatInt(s, 10)
	default:
		return fmt.Sprintf("%v", v)
	}
}

func getStringSlice(m map[string]any, key string) []string {
	values, ok := m[key].([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(values))
	for _, value := range values {
		if text := strings.TrimSpace(fmt.Sprintf("%v", value)); text != "" {
			out = append(out, text)
		}
	}
	return out
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func parseTime(value string) time.Time {
	if value == "" {
		return time.Time{}
	}
	formats := []string{time.RFC3339Nano, time.RFC3339, "2006-01-02T15:04:05.000-0700", "2006-01-02T15:04:05-0700"}
	for _, format := range formats {
		if parsed, err := time.Parse(format, value); err == nil {
			return parsed.UTC()
		}
	}
	return time.Time{}
}
