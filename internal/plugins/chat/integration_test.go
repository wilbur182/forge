package chat

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/wilbur182/forge/internal/opencode/client"
)

func TestIntegrationMultiTurn(t *testing.T) {
	server := newMultiTurnMockOpenCode(t, [][]string{
		{
			"event: message.part.delta\ndata: {\"content\":\"first reply\"}\nid: 1",
			"event: message.complete\ndata: {\"sessionID\":\"ses_test\",\"messageID\":\"msg_1\"}\nid: 2",
		},
		{
			"event: message.part.delta\ndata: {\"content\":\"second reply\"}\nid: 3",
			"event: message.complete\ndata: {\"sessionID\":\"ses_test\",\"messageID\":\"msg_2\"}\nid: 4",
		},
	})
	defer server.Close()

	p := New()
	if err := p.Init(makeCtx(1)); err != nil {
		t.Fatalf("Init() error: %v", err)
	}
	p.apiClient = client.New(server.URL, "/tmp/test")

	startMsg := p.Start()()
	sessionMsg, ok := startMsg.(SessionCreatedMsg)
	if !ok {
		t.Fatalf("Start cmd returned %T, want SessionCreatedMsg", startMsg)
	}

	_, sseCmd := p.Update(sessionMsg)
	if sseCmd == nil {
		t.Fatal("expected SSE listener cmd after SessionCreatedMsg")
	}

	_, sendFirstCmd := p.Update(SendPromptMsg{Content: "first prompt"})
	if sendFirstCmd == nil {
		t.Fatal("expected send cmd for first prompt")
	}
	if msg := sendFirstCmd(); msg != nil {
		t.Fatalf("first send cmd returned %T, want nil", msg)
	}

	firstTokenMsg := sseCmd()
	firstToken, ok := firstTokenMsg.(StreamTokenMsg)
	if !ok {
		t.Fatalf("first SSE msg = %T, want StreamTokenMsg", firstTokenMsg)
	}
	if firstToken.Token != "first reply" {
		t.Fatalf("first token = %q, want %q", firstToken.Token, "first reply")
	}

	_, nextCmd := p.Update(firstToken)
	firstCompleteMsg := nextCmd()
	if _, ok := firstCompleteMsg.(StreamCompleteMsg); !ok {
		t.Fatalf("second SSE msg = %T, want StreamCompleteMsg", firstCompleteMsg)
	}
	p.Update(firstCompleteMsg)

	sseCmd = listenForSSE(p.sseChan, p.epoch)
	if sseCmd == nil {
		t.Fatal("expected SSE listener cmd for second turn")
	}

	_, sendSecondCmd := p.Update(SendPromptMsg{Content: "follow up"})
	if sendSecondCmd == nil {
		t.Fatal("expected send cmd for second prompt")
	}
	if msg := sendSecondCmd(); msg != nil {
		t.Fatalf("second send cmd returned %T, want nil", msg)
	}

	secondTokenMsg := sseCmd()
	secondToken, ok := secondTokenMsg.(StreamTokenMsg)
	if !ok {
		t.Fatalf("third SSE msg = %T, want StreamTokenMsg", secondTokenMsg)
	}
	if secondToken.Token != "second reply" {
		t.Fatalf("second token = %q, want %q", secondToken.Token, "second reply")
	}

	_, nextCmd = p.Update(secondToken)
	secondCompleteMsg := nextCmd()
	if _, ok := secondCompleteMsg.(StreamCompleteMsg); !ok {
		t.Fatalf("fourth SSE msg = %T, want StreamCompleteMsg", secondCompleteMsg)
	}
	p.Update(secondCompleteMsg)

	if p.streaming {
		t.Fatal("streaming should be false after second completion")
	}
	if p.input == nil || p.input.IsSubmitting() {
		t.Fatal("input should not be submitting after second completion")
	}
	if len(p.messages) != 4 {
		t.Fatalf("messages len = %d, want 4", len(p.messages))
	}

	if p.messages[0].Role != "user" || p.messages[0].Content != "first prompt" {
		t.Fatalf("unexpected first message: %+v", p.messages[0])
	}
	if p.messages[1].Role != "assistant" || p.messages[1].Content != "first reply" {
		t.Fatalf("unexpected second message: %+v", p.messages[1])
	}
	if p.messages[2].Role != "user" || p.messages[2].Content != "follow up" {
		t.Fatalf("unexpected third message: %+v", p.messages[2])
	}
	if p.messages[3].Role != "assistant" || p.messages[3].Content != "second reply" {
		t.Fatalf("unexpected fourth message: %+v", p.messages[3])
	}
}

func TestIntegrationStreamDrop(t *testing.T) {
	server := newMockOpenCode(t, mockOpenCodeOptions{
		sseBlocks: []string{
			"event: message.part.delta\ndata: {\"content\":\"partial\"}\nid: 1",
		},
	})
	defer server.Close()

	p := New()
	if err := p.Init(makeCtx(1)); err != nil {
		t.Fatalf("Init() error: %v", err)
	}
	p.apiClient = client.New(server.URL, "/tmp/test")

	startMsg := p.Start()()
	sessionMsg, ok := startMsg.(SessionCreatedMsg)
	if !ok {
		t.Fatalf("Start cmd returned %T, want SessionCreatedMsg", startMsg)
	}

	_, sseCmd := p.Update(sessionMsg)
	if sseCmd == nil {
		t.Fatal("expected SSE listener cmd")
	}

	_, sendCmd := p.Update(SendPromptMsg{Content: "please stream"})
	if sendCmd == nil {
		t.Fatal("expected send cmd")
	}
	if msg := sendCmd(); msg != nil {
		t.Fatalf("send cmd returned %T, want nil", msg)
	}

	tokenMsg := sseCmd()
	token, ok := tokenMsg.(StreamTokenMsg)
	if !ok {
		t.Fatalf("first SSE msg = %T, want StreamTokenMsg", tokenMsg)
	}
	if token.Token != "partial" {
		t.Fatalf("token = %q, want %q", token.Token, "partial")
	}

	_, nextCmd := p.Update(token)
	if nextCmd == nil {
		t.Fatal("expected follow-up SSE cmd")
	}

	endMsg := nextCmd()
	if _, ok := endMsg.(StreamCompleteMsg); !ok {
		if _, ok := endMsg.(StreamErrorMsg); !ok {
			t.Fatalf("end SSE msg = %T, want StreamCompleteMsg or StreamErrorMsg", endMsg)
		}
	}
	p.Update(endMsg)

	if p.streaming {
		t.Fatal("streaming should be false after stream drop")
	}
	if p.input == nil || p.input.IsSubmitting() {
		t.Fatal("input should not be submitting after stream drop")
	}
	if len(p.messages) != 2 {
		t.Fatalf("messages len = %d, want 2", len(p.messages))
	}
	if p.messages[1].Role != "assistant" {
		t.Fatalf("assistant message role = %q, want assistant", p.messages[1].Role)
	}
	if p.messages[1].Content == "" {
		t.Fatal("assistant message should preserve partial content")
	}
	if !strings.Contains(p.messages[1].Content, "partial") {
		t.Fatalf("assistant message should contain partial content, got %q", p.messages[1].Content)
	}
}

func TestIntegrationEmptyPromptBlocked(t *testing.T) {
	in := NewInput()
	in.Focus()

	_, cmd := in.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		if _, ok := cmd().(SendPromptMsg); ok {
			t.Fatal("empty input should not emit SendPromptMsg")
		}
	}

	for _, ch := range "   " {
		in.textarea.InsertRune(ch)
	}

	_, cmd = in.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		if _, ok := cmd().(SendPromptMsg); ok {
			t.Fatal("whitespace-only input should not emit SendPromptMsg")
		}
	}
}

func TestIntegrationSendWhileStreaming(t *testing.T) {
	server := newMockOpenCode(t, mockOpenCodeOptions{
		sseBlocks: []string{
			"event: message.part.delta\ndata: {\"content\":\"partial\"}\nid: 1",
		},
		keepOpen: true,
	})
	defer server.Close()

	p := New()
	if err := p.Init(makeCtx(1)); err != nil {
		t.Fatalf("Init() error: %v", err)
	}
	p.apiClient = client.New(server.URL, "/tmp/test")
	defer p.Stop()

	startMsg := p.Start()()
	sessionMsg, ok := startMsg.(SessionCreatedMsg)
	if !ok {
		t.Fatalf("Start cmd returned %T, want SessionCreatedMsg", startMsg)
	}

	_, sseCmd := p.Update(sessionMsg)
	if sseCmd == nil {
		t.Fatal("expected SSE listener cmd")
	}

	_, sendFirstCmd := p.Update(SendPromptMsg{Content: "first"})
	if sendFirstCmd == nil {
		t.Fatal("expected send cmd for first prompt")
	}
	if msg := sendFirstCmd(); msg != nil {
		t.Fatalf("first send cmd returned %T, want nil", msg)
	}

	if !p.streaming {
		t.Fatal("plugin should be streaming after first prompt")
	}
	if p.input == nil || !p.input.IsSubmitting() {
		t.Fatal("input should be submitting while streaming")
	}

	_, sendSecondCmd := p.Update(SendPromptMsg{Content: "second"})
	if sendSecondCmd == nil {
		t.Fatal("expected send cmd for second prompt while streaming")
	}
	if msg := sendSecondCmd(); msg != nil {
		t.Fatalf("second send cmd returned %T, want nil", msg)
	}

	if len(p.messages) != 2 {
		t.Fatalf("messages len = %d, want 2", len(p.messages))
	}
	if p.messages[0].Role != "user" || p.messages[0].Content != "first" {
		t.Fatalf("unexpected first message: %+v", p.messages[0])
	}
	if p.messages[1].Role != "user" || p.messages[1].Content != "second" {
		t.Fatalf("unexpected second message: %+v", p.messages[1])
	}
}

func newMultiTurnMockOpenCode(t *testing.T, turns [][]string) *httptest.Server {
	t.Helper()

	messageTurn := make(chan int, len(turns))
	messageCalls := 0

	mux := http.NewServeMux()
	mux.HandleFunc("/session", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			if err := json.NewEncoder(w).Encode(client.SessionCreateResponse{ID: "ses_test", CreatedAt: 1}); err != nil {
				t.Fatalf("encode session create response: %v", err)
			}
		case http.MethodGet:
			if err := json.NewEncoder(w).Encode([]client.SessionListItem{}); err != nil {
				t.Fatalf("encode session list response: %v", err)
			}
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/session/ses_test/message", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		messageCalls++
		if messageCalls > len(turns) {
			t.Fatalf("received %d message calls, want <= %d", messageCalls, len(turns))
		}

		messageTurn <- messageCalls
		w.WriteHeader(http.StatusNoContent)
	})

	mux.HandleFunc("/event", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher, ok := w.(http.Flusher)
		if !ok {
			t.Fatal("response writer does not implement http.Flusher")
		}

		if _, err := fmt.Fprint(w, ": connected\n\n"); err != nil {
			return
		}
		flusher.Flush()

		for i := 0; i < len(turns); i++ {
			var turn int
			select {
			case turn = <-messageTurn:
			case <-r.Context().Done():
				return
			}

			for _, block := range turns[turn-1] {
				if _, err := fmt.Fprintf(w, "%s\n\n", block); err != nil {
					return
				}
				flusher.Flush()
			}
		}
	})

	mux.HandleFunc("/session/ses_test/abort", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if err := json.NewEncoder(w).Encode(client.AbortResponse{Success: true}); err != nil {
			t.Fatalf("encode abort response: %v", err)
		}
	})

	return httptest.NewServer(mux)
}
