package db

import (
	"errors"
	"fmt"
	"log"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres" // регистрация драйвера Postgres
	_ "github.com/golang-migrate/migrate/v4/source/file"       // регистрация файлового источника
	_ "github.com/lib/pq"                                      // регистрация драйвера Postgres для миграций
)

// RunMigrations применяет миграции базы данных.
func RunMigrations(dsn string) error {
	migrationsPath := "file://./migrations"
	m, err := migrate.New(migrationsPath, dsn)
	if err != nil {
		return fmt.Errorf("failed to init migrations: %v", err)
	}

	log.Println("Migration files found. Applying migrations...")

	if err := m.Up(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			log.Println("No new migrations to apply. Database is up-to-date.")
		} else {
			return fmt.Errorf("failed to apply migrations: %v", err)
		}
	} else {
		log.Println("Migrations applied successfully.")
	}
	return nil
}
