package client

import (
	"encoding/json"
	"testing"
)

// TestSessionCreateResponseRoundtrip tests JSON marshal/unmarshal roundtrip
func TestSessionCreateResponseRoundtrip(t *testing.T) {
	original := SessionCreateResponse{
		ID:        "sess-123",
		CreatedAt: 1705001234567,
	}

	// Marshal
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	// Unmarshal
	var result SessionCreateResponse
	err = json.Unmarshal(data, &result)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	// Verify
	if result.ID != original.ID || result.CreatedAt != original.CreatedAt {
		t.Errorf("Roundtrip mismatch: got %+v, want %+v", result, original)
	}
}

// TestMessageSendRequestRoundtrip tests JSON marshal/unmarshal roundtrip
func TestMessageSendRequestRoundtrip(t *testing.T) {
	original := MessageSendRequest{
		Content: "Hello, OpenCode!",
	}

	// Marshal
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	// Unmarshal
	var result MessageSendRequest
	err = json.Unmarshal(data, &result)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	// Verify
	if result.Content != original.Content {
		t.Errorf("Roundtrip mismatch: got %+v, want %+v", result, original)
	}
}

// TestSessionListResponseRoundtrip tests JSON marshal/unmarshal roundtrip
func TestSessionListResponseRoundtrip(t *testing.T) {
	original := SessionListResponse{
		{
			ID:        "sess-1",
			CreatedAt: 1705001234567,
			Title:     "Session 1",
		},
		{
			ID:        "sess-2",
			CreatedAt: 1705001234568,
			Title:     "Session 2",
		},
	}

	// Marshal
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	// Unmarshal
	var result SessionListResponse
	err = json.Unmarshal(data, &result)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	// Verify
	if len(result) != len(original) {
		t.Errorf("Length mismatch: got %d, want %d", len(result), len(original))
	}
	for i, item := range result {
		if item.ID != original[i].ID || item.CreatedAt != original[i].CreatedAt || item.Title != original[i].Title {
			t.Errorf("Item %d mismatch: got %+v, want %+v", i, item, original[i])
		}
	}
}

// TestErrorResponseRoundtrip tests JSON marshal/unmarshal roundtrip
func TestErrorResponseRoundtrip(t *testing.T) {
	original := ErrorResponse{
		Error: "Not found",
		Code:  404,
	}

	// Marshal
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	// Unmarshal
	var result ErrorResponse
	err = json.Unmarshal(data, &result)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	// Verify
	if result.Error != original.Error || result.Code != original.Code {
		t.Errorf("Roundtrip mismatch: got %+v, want %+v", result, original)
	}
}

// TestAbortResponseRoundtrip tests JSON marshal/unmarshal roundtrip
func TestAbortResponseRoundtrip(t *testing.T) {
	original := AbortResponse{
		Success: true,
	}

	// Marshal
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	// Unmarshal
	var result AbortResponse
	err = json.Unmarshal(data, &result)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	// Verify
	if result.Success != original.Success {
		t.Errorf("Roundtrip mismatch: got %+v, want %+v", result, original)
	}
}

// TestMessagePartDeltaRoundtrip tests JSON marshal/unmarshal roundtrip
func TestMessagePartDeltaRoundtrip(t *testing.T) {
	original := MessagePartDelta{
		Content: "streaming content here",
	}

	// Marshal
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	// Unmarshal
	var result MessagePartDelta
	err = json.Unmarshal(data, &result)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	// Verify
	if result.Content != original.Content {
		t.Errorf("Roundtrip mismatch: got %+v, want %+v", result, original)
	}
}

// TestMessageCompleteRoundtrip tests JSON marshal/unmarshal roundtrip
func TestMessageCompleteRoundtrip(t *testing.T) {
	original := MessageComplete{
		SessionID: "sess-123",
		MessageID: "msg-456",
	}

	// Marshal
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	// Unmarshal
	var result MessageComplete
	err = json.Unmarshal(data, &result)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	// Verify
	if result.SessionID != original.SessionID || result.MessageID != original.MessageID {
		t.Errorf("Roundtrip mismatch: got %+v, want %+v", result, original)
	}
}

// TestParseSSEEventSingleEvent tests parsing a single SSE event
func TestParseSSEEventSingleEvent(t *testing.T) {
	raw := "event: message.part.delta\ndata: {\"content\":\"hello\"}\nid: 123\n"
	event, err := ParseSSEEvent(raw)
	if err != nil {
		t.Fatalf("ParseSSEEvent failed: %v", err)
	}

	if event.Event != "message.part.delta" {
		t.Errorf("Event mismatch: got %q, want %q", event.Event, "message.part.delta")
	}
	if event.Data != "{\"content\":\"hello\"}" {
		t.Errorf("Data mismatch: got %q, want %q", event.Data, "{\"content\":\"hello\"}")
	}
	if event.ID != "123" {
		t.Errorf("ID mismatch: got %q, want %q", event.ID, "123")
	}
}

// TestParseSSEEventWithoutID tests parsing SSE event without id field
func TestParseSSEEventWithoutID(t *testing.T) {
	raw := "event: message.complete\ndata: {\"sessionID\":\"s1\",\"messageID\":\"m1\"}\n"
	event, err := ParseSSEEvent(raw)
	if err != nil {
		t.Fatalf("ParseSSEEvent failed: %v", err)
	}

	if event.Event != "message.complete" {
		t.Errorf("Event mismatch: got %q, want %q", event.Event, "message.complete")
	}
	if event.Data != "{\"sessionID\":\"s1\",\"messageID\":\"m1\"}" {
		t.Errorf("Data mismatch: got %q, want %q", event.Data, "{\"sessionID\":\"s1\",\"messageID\":\"m1\"}")
	}
	if event.ID != "" {
		t.Errorf("ID should be empty: got %q", event.ID)
	}
}

// TestParseSSEEventWithMultilineData tests parsing SSE event with multiline data
func TestParseSSEEventWithMultilineData(t *testing.T) {
	raw := "event: test.event\ndata: {\"field1\":\"value1\",\ndata: \"field2\":\"value2\"}\nid: 456\n"
	event, err := ParseSSEEvent(raw)
	if err != nil {
		t.Fatalf("ParseSSEEvent failed: %v", err)
	}

	if event.Event != "test.event" {
		t.Errorf("Event mismatch: got %q, want %q", event.Event, "test.event")
	}
	// Multiple data lines are concatenated without newline
	expectedData := "{\"field1\":\"value1\",\"field2\":\"value2\"}"
	if event.Data != expectedData {
		t.Errorf("Data mismatch: got %q, want %q", event.Data, expectedData)
	}
	if event.ID != "456" {
		t.Errorf("ID mismatch: got %q, want %q", event.ID, "456")
	}
}

// TestParseSSEStreamMultipleEvents tests parsing multiple SSE events
func TestParseSSEStreamMultipleEvents(t *testing.T) {
	raw := `event: message.part.delta
data: {"content":"hello"}
id: 1

event: message.part.delta
data: {"content":" world"}
id: 2

event: message.complete
data: {"sessionID":"s1","messageID":"m1"}
id: 3

`

	events := ParseSSEStream(raw)

	if len(events) != 3 {
		t.Fatalf("Expected 3 events, got %d", len(events))
	}

	// Check first event
	if events[0].Event != "message.part.delta" {
		t.Errorf("Event 1: got %q, want %q", events[0].Event, "message.part.delta")
	}
	if events[0].ID != "1" {
		t.Errorf("Event 1 ID: got %q, want %q", events[0].ID, "1")
	}

	// Check second event
	if events[1].Event != "message.part.delta" {
		t.Errorf("Event 2: got %q, want %q", events[1].Event, "message.part.delta")
	}
	if events[1].ID != "2" {
		t.Errorf("Event 2 ID: got %q, want %q", events[1].ID, "2")
	}

	// Check third event
	if events[2].Event != "message.complete" {
		t.Errorf("Event 3: got %q, want %q", events[2].Event, "message.complete")
	}
	if events[2].ID != "3" {
		t.Errorf("Event 3 ID: got %q, want %q", events[2].ID, "3")
	}
}

// TestParseSSEStreamEmpty tests parsing empty SSE stream
func TestParseSSEStreamEmpty(t *testing.T) {
	raw := ""
	events := ParseSSEStream(raw)

	if len(events) != 0 {
		t.Errorf("Expected 0 events, got %d", len(events))
	}
}
