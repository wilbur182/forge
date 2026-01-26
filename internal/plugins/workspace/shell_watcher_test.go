package workspace

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestShellWatcher_DetectsFileChange(t *testing.T) {
	dir := t.TempDir()
	sidecarDir := filepath.Join(dir, ".sidecar")
	os.MkdirAll(sidecarDir, 0755)
	manifestPath := filepath.Join(sidecarDir, "shells.json")

	// Create initial file
	os.WriteFile(manifestPath, []byte(`{"version":1,"shells":[]}`), 0644)

	w, err := NewShellWatcher(manifestPath)
	if err != nil {
		t.Fatalf("NewShellWatcher() error = %v", err)
	}
	defer w.Stop()

	msgChan := w.Start()

	// Modify the file
	time.Sleep(50 * time.Millisecond) // Let watcher settle
	os.WriteFile(manifestPath, []byte(`{"version":1,"shells":[{"tmuxName":"test"}]}`), 0644)

	// Should receive change notification
	select {
	case msg := <-msgChan:
		if _, ok := msg.(ShellManifestChangedMsg); !ok {
			t.Errorf("expected ShellManifestChangedMsg, got %T", msg)
		}
	case <-time.After(500 * time.Millisecond):
		t.Error("timeout waiting for ShellManifestChangedMsg")
	}
}

func TestShellWatcher_DebounceRapidChanges(t *testing.T) {
	dir := t.TempDir()
	sidecarDir := filepath.Join(dir, ".sidecar")
	os.MkdirAll(sidecarDir, 0755)
	manifestPath := filepath.Join(sidecarDir, "shells.json")

	// Create initial file
	os.WriteFile(manifestPath, []byte(`{"version":1}`), 0644)

	w, err := NewShellWatcher(manifestPath)
	if err != nil {
		t.Fatalf("NewShellWatcher() error = %v", err)
	}
	defer w.Stop()

	msgChan := w.Start()
	time.Sleep(50 * time.Millisecond)

	// Make multiple rapid changes
	for i := 0; i < 5; i++ {
		os.WriteFile(manifestPath, []byte(`{"version":1,"i":`+string(rune('0'+i))+`}`), 0644)
		time.Sleep(10 * time.Millisecond)
	}

	// Should receive only one debounced notification
	count := 0
	timeout := time.After(300 * time.Millisecond)
	for {
		select {
		case <-msgChan:
			count++
		case <-timeout:
			goto done
		}
	}
done:

	// Expect 1-2 messages due to debouncing (may vary slightly by timing)
	if count == 0 {
		t.Error("expected at least 1 message")
	}
	if count > 2 {
		t.Errorf("expected debounced messages (1-2), got %d", count)
	}
}

func TestShellWatcher_Stop(t *testing.T) {
	dir := t.TempDir()
	manifestPath := filepath.Join(dir, ".sidecar", "shells.json")

	w, err := NewShellWatcher(manifestPath)
	if err != nil {
		t.Fatalf("NewShellWatcher() error = %v", err)
	}

	msgChan := w.Start()
	w.Stop()

	// Channel should be closed after run() exits
	select {
	case _, ok := <-msgChan:
		if ok {
			t.Error("expected channel to be closed")
		}
	case <-time.After(200 * time.Millisecond):
		t.Error("timeout waiting for channel close")
	}

	// Double stop should not panic
	w.Stop()
}
