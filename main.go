package main

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"text/tabwriter"
	"time"

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
	"github.com/danielvm/bigbase/components/projects"
	"github.com/danielvm/bigbase/components/proxy"
	"github.com/danielvm/bigbase/components/realtime"
	"github.com/danielvm/bigbase/components/secrets"
	"github.com/danielvm/bigbase/components/sites"
	"github.com/danielvm/bigbase/components/storage"
	"github.com/danielvm/bigbase/config"
	"github.com/danielvm/bigbase/kernel"
	"github.com/newrelic/go-agent/v3/integrations/logcontext-v2/nrslog"
	"github.com/newrelic/go-agent/v3/newrelic"
)

var (
	version = kernel.Version
	nrApp   *newrelic.Application
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

// buildHandler returns the slog handler for the application logger. When a New
// Relic application is present, logs are routed through nrslog so they are
// forwarded to New Relic Logs and correlated with the active APM transaction.
// When nrApp is nil (New Relic disabled), it returns a plain JSON handler so
// local/dev behaviour is unchanged.
func buildHandler(nrApp *newrelic.Application, w io.Writer, opts *slog.HandlerOptions) slog.Handler {
	base := slog.NewJSONHandler(w, opts)
	if nrApp == nil {
		return base
	}
	return nrslog.WrapHandler(nrApp, base)
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
	case "deploy":
		runDeployCmd()
		return
	case "init":
		runInitCmd()
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
	publicURL := serveFS.String("public-url", "", "Public base URL for OAuth redirects (env: BIGBASE_PUBLIC_URL)")
	githubAppID := serveFS.String("github-app-id", "", "GitHub App ID")
	githubAppSlug := serveFS.String("github-app-slug", "", "GitHub App slug")
	githubPrivateKeyPath := serveFS.String("github-app-private-key-path", "", "GitHub App private key path")
	githubWebhookSecret := serveFS.String("github-webhook-secret", "", "GitHub App webhook secret")
	sitesDomain := serveFS.String("sites-domain", "", "Parent domain for deployed site subdomains (e.g. bigbase.click)")
	logLevel := serveFS.String("log-level", "info", "Log level: debug, info, warn, error")
	corsOrigins := serveFS.String("cors-allowed-origins", "", "Comma-separated list of allowed CORS origins (empty = CORS disabled)")
	postLoginRedirect := serveFS.String("auth-post-login-redirect", "/admin/", "Post-login redirect URL for OAuth callbacks")
	spaOriginAllowlist := serveFS.String("auth-spa-origin-allowlist", "", "Comma-separated list of allowed SPA origins for OAuth token delivery (empty = disabled)")
	jwtAccessExpiry := serveFS.String("jwt-access-expiry", "", "JWT access token lifetime (e.g. 24h, 30m). Default: 24h (env: BIGBASE_JWT_ACCESS_EXPIRY)")
	jwtRefreshExpiry := serveFS.String("jwt-refresh-expiry", "", "JWT refresh token lifetime (e.g. 720h). Default: 720h (env: BIGBASE_JWT_REFRESH_EXPIRY)")
	rateLimitEnabled := serveFS.Bool("rate-limit-enabled", true, "Enable rate limiting on auth endpoints (env: BIGBASE_RATE_LIMIT_ENABLED)")
	rateLimitIPMaxStr := serveFS.String("rate-limit-ip-max", "60", "Max requests per minute per IP (env: BIGBASE_RATE_LIMIT_IP_MAX)")
	rateLimitUserMaxStr := serveFS.String("rate-limit-user-max", "300", "Max requests per minute per authenticated user (env: BIGBASE_RATE_LIMIT_USER_MAX)")
	mcpDisabled := serveFS.Bool("mcp-disabled", false, "Disable MCP server")
	mcpPort := serveFS.Int("mcp-port", 3900, "MCP server HTTP port")
	mcpTransport := serveFS.String("mcp-transport", "http", "MCP transport (stdio, http)")
	nrLicenseKey := serveFS.String("newrelic-license-key", "", "New Relic license key (env: NEW_RELIC_LICENSE_KEY)")
	nrAppName := serveFS.String("newrelic-app-name", "BigBase", "New Relic application name (env: NEW_RELIC_APP_NAME)")
	nrEnabled := serveFS.Bool("newrelic-enabled", true, "Enable New Relic agent (env: NEW_RELIC_ENABLED)")
	_ = serveFS.Parse(os.Args[2:])

	googleID := config.FlagOrEnv(*googleClientID, "GOOGLE_CLIENT_ID")
	googleSecret := config.FlagOrEnv(*googleClientSecret, "GOOGLE_CLIENT_SECRET")
	ghAppID := config.FlagOrEnv(*githubAppID, "GITHUB_APP_ID")
	ghAppSlug := config.FlagOrEnv(*githubAppSlug, "GITHUB_APP_SLUG")
	ghPrivateKeyPath := config.FlagOrEnv(*githubPrivateKeyPath, "GITHUB_APP_PRIVATE_KEY_PATH")
	ghWebhookSecret := config.FlagOrEnv(*githubWebhookSecret, "GITHUB_WEBHOOK_SECRET")
	sitesDomainVal := config.FlagOrEnv(*sitesDomain, "BIGBASE_SITES_DOMAIN")
	dbDriverVal := config.FlagOrEnv(*dbDriver, "BIGBASE_DB_DRIVER")
	rootKeyRaw := config.FlagOrEnv("", "BIGBASE_ROOT_ENCRYPTION_KEY")
	runtimeEnv := strings.ToLower(config.FlagOrEnv("production", "BIGBASE_ENV"))
	allowPlaintext := runtimeEnv == "development" && config.FlagOrEnvBool(false, "BIGBASE_ALLOW_PLAINTEXT_SECRETS")
	var encryptionKey []byte
	if rootKeyRaw == "" {
		if !allowPlaintext {
			fmt.Fprintln(os.Stderr, "configuration error: BIGBASE_ROOT_ENCRYPTION_KEY is required")
			os.Exit(1)
		}
	} else {
		var keyErr error
		encryptionKey, keyErr = sites.ParseRootEncryptionKey(rootKeyRaw)
		if keyErr != nil {
			fmt.Fprintln(os.Stderr, "configuration error: site encryption configuration is invalid")
			os.Exit(1)
		}
	}
	dbDSNVal := config.FlagOrEnv(*dbDSN, "BIGBASE_DB_DSN")

	// JWT expiry config with env var fallbacks.
	accessExpiryStr := config.FlagOrEnv(*jwtAccessExpiry, "BIGBASE_JWT_ACCESS_EXPIRY")
	refreshExpiryStr := config.FlagOrEnv(*jwtRefreshExpiry, "BIGBASE_JWT_REFRESH_EXPIRY")
	var accessExpiry, refreshExpiry time.Duration
	if accessExpiryStr != "" {
		if d, parseErr := time.ParseDuration(accessExpiryStr); parseErr != nil || d <= 0 {
			slog.Warn("invalid jwt-access-expiry, using default 24h", "value", accessExpiryStr)
		} else {
			accessExpiry = d
		}
	}
	if refreshExpiryStr != "" {
		if d, parseErr := time.ParseDuration(refreshExpiryStr); parseErr != nil || d <= 0 {
			slog.Warn("invalid jwt-refresh-expiry, using default 720h", "value", refreshExpiryStr)
		} else {
			refreshExpiry = d
		}
	}

	// Rate limit config with env var fallbacks (BIGBASE_ prefix for consistency).
	// Boolean flags use config.FlagOrEnvBool; string/integer flags use config.FlagOrEnv.
	// When adding new env-backed flags, use these helpers — never raw os.Getenv.
	rlIPMaxStr := config.FlagOrEnv(*rateLimitIPMaxStr, "BIGBASE_RATE_LIMIT_IP_MAX")
	rlUserMaxStr := config.FlagOrEnv(*rateLimitUserMaxStr, "BIGBASE_RATE_LIMIT_USER_MAX")

	var rlIPMax, rlUserMax int
	var err error
	rlIPMax, err = strconv.Atoi(rlIPMaxStr)
	if err != nil {
		slog.Warn("invalid rate-limit-ip-max value, defaulting", "value", rlIPMaxStr, "error", err)
		rlIPMax = 0
	}
	rlUserMax, err = strconv.Atoi(rlUserMaxStr)
	if err != nil {
		slog.Warn("invalid rate-limit-user-max value, defaulting", "value", rlUserMaxStr, "error", err)
		rlUserMax = 0
	}
	if rlIPMax < 1 {
		rlIPMax = 60
	}
	if rlUserMax < 1 {
		rlUserMax = 300
	}
	rlEnabled := config.FlagOrEnvBool(*rateLimitEnabled, "BIGBASE_RATE_LIMIT_ENABLED")

	newRelicLicenseKey := config.FlagOrEnv(*nrLicenseKey, "NEW_RELIC_LICENSE_KEY")
	newRelicAppName := config.FlagOrEnv(*nrAppName, "NEW_RELIC_APP_NAME")
	newRelicEnabled := *nrEnabled

	level, err := parseLogLevel(*logLevel)
	if err != nil {
		logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
		logger.Error("invalid log level, defaulting", "provided", *logLevel, "error", err)
		level = slog.LevelInfo
	}

	// Parse CORS allowed origins (empty = CORS disabled, default safe).
	corsAllowedOrigins := parseCORSOrigins(*corsOrigins)

	// New Relic Application agent initialization. Must precede logger
	// construction so buildHandler can route logs through the agent for
	// forwarding to New Relic Logs (Logs in Context).
	var nrInitErr error
	if newRelicEnabled && newRelicLicenseKey != "" {
		nrApp, nrInitErr = newrelic.NewApplication(
			newrelic.ConfigAppName(newRelicAppName),
			newrelic.ConfigLicense(newRelicLicenseKey),
			newrelic.ConfigAppLogForwardingEnabled(true),
			newrelic.ConfigDebugLogger(os.Stdout),
		)
	}

	logger := slog.New(buildHandler(nrApp, os.Stdout, &slog.HandlerOptions{Level: level}))

	switch {
	case nrInitErr != nil:
		logger.Warn("new relic agent initialization failed", "error", nrInitErr)
	case nrApp != nil:
		logger.Info("new relic agent initialized", "app", newRelicAppName)
	default:
		logger.Debug("new relic agent disabled")
	}
	k := kernel.New(logger)

	p := proxy.New(proxy.Options{
		Port:               *port,
		Kernel:             k,
		Logger:             logger,
		CORSAllowedOrigins: corsAllowedOrigins,
		NRApp:              nrApp,
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
		PublicURL:          config.FlagOrEnv(*publicURL, "BIGBASE_PUBLIC_URL"),
		CORSAllowedOrigins: corsAllowedOrigins,
		PostLoginRedirect:  *postLoginRedirect,
		SPAOriginAllowlist: parseCORSOrigins(*spaOriginAllowlist),
		AccessExpiry:       accessExpiry,
		RefreshExpiry:      refreshExpiry,
	})

	p.SetDB(d)
	p.SetValidators(
		func(token string) (int64, string, error) {
			claims, err := authComp.ValidateToken(token)
			if err != nil {
				return 0, "", err
			}
			return claims.UserID, claims.Role, nil
		},
		func(siteID string, token string) error {
			resolvedSiteID, err := authComp.ResolveSiteKey(token)
			if err != nil {
				return err
			}
			if resolvedSiteID != siteID {
				return fmt.Errorf("site key unauthorized for this site")
			}
			return nil
		},
	)

	// Rate limiter for auth public endpoints
	rlCfg := auth.RateLimiterConfig{
		IPLimit:      rlIPMax,
		IPWindow:     time.Minute,
		UserLimit:    rlUserMax,
		UserWindow:   time.Minute,
		CleanupEvery: 5 * time.Minute,
	}
	rl := auth.NewRateLimiter(rlCfg)
	if rlEnabled {
		logger.Info("rate limiter enabled", "ip_max_per_min", rlIPMax, "user_max_per_min", rlUserMax)
	} else {
		logger.Info("rate limiter disabled")
	}

	ad := admin.New(admin.Options{Logger: logger})
	s := storage.New(storage.Options{DB: d, Logger: logger})
	g := git.New(git.Options{DB: d, Logger: logger})
	f := forge.New(forge.Options{DB: d, Logger: logger})
	ci := cici.New(cici.Options{DB: d, Logger: logger})
	fn := functions.New(functions.Options{DB: d, Logger: logger})
	projectsComp := projects.New(projects.Options{DB: d, Logger: logger})
	secretsKey := encryptionKey
	if len(secretsKey) == 0 && allowPlaintext {
		// Development-only composition fallback: a random in-memory root key so
		// the project secret manager is usable without key management. Never
		// persisted, never the production path (production requires the env key).
		secretsKey = make([]byte, 32)
		if _, err := rand.Read(secretsKey); err != nil {
			logger.Error("generate development secret key", "error", err)
			os.Exit(1)
		}
	}
	secretsComp, err := secrets.New(secrets.Options{DB: d, Logger: logger, RootKey: secretsKey})
	if err != nil {
		logger.Error("initialize secret manager", "error", err)
		os.Exit(1)
	}
	msgComp := messaging.New(messaging.Options{
		DB:     d,
		Logger: logger,
	})
	effectiveDSN := dbDSNVal
	if effectiveDSN == "" {
		effectiveDSN = *dbPath
	}
	depComp := deploy.New(deploy.Options{
		DB:             d,
		Logger:         logger,
		BuildHome:      os.Getenv("BIGBASE_HOME"),
		PublicDomain:   sitesDomainVal,
		HostRouter:     p,
		EncryptionKey:  encryptionKey,
		AllowPlaintext: allowPlaintext,
		DBDriver:       dbDriverVal,
		DBDSN:          effectiveDSN,
	})
	mComp := monitoring.New(monitoring.Options{DB: d, Logger: logger})
	// Wire alert.triggered → SMTP email delivery (Issue #178). The notifier is
	// only installed when SMTP host + at least one recipient are configured, so
	// local/dev setups with no mail server are unaffected. The subscriber that
	// forwards events to the notifier is registered inside monitoring.Start.
	if smtpHost := config.FlagOrEnv("", "BIGBASE_SMTP_HOST"); smtpHost != "" {
		toAddrs := strings.Split(config.FlagOrEnv("", "BIGBASE_ALERT_EMAIL_TO"), ",")
		var recipients []string
		for _, a := range toAddrs {
			if a = strings.TrimSpace(a); a != "" {
				recipients = append(recipients, a)
			}
		}
		if len(recipients) > 0 {
			alertNotifier := messaging.NewSMTPAlertNotifier(messaging.SMTPAlertNotifierOptions{
				Host:     smtpHost,
				Port:     config.FlagOrEnv("587", "BIGBASE_SMTP_PORT"),
				Username: config.FlagOrEnv("", "BIGBASE_SMTP_USER"),
				Password: config.FlagOrEnv("", "BIGBASE_SMTP_PASS"),
				From:     config.FlagOrEnv("no-reply@bigbase.local", "BIGBASE_SMTP_FROM"),
				To:       recipients,
				AlertURL: config.FlagOrEnv(*publicURL, "BIGBASE_PUBLIC_URL"),
			})
			mComp.SetAlertNotifier(alertNotifierAdapter{m: alertNotifier})
		} else {
			logger.Warn("SMTP host configured but BIGBASE_ALERT_EMAIL_TO empty; alert email delivery disabled")
		}
	}
	depComp.SetDiagnosisReader(deployDiagnosisAdapter{m: mComp})
	depComp.SetRelatedEventsReader(deployRelatedEventsAdapter{m: mComp})
	gh := github.New(github.Options{
		DB:             d,
		Logger:         logger,
		AppID:          ghAppID,
		AppSlug:        ghAppSlug,
		PrivateKeyPath: ghPrivateKeyPath,
		WebhookSecret:  ghWebhookSecret,
	})
	st := sites.New(sites.Options{
		DB:                 d,
		Logger:             logger,
		EncryptionKey:      encryptionKey,
		AllowPlaintext:     allowPlaintext,
		ProjectProvisioner: projectsComp,
		TriggerDeploy: func(ctx context.Context, repoID, branch, siteName, siteID string, passthroughPaths []string, appType string) (*sites.Deployment, error) {
			dep, err := depComp.Trigger(ctx, repoID, branch, siteName, siteID, passthroughPaths, appType, "")
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
		CertInfo:         p.CertInfo,
		UnregisterHost:   p.UnregisterDeploymentHost,
		ActivateDomain:   depComp.ActivateCustomDomain,
		UpdateAuthPolicy: p.SetSiteAuthPolicy,
		UpdateCSP:        p.SetSiteCSP,
		ValidateManifest: deploy.ValidateManifest,
	})
	rt := realtime.New(realtime.Options{
		Logger:         logger,
		AllowedOrigins: corsAllowedOrigins,
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
	k.Register(projectsComp)
	k.Register(secretsComp)
	k.Register(ci)
	k.Register(fn)
	k.Register(rt)
	k.Register(msgComp)
	k.Register(depComp)
	k.Register(mComp)

	mcpComp := mcp.New(mcp.Options{
		Logger:               logger,
		Enabled:              !*mcpDisabled,
		Port:                 *mcpPort,
		Transport:            *mcpTransport,
		DB:                   d,
		Deployer:             mcpDeployAdapter{d: depComp},
		GitCreator:           g,
		SiteCreator:          st,
		SiteLister:           mcpSiteListerAdapter{s: st},
		SiteKeyCreator:       mcpSiteKeyAdapter{a: authComp},
		SiteKeyResolver:      mcpSiteKeyAdapter{a: authComp},
		SiteEnvVarManager:    mcpEnvVarAdapter{s: st},
		SiteTargetAuthorizer: mcpEnvVarAdapter{s: st},
		OrgKeyAuthenticator:  authComp,
		UpdateAuthPolicy:     p.SetSiteAuthPolicy,
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
	projectsHandler := mComp.Middleware(authComp.Middleware(projectsComp.Handler()))

	p.Handle("/api/projects", projectsHandler.ServeHTTP)
	p.Handle("/api/projects/", projectsHandler.ServeHTTP)
	secretAPI, apiErr := api.NewSecretsAPI(api.SecretsAPIOptions{Manager: secretsComp, DB: d, Logger: logger})
	if apiErr != nil {
		logger.Error("initialize secret REST API", "error", apiErr)
		os.Exit(1)
	}
	secretsHandler := mComp.Middleware(authComp.Middleware(http.HandlerFunc(secretAPI.ServeHTTP)))
	p.Handle("/api/projects/{project}/environments/{env}/secrets", secretsHandler.ServeHTTP)
	p.Handle("/api/projects/{project}/environments/{env}/secrets/", secretsHandler.ServeHTTP)
	p.Handle("/api/collections/", protectedAPI.ServeHTTP)
	// /api/sql requires admin role. The access rule is now declared as a Policy
	// (issue #43) rather than a hand-threaded middleware chain: PolicyAdmin()
	// documents the requirement at registration time and enforces it the same
	// way the former bare auth.RequireAdmin middleware did.
	sqlHandler := mComp.Middleware(authComp.Middleware(auth.PolicyAdmin().Middleware(orgBridge(publicAPI))))
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
	// Auth public routes: rate-limited when enabled
	if rlEnabled {
		p.Handle("/api/auth/", mComp.Middleware(rl.Middleware(authComp.Handler())).ServeHTTP)
	} else {
		p.Handle("/api/auth/", mComp.Middleware(authComp.Handler()).ServeHTTP)
	}
	p.Handle("GET /api/auth/users", mComp.Middleware(authComp.ProtectedHandler()).ServeHTTP)
	p.Handle("GET /api/auth/me", mComp.Middleware(authComp.ProtectedHandler()).ServeHTTP)
	p.Handle("PATCH /api/auth/me", mComp.Middleware(authComp.ProtectedHandler()).ServeHTTP)
	p.Handle("GET /api/auth/me/identities", mComp.Middleware(authComp.ProtectedHandler()).ServeHTTP)
	p.Handle("POST /api/auth/me/identities", mComp.Middleware(authComp.ProtectedHandler()).ServeHTTP)
	p.Handle("DELETE /api/auth/me/identities/{provider}", mComp.Middleware(authComp.ProtectedHandler()).ServeHTTP)
	p.Handle("POST /api/auth/logout-all", mComp.Middleware(authComp.ProtectedHandler()).ServeHTTP)
	p.Handle("DELETE /api/auth/users/", mComp.Middleware(authComp.ProtectedHandler()).ServeHTTP)
	// Org routes
	p.Handle("POST /api/orgs", mComp.Middleware(authComp.ProtectedHandler()).ServeHTTP)
	p.Handle("GET /api/orgs", mComp.Middleware(authComp.ProtectedHandler()).ServeHTTP)
	p.Handle("GET /api/orgs/", mComp.Middleware(authComp.ProtectedHandler()).ServeHTTP)
	p.Handle("PATCH /api/orgs/", mComp.Middleware(authComp.ProtectedHandler()).ServeHTTP)
	p.Handle("DELETE /api/orgs/", mComp.Middleware(authComp.ProtectedHandler()).ServeHTTP)
	p.Handle("POST /api/orgs/{id}/api-keys", mComp.Middleware(authComp.ProtectedHandler()).ServeHTTP)
	p.Handle("GET /api/orgs/{id}/api-keys", mComp.Middleware(authComp.ProtectedHandler()).ServeHTTP)
	p.Handle("DELETE /api/orgs/{id}/api-keys/{keyID}", mComp.Middleware(authComp.ProtectedHandler()).ServeHTTP)
	p.Handle("POST /api/orgs/{id}/invites", mComp.Middleware(authComp.ProtectedHandler()).ServeHTTP)
	p.Handle("POST /api/orgs/{id}/invites/{token}/accept", mComp.Middleware(authComp.ProtectedHandler()).ServeHTTP)
	p.Handle("GET /api/orgs/{id}/members", mComp.Middleware(authComp.ProtectedHandler()).ServeHTTP)
	// Site deploy-key routes
	p.Handle("POST /api/sites/{id}/deploy-keys", mComp.Middleware(authComp.ProtectedHandler()).ServeHTTP)
	p.Handle("GET /api/sites/{id}/deploy-keys", mComp.Middleware(authComp.ProtectedHandler()).ServeHTTP)
	p.Handle("DELETE /api/sites/{id}/deploy-keys/{keyID}", mComp.Middleware(authComp.ProtectedHandler()).ServeHTTP)
	p.Handle("/api/monitoring/health", mComp.Handler().ServeHTTP)
	p.Handle("/api/monitoring/metrics", authComp.Middleware(mComp.Handler()).ServeHTTP)
	p.Handle("/api/monitoring/metrics/prometheus", mComp.Handler().ServeHTTP)
	p.Handle("/api/monitoring/logs", authComp.Middleware(mComp.Handler()).ServeHTTP)
	p.Handle("/api/monitoring/logs/", authComp.Middleware(mComp.Handler()).ServeHTTP)
	p.Handle("/api/monitoring/alerts", authComp.Middleware(mComp.Handler()).ServeHTTP)
	p.Handle("/api/monitoring/events", authComp.Middleware(mComp.Handler()).ServeHTTP)
	p.Handle("/api/monitoring/incidents", authComp.Middleware(mComp.Handler()).ServeHTTP)
	p.Handle("/api/monitoring/incidents/", authComp.Middleware(mComp.Handler()).ServeHTTP)
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
  bigbase deploy [--server URL] [--repo ID]     Deploy a git repo
  bigbase init [--repo PATH]                     Generate default bigbase.yaml
  bigbase help                                  Show this help`)
}

func runInitCmd() {
	fs := flag.NewFlagSet("init", flag.ExitOnError)
	repo := fs.String("repo", ".", "Repository path")
	_ = fs.Parse(os.Args[2:])

	repoPath := *repo
	absPath, err := filepath.Abs(repoPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: resolve path: %v\n", err)
		os.Exit(1)
	}

	fi, err := os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "error: directory %s does not exist\n", absPath)
		} else {
			fmt.Fprintf(os.Stderr, "error: stat path: %v\n", err)
		}
		os.Exit(1)
	}
	if !fi.IsDir() {
		fmt.Fprintf(os.Stderr, "error: %s is not a directory\n", absPath)
		os.Exit(1)
	}

	if err := deploy.InitManifest(absPath); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Initialized default bigbase.yaml in %s\n", absPath)
}

func runDeployCmd() {
	fs := flag.NewFlagSet("deploy", flag.ExitOnError)
	server := fs.String("server", "http://localhost:8080", "BigBase server URL")
	repoID := fs.String("repo", "", "Repository ID (required)")
	branch := fs.String("branch", "main", "Branch to deploy")
	siteName := fs.String("site-name", "", "Site name (defaults to repo name)")
	siteID := fs.String("site-id", "", "Site ID (optional)")
	apiKey := fs.String("api-key", "", "API key for authentication")
	wait := fs.Bool("wait", true, "Wait for deployment to complete")
	manifest := fs.String("manifest", "", "Manifest file path relative to repo root")
	_ = fs.Parse(os.Args[2:])

	serverURL := config.FlagOrEnv(*server, "BIGBASE_SERVER")
	key := config.FlagOrEnv(*apiKey, "BIGBASE_API_KEY")

	if *repoID == "" {
		fmt.Fprintln(os.Stderr, "error: --repo is required")
		fmt.Fprintln(os.Stderr, "Usage: bigbase deploy --repo <repo_id> [--branch main] [--server http://...] [--manifest path]")
		os.Exit(1)
	}

	if serverURL == "" {
		serverURL = "http://localhost:8080"
	}

	serverURL = strings.TrimRight(serverURL, "/")

	body := map[string]string{
		"repo_id":       *repoID,
		"branch":        *branch,
		"site_name":     *siteName,
		"site_id":       *siteID,
		"manifest_path": *manifest,
	}
	jsonBody, err := json.Marshal(body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: marshal body: %v\n", err)
		os.Exit(1)
	}

	req, err := http.NewRequest("POST", serverURL+"/api/deploy", strings.NewReader(string(jsonBody)))
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: create request: %v\n", err)
		os.Exit(1)
	}
	req.Header.Set("Content-Type", "application/json")
	if key != "" {
		req.Header.Set("Authorization", "Bearer "+key)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: connect to %s: %v\n", serverURL, err)
		os.Exit(1)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: read response: %v\n", err)
		os.Exit(1)
	}

	if resp.StatusCode != http.StatusCreated {
		fmt.Fprintf(os.Stderr, "error: server returned %d\n%s\n", resp.StatusCode, string(respBody))
		os.Exit(1)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(respBody, &raw); err != nil {
		fmt.Fprintf(os.Stderr, "error: parse response: %v\n", err)
		os.Exit(1)
	}

	deployID, _ := raw["id"].(string)
	deployURL, _ := raw["url"].(string)
	deployStatus, _ := raw["status"].(string)

	if deployID == "" && deployURL == "" {
		fmt.Fprintf(os.Stderr, "error: unexpected response: %s\n", string(respBody))
		os.Exit(1)
	}

	shortID := deployID
	if len(shortID) > 8 {
		shortID = shortID[:8]
	}

	fmt.Println("Deployment created!")
	fmt.Println("  ID:", shortID)
	fmt.Println("  URL:", deployURL)
	fmt.Println("  Status:", deployStatus)

	if !*wait {
		return
	}

	fmt.Println()
	fmt.Println("Waiting for deployment to complete...")
	client := &http.Client{Timeout: 5 * time.Second}
	for {
		time.Sleep(2 * time.Second)

		statusReq, err := http.NewRequest("GET", serverURL+"/api/deploy/"+deployID, nil)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: poll request: %v\n", err)
			continue
		}
		if key != "" {
			statusReq.Header.Set("Authorization", "Bearer "+key)
		}

		statusResp, err := client.Do(statusReq)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: poll failed: %v\n", err)
			continue
		}

		statusBody, err := io.ReadAll(statusResp.Body)
		_ = statusResp.Body.Close()
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: poll read: %v\n", err)
			continue
		}

		var rawStatus map[string]interface{}
		if err := json.Unmarshal(statusBody, &rawStatus); err != nil {
			fmt.Fprintf(os.Stderr, "warning: poll parse: %v\n", err)
			continue
		}

		status, _ := rawStatus["status"].(string)
		buildLog, _ := rawStatus["build_log"].(string)
		errMsg, _ := rawStatus["error_message"].(string)

		fmt.Printf("  Status: %s\n", status)

		switch status {
		case "running":
			fmt.Println()
			fmt.Println("Deployment is live!")
			return
		case "failed":
			if errMsg != "" {
				fmt.Fprintf(os.Stderr, "\nError: %s\n", errMsg)
			}
			if buildLog != "" {
				fmt.Println()
				fmt.Println("Build log:")
				fmt.Println(buildLog)
			}
			os.Exit(1)
			return
		case "pending":
			// still building, keep polling
		case "replaced":
			fmt.Println()
			fmt.Println("Deployment was replaced by a newer one.")
			return
		}
	}
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
		"COMPONENT\tVERSION\tSTATUS",
		func(c kernel.ComponentStatus) string {
			status := "stopped"
			if c.Running {
				status = "running"
			}
			return fmt.Sprintf("%s\t%s\t%s", c.Name, c.Version, status)
		},
	)
}

func printComponents(k *kernel.Kernel) {
	printTable(k.ListComponents(),
		"NAME\tVERSION\tDEPENDENCIES",
		func(c kernel.ComponentStatus) string {
			return fmt.Sprintf("%s\t%s\t%s", c.Name, c.Version, joinOrNone(c.Dependencies))
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
	_, _ = fmt.Fprintf(os.Stdout, "backup written to %s\n", *output)
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
	_, _ = fmt.Fprintln(os.Stdout, "restore complete")
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
		_, _ = fmt.Fprintln(os.Stdout, "migrations applied")
	case "down":
		if err := m.Down(context.Background()); err != nil {
			fmt.Fprintf(os.Stderr, "error: migrate down: %v\n", err)
			os.Exit(1)
		}
		_, _ = fmt.Fprintln(os.Stdout, "migrations rolled back")
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
