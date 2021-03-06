package main

import (
	"database/sql"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
	"github.com/osm/migrator"
	"github.com/osm/migrator/repository"
)

// initSqlDBSqlite initializes a sqlite database.
func (b *bot) initDBSqlite(dbPath string) error {
	var err error
	if b.DB.client, err = sql.Open("sqlite3", dbPath); err != nil {
		return fmt.Errorf("can't initialize database connection: %v", err)
	}

	return migrator.ToLatest(b.DB.client, getDatabaseRepositorySqlite())
}

// getDatabaseRepositorySqlite returns an in memory repository with the
// database migrations. It is not supported to edit a existing migration, you
// should only do this if you are prepared to wipe your already existing
// database. If you need to alter an existing table you have to write a new
// migration entry that contains the SQL alters the table in the way you want.
func getDatabaseRepositorySqlite() repository.Source {
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
		15: "ALTER TABLE smhi_forecast ADD COLUMN timestamp TIMESTAMP;",
		16: `
			DROP INDEX smhi_forecast_name;
			CREATE INDEX smhi_forecast_name_timestamp ON smhi_forecast(name, timestamp);
		`,
		17: "ALTER TABLE smhi_forecast ADD COLUMN hash TEXT",
		18: "ALTER TABLE smhi_forecast ADD COLUMN wind_speed_description TEXT",
		19: `
			CREATE TABLE parcel_tracking (
				id VARCHAR(36) NOT NULL PRIMARY KEY,
				alias TEXT NOT NULL,
				parcel_tracking_id TEXT NOT NULL,
				inserted_at TEXT NOT NULL,
				is_deleted BOOLEAN NOT NULL
			);
			CREATE INDEX parcel_tracking_alias_id ON parcel_tracking(alias, parcel_tracking_id);
		`,
		20: `ALTER TABLE parcel_tracking ADD COLUMN nick TEXT;`,
	})
}
