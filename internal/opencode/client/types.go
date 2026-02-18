package client

import (
	"fmt"
	"strings"
)

// SessionCreateRequest is the request body for creating a session.
type SessionCreateRequest struct{}

// SessionCreateResponse is the response body when creating a session.
type SessionCreateResponse struct {
	ID        string `json:"id"`
	CreatedAt int64  `json:"createdAt"`
}

// MessageSendRequest is the request body for sending a message.
type MessageSendRequest struct {
	Content string `json:"content"`
}

// MessageSendResponse is the response body when sending a message.
type MessageSendResponse struct{}

// SessionListItem is a single session in the list response.
type SessionListItem struct {
	ID        string `json:"id"`
	CreatedAt int64  `json:"createdAt"`
	Title     string `json:"title"`
}

// SessionListResponse is the response body for listing sessions.
type SessionListResponse []SessionListItem

// SSEEvent represents a parsed Server-Sent Event.
type SSEEvent struct {
	Event string
	Data  string
	ID    string
}

// MessagePartDelta is the payload for message.part.delta SSE events.
type MessagePartDelta struct {
	Content string `json:"content"`
}

// MessageComplete is the payload for message.complete SSE events.
type MessageComplete struct {
	SessionID string `json:"sessionID"`
	MessageID string `json:"messageID"`
}

// ErrorResponse is the response body for errors.
type ErrorResponse struct {
	Error string `json:"error"`
	Code  int    `json:"code"`
}

// AbortResponse is the response body for abort requests.
type AbortResponse struct {
	Success bool `json:"success"`
}

// ParseSSEEvent parses a single SSE event block.
// Format: "event: <type>\ndata: <json>\nid: <id>\n"
// Fields are optional except for event and data.
func ParseSSEEvent(raw string) (SSEEvent, error) {
	if raw == "" {
		return SSEEvent{}, fmt.Errorf("empty SSE event")
	}

	event := SSEEvent{}
	lines := strings.Split(strings.TrimRight(raw, "\n"), "\n")

	for _, line := range lines {
		if strings.HasPrefix(line, "event: ") {
			event.Event = strings.TrimPrefix(line, "event: ")
		} else if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")
			if event.Data != "" {
				event.Data += data
			} else {
				event.Data = data
			}
		} else if strings.HasPrefix(line, "id: ") {
			event.ID = strings.TrimPrefix(line, "id: ")
		}
	}

	return event, nil
}

// ParseSSEStream parses a stream of SSE events separated by double newlines.
func ParseSSEStream(raw string) []SSEEvent {
	if raw == "" {
		return []SSEEvent{}
	}

	var events []SSEEvent
	blocks := strings.Split(raw, "\n\n")

	for _, block := range blocks {
		block = strings.TrimSpace(block)
		if block == "" {
			continue
		}
		event, err := ParseSSEEvent(block)
		if err == nil {
			events = append(events, event)
		}
	}

	return events
}
