package workspace

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestShellManifest_LoadMissing(t *testing.T) {
	// Load from non-existent file should return empty manifest
	path := filepath.Join(t.TempDir(), ".sidecar", "shells.json")
	m, err := LoadShellManifest(path)
	if err != nil {
		t.Fatalf("LoadShellManifest() error = %v", err)
	}
	if m == nil {
		t.Fatal("LoadShellManifest() returned nil")
	}
	if len(m.Shells) != 0 {
		t.Errorf("expected empty shells, got %d", len(m.Shells))
	}
}

func TestShellManifest_SaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".sidecar", "shells.json")

	// Create and save manifest
	m := &ShellManifest{
		Version: manifestVersion,
		Shells:  []ShellDefinition{},
		path:    path,
	}

	def := ShellDefinition{
		TmuxName:    "sidecar-sh-test-1",
		DisplayName: "Test Shell",
		CreatedAt:   time.Now().Truncate(time.Second),
		AgentType:   "claude",
		SkipPerms:   true,
	}

	if err := m.AddShell(def); err != nil {
		t.Fatalf("AddShell() error = %v", err)
	}

	// Load and verify
	m2, err := LoadShellManifest(path)
	if err != nil {
		t.Fatalf("LoadShellManifest() error = %v", err)
	}
	if len(m2.Shells) != 1 {
		t.Fatalf("expected 1 shell, got %d", len(m2.Shells))
	}
	if m2.Shells[0].TmuxName != def.TmuxName {
		t.Errorf("TmuxName = %q, want %q", m2.Shells[0].TmuxName, def.TmuxName)
	}
	if m2.Shells[0].DisplayName != def.DisplayName {
		t.Errorf("DisplayName = %q, want %q", m2.Shells[0].DisplayName, def.DisplayName)
	}
	if m2.Shells[0].AgentType != def.AgentType {
		t.Errorf("AgentType = %q, want %q", m2.Shells[0].AgentType, def.AgentType)
	}
	if m2.Shells[0].SkipPerms != def.SkipPerms {
		t.Errorf("SkipPerms = %v, want %v", m2.Shells[0].SkipPerms, def.SkipPerms)
	}
}

func TestShellManifest_AddRemove(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".sidecar", "shells.json")
	m, _ := LoadShellManifest(path)

	// Add two shells
	m.AddShell(ShellDefinition{TmuxName: "shell-1", DisplayName: "Shell 1"})
	m.AddShell(ShellDefinition{TmuxName: "shell-2", DisplayName: "Shell 2"})

	if len(m.Shells) != 2 {
		t.Fatalf("expected 2 shells, got %d", len(m.Shells))
	}

	// Remove first
	if err := m.RemoveShell("shell-1"); err != nil {
		t.Fatalf("RemoveShell() error = %v", err)
	}

	if len(m.Shells) != 1 {
		t.Fatalf("expected 1 shell after remove, got %d", len(m.Shells))
	}
	if m.Shells[0].TmuxName != "shell-2" {
		t.Errorf("wrong shell remaining: %q", m.Shells[0].TmuxName)
	}
}

func TestShellManifest_FindShell(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".sidecar", "shells.json")
	m, _ := LoadShellManifest(path)

	m.AddShell(ShellDefinition{TmuxName: "shell-1", DisplayName: "Shell 1"})

	// Find existing
	found := m.FindShell("shell-1")
	if found == nil {
		t.Fatal("FindShell() returned nil for existing shell")
	}
	if found.DisplayName != "Shell 1" {
		t.Errorf("DisplayName = %q, want %q", found.DisplayName, "Shell 1")
	}

	// Find non-existing
	if m.FindShell("shell-999") != nil {
		t.Error("FindShell() should return nil for non-existing shell")
	}
}

func TestShellManifest_UpdateShell(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".sidecar", "shells.json")
	m, _ := LoadShellManifest(path)

	m.AddShell(ShellDefinition{TmuxName: "shell-1", DisplayName: "Original"})

	// Update
	if err := m.UpdateShell(ShellDefinition{TmuxName: "shell-1", DisplayName: "Updated"}); err != nil {
		t.Fatalf("UpdateShell() error = %v", err)
	}

	found := m.FindShell("shell-1")
	if found.DisplayName != "Updated" {
		t.Errorf("DisplayName = %q, want %q", found.DisplayName, "Updated")
	}
}

func TestShellManifest_CorruptedFile(t *testing.T) {
	dir := t.TempDir()
	sidecarDir := filepath.Join(dir, ".sidecar")
	os.MkdirAll(sidecarDir, 0755)
	path := filepath.Join(sidecarDir, "shells.json")

	// Write corrupted JSON
	os.WriteFile(path, []byte("{invalid json"), 0644)

	// Should return empty manifest, not error
	m, err := LoadShellManifest(path)
	if err != nil {
		t.Fatalf("LoadShellManifest() error = %v", err)
	}
	if len(m.Shells) != 0 {
		t.Errorf("expected empty shells for corrupted file, got %d", len(m.Shells))
	}
}

func TestShellManifest_AddDuplicate(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".sidecar", "shells.json")
	m, _ := LoadShellManifest(path)

	m.AddShell(ShellDefinition{TmuxName: "shell-1", DisplayName: "Original"})
	m.AddShell(ShellDefinition{TmuxName: "shell-1", DisplayName: "Updated"})

	// Should update, not duplicate
	if len(m.Shells) != 1 {
		t.Fatalf("expected 1 shell, got %d", len(m.Shells))
	}
	if m.Shells[0].DisplayName != "Updated" {
		t.Errorf("DisplayName = %q, want %q", m.Shells[0].DisplayName, "Updated")
	}
}
