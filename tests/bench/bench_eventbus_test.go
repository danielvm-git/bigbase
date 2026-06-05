package bench

import (
	"testing"

	"github.com/danielvm/bigbase/kernel"
)

func BenchmarkEventBus10k(b *testing.B) {
	eb := kernel.NewEventBus()

	var callCount int
	eb.Subscribe(kernel.HookDef{
		Name:     "bench.event",
		Priority: 1,
		Handler: func(ctx *kernel.Context, e kernel.Event) error {
			callCount++
			return nil
		},
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for j := 0; j < 10000; j++ {
			_ = eb.Emit(kernel.Event{Name: "bench.event"}, nil)
		}
	}
}

func BenchmarkEventBusSubscribe(b *testing.B) {
	eb := kernel.NewEventBus()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		unsub := eb.Subscribe(kernel.HookDef{
			Name:     "bench.sub",
			Priority: 1,
			Handler: func(ctx *kernel.Context, e kernel.Event) error {
				return nil
			},
		})
		unsub()
	}
}
