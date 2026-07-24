package functions

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"

	"github.com/robfig/cron/v3"

	"github.com/danielvm/bigbase/components/auth"
	"github.com/danielvm/bigbase/kernel"
)

var (
	errMissingName   = errors.New("name is required")
	errMissingSource = errors.New("source is required")
)

const version = "0.1.0"

// DBer is an alias for kernel.DBer — the shared database abstraction.
type DBer = kernel.DBer

type Function struct {
	ID        string            `json:"id"`
	Name      string            `json:"name"`
	Runtime   string            `json:"runtime"`
	Source    string            `json:"source"`
	Trigger   string            `json:"trigger"`
	Schedule  string            `json:"schedule,omitempty"`
	Env       map[string]string `json:"env,omitempty"`
	Timeout   int               `json:"timeout"`
	CreatedAt string            `json:"created_at"`
}

type Functions struct {
	db               DBer
	logger           kernel.Logger
	timeout          int
	runtimes         map[string]Runtime
	cron             *cron.Cron
	scheduleEnabled  bool
	mu               sync.Mutex
	scheduleEntryIDs map[string]cron.EntryID // fnID → cron entry
}

var _ kernel.Component = (*Functions)(nil)

type Options struct {
	DB              DBer
	Logger          kernel.Logger
	Timeout         int
	ScheduleEnabled bool
}

func New(opts Options) *Functions {
	logger := opts.Logger
	if logger == nil {
		logger = kernel.NoopLogger{}
	}
	timeout := opts.Timeout
	if timeout == 0 {
		timeout = 30
	}
	return &Functions{
		db:               opts.DB,
		logger:           logger,
		timeout:          timeout,
		runtimes:         map[string]Runtime{"javascript": &jsRuntime{}},
		scheduleEnabled:  opts.ScheduleEnabled,
		scheduleEntryIDs: make(map[string]cron.EntryID),
	}
}

func (f *Functions) Name() string                  { return "functions" }
func (f *Functions) Version() string               { return version }
func (f *Functions) Dependencies() []string        { return []string{"db"} }

func (f *Functions) Init(ctx *kernel.Context, config json.RawMessage) error {
	return nil
}

func (f *Functions) Start(ctx *kernel.Context) error {
	if err := f.db.Migrate(`CREATE TABLE IF NOT EXISTS functions (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		runtime TEXT NOT NULL DEFAULT 'javascript',
		source TEXT NOT NULL,
		trigger TEXT NOT NULL DEFAULT 'http',
		schedule TEXT DEFAULT '',
		env TEXT DEFAULT '{}',
		timeout INTEGER NOT NULL DEFAULT 30,
		created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now'))
	)`); err != nil {
		return fmt.Errorf("migrate functions table: %w", err)
	}
	// Add org_id column for multi-tenant isolation (BUG-131).
	_ = f.db.Migrate(`ALTER TABLE functions ADD COLUMN org_id INTEGER NOT NULL DEFAULT 0`)
	if err := f.db.Migrate(`CREATE TABLE IF NOT EXISTS function_executions (
		id TEXT PRIMARY KEY,
		function_id TEXT NOT NULL,
		status TEXT NOT NULL DEFAULT 'success',
		logs TEXT NOT NULL DEFAULT '[]',
		error TEXT DEFAULT '',
		created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now')),
		FOREIGN KEY(function_id) REFERENCES functions(id) ON DELETE CASCADE
	)`); err != nil {
		return fmt.Errorf("migrate function_executions table: %w", err)
	}
	// Add org_id column to executions for audit trail.
	_ = f.db.Migrate(`ALTER TABLE function_executions ADD COLUMN org_id INTEGER NOT NULL DEFAULT 0`)
	// Start schedule loop if enabled
	if f.scheduleEnabled {
		if err := f.startScheduleLoop(); err != nil {
			return fmt.Errorf("start schedule loop: %w", err)
		}
	}

	f.logger.Info("functions component ready")
	return nil
}

func (f *Functions) Stop(ctx *kernel.Context) error {
	if f.cron != nil {
		<-f.cron.Stop().Done()
		f.cron = nil
	}
	return nil
}

func (f *Functions) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/functions", f.handleFunctions)
	mux.HandleFunc("/api/functions/", f.handleFunctionByID)
	return mux
}

func (f *Functions) scanFunction(row interface {
	Scan(dest ...any) error
}) (Function, error) {
	var fn Function
	var envStr string
	err := row.Scan(&fn.ID, &fn.Name, &fn.Runtime, &fn.Source, &fn.Trigger, &fn.Schedule, &envStr, &fn.Timeout, &fn.CreatedAt)
	if err != nil {
		return fn, err
	}
	if err := json.Unmarshal([]byte(envStr), &fn.Env); err != nil {
		f.logger.Warn("unmarshal function env", "error", err)
	}
	if fn.Env == nil {
		fn.Env = map[string]string{}
	}
	return fn, nil
}

// fetchFunctionByIDForOrg fetches a function by ID, scoped to the given org_id.
// Returns sql.ErrNoRows if the function exists but belongs to a different org.
func (f *Functions) fetchFunctionByIDForOrg(ctx context.Context, id string, orgID int64) (Function, error) {
	return f.scanFunction(f.db.QueryRowContext(ctx,
		"SELECT id, name, runtime, source, trigger, schedule, env, timeout, created_at FROM functions WHERE id = ? AND org_id = ?", id, orgID))
}

// startScheduleLoop initializes the cron scheduler and registers all
// functions with trigger=schedule.
func (f *Functions) startScheduleLoop() error {
	f.cron = cron.New(cron.WithSeconds())
	if err := f.reloadSchedule(); err != nil {
		return err
	}
	f.cron.Start()
	f.logger.Info("schedule loop started")
	return nil
}

// reloadSchedule reloads all schedule-triggered functions from the database
// and registers them with the cron scheduler.
func (f *Functions) reloadSchedule() error {
	f.mu.Lock()
	defer f.mu.Unlock()

	// Remove existing entries
	for fnID, entryID := range f.scheduleEntryIDs {
		f.cron.Remove(entryID)
		delete(f.scheduleEntryIDs, fnID)
	}

	// Load schedule functions
	rows, err := f.db.QueryContext(context.Background(),
		"SELECT id, name, runtime, source, trigger, schedule, env, timeout, created_at FROM functions WHERE trigger = 'schedule' AND schedule != ''")
	if err != nil {
		return fmt.Errorf("load scheduled functions: %w", err)
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		fn, err := f.scanFunction(rows)
		if err != nil {
			f.logger.Error("scan scheduled function", "error", err)
			continue
		}
		fnCopy := fn
		entryID, err := f.cron.AddFunc(fn.Schedule, func() {
			f.executeScheduled(fnCopy)
		})
		if err != nil {
			f.logger.Error("register schedule", "fn", fn.Name, "schedule", fn.Schedule, "error", err)
			continue
		}
		f.scheduleEntryIDs[fn.ID] = entryID
	}
	return rows.Err()
}

// executeScheduled runs a schedule-triggered function and logs the execution.
func (f *Functions) executeScheduled(fn Function) {
	rt, ok := f.runtimes[fn.Runtime]
	if !ok {
		f.logger.Error("unsupported runtime for scheduled function", "fn", fn.Name, "runtime", fn.Runtime)
		return
	}

	orgID, _ := auth.OrgIDFromContext(context.Background())
	runCtx := RunContext{
		Env:   fn.Env,
		DB:    f.db,
		OrgID: orgID,
	}
	output, execErr := rt.Execute(fn.Source, fn.Timeout, runCtx)

	execID, saveErr := f.saveExecution(context.Background(), fn.ID, orgID, output, execErr)
	if saveErr != nil {
		f.logger.Error("save scheduled execution", "fn", fn.Name, "error", saveErr)
		return
	}

	if execErr != nil {
		f.logger.Error("scheduled execution failed", "fn", fn.Name, "exec_id", execID, "error", execErr)
	} else {
		f.logger.Info("scheduled execution complete", "fn", fn.Name, "exec_id", execID)
	}
}
