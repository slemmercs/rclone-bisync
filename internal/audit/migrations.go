package audit

import (
	"database/sql"
	"embed"
	"fmt"

	"github.com/pressly/goose/v3"
)

//go:embed migrations/*.sql
var embedMigrations embed.FS

// RunMigrations executes pending database schema upgrades automatically on application boot.
func RunMigrations(db *sql.DB) error {
	goose.SetLogger(goose.NopLogger())
	goose.SetBaseFS(embedMigrations)
	if err := goose.SetDialect("sqlite3"); err != nil {
		return fmt.Errorf("failed to set goose sqlite3 dialect: %w", err)
	}

	if err := goose.Up(db, "migrations"); err != nil {
		return fmt.Errorf("failed to apply audit schema migrations: %w", err)
	}
	return nil
}
