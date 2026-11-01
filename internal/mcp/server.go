package mcp

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/mark3labs/mcp-go/server"
	"github.com/slemmercs/rclone-bisync/internal/app"
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
