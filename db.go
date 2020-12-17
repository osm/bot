package main

import (
	"database/sql"
	"fmt"
	"os"
)

// initDB opens a new connection to the database, it will also run all
// migrations to make sure that the schema is migrated to the latest version
// possible. An error will be returned if there's any.
func (b *bot) initDB() error {
	dbPath := b.DB.Path
	if p := os.Getenv("BOT_DB_PATH"); p != "" {
		dbPath = p
	}

	if dbPath == "" {
		return fmt.Errorf("database path can't be empty")
	}

	if b.DB.Engine != "postgres" {
		return b.initDBSqlite(dbPath)
	} else {
		return b.initDBPostgres(dbPath)
	}
}

// query sends the given query to the database, it returns the rows and an
// error if there is one.
func (b *bot) query(query string, args ...interface{}) (*sql.Rows, error) {
	return b.DB.client.Query(query, args...)
}

// queryRow sends the given query to the database and returns a row.
func (b *bot) queryRow(query string, args ...interface{}) *sql.Row {
	return b.DB.client.QueryRow(query, args...)
}

// prepare returns a statement for the given query or an error if there is
// one.
func (b *bot) prepare(query string) (*sql.Stmt, error) {
	return b.DB.client.Prepare(query)
}
