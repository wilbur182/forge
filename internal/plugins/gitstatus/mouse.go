package gitstatus

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/marcus/sidecar/internal/mouse"
)

// Hit region IDs
const (
	regionSidebar     = "sidebar"
	regionDiffPane    = "diff-pane"
	regionPaneDivider = "pane-divider"
	regionFile        = "file"
	regionCommit      = "commit"
)

// handleMouse processes mouse events in the status view.
func (p *Plugin) handleMouse(msg tea.MouseMsg) (*Plugin, tea.Cmd) {
	action := p.mouseHandler.HandleMouse(msg)

	switch action.Type {
	case mouse.ActionClick:
		return p.handleMouseClick(action)

	case mouse.ActionDoubleClick:
		return p.handleMouseDoubleClick(action)

	case mouse.ActionScrollUp, mouse.ActionScrollDown:
		return p.handleMouseScroll(action)

	case mouse.ActionDrag:
		return p.handleMouseDrag(action)

	case mouse.ActionDragEnd:
		return p, nil
	}

	return p, nil
}

// handleMouseClick handles single click events.
func (p *Plugin) handleMouseClick(action mouse.MouseAction) (*Plugin, tea.Cmd) {
	if action.Region == nil {
		return p, nil
	}

	switch action.Region.ID {
	case regionSidebar:
		p.activePane = PaneSidebar
		return p, nil

	case regionDiffPane:
		p.activePane = PaneDiff
		return p, nil

	case regionPaneDivider:
		// Start drag for pane resizing
		p.mouseHandler.StartDrag(action.X, action.Y, regionPaneDivider, p.sidebarWidth)
		return p, nil

	case regionFile:
		// Click on file - select it
		if idx, ok := action.Region.Data.(int); ok {
			if idx != p.cursor {
				p.cursor = idx
				p.ensureCursorVisible()
				if p.cursorOnCommit() {
					return p, p.autoLoadCommitPreview()
				}
				return p, p.autoLoadDiff()
			}
		}
		return p, nil

	case regionCommit:
		// Click on commit - select it
		// idx is now absolute index into recentCommits
		if idx, ok := action.Region.Data.(int); ok {
			fileCount := len(p.tree.AllEntries())
			newCursor := fileCount + idx
			if newCursor != p.cursor {
				p.cursor = newCursor
				p.ensureCursorVisible()
				p.ensureCommitVisible(idx)
				return p, p.autoLoadCommitPreview()
			}
		}
		return p, nil
	}

	return p, nil
}

// handleMouseDoubleClick handles double-click events.
func (p *Plugin) handleMouseDoubleClick(action mouse.MouseAction) (*Plugin, tea.Cmd) {
	if action.Region == nil {
		return p, nil
	}

	switch action.Region.ID {
	case regionFile:
		// Double-click on file - open it or toggle folder
		entries := p.tree.AllEntries()
		if idx, ok := action.Region.Data.(int); ok && idx < len(entries) {
			entry := entries[idx]
			if entry.IsFolder {
				// Toggle folder expansion
				entry.IsExpanded = !entry.IsExpanded
				return p, p.autoLoadDiff()
			}
			// Open file in editor
			return p, p.openFile(entry.Path)
		}
		return p, nil

	case regionCommit:
		// Double-click on commit - focus preview pane
		// idx is now absolute index into recentCommits
		if idx, ok := action.Region.Data.(int); ok {
			fileCount := len(p.tree.AllEntries())
			p.cursor = fileCount + idx
			p.ensureCursorVisible()
			p.ensureCommitVisible(idx)
			if p.previewCommit != nil {
				p.activePane = PaneDiff
			}
			return p, p.autoLoadCommitPreview()
		}
		return p, nil

	case regionDiffPane:
		// Double-click in diff pane when on a file - open full-screen diff
		if !p.cursorOnCommit() {
			entries := p.tree.AllEntries()
			if p.cursor < len(entries) {
				entry := entries[p.cursor]
				p.diffReturnMode = p.viewMode
				p.viewMode = ViewModeDiff
				p.diffFile = entry.Path
				p.diffCommit = ""
				p.diffScroll = 0
				if entry.IsFolder {
					return p, p.loadFullFolderDiff(entry)
				}
				return p, p.loadDiff(entry.Path, entry.Staged, entry.Status)
			}
		}
		return p, nil
	}

	return p, nil
}

// handleMouseScroll handles scroll wheel events.
func (p *Plugin) handleMouseScroll(action mouse.MouseAction) (*Plugin, tea.Cmd) {
	if action.Region == nil {
		// No hit region - scroll based on pane position
		if action.X < p.sidebarWidth+2 {
			return p.scrollSidebar(action.Delta)
		}
		return p.scrollDiffPane(action.Delta)
	}

	switch action.Region.ID {
	case regionSidebar, regionFile, regionCommit:
		return p.scrollSidebar(action.Delta)

	case regionDiffPane:
		return p.scrollDiffPane(action.Delta)
	}

	return p, nil
}

// scrollSidebar scrolls the sidebar file list.
func (p *Plugin) scrollSidebar(delta int) (*Plugin, tea.Cmd) {
	totalItems := p.totalSelectableItems()
	if totalItems == 0 {
		return p, nil
	}

	// Move cursor by scroll amount
	newCursor := p.cursor + delta
	if newCursor < 0 {
		newCursor = 0
	}
	if newCursor >= totalItems {
		newCursor = totalItems - 1
	}

	if newCursor != p.cursor {
		p.cursor = newCursor
		p.ensureCursorVisible()
		if p.cursorOnCommit() {
			commitIdx := p.selectedCommitIndex()
			p.ensureCommitVisible(commitIdx)
			// Trigger load-more when within 3 commits of end
			var loadMoreCmd tea.Cmd
			if commitIdx >= len(p.recentCommits)-3 && !p.loadingMoreCommits {
				loadMoreCmd = p.loadMoreCommits()
			}
			return p, tea.Batch(p.autoLoadCommitPreview(), loadMoreCmd)
		}
		return p, p.autoLoadDiff()
	}

	return p, nil
}

// scrollDiffPane scrolls the diff pane content.
func (p *Plugin) scrollDiffPane(delta int) (*Plugin, tea.Cmd) {
	// If showing commit preview, scroll its file list
	if p.previewCommit != nil && p.cursorOnCommit() {
		p.previewCommitScroll += delta
		if p.previewCommitScroll < 0 {
			p.previewCommitScroll = 0
		}
		maxScroll := len(p.previewCommit.Files) - 5
		if maxScroll < 0 {
			maxScroll = 0
		}
		if p.previewCommitScroll > maxScroll {
			p.previewCommitScroll = maxScroll
		}
		return p, nil
	}

	// Otherwise scroll the diff content
	p.diffPaneScroll += delta
	if p.diffPaneScroll < 0 {
		p.diffPaneScroll = 0
	}

	// Clamp to max if we have parsed diff content
	if p.diffPaneParsedDiff != nil {
		lines := countParsedDiffLines(p.diffPaneParsedDiff)
		maxScroll := lines - (p.height - 6)
		if maxScroll < 0 {
			maxScroll = 0
		}
		if p.diffPaneScroll > maxScroll {
			p.diffPaneScroll = maxScroll
		}
	}

	return p, nil
}

// handleMouseDrag handles drag motion events.
func (p *Plugin) handleMouseDrag(action mouse.MouseAction) (*Plugin, tea.Cmd) {
	if p.mouseHandler.DragRegion() == regionPaneDivider {
		// Calculate new sidebar width based on drag
		startValue := p.mouseHandler.DragStartValue()
		newWidth := startValue + action.DragDX

		// Clamp to reasonable bounds
		minWidth := 25
		maxWidth := p.width * 50 / 100 // Max 50% of screen

		if newWidth < minWidth {
			newWidth = minWidth
		}
		if newWidth > maxWidth {
			newWidth = maxWidth
		}

		p.sidebarWidth = newWidth
		p.calculatePaneWidths()
		return p, nil
	}

	return p, nil
}
