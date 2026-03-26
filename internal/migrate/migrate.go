package migrate

import (
	"database/sql"
	"embed"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/mysql"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

//go:embed migrations/*.sql
var migrations embed.FS

func Run(db *sql.DB) (int, error) {
	source, err := iofs.New(migrations, "migrations")
	if err != nil {
		return 0, fmt.Errorf("creating migration source: %w", err)
	}

	driver, err := mysql.WithInstance(db, &mysql.Config{})
	if err != nil {
		return 0, fmt.Errorf("creating migration driver: %w", err)
	}

	m, err := migrate.NewWithInstance("iofs", source, "mysql", driver)
	if err != nil {
		return 0, fmt.Errorf("creating migrator: %w", err)
	}

	vBefore, _, _ := m.Version()

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return 0, fmt.Errorf("running migrations: %w", err)
	}

	vAfter, _, _ := m.Version()
	applied := int(vAfter) - int(vBefore)
	if applied < 0 {
		applied = 0
	}

	return applied, nil
}

// Clean drops all migrations and re-applies them from scratch.
func Clean(db *sql.DB) (int, error) {
	source, err := iofs.New(migrations, "migrations")
	if err != nil {
		return 0, fmt.Errorf("creating migration source: %w", err)
	}

	driver, err := mysql.WithInstance(db, &mysql.Config{})
	if err != nil {
		return 0, fmt.Errorf("creating migration driver: %w", err)
	}

	m, err := migrate.NewWithInstance("iofs", source, "mysql", driver)
	if err != nil {
		return 0, fmt.Errorf("creating migrator: %w", err)
	}

	if err := m.Drop(); err != nil {
		return 0, fmt.Errorf("dropping database: %w", err)
	}

	// After Drop, golang-migrate removes the schema_migrations table.
	// Re-create the migrator since the driver state is stale.
	driver, err = mysql.WithInstance(db, &mysql.Config{})
	if err != nil {
		return 0, fmt.Errorf("creating migration driver: %w", err)
	}

	m, err = migrate.NewWithInstance("iofs", source, "mysql", driver)
	if err != nil {
		return 0, fmt.Errorf("creating migrator: %w", err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return 0, fmt.Errorf("running migrations: %w", err)
	}

	vAfter, _, _ := m.Version()
	return int(vAfter), nil
}
