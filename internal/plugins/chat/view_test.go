package chat

import (
	"regexp"
	"strings"
	"testing"
	"time"
)

func stripANSI(str string) string {
	re := regexp.MustCompile(`\x1b\[[0-9;]*m`)
	return re.ReplaceAllString(str, "")
}

func TestEmptyState(t *testing.T) {
	output := renderMessages(nil, 80)
	if !strings.Contains(output, "Start a conversation") {
		t.Errorf("Expected empty state to contain 'Start a conversation', got: %s", output)
	}
}

func TestUserMessageRendering(t *testing.T) {
	msgs := []ChatMessage{
		{
			Role:      "user",
			Content:   "Hello world",
			Timestamp: time.Now(),
		},
	}
	output := renderMessages(msgs, 80)
	cleanOutput := stripANSI(output)
	if !strings.Contains(cleanOutput, "Hello world") {
		t.Errorf("Expected output to contain 'Hello world', got: %s", cleanOutput)
	}
	if !strings.Contains(cleanOutput, ">") {
		t.Errorf("Expected output to contain user prefix '>', got: %s", cleanOutput)
	}
}

func TestAssistantMessageRendering(t *testing.T) {
	msgs := []ChatMessage{
		{
			Role:      "assistant",
			Content:   "I am here",
			Timestamp: time.Now(),
		},
	}
	output := renderMessages(msgs, 80)
	cleanOutput := stripANSI(output)
	if !strings.Contains(cleanOutput, "I am here") {
		t.Errorf("Expected output to contain 'I am here', got: %s", cleanOutput)
	}
}

func TestStreamingIndicator(t *testing.T) {
	msgs := []ChatMessage{
		{
			Role:        "assistant",
			Content:     "Generating",
			Timestamp:   time.Now(),
			IsStreaming: true,
		},
	}
	output := renderMessages(msgs, 80)
	if !strings.Contains(output, "▍") {
		t.Errorf("Expected output to contain streaming cursor '▍', got: %s", output)
	}
}

func TestHeightConstraint(t *testing.T) {
	width, height := 80, 5
	vp := NewMessageViewport(width, height)

	msgs := []ChatMessage{}
	for i := 0; i < 10; i++ {
		msgs = append(msgs, ChatMessage{
			Role:      "user",
			Content:   "Line",
			Timestamp: time.Now(),
		})
	}

	vp.SetMessages(msgs)
	view := vp.View()
	lines := strings.Split(view, "\n")

	if len(lines) > height {
		t.Errorf("Expected view height <= %d, got %d lines", height, len(lines))
	}
}

func TestStatusBarReady(t *testing.T) {
	output := renderStatusBar("session-123", false, 80)
	if !strings.Contains(output, "Ready") {
		t.Errorf("Expected status bar to contain 'Ready', got: %s", output)
	}
	if !strings.Contains(output, "session-") {
		t.Errorf("Expected status bar to contain session ID, got: %s", output)
	}
}

func TestStatusBarStreaming(t *testing.T) {
	output := renderStatusBar("session-123", true, 80)
	if !strings.Contains(output, "Generating") {
		t.Errorf("Expected status bar to contain 'Generating', got: %s", output)
	}
}
