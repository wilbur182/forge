package agentwasmruntime

// HostCallbacks defines the functions that WASM can call back into the Go host.
type HostCallbacks struct {
	OnReadFile      func(path string) ([]byte, error)
	OnWriteFile     func(path string, content []byte) error
	OnExecuteCmd    func(cmd string, args []string) (string, error)
	OnListDirectory func(path string) ([]FileInfo, error)
	OnSearchCode    func(query string) ([]SearchResult, error)
	OnSendLLM       func(provider string, messages []Message) (string, error)
}

// NewHostCallbacks returns a default set of host callbacks.
func NewHostCallbacks() *HostCallbacks {
	return &HostCallbacks{
		OnReadFile:      func(path string) ([]byte, error) { return nil, nil },
		OnWriteFile:     func(path string, content []byte) error { return nil },
		OnExecuteCmd:    func(cmd string, args []string) (string, error) { return "", nil },
		OnListDirectory: func(path string) ([]FileInfo, error) { return nil, nil },
		OnSearchCode:    func(query string) ([]SearchResult, error) { return nil, nil },
		OnSendLLM:       func(provider string, messages []Message) (string, error) { return "", nil },
	}
}
