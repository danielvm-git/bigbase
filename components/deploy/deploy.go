package deploy

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/danielvm/bigbase/kernel"
)

const version = "0.1.0"

// killProcessGroup kills a process by PID and also signals its entire process group.
// Sending to both the PID directly and the PGID ensures the process dies even if
// Setpgid wasn't applied (e.g. orphaned processes from a previous session).
func killProcessGroup(pgid int) {
	// Kill the process directly (always works).
	_ = syscall.Kill(pgid, syscall.SIGKILL)
	// Also kill the process group in case child processes exist as a separate group.
	_ = syscall.Kill(-pgid, syscall.SIGKILL)
}

// DBer is an alias for kernel.DBer — the shared database abstraction.
type DBer = kernel.DBer

type AppType string

const (
	AppNode          AppType = "node"
	AppGo            AppType = "go"
	AppPython        AppType = "python"
	AppPHP           AppType = "php"
	AppStatic        AppType = "static"
	AppStaticSidecar AppType = "static-sidecar"
)

// IsValid reports whether the AppType is one of the known application types.
func (a AppType) IsValid() bool {
	switch a {
	case AppNode, AppGo, AppPython, AppPHP, AppStatic, AppStaticSidecar:
		return true
	default:
		return false
	}
}

// AllAppTypes returns all known application types. New types should be added
// to the const block above and this function to maintain OCP compliance.
func AllAppTypes() []AppType {
	return []AppType{AppNode, AppGo, AppPython, AppPHP, AppStatic, AppStaticSidecar}
}

type Deployment struct {
	ID               string             `json:"id"`
	RepoID           string             `json:"repo_id"`
	SiteID           string             `json:"site_id"`
	Branch           string             `json:"branch"`
	CommitSHA        string             `json:"commit_sha"`
	Status           string             `json:"status"`
	URL              string             `json:"url"`
	Port             int                `json:"port"`
	AppType          AppType            `json:"app_type"`
	ErrorMessage     string             `json:"error_message,omitempty"`
	PassthroughPaths []string           `json:"passthrough_paths,omitempty"`
	ManifestPath     string             `json:"manifest_path,omitempty"`
	CreatedAt        string             `json:"created_at"`
	StatusHistory    []StatusTransition `json:"status_history,omitempty"`
	HealthSummary    string             `json:"health_summary,omitempty"`
	PipelineTimeline *PipelineTimeline  `json:"pipeline_timeline,omitempty"`
}

type runningApp struct {
	cmd     *exec.Cmd
	server  *http.Server
	port    int
	buildID string
	host    string
}

// DeploymentHostRegistry routes public hostnames to local deployment ports.
type DeploymentHostRegistry interface {
	RegisterDeploymentHost(host string, port int, siteID string, passthroughPaths []string, metadata map[string]string) error
	UnregisterDeploymentHost(host string)
}

type Deploy struct {
	db                  DBer
	logger              kernel.Logger
	buildsDir           string
	gitDir              string
	buildHome           string
	basePort            int
	publicDomain        string
	useHTTPS            bool
	hostRouter          DeploymentHostRegistry
	logs                *logDeployments
	mu                  sync.Mutex
	nextPort            int
	apps                map[string]*runningApp
	unsubscribe         func()
	logHubsMu           sync.RWMutex
	logHubs             map[string]*logHub
	sm                  *StateMachine
	eventBus            *kernel.EventBus
	supervisor          *Supervisor
	envKey              []byte
	cache               *Cache
	DrainTimeout        time.Duration
	oldDeploymentsMu    sync.RWMutex
	oldDeployments      map[string][]string
	dbDriver            string
	dbDSN               string
	diagnosisReader     DeployDiagnosisReader
	relatedEventsReader DeployRelatedEventsReader
	envResolver         *EnvResolver
}

var _ kernel.Component = (*Deploy)(nil)

type Options struct {
	DB           DBer
	Logger       kernel.Logger
	BuildsDir    string
	GitDir       string
	BuildHome    string
	BasePort     int
	PublicDomain string
	UseHTTPS     bool
	HostRouter   DeploymentHostRegistry
	// EncryptionKey is the canonical, already-validated root key. The legacy
	// EnvEncryptionKey is retained only for explicit migration/test inputs.
	EncryptionKey    []byte
	EnvEncryptionKey string
	AllowPlaintext   bool
	// Secrets is the native SecretManager seam (e89s03) used for Project
	// Environment resolution. Nil disables native resolution (legacy-only
	// deployments still resolve through the Site compatibility layer).
	Secrets SecretResolver
	// CacheDir is the directory for build dependency archives (default: data/cache/deploy).
	CacheDir string
	// CacheMaxSize is the maximum cache size in bytes (default: 2 GiB).
	CacheMaxSize int64
	// Runner is injected for tests; nil uses the production deployRunner.
	Runner Runner
	// DrainTimeout is how long to wait for existing connections to complete
	// during zero-downtime deployment drain. Default: 30s.
	DrainTimeout time.Duration
	// DBDriver is the active database driver (sqlite or postgres).
	DBDriver string
	// DBDSN is the platform database connection string passed to deployed apps.
	DBDSN string
}

func New(opts Options) *Deploy {
	logger := opts.Logger
	if logger == nil {
		logger = kernel.NoopLogger{}
	}
	dir := opts.BuildsDir
	if dir == "" {
		dir = "data/builds"
	}
	absDir, err := filepath.Abs(dir)
	if err == nil {
		dir = absDir
	}
	gitDir := opts.GitDir
	if gitDir == "" {
		gitDir = "data/git"
	}
	absGitDir, err := filepath.Abs(gitDir)
	if err == nil {
		gitDir = absGitDir
	}
	basePort := opts.BasePort
	if basePort == 0 {
		basePort = 10000
	}
	useHTTPS := opts.UseHTTPS
	if strings.TrimSpace(opts.PublicDomain) != "" {
		useHTTPS = true
	}
	envKey := opts.EncryptionKey
	if len(envKey) == 0 && opts.EnvEncryptionKey != "" {
		envKey, _ = parseEnvEncryptionKey(opts.EnvEncryptionKey)
	}

	drainTimeout := opts.DrainTimeout
	if drainTimeout <= 0 {
		drainTimeout = 30 * time.Second
	}
	d := &Deploy{
		db:             opts.DB,
		logger:         logger,
		buildsDir:      dir,
		gitDir:         gitDir,
		buildHome:      strings.TrimSpace(opts.BuildHome),
		basePort:       basePort,
		publicDomain:   strings.TrimSpace(opts.PublicDomain),
		useHTTPS:       useHTTPS,
		hostRouter:     opts.HostRouter,
		nextPort:       basePort,
		apps:           make(map[string]*runningApp),
		logHubs:        make(map[string]*logHub),
		logs:           &logDeployments{},
		sm:             NewStateMachine(),
		DrainTimeout:   drainTimeout,
		oldDeployments: make(map[string][]string),
		dbDriver:       opts.DBDriver,
		dbDSN:          opts.DBDSN,
	}
	if len(envKey) > 0 {
		d.envKey = append([]byte(nil), envKey...)
	}
	// Single owner of env resolution + redaction (issue #41). Both the build
	// and runtime paths resolve through it, sharing one precedence definition.
	d.envResolver = NewEnvResolver(d.db, d.envKey, d.logger, d.buildHome, opts.Secrets)
	cacheDir := opts.CacheDir
	if cacheDir == "" {
		cacheDir = "data/cache/deploy"
	}
	if abs, err := filepath.Abs(cacheDir); err == nil {
		cacheDir = abs
	}
	d.cache = NewCache(cacheDir, opts.CacheMaxSize)

	runner := opts.Runner
	if runner == nil {
		runner = &deployRunner{db: d.db}
	}
	d.supervisor = NewSupervisor(runner, &wallClock{}, opts.HostRouter,
		// Crash-loop trip is a terminal failure: record a reason even when the
		// app-exit handler hasn't written a specific one yet (previously the
		// row could reach status=failed with an empty error_message — the
		// TestStaticSidecar_BuildAndStart flake), then transition.
		func(id string) {
			_, _ = d.db.ExecContext(context.Background(),
				"UPDATE deployments SET error_message = COALESCE(NULLIF(error_message,''), 'Application crashed too many times (crash-loop)') WHERE id = ?",
				id)
			d.updateStatus(id, "failed")
		},
		func(name string, _ Spec) { d.logger.Info("supervisor event", "event", name) },
	)
	return d
}
func (d *Deploy) SetDiagnosisReader(r DeployDiagnosisReader) {
	d.diagnosisReader = r
}
func (d *Deploy) SetRelatedEventsReader(r DeployRelatedEventsReader) {
	d.relatedEventsReader = r
}
func (d *Deploy) Name() string           { return "deploy" }
func (d *Deploy) Version() string        { return version }
func (d *Deploy) Dependencies() []string { return []string{"db", "git"} }

func (d *Deploy) EventBus(bus *kernel.EventBus) {
	d.eventBus = bus
}
func (d *Deploy) Init(ctx *kernel.Context, config json.RawMessage) error {
	return nil
}
func (d *Deploy) Start(ctx *kernel.Context) error {
	if ctx != nil && ctx.Kernel != nil {
		d.eventBus = ctx.Kernel.EventBus()
	}
	if err := os.MkdirAll(d.buildsDir, 0755); err != nil {
		return fmt.Errorf("create builds dir: %w", err)
	}
	if err := d.db.Migrate(`CREATE TABLE IF NOT EXISTS deployments (
		id TEXT PRIMARY KEY,
		repo_id TEXT NOT NULL,
		site_id TEXT NOT NULL DEFAULT '',
		branch TEXT NOT NULL DEFAULT 'main',
		commit_sha TEXT DEFAULT '',
		status TEXT NOT NULL DEFAULT 'pending',
		url TEXT DEFAULT '',
		port INTEGER DEFAULT 0,
		app_type TEXT DEFAULT '',
		error_message TEXT NOT NULL DEFAULT '',
		build_log TEXT DEFAULT '',
		manifest_path TEXT DEFAULT '',
		created_at TEXT NOT NULL DEFAULT (datetime('now'))
	)`); err != nil {
		return fmt.Errorf("migrate deployments table: %w", err)
	}
	if err := d.ensureErrorMessageColumn(); err != nil {
		return err
	}
	if err := d.ensureBuildLogColumn(); err != nil {
		return err
	}
	if err := d.ensureSiteIDColumn(); err != nil {
		return err
	}
	if err := d.ensurePassthroughPathsColumn(); err != nil {
		return err
	}
	if err := d.ensurePIDColumn(); err != nil {
		return err
	}
	if err := d.ensureStatusHistoryColumn(); err != nil {
		return err
	}
	if err := d.ensureManifestPathColumn(); err != nil {
		return err
	}
	if err := d.ensureHealthSummaryColumn(); err != nil {
		return err
	}
	if err := d.ensurePipelineTimelineColumn(); err != nil {
		return err
	}
	if err := d.ensureRelatedEventsSnapshotColumn(); err != nil {
		return err
	}
	if err := d.ensureDeploymentsRepoCreatedIndex(); err != nil {
		return err
	}
	if err := d.ensureRollbackEventsTable(); err != nil {
		return fmt.Errorf("migrate rollback_events table: %w", err)
	}
	if err := d.db.Migrate(`CREATE TABLE IF NOT EXISTS site_request_logs (
		id TEXT PRIMARY KEY,
		site_id TEXT NOT NULL,
		method TEXT NOT NULL,
		path TEXT NOT NULL,
		status INTEGER NOT NULL DEFAULT 0,
		duration_ms INTEGER NOT NULL DEFAULT 0,
		created_at TEXT NOT NULL DEFAULT (datetime('now'))
	)`); err != nil {
		return fmt.Errorf("migrate site_request_logs table: %w", err)
	}
	if err := d.db.Migrate(`CREATE INDEX IF NOT EXISTS idx_srl_site_id ON site_request_logs(site_id, created_at DESC)`); err != nil {
		return fmt.Errorf("migrate site_request_logs index: %w", err)
	}
	if err := d.ensureCacheConfigTable(); err != nil {
		return fmt.Errorf("migrate deploy_cache_config table: %w", err)
	}
	d.loadCacheConfig()
	d.restoreRunningDeploymentHosts()
	candidates := d.loadResumeCandidates()
	d.logger.Info("deploy component ready", "dir", d.buildsDir, "resume", len(candidates))
	go d.resumeCandidates(candidates)
	return nil
}
func (d *Deploy) Stop(ctx *kernel.Context) error {
	if d.unsubscribe != nil {
		d.unsubscribe()
	}
	d.mu.Lock()
	defer d.mu.Unlock()

	for id, app := range d.apps {
		if app.host != "" && d.hostRouter != nil {
			d.hostRouter.UnregisterDeploymentHost(app.host)
		}
		// Shutdown static HTTP servers
		if app.server != nil {
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			_ = app.server.Shutdown(shutdownCtx)
			cancel()
		}
		// Kill process-based deployments and their entire process tree.
		if app.cmd != nil && app.cmd.Process != nil {
			killProcessGroup(app.cmd.Process.Pid)
		}
		delete(d.apps, id)
	}
	return nil
}
