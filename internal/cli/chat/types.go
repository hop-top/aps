package chat

import "context"

// CoreEngine is the CLI adapter boundary expected from internal/core/chat.
// newEngine is a placeholder; swapping it for a core-backed implementation
// must not require changes to the Cobra command or TUI layer.
type CoreEngine interface {
	Turn(context.Context, TurnRequest) (TurnResponse, error)
	StreamTurn(context.Context, TurnRequest) (<-chan StreamChunk, error)
}

type TurnRequest struct {
	SessionID string
	ProfileID string
	Prompt    string
	Model     string
	NoStream  bool
	History   []Message
}

type TurnResponse struct {
	Message Message
}

type StreamChunk struct {
	Delta string
	Done  bool
	Err   error
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type Options struct {
	Once     string
	Model    string
	NoStream bool
	Attach   string
}

const (
	roleUser      = "user"
	roleAssistant = "assistant"
	sessionType   = "chat"
)
