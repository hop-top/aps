package chat

import (
	"context"
	"errors"
	"io"
	"os"
	"strings"
	"time"

	"hop.top/aps/internal/core"
	corechat "hop.top/aps/internal/core/chat"
	"hop.top/kit/go/ai/llm"
)

const fakeResponseEnv = "APS_CHAT_FAKE_RESPONSE"

func newEngine(profile *core.Profile) (CoreEngine, error) {
	if fake := os.Getenv(fakeResponseEnv); fake != "" {
		return fakeEngine{response: fake}, nil
	}
	return newSingleProfileEngine(context.Background(), profile)
}

func newSingleProfileEngine(ctx context.Context, profile *core.Profile) (CoreEngine, error) {
	resolved, err := corechat.ResolveLLMConfig(profile, corechat.LLMResolveOptions{})
	if err != nil {
		return nil, err
	}
	client, err := corechat.NewLLMClient(ctx, profile.ID, resolved)
	if err != nil {
		var unauth *corechat.UnauthorizedError
		if errors.As(err, &unauth) {
			return nil, unauth
		}
		return nil, err
	}
	return &llmEngine{
		client:  client,
		profile: profile,
		model:   resolved.Model,
	}, nil
}

// newMultiParticipantEngine returns an engine that, for each turn, recomposes
// the system prompt with the next active speaker selected by policy. The CLI
// already drives speaker selection per turn, so the engine takes a function
// that resolves a participant by ID at call time.
func newMultiParticipantEngine(ctx context.Context, owner *core.Profile, participants []corechat.Participant, lookupProfile func(string) (*core.Profile, error)) (CoreEngine, error) {
	if fake := os.Getenv(fakeResponseEnv); fake != "" {
		return fakeEngine{response: fake}, nil
	}
	resolved, err := corechat.ResolveLLMConfig(owner, corechat.LLMResolveOptions{})
	if err != nil {
		return nil, err
	}
	client, err := corechat.NewLLMClient(ctx, owner.ID, resolved)
	if err != nil {
		var unauth *corechat.UnauthorizedError
		if errors.As(err, &unauth) {
			return nil, unauth
		}
		return nil, err
	}
	return &multiParticipantEngine{
		client:        client,
		owner:         owner,
		participants:  participants,
		lookupProfile: lookupProfile,
		model:         resolved.Model,
	}, nil
}

type llmEngine struct {
	client  *llm.Client
	profile *core.Profile
	model   string
}

func (e *llmEngine) Turn(ctx context.Context, req TurnRequest) (TurnResponse, error) {
	llmReq := buildLLMRequest(corechat.RenderSystemPrompt(e.profile), req, modelOrDefault(req.Model, e.model))
	resp, err := e.client.Complete(ctx, llmReq)
	if err != nil {
		return TurnResponse{}, err
	}
	return TurnResponse{Message: Message{
		Role:    roleFromLLM(resp.Role),
		Content: resp.Content,
	}}, nil
}

func (e *llmEngine) StreamTurn(ctx context.Context, req TurnRequest) (<-chan StreamChunk, error) {
	llmReq := buildLLMRequest(corechat.RenderSystemPrompt(e.profile), req, modelOrDefault(req.Model, e.model))
	iter, err := e.client.Stream(ctx, llmReq)
	if err != nil {
		return nil, err
	}
	return drainTokenIterator(ctx, iter), nil
}

type multiParticipantEngine struct {
	client        *llm.Client
	owner         *core.Profile
	participants  []corechat.Participant
	lookupProfile func(string) (*core.Profile, error)
	model         string
}

func (e *multiParticipantEngine) Turn(ctx context.Context, req TurnRequest) (TurnResponse, error) {
	systemPrompt, err := e.systemPromptFor(req.ProfileID)
	if err != nil {
		return TurnResponse{}, err
	}
	llmReq := buildLLMRequest(systemPrompt, req, modelOrDefault(req.Model, e.model))
	resp, err := e.client.Complete(ctx, llmReq)
	if err != nil {
		return TurnResponse{}, err
	}
	return TurnResponse{Message: Message{
		Role:    roleFromLLM(resp.Role),
		Content: resp.Content,
	}}, nil
}

func (e *multiParticipantEngine) StreamTurn(ctx context.Context, req TurnRequest) (<-chan StreamChunk, error) {
	systemPrompt, err := e.systemPromptFor(req.ProfileID)
	if err != nil {
		return nil, err
	}
	llmReq := buildLLMRequest(systemPrompt, req, modelOrDefault(req.Model, e.model))
	iter, err := e.client.Stream(ctx, llmReq)
	if err != nil {
		return nil, err
	}
	return drainTokenIterator(ctx, iter), nil
}

func (e *multiParticipantEngine) systemPromptFor(activeID string) (string, error) {
	if len(e.participants) == 0 {
		return corechat.RenderSystemPrompt(e.owner), nil
	}
	return corechat.ComposeSystemPrompt(e.participants, activeID)
}

// buildLLMRequest assembles the llm.Request from a system prompt and the
// caller's History. Callers (runOnce, turnCmd, streamStartCmd) always append
// the user's prompt to History before invoking the engine, so the History is
// the single source of truth for conversation state — req.Prompt is not
// re-appended to avoid duplicating the latest turn.
func buildLLMRequest(systemPrompt string, req TurnRequest, model string) llm.Request {
	messages := make([]llm.Message, 0, len(req.History)+1)
	if strings.TrimSpace(systemPrompt) != "" {
		messages = append(messages, llm.Message{Role: string(corechat.RoleSystem), Content: systemPrompt})
	}
	for _, h := range req.History {
		if strings.TrimSpace(h.Content) == "" {
			continue
		}
		messages = append(messages, llm.Message{Role: h.Role, Content: h.Content})
	}
	return llm.Request{Messages: messages, Model: model}
}

func drainTokenIterator(ctx context.Context, iter llm.TokenIterator) <-chan StreamChunk {
	ch := make(chan StreamChunk)
	go func() {
		defer close(ch)
		defer iter.Close()
		for {
			if err := ctx.Err(); err != nil {
				select {
				case ch <- StreamChunk{Err: err}:
				default:
				}
				return
			}
			tok, err := iter.Next()
			if errors.Is(err, io.EOF) {
				return
			}
			if err != nil {
				select {
				case <-ctx.Done():
				case ch <- StreamChunk{Err: err}:
				}
				return
			}
			if tok.Content != "" {
				select {
				case <-ctx.Done():
					return
				case ch <- StreamChunk{Delta: tok.Content}:
				}
			}
			if tok.Done {
				select {
				case <-ctx.Done():
				case ch <- StreamChunk{Done: true}:
				}
				return
			}
		}
	}()
	return ch
}

func modelOrDefault(override, fallback string) string {
	if strings.TrimSpace(override) != "" {
		return override
	}
	return fallback
}

func roleFromLLM(role string) string {
	if role == "" {
		return roleAssistant
	}
	return role
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
