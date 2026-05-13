package chat

import (
	"context"
	"time"

	"hop.top/kit/go/ai/llm"
)

type Role string

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
)

type Turn struct {
	Role      Role      `json:"role"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

type Transcript struct {
	SessionID string    `json:"session_id"`
	ProfileID string    `json:"profile_id"`
	Turns     []Turn    `json:"turns"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Completer interface {
	Complete(ctx context.Context, req llm.Request) (llm.Response, error)
}
