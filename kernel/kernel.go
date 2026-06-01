package kernel

import (
	"encoding/json"
	"fmt"
)

// Version is the build-time app version, injected via ldflags (e.g., -X ...kernel.Version=1.2.3).
// Falls back to "0.0.0-dev" when not set at build time (local dev without ldflags).
var Version = "0.0.0-dev"

type ComponentStatus struct {
	Name         string
	Version      string
	Dependencies []string
	Hooks        []string
	Running      bool
}

type Kernel struct {
	components   map[string]Component
	eventBus     *EventBus
	logger       Logger
	config       map[string]any
	startedOrder []string
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
	k.startedOrder = nil
	order, err := k.resolveOrder()
	if err != nil {
		return fmt.Errorf("resolve order: %w", err)
	}
	for _, name := range order {
		comp := k.components[name]
		ctx := &Context{
			Kernel:     k,
			Logger:     k.logger,
			Components: k.Components(),
			Config:     make(map[string]json.RawMessage),
		}
		if err := comp.Init(ctx, nil); err != nil {
			return fmt.Errorf("init %s: %w", name, err)
		}
		if err := comp.Start(ctx); err != nil {
			return fmt.Errorf("start %s: %w", name, err)
		}
		k.startedOrder = append(k.startedOrder, name)
		k.logger.Info("started component", "name", name)
	}
	return nil
}

func (k *Kernel) Stop() error {
	order, _ := k.resolveOrder()
	for i := len(order) - 1; i >= 0; i-- {
		comp := k.components[order[i]]
		if err := comp.Stop(&Context{
			Kernel: k,
			Logger: k.logger,
			Config: make(map[string]json.RawMessage),
		}); err != nil {
			return fmt.Errorf("stop %s: %w", order[i], err)
		}
	}
	k.startedOrder = nil
	return nil
}

func (k *Kernel) EventBus() *EventBus {
	return k.eventBus
}

func (k *Kernel) Components() map[string]Component {
	result := make(map[string]Component, len(k.components))
	for name, comp := range k.components {
		result[name] = comp
	}
	return result
}

func (k *Kernel) ListComponents() []ComponentStatus {
	running := make(map[string]bool, len(k.startedOrder))
	for _, n := range k.startedOrder {
		running[n] = true
	}

	result := make([]ComponentStatus, 0, len(k.components))
	order, _ := k.resolveOrder()
	for _, name := range order {
		comp := k.components[name]
		result = append(result, ComponentStatus{
			Name:         name,
			Version:      comp.Version(),
			Dependencies: comp.Dependencies(),
			Hooks:        hookNames(comp.Hooks()),
			Running:      running[name],
		})
	}
	return result
}

func hookNames(hooks []HookDef) []string {
	names := make([]string, len(hooks))
	for i, h := range hooks {
		names[i] = h.Name
	}
	return names
}

func (k *Kernel) resolveOrder() ([]string, error) {
	const (
		white = 0
		gray  = 1
		black = 2
	)
	state := make(map[string]int)
	order := make([]string, 0)

	var visit func(name, requiredBy string) error
	visit = func(name, requiredBy string) error {
		switch state[name] {
		case gray:
			return fmt.Errorf("circular dependency detected at %s", name)
		case black:
			return nil
		}
		state[name] = gray
		comp, ok := k.components[name]
		if !ok {
			return fmt.Errorf("dependency %q required by %q is not registered", name, requiredBy)
		}
		for _, dep := range comp.Dependencies() {
			if err := visit(dep, name); err != nil {
				return err
			}
		}
		state[name] = black
		order = append(order, name)
		return nil
	}

	for name := range k.components {
		if err := visit(name, ""); err != nil {
			return nil, err
		}
	}

	return order, nil
}
