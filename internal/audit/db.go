package audit

import (
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

// OpenDB opens a connection to the SQLite audit database at dbPath,
// configures WAL mode and connection pooling, and executes migrations.
func OpenDB(dbPath string) (*sql.DB, error) {
	if dbPath == "" {
		resolved, err := ResolveDBPath("")
		if err != nil {
			return nil, fmt.Errorf("failed to resolve default audit DB path: %w", err)
		}
		dbPath = resolved
	}

	dsn := fmt.Sprintf("file:%s?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)", dbPath)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite database at %s: %w", dbPath, err)
	}

	// For embedded SQLite, limit open connections to avoid lock contention
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(time.Hour)

	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to ping sqlite database: %w", err)
	}

	if err := RunMigrations(db); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to apply migrations: %w", err)
	}

	return db, nil
}
