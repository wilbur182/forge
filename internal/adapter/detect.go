package adapter

// adapterFactories holds registered adapter constructors.
var adapterFactories []func() Adapter

// RegisterFactory registers an adapter constructor.
func RegisterFactory(factory func() Adapter) {
	adapterFactories = append(adapterFactories, factory)
}

// DetectAdapters scans for available adapters for the given project.
func DetectAdapters(projectRoot string) (map[string]Adapter, error) {
	adapters := make(map[string]Adapter)
	for _, factory := range adapterFactories {
		instance := factory()
		detected, err := instance.Detect(projectRoot)
		if err != nil || !detected {
			continue
		}
		adapters[instance.ID()] = instance
	}
	return adapters, nil
}
