package db

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/golang-migrate/migrate/v4"
	// Import postgres to give it access
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

// OpenDB returns a *DB or an error
func OpenDB(ctx context.Context) (*DB, error) {
	if os.Getenv("env") != "production" {
		// return development configuration
		return Open("postgres", connString(
			"localhost",
			5432,
			"postgres",
			"",
			"",
		))
	}

	// return the production configuration
	port, err := strconv.Atoi(os.Getenv("DB_PORT"))
	if err != nil {
		return nil, err
	}

	return Open("postgres", connString(
		os.Getenv("DB_HOST"),
		port,
		os.Getenv("DB_USERNAME"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_DATABASE"),
	))
}

// WithTx gives you a callback function that exposes any errors that occur
func (db *DB) WithTx(ctx context.Context,
	fn func(context.Context, *Tx) error) (err error) {

	tx, err := db.Open(ctx)
	if err != nil {
		return err
	}
	defer func() {
		if err == nil {
			err = tx.Commit()
		} else {
			log.Printf("tx rolled back - %+v", tx.Tx)
			tx.Rollback() // log this perhaps?
		}
	}()
	return fn(ctx, tx)
}

// Migrate runs all of the migrations in `./migrations` against the DB on startup
func (db *DB) Migrate(driver string, migrations string) error {
	// ensure DB connection
	if err := db.Ping(); err != nil {
		log.Printf("could not ping db: %s", err)
		return err
	}

	m, err := migrate.New(
		migrations,
		driver,
	)
	if err != nil {
		log.Fatalf("could not start migrations: %+v", err)
		return err
	}

	// run the Up migrations
	if err := m.Up(); err != nil {
		if err.Error() == "no change" {
			log.Print("DB: migrations are up to date")
			return nil
		}

		log.Fatalf("failed to run migrations: %+v", err)
		return err
	}

	return nil
}

// returns a connection string
func connString(host string, port int, user, password, dbname string) string {
	return fmt.Sprintf("host=%s port=%d user=%s "+
		"password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)
}
