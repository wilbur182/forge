package chat

import (
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	inputStyle           = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 1)
	generatingStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Italic(true)
	generatingLabelStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Bold(true)
)

// Input is the prompt input component backed by a bubbles textarea.
type Input struct {
	textarea   textarea.Model
	focused    bool
	submitting bool // prevents double-submit during streaming
}

// NewInput creates a new Input with default settings.
func NewInput() *Input {
	ta := textarea.New()
	ta.Placeholder = "Type a message..."
	ta.MaxHeight = 5
	ta.CharLimit = 0
	ta.ShowLineNumbers = false
	ta.Blur()

	return &Input{
		textarea:   ta,
		focused:    false,
		submitting: false,
	}
}

// Update handles incoming tea messages.
func (i *Input) Update(msg tea.Msg) (*Input, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.Type {
		case tea.KeyEnter:
			// Submit on Enter (Shift+Enter / Alt+Enter inserts newline via textarea default).
			if i.submitting {
				return i, nil // ignore while streaming
			}
			val := strings.TrimSpace(i.textarea.Value())
			if val == "" {
				return i, nil // don't submit empty
			}
			i.textarea.Reset()
			return i, func() tea.Msg { return SendPromptMsg{Content: val} }

		case tea.KeyCtrlC:
			if i.submitting {
				return i, func() tea.Msg { return AbortMsg{} }
			}
			// Fall through to textarea
		}
	}

	var cmd tea.Cmd
	i.textarea, cmd = i.textarea.Update(msg)
	return i, cmd
}

// View renders the input component constrained to the given width.
func (i *Input) View(width int) string {
	if width <= 0 {
		width = 80
	}

	// Account for border + padding (2 sides * (border 1 + padding 1) = 4)
	innerWidth := width - 4
	if innerWidth < 1 {
		innerWidth = 1
	}
	i.textarea.SetWidth(innerWidth)

	var content string
	if i.submitting {
		// Dim the textarea view and show a generating indicator
		dimmedTA := generatingStyle.Render(i.textarea.View())
		label := generatingLabelStyle.Render("Generating...")
		content = dimmedTA + "\n" + label
	} else {
		content = i.textarea.View()
	}

	return inputStyle.Width(width - 2).Render(content)
}

// Focus focuses the textarea and marks it as focused.
func (i *Input) Focus() {
	i.textarea.Focus()
	i.focused = true
}

// Blur blurs the textarea and marks it as unfocused.
func (i *Input) Blur() {
	i.textarea.Blur()
	i.focused = false
}

// SetSubmitting sets the submitting/streaming state.
func (i *Input) SetSubmitting(v bool) {
	i.submitting = v
}

// IsSubmitting returns whether the input is in the submitting/streaming state.
func (i *Input) IsSubmitting() bool {
	return i.submitting
}

// Value returns the current textarea text.
func (i *Input) Value() string {
	return i.textarea.Value()
}

// Reset clears the textarea content.
func (i *Input) Reset() {
	i.textarea.Reset()
}

// IsFocused returns whether the input is focused.
func (i *Input) IsFocused() bool {
	return i.focused
}
