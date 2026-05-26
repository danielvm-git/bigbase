package kernel

import (
	"encoding/json"
	"fmt"
)

const Version = "0.1.0"

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
	for _, name := range k.resolveOrder() {
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
	order := k.resolveOrder()
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
	for _, name := range k.resolveOrder() {
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
