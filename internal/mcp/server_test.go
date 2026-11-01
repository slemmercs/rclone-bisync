package mcp

import (
	"context"
	"testing"
	"time"
)

func TestNewServer_Initialization(t *testing.T) {
	cfg := ServerConfig{
		Version:             "v1.0.0-test",
		ReadOnly:            true,
		RequireConfirmation: true,
		Transport:           "stdio",
		Host:                "127.0.0.1",
		Port:                9090,
	}

	srv, err := NewServer(nil, cfg)
	if err != nil {
		t.Fatalf("unexpected error creating server: %v", err)
	}

	if srv.MCPServer() == nil {
		t.Fatal("expected MCPServer to be non-nil")
	}
	if srv.TicketManager() == nil {
		t.Fatal("expected TicketManager to be non-nil")
	}
	if srv.Config().Version != "v1.0.0-test" {
		t.Errorf("expected version v1.0.0-test, got %s", srv.Config().Version)
	}
	if !srv.Config().ReadOnly {
		t.Error("expected ReadOnly to be true")
	}
	if !srv.Config().RequireConfirmation {
		t.Error("expected RequireConfirmation to be true")
	}
	if srv.Config().Port != 9090 {
		t.Errorf("expected port 9090, got %d", srv.Config().Port)
	}
}

func TestNewServer_DefaultConfig(t *testing.T) {
	srv, err := NewServer(nil, ServerConfig{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if srv.Config().Version != "dev" {
		t.Errorf("expected default version 'dev', got %s", srv.Config().Version)
	}
	if srv.Config().Host != "127.0.0.1" {
		t.Errorf("expected default host '127.0.0.1', got %s", srv.Config().Host)
	}
	if srv.Config().Port != 8081 {
		t.Errorf("expected default port 8081, got %d", srv.Config().Port)
	}
}

func TestServer_ServeSSE_GracefulShutdown(t *testing.T) {
	srv, err := NewServer(nil, ServerConfig{
		Host: "127.0.0.1",
		Port: 29876,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)

	go func() {
		errCh <- srv.ServeSSE(ctx, "127.0.0.1", 29876)
	}()

	// Allow server to bind and listen
	time.Sleep(50 * time.Millisecond)

	// Cancel context to trigger shutdown
	cancel()

	select {
	case err := <-errCh:
		if err != nil {
			t.Errorf("expected clean shutdown, got error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("ServeSSE did not shut down in time")
	}
}
