package database

import (
	"database/sql"
	"fmt"

	_ "github.com/jackc/pgx/v5/stdlib" // register pgx database/sql driver
	_ "modernc.org/sqlite"             // register sqlite database/sql driver
)

// New opens a database connection based on the given driver and DSN.
// Supported drivers: "sqlite" and "postgres".
func New(driver, dsn string) (*sql.DB, error) {
	switch driver {
	case "sqlite":
		return openSQLite(dsn)
	case "postgres":
		return openPostgres(dsn)
	default:
		return nil, fmt.Errorf("unsupported database driver: %s", driver)
	}
}

func openSQLite(dsn string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("opening sqlite database: %w", err)
	}

	// Enable WAL mode for better concurrent read performance.
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		db.Close()
		return nil, fmt.Errorf("enabling WAL mode: %w", err)
	}

	// Set busy timeout so concurrent writers retry instead of failing immediately.
	if _, err := db.Exec("PRAGMA busy_timeout=5000"); err != nil {
		db.Close()
		return nil, fmt.Errorf("setting busy timeout: %w", err)
	}

	// Enable foreign key enforcement.
	if _, err := db.Exec("PRAGMA foreign_keys=ON"); err != nil {
		db.Close()
		return nil, fmt.Errorf("enabling foreign keys: %w", err)
	}

	return db, nil
}

func openPostgres(dsn string) (*sql.DB, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("opening postgres database: %w", err)
	}

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("pinging postgres database: %w", err)
	}

	return db, nil
}
