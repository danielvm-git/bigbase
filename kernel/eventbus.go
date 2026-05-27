package kernel

import "sort"

type EventBus struct {
	hooks  map[string][]HookDef
	nextID uint64
}

func NewEventBus() *EventBus {
	return &EventBus{
		hooks: make(map[string][]HookDef),
	}
}

func (eb *EventBus) Subscribe(hook HookDef) func() {
	eb.nextID++
	hook.subID = eb.nextID
	eb.hooks[hook.Name] = append(eb.hooks[hook.Name], hook)
	return func() {
		hooks := eb.hooks[hook.Name]
		for i, h := range hooks {
			if h.subID == hook.subID {
				eb.hooks[hook.Name] = append(hooks[:i], hooks[i+1:]...)
				return
			}
		}
	}
}

func (eb *EventBus) SubscriberCount() int {
	return len(eb.hooks)
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
