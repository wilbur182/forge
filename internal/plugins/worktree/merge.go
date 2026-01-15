package worktree

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/marcus/sidecar/internal/plugins/gitstatus"
)

// MergeWorkflowStep represents the current step in the merge workflow.
type MergeWorkflowStep int

const (
	MergeStepReviewDiff MergeWorkflowStep = iota
	MergeStepPush
	MergeStepCreatePR
	MergeStepWaitingMerge
	MergeStepPostMergeConfirmation // User confirms cleanup options after PR merge
	MergeStepCleanup
	MergeStepDone
)

// String returns a display name for the merge step.
func (s MergeWorkflowStep) String() string {
	switch s {
	case MergeStepReviewDiff:
		return "Review Diff"
	case MergeStepPush:
		return "Push Branch"
	case MergeStepCreatePR:
		return "Create PR"
	case MergeStepWaitingMerge:
		return "Waiting for Merge"
	case MergeStepPostMergeConfirmation:
		return "Confirm Cleanup"
	case MergeStepCleanup:
		return "Cleanup"
	case MergeStepDone:
		return "Done"
	default:
		return "Unknown"
	}
}

// MergeWorkflowState holds the state for the merge workflow modal.
type MergeWorkflowState struct {
	Worktree         *Worktree
	Step             MergeWorkflowStep
	DiffSummary      string
	PRTitle          string
	PRBody           string
	PRURL            string
	Error            error
	StepStatus       map[MergeWorkflowStep]string // "pending", "running", "done", "error", "skipped"
	DeleteAfterMerge bool                         // true = delete worktree after merge (default)

	// Post-merge confirmation options
	DeleteLocalWorktree bool // Checkbox: delete local worktree (default: true)
	DeleteLocalBranch   bool // Checkbox: delete local branch (default: true)
	DeleteRemoteBranch  bool // Checkbox: delete remote branch (default: false)
	ConfirmationFocus   int  // 0-2=checkboxes, 3=confirm btn, 4=skip btn
	ConfirmationHover   int  // Mouse hover state

	// Cleanup results for summary display
	CleanupResults *CleanupResults
}

// CleanupResults holds the results of cleanup operations for display in summary.
type CleanupResults struct {
	LocalWorktreeDeleted bool
	LocalBranchDeleted   bool
	RemoteBranchDeleted  bool
	Errors               []string
}

// MergeStepCompleteMsg signals a merge workflow step completed.
type MergeStepCompleteMsg struct {
	WorktreeName string
	Step         MergeWorkflowStep
	Data         string // Step-specific data (e.g., PR URL)
	Err          error
}

// CheckPRMergedMsg signals the result of checking if a PR was merged.
type CheckPRMergedMsg struct {
	WorktreeName string
	Merged       bool
	Err          error
}

// UncommittedChangesCheckMsg signals the result of checking for uncommitted changes.
type UncommittedChangesCheckMsg struct {
	WorktreeName     string
	HasChanges       bool
	StagedCount      int
	ModifiedCount    int
	UntrackedCount   int
	Err              error
}

// MergeCommitDoneMsg signals that the commit before merge completed.
type MergeCommitDoneMsg struct {
	WorktreeName string
	CommitHash   string
	Err          error
}

// MergeCommitState holds state for the commit-before-merge modal.
type MergeCommitState struct {
	Worktree       *Worktree
	StagedCount    int
	ModifiedCount  int
	UntrackedCount int
	CommitMessage  string
	Error          string
}

// RemoteBranchDeleteMsg signals the result of deleting a remote branch.
type RemoteBranchDeleteMsg struct {
	WorktreeName string
	BranchName   string
	Err          error
}

// CleanupDoneMsg signals that cleanup operations completed.
type CleanupDoneMsg struct {
	WorktreeName string
	Results      *CleanupResults
}

// checkUncommittedChanges checks if a worktree has uncommitted changes.
func (p *Plugin) checkUncommittedChanges(wt *Worktree) tea.Cmd {
	return func() tea.Msg {
		tree := gitstatus.NewFileTree(wt.Path)
		if err := tree.Refresh(); err != nil {
			return UncommittedChangesCheckMsg{
				WorktreeName: wt.Name,
				HasChanges:   false,
				Err:          err,
			}
		}

		stagedCount := len(tree.Staged)
		modifiedCount := len(tree.Modified)
		untrackedCount := len(tree.Untracked)
		hasChanges := stagedCount > 0 || modifiedCount > 0 || untrackedCount > 0

		return UncommittedChangesCheckMsg{
			WorktreeName:   wt.Name,
			HasChanges:     hasChanges,
			StagedCount:    stagedCount,
			ModifiedCount:  modifiedCount,
			UntrackedCount: untrackedCount,
		}
	}
}

// stageAllAndCommit stages all changes and commits with the given message.
func (p *Plugin) stageAllAndCommit(wt *Worktree, message string) tea.Cmd {
	return func() tea.Msg {
		tree := gitstatus.NewFileTree(wt.Path)
		if tree == nil {
			return MergeCommitDoneMsg{
				WorktreeName: wt.Name,
				Err:          fmt.Errorf("failed to initialize git tree for %s", wt.Path),
			}
		}

		// Stage all changes
		if err := tree.StageAll(); err != nil {
			return MergeCommitDoneMsg{
				WorktreeName: wt.Name,
				Err:          fmt.Errorf("failed to stage: %w", err),
			}
		}

		// Execute commit
		hash, err := gitstatus.ExecuteCommit(wt.Path, message)
		if err != nil {
			return MergeCommitDoneMsg{
				WorktreeName: wt.Name,
				Err:          err,
			}
		}

		return MergeCommitDoneMsg{
			WorktreeName: wt.Name,
			CommitHash:   hash,
		}
	}
}

// startMergeWorkflow initializes the merge workflow for a worktree.
// It first checks for uncommitted changes and shows a commit modal if needed.
func (p *Plugin) startMergeWorkflow(wt *Worktree) tea.Cmd {
	if wt == nil {
		return nil
	}

	// Check for uncommitted changes before proceeding
	return p.checkUncommittedChanges(wt)
}

// proceedToMergeWorkflow initializes the actual merge workflow (after commit check passes).
func (p *Plugin) proceedToMergeWorkflow(wt *Worktree) tea.Cmd {
	// Initialize merge state
	p.mergeState = &MergeWorkflowState{
		Worktree:         wt,
		Step:             MergeStepReviewDiff,
		PRTitle:          wt.Branch, // Default title to branch name
		PRBody:           "",
		StepStatus:       make(map[MergeWorkflowStep]string),
		DeleteAfterMerge: true, // default to delete worktree after merge
	}
	p.mergeState.StepStatus[MergeStepReviewDiff] = "running"

	p.viewMode = ViewModeMerge

	// Load diff summary for review
	return p.loadMergeDiff(wt)
}

// loadMergeDiff loads the diff summary for the merge workflow.
func (p *Plugin) loadMergeDiff(wt *Worktree) tea.Cmd {
	return func() tea.Msg {
		// Get diff against base branch
		baseBranch := wt.BaseBranch
		if baseBranch == "" {
			baseBranch = "main"
		}

		diff, err := getDiffFromBase(wt.Path, baseBranch)
		if err != nil {
			return MergeStepCompleteMsg{
				WorktreeName: wt.Name,
				Step:         MergeStepReviewDiff,
				Data:         "",
				Err:          err,
			}
		}

		// Get a summary (stat output)
		summary, _ := getDiffSummary(wt.Path)

		return MergeStepCompleteMsg{
			WorktreeName: wt.Name,
			Step:         MergeStepReviewDiff,
			Data:         summary + "\n\n" + truncateDiff(diff, 50),
		}
	}
}

// truncateDiff truncates a diff to a maximum number of lines.
func truncateDiff(diff string, maxLines int) string {
	lines := strings.Split(diff, "\n")
	if len(lines) <= maxLines {
		return diff
	}
	truncated := strings.Join(lines[:maxLines], "\n")
	return truncated + fmt.Sprintf("\n... (%d more lines)", len(lines)-maxLines)
}

// pushForMerge pushes the branch for the merge workflow.
func (p *Plugin) pushForMerge(wt *Worktree) tea.Cmd {
	return func() tea.Msg {
		err := doPush(wt.Path, wt.Branch, false, true)
		return MergeStepCompleteMsg{
			WorktreeName: wt.Name,
			Step:         MergeStepPush,
			Err:          err,
		}
	}
}

// createPR creates a pull request using gh CLI.
func (p *Plugin) createPR(wt *Worktree, title, body string) tea.Cmd {
	return func() tea.Msg {
		baseBranch := wt.BaseBranch
		if baseBranch == "" {
			baseBranch = "main"
		}

		// Build gh pr create command
		args := []string{"pr", "create",
			"--title", title,
			"--body", body,
			"--base", baseBranch,
		}

		cmd := exec.Command("gh", args...)
		cmd.Dir = wt.Path
		output, err := cmd.CombinedOutput()

		if err != nil {
			return MergeStepCompleteMsg{
				WorktreeName: wt.Name,
				Step:         MergeStepCreatePR,
				Err:          fmt.Errorf("gh pr create: %s: %w", strings.TrimSpace(string(output)), err),
			}
		}

		// Output should contain the PR URL
		prURL := strings.TrimSpace(string(output))

		return MergeStepCompleteMsg{
			WorktreeName: wt.Name,
			Step:         MergeStepCreatePR,
			Data:         prURL,
		}
	}
}

// checkPRMerged checks if a PR has been merged using gh CLI.
func (p *Plugin) checkPRMerged(wt *Worktree) tea.Cmd {
	return func() tea.Msg {
		// Use gh pr view to check status
		cmd := exec.Command("gh", "pr", "view", "--json", "state,mergedAt")
		cmd.Dir = wt.Path
		output, err := cmd.Output()

		if err != nil {
			return CheckPRMergedMsg{
				WorktreeName: wt.Name,
				Merged:       false,
				Err:          err,
			}
		}

		// Parse JSON response
		var prStatus struct {
			State    string `json:"state"`
			MergedAt string `json:"mergedAt"`
		}

		merged := false
		if err := json.Unmarshal(output, &prStatus); err == nil {
			merged = prStatus.MergedAt != "" || prStatus.State == "MERGED"
		}

		return CheckPRMergedMsg{
			WorktreeName: wt.Name,
			Merged:       merged,
		}
	}
}

// cleanupAfterMerge removes the worktree and branch after a successful merge.
func (p *Plugin) cleanupAfterMerge(wt *Worktree) tea.Cmd {
	return func() tea.Msg {
		name := wt.Name
		path := wt.Path
		branch := wt.Branch

		// Stop agent if running
		if wt.Agent != nil {
			sessionName := wt.Agent.TmuxSession
			exec.Command("tmux", "kill-session", "-t", sessionName).Run()
		}

		// Remove worktree
		if err := doDeleteWorktree(path); err != nil {
			return MergeStepCompleteMsg{
				WorktreeName: name,
				Step:         MergeStepCleanup,
				Err:          fmt.Errorf("remove worktree: %w", err),
			}
		}

		// Delete the branch (it's been merged)
		cmd := exec.Command("git", "branch", "-d", branch)
		cmd.Dir = p.ctx.WorkDir
		if output, err := cmd.CombinedOutput(); err != nil {
			// Try force delete if regular delete fails
			cmd = exec.Command("git", "branch", "-D", branch)
			cmd.Dir = p.ctx.WorkDir
			if output, err = cmd.CombinedOutput(); err != nil {
				return MergeStepCompleteMsg{
					WorktreeName: name,
					Step:         MergeStepCleanup,
					Err:          fmt.Errorf("delete branch: %s: %w", strings.TrimSpace(string(output)), err),
				}
			}
		}

		return MergeStepCompleteMsg{
			WorktreeName: name,
			Step:         MergeStepCleanup,
		}
	}
}

// deleteRemoteBranch deletes the remote branch from origin.
func (p *Plugin) deleteRemoteBranch(wt *Worktree) tea.Cmd {
	return func() tea.Msg {
		branch := wt.Branch
		name := wt.Name

		// Delete remote branch: git push origin --delete <branch>
		cmd := exec.Command("git", "push", "origin", "--delete", branch)
		cmd.Dir = p.ctx.WorkDir
		output, err := cmd.CombinedOutput()

		if err != nil {
			outputStr := string(output)
			// Check if branch was already deleted (GitHub auto-delete)
			if strings.Contains(outputStr, "remote ref does not exist") ||
				strings.Contains(outputStr, "unable to delete") ||
				strings.Contains(outputStr, "couldn't find remote ref") {
				// Not an error - branch already gone
				return RemoteBranchDeleteMsg{
					WorktreeName: name,
					BranchName:   branch,
				}
			}
			return RemoteBranchDeleteMsg{
				WorktreeName: name,
				BranchName:   branch,
				Err:          fmt.Errorf("delete remote branch: %s", strings.TrimSpace(outputStr)),
			}
		}

		return RemoteBranchDeleteMsg{
			WorktreeName: name,
			BranchName:   branch,
		}
	}
}

// performSelectedCleanup executes only the user-selected cleanup actions.
func (p *Plugin) performSelectedCleanup(wt *Worktree, state *MergeWorkflowState) tea.Cmd {
	return func() tea.Msg {
		results := &CleanupResults{}
		name := wt.Name
		path := wt.Path
		branch := wt.Branch

		// Stop agent if running (always do this)
		if wt.Agent != nil {
			sessionName := wt.Agent.TmuxSession
			exec.Command("tmux", "kill-session", "-t", sessionName).Run()
		}

		// Delete local worktree if selected
		if state.DeleteLocalWorktree {
			if err := doDeleteWorktree(path); err != nil {
				results.Errors = append(results.Errors, fmt.Sprintf("Worktree: %v", err))
			} else {
				results.LocalWorktreeDeleted = true
			}
		}

		// Delete local branch if selected
		if state.DeleteLocalBranch {
			cmd := exec.Command("git", "branch", "-d", branch)
			cmd.Dir = p.ctx.WorkDir
			if output, err := cmd.CombinedOutput(); err != nil {
				// Try force delete if safe delete fails
				cmd = exec.Command("git", "branch", "-D", branch)
				cmd.Dir = p.ctx.WorkDir
				if output, err = cmd.CombinedOutput(); err != nil {
					results.Errors = append(results.Errors,
						fmt.Sprintf("Branch: %s", strings.TrimSpace(string(output))))
				} else {
					results.LocalBranchDeleted = true
				}
			} else {
				results.LocalBranchDeleted = true
			}
		}

		return CleanupDoneMsg{WorktreeName: name, Results: results}
	}
}

// schedulePRCheck schedules a periodic check for PR merge status.
func (p *Plugin) schedulePRCheck(worktreeName string, delay time.Duration) tea.Cmd {
	return tea.Tick(delay, func(t time.Time) tea.Msg {
		return checkPRMergeMsg{WorktreeName: worktreeName}
	})
}

// checkPRMergeMsg triggers a PR merge status check.
type checkPRMergeMsg struct {
	WorktreeName string
}

// advanceMergeStep moves to the next step in the merge workflow.
// It marks the current step as "done" and advances to the next step.
func (p *Plugin) advanceMergeStep() tea.Cmd {
	if p.mergeState == nil {
		return nil
	}

	switch p.mergeState.Step {
	case MergeStepReviewDiff:
		// Move to push step (ReviewDiff marked done by message handler)
		p.mergeState.Step = MergeStepPush
		p.mergeState.StepStatus[MergeStepPush] = "running"
		return p.pushForMerge(p.mergeState.Worktree)

	case MergeStepPush:
		// Mark Push as done, move to create PR step
		p.mergeState.StepStatus[MergeStepPush] = "done"
		p.mergeState.Step = MergeStepCreatePR
		p.mergeState.StepStatus[MergeStepCreatePR] = "running"
		title := p.mergeState.PRTitle
		if title == "" {
			title = p.mergeState.Worktree.Branch
		}
		body := p.mergeState.PRBody
		if body == "" {
			body = "Created from worktree manager"
		}
		return p.createPR(p.mergeState.Worktree, title, body)

	case MergeStepCreatePR:
		// Mark CreatePR as done, move to waiting for merge
		p.mergeState.StepStatus[MergeStepCreatePR] = "done"
		p.mergeState.Step = MergeStepWaitingMerge
		p.mergeState.StepStatus[MergeStepWaitingMerge] = "running"
		// Schedule periodic checks
		return p.schedulePRCheck(p.mergeState.Worktree.Name, 10*time.Second)

	case MergeStepWaitingMerge:
		// Mark WaitingMerge as done, go to confirmation step
		p.mergeState.StepStatus[MergeStepWaitingMerge] = "done"
		p.mergeState.Step = MergeStepPostMergeConfirmation
		p.mergeState.StepStatus[MergeStepPostMergeConfirmation] = "running"

		// Initialize default checkbox values
		p.mergeState.DeleteLocalWorktree = true  // Default: checked
		p.mergeState.DeleteLocalBranch = true    // Default: checked
		p.mergeState.DeleteRemoteBranch = false  // Default: unchecked (safer)
		p.mergeState.ConfirmationFocus = 0
		return nil // Wait for user interaction

	case MergeStepPostMergeConfirmation:
		// Mark confirmation as done
		p.mergeState.StepStatus[MergeStepPostMergeConfirmation] = "done"

		// Check if any cleanup is selected
		hasCleanup := p.mergeState.DeleteLocalWorktree ||
			p.mergeState.DeleteLocalBranch ||
			p.mergeState.DeleteRemoteBranch

		if !hasCleanup {
			// Skip cleanup, go directly to done
			p.mergeState.Step = MergeStepDone
			p.mergeState.StepStatus[MergeStepCleanup] = "skipped"
			p.mergeState.StepStatus[MergeStepDone] = "done"
			return nil
		}

		// Proceed to cleanup
		p.mergeState.Step = MergeStepCleanup
		p.mergeState.StepStatus[MergeStepCleanup] = "running"

		var cmds []tea.Cmd

		// Local cleanup (worktree + branch)
		if p.mergeState.DeleteLocalWorktree || p.mergeState.DeleteLocalBranch {
			cmds = append(cmds, p.performSelectedCleanup(p.mergeState.Worktree, p.mergeState))
		}

		// Remote cleanup (in parallel)
		if p.mergeState.DeleteRemoteBranch {
			cmds = append(cmds, p.deleteRemoteBranch(p.mergeState.Worktree))
		}

		return tea.Batch(cmds...)

	case MergeStepCleanup:
		// Done
		p.mergeState.Step = MergeStepDone
		p.mergeState.StepStatus[MergeStepDone] = "done"
		return nil
	}

	return nil
}

// cancelMergeWorkflow cancels the merge workflow and returns to list view.
func (p *Plugin) cancelMergeWorkflow() {
	p.mergeState = nil
	p.viewMode = ViewModeList
}

// checkCleanupComplete checks if all cleanup operations are done and advances to done step.
func (p *Plugin) checkCleanupComplete() tea.Cmd {
	return func() tea.Msg {
		if p.mergeState == nil || p.mergeState.Step != MergeStepCleanup {
			return nil
		}

		// Check if we're waiting for any operations
		// If remote cleanup was requested but not done yet, wait
		if p.mergeState.DeleteRemoteBranch && p.mergeState.CleanupResults != nil &&
			!p.mergeState.CleanupResults.RemoteBranchDeleted &&
			!hasRemoteBranchError(p.mergeState.CleanupResults.Errors) {
			return nil // Still waiting for remote
		}

		// If local cleanup was requested but not done yet, wait
		localRequested := p.mergeState.DeleteLocalWorktree || p.mergeState.DeleteLocalBranch
		localDone := p.mergeState.CleanupResults != nil &&
			(p.mergeState.CleanupResults.LocalWorktreeDeleted || p.mergeState.CleanupResults.LocalBranchDeleted ||
				hasLocalError(p.mergeState.CleanupResults.Errors))

		if localRequested && !localDone {
			return nil // Still waiting for local
		}

		// All done - advance to done step
		p.mergeState.StepStatus[MergeStepCleanup] = "done"
		p.mergeState.Step = MergeStepDone
		p.mergeState.StepStatus[MergeStepDone] = "done"

		return nil
	}
}

// hasRemoteBranchError checks if there's a remote branch error in the errors list.
func hasRemoteBranchError(errors []string) bool {
	for _, err := range errors {
		if len(err) > 7 && err[:7] == "Remote " {
			return true
		}
	}
	return false
}

// hasLocalError checks if there's a local worktree/branch error in the errors list.
func hasLocalError(errors []string) bool {
	for _, err := range errors {
		if len(err) > 9 && (err[:9] == "Worktree:" || err[:7] == "Branch:") {
			return true
		}
	}
	return false
}
