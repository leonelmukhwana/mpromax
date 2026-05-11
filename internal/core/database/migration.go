package database

import (
	"errors"
	"fmt"
	"log"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func RunMigrations(databaseURL string) {
	fmt.Println("Running database migrations...")

	// 1. Initialize the migrate instance
	// This looks for your SQL files in the db/migrations folder
	m, err := migrate.New("file://db/migrations", databaseURL)
	if err != nil {
		log.Fatalf("Migration init failed: %v", err)
	}

	// 2. Apply migrations
	if err := m.Up(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			fmt.Println("Database is already up to date.")
		} else {
			log.Fatalf("Migration failed: %v", err)
		}
	} else {
		fmt.Println("Migrations applied successfully!")
	}
}
