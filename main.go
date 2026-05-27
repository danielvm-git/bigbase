package main

import (
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
	"github.com/danielvm/bigbase/components/db"
	"github.com/danielvm/bigbase/components/proxy"
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
	_ = serveFS.Parse(os.Args[2:])

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
		DB:     d,
		Logger: logger,
	})

	ad := admin.New(admin.Options{Logger: logger})
	s := storage.New(storage.Options{DB: d, Logger: logger})
	k.Register(p)
	k.Register(d)
	k.Register(a)
	k.Register(authComp)
	k.Register(ad)
	k.Register(s)

	// Register routes before kernel.Start to avoid race on proxy mux
	publicAPI := a.Handler()
	protectedAPI := authComp.Middleware(publicAPI)
	storageHandler := authComp.Middleware(s.Handler())

	p.Handle("/api/collections/", protectedAPI.ServeHTTP)
	p.Handle("/api/sql", protectedAPI.ServeHTTP)
	p.Handle("/api/storage/upload", storageHandler.ServeHTTP)
	p.Handle("/api/storage/files/", storageHandler.ServeHTTP)
	p.Handle("/api/storage/files", storageHandler.ServeHTTP)
	p.Handle("/api/auth/", authComp.Handler().ServeHTTP)
	p.Handle("GET /api/auth/users", authComp.ProtectedHandler().ServeHTTP)
	p.Handle("DELETE /api/auth/users/", authComp.ProtectedHandler().ServeHTTP)
	p.Handle("/admin/", http.StripPrefix("/admin/", ad.Handler()).ServeHTTP)

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
