package main

import (
	"database/sql"
	"fmt"
	"os"

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
		1:  "CREATE TABLE migration (version TEXT NOT NULL PRIMARY KEY);",
		2:  "CREATE TABLE log (id VARCHAR(36) NOT NULL PRIMARY KEY, timestamp TEXT NOT NULL, nick TEXT NOT NULL, message TEXT NOT NULL); CREATE INDEX log_timestamp ON log(timestamp); CREATE INDEX log_nick_timestamp ON log(nick, timestamp);",
		3:  "CREATE TABLE url_check (id VARCHAR(36) NOT NULL PRIMARY KEY, timestamp TEXT NOT NULL, nick TEXT NOT NULL, url TEXT NOT NULL); CREATE INDEX url_check_url ON url_check(url);",
		4:  "CREATE TABLE factoid (id VARCHAR(36) NOT NULL PRIMARY KEY, timestamp TEXT NOT NULL, author TEXT NOT NULL, trigger TEXT NOT NULL, reply TEXT NOT NULL, is_deleted BOOLEAN NOT NULL); CREATE INDEX factoid_trigger ON factoid(trigger);",
		5:  "CREATE TABLE cron (id VARCHAR(36) NOT NULL PRIMARY KEY, expression TEXT NOT NULL, message TEXT NOT NULL, is_deleted BOOLEAN NOT NULL, inserted_at TEXT NOT NULL, updated_at TEXT);",
		6:  "ALTER TABLE cron ADD COLUMN is_limited BOOL NOT NULL DEFAULT false;",
		7:  "ALTER TABLE cron ADD COLUMN exec_limit INT NOT NULL DEFAULT 0;",
		8:  "ALTER TABLE cron ADD COLUMN exec_count INT NOT NULL DEFAULT 0;",
		9:  "CREATE TABLE quiz_stat (id VARCHAR(36) NOT NULL PRIMARY KEY, nick TEXT NOT NULL, quiz_round_id VARCHAR(36) NOT NULL, quiz_name TEXT NOT NULL, category TEXT NOT NULL, question TEXT NOT NULL, answer TEXT NOT NULL, inserted_at TEXT NOT NULL);",
		10: "CREATE TABLE supernytt (id VARCHAR(36) NOT NULL PRIMARY KEY, external_id TEXT NOT NULL, title TEXT NOT NULL, content TEXT NOT NULL, external_created TEXT NOT NULL, inserted_at timestamp NOT NULL);",
		11: "ALTER TABLE factoid ADD COLUMN rate INTEGER;",
		12: "CREATE TABLE march (id VARCHAR(36) NOT NULL PRIMARY KEY, url TEXT NOT NULL, foreign_id VARCHAR(36) NOT NULL, inserted_at TIMESTAMP NOT NULL); CREATE INDEX march_url ON march(url);",
		13: `
			CREATE TABLE smhi_forecast (
				id TIMESTAMP NOT NULL PRIMARY KEY,
				inserted_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
				updated_at TIMESTAMP NOT NULL,
				name TEXT NOT NULL,
				air_pressure REAL NOT NULL,
				air_temperature REAL NOT NULL,
				horizontal_visibility REAL NOT NULL,
				maximum_precipitation_intensity REAL NOT NULL,
				mean_precipitation_intensity REAL NOT NULL,
				mean_value_of_high_level_cloud_cover INTEGER NOT NULL,
				mean_value_of_low_level_cloud_cover INTEGER NOT NULL,
				mean_value_of_medium_level_cloud_cover INTEGER NOT NULL,
				mean_value_of_total_cloud_cover INTEGER NOT NULL,
				median_precipitation_intensity REAL NOT NULL,
				minimum_precipitation_intensity REAL NOT NULL,
				percent_of_precipitation_in_frozen_form INTEGER NOT NULL,
				precipitation_category INTEGER NOT NULL,
				precipitation_category_description TEXT NOT NULL,
				relative_humidity INTEGER NOT NULL,
				thunder_probability INTEGER NOT NULL,
				weather_symbol INTEGER NOT NULL,
				weather_symbol_description TEXT NOT NULL,
				wind_direction INTEGER NOT NULL,
				wind_gust_speed REAL NOT NULL,
				wind_speed REAL NOT NULL
			);
		`,
		14: "CREATE INDEX smhi_forecast_name ON smhi_forecast(name);",
	})
}

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

	var err error
	if b.DB.client, err = sql.Open("sqlite3", dbPath); err != nil {
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
