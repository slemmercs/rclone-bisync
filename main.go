package main

import (
	"context"
	"embed"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/slemmercs/rclone-bisync/internal/config"
	"github.com/slemmercs/rclone-bisync/internal/engine/runner"
	"github.com/slemmercs/rclone-bisync/internal/server"
	"github.com/slemmercs/rclone-bisync/internal/services"
)

//go:embed dashboard/frontend/*
var frontendFS embed.FS

// Version is injected at link/build time via -ldflags="-X main.Version=vX.Y.Z"
var Version = "dev"

func main() {
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
			_ = dashCmd.Parse(os.Args[2:])

			runDashboard(*host, *port, *dashEnv)
			return

		case "sync":
			syncCmd := flag.NewFlagSet("sync", flag.ExitOnError)
			resync := syncCmd.Bool("resync", false, "Rebuild synchronization cache tracking from baseline")
			force := syncCmd.Bool("force", false, "Authorize safety overrides including mass deletions")
			dryRun := syncCmd.Bool("dry-run", false, "Simulate synchronization without applying filesystem changes")
			noNotify := syncCmd.Bool("no-notify", false, "Suppress desktop notifications")
			origin := syncCmd.String("origin", "cli", "Origin caller context (cli, timer, dashboard)")
			syncEnv := syncCmd.String("env-file", "", "Path to custom bisync.env")
			_ = syncCmd.Parse(os.Args[2:])

			runSync(*resync, *force, *dryRun, *noNotify, *origin, *syncEnv)
			return

		case "canary":
			canaryCmd := flag.NewFlagSet("canary", flag.ExitOnError)
			remote := canaryCmd.String("remote", "local", "Remote storage identifier (local or remote name)")
			path := canaryCmd.String("path", "", "Subfolder path to provision RCLONE_TEST canary")
			canaryEnv := canaryCmd.String("env-file", "", "Path to custom bisync.env")
			_ = canaryCmd.Parse(os.Args[2:])

			runCanary(*remote, *path, *canaryEnv)
			return
		}
	}

	// Default fallback to dashboard if no subcommand specified or flags given directly
	host := flag.String("host", "127.0.0.1", "Dashboard host address")
	port := flag.Int("port", 8080, "Dashboard port")
	flag.Parse()
	runDashboard(*host, *port, *envFile)
}

func runDashboard(host string, port int, envFile string) {
	cfg, err := config.LoadConfig(envFile)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	syncSvc := services.NewSyncService(cfg, nil)
	srv := server.NewServer(cfg, syncSvc, frontendFS)
	srv.Version = Version

	addr := fmt.Sprintf("%s:%d", host, port)
	httpServer := &http.Server{
		Addr:    addr,
		Handler: srv,
	}

	stopSig := make(chan os.Signal, 1)
	signal.Notify(stopSig, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Printf("Bisync Hub dashboard running at http://%s", addr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server failure: %v", err)
		}
	}()

	<-stopSig
	log.Println("Shutting down dashboard server gracefully...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("HTTP server shutdown error: %v", err)
	}
	log.Println("Dashboard server stopped.")
}

func runSync(resync, force, dryRun, noNotify bool, origin, envFile string) {
	cfg, err := config.LoadConfig(envFile)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
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

	canarySvc := services.NewCanaryService(cfg, nil)
	res, err := canarySvc.CreateCanary(context.Background(), remote, path)
	if err != nil {
		log.Fatalf("Failed to provision canary: %v", err)
	}
	if status, ok := res["status"].(string); ok && status == "error" {
		log.Fatalf("Error provisioning canary: %v", res["error"])
	}
	fmt.Printf("Successfully provisioned RCLONE_TEST canary on [%s] %s\n", remote, path)
}
