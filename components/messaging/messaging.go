package messaging

import (
	"encoding/json"

	"github.com/danielvm/bigbase/kernel"
)

const version = "0.1.0"

type Logger interface {
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
	Debug(msg string, args ...any)
}

type noopLogger struct{}

func (noopLogger) Info(msg string, args ...any)  {}
func (noopLogger) Warn(msg string, args ...any)  {}
func (noopLogger) Error(msg string, args ...any) {}
func (noopLogger) Debug(msg string, args ...any) {}

type Messaging struct {
	logger Logger
}

type Options struct {
	Logger Logger
}

func New(opts Options) *Messaging {
	logger := opts.Logger
	if logger == nil {
		logger = noopLogger{}
	}
	return &Messaging{logger: logger}
}

func (m *Messaging) Name() string                   { return "messaging" }
func (m *Messaging) Version() string                { return version }
func (m *Messaging) Dependencies() []string         { return []string{"db"} }
func (m *Messaging) ConfigSchema() json.RawMessage  { return nil }
func (m *Messaging) Hooks() []kernel.HookDef        { return nil }

func (m *Messaging) Init(ctx *kernel.Context, config json.RawMessage) error {
	return nil
}

func (m *Messaging) Start(ctx *kernel.Context) error {
	m.logger.Info("messaging component ready")
	return nil
}

func (m *Messaging) Stop(ctx *kernel.Context) error {
	return nil
}

var _ kernel.Component = (*Messaging)(nil)
