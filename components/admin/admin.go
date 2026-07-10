package admin

import (
	"encoding/json"
	"net/http"

	"github.com/danielvm/bigbase/kernel"
	"github.com/danielvm/bigbase/ui"
)

const version = "0.1.0"

type Admin struct {
	logger kernel.Logger
}

type Options struct {
	Logger kernel.Logger
}

func New(opts Options) *Admin {
	return &Admin{logger: opts.Logger}
}

func (ad *Admin) Name() string                  { return "admin" }
func (ad *Admin) Version() string               { return version }
func (ad *Admin) Dependencies() []string        { return nil }
func (ad *Admin) ConfigSchema() json.RawMessage { return nil }
func (ad *Admin) Hooks() []kernel.HookDef       { return nil }

func (ad *Admin) Init(ctx *kernel.Context, config json.RawMessage) error {
	return nil
}

func (ad *Admin) Start(ctx *kernel.Context) error {
	ad.logger.Info("admin component ready")
	return nil
}

func (ad *Admin) Stop(ctx *kernel.Context) error {
	return nil
}

func (ad *Admin) Handler() http.Handler {
	return http.FileServer(http.FS(ui.DistFS))
}
