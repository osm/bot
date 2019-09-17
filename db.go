package main

import (
	"database/sql"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
	"github.com/osm/migrator"
	"github.com/osm/migrator/repository"
)

// getDatabaseRepository returns an in memory repository with the database
// migrations. It is not supported to edit a existing migration, you should
// only do this if you are prepared to wipe your already existing database. If
// you need to alter an existing table you have to write a new migration entry
// that contains the SQL alters the table in the way you want.
func getDatabaseRepository() repository.Source {
	return repository.FromMemory(map[int]string{
		1: "CREATE TABLE migration (version TEXT NOT NULL PRIMARY KEY);",
		2: "CREATE TABLE log (id VARCHAR(36) NOT NULL PRIMARY KEY, timestamp TEXT NOT NULL, nick TEXT NOT NULL, message TEXT NOT NULL); CREATE INDEX log_timestamp ON log(timestamp); CREATE INDEX log_nick_timestamp ON log(nick, timestamp);",
		3: "CREATE TABLE url_check (id VARCHAR(36) NOT NULL PRIMARY KEY, timestamp TEXT NOT NULL, nick TEXT NOT NULL, url TEXT NOT NULL); CREATE INDEX url_check_url ON url_check(url);",
		4: "CREATE TABLE factoid (id VARCHAR(36) NOT NULL PRIMARY KEY, timestamp TEXT NOT NULL, author TEXT NOT NULL, trigger TEXT NOT NULL, reply TEXT NOT NULL, is_deleted BOOLEAN NOT NULL); CREATE INDEX factoid_trigger ON factoid(trigger);",
		5: "CREATE TABLE cron (id VARCHAR(36) NOT NULL PRIMARY KEY, expression TEXT NOT NULL, message TEXT NOT NULL, is_deleted BOOLEAN NOT NULL, inserted_at TEXT NOT NULL, updated_at TEXT);",
		6: "ALTER TABLE cron ADD COLUMN is_limited BOOL NOT NULL DEFAULT false;",
		7: "ALTER TABLE cron ADD COLUMN exec_limit INT NOT NULL DEFAULT 0;",
		8: "ALTER TABLE cron ADD COLUMN exec_count INT NOT NULL DEFAULT 0;",
	})
}

// initDB opens a new connection to the database, it will also run all
// migrations to make sure that the schema is migrated to the latest version
// possible. An error will be returned if there's any.
func (b *bot) initDB() error {
	if b.DB.Path == "" {
		return fmt.Errorf("database path can't be empty")
	}

	var err error
	if b.DB.client, err = sql.Open("sqlite3", b.DB.Path); err != nil {
		return fmt.Errorf("can't initialize database connection: %v", err)
	}

	return migrator.ToLatest(b.DB.client, getDatabaseRepository())
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
