package workspace

import (
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/fsnotify/fsnotify"
)

// ShellManifestChangedMsg is emitted when the manifest file changes.
// This can be triggered by another sidecar instance modifying the manifest.
type ShellManifestChangedMsg struct{}

// ShellWatcher monitors the shell manifest file for changes.
// When changes are detected, it emits ShellManifestChangedMsg for cross-instance sync.
type ShellWatcher struct {
	fsWatcher *fsnotify.Watcher
	path      string            // path to shells.json
	msgChan   chan tea.Msg      // channel for emitting messages
	stopChan  chan struct{}     // signal to stop
	mu        sync.Mutex
	stopped   bool
}

// debounceDelay for batching rapid file changes.
const shellWatcherDebounce = 100 * time.Millisecond

// NewShellWatcher creates a watcher for the shell manifest file.
func NewShellWatcher(manifestPath string) (*ShellWatcher, error) {
	fsWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	w := &ShellWatcher{
		fsWatcher: fsWatcher,
		path:      manifestPath,
		msgChan:   make(chan tea.Msg, 1),
		stopChan:  make(chan struct{}),
	}

	// Watch the parent directory (for file creation)
	dir := filepath.Dir(manifestPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		fsWatcher.Close()
		return nil, err
	}

	if err := fsWatcher.Add(dir); err != nil {
		slog.Debug("shellwatcher: add dir", "err", err)
	}

	// Also watch the manifest file itself if it exists
	if _, err := os.Stat(manifestPath); err == nil {
		if err := fsWatcher.Add(manifestPath); err != nil {
			slog.Debug("shellwatcher: add manifest", "err", err)
		}
	}

	return w, nil
}

// Start begins watching and returns a channel for receiving messages.
// The channel emits ShellManifestChangedMsg when the manifest changes.
func (w *ShellWatcher) Start() <-chan tea.Msg {
	go w.run()
	return w.msgChan
}

// Stop stops the watcher and closes channels.
func (w *ShellWatcher) Stop() {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.stopped {
		return
	}
	w.stopped = true

	close(w.stopChan)
	w.fsWatcher.Close()
}

// run is the main watch loop.
func (w *ShellWatcher) run() {
	defer close(w.msgChan)

	var debounceTimer *time.Timer
	manifestName := filepath.Base(w.path)

	for {
		select {
		case <-w.stopChan:
			if debounceTimer != nil {
				debounceTimer.Stop()
			}
			return

		case event, ok := <-w.fsWatcher.Events:
			if !ok {
				return
			}

			// Only care about the manifest file
			eventName := filepath.Base(event.Name)
			if eventName != manifestName {
				continue
			}

			// Care about writes, creates, and renames
			if event.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Rename) == 0 {
				continue
			}

			slog.Debug("shellwatcher: event", "op", event.Op, "name", event.Name)

			// If file was just created, add it to watch list
			if event.Op&fsnotify.Create != 0 {
				w.fsWatcher.Add(w.path)
			}

			// Debounce rapid events
			if debounceTimer != nil {
				debounceTimer.Stop()
			}
			debounceTimer = time.AfterFunc(shellWatcherDebounce, func() {
				select {
				case w.msgChan <- ShellManifestChangedMsg{}:
				default:
					// Channel full, skip
				}
			})

		case err, ok := <-w.fsWatcher.Errors:
			if !ok {
				return
			}
			slog.Debug("shellwatcher: error", "err", err)
		}
	}
}
