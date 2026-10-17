package main

import (
	"embed"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/slemmercs/rclone-bisync/internal/config"
	"github.com/slemmercs/rclone-bisync/internal/server"
	"github.com/slemmercs/rclone-bisync/internal/services"
)

//go:embed dashboard/frontend/*
var frontendFS embed.FS

func main() {
	envFile := flag.String("env-file", "", "Path to custom bisync.env configuration file")
	host := flag.String("host", "127.0.0.1", "Dashboard host address")
	port := flag.Int("port", 8080, "Dashboard port")
	flag.Parse()

	cfg, err := config.LoadConfig(*envFile)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	syncSvc := services.NewSyncService(cfg, nil)
	srv := server.NewServer(cfg, syncSvc, frontendFS)

	addr := fmt.Sprintf("%s:%d", *host, *port)
	httpServer := &http.Server{
		Addr:    addr,
		Handler: srv,
	}

	stopSig := make(chan os.Signal, 1)
	signal.Notify(stopSig, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Printf("rclone-bisync dashboard running at http://%s", addr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server failure: %v", err)
		}
	}()

	<-stopSig
	log.Println("Shutting down dashboard server gracefully...")
}
