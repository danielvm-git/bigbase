package kernel

import "encoding/json"

type Kernel struct {
	components map[string]Component
	eventBus   *EventBus
	logger     Logger
	config     map[string]any
}

func New(logger Logger) *Kernel {
	return &Kernel{
		components: make(map[string]Component),
		eventBus:   NewEventBus(),
		logger:     logger,
		config:     make(map[string]any),
	}
}

func (k *Kernel) Register(component Component) {
	name := component.Name()
	k.components[name] = component
	k.logger.Info("registered component", "name", name, "version", component.Version())
}

func (k *Kernel) Start() error {
	for _, name := range k.resolveOrder() {
		comp := k.components[name]
		ctx := &Context{
			Kernel:     k,
			Logger:     k.logger,
			Components: k.components,
			Config:     make(map[string]json.RawMessage),
		}
		if err := comp.Init(ctx, nil); err != nil {
			return err
		}
		if err := comp.Start(ctx); err != nil {
			return err
		}
		k.logger.Info("started component", "name", name)
	}
	return nil
}

func (k *Kernel) Stop() error {
	for _, name := range k.resolveOrder() {
		comp := k.components[name]
		if err := comp.Stop(&Context{}); err != nil {
			return err
		}
	}
	return nil
}

func (k *Kernel) EventBus() *EventBus {
	return k.eventBus
}

func (k *Kernel) resolveOrder() []string {
	visited := make(map[string]bool)
	order := make([]string, 0)

	var visit func(name string)
	visit = func(name string) {
		if visited[name] {
			return
		}
		visited[name] = true
		comp, ok := k.components[name]
		if !ok {
			return
		}
		for _, dep := range comp.Dependencies() {
			visit(dep)
		}
		order = append(order, name)
	}

	for name := range k.components {
		visit(name)
	}

	return order
}
