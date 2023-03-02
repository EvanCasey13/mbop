package store

import (
	"embed"
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx"
	"github.com/redhatinsights/mbop/internal/config"

	// this is the iofs:// driver for go-migrate.
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

//go:embed migrations
var migrations embed.FS

func migrateDatabase() error {
	fs, err := iofs.New(migrations, "migrations")
	if err != nil {
		return err
	}

	c := config.Get()
	connStr := fmt.Sprintf("pgx://%s:%s@%s:%s/%s?sslmode=prefer",
		c.DatabaseUser, c.DatabasePassword, c.DatabaseHost, c.DatabasePort, c.DatabaseName)

	m, err := migrate.NewWithSourceInstance("iofs", fs, connStr)
	if err != nil {
		return err
	}

	err = m.Up()
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}

	return nil
}
