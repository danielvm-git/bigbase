package kernel

import (
	"sort"
	"sync"
)

type EventBus struct {
	mu     sync.RWMutex
	hooks  map[string][]HookDef
	nextID uint64
}

func NewEventBus() *EventBus {
	return &EventBus{
		hooks: make(map[string][]HookDef),
	}
}

func (eb *EventBus) Subscribe(hook HookDef) func() {
	eb.mu.Lock()
	eb.nextID++
	hook.subID = eb.nextID
	eb.hooks[hook.Name] = append(eb.hooks[hook.Name], hook)
	eb.mu.Unlock()

	return func() {
		eb.mu.Lock()
		hooks := eb.hooks[hook.Name]
		for i, h := range hooks {
			if h.subID == hook.subID {
				eb.hooks[hook.Name] = append(hooks[:i], hooks[i+1:]...)
				eb.mu.Unlock()
				return
			}
		}
		eb.mu.Unlock()
	}
}

func (eb *EventBus) SubscriberCount() int {
	eb.mu.RLock()
	defer eb.mu.RUnlock()
	return len(eb.hooks)
}

func (eb *EventBus) Emit(event Event, ctx *Context) error {
	eb.mu.RLock()
	hooks := eb.hooks[event.Name]
	if len(hooks) == 0 {
		eb.mu.RUnlock()
		return nil
	}
	sorted := make([]HookDef, len(hooks))
	copy(sorted, hooks)
	eb.mu.RUnlock()

	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Priority < sorted[j].Priority
	})

	for _, hook := range sorted {
		if err := hook.Handler(ctx, event); err != nil {
			return err
		}
	}

	return nil
}
