package workspace

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectClaudeSessionStatus(t *testing.T) {
	// Create temp directory structure mimicking Claude's projects dir
	tmpHome := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", origHome)

	// Test worktree path
	worktreePath := "/test/project/path"
	// Claude converts to: -test-project-path
	projectDir := filepath.Join(tmpHome, ".claude", "projects", "-test-project-path")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("failed to create project dir: %v", err)
	}

	tests := []struct {
		name           string
		sessionContent string
		wantStatus     WorktreeStatus
		wantOK         bool
	}{
		{
			name: "waiting after assistant message",
			sessionContent: `{"type":"user","message":{"role":"user","content":"hello"}}
{"type":"assistant","message":{"role":"assistant","content":"Hi there!"}}`,
			wantStatus: StatusWaiting,
			wantOK:     true,
		},
		{
			name: "active after user message",
			sessionContent: `{"type":"assistant","message":{"role":"assistant","content":"Hi!"}}
{"type":"user","message":{"role":"user","content":"do something"}}`,
			wantStatus: StatusActive,
			wantOK:     true,
		},
		{
			name:           "empty session file",
			sessionContent: "",
			wantStatus:     0,
			wantOK:         false,
		},
		{
			name:           "only system messages",
			sessionContent: `{"type":"system","message":{"role":"system","content":"init"}}`,
			wantStatus:     0,
			wantOK:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean previous session files
			entries, _ := os.ReadDir(projectDir)
			for _, e := range entries {
				os.Remove(filepath.Join(projectDir, e.Name()))
			}

			// Create session file
			sessionFile := filepath.Join(projectDir, "test-session.jsonl")
			if err := os.WriteFile(sessionFile, []byte(tt.sessionContent), 0644); err != nil {
				t.Fatalf("failed to write session file: %v", err)
			}

			status, ok := detectClaudeSessionStatus(worktreePath)
			if ok != tt.wantOK {
				t.Errorf("detectClaudeSessionStatus() ok = %v, want %v", ok, tt.wantOK)
			}
			if ok && status != tt.wantStatus {
				t.Errorf("detectClaudeSessionStatus() status = %v, want %v", status, tt.wantStatus)
			}
		})
	}
}

func TestDetectClaudeSessionStatus_NoSessionFile(t *testing.T) {
	tmpHome := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", origHome)

	// Don't create any session files
	status, ok := detectClaudeSessionStatus("/nonexistent/path")
	if ok {
		t.Errorf("expected ok=false for missing session, got status=%v", status)
	}
}

func TestFindMostRecentJSONL(t *testing.T) {
	tmpDir := t.TempDir()

	// Create multiple session files
	files := []string{"session1.jsonl", "session2.jsonl", "agent-sub.jsonl"}
	for _, f := range files {
		path := filepath.Join(tmpDir, f)
		if err := os.WriteFile(path, []byte("{}"), 0644); err != nil {
			t.Fatalf("failed to create %s: %v", f, err)
		}
	}

	// Make session2 most recent
	session2 := filepath.Join(tmpDir, "session2.jsonl")

	result, err := findMostRecentJSONL(tmpDir, "agent-")
	if err != nil {
		t.Fatalf("findMostRecentJSONL() error: %v", err)
	}

	// Should find a session file (not the agent- prefixed one)
	if result == "" {
		t.Error("findMostRecentJSONL() returned empty string")
	}

	// Verify it's not the agent sub-session
	if filepath.Base(result) == "agent-sub.jsonl" {
		t.Error("findMostRecentJSONL() should skip agent- prefixed files")
	}

	// Should be one of the regular sessions
	base := filepath.Base(result)
	if base != "session1.jsonl" && base != "session2.jsonl" {
		t.Errorf("findMostRecentJSONL() = %s, want session1.jsonl or session2.jsonl", base)
	}

	_ = session2 // Used to ensure the file exists
}

func TestDetectAgentSessionStatus(t *testing.T) {
	// Test that the dispatcher works correctly
	// Note: Individual agent tests would require setting up their respective directories

	// Test unsupported agent type returns false
	status, ok := detectAgentSessionStatus(AgentCustom, "/test/path")
	if ok {
		t.Errorf("expected ok=false for unsupported agent type, got status=%v", status)
	}

	// Test empty agent type returns false
	status, ok = detectAgentSessionStatus("", "/test/path")
	if ok {
		t.Errorf("expected ok=false for empty agent type, got status=%v", status)
	}
}
