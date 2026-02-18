package agentwasmruntime

// Bridge provides the interface between Go and WASM for tool execution.
type Bridge struct {
	readFile      func(path string) ([]byte, error)
	writeFile     func(path string, content []byte) error
	executeCmd    func(cmd string, args []string) (string, error)
	listDirectory func(path string) ([]FileInfo, error)
	searchCode    func(query string) ([]SearchResult, error)
	sendLLM       func(provider string, messages []Message) (string, error)
}

// FileInfo describes a file in the project.
type FileInfo struct {
	Name  string
	Path  string
	IsDir bool
	Size  int64
}

// SearchResult represents a code search result.
type SearchResult struct {
	FilePath string
	Line     int
	Content  string
	Match    string
}

// Message represents an LLM API message.
type Message struct {
	Role    string
	Content string
}

// NewBridge creates a new Go ↔ WASM bridge with default handlers.
func NewBridge() *Bridge {
	return &Bridge{
		readFile:      func(path string) ([]byte, error) { return nil, nil },
		writeFile:     func(path string, content []byte) error { return nil },
		executeCmd:    func(cmd string, args []string) (string, error) { return "", nil },
		listDirectory: func(path string) ([]FileInfo, error) { return nil, nil },
		searchCode:    func(query string) ([]SearchResult, error) { return nil, nil },
		sendLLM:       func(provider string, messages []Message) (string, error) { return "", nil },
	}
}

// RegisterFileOperations sets the handlers for file I/O.
func (b *Bridge) RegisterFileOperations(readFn func(string) ([]byte, error), writeFn func(string, []byte) error) {
	b.readFile = readFn
	b.writeFile = writeFn
}

// RegisterCommandExecution sets the handler for command execution.
func (b *Bridge) RegisterCommandExecution(fn func(string, []string) (string, error)) {
	b.executeCmd = fn
}

// RegisterFileSystemOperations sets the handlers for filesystem navigation.
func (b *Bridge) RegisterFileSystemOperations(listFn func(string) ([]FileInfo, error)) {
	b.listDirectory = listFn
}

// RegisterCodeSearch sets the handler for code search operations.
func (b *Bridge) RegisterCodeSearch(fn func(string) ([]SearchResult, error)) {
	b.searchCode = fn
}

// RegisterLLMProvider sets the handler for LLM API requests.
func (b *Bridge) RegisterLLMProvider(fn func(string, []Message) (string, error)) {
	b.sendLLM = fn
}
