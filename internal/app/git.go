package app

import (
	"os/exec"
	"path/filepath"
	"strings"
)

// GetRepoName returns the git repository name for the given directory.
// It tries to extract the name from the remote URL first, falling back
// to the directory name if no remote is configured.
// Returns empty string if the directory is not a git repository.
func GetRepoName(workDir string) string {
	// Check if this is a git repo
	cmd := exec.Command("git", "rev-parse", "--git-dir")
	cmd.Dir = workDir
	if err := cmd.Run(); err != nil {
		return ""
	}

	// Try to get remote URL
	cmd = exec.Command("git", "remote", "get-url", "origin")
	cmd.Dir = workDir
	output, err := cmd.Output()
	if err == nil {
		url := strings.TrimSpace(string(output))
		if name := parseRepoNameFromURL(url); name != "" {
			return name
		}
	}

	// Fallback to directory name
	absPath, err := filepath.Abs(workDir)
	if err != nil {
		return ""
	}
	return filepath.Base(absPath)
}

// parseRepoNameFromURL extracts the repository name from a git URL.
// Handles both SSH (git@github.com:user/repo.git) and HTTPS URLs.
func parseRepoNameFromURL(url string) string {
	// Remove .git suffix
	url = strings.TrimSuffix(url, ".git")

	// Handle SSH format: git@github.com:user/repo
	if idx := strings.LastIndex(url, ":"); idx != -1 && !strings.Contains(url, "://") {
		url = url[idx+1:]
	}

	// Handle HTTPS format: https://github.com/user/repo
	if idx := strings.LastIndex(url, "/"); idx != -1 {
		return url[idx+1:]
	}

	return url
}
