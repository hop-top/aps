package chat

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"hop.top/aps/internal/cli/exit"
	"hop.top/aps/internal/core"
	corechat "hop.top/aps/internal/core/chat"
)

const fakeResponseEnv = "APS_CHAT_FAKE_RESPONSE"

func newEngine(ctx context.Context, profile *core.Profile, opts Options, systemPrompt string) (CoreEngine, error) {
	if fake := os.Getenv(fakeResponseEnv); fake != "" {
		return fakeEngine{response: fake}, nil
	}

	service, err := corechat.NewService(ctx, profile.ID, corechat.ServiceOptions{
		LLM: corechat.LLMResolveOptions{
			ModelOverride: opts.Model,
		},
		SystemPrompt: systemPrompt,
	})
	if err != nil {
		if errors.Is(err, corechat.ErrUnauthorized) {
			return nil, fmt.Errorf("%w: %w", exit.ErrUnauthorized, err)
		}
		return nil, err
	}
	return coreEngine{service: service}, nil
}

type coreEngine struct {
	service *corechat.Service
}

func (e coreEngine) Turn(ctx context.Context, req TurnRequest) (TurnResponse, error) {
	reply, err := e.service.Send(ctx, req.SessionID, req.Prompt)
	if err != nil {
		return TurnResponse{}, err
	}
	return TurnResponse{Message: Message{Role: string(reply.Role), Content: reply.Content}}, nil
}

func (e coreEngine) StreamTurn(ctx context.Context, req TurnRequest) (<-chan StreamChunk, error) {
	resp, err := e.Turn(ctx, req)
	if err != nil {
		return nil, err
	}
	ch := make(chan StreamChunk)
	go func() {
		defer close(ch)
		for _, r := range resp.Message.Content {
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
