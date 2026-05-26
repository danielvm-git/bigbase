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
}

func (t *testComponent) Name() string                     { return t.name }
func (t *testComponent) Version() string                  { return "0.0.1" }
func (t *testComponent) Dependencies() []string            { return nil }
func (t *testComponent) ConfigSchema() json.RawMessage     { return nil }
func (t *testComponent) Init(ctx *kernel.Context, config json.RawMessage) error { return nil }
func (t *testComponent) Start(ctx *kernel.Context) error  { return nil }
func (t *testComponent) Stop(ctx *kernel.Context) error   { return nil }
func (t *testComponent) Hooks() []kernel.HookDef          { return nil }

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
