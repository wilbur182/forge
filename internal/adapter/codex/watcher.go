package codex

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/sst/sidecar/internal/adapter"
)

// NewWatcher creates a watcher for Codex session changes.
func NewWatcher(root string) (<-chan adapter.Event, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	if err := addWatchTree(watcher, root); err != nil {
		watcher.Close()
		return nil, err
	}

	events := make(chan adapter.Event, 32)

	go func() {
		defer watcher.Close()
		defer close(events)

		var debounceTimer *time.Timer
		var lastEvent fsnotify.Event
		debounceDelay := 100 * time.Millisecond

		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}

				if event.Op&fsnotify.Create != 0 {
					if info, err := os.Stat(event.Name); err == nil && info.IsDir() {
						_ = addWatchTree(watcher, event.Name)
						continue
					}
				}

				if !strings.HasSuffix(event.Name, ".jsonl") {
					continue
				}

				lastEvent = event
				if debounceTimer != nil {
					debounceTimer.Stop()
				}
				debounceTimer = time.AfterFunc(debounceDelay, func() {
					sessionID := strings.TrimSuffix(filepath.Base(lastEvent.Name), ".jsonl")
					var eventType adapter.EventType
					switch {
					case lastEvent.Op&fsnotify.Create != 0:
						eventType = adapter.EventSessionCreated
					case lastEvent.Op&fsnotify.Write != 0:
						eventType = adapter.EventMessageAdded
					case lastEvent.Op&fsnotify.Remove != 0:
						return
					default:
						eventType = adapter.EventSessionUpdated
					}

					select {
					case events <- adapter.Event{Type: eventType, SessionID: sessionID}:
					default:
					}
				})

			case _, ok := <-watcher.Errors:
				if !ok {
					return
				}
			}
		}
	}()

	return events, nil
}

func addWatchTree(watcher *fsnotify.Watcher, root string) error {
	info, err := os.Stat(root)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return nil
	}

	return filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			_ = watcher.Add(path)
		}
		return nil
	})
}
