package chat

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/wilbur182/forge/internal/opencode/client"
	"github.com/wilbur182/forge/internal/plugin"
)

const (
	pluginID   = "chat"
	pluginName = "Chat"
	pluginIcon = "⚡"
)

// ChatMessage represents a single message in the chat history.
type ChatMessage struct {
	Role        string // "user" or "assistant"
	Content     string
	Timestamp   time.Time
	IsStreaming bool
}

// Plugin implements the chat TUI plugin.
type Plugin struct {
	ctx          *plugin.Context
	focused      bool
	width        int
	height       int
	sessionID    string
	messages     []ChatMessage
	streaming    bool
	streamBuffer string
	err          error
	epoch        uint64
	inputFocused bool
	apiClient    *client.Client
	sseCancel    context.CancelFunc
	sseChan      <-chan client.SSEEvent
	input        *Input
	msgViewport  MessageViewport
}

// Compile-time interface assertion.
var _ plugin.Plugin = (*Plugin)(nil)
var _ plugin.TextInputConsumer = (*Plugin)(nil)

// New creates a new Chat plugin.
func New() *Plugin {
	return &Plugin{}
}

// ID returns the plugin identifier.
func (p *Plugin) ID() string { return pluginID }

// Name returns the plugin display name.
func (p *Plugin) Name() string { return pluginName }

// Icon returns the plugin icon character.
func (p *Plugin) Icon() string { return pluginIcon }

// Init initializes the plugin with context.
func (p *Plugin) Init(ctx *plugin.Context) error {
	p.ctx = ctx
	p.messages = nil
	p.sessionID = ""
	p.streaming = false
	p.streamBuffer = ""
	p.err = nil
	p.epoch = ctx.Epoch
	p.inputFocused = true
	p.apiClient = client.New("http://localhost:3725", ctx.ProjectRoot)
	p.input = NewInput()
	p.input.Focus()
	p.msgViewport = NewMessageViewport(80, 20)
	return nil
}

// Start begins plugin operation.
func (p *Plugin) Start() tea.Cmd {
	epoch := p.epoch
	apiClient := p.apiClient
	return func() tea.Msg {
		ctx := context.Background()
		if err := apiClient.Ping(ctx); err != nil {
			return ConnectionErrorMsg{
				Err:   fmt.Errorf("cannot connect to OpenCode. Is it running? Try: opencode web --port 3725\n%w", err),
				Epoch: epoch,
			}
		}

		resp, err := apiClient.CreateSession(ctx)
		if err != nil {
			return ConnectionErrorMsg{Err: err, Epoch: epoch}
		}

		return SessionCreatedMsg{SessionID: resp.ID, Epoch: epoch}
	}
}

// Stop cleans up plugin resources.
func (p *Plugin) Stop() {
	if p.sseCancel != nil {
		p.sseCancel()
		p.sseCancel = nil
	}
	p.messages = nil
	p.sessionID = ""
	p.streaming = false
	p.streamBuffer = ""
	p.err = nil
	p.sseChan = nil
}

// Update handles tea messages.
func (p *Plugin) Update(msg tea.Msg) (plugin.Plugin, tea.Cmd) {
	switch m := msg.(type) {
	case tea.WindowSizeMsg:
		p.width = m.Width
		p.height = m.Height
		inputHeight := 3
		statusHeight := 1
		vpHeight := m.Height - inputHeight - statusHeight
		if vpHeight < 1 {
			vpHeight = 1
		}
		p.msgViewport.SetSize(m.Width, vpHeight)
		return p, nil

	case plugin.PluginFocusedMsg:
		if p.input != nil {
			p.input.Focus()
			p.inputFocused = true
		}
		return p, nil

	case tea.KeyMsg:
		if p.inputFocused && p.input != nil {
			newInput, cmd := p.input.Update(msg)
			p.input = newInput
			return p, cmd
		}

		newVP, cmd := p.msgViewport.Update(msg)
		p.msgViewport = newVP
		return p, cmd

	case StreamTokenMsg:
		if plugin.IsStale(p.ctx, m) {
			return p, nil
		}
		p.streamBuffer += m.Token
		p.msgViewport.SetMessages(append(p.messages, ChatMessage{
			Role:        "assistant",
			Content:     p.streamBuffer,
			IsStreaming: true,
		}))
		return p, listenForSSE(p.sseChan, p.epoch)

	case StreamCompleteMsg:
		if plugin.IsStale(p.ctx, m) {
			return p, nil
		}
		if p.streamBuffer != "" {
			p.messages = append(p.messages, ChatMessage{
				Role:      "assistant",
				Content:   p.streamBuffer,
				Timestamp: time.Now(),
			})
			p.streamBuffer = ""
		}
		p.streaming = false
		if p.input != nil {
			p.input.SetSubmitting(false)
		}
		p.msgViewport.SetMessages(p.messages)
		return p, nil

	case StreamErrorMsg:
		if plugin.IsStale(p.ctx, m) {
			return p, nil
		}
		p.err = m.Err
		p.streaming = false
		if p.input != nil {
			p.input.SetSubmitting(false)
		}
		if p.streamBuffer != "" {
			p.messages = append(p.messages, ChatMessage{
				Role:      "assistant",
				Content:   p.streamBuffer + "\n[Error: stream interrupted]",
				Timestamp: time.Now(),
			})
			p.streamBuffer = ""
		}
		p.msgViewport.SetMessages(p.messages)
		return p, nil

	case SessionCreatedMsg:
		if plugin.IsStale(p.ctx, m) {
			return p, nil
		}
		p.sessionID = m.SessionID
		ctx, cancel := context.WithCancel(context.Background())
		p.sseCancel = cancel
		ch, err := p.apiClient.StreamEvents(ctx)
		if err != nil {
			p.err = err
			return p, nil
		}
		p.sseChan = ch
		return p, listenForSSE(ch, p.epoch)

	case AbortCompleteMsg:
		if plugin.IsStale(p.ctx, m) {
			return p, nil
		}
		p.streaming = false
		if p.input != nil {
			p.input.SetSubmitting(false)
		}
		if p.streamBuffer != "" {
			p.messages = append(p.messages, ChatMessage{
				Role:      "assistant",
				Content:   p.streamBuffer + "\n[Aborted]",
				Timestamp: time.Now(),
			})
			p.streamBuffer = ""
		}
		p.msgViewport.SetMessages(p.messages)
		return p, nil

	case ConnectionErrorMsg:
		if plugin.IsStale(p.ctx, m) {
			return p, nil
		}
		p.err = m.Err
		// Blur input on connection error so number keys can switch tabs.
		if p.sessionID == "" && p.input != nil {
			p.input.Blur()
		}
		return p, nil

	case SendPromptMsg:
		p.messages = append(p.messages, ChatMessage{
			Role:      "user",
			Content:   m.Content,
			Timestamp: time.Now(),
		})
		p.streaming = true
		p.streamBuffer = ""
		if p.input != nil {
			p.input.SetSubmitting(true)
		}
		p.msgViewport.SetMessages(p.messages)

		sessionID := p.sessionID
		content := m.Content
		epoch := p.epoch
		apiClient := p.apiClient
		return p, func() tea.Msg {
			if err := apiClient.SendMessage(context.Background(), sessionID, content); err != nil {
				return StreamErrorMsg{Err: err, Epoch: epoch}
			}
			return nil
		}

	case AbortMsg:
		if p.sessionID == "" || !p.streaming {
			return p, nil
		}
		sessionID := p.sessionID
		epoch := p.epoch
		apiClient := p.apiClient
		return p, func() tea.Msg {
			if err := apiClient.AbortGeneration(context.Background(), sessionID); err != nil {
				return StreamErrorMsg{Err: err, Epoch: epoch}
			}
			return AbortCompleteMsg{Epoch: epoch}
		}
	}

	return p, nil
}

// IsFocused returns whether the plugin is focused.
func (p *Plugin) IsFocused() bool { return p.focused }

// SetFocused sets the focus state.
func (p *Plugin) SetFocused(f bool) { p.focused = f }

// Commands returns the available plugin commands.
func (p *Plugin) Commands() []plugin.Command {
	return []plugin.Command{
		{ID: "send", Name: "Send", Description: "Send message", Context: "chat", Priority: 1},
		{ID: "abort", Name: "Abort", Description: "Abort generation", Context: "chat", Priority: 2},
		{ID: "new-session", Name: "New", Description: "New chat session", Context: "chat", Priority: 3},
	}
}

// FocusContext returns the current focus context string.
func (p *Plugin) FocusContext() string {
	return "chat"
}

// ConsumesTextInput reports whether the plugin is in a text-input context.
func (p *Plugin) ConsumesTextInput() bool {
	return p.input != nil && p.input.IsFocused()
}

func listenForSSE(ch <-chan client.SSEEvent, epoch uint64) tea.Cmd {
	if ch == nil {
		return nil
	}

	return func() tea.Msg {
		for {
			event, ok := <-ch
			if !ok {
				return StreamCompleteMsg{Epoch: epoch}
			}

			switch event.Event {
			case "message.part.delta":
				var delta client.MessagePartDelta
				if err := json.Unmarshal([]byte(event.Data), &delta); err == nil {
					return StreamTokenMsg{Token: delta.Content, Epoch: epoch}
				}
			case "message.complete":
				return StreamCompleteMsg{Epoch: epoch}
			}
		}
	}
}
