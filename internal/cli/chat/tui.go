package chat

import (
	"context"
	"fmt"
	"os"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/colorprofile"
	"hop.top/aps/internal/core"
)

type model struct {
	ctx       context.Context
	engine    CoreEngine
	session   *chatSession
	profile   *core.Profile
	opts      Options
	input     textinput.Model
	width     int
	height    int
	err       error
	streaming bool
	streamBuf strings.Builder
}

type responseMsg struct {
	content string
	err     error
}

type streamMsg struct {
	ch    <-chan StreamChunk
	delta string
	done  bool
	err   error
}

func runTUI(ctx context.Context, engine CoreEngine, sess *chatSession, profile *core.Profile, opts Options) error {
	if os.Getenv("APS_CHAT_TUI_TEST") == "1" {
		fmt.Fprint(os.Stdout, renderTranscript(profile.ID, sess, opts, 17))
	}
	p := tea.NewProgram(
		newModel(ctx, engine, sess, profile, opts),
		tea.WithWindowSize(80, 24),
		tea.WithEnvironment(os.Environ()),
		tea.WithColorProfile(colorprofile.ANSI256),
	)
	_, err := p.Run()
	return err
}

func newModel(ctx context.Context, engine CoreEngine, sess *chatSession, profile *core.Profile, opts Options) model {
	input := textinput.New()
	input.Prompt = "> "
	input.Placeholder = "Message " + profile.ID
	input.SetWidth(72)
	_ = input.Focus()
	return model{
		ctx:     ctx,
		engine:  engine,
		session: sess,
		profile: profile,
		opts:    opts,
		input:   input,
		width:   80,
		height:  24,
	}
}

func (m model) Init() tea.Cmd {
	return textinput.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.input.SetWidth(max(10, msg.Width-4))
	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			return m, tea.Quit
		case "enter":
			prompt := strings.TrimSpace(m.input.Value())
			if prompt == "" || m.streaming {
				return m, nil
			}
			m.input.SetValue("")
			if err := m.session.append(roleUser, prompt); err != nil {
				m.err = err
				return m, nil
			}
			m.session.messages = append(m.session.messages, Message{Role: roleAssistant})
			m.streaming = true
			m.streamBuf.Reset()
			if m.opts.NoStream {
				return m, turnCmd(m.ctx, m.engine, m.session, m.opts, prompt)
			}
			return m, streamStartCmd(m.ctx, m.engine, m.session, m.opts, prompt)
		}
	case responseMsg:
		m.streaming = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		if err := m.session.replaceLastAssistant(msg.content); err != nil {
			m.err = err
		}
	case streamMsg:
		if msg.err != nil {
			m.streaming = false
			m.err = msg.err
			return m, nil
		}
		if msg.delta != "" {
			m.streamBuf.WriteString(msg.delta)
			if err := m.session.replaceLastAssistant(m.streamBuf.String()); err != nil {
				m.streaming = false
				m.err = err
				return m, nil
			}
		}
		if msg.done {
			m.streaming = false
			return m, nil
		}
		return m, streamNextCmd(msg.ch)
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m model) View() tea.View {
	var b strings.Builder
	b.WriteString(renderTranscript(m.profile.ID, m.session, m.opts, max(4, m.height-7)))
	b.WriteString("\n")
	if m.err != nil {
		b.WriteString(errorStyle.Render("error: " + m.err.Error()))
		b.WriteString("\n")
	}
	if m.streaming {
		b.WriteString(statusStyle.Render("streaming..."))
		b.WriteString("\n")
	}
	b.WriteString(m.input.View())
	b.WriteString("\n")
	b.WriteString(statusStyle.Render("esc/ctrl+c quit"))
	return tea.NewView(b.String())
}

func renderTranscript(profileID string, sess *chatSession, opts Options, maxMessages int) string {
	var b strings.Builder
	title := titleStyle.Render("aps chat " + profileID)
	status := statusStyle.Render(fmt.Sprintf("session %s", sess.id))
	if opts.Model != "" {
		status = statusStyle.Render(fmt.Sprintf("session %s  model %s", sess.id, opts.Model))
	}
	b.WriteString(title)
	b.WriteString("\n")
	b.WriteString(status)
	b.WriteString("\n\n")
	for _, msg := range visibleMessages(sess.messages, maxMessages) {
		switch msg.Role {
		case roleUser:
			b.WriteString(userStyle.Render("you: "))
		case roleAssistant:
			b.WriteString(assistantStyle.Render(profileID + ": "))
		default:
			b.WriteString(statusStyle.Render(msg.Role + ": "))
		}
		b.WriteString(msg.Content)
		b.WriteString("\n\n")
	}
	return b.String()
}

func visibleMessages(messages []Message, maxLines int) []Message {
	if maxLines <= 0 || len(messages) <= maxLines {
		return messages
	}
	return messages[len(messages)-maxLines:]
}

func turnCmd(ctx context.Context, engine CoreEngine, sess *chatSession, opts Options, prompt string) tea.Cmd {
	return func() tea.Msg {
		resp, err := engine.Turn(ctx, TurnRequest{
			SessionID: sess.id,
			ProfileID: sess.profileID,
			Prompt:    prompt,
			Model:     opts.Model,
			NoStream:  opts.NoStream,
			History:   sess.messages,
		})
		if err != nil {
			return responseMsg{err: err}
		}
		return responseMsg{content: resp.Message.Content}
	}
}

func streamStartCmd(ctx context.Context, engine CoreEngine, sess *chatSession, opts Options, prompt string) tea.Cmd {
	return func() tea.Msg {
		ch, err := engine.StreamTurn(ctx, TurnRequest{
			SessionID: sess.id,
			ProfileID: sess.profileID,
			Prompt:    prompt,
			Model:     opts.Model,
			History:   sess.messages,
		})
		if err != nil {
			return streamMsg{err: err}
		}
		return readStreamChunk(ch)
	}
}

func streamNextCmd(ch <-chan StreamChunk) tea.Cmd {
	return func() tea.Msg {
		return readStreamChunk(ch)
	}
}

func readStreamChunk(ch <-chan StreamChunk) tea.Msg {
	chunk, ok := <-ch
	if !ok {
		return streamMsg{done: true}
	}
	return streamMsg{ch: ch, delta: chunk.Delta, done: chunk.Done, err: chunk.Err}
}

var (
	titleStyle     = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	statusStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	userStyle      = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("75"))
	assistantStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("114"))
	errorStyle     = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("196"))
)
