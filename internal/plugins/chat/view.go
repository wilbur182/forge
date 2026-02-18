package chat

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/wilbur182/forge/internal/markdown"
	"github.com/wilbur182/forge/internal/styles"
)

func (p *Plugin) View(width, height int) string {
	p.width = width
	p.height = height

	if p.err != nil && p.sessionID == "" {
		content := lipgloss.NewStyle().
			Width(width).
			Align(lipgloss.Center).
			Foreground(lipgloss.Color("196")).
			Render(fmt.Sprintf("\n\n⚠ %s", p.err.Error()))
		return lipgloss.NewStyle().Width(width).Height(height).MaxHeight(height).Render(content)
	}

	inputView := ""
	inputHeight := 0
	if p.input != nil {
		inputView = p.input.View(width)
		inputHeight = lipgloss.Height(inputView)
	}

	statusView := renderStatusBar(p.sessionID, p.streaming, width)
	statusHeight := 1

	vpHeight := height - inputHeight - statusHeight
	if vpHeight < 1 {
		vpHeight = 1
	}
	p.msgViewport.SetSize(width, vpHeight)

	msgs := p.messages
	if p.streamBuffer != "" {
		msgs = append(msgs, ChatMessage{
			Role:        "assistant",
			Content:     p.streamBuffer,
			IsStreaming: true,
		})
	}
	p.msgViewport.SetMessages(msgs)

	content := lipgloss.JoinVertical(lipgloss.Left,
		statusView,
		p.msgViewport.View(),
		inputView,
	)

	return lipgloss.NewStyle().Width(width).Height(height).MaxHeight(height).Render(content)
}

// MessageViewport handles the rendering and scrolling of chat messages.
type MessageViewport struct {
	viewport viewport.Model
	width    int
	height   int
	atBottom bool
}

// NewMessageViewport creates a new message viewport with the given dimensions.
func NewMessageViewport(width, height int) MessageViewport {
	vp := viewport.New(width, height)
	vp.YPosition = 0
	return MessageViewport{
		viewport: vp,
		width:    width,
		height:   height,
		atBottom: true,
	}
}

// SetMessages updates the viewport content with the given messages.
func (v *MessageViewport) SetMessages(msgs []ChatMessage) {
	content := renderMessages(msgs, v.width)
	v.viewport.SetContent(content)
	if v.atBottom {
		v.viewport.GotoBottom()
	}
}

// SetSize updates the viewport dimensions.
func (v *MessageViewport) SetSize(width, height int) {
	v.width = width
	v.height = height
	v.viewport.Width = width
	v.viewport.Height = height
}

// Update handles tea messages for the viewport.
func (v *MessageViewport) Update(msg tea.Msg) (MessageViewport, tea.Cmd) {
	var cmd tea.Cmd
	v.viewport, cmd = v.viewport.Update(msg)
	v.atBottom = v.viewport.AtBottom()
	return *v, cmd
}

// View returns the viewport string.
func (v *MessageViewport) View() string {
	return v.viewport.View()
}

// renderMessages renders the conversation as formatted text.
func renderMessages(messages []ChatMessage, width int) string {
	if len(messages) == 0 {
		return lipgloss.NewStyle().
			Width(width).
			Align(lipgloss.Center).
			Foreground(styles.TextMuted).
			Render("\n\nStart a conversation — type a message below")
	}

	var sb strings.Builder
	mdRenderer, _ := markdown.NewRenderer()

	for i, msg := range messages {
		if i > 0 {
			sb.WriteString("\n\n")
		}

		if msg.Role == "user" {
			prefix := lipgloss.NewStyle().
				Foreground(styles.Secondary).
				Bold(true).
				Render("> ")

			content := lipgloss.NewStyle().
				Foreground(styles.TextPrimary).
				Render(msg.Content)

			sb.WriteString(prefix + content)
		} else {
			if mdRenderer != nil {
				renderedLines := mdRenderer.RenderContent(msg.Content, width)
				sb.WriteString(strings.Join(renderedLines, "\n"))
			} else {
				sb.WriteString(msg.Content)
			}

			if msg.IsStreaming {
				sb.WriteString(lipgloss.NewStyle().
					Foreground(styles.Accent).
					Blink(true).
					Render(" ▍"))
			}
		}
	}

	return sb.String()
}

// renderStatusBar renders the chat status bar.
func renderStatusBar(sessionID string, streaming bool, width int) string {
	statusStyle := lipgloss.NewStyle().
		Width(width).
		Background(styles.BgSecondary).
		Foreground(styles.TextMuted)

	left := ""
	if sessionID != "" {
		dispID := sessionID
		if len(dispID) > 8 {
			dispID = dispID[:8]
		}
		left = fmt.Sprintf(" ID: %s", dispID)
	}

	right := "Ready"
	if streaming {
		right = "⟳ Generating..."
	}

	availSpace := width - len(left) - len(right) - 2
	if availSpace < 0 {
		availSpace = 0
	}
	space := strings.Repeat(" ", availSpace)

	return statusStyle.Render(fmt.Sprintf("%s%s%s", left, space, right))
}
