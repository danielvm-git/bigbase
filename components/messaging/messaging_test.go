package messaging_test

import (
	"testing"

	"github.com/danielvm/bigbase/components/messaging"
	"github.com/danielvm/bigbase/kernel"
)

func TestMessagingImplementsComponent(t *testing.T) {
	var _ kernel.Component = (*messaging.Messaging)(nil)
}

func TestMessagingName(t *testing.T) {
	m := &messaging.Messaging{}
	if got := m.Name(); got != "messaging" {
		t.Fatalf("expected Name()='messaging', got '%s'", got)
	}
}

func TestMessagingVersion(t *testing.T) {
	m := &messaging.Messaging{}
	if got := m.Version(); got == "" {
		t.Fatal("expected non-empty version")
	}
}

func TestMessagingDependencies(t *testing.T) {
	m := &messaging.Messaging{}
	deps := m.Dependencies()
	if len(deps) != 1 || deps[0] != "db" {
		t.Fatalf("expected dependency on 'db', got %v", deps)
	}
}

func TestMessagingHooks(t *testing.T) {
	m := &messaging.Messaging{}
	if got := m.Hooks(); len(got) != 0 {
		t.Fatalf("expected no hooks, got %v", got)
	}
}
