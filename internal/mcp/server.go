package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/mark3labs/mcp-go/server"
	"github.com/slemmercs/rclone-bisync/internal/app"
	"github.com/slemmercs/rclone-bisync/internal/models"
)

// ServerConfig configures the MCP server operational parameters.
type ServerConfig struct {
	Version             string
	ReadOnly            bool
	RequireConfirmation bool
	Transport           string // "stdio" or "sse"
	Host                string
	Port                int
}

// Server encapsulates the Model Context Protocol (MCP) server for Bisync Hub.
type Server struct {
	cfg       ServerConfig
	container *app.Container
	mcpServer *server.MCPServer
	tickets   *TicketManager
}

// NewServer instantiates an MCP server connected to the container services.
func NewServer(ctr *app.Container, cfg ServerConfig) (*Server, error) {
	if cfg.Version == "" {
		cfg.Version = "dev"
	}
	if cfg.Host == "" {
		cfg.Host = "127.0.0.1"
	}
	if cfg.Port == 0 {
		cfg.Port = 8081
	}

	mcpSrv := server.NewMCPServer(
		"bisync-hub",
		cfg.Version,
		server.WithResourceCapabilities(true, true),
		server.WithToolCapabilities(true),
	)

	s := &Server{
		cfg:       cfg,
		container: ctr,
		mcpServer: mcpSrv,
		tickets:   NewTicketManager(5 * time.Minute),
	}

	s.registerReadTools()
	s.registerMutatingTools()
	s.registerResources()

	return s, nil
}

// MCPServer returns the underlying mark3labs MCPServer instance.
func (s *Server) MCPServer() *server.MCPServer {
	return s.mcpServer
}

// Config returns the active server configuration.
func (s *Server) Config() ServerConfig {
	return s.cfg
}

// TicketManager returns the confirmation ticket manager.
func (s *Server) TicketManager() *TicketManager {
	return s.tickets
}

// ServeStdio starts serving MCP over standard input/output streams.
func (s *Server) ServeStdio(_ context.Context) error {
	s.writeStatusFile("stdio", "", 0)
	defer s.removeStatusFile()
	return server.ServeStdio(s.mcpServer)
}

// ServeSSE starts serving MCP over HTTP Server-Sent Events (SSE).
func (s *Server) ServeSSE(ctx context.Context, host string, port int) error {
	if host == "" {
		host = s.cfg.Host
	}
	if port == 0 {
		port = s.cfg.Port
	}

	s.writeStatusFile("sse", host, port)
	defer s.removeStatusFile()

	addr := fmt.Sprintf("%s:%d", host, port)
	streamSrv := server.NewStreamableHTTPServer(s.mcpServer)

	httpServer := &http.Server{
		Addr:              addr,
		Handler:           streamSrv,
		ReadHeaderTimeout: 10 * time.Second,
	}

	errChan := make(chan error, 1)
	go func() {
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errChan <- err
		}
		close(errChan)
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = httpServer.Shutdown(shutdownCtx)
		return nil
	case err := <-errChan:
		return err
	}
}

func (s *Server) writeStatusFile(transport, host string, port int) {
	if s.container == nil || s.container.Config == nil || s.container.Config.MCPStatusFile == "" {
		return
	}
	status := models.MCPStatus{
		Status:    "active",
		Transport: transport,
		ReadOnly:  s.cfg.ReadOnly,
		Host:      host,
		Port:      port,
		PID:       os.Getpid(),
		StartedAt: time.Now().Format(time.RFC3339),
	}
	data, err := json.MarshalIndent(status, "", "  ")
	if err != nil {
		return
	}
	targetFile := s.container.Config.MCPStatusFile
	_ = os.MkdirAll(filepath.Dir(targetFile), 0750)
	tmpFile := fmt.Sprintf("%s.tmp.%d", targetFile, os.Getpid())
	if err := os.WriteFile(tmpFile, data, 0600); err == nil {
		_ = os.Rename(tmpFile, targetFile)
	}
}

func (s *Server) removeStatusFile() {
	if s.container != nil && s.container.Config != nil && s.container.Config.MCPStatusFile != "" {
		_ = os.Remove(s.container.Config.MCPStatusFile)
	}
}
