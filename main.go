package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"text/tabwriter"

	"github.com/danielvm/bigbase/components/admin"
	"github.com/danielvm/bigbase/components/api"
	"github.com/danielvm/bigbase/components/auth"
	"github.com/danielvm/bigbase/components/backup"
	"github.com/danielvm/bigbase/components/cici"
	"github.com/danielvm/bigbase/components/db"
	"github.com/danielvm/bigbase/components/deploy"
	"github.com/danielvm/bigbase/components/forge"
	"github.com/danielvm/bigbase/components/functions"
	"github.com/danielvm/bigbase/components/git"
	"github.com/danielvm/bigbase/components/github"
	"github.com/danielvm/bigbase/components/mcp"
	"github.com/danielvm/bigbase/components/messaging"
	"github.com/danielvm/bigbase/components/monitoring"
	"github.com/danielvm/bigbase/components/proxy"
	"github.com/danielvm/bigbase/components/realtime"
	"github.com/danielvm/bigbase/components/sites"
	"github.com/danielvm/bigbase/components/storage"
	"github.com/danielvm/bigbase/config"
	"github.com/danielvm/bigbase/kernel"
)

var (
	version = kernel.Version
)

// parseLogLevel converts a case-insensitive log level string to slog.Level.
// Returns an error for unknown or empty values.
// parseCORSOrigins splits a comma-separated string of origins into a slice.
// Empty input returns nil (CORS disabled).
func parseCORSOrigins(raw string) []string {
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	origins := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			origins = append(origins, p)
		}
	}
	return origins
}

func parseLogLevel(level string) (slog.Level, error) {
	switch level {
	case "debug", "DEBUG":
		return slog.LevelDebug, nil
	case "info", "INFO":
		return slog.LevelInfo, nil
	case "warn", "WARN":
		return slog.LevelWarn, nil
	case "error", "ERROR":
		return slog.LevelError, nil
	case "":
		return slog.LevelInfo, fmt.Errorf("empty log level")
	default:
		return slog.LevelInfo, fmt.Errorf("unknown log level: %q (valid: debug, info, warn, error)", level)
	}
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		return
	}

	cmd := os.Args[1]
	switch cmd {
	case "version":
		fmt.Println("bigbase version", version)
		return
	case "help", "--help", "-h":
		printUsage()
		return
	case "serve":
		startProxy()
		return
	case "backup":
		runBackup()
		return
	case "restore":
		runRestore()
		return
	case "migrate":
		runMigrate()
		return
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
	k := kernel.New(logger)

	// Register all components for discovery (CLI listing)
	k.Register(mcp.New(mcp.Options{}))

	if err := k.Start(); err != nil {
		logger.Error("failed to start kernel", "error", err)
		os.Exit(1)
	}

	switch cmd {
	case "status":
		printStatus(k)
	case "components":
		if len(os.Args) < 3 || os.Args[2] != "list" {
			fmt.Fprintln(os.Stderr, "Usage: bigbase components list")
			os.Exit(1)
		}
		printComponents(k)
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", cmd)
		printUsage()
		os.Exit(1)
	}
}

func startProxy() {
	serveFS := flag.NewFlagSet("serve", flag.ContinueOnError)
	port := serveFS.String("port", "8080", "HTTP server port")
	dbPath := serveFS.String("db", "bigbase.db", "SQLite database path (legacy; use --db-dsn)")
	dbDriver := serveFS.String("db-driver", "", "Database driver: sqlite (default) or postgres")
	dbDSN := serveFS.String("db-dsn", "", "Database DSN (file path for sqlite, connection URL for postgres)")
	googleClientID := serveFS.String("google-client-id", "", "Google OAuth client ID")
	googleClientSecret := serveFS.String("google-client-secret", "", "Google OAuth client secret")
	githubAppID := serveFS.String("github-app-id", "", "GitHub App ID")
	githubAppSlug := serveFS.String("github-app-slug", "", "GitHub App slug")
	githubPrivateKeyPath := serveFS.String("github-app-private-key-path", "", "GitHub App private key path")
	githubWebhookSecret := serveFS.String("github-webhook-secret", "", "GitHub App webhook secret")
	sitesDomain := serveFS.String("sites-domain", "", "Parent domain for deployed site subdomains (e.g. bigbase.click)")
	logLevel := serveFS.String("log-level", "info", "Log level: debug, info, warn, error")
	corsOrigins := serveFS.String("cors-allowed-origins", "", "Comma-separated list of allowed CORS origins (empty = CORS disabled)")
	postLoginRedirect := serveFS.String("auth-post-login-redirect", "/admin/", "Post-login redirect URL for OAuth callbacks")
	spaOriginAllowlist := serveFS.String("auth-spa-origin-allowlist", "", "Comma-separated list of allowed SPA origins for OAuth token delivery (empty = disabled)")
	mcpDisabled := serveFS.Bool("mcp-disabled", false, "Disable MCP server")
	mcpPort := serveFS.Int("mcp-port", 3900, "MCP server HTTP port")
	mcpTransport := serveFS.String("mcp-transport", "http", "MCP transport (stdio, http)")
	_ = serveFS.Parse(os.Args[2:])

	googleID := config.FlagOrEnv(*googleClientID, "GOOGLE_CLIENT_ID")
	googleSecret := config.FlagOrEnv(*googleClientSecret, "GOOGLE_CLIENT_SECRET")
	ghAppID := config.FlagOrEnv(*githubAppID, "GITHUB_APP_ID")
	ghAppSlug := config.FlagOrEnv(*githubAppSlug, "GITHUB_APP_SLUG")
	ghPrivateKeyPath := config.FlagOrEnv(*githubPrivateKeyPath, "GITHUB_APP_PRIVATE_KEY_PATH")
	ghWebhookSecret := config.FlagOrEnv(*githubWebhookSecret, "GITHUB_WEBHOOK_SECRET")
	sitesDomainVal := config.FlagOrEnv(*sitesDomain, "BIGBASE_SITES_DOMAIN")
	dbDriverVal := config.FlagOrEnv(*dbDriver, "BIGBASE_DB_DRIVER")
	dbDSNVal := config.FlagOrEnv(*dbDSN, "BIGBASE_DB_DSN")

	level, err := parseLogLevel(*logLevel)
	if err != nil {
		logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
		logger.Error("invalid log level, defaulting", "provided", *logLevel, "error", err)
		level = slog.LevelInfo
	}

	// Parse CORS allowed origins (empty = CORS disabled, default safe).
	corsAllowedOrigins := parseCORSOrigins(*corsOrigins)

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level}))
	k := kernel.New(logger)

	p := proxy.New(proxy.Options{
		Port:               *port,
		Kernel:             k,
		Logger:             logger,
		CORSAllowedOrigins: corsAllowedOrigins,
	})
	d := db.New(db.Options{
		Driver: dbDriverVal,
		DSN:    dbDSNVal,
		Path:   *dbPath, // legacy fallback when --db-dsn not provided
		Logger: logger,
	})
	a := api.New(api.Options{
		DB:     d,
		Logger: logger,
	})
	authComp := auth.New(auth.Options{
		DB:                 d,
		Logger:             logger,
		GoogleClientID:     googleID,
		GoogleClientSecret: googleSecret,
		CORSAllowedOrigins: corsAllowedOrigins,
		PostLoginRedirect:  *postLoginRedirect,
		SPAOriginAllowlist: parseCORSOrigins(*spaOriginAllowlist),
	})

	ad := admin.New(admin.Options{Logger: logger})
	s := storage.New(storage.Options{DB: d, Logger: logger})
	g := git.New(git.Options{DB: d, Logger: logger})
	f := forge.New(forge.Options{DB: d, Logger: logger})
	ci := cici.New(cici.Options{DB: d, Logger: logger})
	fn := functions.New(functions.Options{DB: d, Logger: logger})
	msgComp := messaging.New(messaging.Options{
		DB:     d,
		Logger: logger,
	})
	depComp := deploy.New(deploy.Options{
		DB:           d,
		Logger:       logger,
		BuildHome:    os.Getenv("BIGBASE_HOME"),
		PublicDomain: sitesDomainVal,
		HostRouter:   p,
	})
	p.SetRequestLogger(depComp)
	mComp := monitoring.New(monitoring.Options{
		DB:     d,
		Logger: logger,
	})
	gh := github.New(github.Options{
		DB:             d,
		Logger:         logger,
		AppID:          ghAppID,
		AppSlug:        ghAppSlug,
		PrivateKeyPath: ghPrivateKeyPath,
		WebhookSecret:  ghWebhookSecret,
	})
	st := sites.New(sites.Options{
		DB:     d,
		Logger: logger,
		TriggerDeploy: func(ctx context.Context, repoID, branch, siteName, siteID string) (*sites.Deployment, error) {
			dep, err := depComp.Trigger(ctx, repoID, branch, siteName, siteID)
			if err != nil {
				return nil, err
			}
			return &sites.Deployment{
				ID:        dep.ID,
				RepoID:    dep.RepoID,
				Branch:    dep.Branch,
				CommitSHA: dep.CommitSHA,
				Status:    dep.Status,
				URL:       dep.URL,
				Port:      dep.Port,
				AppType:   string(dep.AppType),
				CreatedAt: dep.CreatedAt,
			}, nil
		},
		DeleteSiteCleanup: func(ctx context.Context, siteID, repoID string) error {
			return depComp.DeleteSiteDeployments(ctx, siteID, repoID)
		},
	})
	rt := realtime.New(realtime.Options{
		Logger: logger,
		Validate: func(token string) (int64, error) {
			claims, err := authComp.ValidateToken(token)
			if err != nil {
				return 0, err
			}
			return claims.UserID, nil
		},
	})
	k.Register(p)
	k.Register(d)
	k.Register(a)
	k.Register(authComp)
	k.Register(ad)
	k.Register(s)
	k.Register(g)
	k.Register(f)
	k.Register(gh)
	k.Register(st)
	k.Register(ci)
	k.Register(fn)
	k.Register(rt)
	k.Register(msgComp)
	k.Register(depComp)
	k.Register(mComp)

	mcpComp := mcp.New(mcp.Options{
		Logger:    logger,
		Enabled:   !*mcpDisabled,
		Port:      *mcpPort,
		Transport: *mcpTransport,
	})
	k.Register(mcpComp)

	// orgBridge reads auth.OrgIDFromContext and auth.UserRoleFromContext (set by
	// authComp.Middleware) and bridges them to api.WithOrgID / api.WithUserRole so
	// API handlers see the caller's org and role.
	orgBridge := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if orgID, ok := auth.OrgIDFromContext(r.Context()); ok {
				r = r.WithContext(api.WithOrgID(r.Context(), orgID))
			}
			if role, ok := auth.UserRoleFromContext(r.Context()); ok {
				r = r.WithContext(api.WithUserRole(r.Context(), role))
			}
			next.ServeHTTP(w, r)
		})
	}

	// Register routes before kernel.Start to avoid race on proxy mux
	publicAPI := a.Handler()
	protectedAPI := mComp.Middleware(authComp.Middleware(orgBridge(publicAPI)))
	storageHandler := mComp.Middleware(authComp.Middleware(s.Handler()))
	gitHandler := mComp.Middleware(authComp.Middleware(g.Handler()))
	forgeHandler := mComp.Middleware(authComp.Middleware(f.Handler()))
	githubPublic := mComp.Middleware(gh.PublicHandler())
	githubProtected := mComp.Middleware(authComp.Middleware(gh.ProtectedHandler()))
	sitesHandler := mComp.Middleware(authComp.Middleware(st.Handler()))

	p.Handle("/api/collections/", protectedAPI.ServeHTTP)
	// /api/sql requires admin role — RequireAdmin runs after auth.Middleware sets the role.
	sqlHandler := mComp.Middleware(authComp.Middleware(auth.RequireAdmin(orgBridge(publicAPI))))
	p.Handle("/api/sql", sqlHandler.ServeHTTP)
	p.Handle("/api/storage/upload", storageHandler.ServeHTTP)
	p.Handle("/api/storage/files/", storageHandler.ServeHTTP)
	p.Handle("/api/storage/files", storageHandler.ServeHTTP)
	p.Handle("/api/git/repos", gitHandler.ServeHTTP)
	p.Handle("/api/git/repos/", gitHandler.ServeHTTP)
	p.Handle("/api/forge/", forgeHandler.ServeHTTP)
	p.Handle("/api/github/callback", githubPublic.ServeHTTP)
	p.Handle("/api/github/webhook", githubPublic.ServeHTTP)
	p.Handle("/api/github/", githubProtected.ServeHTTP)
	p.Handle("/api/sites", sitesHandler.ServeHTTP)
	p.Handle("/api/sites/", sitesHandler.ServeHTTP)
	p.Handle("/api/cici/", mComp.Middleware(authComp.Middleware(ci.Handler())).ServeHTTP)
	p.Handle("/api/functions/", mComp.Middleware(authComp.Middleware(fn.Handler())).ServeHTTP)
	p.Handle("/api/functions", mComp.Middleware(authComp.Middleware(fn.Handler())).ServeHTTP)
	p.Handle("/api/messaging/", mComp.Middleware(authComp.Middleware(msgComp.Handler())).ServeHTTP)
	p.Handle("/api/deploy", mComp.Middleware(authComp.Middleware(depComp.Handler())).ServeHTTP)
	p.Handle("/api/deploy/", mComp.Middleware(authComp.Middleware(depComp.Handler())).ServeHTTP)
	p.Handle("/realtime", mComp.Middleware(rt.Handler()).ServeHTTP)
	p.Handle("/api/auth/", mComp.Middleware(authComp.Handler()).ServeHTTP)
	p.Handle("GET /api/auth/users", mComp.Middleware(authComp.ProtectedHandler()).ServeHTTP)
	p.Handle("GET /api/auth/me", mComp.Middleware(authComp.ProtectedHandler()).ServeHTTP)
	p.Handle("DELETE /api/auth/users/", mComp.Middleware(authComp.ProtectedHandler()).ServeHTTP)
	// Org routes
	p.Handle("POST /api/orgs", mComp.Middleware(authComp.ProtectedHandler()).ServeHTTP)
	p.Handle("GET /api/orgs", mComp.Middleware(authComp.ProtectedHandler()).ServeHTTP)
	p.Handle("GET /api/orgs/", mComp.Middleware(authComp.ProtectedHandler()).ServeHTTP)
	p.Handle("PATCH /api/orgs/", mComp.Middleware(authComp.ProtectedHandler()).ServeHTTP)
	p.Handle("DELETE /api/orgs/", mComp.Middleware(authComp.ProtectedHandler()).ServeHTTP)
	p.Handle("/api/monitoring/health", mComp.Handler().ServeHTTP)
	p.Handle("/api/monitoring/metrics", authComp.Middleware(mComp.Handler()).ServeHTTP)
	p.Handle("/api/monitoring/metrics/prometheus", mComp.Handler().ServeHTTP)
	p.Handle("/api/monitoring/logs", authComp.Middleware(mComp.Handler()).ServeHTTP)
	p.Handle("/api/monitoring/logs/", authComp.Middleware(mComp.Handler()).ServeHTTP)
	p.Handle("/api/monitoring/alerts", authComp.Middleware(mComp.Handler()).ServeHTTP)
	p.Handle("/admin/", http.StripPrefix("/admin/", ad.Handler()).ServeHTTP)

	spaPaths := []string{"/repos", "/deploy", "/deploy/new", "/messaging", "/storage", "/functions", "/forge", "/cici", "/monitoring", "/data", "/sql", "/users", "/login"}
	for _, sp := range spaPaths {
		path := sp
		p.Handle("GET "+path, func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, "/admin/#"+path, http.StatusFound)
		})
	}

	if err := k.Start(); err != nil {
		logger.Error("failed to start kernel", "error", err)
		os.Exit(1)
	}

	effectiveDSN := dbDSNVal
	if effectiveDSN == "" {
		effectiveDSN = *dbPath
	}
	logger.Info("bigbase running", "port", *port, "db-driver", dbDriverVal, "dsn", effectiveDSN)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	logger.Info("shutting down")

	_ = p.Stop(&kernel.Context{})
	_ = k.Stop()
	os.Exit(0)
}

func printUsage() {
	fmt.Println(`bigbase - BigBase BaaS Platform

Usage:
  bigbase version                               Show version
  bigbase status                                Show kernel and component status
  bigbase components list                       List registered components
  bigbase serve [--port PORT] [--db PATH]       Start HTTP server (default :8080)
  bigbase backup --db PATH --output FILE        Dump database to SQL file
  bigbase restore --input FILE --db PATH        Replay SQL dump into database
  bigbase migrate up|down|status [--db PATH]    Run database migrations
  bigbase help                                  Show this help`)
}

func printStatus(k *kernel.Kernel) {
	fmt.Println("Kernel Status")
	fmt.Println("=============")
	fmt.Println("Version:", version)
	fmt.Println("Components:", len(k.Components()))
	bus := k.EventBus()
	if bus != nil {
		fmt.Println("Subscriptions:", bus.SubscriberCount())
	}
	fmt.Println()

	components := k.ListComponents()
	printTable(components,
		"COMPONENT\tVERSION\tSTATUS\tHOOKS",
		func(c kernel.ComponentStatus) string {
			status := "stopped"
			if c.Running {
				status = "running"
			}
			return fmt.Sprintf("%s\t%s\t%s\t%s", c.Name, c.Version, status, joinOrNone(c.Hooks))
		},
	)
}

func printComponents(k *kernel.Kernel) {
	printTable(k.ListComponents(),
		"NAME\tVERSION\tDEPENDENCIES\tHOOKS",
		func(c kernel.ComponentStatus) string {
			return fmt.Sprintf("%s\t%s\t%s\t%s", c.Name, c.Version, joinOrNone(c.Dependencies), joinOrNone(c.Hooks))
		},
	)
}

func printTable(components []kernel.ComponentStatus, header string, row func(kernel.ComponentStatus) string) {
	if len(components) == 0 {
		fmt.Println("No components registered.")
		return
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, header)
	for _, c := range components {
		_, _ = fmt.Fprintln(w, row(c))
	}
	_ = w.Flush()
}

func runBackup() {
	fs := flag.NewFlagSet("backup", flag.ExitOnError)
	dbPath := fs.String("db", "bigbase.db", "SQLite database path")
	output := fs.String("output", "", "Output SQL file path (required)")
	_ = fs.Parse(os.Args[2:])

	if *output == "" {
		fmt.Fprintln(os.Stderr, "error: --output is required")
		fs.Usage()
		os.Exit(1)
	}

	sqldb, err := openSQLite(*dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: open db: %v\n", err)
		os.Exit(1)
	}
	defer func() { _ = sqldb.Close() }()

	f, err := os.Create(*output)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: create output file: %v\n", err)
		os.Exit(1)
	}
	defer func() { _ = f.Close() }()

	if err := backup.Dump(context.Background(), sqldb, f); err != nil {
		fmt.Fprintf(os.Stderr, "error: dump: %v\n", err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stdout, "backup written to %s\n", *output)
}

func runRestore() {
	fs := flag.NewFlagSet("restore", flag.ExitOnError)
	input := fs.String("input", "", "Input SQL dump file (required)")
	dbPath := fs.String("db", "bigbase.db", "SQLite database path")
	_ = fs.Parse(os.Args[2:])

	if *input == "" {
		fmt.Fprintln(os.Stderr, "error: --input is required")
		fs.Usage()
		os.Exit(1)
	}

	data, err := os.ReadFile(*input)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: read input: %v\n", err)
		os.Exit(1)
	}

	sqldb, err := openSQLite(*dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: open db: %v\n", err)
		os.Exit(1)
	}
	defer func() { _ = sqldb.Close() }()

	if err := backup.Restore(context.Background(), sqldb, string(data)); err != nil {
		fmt.Fprintf(os.Stderr, "error: restore: %v\n", err)
		os.Exit(1)
	}
	fmt.Fprintln(os.Stdout, "restore complete")
}

func runMigrate() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "Usage: bigbase migrate up|down|status [--db PATH]")
		os.Exit(1)
	}
	subCmd := os.Args[2]
	fs := flag.NewFlagSet("migrate", flag.ExitOnError)
	dbPath := fs.String("db", "bigbase.db", "SQLite database path")
	_ = fs.Parse(os.Args[3:])

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
	k := kernel.New(logger)
	d := db.New(db.Options{Path: *dbPath, Logger: logger})
	k.Register(d)
	if err := k.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "error: start db: %v\n", err)
		os.Exit(1)
	}
	defer func() { _ = k.Stop() }()

	m := backup.NewMigrationRunner(d.DB(), "migrations")
	switch subCmd {
	case "up":
		if err := m.Up(context.Background()); err != nil {
			fmt.Fprintf(os.Stderr, "error: migrate up: %v\n", err)
			os.Exit(1)
		}
		fmt.Fprintln(os.Stdout, "migrations applied")
	case "down":
		if err := m.Down(context.Background()); err != nil {
			fmt.Fprintf(os.Stderr, "error: migrate down: %v\n", err)
			os.Exit(1)
		}
		fmt.Fprintln(os.Stdout, "migrations rolled back")
	case "status":
		ver, dirty, err := m.Status(context.Background())
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: migrate status: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("version: %d  dirty: %v\n", ver, dirty)
	default:
		fmt.Fprintf(os.Stderr, "unknown migrate subcommand: %s\n", subCmd)
		os.Exit(1)
	}
}

func openSQLite(path string) (backup.SQLiteDB, error) {
	return backup.OpenSQLite(path)
}

func joinOrNone(items []string) string {
	if len(items) == 0 {
		return "none"
	}
	return strings.Join(items, ", ")
}
