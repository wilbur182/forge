package gitstatus

import (
	"testing"

	"github.com/marcus/sidecar/internal/mouse"
)

func makeEntries(count int, status FileStatus) []*FileEntry {
	entries := make([]*FileEntry, count)
	for i := 0; i < count; i++ {
		entries[i] = &FileEntry{
			Path:   "file",
			Status: status,
		}
	}
	return entries
}

func TestCommitSectionCapacity_TruncatesFiles(t *testing.T) {
	p := &Plugin{
		tree: &FileTree{
			Staged:    makeEntries(2, StatusAdded),
			Modified:  makeEntries(3, StatusModified),
			Untracked: makeEntries(2, StatusUntracked),
		},
		recentCommits: make([]*Commit, 10),
	}

	got := p.commitSectionCapacity(16)
	want := 5
	if got != want {
		t.Fatalf("commitSectionCapacity = %d, want %d", got, want)
	}
}

func TestCommitSectionCapacity_CleanWithStatus(t *testing.T) {
	p := &Plugin{
		tree:           &FileTree{},
		pushInProgress: true,
	}

	got := p.commitSectionCapacity(10)
	want := 5
	if got != want {
		t.Fatalf("commitSectionCapacity = %d, want %d", got, want)
	}
}

func TestCommitSectionCapacity_Minimum(t *testing.T) {
	p := &Plugin{
		tree: &FileTree{},
	}

	got := p.commitSectionCapacity(5)
	want := 2
	if got != want {
		t.Fatalf("commitSectionCapacity = %d, want %d", got, want)
	}
}

func makeCommitsWithHash(count int) []*Commit {
	commits := make([]*Commit, count)
	for i := range count {
		commits[i] = &Commit{
			Hash:    "abc1234" + string(rune('a'+i)),
			Subject: "Test commit",
			Pushed:  true,
		}
	}
	return commits
}

func TestCommitHitRegions_CleanTree(t *testing.T) {
	// When tree is clean ("Working tree clean" shown), commit hit regions
	// must align with visual positions. This tests the fix for the off-by-one
	// bug where clicking a commit row selected the row above.
	handler := mouse.NewHandler()
	p := &Plugin{
		tree:          &FileTree{}, // Empty = "Working tree clean"
		sidebarWidth:  40,
		mouseHandler:  handler,
		recentCommits: makeCommitsWithHash(5),
	}

	// Render sidebar to populate hit regions
	_ = p.renderSidebar(20)

	// Find all commit regions and verify they have sequential indices
	var commitRegions []mouse.Region
	for _, r := range handler.HitMap.Regions() {
		if r.ID == regionCommit {
			commitRegions = append(commitRegions, r)
		}
	}

	if len(commitRegions) == 0 {
		t.Fatal("no commit hit regions registered")
	}

	// Verify regions have correct data indices (0, 1, 2, ...)
	for i, r := range commitRegions {
		idx, ok := r.Data.(int)
		if !ok {
			t.Fatalf("region %d has non-int data: %T", i, r.Data)
		}
		if idx != i {
			t.Fatalf("region %d has data=%d, want %d", i, idx, i)
		}
	}

	// Verify hit test at first commit Y returns commit 0
	firstCommitY := commitRegions[0].Rect.Y
	region := handler.HitMap.Test(5, firstCommitY)
	if region == nil {
		t.Fatalf("hit test at Y=%d returned nil", firstCommitY)
	}
	if region.ID != regionCommit {
		t.Fatalf("hit test returned region %q, want %q", region.ID, regionCommit)
	}
	if idx, _ := region.Data.(int); idx != 0 {
		t.Fatalf("hit test returned commit %d, want 0", idx)
	}
}
