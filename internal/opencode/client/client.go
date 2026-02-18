package client

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
	projectDir string
}

func New(baseURL string, projectDir string) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		projectDir: projectDir,
	}
}

func (c *Client) CreateSession(ctx context.Context) (*SessionCreateResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url("/session"), nil)
	if err != nil {
		return nil, fmt.Errorf("opencode: create session: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("opencode: create session: request failed (OpenCode may not be running): %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, c.parseErrorResponse("create session", resp)
	}

	var out SessionCreateResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("opencode: create session: decode response: %w", err)
	}

	return &out, nil
}

func (c *Client) SendMessage(ctx context.Context, sessionID string, content string) error {
	reqBody := MessageSendRequest{Content: content}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("opencode: send message: encode request: %w", err)
	}

	path := fmt.Sprintf("/session/%s/message", url.PathEscape(sessionID))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url(path), bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("opencode: send message: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("opencode: send message: request failed (OpenCode may not be running): %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return c.parseErrorResponse("send message", resp)
	}

	return nil
}

func (c *Client) StreamEvents(ctx context.Context) (<-chan SSEEvent, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.url("/event"), nil)
	if err != nil {
		return nil, fmt.Errorf("opencode: stream events: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("opencode: stream events: request failed (OpenCode may not be running): %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		defer resp.Body.Close()
		return nil, c.parseErrorResponse("stream events", resp)
	}

	ch := make(chan SSEEvent)

	go func() {
		defer close(ch)
		defer resp.Body.Close()

		scanner := bufio.NewScanner(resp.Body)
		buf := make([]byte, 0, 64*1024)
		scanner.Buffer(buf, 1024*1024)

		current := SSEEvent{}

		emit := func(ev SSEEvent) bool {
			if ev.Event == "" && ev.Data == "" && ev.ID == "" {
				return true
			}

			select {
			case <-ctx.Done():
				return false
			case ch <- ev:
				return true
			}
		}

		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			if !scanner.Scan() {
				break
			}

			line := scanner.Text()
			if line == "" {
				if !emit(current) {
					return
				}
				current = SSEEvent{}
				continue
			}

			switch {
			case strings.HasPrefix(line, "event: "):
				current.Event = strings.TrimPrefix(line, "event: ")
			case strings.HasPrefix(line, "data: "):
				current.Data += strings.TrimPrefix(line, "data: ")
			case strings.HasPrefix(line, "id: "):
				current.ID = strings.TrimPrefix(line, "id: ")
			}
		}

		_ = scanner.Err()
		_ = emit(current)
	}()

	return ch, nil
}

func (c *Client) AbortGeneration(ctx context.Context, sessionID string) error {
	path := fmt.Sprintf("/session/%s/abort", url.PathEscape(sessionID))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url(path), nil)
	if err != nil {
		return fmt.Errorf("opencode: abort generation: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("opencode: abort generation: request failed (OpenCode may not be running): %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return c.parseErrorResponse("abort generation", resp)
	}

	var out AbortResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil && err != io.EOF {
		return fmt.Errorf("opencode: abort generation: decode response: %w", err)
	}

	return nil
}

func (c *Client) ListSessions(ctx context.Context) ([]SessionListItem, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.url("/session"), nil)
	if err != nil {
		return nil, fmt.Errorf("opencode: list sessions: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("opencode: list sessions: request failed (OpenCode may not be running): %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, c.parseErrorResponse("list sessions", resp)
	}

	var out SessionListResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("opencode: list sessions: decode response: %w", err)
	}

	return out, nil
}

func (c *Client) Ping(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.url("/session"), nil)
	if err != nil {
		return fmt.Errorf("opencode: ping: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("opencode: server unreachable at %s: %w", c.baseURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return c.parseErrorResponse("ping", resp)
	}

	return nil
}

func (c *Client) parseErrorResponse(operation string, resp *http.Response) error {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("opencode: %s: status %d: read error body: %w", operation, resp.StatusCode, err)
	}

	var apiErr ErrorResponse
	if err := json.Unmarshal(body, &apiErr); err == nil && apiErr.Error != "" {
		if apiErr.Code != 0 {
			return fmt.Errorf("opencode: %s: status %d: %s (code=%d)", operation, resp.StatusCode, apiErr.Error, apiErr.Code)
		}
		return fmt.Errorf("opencode: %s: status %d: %s", operation, resp.StatusCode, apiErr.Error)
	}

	msg := strings.TrimSpace(string(body))
	if msg == "" {
		return fmt.Errorf("opencode: %s: status %d", operation, resp.StatusCode)
	}

	return fmt.Errorf("opencode: %s: status %d: %s", operation, resp.StatusCode, msg)
}

func (c *Client) url(path string) string {
	return fmt.Sprintf("%s%s?directory=%s", c.baseURL, path, url.QueryEscape(c.projectDir))
}
