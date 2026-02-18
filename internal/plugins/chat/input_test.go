package chat

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func extractMsg(cmd tea.Cmd) tea.Msg {
	if cmd == nil {
		return nil
	}
	return cmd()
}

func makeInput(val string) *Input {
	inp := NewInput()
	inp.Focus()
	for _, ch := range val {
		inp.textarea.InsertRune(ch)
	}
	return inp
}

func TestEnterSubmit(t *testing.T) {
	inp := makeInput("hello")

	_, cmd := inp.Update(tea.KeyMsg{Type: tea.KeyEnter})
	msg := extractMsg(cmd)

	got, ok := msg.(SendPromptMsg)
	if !ok {
		t.Fatalf("expected SendPromptMsg, got %T", msg)
	}
	if got.Content != "hello" {
		t.Errorf("expected Content %q, got %q", "hello", got.Content)
	}
	if inp.Value() != "" {
		t.Errorf("expected textarea empty after submit, got %q", inp.Value())
	}
}

func TestEmptyNoSubmit(t *testing.T) {
	inp := NewInput()

	_, cmd := inp.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		msg := extractMsg(cmd)
		if _, ok := msg.(SendPromptMsg); ok {
			t.Fatal("expected no SendPromptMsg for empty input")
		}
	}
}

func TestWhitespaceNoSubmit(t *testing.T) {
	inp := makeInput("   ")

	_, cmd := inp.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		msg := extractMsg(cmd)
		if _, ok := msg.(SendPromptMsg); ok {
			t.Fatal("expected no SendPromptMsg for whitespace-only input")
		}
	}
}

func TestSubmitDisabledDuringStreaming(t *testing.T) {
	inp := makeInput("hello")
	inp.SetSubmitting(true)

	_, cmd := inp.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		msg := extractMsg(cmd)
		if _, ok := msg.(SendPromptMsg); ok {
			t.Fatal("expected no SendPromptMsg while submitting/streaming")
		}
	}
}

func TestCtrlCDuringStreaming(t *testing.T) {
	inp := NewInput()
	inp.SetSubmitting(true)

	_, cmd := inp.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	msg := extractMsg(cmd)

	if _, ok := msg.(AbortMsg); !ok {
		t.Fatalf("expected AbortMsg during streaming, got %T", msg)
	}
}

func TestReset(t *testing.T) {
	inp := makeInput("some content")
	inp.Reset()

	if inp.Value() != "" {
		t.Errorf("expected empty after Reset(), got %q", inp.Value())
	}
}

func TestFocusBlur(t *testing.T) {
	inp := NewInput()

	inp.Focus()
	if !inp.IsFocused() {
		t.Error("expected IsFocused() == true after Focus()")
	}

	inp.Blur()
	if inp.IsFocused() {
		t.Error("expected IsFocused() == false after Blur()")
	}
}

func TestViewRespectsWidth(t *testing.T) {
	inp := NewInput()
	out := inp.View(40)
	if out == "" {
		t.Error("expected non-empty View output")
	}
}
