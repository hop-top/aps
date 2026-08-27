package chat

import "context"

// CoreEngine is the CLI adapter boundary expected from internal/core/chat.
// Worker 1 should be able to replace newEngine with a core-backed
// implementation without changing Cobra or TUI code.
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
	Once         string
	Model        string
	NoStream     bool
	Attach       string
	Invite       []string
	MaxAutoTurns int
	// Temperature and MaxTokens are pointers so a flag left unset means
	// "no override" while an explicit --temperature 0 still overrides
	// the merged config value to zero.
	Temperature *float64
	MaxTokens   *int
	Effort      string
	Verbosity   string
}

const (
	roleUser      = "user"
	roleAssistant = "assistant"
	sessionType   = "chat"
)
