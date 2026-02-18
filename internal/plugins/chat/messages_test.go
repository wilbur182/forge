package chat

import (
	"errors"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/wilbur182/forge/internal/plugin"
)

// TestAllMessagesImplementTeaMsg verifies all message types satisfy tea.Msg interface.
func TestAllMessagesImplementTeaMsg(t *testing.T) {
	messages := []interface{}{
		// Async messages
		StreamTokenMsg{Token: "hello", Epoch: 1},
		StreamCompleteMsg{Epoch: 1},
		StreamErrorMsg{Err: errors.New("test"), Epoch: 1},
		SessionCreatedMsg{SessionID: "session-123", Epoch: 1},
		AbortCompleteMsg{Epoch: 1},
		ConnectionErrorMsg{Err: errors.New("test"), Epoch: 1},
		// Sync messages
		SendPromptMsg{Content: "test prompt"},
		AbortMsg{},
	}

	for _, msg := range messages {
		var _ tea.Msg = msg
	}
}

// TestAsyncMessagesImplementEpochMessage verifies async messages implement plugin.EpochMessage.
func TestAsyncMessagesImplementEpochMessage(t *testing.T) {
	tests := []struct {
		name      string
		msg       plugin.EpochMessage
		wantEpoch uint64
	}{
		{
			name:      "StreamTokenMsg",
			msg:       StreamTokenMsg{Token: "token", Epoch: 5},
			wantEpoch: 5,
		},
		{
			name:      "StreamCompleteMsg",
			msg:       StreamCompleteMsg{Epoch: 10},
			wantEpoch: 10,
		},
		{
			name:      "StreamErrorMsg",
			msg:       StreamErrorMsg{Err: errors.New("test"), Epoch: 15},
			wantEpoch: 15,
		},
		{
			name:      "SessionCreatedMsg",
			msg:       SessionCreatedMsg{SessionID: "sess-1", Epoch: 20},
			wantEpoch: 20,
		},
		{
			name:      "AbortCompleteMsg",
			msg:       AbortCompleteMsg{Epoch: 25},
			wantEpoch: 25,
		},
		{
			name:      "ConnectionErrorMsg",
			msg:       ConnectionErrorMsg{Err: errors.New("test"), Epoch: 30},
			wantEpoch: 30,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.msg.GetEpoch()
			if got != tt.wantEpoch {
				t.Errorf("GetEpoch() = %d, want %d", got, tt.wantEpoch)
			}
		})
	}
}

// TestSyncMessagesDoNotImplementEpochMessage verifies sync messages don't have GetEpoch.
func TestSyncMessagesDoNotImplementEpochMessage(t *testing.T) {
	// These should NOT compile if they accidentally implement EpochMessage
	var sendPrompt interface{} = SendPromptMsg{Content: "test"}
	var abort interface{} = AbortMsg{}

	// Verify they are tea.Msg
	var _ tea.Msg = sendPrompt
	var _ tea.Msg = abort

	// Verify they are NOT plugin.EpochMessage (using negative assertion)
	_, sendPromptIsEpoch := sendPrompt.(plugin.EpochMessage)
	_, abortIsEpoch := abort.(plugin.EpochMessage)

	if sendPromptIsEpoch {
		t.Error("SendPromptMsg should not implement EpochMessage")
	}
	if abortIsEpoch {
		t.Error("AbortMsg should not implement EpochMessage")
	}
}
