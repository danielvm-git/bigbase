package kernel_test

import (
	"testing"

	"encoding/json"

	"github.com/danielvm/bigbase/kernel"
)

type testLogger struct{}

func (testLogger) Info(msg string, args ...any)  {}
func (testLogger) Warn(msg string, args ...any)  {}
func (testLogger) Error(msg string, args ...any) {}
func (testLogger) Debug(msg string, args ...any) {}

type testComponent struct {
	name string
	deps []string
}

func (t *testComponent) Name() string                                           { return t.name }
func (t *testComponent) Version() string                                        { return "0.0.1" }
func (t *testComponent) Dependencies() []string                                 { return t.deps }
func (t *testComponent) ConfigSchema() json.RawMessage                          { return nil }
func (t *testComponent) Init(ctx *kernel.Context, config json.RawMessage) error { return nil }
func (t *testComponent) Start(ctx *kernel.Context) error                        { return nil }
func (t *testComponent) Stop(ctx *kernel.Context) error                         { return nil }
func (t *testComponent) Hooks() []kernel.HookDef                                { return nil }

func TestNewKernel(t *testing.T) {
	k := kernel.New(testLogger{})
	if k == nil {
		t.Fatal("expected kernel to be non-nil")
	}
}

func TestRegister(t *testing.T) {
	k := kernel.New(testLogger{})
	comp := &testComponent{name: "test"}
	k.Register(comp)
}

func TestVersion(t *testing.T) {
	if kernel.Version == "" {
		t.Fatal("expected version to be non-empty")
	}
}

func TestComponents(t *testing.T) {
	k := kernel.New(testLogger{})
	if len(k.Components()) != 0 {
		t.Fatal("expected no components initially")
	}
	comp := &testComponent{name: "test"}
	k.Register(comp)
	if len(k.Components()) != 1 {
		t.Fatal("expected 1 component after register")
	}
}

func TestListComponents(t *testing.T) {
	k := kernel.New(testLogger{})
	comp := &testComponent{name: "test"}
	k.Register(comp)

	statuses := k.ListComponents()
	if len(statuses) != 1 {
		t.Fatalf("expected 1 component, got %d", len(statuses))
	}
	if statuses[0].Name != "test" {
		t.Fatalf("expected name 'test', got '%s'", statuses[0].Name)
	}
	if statuses[0].Running {
		t.Fatal("expected component to be not running before Start")
	}

	if err := k.Start(); err != nil {
		t.Fatalf("unexpected start error: %v", err)
	}
	defer func() { _ = k.Stop() }()

	statuses = k.ListComponents()
	if !statuses[0].Running {
		t.Fatal("expected component to be running after Start")
	}
}

func TestListComponentsWithDeps(t *testing.T) {
	k := kernel.New(testLogger{})
	dep := &testComponent{name: "db"}
	main := &testComponent{name: "api", deps: []string{"db"}}
	k.Register(main)
	k.Register(dep)

	statuses := k.ListComponents()
	if len(statuses) != 2 {
		t.Fatalf("expected 2 components, got %d", len(statuses))
	}
}

func TestSubscriberCount(t *testing.T) {
	bus := kernel.NewEventBus()
	if bus.SubscriberCount() != 0 {
		t.Fatal("expected 0 subscribers initially")
	}
	bus.Subscribe(kernel.HookDef{Name: "test", Priority: 0, Handler: func(ctx *kernel.Context, e kernel.Event) error { return nil }})
	if bus.SubscriberCount() != 1 {
		t.Fatal("expected 1 subscriber after subscribe")
	}
}

func TestStartStop(t *testing.T) {
	k := kernel.New(testLogger{})
	comp := &testComponent{name: "test"}
	k.Register(comp)

	if err := k.Start(); err != nil {
		t.Fatalf("unexpected start error: %v", err)
	}
	if err := k.Stop(); err != nil {
		t.Fatalf("unexpected stop error: %v", err)
	}
}
