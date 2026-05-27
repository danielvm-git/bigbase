package kernel

import "encoding/json"

type Component interface {
	Name() string
	Version() string
	Dependencies() []string
	ConfigSchema() json.RawMessage
	Init(ctx *Context, config json.RawMessage) error
	Start(ctx *Context) error
	Stop(ctx *Context) error
	Hooks() []HookDef
}

type HookDef struct {
	Name     string
	Priority int
	Handler  HookFunc
	subID    uint64
}

type HookFunc func(ctx *Context, event Event) error

type Event struct {
	Name string
	Data map[string]any
}

type Context struct {
	Kernel     *Kernel
	Logger     Logger
	Components map[string]Component
	Config     map[string]json.RawMessage
}

type Logger interface {
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
	Debug(msg string, args ...any)
}
