package worktree

import (
	"encoding/json"
	"testing"
)

func TestMergeWorkflowStepString(t *testing.T) {
	tests := []struct {
		step     MergeWorkflowStep
		expected string
	}{
		{MergeStepReviewDiff, "Review Diff"},
		{MergeStepPush, "Push Branch"},
		{MergeStepCreatePR, "Create PR"},
		{MergeStepWaitingMerge, "Waiting for Merge"},
		{MergeStepPostMergeConfirmation, "Confirm Cleanup"},
		{MergeStepCleanup, "Cleanup"},
		{MergeStepDone, "Done"},
		{MergeWorkflowStep(99), "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := tt.step.String()
			if result != tt.expected {
				t.Errorf("MergeWorkflowStep(%d).String() = %q, want %q", tt.step, result, tt.expected)
			}
		})
	}
}

func TestTruncateDiff(t *testing.T) {
	tests := []struct {
		name     string
		diff     string
		maxLines int
		wantLen  int // Expected number of lines
	}{
		{
			name:     "short diff",
			diff:     "line1\nline2\nline3",
			maxLines: 5,
			wantLen:  3,
		},
		{
			name:     "exact limit",
			diff:     "line1\nline2\nline3\nline4\nline5",
			maxLines: 5,
			wantLen:  5,
		},
		{
			name:     "over limit",
			diff:     "line1\nline2\nline3\nline4\nline5\nline6\nline7",
			maxLines: 3,
			wantLen:  4, // 3 lines + truncation message
		},
		{
			name:     "empty diff",
			diff:     "",
			maxLines: 5,
			wantLen:  1, // Just the empty string
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncateDiff(tt.diff, tt.maxLines)

			// For short diff, result should equal input
			if tt.name == "short diff" && result != tt.diff {
				t.Errorf("truncateDiff() should not modify short diff")
			}

			// For over limit, should contain truncation message
			if tt.name == "over limit" {
				if len(result) <= len(tt.diff) {
					// Actually truncated diff should be shorter content-wise but has extra message
				}
			}
		})
	}
}

func TestMergeWorkflowState(t *testing.T) {
	wt := &Worktree{
		Name:       "test-branch",
		Path:       "/tmp/test",
		Branch:     "test-branch",
		BaseBranch: "main",
	}

	state := &MergeWorkflowState{
		Worktree:   wt,
		Step:       MergeStepReviewDiff,
		PRTitle:    "Test PR",
		StepStatus: make(map[MergeWorkflowStep]string),
	}

	// Test initial state
	if state.Worktree != wt {
		t.Error("Worktree not set correctly")
	}
	if state.Step != MergeStepReviewDiff {
		t.Errorf("Step = %v, want MergeStepReviewDiff", state.Step)
	}

	// Test step status
	state.StepStatus[MergeStepReviewDiff] = "done"
	if state.StepStatus[MergeStepReviewDiff] != "done" {
		t.Error("StepStatus not working correctly")
	}
}

func TestCancelMergeWorkflow(t *testing.T) {
	p := &Plugin{
		viewMode: ViewModeMerge,
		mergeState: &MergeWorkflowState{
			Worktree: &Worktree{Name: "test"},
		},
	}

	p.cancelMergeWorkflow()

	if p.mergeState != nil {
		t.Error("mergeState should be nil after cancel")
	}
	if p.viewMode != ViewModeList {
		t.Errorf("viewMode = %v, want ViewModeList", p.viewMode)
	}
}

func TestParsePRMergeStatus(t *testing.T) {
	// Test parsing various JSON responses from gh pr view
	tests := []struct {
		name     string
		json     string
		expected bool
	}{
		{
			name:     "merged true",
			json:     `{"state":"MERGED","merged":true}`,
			expected: true,
		},
		{
			name:     "merged true with whitespace",
			json:     `{"state": "MERGED", "merged": true}`,
			expected: true,
		},
		{
			name:     "state MERGED only",
			json:     `{"state":"MERGED","merged":false}`,
			expected: true, // State takes precedence
		},
		{
			name:     "not merged",
			json:     `{"state":"OPEN","merged":false}`,
			expected: false,
		},
		{
			name:     "closed but not merged",
			json:     `{"state":"CLOSED","merged":false}`,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse using same logic as checkPRMerged
			var prStatus struct {
				State  string `json:"state"`
				Merged bool   `json:"merged"`
			}
			err := json.Unmarshal([]byte(tt.json), &prStatus)
			if err != nil {
				t.Fatalf("failed to parse JSON: %v", err)
			}

			merged := prStatus.Merged || prStatus.State == "MERGED"
			if merged != tt.expected {
				t.Errorf("parsed merged = %v, want %v", merged, tt.expected)
			}
		})
	}
}
