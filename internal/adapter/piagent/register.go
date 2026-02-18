package piagent

import "github.com/wilbur182/forge/internal/adapter"

func init() {
	adapter.RegisterFactory(func() adapter.Adapter {
		return New()
	})
}
