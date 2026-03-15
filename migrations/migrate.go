package migrations

import (
	"errors"
	"fmt"
	"os"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

// migrationDSN builds a DSN from env vars for golang-migrate.
// golang-migrate needs its own DSN — it doesn't share the GORM connection.
func migrationDSN() string {
	sslMode := "require"
	if os.Getenv("CHECK_SSL") == "0" {
		sslMode = "disable"
	}

	env := os.Getenv("APP_ENV")
	if env == "" {
		env = "development"
	}

	var user, password, host, port, name string
	switch env {
	case "development":
		user = os.Getenv("DB_USER_DEV")
		password = os.Getenv("DB_PASS_DEV")
		host = os.Getenv("DB_HOST_DEV")
		port = os.Getenv("DB_PORT_DEV")
		name = os.Getenv("DB_NAME_DEV")
	case "test":
		user = os.Getenv("DB_USER_TEST")
		password = os.Getenv("DB_PASSWORD_TEST")
		host = os.Getenv("DB_HOST_TEST")
		port = os.Getenv("DB_PORT_TEST")
		name = os.Getenv("DB_NAME_TEST")
	case "staging", "production":
		user = os.Getenv("DB_USER")
		password = os.Getenv("DB_PASSWORD")
		host = os.Getenv("DB_HOST")
		port = os.Getenv("DB_PORT")
		name = os.Getenv("DB_NAME")
	default:
		user = os.Getenv("DB_USER_DEV")
		password = os.Getenv("DB_PASS_DEV")
		host = os.Getenv("DB_HOST_DEV")
		port = os.Getenv("DB_PORT_DEV")
		name = os.Getenv("DB_NAME_DEV")
	}

	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		user, password, host, port, name, sslMode,
	)
}

// migrationsPath returns the path to the migrations folder.
// Defaults to "migrations" relative to the working directory.
func migrationsPath() string {
	p := os.Getenv("MIGRATIONS_PATH")
	if p == "" {
		return "file://migrations"
	}
	return "file://" + p
}

// newMigrate constructs a migrate.Migrate instance
func newMigrate() (*migrate.Migrate, error) {
	m, err := migrate.New(migrationsPath(), migrationDSN())
	if err != nil {
		return nil, fmt.Errorf("failed to init migrate: %w", err)
	}
	return m, nil
}

// Up runs all pending migrations
func Up() error {
	m, err := newMigrate()
	if err != nil {
		return err
	}
	defer m.Close()

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("migration up failed: %w", err)
	}

	return nil
}

// Down rolls back all migrations
func Down() error {
	m, err := newMigrate()
	if err != nil {
		return err
	}
	defer m.Close()

	if err := m.Down(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("migration down failed: %w", err)
	}

	return nil
}

// Steps runs n migrations forward (positive) or backward (negative)
func Steps(n int) error {
	m, err := newMigrate()
	if err != nil {
		return err
	}
	defer m.Close()

	if err := m.Steps(n); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("migration steps failed: %w", err)
	}

	return nil
}

// Version returns the current migration version and whether it's dirty
func Version() (uint, bool, error) {
	m, err := newMigrate()
	if err != nil {
		return 0, false, err
	}
	defer m.Close()

	return m.Version()
}

// Force sets the migration version manually — useful to fix a dirty state
func Force(version int) error {
	m, err := newMigrate()
	if err != nil {
		return err
	}
	defer m.Close()

	return m.Force(version)
}
