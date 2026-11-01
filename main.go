package main

import (
	"context"
	"embed"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/slemmercs/rclone-bisync/internal/app"
	"github.com/slemmercs/rclone-bisync/internal/config"
	"github.com/slemmercs/rclone-bisync/internal/engine/runner"
	"github.com/slemmercs/rclone-bisync/internal/mcp"
	"github.com/slemmercs/rclone-bisync/internal/server"
)

//go:embed dashboard/frontend/*
var frontendFS embed.FS

// Version is injected at link/build time via -ldflags="-X main.Version=vX.Y.Z"
var Version = "dev"

func main() {
	// Enforce restrictive POSIX umask 0077 so runtime files, sqlite ledgers, and logs are user-accessible only
	syscall.Umask(0077)

	envFile := flag.String("env-file", "", "Path to custom bisync.env configuration file")

	// Subcommands:
	// rclone-bisync dashboard [--host] [--port]
	// rclone-bisync sync [--resync] [--force] [--dry-run] [--origin] [--no-notify]
	// rclone-bisync canary [--remote] [--path]

	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "version", "--version", "-v":
			fmt.Printf("Bisync Hub version %s\n", Version)
			return

		case "dashboard":
			dashCmd := flag.NewFlagSet("dashboard", flag.ExitOnError)
			host := dashCmd.String("host", "127.0.0.1", "Dashboard host address")
			port := dashCmd.Int("port", 8080, "Dashboard port")
			dashEnv := dashCmd.String("env-file", "", "Path to custom bisync.env")
			debug := dashCmd.Bool("debug", false, "Enable verbose debug logging")
			dashCmd.BoolVar(debug, "d", false, "Enable verbose debug logging (shorthand)")
			_ = dashCmd.Parse(os.Args[2:])

			runDashboard(*host, *port, *dashEnv, *debug)
			return

		case "sync":
			syncCmd := flag.NewFlagSet("sync", flag.ExitOnError)
			resync := syncCmd.Bool("resync", false, "Rebuild synchronization cache tracking from baseline")
			force := syncCmd.Bool("force", false, "Authorize safety overrides including mass deletions")
			dryRun := syncCmd.Bool("dry-run", false, "Simulate synchronization without applying filesystem changes")
			noNotify := syncCmd.Bool("no-notify", false, "Suppress desktop notifications")
			origin := syncCmd.String("origin", "cli", "Origin caller context (cli, timer, dashboard)")
			syncEnv := syncCmd.String("env-file", "", "Path to custom bisync.env")
			debug := syncCmd.Bool("debug", false, "Enable verbose debug logging")
			syncCmd.BoolVar(debug, "d", false, "Enable verbose debug logging (shorthand)")
			_ = syncCmd.Parse(os.Args[2:])

			runSync(*resync, *force, *dryRun, *noNotify, *origin, *syncEnv, *debug)
			return

		case "canary":
			canaryCmd := flag.NewFlagSet("canary", flag.ExitOnError)
			remote := canaryCmd.String("remote", "local", "Remote storage identifier (local or remote name)")
			path := canaryCmd.String("path", "", "Subfolder path to provision RCLONE_TEST canary")
			canaryEnv := canaryCmd.String("env-file", "", "Path to custom bisync.env")
			_ = canaryCmd.Parse(os.Args[2:])

			runCanary(*remote, *path, *canaryEnv)
			return

		case "mcp":
			mcpCmd := flag.NewFlagSet("mcp", flag.ExitOnError)
			transport := mcpCmd.String("transport", "stdio", "Transport protocol: 'stdio' or 'sse'")
			host := mcpCmd.String("host", "127.0.0.1", "Host address for SSE transport")
			port := mcpCmd.Int("port", 8081, "Port for SSE transport")
			readOnly := mcpCmd.Bool("read-only", false, "Run in read-only inspection mode (disallow mutations)")
			requireConfirmation := mcpCmd.Bool("require-confirmation", true, "Require TTL tickets for high-blast-radius operations")
			mcpEnv := mcpCmd.String("env-file", "", "Path to custom bisync.env")
			debug := mcpCmd.Bool("debug", false, "Enable verbose debug logging")
			mcpCmd.BoolVar(debug, "d", false, "Enable verbose debug logging (shorthand)")
			_ = mcpCmd.Parse(os.Args[2:])

			runMCP(*transport, *host, *port, *readOnly, *requireConfirmation, *mcpEnv, *debug)
			return
		}
	}

	// Default fallback to dashboard if no subcommand specified or flags given directly
	host := flag.String("host", "127.0.0.1", "Dashboard host address")
	port := flag.Int("port", 8080, "Dashboard port")
	debug := flag.Bool("debug", false, "Enable verbose debug logging")
	flag.BoolVar(debug, "d", false, "Enable verbose debug logging (shorthand)")
	flag.Parse()
	runDashboard(*host, *port, *envFile, *debug)
}

func runDashboard(host string, port int, envFile string, debug bool) {
	cfg, err := config.LoadConfig(envFile)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}
	if debug {
		cfg.DebugMode = true
	}

	appContainer, err := app.NewContainer(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize application container: %v", err)
	}
	if err := appContainer.Start(context.Background()); err != nil {
		slog.Warn("failed to start background services", "error", err)
	}
	defer func() {
		if err := appContainer.Close(); err != nil {
			slog.Error("error during container shutdown", "error", err)
		}
	}()

	srv := server.NewServerWithContainer(appContainer, frontendFS)
	srv.Version = Version

	addr := fmt.Sprintf("%s:%d", host, port)
	httpServer := &http.Server{
		Addr:              addr,
		Handler:           srv,
		ReadHeaderTimeout: 10 * time.Second,
	}

	stopSig := make(chan os.Signal, 1)
	signal.Notify(stopSig, os.Interrupt, syscall.SIGTERM)

	go func() {
		slog.Info("Bisync Hub dashboard running", "addr", fmt.Sprintf("http://%s", addr), "debug", cfg.DebugMode)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server failure: %v", err)
		}
	}()

	<-stopSig
	slog.Info("Shutting down dashboard server gracefully...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		slog.Error("HTTP server shutdown error", "error", err)
	}
	slog.Info("Dashboard server stopped.")
}

func runSync(resync, force, dryRun, noNotify bool, origin, envFile string, debug bool) {
	cfg, err := config.LoadConfig(envFile)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}
	if debug {
		cfg.DebugMode = true
	}

	syncRunner := runner.NewSyncRunner(cfg)
	success := syncRunner.Run(context.Background(), resync, force, dryRun, origin, "", noNotify)
	if !success {
		os.Exit(1)
	}
}

func runCanary(remote, path, envFile string) {
	cfg, err := config.LoadConfig(envFile)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	appContainer, err := app.NewContainer(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize application container: %v", err)
	}
	defer func() {
		_ = appContainer.Close()
	}()

	res, err := appContainer.CanaryService.CreateCanary(context.Background(), remote, path)
	if err != nil {
		log.Fatalf("Failed to provision canary: %v", err)
	}
	if res.Status == "error" {
		log.Fatalf("Error provisioning canary: %v", res.Error)
	}
	fmt.Printf("Successfully provisioned RCLONE_TEST canary on [%s] %s\n", remote, path)
}

func runMCP(transport, host string, port int, readOnly, requireConfirmation bool, envFile string, debug bool) {
	cfg, err := config.LoadConfig(envFile)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}
	if debug {
		cfg.DebugMode = true
	}

	appContainer, err := app.NewContainer(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize application container: %v", err)
	}
	if err := appContainer.Start(context.Background()); err != nil {
		slog.Warn("failed to start background services", "error", err)
	}
	defer func() {
		_ = appContainer.Close()
	}()

	mcpSrv, err := mcp.NewServer(appContainer, mcp.ServerConfig{
		Version:             Version,
		ReadOnly:            readOnly,
		RequireConfirmation: requireConfirmation,
		Transport:           transport,
		Host:                host,
		Port:                port,
	})
	if err != nil {
		log.Fatalf("Failed to initialize MCP server: %v", err)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	if transport == "sse" {
		slog.Info("Starting Bisync Hub MCP server (SSE)", "host", host, "port", port, "read_only", readOnly)
		if err := mcpSrv.ServeSSE(ctx, host, port); err != nil {
			log.Fatalf("MCP SSE server error: %v", err)
		}
	} else {
		if err := mcpSrv.ServeStdio(ctx); err != nil {
			log.Fatalf("MCP stdio server error: %v", err)
		}
	}
}
