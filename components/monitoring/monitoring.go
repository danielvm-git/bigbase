package monitoring

import (
	"encoding/json"
	"net/http"

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

type Monitoring struct {
	logger Logger
}

type Options struct {
	Logger Logger
}

func New(opts Options) *Monitoring {
	logger := opts.Logger
	if logger == nil {
		logger = noopLogger{}
	}
	return &Monitoring{logger: logger}
}

func (m *Monitoring) Name() string                  { return "monitoring" }
func (m *Monitoring) Version() string               { return version }
func (m *Monitoring) Dependencies() []string         { return nil }
func (m *Monitoring) ConfigSchema() json.RawMessage { return nil }
func (m *Monitoring) Hooks() []kernel.HookDef        { return nil }
func (m *Monitoring) Init(ctx *kernel.Context, config json.RawMessage) error { return nil }
func (m *Monitoring) Start(ctx *kernel.Context) error { return nil }
func (m *Monitoring) Stop(ctx *kernel.Context) error  { return nil }

func (m *Monitoring) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/monitoring/health", m.handleHealth)
	return mux
}

func (m *Monitoring) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}

var _ kernel.Component = (*Monitoring)(nil)
