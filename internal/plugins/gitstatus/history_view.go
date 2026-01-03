package gitstatus

import (
	"fmt"
	"strings"

	"github.com/marcus/sidecar/internal/styles"
)

// renderHistory renders the commit history list.
func (p *Plugin) renderHistory() string {
	var sb strings.Builder

	// Header with push status
	header := " Commit History"
	if p.pushStatus != nil {
		status := p.pushStatus.FormatAheadBehind()
		if status != "" {
			header = fmt.Sprintf(" Commit History  %s", styles.StatusModified.Render(status))
		}
	}
	sb.WriteString(styles.PanelHeader.Render(header))
	sb.WriteString("\n")
	sb.WriteString(styles.Muted.Render(strings.Repeat("━", p.width-2)))
	sb.WriteString("\n")

	if p.commits == nil || len(p.commits) == 0 {
		sb.WriteString(styles.Muted.Render(" Loading commits..."))
		return sb.String()
	}

	// Calculate visible area
	contentHeight := p.height - 3 // header + separator + padding
	if contentHeight < 1 {
		contentHeight = 1
	}

	// Render commits
	start := p.historyScroll
	if start >= len(p.commits) {
		start = 0
	}
	end := start + contentHeight
	if end > len(p.commits) {
		end = len(p.commits)
	}

	for i := start; i < end; i++ {
		commit := p.commits[i]
		selected := i == p.historyCursor

		line := p.renderCommitLine(commit, selected)
		sb.WriteString(line)
		sb.WriteString("\n")
	}

	return sb.String()
}

// renderCommitLine renders a single commit entry.
func (p *Plugin) renderCommitLine(c *Commit, selected bool) string {
	// Cursor indicator
	cursor := "  "
	if selected {
		cursor = styles.ListCursor.Render("> ")
	}

	// Push status indicator
	var pushIndicator string
	if !c.Pushed {
		pushIndicator = styles.StatusModified.Render("↑") + " "
	} else {
		pushIndicator = "  " // Two spaces for alignment
	}

	// Hash
	hash := styles.Code.Render(c.ShortHash)

	// Subject (truncate if needed)
	maxSubjectWidth := p.width - 34 // Reserve space for hash, time, indicator
	subject := c.Subject
	if len(subject) > maxSubjectWidth && maxSubjectWidth > 3 {
		subject = subject[:maxSubjectWidth-3] + "..."
	}

	// Relative time
	timeStr := styles.Muted.Render(RelativeTime(c.Date))

	if selected {
		plainCursor := "> "
		plainIndicator := "  "
		if !c.Pushed {
			plainIndicator = "↑ "
		}
		plainLine := fmt.Sprintf("%s%s%s %s  %s", plainCursor, plainIndicator, c.ShortHash, subject, RelativeTime(c.Date))
		maxWidth := p.width - 4
		if len(plainLine) < maxWidth {
			plainLine += strings.Repeat(" ", maxWidth-len(plainLine))
		}
		return styles.ListItemSelected.Render(plainLine)
	}

	return styles.ListItemNormal.Render(fmt.Sprintf("%s%s%s %s  %s", cursor, pushIndicator, hash, subject, timeStr))
}

// renderCommitDetail renders the commit detail view.
func (p *Plugin) renderCommitDetail() string {
	// Clear hit regions for mouse support
	p.mouseHandler.Clear()

	var sb strings.Builder

	if p.selectedCommit == nil {
		sb.WriteString(styles.Muted.Render(" Loading commit..."))
		return sb.String()
	}

	c := p.selectedCommit

	// Track Y position for hit regions
	currentY := 0

	// Header with commit info
	sb.WriteString(styles.ModalTitle.Render(" Commit: " + c.ShortHash))
	sb.WriteString("\n")
	currentY++
	sb.WriteString(styles.Muted.Render(strings.Repeat("━", p.width-2)))
	sb.WriteString("\n\n")
	currentY += 2

	// Metadata
	sb.WriteString(styles.Subtitle.Render(" Author: "))
	sb.WriteString(styles.Body.Render(fmt.Sprintf("%s <%s>", c.Author, c.AuthorEmail)))
	sb.WriteString("\n")
	currentY++

	sb.WriteString(styles.Subtitle.Render(" Date:   "))
	sb.WriteString(styles.Body.Render(c.Date.Format("Mon Jan 2 15:04:05 2006")))
	sb.WriteString("\n\n")
	currentY += 2

	// Subject
	sb.WriteString(styles.Title.Render(" " + c.Subject))
	sb.WriteString("\n")
	currentY++

	// Body (if present)
	if c.Body != "" {
		sb.WriteString("\n")
		currentY++
		for _, line := range strings.Split(c.Body, "\n") {
			sb.WriteString(styles.Body.Render(" " + line))
			sb.WriteString("\n")
			currentY++
		}
	}

	sb.WriteString("\n")
	currentY++
	sb.WriteString(styles.Muted.Render(strings.Repeat("─", p.width-2)))
	sb.WriteString("\n")
	currentY++

	// Stats
	statsLine := fmt.Sprintf(" %d files changed", c.Stats.FilesChanged)
	if c.Stats.Additions > 0 {
		statsLine += ", " + styles.DiffAdd.Render(fmt.Sprintf("+%d", c.Stats.Additions))
	}
	if c.Stats.Deletions > 0 {
		statsLine += ", " + styles.DiffRemove.Render(fmt.Sprintf("-%d", c.Stats.Deletions))
	}
	sb.WriteString(statsLine)
	sb.WriteString("\n\n")
	currentY += 2

	// Files list
	contentHeight := p.height - 12 // Account for header, metadata, etc.
	if contentHeight < 1 {
		contentHeight = 1
	}

	start := p.commitDetailScroll
	if start >= len(c.Files) {
		start = 0
	}
	end := start + contentHeight
	if end > len(c.Files) {
		end = len(c.Files)
	}

	for i := start; i < end; i++ {
		file := c.Files[i]
		selected := i == p.commitDetailCursor

		// Register hit region for this file row
		p.mouseHandler.HitMap.AddRect(regionCommitDetailFile, 0, currentY, p.width, 1, i)

		line := p.renderCommitFile(file, selected)
		sb.WriteString(line)
		sb.WriteString("\n")
		currentY++
	}

	return sb.String()
}

// renderCommitFile renders a single file in commit detail.
func (p *Plugin) renderCommitFile(f CommitFile, selected bool) string {
	// Cursor
	cursor := "  "
	if selected {
		cursor = styles.ListCursor.Render("> ")
	}

	// Path
	path := f.Path
	if f.OldPath != "" {
		path = fmt.Sprintf("%s → %s", f.OldPath, f.Path)
	}

	// Stats
	stats := ""
	if f.Additions > 0 || f.Deletions > 0 {
		addStr := ""
		delStr := ""
		if f.Additions > 0 {
			addStr = styles.DiffAdd.Render(fmt.Sprintf("+%d", f.Additions))
		}
		if f.Deletions > 0 {
			delStr = styles.DiffRemove.Render(fmt.Sprintf("-%d", f.Deletions))
		}
		stats = fmt.Sprintf(" %s %s", addStr, delStr)
	}

	// Truncate path if needed
	maxPathWidth := p.width - 20
	if len(path) > maxPathWidth && maxPathWidth > 3 {
		path = "..." + path[len(path)-maxPathWidth+3:]
	}

	if selected {
		plainCursor := "> "
		plainStats := ""
		if f.Additions > 0 || f.Deletions > 0 {
			plainStats = fmt.Sprintf(" +%d -%d", f.Additions, f.Deletions)
		}
		plainLine := fmt.Sprintf("%s%s%s", plainCursor, path, plainStats)
		maxWidth := p.width - 4
		if len(plainLine) < maxWidth {
			plainLine += strings.Repeat(" ", maxWidth-len(plainLine))
		}
		return styles.ListItemSelected.Render(plainLine)
	}

	return styles.ListItemNormal.Render(fmt.Sprintf("%s%s%s", cursor, path, stats))
}
