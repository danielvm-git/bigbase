package kernel

import "sort"

type EventBus struct {
	hooks map[string][]HookDef
}

func NewEventBus() *EventBus {
	return &EventBus{
		hooks: make(map[string][]HookDef),
	}
}

func (eb *EventBus) Subscribe(hook HookDef) {
	eb.hooks[hook.Name] = append(eb.hooks[hook.Name], hook)
}

func (eb *EventBus) Emit(event Event, ctx *Context) error {
	hooks := eb.hooks[event.Name]
	if len(hooks) == 0 {
		return nil
	}

	sort.Slice(hooks, func(i, j int) bool {
		return hooks[i].Priority < hooks[j].Priority
	})

	for _, hook := range hooks {
		if err := hook.Handler(ctx, event); err != nil {
			return err
		}
	}

	return nil
}
