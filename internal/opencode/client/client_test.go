package client

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestCreateSession(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/session" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("directory"); got != "/tmp/project" {
			t.Fatalf("unexpected directory query: %q", got)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"ses_1","createdAt":123}`))
	}))
	defer server.Close()

	c := New(server.URL, "/tmp/project")

	resp, err := c.CreateSession(context.Background())
	if err != nil {
		t.Fatalf("CreateSession returned error: %v", err)
	}
	if resp.ID != "ses_1" {
		t.Fatalf("unexpected id: %q", resp.ID)
	}
	if resp.CreatedAt != 123 {
		t.Fatalf("unexpected createdAt: %d", resp.CreatedAt)
	}
}

func TestSendMessage(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/session/ses_123/message" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("directory"); got != "/tmp/project" {
			t.Fatalf("unexpected directory query: %q", got)
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		defer r.Body.Close()

		var req MessageSendRequest
		if err := json.Unmarshal(body, &req); err != nil {
			t.Fatalf("unmarshal body: %v", err)
		}
		if req.Content != "hello world" {
			t.Fatalf("unexpected content: %q", req.Content)
		}

		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	c := New(server.URL, "/tmp/project")
	if err := c.SendMessage(context.Background(), "ses_123", "hello world"); err != nil {
		t.Fatalf("SendMessage returned error: %v", err)
	}
}

func TestStreamEvents(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/event" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("directory"); got != "/tmp/project" {
			t.Fatalf("unexpected directory query: %q", got)
		}

		w.Header().Set("Content-Type", "text/event-stream")
		flusher, ok := w.(http.Flusher)
		if !ok {
			t.Fatalf("response writer does not support flushing")
		}

		events := []string{
			"event: message.part.delta\ndata: {\"content\":\"a\"}\nid: 1\n\n",
			"event: message.part.delta\ndata: {\"content\":\"b\"}\nid: 2\n\n",
			"event: message.complete\ndata: {\"sessionID\":\"ses_1\",\"messageID\":\"msg_1\"}\nid: 3\n\n",
		}

		for _, raw := range events {
			_, _ = w.Write([]byte(raw))
			flusher.Flush()
			time.Sleep(20 * time.Millisecond)
		}
	}))
	defer server.Close()

	c := New(server.URL, "/tmp/project")
	ch, err := c.StreamEvents(context.Background())
	if err != nil {
		t.Fatalf("StreamEvents returned error: %v", err)
	}

	var got []SSEEvent
	timer := time.NewTimer(2 * time.Second)
	defer timer.Stop()

	for len(got) < 3 {
		select {
		case ev, ok := <-ch:
			if !ok {
				t.Fatalf("stream channel closed early after %d events", len(got))
			}
			got = append(got, ev)
		case <-timer.C:
			t.Fatalf("timed out waiting for events, got=%d", len(got))
		}
	}

	if got[0].ID != "1" || got[0].Event != "message.part.delta" {
		t.Fatalf("unexpected first event: %+v", got[0])
	}
	if got[1].ID != "2" || got[1].Event != "message.part.delta" {
		t.Fatalf("unexpected second event: %+v", got[1])
	}
	if got[2].ID != "3" || got[2].Event != "message.complete" {
		t.Fatalf("unexpected third event: %+v", got[2])
	}
}

func TestStreamEventsCancel(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher, ok := w.(http.Flusher)
		if !ok {
			t.Fatalf("response writer does not support flushing")
		}

		_, _ = w.Write([]byte("event: message.part.delta\ndata: {\"content\":\"start\"}\nid: 1\n\n"))
		flusher.Flush()

		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-r.Context().Done():
				return
			case <-ticker.C:
				_, err := w.Write([]byte("event: message.part.delta\ndata: {\"content\":\"tick\"}\nid: 2\n\n"))
				if err != nil {
					return
				}
				flusher.Flush()
			}
		}
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	c := New(server.URL, "/tmp/project")

	ch, err := c.StreamEvents(ctx)
	if err != nil {
		t.Fatalf("StreamEvents returned error: %v", err)
	}

	select {
	case <-ch:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for first event")
	}

	cancel()

	select {
	case _, ok := <-ch:
		if ok {
			t.Fatal("expected closed channel after cancel")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for stream channel close")
	}
}

func TestAbortGeneration(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/session/ses_1/abort" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true}`))
	}))
	defer server.Close()

	c := New(server.URL, "/tmp/project")
	if err := c.AbortGeneration(context.Background(), "ses_1"); err != nil {
		t.Fatalf("AbortGeneration returned error: %v", err)
	}
}

func TestListSessions(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/session" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"id":"ses_1","createdAt":1,"title":"first"},{"id":"ses_2","createdAt":2,"title":"second"}]`))
	}))
	defer server.Close()

	c := New(server.URL, "/tmp/project")
	sessions, err := c.ListSessions(context.Background())
	if err != nil {
		t.Fatalf("ListSessions returned error: %v", err)
	}
	if len(sessions) != 2 {
		t.Fatalf("unexpected sessions len: %d", len(sessions))
	}
	if sessions[0].ID != "ses_1" || sessions[1].ID != "ses_2" {
		t.Fatalf("unexpected sessions: %+v", sessions)
	}
}

func TestPing(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/session" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	c := New(server.URL, "/tmp/project")
	if err := c.Ping(context.Background()); err != nil {
		t.Fatalf("Ping returned error: %v", err)
	}
}

func TestPingUnreachable(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	baseURL := server.URL
	server.Close()

	c := New(baseURL, "/tmp/project")
	err := c.Ping(context.Background())
	if err == nil {
		t.Fatal("expected ping error for unreachable server")
	}
	if !strings.Contains(err.Error(), "opencode: server unreachable") {
		t.Fatalf("expected descriptive unreachable error, got: %v", err)
	}
}

func TestErrorResponse(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"bad request","code":400}`))
	}))
	defer server.Close()

	c := New(server.URL, "/tmp/project")
	_, err := c.CreateSession(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "bad request") {
		t.Fatalf("expected parsed error body, got: %v", err)
	}
	if !strings.Contains(err.Error(), "400") {
		t.Fatalf("expected status/code in error, got: %v", err)
	}
}
