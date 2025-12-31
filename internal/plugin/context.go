package plugin

import (
	"log/slog"

	"github.com/sst/sidecar/internal/adapter"
	"github.com/sst/sidecar/internal/event"
)

// BindingRegistrar allows plugins to register key bindings dynamically.
// This is implemented by keymap.Registry.
type BindingRegistrar interface {
	RegisterPluginBinding(key, command, context string)
}

// Context provides shared resources to plugins during initialization.
type Context struct {
	WorkDir   string
	ConfigDir string
	Adapters  map[string]adapter.Adapter
	EventBus  *event.Dispatcher
	Logger    *slog.Logger
	Keymap    BindingRegistrar // For plugins to register dynamic bindings
}
