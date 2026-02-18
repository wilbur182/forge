package bridge

import (
	tea "github.com/charmbracelet/bubbletea"
	"sync"
)

// EventBus manages inter-plugin communication via events.
type EventBus struct {
	mu        sync.RWMutex
	listeners map[string][]tea.Cmd
}

// FileChangedEvent signals that a file has been modified.
type FileChangedEvent struct {
	Path    string
	Content []byte
}

// TaskUpdatedEvent signals that a task has been updated.
type TaskUpdatedEvent struct {
	Task *Task
}

// AgentMessageEvent signals that the agent has sent a message.
type AgentMessageEvent struct {
	Message AgentMessage
}

// BranchChangedEvent signals that the git branch has changed.
type BranchChangedEvent struct {
	Branch string
}

// NewEventBus creates a new event bus.
func NewEventBus() *EventBus {
	return &EventBus{
		listeners: make(map[string][]tea.Cmd),
	}
}

// Subscribe registers a listener for an event type.
func (eb *EventBus) Subscribe(eventType string, handler tea.Cmd) {
	eb.mu.Lock()
	defer eb.mu.Unlock()
	eb.listeners[eventType] = append(eb.listeners[eventType], handler)
}

// Publish broadcasts an event to all subscribers.
func (eb *EventBus) Publish(eventType string, event interface{}) []tea.Cmd {
	eb.mu.RLock()
	handlers := eb.listeners[eventType]
	eb.mu.RUnlock()

	if len(handlers) == 0 {
		return nil
	}

	cmds := make([]tea.Cmd, len(handlers))
	copy(cmds, handlers)
	return cmds
}

// Clear removes all listeners.
func (eb *EventBus) Clear() {
	eb.mu.Lock()
	defer eb.mu.Unlock()
	eb.listeners = make(map[string][]tea.Cmd)
}
