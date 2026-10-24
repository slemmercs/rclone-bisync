package audit

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveDBPath(t *testing.T) {
	tmpDir := t.TempDir()

	// 1. Explicit override
	customPath := filepath.Join(tmpDir, "custom", "test_audit.db")
	resolved, err := ResolveDBPath(customPath)
	if err != nil {
		t.Fatalf("unexpected error resolving override path: %v", err)
	}
	if resolved != customPath {
		t.Fatalf("expected resolved path %s, got %s", customPath, resolved)
	}
	if _, err := os.Stat(filepath.Dir(customPath)); err != nil {
		t.Fatalf("expected directory to exist: %v", err)
	}

	// 2. Environment variable override
	envPath := filepath.Join(tmpDir, "env", "audit.db")
	t.Setenv(EnvAuditDBPath, envPath)
	resolvedEnv, err := ResolveDBPath("")
	if err != nil {
		t.Fatalf("unexpected error resolving env path: %v", err)
	}
	if resolvedEnv != envPath {
		t.Fatalf("expected resolved env path %s, got %s", envPath, resolvedEnv)
	}

	// 3. Default path (unset env)
	t.Setenv(EnvAuditDBPath, "")
	resolvedDef, err := ResolveDBPath("")
	if err != nil {
		t.Fatalf("unexpected error resolving default path: %v", err)
	}
	if !filepath.IsAbs(resolvedDef) {
		t.Fatalf("expected absolute path, got %s", resolvedDef)
	}
	if filepath.Base(resolvedDef) != DefaultDBFilename {
		t.Fatalf("expected filename %s, got %s", DefaultDBFilename, filepath.Base(resolvedDef))
	}
}

func TestMigrationsAndDBOpen(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_audit.db")

	db, err := OpenDB(dbPath)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	// Verify table exists
	var tableName string
	err = db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='audit_logs'").Scan(&tableName)
	if err != nil {
		t.Fatalf("audit_logs table does not exist: %v", err)
	}
	if tableName != "audit_logs" {
		t.Fatalf("expected table name audit_logs, got %s", tableName)
	}

	// Verify triggers exist
	triggers := []string{"prevent_audit_update", "prevent_audit_delete"}
	for _, trig := range triggers {
		var name string
		err = db.QueryRow("SELECT name FROM sqlite_master WHERE type='trigger' AND name=?", trig).Scan(&name)
		if err != nil {
			t.Fatalf("trigger %s does not exist: %v", trig, err)
		}
	}
}

func TestOpenDB_DefaultPath(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv(EnvAuditDBPath, filepath.Join(tmpDir, "auto.db"))

	db, err := OpenDB("")
	if err != nil {
		t.Fatalf("failed to open default path db: %v", err)
	}
	defer db.Close()
}
