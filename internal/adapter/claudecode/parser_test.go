package claudecode

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sst/sidecar/internal/adapter"
)

func TestParseContent_String(t *testing.T) {
	a := &Adapter{}

	content := json.RawMessage(`"Hello, world!"`)
	text, toolUses := a.parseContent(content)

	if text != "Hello, world!" {
		t.Errorf("got text %q, want %q", text, "Hello, world!")
	}
	if len(toolUses) != 0 {
		t.Errorf("got %d tool uses, want 0", len(toolUses))
	}
}

func TestParseContent_TextBlock(t *testing.T) {
	a := &Adapter{}

	content := json.RawMessage(`[{"type":"text","text":"First line"},{"type":"text","text":"Second line"}]`)
	text, toolUses := a.parseContent(content)

	if text != "First line\nSecond line" {
		t.Errorf("got text %q, want %q", text, "First line\nSecond line")
	}
	if len(toolUses) != 0 {
		t.Errorf("got %d tool uses, want 0", len(toolUses))
	}
}

func TestParseContent_ToolUse(t *testing.T) {
	a := &Adapter{}

	content := json.RawMessage(`[{"type":"text","text":"Let me read that."},{"type":"tool_use","id":"tool-123","name":"Read","input":{"file_path":"/tmp/test.go"}}]`)
	text, toolUses := a.parseContent(content)

	if text != "Let me read that." {
		t.Errorf("got text %q, want %q", text, "Let me read that.")
	}
	if len(toolUses) != 1 {
		t.Fatalf("got %d tool uses, want 1", len(toolUses))
	}
	if toolUses[0].ID != "tool-123" {
		t.Errorf("got tool ID %q, want %q", toolUses[0].ID, "tool-123")
	}
	if toolUses[0].Name != "Read" {
		t.Errorf("got tool name %q, want %q", toolUses[0].Name, "Read")
	}
}

func TestParseContent_Empty(t *testing.T) {
	a := &Adapter{}

	text, toolUses := a.parseContent(nil)
	if text != "" {
		t.Errorf("got text %q, want empty", text)
	}
	if len(toolUses) != 0 {
		t.Errorf("got %d tool uses, want 0", len(toolUses))
	}

	text, toolUses = a.parseContent(json.RawMessage{})
	if text != "" {
		t.Errorf("got text %q, want empty", text)
	}
}

func TestParseContent_InvalidJSON(t *testing.T) {
	a := &Adapter{}

	content := json.RawMessage(`{invalid}`)
	text, toolUses := a.parseContent(content)

	if text != "" {
		t.Errorf("got text %q, want empty", text)
	}
	if len(toolUses) != 0 {
		t.Errorf("got %d tool uses, want 0", len(toolUses))
	}
}

func TestParseContent_ThinkingBlock(t *testing.T) {
	a := &Adapter{}

	content := json.RawMessage(`[{"type":"thinking","thinking":"Let me think..."},{"type":"text","text":"Answer here."}]`)
	text, toolUses := a.parseContent(content)

	// Thinking blocks should not be included in text
	if text != "Answer here." {
		t.Errorf("got text %q, want %q", text, "Answer here.")
	}
	if len(toolUses) != 0 {
		t.Errorf("got %d tool uses, want 0", len(toolUses))
	}
}

func TestParseSessionMetadata_ValidFile(t *testing.T) {
	a := &Adapter{}
	testFile := filepath.Join("testdata", "valid_session.jsonl")

	meta, err := a.parseSessionMetadata(testFile)
	if err != nil {
		t.Fatalf("parseSessionMetadata failed: %v", err)
	}

	if meta.SessionID != "valid_session" {
		t.Errorf("got SessionID %q, want %q", meta.SessionID, "valid_session")
	}
	if meta.MsgCount != 4 {
		t.Errorf("got MsgCount %d, want 4", meta.MsgCount)
	}
	if meta.CWD != "/home/user/project" {
		t.Errorf("got CWD %q, want %q", meta.CWD, "/home/user/project")
	}
	if meta.Version != "1.0.0" {
		t.Errorf("got Version %q, want %q", meta.Version, "1.0.0")
	}
	if meta.GitBranch != "main" {
		t.Errorf("got GitBranch %q, want %q", meta.GitBranch, "main")
	}
}

func TestParseSessionMetadata_EmptyFile(t *testing.T) {
	a := &Adapter{}
	testFile := filepath.Join("testdata", "empty.jsonl")

	meta, err := a.parseSessionMetadata(testFile)
	if err != nil {
		t.Fatalf("parseSessionMetadata failed: %v", err)
	}

	if meta.MsgCount != 0 {
		t.Errorf("got MsgCount %d, want 0", meta.MsgCount)
	}
	// FirstMsg and LastMsg should be set to current time
	if meta.FirstMsg.IsZero() {
		t.Error("FirstMsg should not be zero")
	}
}

func TestParseSessionMetadata_MalformedFile(t *testing.T) {
	a := &Adapter{}
	testFile := filepath.Join("testdata", "malformed.jsonl")

	meta, err := a.parseSessionMetadata(testFile)
	if err != nil {
		t.Fatalf("parseSessionMetadata should handle malformed lines: %v", err)
	}

	// Should only count the valid line
	if meta.MsgCount != 1 {
		t.Errorf("got MsgCount %d, want 1 (only valid lines)", meta.MsgCount)
	}
}

func TestParseSessionMetadata_NonExistent(t *testing.T) {
	a := &Adapter{}

	_, err := a.parseSessionMetadata("/nonexistent/path.jsonl")
	if err == nil {
		t.Error("expected error for non-existent file")
	}
}

func TestMessagesFromTestdata(t *testing.T) {
	a := &Adapter{}
	testFile := filepath.Join("testdata", "valid_session.jsonl")

	messages := parseMessagesFromFile(t, a, testFile)

	if len(messages) != 4 {
		t.Fatalf("got %d messages, want 4", len(messages))
	}

	// Check first message (user)
	if messages[0].Role != "user" {
		t.Errorf("msg[0] role = %q, want user", messages[0].Role)
	}
	if messages[0].ID != "msg-001" {
		t.Errorf("msg[0] ID = %q, want msg-001", messages[0].ID)
	}

	// Check second message (assistant with usage)
	if messages[1].Role != "assistant" {
		t.Errorf("msg[1] role = %q, want assistant", messages[1].Role)
	}
	if messages[1].InputTokens != 50 {
		t.Errorf("msg[1] InputTokens = %d, want 50", messages[1].InputTokens)
	}
	if messages[1].OutputTokens != 20 {
		t.Errorf("msg[1] OutputTokens = %d, want 20", messages[1].OutputTokens)
	}
	if messages[1].CacheRead != 10 {
		t.Errorf("msg[1] CacheRead = %d, want 10", messages[1].CacheRead)
	}

	// Check fourth message (assistant with tool use)
	if len(messages[3].ToolUses) != 1 {
		t.Fatalf("msg[3] ToolUses len = %d, want 1", len(messages[3].ToolUses))
	}
	if messages[3].ToolUses[0].Name != "Read" {
		t.Errorf("msg[3] tool name = %q, want Read", messages[3].ToolUses[0].Name)
	}
}

func TestMessagesFromMalformed(t *testing.T) {
	a := &Adapter{}
	testFile := filepath.Join("testdata", "malformed.jsonl")

	messages := parseMessagesFromFile(t, a, testFile)

	// Should parse the one valid line
	if len(messages) != 1 {
		t.Errorf("got %d messages, want 1", len(messages))
	}
}

func TestMessagesFromEmpty(t *testing.T) {
	a := &Adapter{}
	testFile := filepath.Join("testdata", "empty.jsonl")

	messages := parseMessagesFromFile(t, a, testFile)

	if len(messages) != 0 {
		t.Errorf("got %d messages, want 0", len(messages))
	}
}

func TestRawMessageParsing(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantType string
		wantUUID string
		wantErr  bool
	}{
		{
			name:     "user message",
			input:    `{"type":"user","uuid":"u-001","timestamp":"2024-01-15T10:00:00Z"}`,
			wantType: "user",
			wantUUID: "u-001",
		},
		{
			name:     "assistant message",
			input:    `{"type":"assistant","uuid":"a-001","timestamp":"2024-01-15T10:00:00Z"}`,
			wantType: "assistant",
			wantUUID: "a-001",
		},
		{
			name:     "tool result skipped",
			input:    `{"type":"tool_result","uuid":"t-001","timestamp":"2024-01-15T10:00:00Z"}`,
			wantType: "tool_result",
			wantUUID: "t-001",
		},
		{
			name:    "invalid json",
			input:   `{not valid json`,
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var raw RawMessage
			err := json.Unmarshal([]byte(tc.input), &raw)

			if tc.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if raw.Type != tc.wantType {
				t.Errorf("got type %q, want %q", raw.Type, tc.wantType)
			}
			if raw.UUID != tc.wantUUID {
				t.Errorf("got uuid %q, want %q", raw.UUID, tc.wantUUID)
			}
		})
	}
}

func TestTimestampParsing(t *testing.T) {
	input := `{"type":"user","uuid":"u-001","timestamp":"2024-01-15T10:30:45Z"}`

	var raw RawMessage
	if err := json.Unmarshal([]byte(input), &raw); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	expected := time.Date(2024, 1, 15, 10, 30, 45, 0, time.UTC)
	if !raw.Timestamp.Equal(expected) {
		t.Errorf("got timestamp %v, want %v", raw.Timestamp, expected)
	}
}

// testMessage mirrors adapter.Message for test assertions.
type testMessage struct {
	ID           string
	Role         string
	Content      string
	ToolUses     []adapter.ToolUse
	InputTokens  int
	OutputTokens int
	CacheRead    int
}

// parseMessagesFromFile is a helper that mimics Messages() but for local files.
func parseMessagesFromFile(t *testing.T, a *Adapter, path string) []testMessage {
	t.Helper()

	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("failed to open %s: %v", path, err)
	}
	defer file.Close()

	var messages []testMessage
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 1024*1024), 10*1024*1024)

	for scanner.Scan() {
		var raw RawMessage
		if err := json.Unmarshal(scanner.Bytes(), &raw); err != nil {
			continue
		}

		if raw.Type != "user" && raw.Type != "assistant" {
			continue
		}
		if raw.Message == nil {
			continue
		}

		msg := testMessage{
			ID:   raw.UUID,
			Role: raw.Message.Role,
		}

		content, toolUses := a.parseContent(raw.Message.Content)
		msg.Content = content
		msg.ToolUses = toolUses

		if raw.Message.Usage != nil {
			msg.InputTokens = raw.Message.Usage.InputTokens
			msg.OutputTokens = raw.Message.Usage.OutputTokens
			msg.CacheRead = raw.Message.Usage.CacheReadInputTokens
		}

		messages = append(messages, msg)
	}

	return messages
}
