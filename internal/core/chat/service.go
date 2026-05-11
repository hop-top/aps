package chat

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"
	"hop.top/aps/internal/core"
	"hop.top/aps/internal/core/session"
	"hop.top/kit/go/ai/llm"
)

type Service struct {
	profile *core.Profile
	reg     *session.SessionRegistry
	store   *Store
	client  Completer
	model   string
	system  string
	now     func() time.Time
	newID   func() string
}

type ServiceOptions struct {
	Registry     *session.SessionRegistry
	Store        *Store
	Client       Completer
	Now          func() time.Time
	NewID        func() string
	LLM          LLMResolveOptions
	Model        string
	SystemPrompt string
}

func NewService(ctx context.Context, profileID string, opts ServiceOptions) (*Service, error) {
	profile, err := core.LoadProfile(profileID)
	if err != nil {
		return nil, err
	}
	resolved, err := ResolveLLMConfig(profile, opts.LLM)
	if err != nil {
		return nil, err
	}
	client := opts.Client
	if client == nil {
		client, err = NewLLMClient(ctx, profile.ID, resolved)
		if err != nil {
			return nil, err
		}
	}
	opts.Model = resolved.Model
	return NewServiceWithProfile(profile, client, opts)
}

func NewServiceWithProfile(profile *core.Profile, client Completer, opts ServiceOptions) (*Service, error) {
	if profile == nil {
		return nil, fmt.Errorf("chat profile cannot be nil")
	}
	if client == nil {
		return nil, fmt.Errorf("chat llm client cannot be nil")
	}
	reg := opts.Registry
	if reg == nil {
		reg = session.GetRegistry()
	}
	store := opts.Store
	if store == nil {
		dataDir, err := core.GetDataDir()
		if err != nil {
			return nil, err
		}
		store = NewStore(dataDir)
	}
	now := opts.Now
	if now == nil {
		now = time.Now
	}
	newID := opts.NewID
	if newID == nil {
		newID = func() string { return uuid.NewString() }
	}
	return &Service{
		profile: profile,
		reg:     reg,
		store:   store,
		client:  client,
		model:   opts.Model,
		system:  opts.SystemPrompt,
		now:     now,
		newID:   newID,
	}, nil
}

func (s *Service) Open(ctx context.Context) (*Transcript, error) {
	id := s.newID()
	info := &session.SessionInfo{
		ID:        id,
		ProfileID: s.profile.ID,
		Command:   "aps chat",
		Status:    session.SessionActive,
		Type:      session.SessionTypeChat,
		Environment: map[string]string{
			"chat_transcript": "sessions/chat/" + id + ".json",
		},
	}
	if err := s.reg.RegisterWithContext(ctx, info); err != nil {
		return nil, fmt.Errorf("register chat session: %w", err)
	}

	transcript := &Transcript{
		SessionID: id,
		ProfileID: s.profile.ID,
		UpdatedAt: s.now(),
	}
	if err := s.store.Save(transcript); err != nil {
		_ = s.reg.UnregisterWithContext(ctx, id)
		return nil, err
	}
	return transcript, nil
}

func (s *Service) Send(ctx context.Context, sessionID, content string) (Turn, error) {
	if content == "" {
		return Turn{}, fmt.Errorf("chat content cannot be empty")
	}
	transcript, err := s.store.Load(sessionID)
	if errors.Is(err, os.ErrNotExist) {
		transcript = &Transcript{
			SessionID: sessionID,
			ProfileID: s.profile.ID,
		}
		if err := s.ensureRegistered(ctx, sessionID); err != nil {
			return Turn{}, err
		}
	} else if err != nil {
		return Turn{}, err
	}
	if transcript.ProfileID == "" {
		transcript.ProfileID = s.profile.ID
	}
	if transcript.ProfileID != s.profile.ID {
		return Turn{}, fmt.Errorf("chat session %s belongs to profile %s, not %s", sessionID, transcript.ProfileID, s.profile.ID)
	}

	now := s.now()
	transcript.Turns = append(transcript.Turns, Turn{Role: RoleUser, Content: content, CreatedAt: now})
	req := s.request(transcript.Turns)
	resp, err := s.client.Complete(ctx, req)
	if err != nil {
		return Turn{}, err
	}
	role := RoleAssistant
	if resp.Role != "" {
		role = Role(resp.Role)
	}
	reply := Turn{Role: role, Content: resp.Content, CreatedAt: s.now()}
	transcript.Turns = append(transcript.Turns, reply)
	transcript.UpdatedAt = reply.CreatedAt
	if err := s.store.Save(transcript); err != nil {
		return Turn{}, err
	}
	_ = s.reg.UpdateHeartbeat(sessionID)
	return reply, nil
}

func (s *Service) History(sessionID string) (*Transcript, error) {
	return s.store.Load(sessionID)
}

func (s *Service) request(turns []Turn) llm.Request {
	messages := make([]llm.Message, 0, len(turns)+1)
	system := s.system
	if system == "" {
		system = RenderSystemPrompt(s.profile)
	}
	if system != "" {
		messages = append(messages, llm.Message{
			Role:    string(RoleSystem),
			Content: system,
		})
	}
	for _, turn := range turns {
		messages = append(messages, llm.Message{
			Role:    string(turn.Role),
			Content: turn.Content,
		})
	}
	return llm.Request{Messages: messages, Model: s.model}
}

func (s *Service) ensureRegistered(ctx context.Context, sessionID string) error {
	if _, err := s.reg.Get(sessionID); err == nil {
		return nil
	}
	profileDir, err := core.GetProfileDir(s.profile.ID)
	if err != nil {
		return err
	}
	info := &session.SessionInfo{
		ID:         sessionID,
		ProfileID:  s.profile.ID,
		ProfileDir: profileDir,
		Command:    "aps chat",
		PID:        os.Getpid(),
		Status:     session.SessionActive,
		Type:       session.SessionTypeChat,
		Environment: map[string]string{
			"chat_transcript": "sessions/chat/" + sessionID + ".json",
		},
	}
	if s.profile.Workspace != nil {
		info.WorkspaceID = s.profile.Workspace.Name
	}
	if err := s.reg.RegisterWithContext(ctx, info); err != nil {
		return fmt.Errorf("register chat session: %w", err)
	}
	return nil
}
