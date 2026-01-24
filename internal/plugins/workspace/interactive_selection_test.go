package workspace

import (
	"strings"
	"testing"

	"github.com/marcus/sidecar/internal/mouse"
)

// newSelectionTestPlugin creates a Plugin with interactive state configured for selection tests.
// The pane starts at Y=2, has 1 row of content offset (tab bar), and shows lines 0-9.
func newSelectionTestPlugin() *Plugin {
	p := &Plugin{
		viewMode:     ViewModeInteractive,
		mouseHandler: mouse.NewHandler(),
		interactiveState: &InteractiveState{
			Active:           true,
			VisibleStart:     0,
			VisibleEnd:       10,
			ContentRowOffset: 1,
		},
		interactiveSelStart:  selectionPoint{-1, -1},
		interactiveSelEnd:    selectionPoint{-1, -1},
		interactiveSelAnchor: selectionPoint{-1, -1},
	}
	return p
}

// actionAt creates a mouse action at the given content column with the preview pane region.
// The x parameter is a content column; panelOverhead/2 is added to simulate the viewport
// X coordinate (accounting for left border + left padding).
func actionAt(x, y int) mouse.MouseAction {
	return mouse.MouseAction{
		Type: mouse.ActionClick,
		X:    x + panelOverhead/2,
		Y:    y,
		Region: &mouse.Region{
			ID:   regionPreviewPane,
			Rect: mouse.Rect{X: 0, Y: 2, W: 80, H: 12},
		},
	}
}

func TestPrepareInteractiveDrag_NoSelection(t *testing.T) {
	p := newSelectionTestPlugin()

	// Y=6: contentRow = 6-2-1 = 3, outputRow = 3-1 = 2, lineIdx = 0+2 = 2
	action := actionAt(10, 6)
	p.prepareInteractiveDrag(action)

	if p.hasInteractiveSelection() {
		t.Error("click without drag should not create selection")
	}
	if p.interactiveSelStart.valid() {
		t.Errorf("start should be invalid, got %+v", p.interactiveSelStart)
	}
	if p.interactiveSelEnd.valid() {
		t.Errorf("end should be invalid, got %+v", p.interactiveSelEnd)
	}
	if p.interactiveSelAnchor.line != 2 {
		t.Errorf("anchor line should be 2, got %d", p.interactiveSelAnchor.line)
	}
}

func TestDragAfterClick_CreatesSelection(t *testing.T) {
	p := newSelectionTestPlugin()

	// Click at line 2 (Y=6)
	action := actionAt(10, 6)
	p.prepareInteractiveDrag(action)

	// Drag to line 4 (Y=8)
	dragAction := mouse.MouseAction{
		Type: mouse.ActionDrag,
		X:    10,
		Y:    8,
		Region: &mouse.Region{
			ID:   regionPreviewPane,
			Rect: mouse.Rect{X: 0, Y: 2, W: 80, H: 12},
		},
	}
	p.handleInteractiveSelectionDrag(dragAction)

	if !p.hasInteractiveSelection() {
		t.Error("drag should create selection")
	}
	if !p.interactiveSelectionActive {
		t.Error("selection should be active after drag")
	}
	if p.interactiveSelStart.line != 2 {
		t.Errorf("start line should be 2, got %d", p.interactiveSelStart.line)
	}
	if p.interactiveSelEnd.line != 4 {
		t.Errorf("end line should be 4, got %d", p.interactiveSelEnd.line)
	}
}

func TestDragUpward_FromAnchor(t *testing.T) {
	p := newSelectionTestPlugin()

	// Click at line 4 (Y=8)
	action := actionAt(10, 8)
	p.prepareInteractiveDrag(action)

	// Drag up to line 1 (Y=5)
	dragAction := mouse.MouseAction{
		Type: mouse.ActionDrag,
		X:    10,
		Y:    5,
		Region: &mouse.Region{
			ID:   regionPreviewPane,
			Rect: mouse.Rect{X: 0, Y: 2, W: 80, H: 12},
		},
	}
	p.handleInteractiveSelectionDrag(dragAction)

	if !p.hasInteractiveSelection() {
		t.Error("upward drag should create selection")
	}
	if p.interactiveSelStart.line != 1 {
		t.Errorf("start line should be 1, got %d", p.interactiveSelStart.line)
	}
	if p.interactiveSelEnd.line != 4 {
		t.Errorf("end line should be 4, got %d", p.interactiveSelEnd.line)
	}
}

func TestFinishInteractiveSelection_UnstartedClears(t *testing.T) {
	p := newSelectionTestPlugin()

	// Click without drag
	action := actionAt(10, 6)
	p.prepareInteractiveDrag(action)

	// Finish without any drag motion
	p.finishInteractiveSelection()

	if p.hasInteractiveSelection() {
		t.Error("finish without drag should not leave selection")
	}
	if p.interactiveSelStart.valid() {
		t.Errorf("start should be invalid after clear, got %+v", p.interactiveSelStart)
	}
	if p.interactiveSelAnchor.valid() {
		t.Errorf("anchor should be invalid after clear, got %+v", p.interactiveSelAnchor)
	}
}

func TestFinishInteractiveSelection_AfterDrag(t *testing.T) {
	p := newSelectionTestPlugin()

	// Click and drag
	action := actionAt(10, 6)
	p.prepareInteractiveDrag(action)
	dragAction := mouse.MouseAction{
		Type: mouse.ActionDrag,
		X:    10,
		Y:    8,
		Region: &mouse.Region{
			ID:   regionPreviewPane,
			Rect: mouse.Rect{X: 0, Y: 2, W: 80, H: 12},
		},
	}
	p.handleInteractiveSelectionDrag(dragAction)

	// Finish
	p.finishInteractiveSelection()

	// Selection should persist (active=false but range preserved)
	if !p.hasInteractiveSelection() {
		t.Error("selection range should persist after finish")
	}
	if p.interactiveSelectionActive {
		t.Error("active should be false after finish")
	}
	if p.interactiveSelStart.line != 2 {
		t.Errorf("start line should be 2, got %d", p.interactiveSelStart.line)
	}
	if p.interactiveSelEnd.line != 4 {
		t.Errorf("end line should be 4, got %d", p.interactiveSelEnd.line)
	}
}

func TestClearInteractiveSelection_ResetsSentinels(t *testing.T) {
	p := newSelectionTestPlugin()

	// Create a valid selection
	action := actionAt(10, 6)
	p.prepareInteractiveDrag(action)
	dragAction := mouse.MouseAction{
		Type: mouse.ActionDrag,
		X:    10,
		Y:    8,
		Region: &mouse.Region{
			ID:   regionPreviewPane,
			Rect: mouse.Rect{X: 0, Y: 2, W: 80, H: 12},
		},
	}
	p.handleInteractiveSelectionDrag(dragAction)

	// Clear
	p.clearInteractiveSelection()

	if p.interactiveSelectionActive {
		t.Error("active should be false after clear")
	}
	if p.interactiveSelStart.valid() {
		t.Errorf("start should be invalid, got %+v", p.interactiveSelStart)
	}
	if p.interactiveSelEnd.valid() {
		t.Errorf("end should be invalid, got %+v", p.interactiveSelEnd)
	}
	if p.interactiveSelAnchor.valid() {
		t.Errorf("anchor should be invalid, got %+v", p.interactiveSelAnchor)
	}
	if p.hasInteractiveSelection() {
		t.Error("hasInteractiveSelection should return false after clear")
	}
}

func TestDragToSameLine_SelectsSingleLine(t *testing.T) {
	p := newSelectionTestPlugin()
	buf := NewOutputBuffer(100)
	buf.Write("line0\nline1\nline2\nline three has enough text to test\nline4")
	p.shellSelected = true
	p.shells = []*ShellSession{{
		Agent: &Agent{OutputBuf: buf},
	}}
	p.selectedShellIdx = 0

	// Click at line 3 (Y=7: contentRow=7-2-1=4, outputRow=4-1=3, lineIdx=3)
	action := actionAt(10, 7)
	p.prepareInteractiveDrag(action)

	// Drag to same line (different X, same Y)
	dragAction := mouse.MouseAction{
		Type: mouse.ActionDrag,
		X:    30,
		Y:    7,
		Region: &mouse.Region{
			ID:   regionPreviewPane,
			Rect: mouse.Rect{X: 0, Y: 2, W: 80, H: 12},
		},
	}
	p.handleInteractiveSelectionDrag(dragAction)

	if !p.hasInteractiveSelection() {
		t.Error("drag to same line should create selection")
	}
	if p.interactiveSelStart.line != 3 {
		t.Errorf("start line should be 3, got %d", p.interactiveSelStart.line)
	}
	if p.interactiveSelEnd.line != 3 {
		t.Errorf("end line should be 3, got %d", p.interactiveSelEnd.line)
	}
	// Column should differ between start and end
	if p.interactiveSelStart.col == p.interactiveSelEnd.col {
		t.Error("start and end col should differ for same-line drag with different X")
	}
}

func TestPrepareInteractiveDrag_InvalidY(t *testing.T) {
	p := newSelectionTestPlugin()

	// Click above content area (Y=2 → border row)
	action := actionAt(10, 2)
	p.prepareInteractiveDrag(action)

	if p.interactiveSelAnchor.valid() {
		t.Errorf("anchor should be invalid for invalid Y, got %+v", p.interactiveSelAnchor)
	}
}

func TestPrepareInteractiveDrag_NilRegion(t *testing.T) {
	p := &Plugin{
		viewMode:     ViewModeInteractive,
		mouseHandler: mouse.NewHandler(),
		interactiveState: &InteractiveState{
			Active:           true,
			VisibleStart:     0,
			VisibleEnd:       10,
			ContentRowOffset: 1,
		},
		interactiveSelStart:  selectionPoint{-1, -1},
		interactiveSelEnd:    selectionPoint{-1, -1},
		interactiveSelAnchor: selectionPoint{-1, -1},
	}

	action := mouse.MouseAction{
		Type:   mouse.ActionClick,
		X:      10,
		Y:      6,
		Region: nil,
	}
	p.prepareInteractiveDrag(action)

	if p.interactiveSelAnchor.valid() {
		t.Errorf("anchor should remain invalid for nil region, got %+v", p.interactiveSelAnchor)
	}
}

func TestIsInteractiveLineSelected(t *testing.T) {
	p := newSelectionTestPlugin()

	// Set up selection range [3, 5]
	p.interactiveSelStart = selectionPoint{3, 0}
	p.interactiveSelEnd = selectionPoint{5, 10}

	tests := []struct {
		lineIdx  int
		expected bool
	}{
		{2, false},
		{3, true},
		{4, true},
		{5, true},
		{6, false},
	}

	for _, tt := range tests {
		got := p.isInteractiveLineSelected(tt.lineIdx)
		if got != tt.expected {
			t.Errorf("isInteractiveLineSelected(%d) = %v, want %v", tt.lineIdx, got, tt.expected)
		}
	}
}

func TestHasInteractiveSelection_Sentinels(t *testing.T) {
	p := newSelectionTestPlugin()

	// Default: sentinels
	if p.hasInteractiveSelection() {
		t.Error("should return false with sentinel values")
	}

	// Only start set
	p.interactiveSelStart = selectionPoint{3, 0}
	if p.hasInteractiveSelection() {
		t.Error("should return false with only start set")
	}

	// Both set
	p.interactiveSelEnd = selectionPoint{5, 10}
	if !p.hasInteractiveSelection() {
		t.Error("should return true with both start and end set")
	}
}

// --- selectionPoint struct tests ---

func TestSelectionPoint_Before(t *testing.T) {
	tests := []struct {
		name     string
		a, b     selectionPoint
		expected bool
	}{
		{"same line, a before b", selectionPoint{2, 5}, selectionPoint{2, 10}, true},
		{"same line, a after b", selectionPoint{2, 10}, selectionPoint{2, 5}, false},
		{"same position", selectionPoint{2, 5}, selectionPoint{2, 5}, false},
		{"different lines, a before b", selectionPoint{1, 99}, selectionPoint{2, 0}, true},
		{"different lines, a after b", selectionPoint{3, 0}, selectionPoint{2, 99}, false},
	}

	for _, tt := range tests {
		got := tt.a.before(tt.b)
		if got != tt.expected {
			t.Errorf("%s: (%+v).before(%+v) = %v, want %v", tt.name, tt.a, tt.b, got, tt.expected)
		}
	}
}

func TestSelectionPoint_Valid(t *testing.T) {
	tests := []struct {
		name     string
		p        selectionPoint
		expected bool
	}{
		{"both -1", selectionPoint{-1, -1}, false},
		{"line -1", selectionPoint{-1, 0}, false},
		{"col -1", selectionPoint{0, -1}, false},
		{"both 0", selectionPoint{0, 0}, true},
		{"positive", selectionPoint{5, 10}, true},
	}

	for _, tt := range tests {
		got := tt.p.valid()
		if got != tt.expected {
			t.Errorf("%s: %+v.valid() = %v, want %v", tt.name, tt.p, got, tt.expected)
		}
	}
}

// --- getLineSelectionCols tests ---

func TestGetLineSelectionCols(t *testing.T) {
	p := newSelectionTestPlugin()

	tests := []struct {
		name             string
		start, end       selectionPoint
		lineIdx          int
		expectStart      int
		expectEnd        int
	}{
		{
			"line before selection",
			selectionPoint{3, 5}, selectionPoint{6, 10},
			2, -1, -1,
		},
		{
			"line after selection",
			selectionPoint{3, 5}, selectionPoint{6, 10},
			7, -1, -1,
		},
		{
			"first line of multi-line",
			selectionPoint{3, 5}, selectionPoint{6, 10},
			3, 5, -1,
		},
		{
			"middle line",
			selectionPoint{3, 5}, selectionPoint{6, 10},
			4, 0, -1,
		},
		{
			"last line of multi-line",
			selectionPoint{3, 5}, selectionPoint{6, 10},
			6, 0, 10,
		},
		{
			"single-line selection",
			selectionPoint{5, 3}, selectionPoint{5, 15},
			5, 3, 15,
		},
	}

	for _, tt := range tests {
		p.interactiveSelStart = tt.start
		p.interactiveSelEnd = tt.end
		startCol, endCol := p.getLineSelectionCols(tt.lineIdx)
		if startCol != tt.expectStart || endCol != tt.expectEnd {
			t.Errorf("%s: getLineSelectionCols(%d) = (%d, %d), want (%d, %d)",
				tt.name, tt.lineIdx, startCol, endCol, tt.expectStart, tt.expectEnd)
		}
	}
}

// --- visualSubstring tests ---

func TestVisualSubstring_PlainText(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		start    int
		end      int
		expected string
	}{
		{"full word", "hello world", 6, 11, "world"},
		{"mid-word", "hello world", 2, 7, "llo w"},
		{"to end", "hello", 2, -1, "llo"},
		{"from start", "hello", 0, 3, "hel"},
		{"single char", "hello", 2, 3, "l"},
		{"empty string", "", 0, 5, ""},
		{"start beyond len", "hello", 10, 15, ""},
	}

	for _, tt := range tests {
		got := visualSubstring(tt.input, tt.start, tt.end)
		if got != tt.expected {
			t.Errorf("%s: visualSubstring(%q, %d, %d) = %q, want %q",
				tt.name, tt.input, tt.start, tt.end, got, tt.expected)
		}
	}
}

func TestVisualSubstring_WithANSI(t *testing.T) {
	// ANSI codes should be skipped in column counting and stripped from output
	input := "\x1b[31mhello\x1b[0m world"
	got := visualSubstring(input, 6, 11)
	if got != "world" {
		t.Errorf("ANSI: visualSubstring = %q, want %q", got, "world")
	}

	// Selection within colored region
	got = visualSubstring(input, 0, 5)
	if got != "hello" {
		t.Errorf("ANSI within: visualSubstring = %q, want %q", got, "hello")
	}
}

func TestVisualSubstring_MultiWidth(t *testing.T) {
	// Emoji takes 2 columns: "A" at col 0, "🎉" at cols 1-2, "B" at col 3
	input := "A🎉B"

	// Select emoji (cols 1-3 exclusive = cols 1-2 inclusive)
	got := visualSubstring(input, 1, 3)
	if got != "🎉" {
		t.Errorf("emoji: visualSubstring(%q, 1, 3) = %q, want %q", input, got, "🎉")
	}

	// Select all
	got = visualSubstring(input, 0, -1)
	if got != "A🎉B" {
		t.Errorf("all: visualSubstring(%q, 0, -1) = %q, want %q", input, got, "A🎉B")
	}
}

// --- injectCharacterRangeBackground tests ---

func TestInjectCharacterRangeBackground_FullLine(t *testing.T) {
	input := "hello world"
	// Full line (startCol=0, endCol=-1) should delegate to injectSelectionBackground
	result := injectCharacterRangeBackground(input, 0, -1)
	expected := injectSelectionBackground(input)
	if result != expected {
		t.Errorf("full line: got %q, want %q", result, expected)
	}
}

func TestInjectCharacterRangeBackground_Partial(t *testing.T) {
	input := "hello world"
	result := injectCharacterRangeBackground(input, 6, 10)

	// "world" (cols 6-10) should have background, "hello " should not
	selBg := getSelectionBgANSI()
	if !strings.Contains(result, selBg) {
		t.Error("partial: result should contain selection background ANSI")
	}
	// Should contain a background-only reset after the selection
	if !strings.Contains(result, "\x1b[49m") {
		t.Error("partial: result should contain background-only ANSI reset")
	}
}

func TestInjectCharacterRangeBackground_EmptyString(t *testing.T) {
	result := injectCharacterRangeBackground("", 0, 5)
	if result != "" {
		t.Errorf("empty: got %q, want empty", result)
	}
}

// --- interactiveColAtX tests ---

func TestInteractiveColAtX_PlainText(t *testing.T) {
	p := newSelectionTestPlugin()
	buf := NewOutputBuffer(100)
	buf.Write("hello world")

	// Set up a shell with the buffer
	p.shellSelected = true
	p.shells = []*ShellSession{{
		Agent: &Agent{OutputBuf: buf},
	}}
	p.selectedShellIdx = 0
	p.interactiveSelectionRect = mouse.Rect{X: 0, Y: 2, W: 80, H: 12}

	col, ok := p.interactiveColAtX(5+panelOverhead/2, 0)
	if !ok {
		t.Fatal("expected ok=true")
	}
	if col != 5 {
		t.Errorf("plain text: col = %d, want 5", col)
	}
}

func TestInteractiveColAtX_BeyondLineEnd(t *testing.T) {
	p := newSelectionTestPlugin()
	buf := NewOutputBuffer(100)
	buf.Write("hello")

	p.shellSelected = true
	p.shells = []*ShellSession{{
		Agent: &Agent{OutputBuf: buf},
	}}
	p.selectedShellIdx = 0
	p.interactiveSelectionRect = mouse.Rect{X: 0, Y: 2, W: 80, H: 12}

	col, ok := p.interactiveColAtX(100+panelOverhead/2, 0)
	if !ok {
		t.Fatal("expected ok=true")
	}
	// Should clamp to last char (col 4)
	if col != 4 {
		t.Errorf("beyond end: col = %d, want 4", col)
	}
}

func TestInteractiveColAtX_WithHorizOffset(t *testing.T) {
	p := newSelectionTestPlugin()
	buf := NewOutputBuffer(100)
	buf.Write("hello world test")

	p.shellSelected = true
	p.shells = []*ShellSession{{
		Agent: &Agent{OutputBuf: buf},
	}}
	p.selectedShellIdx = 0
	p.interactiveSelectionRect = mouse.Rect{X: 0, Y: 2, W: 80, H: 12}

	// viewport X=4 (content col 2) → visual col 2
	col, ok := p.interactiveColAtX(2+panelOverhead/2, 0)
	if !ok {
		t.Fatal("expected ok=true")
	}
	if col != 2 {
		t.Errorf("col = %d, want 2", col)
	}
}

func TestInteractiveColAtX_EmptyLine(t *testing.T) {
	p := newSelectionTestPlugin()
	buf := NewOutputBuffer(100)
	buf.Write("")

	p.shellSelected = true
	p.shells = []*ShellSession{{
		Agent: &Agent{OutputBuf: buf},
	}}
	p.selectedShellIdx = 0
	p.interactiveSelectionRect = mouse.Rect{X: 0, Y: 2, W: 80, H: 12}

	col, ok := p.interactiveColAtX(0+panelOverhead/2, 0)
	if !ok {
		t.Fatal("expected ok=true")
	}
	if col != 0 {
		t.Errorf("empty line: col = %d, want 0", col)
	}
}

// --- Character-level drag tests ---

func TestCharacterLevelDrag_SameLineRightward(t *testing.T) {
	p := newSelectionTestPlugin()
	buf := NewOutputBuffer(100)
	buf.Write("hello world")

	p.shellSelected = true
	p.shells = []*ShellSession{{
		Agent: &Agent{OutputBuf: buf},
	}}
	p.selectedShellIdx = 0

	// Click at col 5 on line 0 (Y=4: contentRow=4-2-1=1, outputRow=1-1=0, lineIdx=0)
	action := actionAt(5, 4)
	p.prepareInteractiveDrag(action)

	// Drag to col 10
	dragAction := mouse.MouseAction{
		Type: mouse.ActionDrag,
		X:    10 + panelOverhead/2,
		Y:    4,
		Region: &mouse.Region{
			ID:   regionPreviewPane,
			Rect: mouse.Rect{X: 0, Y: 2, W: 80, H: 12},
		},
	}
	p.handleInteractiveSelectionDrag(dragAction)

	if p.interactiveSelStart.line != 0 || p.interactiveSelStart.col != 5 {
		t.Errorf("start = %+v, want {0, 5}", p.interactiveSelStart)
	}
	if p.interactiveSelEnd.line != 0 || p.interactiveSelEnd.col != 10 {
		t.Errorf("end = %+v, want {0, 10}", p.interactiveSelEnd)
	}
}

func TestCharacterLevelDrag_SameLineBackward(t *testing.T) {
	p := newSelectionTestPlugin()
	buf := NewOutputBuffer(100)
	buf.Write("hello world")

	p.shellSelected = true
	p.shells = []*ShellSession{{
		Agent: &Agent{OutputBuf: buf},
	}}
	p.selectedShellIdx = 0

	// Click at col 10
	action := actionAt(10, 4)
	p.prepareInteractiveDrag(action)

	// Drag backward to col 3
	dragAction := mouse.MouseAction{
		Type: mouse.ActionDrag,
		X:    3 + panelOverhead/2,
		Y:    4,
		Region: &mouse.Region{
			ID:   regionPreviewPane,
			Rect: mouse.Rect{X: 0, Y: 2, W: 80, H: 12},
		},
	}
	p.handleInteractiveSelectionDrag(dragAction)

	// Start should be the lesser position
	if p.interactiveSelStart.col != 3 {
		t.Errorf("start col = %d, want 3", p.interactiveSelStart.col)
	}
	if p.interactiveSelEnd.col != 10 {
		t.Errorf("end col = %d, want 10", p.interactiveSelEnd.col)
	}
}

func TestCharacterLevelDrag_MultiLineDown(t *testing.T) {
	p := newSelectionTestPlugin()
	buf := NewOutputBuffer(100)
	buf.Write("line zero\nline one\nline two\nline three")

	p.shellSelected = true
	p.shells = []*ShellSession{{
		Agent: &Agent{OutputBuf: buf},
	}}
	p.selectedShellIdx = 0

	// Click at (5, line 1) → Y=5: contentRow=5-2-1=2, outputRow=2-1=1, lineIdx=1
	action := actionAt(5, 5)
	p.prepareInteractiveDrag(action)

	// Drag to (3, line 3) → Y=7
	dragAction := mouse.MouseAction{
		Type: mouse.ActionDrag,
		X:    3 + panelOverhead/2,
		Y:    7,
		Region: &mouse.Region{
			ID:   regionPreviewPane,
			Rect: mouse.Rect{X: 0, Y: 2, W: 80, H: 12},
		},
	}
	p.handleInteractiveSelectionDrag(dragAction)

	if p.interactiveSelStart.line != 1 || p.interactiveSelStart.col != 5 {
		t.Errorf("start = %+v, want {1, 5}", p.interactiveSelStart)
	}
	if p.interactiveSelEnd.line != 3 || p.interactiveSelEnd.col != 3 {
		t.Errorf("end = %+v, want {3, 3}", p.interactiveSelEnd)
	}
}

func TestCharacterLevelDrag_DirectionReversal(t *testing.T) {
	p := newSelectionTestPlugin()
	buf := NewOutputBuffer(100)
	buf.Write("abcdefghijklmnop")

	p.shellSelected = true
	p.shells = []*ShellSession{{
		Agent: &Agent{OutputBuf: buf},
	}}
	p.selectedShellIdx = 0

	// Click at col 8
	action := actionAt(8, 4)
	p.prepareInteractiveDrag(action)

	// Drag right to col 12
	dragAction := mouse.MouseAction{
		Type: mouse.ActionDrag,
		X:    12 + panelOverhead/2,
		Y:    4,
		Region: &mouse.Region{
			ID:   regionPreviewPane,
			Rect: mouse.Rect{X: 0, Y: 2, W: 80, H: 12},
		},
	}
	p.handleInteractiveSelectionDrag(dragAction)

	if p.interactiveSelStart.col != 8 || p.interactiveSelEnd.col != 12 {
		t.Errorf("after right drag: start.col=%d, end.col=%d, want 8, 12",
			p.interactiveSelStart.col, p.interactiveSelEnd.col)
	}

	// Now reverse past anchor to col 3
	dragAction.X = 3 + panelOverhead/2
	p.handleInteractiveSelectionDrag(dragAction)

	if p.interactiveSelStart.col != 3 || p.interactiveSelEnd.col != 8 {
		t.Errorf("after reversal: start.col=%d, end.col=%d, want 3, 8",
			p.interactiveSelStart.col, p.interactiveSelEnd.col)
	}
}

