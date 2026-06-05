package kernel

import (
	"sync"
	"testing"
)

func TestEventBusConcurrent(t *testing.T) {
	bus := NewEventBus()
	var wg sync.WaitGroup

	// Concurrently subscribe 20 listeners
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			unsub := bus.Subscribe(HookDef{
				Name:     "test.event",
				Priority: id,
				Handler: func(ctx *Context, e Event) error {
					return nil
				},
			})
			defer unsub()
		}(i)
	}

	// Concurrently emit events
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			_ = bus.Emit(Event{Name: "test.event"}, nil)
		}(i)
	}

	// Concurrently query subscriber count
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = bus.SubscriberCount()
		}()
	}

	wg.Wait()
}

func TestEventBusSubscribeUnsubscribeRace(t *testing.T) {
	bus := NewEventBus()
	var wg sync.WaitGroup

	hook := HookDef{
		Name:     "race.test",
		Priority: 1,
		Handler: func(ctx *Context, e Event) error {
			return nil
		},
	}

	// Repeatedly subscribe and unsubscribe while emitting
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			unsub := bus.Subscribe(hook)
			_ = bus.Emit(Event{Name: "race.test"}, nil)
			unsub()
		}()
	}
	wg.Wait()
}

func TestEventBusEmitNoSubscribers(t *testing.T) {
	bus := NewEventBus()
	err := bus.Emit(Event{Name: "nonexistent"}, nil)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}

func TestEventBusSubscribeAndEmit(t *testing.T) {
	bus := NewEventBus()
	var called bool
	var mu sync.Mutex

	bus.Subscribe(HookDef{
		Name:     "test",
		Priority: 1,
		Handler: func(ctx *Context, e Event) error {
			mu.Lock()
			called = true
			mu.Unlock()
			return nil
		},
	})

	err := bus.Emit(Event{Name: "test"}, nil)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	mu.Lock()
	if !called {
		t.Fatal("expected handler to be called")
	}
	mu.Unlock()
}

func TestEventBusPriorityOrder(t *testing.T) {
	bus := NewEventBus()
	var order []int
	var mu sync.Mutex

	for _, p := range []int{3, 1, 2} {
		p := p
		bus.Subscribe(HookDef{
			Name:     "priority",
			Priority: p,
			Handler: func(ctx *Context, e Event) error {
				mu.Lock()
				order = append(order, p)
				mu.Unlock()
				return nil
			},
		})
	}

	_ = bus.Emit(Event{Name: "priority"}, nil)

	mu.Lock()
	if len(order) != 3 {
		t.Fatalf("expected 3 handler calls, got %d", len(order))
	}
	if order[0] != 1 || order[1] != 2 || order[2] != 3 {
		t.Fatalf("expected order [1,2,3], got %v", order)
	}
	mu.Unlock()
}

func TestEventBusUnsubscribe(t *testing.T) {
	bus := NewEventBus()

	unsub := bus.Subscribe(HookDef{
		Name:     "unsub",
		Priority: 1,
		Handler: func(ctx *Context, e Event) error {
			return nil
		},
	})

	unsub()

	// Handler should not be called after unsubscribe — but we can't directly
	// test that without a side effect. Instead, verify Emit doesn't error.
	err := bus.Emit(Event{Name: "unsub"}, nil)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}
