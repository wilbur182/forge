package chat

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/wilbur182/forge/internal/opencode/client"
	"github.com/wilbur182/forge/internal/plugin"
)

var _ plugin.Plugin = (*Plugin)(nil)
var _ plugin.TextInputConsumer = (*Plugin)(nil)

func makeCtx(epoch uint64) *plugin.Context {
	return &plugin.Context{Epoch: epoch, ProjectRoot: "/tmp/test"}
}

func TestNew_ReturnsNonNil(t *testing.T) {
	p := New()
	if p == nil {
		t.Fatal("New() returned nil")
	}
}

func TestID(t *testing.T) {
	if got := New().ID(); got != "chat" {
		t.Fatalf("ID() = %q, want %q", got, "chat")
	}
}

func TestName(t *testing.T) {
	if got := New().Name(); got != "Chat" {
		t.Fatalf("Name() = %q, want %q", got, "Chat")
	}
}

func TestIcon(t *testing.T) {
	if got := New().Icon(); got != "⚡" {
		t.Fatalf("Icon() = %q, want %q", got, "⚡")
	}
}

func TestInit_StoresContextAndInitializesComponents(t *testing.T) {
	p := New()
	ctx := makeCtx(1)
	if err := p.Init(ctx); err != nil {
		t.Fatalf("Init() error: %v", err)
	}
	if p.ctx != ctx {
		t.Fatal("Init() did not store context")
	}
	if p.apiClient == nil {
		t.Fatal("Init() did not initialize api client")
	}
	if p.input == nil {
		t.Fatal("Init() did not initialize input")
	}
	if !p.input.IsFocused() {
		t.Fatal("Init() should focus input")
	}
}

func TestSetFocused_IsFocused(t *testing.T) {
	p := New()
	p.SetFocused(true)
	if !p.IsFocused() {
		t.Fatal("IsFocused() = false after SetFocused(true)")
	}
	p.SetFocused(false)
	if p.IsFocused() {
		t.Fatal("IsFocused() = true after SetFocused(false)")
	}
}

func TestCommands_Returns3(t *testing.T) {
	cmds := New().Commands()
	if len(cmds) != 3 {
		t.Fatalf("Commands() len = %d, want 3", len(cmds))
	}
	wantIDs := []string{"send", "abort", "new-session"}
	for i, want := range wantIDs {
		if cmds[i].ID != want {
			t.Errorf("Commands()[%d].ID = %q, want %q", i, cmds[i].ID, want)
		}
	}
}

func TestFocusContext(t *testing.T) {
	if got := New().FocusContext(); got != "chat" {
		t.Fatalf("FocusContext() = %q, want %q", got, "chat")
	}
}

func TestView_NonEmptyAndHeightConstrained(t *testing.T) {
	p := New()
	_ = p.Init(makeCtx(1))
	const width, height = 80, 10
	out := p.View(width, height)
	if out == "" {
		t.Fatal("View() returned empty string")
	}
	lines := strings.Count(out, "\n") + 1
	if lines > height {
		t.Fatalf("View() produced %d lines, want <= %d", lines, height)
	}
}

func TestUpdate_UnknownMsg_ReturnsSamePointer(t *testing.T) {
	p := New()
	_ = p.Init(makeCtx(1))
	type unknownMsg struct{}
	got, _ := p.Update(unknownMsg{})
	if got != p {
		t.Fatal("Update(unknown) should return same plugin pointer")
	}
}

func TestFullFlow(t *testing.T) {
	server := newMockOpenCode(t, mockOpenCodeOptions{
		sseBlocks: []string{
			"event: message.part.delta\ndata: {\"content\":\"hello\"}\nid: 1",
			"event: message.part.delta\ndata: {\"content\":\" world\"}\nid: 2",
			"event: message.complete\ndata: {\"sessionID\":\"ses_test\",\"messageID\":\"msg_1\"}\nid: 3",
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
	if sessionMsg.SessionID != "ses_test" {
		t.Fatalf("session id = %q, want %q", sessionMsg.SessionID, "ses_test")
	}

	_, sseCmd := p.Update(sessionMsg)
	if p.sessionID != "ses_test" {
		t.Fatalf("plugin session id = %q, want %q", p.sessionID, "ses_test")
	}
	if sseCmd == nil {
		t.Fatal("expected SSE listener cmd after SessionCreatedMsg")
	}

	_, sendCmd := p.Update(SendPromptMsg{Content: "test prompt"})
	if sendCmd == nil {
		t.Fatal("expected send cmd after SendPromptMsg")
	}
	if msg := sendCmd(); msg != nil {
		t.Fatalf("send cmd returned %T, want nil", msg)
	}

	token1Msg := sseCmd()
	token1, ok := token1Msg.(StreamTokenMsg)
	if !ok {
		t.Fatalf("first SSE msg = %T, want StreamTokenMsg", token1Msg)
	}
	if token1.Token != "hello" {
		t.Fatalf("first token = %q, want %q", token1.Token, "hello")
	}

	_, nextCmd := p.Update(token1)
	token2Msg := nextCmd()
	token2, ok := token2Msg.(StreamTokenMsg)
	if !ok {
		t.Fatalf("second SSE msg = %T, want StreamTokenMsg", token2Msg)
	}
	if token2.Token != " world" {
		t.Fatalf("second token = %q, want %q", token2.Token, " world")
	}

	_, nextCmd = p.Update(token2)
	completeMsg := nextCmd()
	if _, ok := completeMsg.(StreamCompleteMsg); !ok {
		t.Fatalf("third SSE msg = %T, want StreamCompleteMsg", completeMsg)
	}
	p.Update(completeMsg)

	if p.streaming {
		t.Fatal("streaming should be false after completion")
	}
	if p.input == nil || p.input.IsSubmitting() {
		t.Fatal("input should not be submitting after completion")
	}
	if len(p.messages) != 2 {
		t.Fatalf("messages len = %d, want 2", len(p.messages))
	}
	if p.messages[0].Role != "user" || p.messages[0].Content != "test prompt" {
		t.Fatalf("unexpected user message: %+v", p.messages[0])
	}
	if p.messages[1].Role != "assistant" || p.messages[1].Content != "hello world" {
		t.Fatalf("unexpected assistant message: %+v", p.messages[1])
	}
}

func TestConnectionError(t *testing.T) {
	p := New()
	if err := p.Init(makeCtx(1)); err != nil {
		t.Fatalf("Init() error: %v", err)
	}
	p.apiClient = client.New("http://127.0.0.1:1", "/tmp/test")

	msg := p.Start()()
	if _, ok := msg.(ConnectionErrorMsg); !ok {
		t.Fatalf("Start cmd returned %T, want ConnectionErrorMsg", msg)
	}
}

func TestAbortFlow(t *testing.T) {
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

	_, sendCmd := p.Update(SendPromptMsg{Content: "please stream"})
	if msg := sendCmd(); msg != nil {
		t.Fatalf("send cmd returned %T, want nil", msg)
	}

	tokenMsg := sseCmd()
	token, ok := tokenMsg.(StreamTokenMsg)
	if !ok {
		t.Fatalf("SSE msg = %T, want StreamTokenMsg", tokenMsg)
	}
	p.Update(token)

	_, abortCmd := p.Update(AbortMsg{})
	if abortCmd == nil {
		t.Fatal("expected abort cmd")
	}
	abortMsg := abortCmd()
	if _, ok := abortMsg.(AbortCompleteMsg); !ok {
		t.Fatalf("abort cmd returned %T, want AbortCompleteMsg", abortMsg)
	}
	p.Update(abortMsg)

	if p.streaming {
		t.Fatal("streaming should be false after abort")
	}
	if p.input == nil || p.input.IsSubmitting() {
		t.Fatal("input should not be submitting after abort")
	}
	if len(p.messages) != 2 {
		t.Fatalf("messages len = %d, want 2", len(p.messages))
	}
	if !strings.Contains(p.messages[1].Content, "partial") {
		t.Fatalf("assistant message missing partial content: %q", p.messages[1].Content)
	}
	if !strings.Contains(p.messages[1].Content, "[Aborted]") {
		t.Fatalf("assistant message missing abort marker: %q", p.messages[1].Content)
	}
}

func TestStaleRejection(t *testing.T) {
	p := New()
	_ = p.Init(makeCtx(1))
	p.streamBuffer = "ok"
	p.Update(StreamTokenMsg{Token: " stale", Epoch: 0})
	if p.streamBuffer != "ok" {
		t.Fatalf("stale token should be ignored, got %q", p.streamBuffer)
	}
}

func TestStop_ClearsState(t *testing.T) {
	p := New()
	_ = p.Init(makeCtx(1))
	p.messages = []ChatMessage{{Role: "user", Content: "hi"}}
	p.sessionID = "sess-1"
	p.streaming = true
	p.streamBuffer = "partial"
	p.Stop()
	if len(p.messages) != 0 {
		t.Error("Stop() should clear messages")
	}
	if p.sessionID != "" {
		t.Error("Stop() should clear sessionID")
	}
	if p.streaming {
		t.Error("Stop() should clear streaming flag")
	}
	if p.streamBuffer != "" {
		t.Error("Stop() should clear stream buffer")
	}
}

func TestConsumesTextInput_TracksInputFocus(t *testing.T) {
	p := New()
	_ = p.Init(makeCtx(1))
	if !p.ConsumesTextInput() {
		t.Fatal("ConsumesTextInput() should be true when input is focused")
	}
	p.input.Blur()
	if p.ConsumesTextInput() {
		t.Fatal("ConsumesTextInput() should be false when input is blurred")
	}
}

func TestUpdate_WindowSizeMsg_StoresDimensionsAndResizesViewport(t *testing.T) {
	p := New()
	_ = p.Init(makeCtx(1))
	p.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	if p.width != 120 || p.height != 40 {
		t.Fatalf("dimensions not stored: width=%d height=%d", p.width, p.height)
	}
	if p.msgViewport.width != 120 {
		t.Fatalf("viewport width = %d, want 120", p.msgViewport.width)
	}
	if p.msgViewport.height != 36 {
		t.Fatalf("viewport height = %d, want 36", p.msgViewport.height)
	}
}

type mockOpenCodeOptions struct {
	sseBlocks []string
	keepOpen  bool
}

func newMockOpenCode(t *testing.T, opts mockOpenCodeOptions) *httptest.Server {
	t.Helper()

	var sendOnce sync.Once
	messageSent := make(chan struct{})

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
		w.WriteHeader(http.StatusNoContent)
		sendOnce.Do(func() {
			close(messageSent)
		})
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

		select {
		case <-messageSent:
		case <-r.Context().Done():
			return
		}

		for _, block := range opts.sseBlocks {
			if _, err := fmt.Fprintf(w, "%s\n\n", block); err != nil {
				return
			}
			flusher.Flush()
		}

		if opts.keepOpen {
			<-r.Context().Done()
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
