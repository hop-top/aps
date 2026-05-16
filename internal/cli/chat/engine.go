package chat

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"hop.top/aps/internal/cli/exit"
	"hop.top/aps/internal/core"
)

const fakeResponseEnv = "APS_CHAT_FAKE_RESPONSE"

var providerKeys = []string{
	"OPENAI_API_KEY",
	"ANTHROPIC_API_KEY",
	"GOOGLE_API_KEY",
	"GEMINI_API_KEY",
	"AZURE_OPENAI_API_KEY",
}

func newEngine(profile *core.Profile) (CoreEngine, error) {
	if fake := os.Getenv(fakeResponseEnv); fake != "" {
		return fakeEngine{response: fake}, nil
	}

	secrets, err := core.LoadProfileSecrets(profile.ID)
	if err != nil {
		return nil, err
	}
	var missing []string
	for _, key := range providerKeys {
		if strings.TrimSpace(secrets[key]) == "" && strings.TrimSpace(os.Getenv(key)) == "" {
			missing = append(missing, key)
		}
	}
	if len(missing) == len(providerKeys) {
		return nil, fmt.Errorf("missing LLM provider key (%s): %w", strings.Join(providerKeys, ", "), exit.ErrUnauthorized)
	}

	return nil, fmt.Errorf("internal/core/chat integration unavailable; expected engine must implement Turn and StreamTurn")
}

type fakeEngine struct {
	response string
}

func (e fakeEngine) Turn(_ context.Context, req TurnRequest) (TurnResponse, error) {
	return TurnResponse{Message: Message{
		Role:    roleAssistant,
		Content: renderFakeResponse(e.response, req),
	}}, nil
}

func (e fakeEngine) StreamTurn(ctx context.Context, req TurnRequest) (<-chan StreamChunk, error) {
	text := renderFakeResponse(e.response, req)
	ch := make(chan StreamChunk)
	go func() {
		defer close(ch)
		for _, r := range text {
			select {
			case <-ctx.Done():
				ch <- StreamChunk{Err: ctx.Err()}
				return
			case ch <- StreamChunk{Delta: string(r)}:
				time.Sleep(time.Millisecond)
			}
		}
		ch <- StreamChunk{Done: true}
	}()
	return ch, nil
}

func renderFakeResponse(template string, req TurnRequest) string {
	replacer := strings.NewReplacer(
		"{{prompt}}", req.Prompt,
		"{{profile}}", req.ProfileID,
		"{{model}}", req.Model,
		"{{session}}", req.SessionID,
	)
	return replacer.Replace(template)
}
