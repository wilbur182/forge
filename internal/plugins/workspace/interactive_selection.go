package workspace

import (
	"fmt"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	app "github.com/marcus/sidecar/internal/app"
	"github.com/marcus/sidecar/internal/mouse"
)

// selectionPoint represents a position in the output buffer as (line, col).
// col is in visual space (post-tab-expansion, accounting for multi-width chars).
// endCol convention: INCLUSIVE — the character under the cursor IS selected.
type selectionPoint struct {
	line int // buffer line index, -1 = unset
	col  int // visual column, -1 = unset
}

// before returns true if p is before other in document order.
func (p selectionPoint) before(other selectionPoint) bool {
	return p.line < other.line || (p.line == other.line && p.col < other.col)
}

// valid returns true if the point has been set.
func (p selectionPoint) valid() bool {
	return p.line >= 0 && p.col >= 0
}

func (p *Plugin) clearInteractiveSelection() {
	p.interactiveSelectionActive = false
	p.interactiveSelStart = selectionPoint{-1, -1}
	p.interactiveSelEnd = selectionPoint{-1, -1}
	p.interactiveSelAnchor = selectionPoint{-1, -1}
	p.interactiveSelectionRect = mouse.Rect{}
}

func (p *Plugin) hasInteractiveSelection() bool {
	return p.interactiveSelStart.valid() && p.interactiveSelEnd.valid()
}

func (p *Plugin) isInteractiveLineSelected(lineIdx int) bool {
	if !p.hasInteractiveSelection() {
		return false
	}
	start := p.interactiveSelStart.line
	end := p.interactiveSelEnd.line
	if start > end {
		start, end = end, start
	}
	return lineIdx >= start && lineIdx <= end
}

// interactiveColAtX maps a viewport X coordinate to a visual column in the given line.
// The returned column is in visual space (post-tab-expansion, accounting for multi-width chars).
func (p *Plugin) interactiveColAtX(x, lineIdx int) (int, bool) {
	relX := x - p.interactiveSelectionRect.X - panelOverhead/2
	if relX < 0 {
		return 0, false
	}

	visualCol := relX

	buf := p.interactiveOutputBuffer()
	if buf == nil {
		return 0, true
	}
	if lineIdx < 0 || lineIdx >= buf.LineCount() {
		return 0, true
	}

	lines := buf.LinesRange(lineIdx, lineIdx+1)
	if len(lines) == 0 {
		return 0, true
	}
	expanded := expandTabs(lines[0], tabStopWidth)

	// Walk expanded line grapheme-by-grapheme to find column
	state := ansi.NormalState
	cumWidth := 0
	lastCharCol := 0
	hasChars := false

	remaining := expanded
	for len(remaining) > 0 {
		seq, width, n, newState := ansi.GraphemeWidth.DecodeSequenceInString(remaining, state, nil)
		if n <= 0 {
			break
		}
		_ = seq
		if width > 0 {
			hasChars = true
			// If visualCol lands within this character's cells
			if visualCol >= cumWidth && visualCol < cumWidth+width {
				return cumWidth, true // snap to start of character
			}
			lastCharCol = cumWidth
			cumWidth += width
		}
		state = newState
		remaining = remaining[n:]
	}

	if !hasChars {
		return 0, true
	}

	// Beyond line end: clamp to last character column (inclusive)
	if visualCol >= cumWidth {
		return lastCharCol, true
	}
	return visualCol, true
}

// interactiveCharAtXY maps viewport coordinates to line index + visual column.
func (p *Plugin) interactiveCharAtXY(x, y int) (int, int, bool) {
	lineIdx, ok := p.interactiveLineIndexAtY(y)
	if !ok {
		return 0, 0, false
	}
	col, ok := p.interactiveColAtX(x, lineIdx)
	return lineIdx, col, ok
}

// getLineSelectionCols returns the visual column range [start, end] (inclusive)
// that is selected on the given line. Returns (-1, -1) if the line is not selected.
// end == -1 means "to end of line".
func (p *Plugin) getLineSelectionCols(lineIdx int) (startCol, endCol int) {
	if !p.hasInteractiveSelection() {
		return -1, -1
	}

	selStart := p.interactiveSelStart
	selEnd := p.interactiveSelEnd

	if lineIdx < selStart.line || lineIdx > selEnd.line {
		return -1, -1
	}

	if selStart.line == selEnd.line {
		// Single-line selection
		return selStart.col, selEnd.col
	} else if lineIdx == selStart.line {
		// First line: from startCol to end of line
		return selStart.col, -1
	} else if lineIdx == selEnd.line {
		// Last line: from start to endCol (inclusive)
		return 0, selEnd.col
	}
	// Middle line: entire line
	return 0, -1
}

func (p *Plugin) interactiveLineIndexAtY(y int) (int, bool) {
	if p.interactiveState == nil || !p.interactiveState.Active {
		return 0, false
	}
	if p.interactiveSelectionRect.W == 0 || p.interactiveSelectionRect.H == 0 {
		return 0, false
	}
	if p.interactiveState.VisibleEnd <= p.interactiveState.VisibleStart {
		return 0, false
	}

	contentRow := y - p.interactiveSelectionRect.Y - 1 // account for top border
	if contentRow < 0 {
		return 0, false
	}
	if p.interactiveState.ContentRowOffset <= 0 {
		return 0, false
	}
	outputRow := contentRow - p.interactiveState.ContentRowOffset
	if outputRow < 0 {
		return 0, false
	}
	lineIdx := p.interactiveState.VisibleStart + outputRow
	if lineIdx < p.interactiveState.VisibleStart || lineIdx >= p.interactiveState.VisibleEnd {
		return 0, false
	}
	return lineIdx, true
}

// prepareInteractiveDrag stores the click position and starts drag tracking
// without initializing selection. Selection only activates on actual drag motion.
func (p *Plugin) prepareInteractiveDrag(action mouse.MouseAction) tea.Cmd {
	if action.Region == nil {
		return nil
	}
	p.interactiveSelectionRect = action.Region.Rect

	lineIdx, col, ok := p.interactiveCharAtXY(action.X, action.Y)
	if !ok {
		p.clearInteractiveSelection()
		return nil
	}

	p.interactiveSelectionActive = false
	p.interactiveSelStart = selectionPoint{-1, -1}
	p.interactiveSelEnd = selectionPoint{-1, -1}
	p.interactiveSelAnchor = selectionPoint{lineIdx, col}
	p.autoScrollOutput = false

	p.mouseHandler.StartDrag(action.X, action.Y, regionPreviewPane, lineIdx)
	return nil
}

func (p *Plugin) handleInteractiveSelectionDrag(action mouse.MouseAction) tea.Cmd {
	lineIdx, col, ok := p.interactiveCharAtXY(action.X, action.Y)
	if !ok {
		return nil
	}

	current := selectionPoint{lineIdx, col}
	anchor := p.interactiveSelAnchor

	// First drag motion: activate selection
	if !p.interactiveSelStart.valid() {
		p.interactiveSelStart = anchor
		p.interactiveSelEnd = anchor
	}

	p.interactiveSelectionActive = true

	// Order start/end by document position
	if current.before(anchor) {
		p.interactiveSelStart = current
		p.interactiveSelEnd = anchor
	} else {
		p.interactiveSelStart = anchor
		p.interactiveSelEnd = current
	}
	return nil
}

func (p *Plugin) finishInteractiveSelection() tea.Cmd {
	if !p.interactiveSelStart.valid() {
		// No drag occurred — click without motion, clear state
		p.clearInteractiveSelection()
		return nil
	}
	p.interactiveSelectionActive = false
	return nil
}

func (p *Plugin) interactiveOutputBuffer() *OutputBuffer {
	if p.shellSelected {
		shell := p.getSelectedShell()
		if shell != nil && shell.Agent != nil {
			return shell.Agent.OutputBuf
		}
		return nil
	}
	wt := p.selectedWorktree()
	if wt == nil || wt.Agent == nil {
		return nil
	}
	return wt.Agent.OutputBuf
}

func (p *Plugin) interactiveSelectionLines() []string {
	if !p.hasInteractiveSelection() {
		return nil
	}
	buf := p.interactiveOutputBuffer()
	if buf == nil {
		return nil
	}

	lineCount := buf.LineCount()
	if lineCount == 0 {
		return nil
	}

	start := p.interactiveSelStart
	end := p.interactiveSelEnd
	startLine := start.line
	endLine := end.line
	if startLine > endLine {
		startLine, endLine = endLine, startLine
	}
	if startLine < 0 {
		startLine = 0
	}
	if endLine >= lineCount {
		endLine = lineCount - 1
	}
	if endLine < startLine {
		return nil
	}

	lines := buf.LinesRange(startLine, endLine+1)
	if len(lines) == 0 {
		return nil
	}

	// Expand tabs to match visual columns
	for i := range lines {
		lines[i] = expandTabs(lines[i], tabStopWidth)
	}

	startCol := start.col
	endCol := end.col

	if len(lines) == 1 {
		// Single line: extract [startCol, endCol] inclusive
		lines[0] = visualSubstring(lines[0], startCol, endCol+1)
	} else {
		// First line: from startCol to end
		lines[0] = visualSubstring(lines[0], startCol, -1)
		// Last line: from start to endCol (inclusive)
		lastIdx := len(lines) - 1
		lines[lastIdx] = visualSubstring(lines[lastIdx], 0, endCol+1)
		// Middle lines: full text, ANSI stripped
		for i := 1; i < lastIdx; i++ {
			lines[i] = ansi.Strip(lines[i])
		}
	}

	return lines
}

func (p *Plugin) interactiveVisibleLines() []string {
	if p.interactiveState == nil || !p.interactiveState.Active {
		return nil
	}
	buf := p.interactiveOutputBuffer()
	if buf == nil {
		return nil
	}
	start := p.interactiveState.VisibleStart
	end := p.interactiveState.VisibleEnd
	if end <= start {
		return nil
	}
	return buf.LinesRange(start, end)
}

func (p *Plugin) copyInteractiveSelectionCmd() tea.Cmd {
	return func() tea.Msg {
		lines := p.interactiveSelectionLines()
		if len(lines) == 0 {
			lines = p.interactiveVisibleLines()
		}
		if len(lines) == 0 {
			return app.ToastMsg{Message: "No output to copy", Duration: 2 * time.Second}
		}

		stripped := make([]string, 0, len(lines))
		for _, line := range lines {
			stripped = append(stripped, ansi.Strip(line))
		}
		text := strings.Join(stripped, "\n")
		if err := clipboard.WriteAll(text); err != nil {
			return app.ToastMsg{Message: "Copy failed: " + err.Error(), Duration: 2 * time.Second, IsError: true}
		}

		return app.ToastMsg{Message: fmt.Sprintf("Copied %d line(s)", len(stripped)), Duration: 2 * time.Second}
	}
}
