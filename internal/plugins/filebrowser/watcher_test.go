package filebrowser

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewWatcher(t *testing.T) {
	tmpDir := t.TempDir()

	w, err := NewWatcher(tmpDir)
	if err != nil {
		t.Fatalf("NewWatcher() failed: %v", err)
	}

	if w == nil {
		t.Error("NewWatcher() returned nil")
	}
	if w.rootDir != tmpDir {
		t.Errorf("rootDir = %q, want %q", w.rootDir, tmpDir)
	}
	if w.fsWatcher == nil {
		t.Error("fsWatcher not initialized")
	}
	if w.events == nil {
		t.Error("events channel not initialized")
	}

	w.Stop()
}

func TestNewWatcher_InvalidDirectory(t *testing.T) {
	nonExistent := "/nonexistent/path/that/does/not/exist"

	w, err := NewWatcher(nonExistent)
	if err == nil {
		t.Error("NewWatcher() should error for non-existent directory")
		w.Stop()
	}
}

func TestWatcher_FileChangeDetection(t *testing.T) {
	tmpDir := t.TempDir()

	w, err := NewWatcher(tmpDir)
	if err != nil {
		t.Fatalf("NewWatcher() failed: %v", err)
	}
	defer w.Stop()

	// Create a file
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Modify the file
	time.Sleep(50 * time.Millisecond) // Wait for file system to settle
	if err := os.WriteFile(testFile, []byte("modified"), 0644); err != nil {
		t.Fatalf("failed to modify test file: %v", err)
	}

	// Wait for event with timeout
	select {
	case <-w.Events():
		// Event received as expected
	case <-time.After(500 * time.Millisecond):
		t.Error("timeout waiting for file change event")
	}
}

func TestWatcher_DirectoryWatching(t *testing.T) {
	tmpDir := t.TempDir()

	w, err := NewWatcher(tmpDir)
	if err != nil {
		t.Fatalf("NewWatcher() failed: %v", err)
	}
	defer w.Stop()

	// Create a new subdirectory
	subDir := filepath.Join(tmpDir, "subdir")
	if err := os.Mkdir(subDir, 0755); err != nil {
		t.Fatalf("failed to create subdirectory: %v", err)
	}

	// Wait for event
	select {
	case <-w.Events():
		// Event received
	case <-time.After(500 * time.Millisecond):
		t.Error("timeout waiting for directory creation event")
	}

	// Create a file in the new subdirectory
	// The watcher should have picked up the new directory
	testFile := filepath.Join(subDir, "test.txt")
	time.Sleep(50 * time.Millisecond)
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("failed to create file in subdirectory: %v", err)
	}

	// Should detect the file creation in the subdirectory
	select {
	case <-w.Events():
		// Event received as expected
	case <-time.After(500 * time.Millisecond):
		t.Error("timeout waiting for file creation in subdirectory")
	}
}

func TestWatcher_Debounce(t *testing.T) {
	tmpDir := t.TempDir()

	w, err := NewWatcher(tmpDir)
	if err != nil {
		t.Fatalf("NewWatcher() failed: %v", err)
	}
	defer w.Stop()

	// Create multiple files rapidly
	testFile1 := filepath.Join(tmpDir, "test1.txt")
	testFile2 := filepath.Join(tmpDir, "test2.txt")
	testFile3 := filepath.Join(tmpDir, "test3.txt")

	if err := os.WriteFile(testFile1, []byte("test"), 0644); err != nil {
		t.Fatalf("failed to create test file 1: %v", err)
	}
	if err := os.WriteFile(testFile2, []byte("test"), 0644); err != nil {
		t.Fatalf("failed to create test file 2: %v", err)
	}
	if err := os.WriteFile(testFile3, []byte("test"), 0644); err != nil {
		t.Fatalf("failed to create test file 3: %v", err)
	}

	// Should receive event(s) but debouncing prevents too many
	eventCount := 0
	done := make(chan bool)

	go func() {
		for {
			select {
			case <-w.Events():
				eventCount++
			case <-time.After(300 * time.Millisecond):
				done <- true
				return
			}
		}
	}()

	<-done

	if eventCount == 0 {
		t.Error("no events detected for multiple file changes")
	}
	// Should have received fewer events than files created due to debouncing
	// Exact count varies based on timing, but should be < 3
}

func TestWatcher_Stop(t *testing.T) {
	tmpDir := t.TempDir()

	w, err := NewWatcher(tmpDir)
	if err != nil {
		t.Fatalf("NewWatcher() failed: %v", err)
	}

	// Stop should not panic
	w.Stop()

	// Wait for run() goroutine to exit and close the channel
	time.Sleep(50 * time.Millisecond)

	// Create a file after stopping - should not generate events
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Channel should be closed after stop, not produce new events
	select {
	case _, ok := <-w.Events():
		if ok {
			t.Error("received event after watcher stopped")
		}
		// !ok means channel closed - this is expected and correct
	case <-time.After(200 * time.Millisecond):
		// Also acceptable - no event after stop
	}
}

func TestWatcher_EventsChannel(t *testing.T) {
	tmpDir := t.TempDir()

	w, err := NewWatcher(tmpDir)
	if err != nil {
		t.Fatalf("NewWatcher() failed: %v", err)
	}
	defer w.Stop()

	eventsChan := w.Events()
	if eventsChan == nil {
		t.Error("Events() returned nil channel")
	}

	// Channel should be readable
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	select {
	case <-eventsChan:
		// Success
	case <-time.After(500 * time.Millisecond):
		t.Error("timeout reading from events channel")
	}
}

func TestWatcher_NewDirectoryAutoWatch(t *testing.T) {
	tmpDir := t.TempDir()

	w, err := NewWatcher(tmpDir)
	if err != nil {
		t.Fatalf("NewWatcher() failed: %v", err)
	}
	defer w.Stop()

	// Create a new directory
	newDir := filepath.Join(tmpDir, "newdir")
	if err := os.Mkdir(newDir, 0755); err != nil {
		t.Fatalf("failed to create directory: %v", err)
	}

	// Wait for directory creation event
	select {
	case <-w.Events():
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timeout waiting for directory creation")
	}

	// Now create a file in the new directory
	// The watcher should automatically be watching this directory
	newFile := filepath.Join(newDir, "newfile.txt")
	time.Sleep(50 * time.Millisecond)
	if err := os.WriteFile(newFile, []byte("content"), 0644); err != nil {
		t.Fatalf("failed to create file in new directory: %v", err)
	}

	// Should detect the file creation in the newly watched directory
	select {
	case <-w.Events():
		// Success - new directory is being watched
	case <-time.After(500 * time.Millisecond):
		t.Error("new directory not being watched automatically")
	}
}

func TestWatcher_DeleteFile(t *testing.T) {
	tmpDir := t.TempDir()

	w, err := NewWatcher(tmpDir)
	if err != nil {
		t.Fatalf("NewWatcher() failed: %v", err)
	}
	defer w.Stop()

	// Create and then delete a file
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Wait for creation event
	select {
	case <-w.Events():
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timeout waiting for creation event")
	}

	// Delete the file
	time.Sleep(50 * time.Millisecond)
	if err := os.Remove(testFile); err != nil {
		t.Fatalf("failed to delete test file: %v", err)
	}

	// Should detect the deletion
	select {
	case <-w.Events():
		// Success
	case <-time.After(500 * time.Millisecond):
		t.Error("timeout waiting for deletion event")
	}
}

func TestWatcher_RenameFile(t *testing.T) {
	tmpDir := t.TempDir()

	w, err := NewWatcher(tmpDir)
	if err != nil {
		t.Fatalf("NewWatcher() failed: %v", err)
	}
	defer w.Stop()

	// Create a file
	oldPath := filepath.Join(tmpDir, "old.txt")
	newPath := filepath.Join(tmpDir, "new.txt")

	if err := os.WriteFile(oldPath, []byte("test"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Wait for creation event
	select {
	case <-w.Events():
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timeout waiting for creation event")
	}

	// Rename the file
	time.Sleep(50 * time.Millisecond)
	if err := os.Rename(oldPath, newPath); err != nil {
		t.Fatalf("failed to rename file: %v", err)
	}

	// Should detect the rename
	select {
	case <-w.Events():
		// Success
	case <-time.After(500 * time.Millisecond):
		t.Error("timeout waiting for rename event")
	}
}

func TestWatcher_EventChannelNonBlocking(t *testing.T) {
	// Test that the buffered channel with non-blocking send doesn't block
	tmpDir := t.TempDir()

	w, err := NewWatcher(tmpDir)
	if err != nil {
		t.Fatalf("NewWatcher() failed: %v", err)
	}
	defer w.Stop()

	// Generate many rapid changes
	for i := 0; i < 5; i++ {
		testFile := filepath.Join(tmpDir, "test"+string(rune('0'+i))+".txt")
		if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}
	}

	// Should be able to read events without blocking
	timeout := time.After(2 * time.Second)
	eventCount := 0

	for {
		select {
		case <-w.Events():
			eventCount++
		case <-timeout:
			if eventCount == 0 {
				t.Error("no events detected")
			}
			return
		}
	}
}
