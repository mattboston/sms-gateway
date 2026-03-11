package database

import (
	"database/sql"
	"embed"
	"fmt"

	"github.com/pressly/goose/v3"
)

// RunMigrations runs all pending database migrations.
func RunMigrations(db *sql.DB, driver string, migrationsFS embed.FS) error {
	goose.SetBaseFS(migrationsFS)

	if err := goose.SetDialect(gooseDriverName(driver)); err != nil {
		return fmt.Errorf("setting goose dialect: %w", err)
	}

	if err := goose.Up(db, "migrations"); err != nil {
		return fmt.Errorf("running migrations up: %w", err)
	}

	return nil
}

// MigrateDown rolls back the last migration.
func MigrateDown(db *sql.DB, driver string, migrationsFS embed.FS) error {
	goose.SetBaseFS(migrationsFS)

	if err := goose.SetDialect(gooseDriverName(driver)); err != nil {
		return fmt.Errorf("setting goose dialect: %w", err)
	}

	if err := goose.Down(db, "migrations"); err != nil {
		return fmt.Errorf("running migration down: %w", err)
	}

	return nil
}

// MigrateStatus prints the migration status.
func MigrateStatus(db *sql.DB, driver string, migrationsFS embed.FS) error {
	goose.SetBaseFS(migrationsFS)

	if err := goose.SetDialect(gooseDriverName(driver)); err != nil {
		return fmt.Errorf("setting goose dialect: %w", err)
	}

	if err := goose.Status(db, "migrations"); err != nil {
		return fmt.Errorf("getting migration status: %w", err)
	}

	return nil
}

func gooseDriverName(driver string) string {
	switch driver {
	case "postgres":
		return "postgres"
	default:
		return "sqlite3"
	}
}
