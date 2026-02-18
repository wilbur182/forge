package bridge

import (
	"context"
	"sync"
)

// ProjectContext holds shared state across all plugins.
type ProjectContext struct {
	mu               sync.RWMutex
	RootPath         string
	CurrentBranch    string
	ActiveTask       *Task
	OpenFiles        []string
	AgentSession     *AgentSession
	PluginData       map[string]interface{}
	lastModifiedTime int64
}

// Task represents a td task.
type Task struct {
	ID    string
	Title string
	State string
}

// AgentSession represents an active agent conversation.
type AgentSession struct {
	ID       string
	Provider string
	Messages []AgentMessage
}

// AgentMessage represents a single message in agent conversation.
type AgentMessage struct {
	Role    string
	Content string
}

// NewProjectContext creates a new shared project context.
func NewProjectContext(rootPath string) *ProjectContext {
	return &ProjectContext{
		RootPath:   rootPath,
		OpenFiles:  make([]string, 0),
		PluginData: make(map[string]interface{}),
	}
}

// SetCurrentBranch updates the active git branch.
func (pc *ProjectContext) SetCurrentBranch(branch string) {
	pc.mu.Lock()
	defer pc.mu.Unlock()
	pc.CurrentBranch = branch
	pc.lastModifiedTime = getCurrentTime()
}

// GetCurrentBranch returns the active git branch.
func (pc *ProjectContext) GetCurrentBranch() string {
	pc.mu.RLock()
	defer pc.mu.RUnlock()
	return pc.CurrentBranch
}

// SetActiveTask updates the current task in focus.
func (pc *ProjectContext) SetActiveTask(task *Task) {
	pc.mu.Lock()
	defer pc.mu.Unlock()
	pc.ActiveTask = task
	pc.lastModifiedTime = getCurrentTime()
}

// GetActiveTask returns the current task in focus.
func (pc *ProjectContext) GetActiveTask() *Task {
	pc.mu.RLock()
	defer pc.mu.RUnlock()
	return pc.ActiveTask
}

// SetOpenFiles updates the list of currently open files.
func (pc *ProjectContext) SetOpenFiles(files []string) {
	pc.mu.Lock()
	defer pc.mu.Unlock()
	pc.OpenFiles = append(make([]string, 0, len(files)), files...)
	pc.lastModifiedTime = getCurrentTime()
}

// GetOpenFiles returns the list of currently open files.
func (pc *ProjectContext) GetOpenFiles() []string {
	pc.mu.RLock()
	defer pc.mu.RUnlock()
	result := make([]string, len(pc.OpenFiles))
	copy(result, pc.OpenFiles)
	return result
}

// SetPluginData stores plugin-specific data in shared context.
func (pc *ProjectContext) SetPluginData(pluginID string, data interface{}) {
	pc.mu.Lock()
	defer pc.mu.Unlock()
	pc.PluginData[pluginID] = data
	pc.lastModifiedTime = getCurrentTime()
}

// GetPluginData retrieves plugin-specific data from shared context.
func (pc *ProjectContext) GetPluginData(pluginID string) interface{} {
	pc.mu.RLock()
	defer pc.mu.RUnlock()
	return pc.PluginData[pluginID]
}

// GetSnapshot returns a read-only snapshot of the current context.
func (pc *ProjectContext) GetSnapshot(ctx context.Context) *ProjectContext {
	pc.mu.RLock()
	defer pc.mu.RUnlock()
	snapshot := &ProjectContext{
		RootPath:      pc.RootPath,
		CurrentBranch: pc.CurrentBranch,
		ActiveTask:    pc.ActiveTask,
		PluginData:    make(map[string]interface{}),
	}
	snapshot.OpenFiles = append(make([]string, 0, len(pc.OpenFiles)), pc.OpenFiles...)
	for k, v := range pc.PluginData {
		snapshot.PluginData[k] = v
	}
	return snapshot
}

func getCurrentTime() int64 {
	// Placeholder for actual time implementation
	return 0
}
