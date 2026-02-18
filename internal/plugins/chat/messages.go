package chat

import tea "github.com/charmbracelet/bubbletea"

// StreamTokenMsg carries a single token from an SSE stream.
// Implements tea.Msg and plugin.EpochMessage.
type StreamTokenMsg struct {
	Token string // The token text
	Epoch uint64 // Epoch when the request was issued (for stale detection)
}

// GetEpoch implements plugin.EpochMessage.
func (m StreamTokenMsg) GetEpoch() uint64 { return m.Epoch }

// StreamCompleteMsg signals that an SSE stream has finished successfully.
// Implements tea.Msg and plugin.EpochMessage.
type StreamCompleteMsg struct {
	Epoch uint64 // Epoch when the request was issued (for stale detection)
}

// GetEpoch implements plugin.EpochMessage.
func (m StreamCompleteMsg) GetEpoch() uint64 { return m.Epoch }

// StreamErrorMsg carries an error from an SSE stream.
// Implements tea.Msg and plugin.EpochMessage.
type StreamErrorMsg struct {
	Err   error  // The error that occurred
	Epoch uint64 // Epoch when the request was issued (for stale detection)
}

// GetEpoch implements plugin.EpochMessage.
func (m StreamErrorMsg) GetEpoch() uint64 { return m.Epoch }

// SessionCreatedMsg signals that a new chat session was created.
// Implements tea.Msg and plugin.EpochMessage.
type SessionCreatedMsg struct {
	SessionID string // The ID of the newly created session
	Epoch     uint64 // Epoch when the request was issued (for stale detection)
}

// GetEpoch implements plugin.EpochMessage.
func (m SessionCreatedMsg) GetEpoch() uint64 { return m.Epoch }

// AbortCompleteMsg signals that an abort operation was confirmed by the API.
// Implements tea.Msg and plugin.EpochMessage.
type AbortCompleteMsg struct {
	Epoch uint64 // Epoch when the request was issued (for stale detection)
}

// GetEpoch implements plugin.EpochMessage.
func (m AbortCompleteMsg) GetEpoch() uint64 { return m.Epoch }

// ConnectionErrorMsg carries an error when unable to connect to OpenCode.
// Implements tea.Msg and plugin.EpochMessage.
type ConnectionErrorMsg struct {
	Err   error  // The connection error
	Epoch uint64 // Epoch when the request was issued (for stale detection)
}

// GetEpoch implements plugin.EpochMessage.
func (m ConnectionErrorMsg) GetEpoch() uint64 { return m.Epoch }

// SendPromptMsg signals that the user submitted a prompt.
// Implements tea.Msg only (synchronous message, no epoch needed).
type SendPromptMsg struct {
	Content string // The prompt text
}

// AbortMsg signals that the user requested an abort operation.
// Implements tea.Msg only (synchronous message, no epoch needed).
type AbortMsg struct{}

var _ tea.Msg = StreamTokenMsg{}
var _ tea.Msg = StreamCompleteMsg{}
var _ tea.Msg = StreamErrorMsg{}
var _ tea.Msg = SessionCreatedMsg{}
var _ tea.Msg = AbortCompleteMsg{}
var _ tea.Msg = ConnectionErrorMsg{}
var _ tea.Msg = SendPromptMsg{}
var _ tea.Msg = AbortMsg{}
