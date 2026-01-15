package migrate

import (
	"database/sql"
	"embed"
	"fmt"
	"log"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

// Config holds migration configuration
type Config struct {
	ServiceName string
	MigrationsFS embed.FS
	MigrationsPath string
}

// Run executes all pending migrations
func Run(db *sql.DB, cfg Config) error {
	log.Printf("[%s] Starting database migrations...", cfg.ServiceName)

	// Create source driver from embedded filesystem
	source, err := iofs.New(cfg.MigrationsFS, cfg.MigrationsPath)
	if err != nil {
		return fmt.Errorf("failed to create migration source: %w", err)
	}

	// Create database driver
	driver, err := postgres.WithInstance(db, &postgres.Config{
		MigrationsTable: fmt.Sprintf("%s_migrations", cfg.ServiceName),
	})
	if err != nil {
		return fmt.Errorf("failed to create database driver: %w", err)
	}

	// Create migrator
	m, err := migrate.NewWithInstance("iofs", source, "postgres", driver)
	if err != nil {
		return fmt.Errorf("failed to create migrator: %w", err)
	}

	// Run migrations
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	version, dirty, err := m.Version()
	if err != nil && err != migrate.ErrNilVersion {
		return fmt.Errorf("failed to get migration version: %w", err)
	}

	if err == migrate.ErrNilVersion {
		log.Printf("[%s] No migrations applied yet", cfg.ServiceName)
	} else {
		log.Printf("[%s] Migrations completed. Current version: %d (dirty: %v)", cfg.ServiceName, version, dirty)
	}

	return nil
}

// RunWithDSN runs migrations using a DSN string
func RunWithDSN(dsn string, cfg Config) error {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer db.Close()

	return Run(db, cfg)
}
