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

	"github.com/danielvm/bigbase/config"
	"github.com/danielvm/bigbase/components/admin"
	"github.com/danielvm/bigbase/components/api"
	"github.com/danielvm/bigbase/components/auth"
	"github.com/danielvm/bigbase/components/cici"
	"github.com/danielvm/bigbase/components/deploy"
	"github.com/danielvm/bigbase/components/db"
	"github.com/danielvm/bigbase/components/forge"
	"github.com/danielvm/bigbase/components/functions"
	"github.com/danielvm/bigbase/components/git"
	"github.com/danielvm/bigbase/components/github"
	"github.com/danielvm/bigbase/components/messaging"
	"github.com/danielvm/bigbase/components/sites"
	"github.com/danielvm/bigbase/components/monitoring"
	"github.com/danielvm/bigbase/components/proxy"
	"github.com/danielvm/bigbase/components/realtime"
	"github.com/danielvm/bigbase/components/storage"
	"github.com/danielvm/bigbase/kernel"
)

var (
	version = kernel.Version
)

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
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
	k := kernel.New(logger)

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
	dbPath := serveFS.String("db", "bigbase.db", "SQLite database path")
	googleClientID := serveFS.String("google-client-id", "", "Google OAuth client ID")
	googleClientSecret := serveFS.String("google-client-secret", "", "Google OAuth client secret")
	githubAppID := serveFS.String("github-app-id", "", "GitHub App ID")
	githubAppSlug := serveFS.String("github-app-slug", "", "GitHub App slug")
	githubPrivateKeyPath := serveFS.String("github-app-private-key-path", "", "GitHub App private key path")
	githubWebhookSecret := serveFS.String("github-webhook-secret", "", "GitHub App webhook secret")
	sitesDomain := serveFS.String("sites-domain", "", "Parent domain for deployed site subdomains (e.g. bigbase.click)")
	_ = serveFS.Parse(os.Args[2:])

	googleID := config.FlagOrEnv(*googleClientID, "GOOGLE_CLIENT_ID")
	googleSecret := config.FlagOrEnv(*googleClientSecret, "GOOGLE_CLIENT_SECRET")
	ghAppID := config.FlagOrEnv(*githubAppID, "GITHUB_APP_ID")
	ghAppSlug := config.FlagOrEnv(*githubAppSlug, "GITHUB_APP_SLUG")
	ghPrivateKeyPath := config.FlagOrEnv(*githubPrivateKeyPath, "GITHUB_APP_PRIVATE_KEY_PATH")
	ghWebhookSecret := config.FlagOrEnv(*githubWebhookSecret, "GITHUB_WEBHOOK_SECRET")
	sitesDomainVal := config.FlagOrEnv(*sitesDomain, "BIGBASE_SITES_DOMAIN")

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	k := kernel.New(logger)

	p := proxy.New(proxy.Options{
		Port:   *port,
		Kernel: k,
		Logger: logger,
	})
	d := db.New(db.Options{
		Path:   *dbPath,
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
		PublicDomain: sitesDomainVal,
		HostRouter:   p,
	})
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
		TriggerDeploy: func(ctx context.Context, repoID, branch string) (*sites.Deployment, error) {
			dep, err := depComp.Trigger(ctx, repoID, branch)
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

	// Register routes before kernel.Start to avoid race on proxy mux
	publicAPI := a.Handler()
	protectedAPI := mComp.Middleware(authComp.Middleware(publicAPI))
	storageHandler := mComp.Middleware(authComp.Middleware(s.Handler()))
	gitHandler := mComp.Middleware(authComp.Middleware(g.Handler()))
	forgeHandler := mComp.Middleware(authComp.Middleware(f.Handler()))
	githubPublic := mComp.Middleware(gh.PublicHandler())
	githubProtected := mComp.Middleware(authComp.Middleware(gh.ProtectedHandler()))
	sitesHandler := mComp.Middleware(authComp.Middleware(st.Handler()))

	p.Handle("/api/collections/", protectedAPI.ServeHTTP)
	p.Handle("/api/sql", protectedAPI.ServeHTTP)
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
	p.Handle("/api/monitoring/health", mComp.Handler().ServeHTTP)
	p.Handle("/api/monitoring/metrics", authComp.Middleware(mComp.Handler()).ServeHTTP)
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

	logger.Info("bigbase running", "port", *port, "db", *dbPath)

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
  bigbase version              Show version
  bigbase status               Show kernel and component status
  bigbase components list      List registered components
  bigbase serve [--port PORT] [--db PATH]  Start HTTP server (default :8080)
  bigbase help                 Show this help`)
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

func joinOrNone(items []string) string {
	if len(items) == 0 {
		return "none"
	}
	return strings.Join(items, ", ")
}
