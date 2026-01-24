package app

import (
	"reflect"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/marcus/sidecar/internal/community"
)

func TestCommunityBrowserFilterAndView(t *testing.T) {
	var m Model
	m.width = 80
	m.height = 24
	m.themeSwitcherOriginal = "default"
	m.initCommunityBrowser()

	if !m.showCommunityBrowser {
		t.Fatal("expected community browser to be visible after init")
	}
	if len(m.communityBrowserFiltered) == 0 {
		t.Fatal("expected community schemes to be available")
	}

	query := "no-such-theme-xyz123"
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(query)}
	updated, _ := m.handleCommunityBrowserKey(msg)
	m = updated.(Model)

	if m.communityBrowserInput.Value() != query {
		t.Errorf("communityBrowserInput = %q, want %q", m.communityBrowserInput.Value(), query)
	}

	expected := filterCommunitySchemes(community.ListSchemes(), query)
	if !reflect.DeepEqual(m.communityBrowserFiltered, expected) {
		t.Errorf("communityBrowserFiltered mismatch: got %d items, want %d", len(m.communityBrowserFiltered), len(expected))
	}

	rendered := ansi.Strip(m.renderCommunityBrowserOverlay(""))
	if !strings.Contains(rendered, "Community Themes") {
		t.Errorf("expected community browser title in view")
	}
	if len(expected) == 0 && !strings.Contains(rendered, "No matches") {
		t.Errorf("expected empty-state text in view when no matches")
	}
}

func TestCommunityBrowserCursorMovement(t *testing.T) {
	var m Model
	m.width = 80
	m.height = 24
	m.themeSwitcherOriginal = "default"
	m.initCommunityBrowser()

	if len(m.communityBrowserFiltered) < 2 {
		t.Skip("not enough schemes to test cursor movement")
	}

	start := m.communityBrowserCursor
	updated, _ := m.handleCommunityBrowserKey(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(Model)
	if m.communityBrowserCursor != start+1 {
		t.Errorf("cursor after down = %d, want %d", m.communityBrowserCursor, start+1)
	}

	updated, _ = m.handleCommunityBrowserKey(tea.KeyMsg{Type: tea.KeyUp})
	m = updated.(Model)
	if m.communityBrowserCursor != start {
		t.Errorf("cursor after up = %d, want %d", m.communityBrowserCursor, start)
	}

	updated, _ = m.handleCommunityBrowserKey(tea.KeyMsg{Type: tea.KeyUp})
	m = updated.(Model)
	if m.communityBrowserCursor != 0 {
		t.Errorf("cursor after clamp up = %d, want 0", m.communityBrowserCursor)
	}
}
