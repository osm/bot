package main

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"
	"github.com/osm/migrator"
	"github.com/osm/migrator/repository"
)

// initSqlDBPostgres initializes a postgres database.
func (b *bot) initDBPostgres(dbPath string) error {
	var err error
	if b.DB.client, err = sql.Open("postgres", dbPath); err != nil {
		return fmt.Errorf("can't initialize database connection: %v", err)
	}

	return migrator.ToLatest(b.DB.client, getDatabaseRepositoryPostgres())
}

// getDatabaseRepositoryPostgres returns an in memory repository with the
// database migrations. It is not supported to edit a existing migration, you
// should only do this if you are prepared to wipe your already existing
// database. If you need to alter an existing table you have to write a new
// migration entry that contains the SQL alters the table in the way you want.
func getDatabaseRepositoryPostgres() repository.Source {
	return repository.FromMemory(map[int]string{
		1: `
			CREATE TABLE migration (
				version int NOT NULL PRIMARY KEY
			);
		`,
		2: `
			CREATE TABLE log (
				id uuid NOT NULL PRIMARY KEY,
				timestamp timestamp NOT NULL,
				nick text NOT NULL,
				message text NOT NULL
			);
			CREATE INDEX log_timestamp ON log(timestamp);
			CREATE INDEX log_nick_timestamp ON log(nick, timestamp);
		`,
		3: `
			CREATE TABLE url_check (
				id uuid NOT NULL PRIMARY KEY,
				timestamp timestamp NOT NULL,
				nick text NOT NULL,
				url text NOT NULL
			);
			CREATE INDEX url_check_url ON url_check(url);
		`,
		4: `
			CREATE TABLE factoid (
				id uuid NOT NULL PRIMARY KEY,
				timestamp timestamp NOT NULL,
				author text NOT NULL,
				trigger text NOT NULL,
				reply text NOT NULL,
				is_deleted boolean NOT NULL
			);
			CREATE INDEX factoid_trigger ON factoid(trigger);
		`,
		5: `
			CREATE TABLE cron (
				id uuid NOT NULL PRIMARY KEY,
				expression text NOT NULL,
				message text NOT NULL,
				is_deleted boolean NOT NULL,
				inserted_at timestamp NOT NULL,
				updated_at timestamp
			);
		`,
		6: `
			ALTER TABLE cron
				ADD COLUMN is_limited boolean NOT NULL DEFAULT false;
		`,
		7: `
			ALTER TABLE cron
				ADD COLUMN exec_limit int NOT NULL DEFAULT 0;
		`,
		8: `
			ALTER TABLE cron
				ADD COLUMN exec_count int NOT NULL DEFAULT 0;
		`,
		9: `
			CREATE TABLE quiz_stat (
				id uuid NOT NULL PRIMARY KEY,
				nick text NOT NULL,
				quiz_round_id uuid NOT NULL,
				quiz_name text NOT NULL,
				category text NOT NULL,
				question text NOT NULL,
				answer text NOT NULL,
				inserted_at timestamp NOT NULL
			);
		`,
		10: `
			CREATE TABLE supernytt (
				id uuid NOT NULL PRIMARY KEY,
				external_id text NOT NULL,
				title text NOT NULL,
				content text NOT NULL,
				external_created text NOT NULL,
				inserted_at timestamp NOT NULL
			);
		`,
		11: `
			ALTER TABLE factoid
				ADD COLUMN rate int;
		`,
		12: `
			CREATE TABLE march (
				id uuid NOT NULL PRIMARY KEY,
				url text NOT NULL,
				foreign_id uuid NOT NULL,
				inserted_at TIMESTAMP NOT NULL
			);
			CREATE INDEX march_url ON march(url);
		`,
		13: `
			CREATE TABLE smhi_forecast (
				id text NOT NULL PRIMARY KEY,
				inserted_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
				updated_at TIMESTAMP NOT NULL,
				name text NOT NULL,
				air_pressure numeric NOT NULL,
				air_temperature numeric NOT NULL,
				horizontal_visibility numeric NOT NULL,
				maximum_precipitation_intensity numeric NOT NULL,
				mean_precipitation_intensity numeric NOT NULL,
				mean_value_of_high_level_cloud_cover int NOT NULL,
				mean_value_of_low_level_cloud_cover int NOT NULL,
				mean_value_of_medium_level_cloud_cover int NOT NULL,
				mean_value_of_total_cloud_cover int NOT NULL,
				median_precipitation_intensity numeric NOT NULL,
				minimum_precipitation_intensity numeric NOT NULL,
				percent_of_precipitation_in_frozen_form int NOT NULL,
				precipitation_category int NOT NULL,
				precipitation_category_description text NOT NULL,
				relative_humidity int NOT NULL,
				thunder_probability int NOT NULL,
				weather_symbol int NOT NULL,
				weather_symbol_description text NOT NULL,
				wind_direction int NOT NULL,
				wind_gust_speed numeric NOT NULL,
				wind_speed numeric NOT NULL
			);
		`,
		14: `
			CREATE INDEX smhi_forecast_name ON smhi_forecast(name);
		`,
		15: `
			ALTER TABLE smhi_forecast
				ADD COLUMN timestamp TIMESTAMP;
		`,
		16: `
			DROP INDEX smhi_forecast_name;
			CREATE INDEX smhi_forecast_name_timestamp ON smhi_forecast(name, timestamp);
		`,
		17: `
			ALTER TABLE smhi_forecast
				ADD COLUMN hash text;
		`,
		18: `
			ALTER TABLE smhi_forecast
				ADD COLUMN wind_speed_description text;
		`,
		19: `
			CREATE TABLE parcel_tracking (
				id uuid NOT NULL PRIMARY KEY,
				alias text NOT NULL,
				parcel_tracking_id text NOT NULL,
				inserted_at text NOT NULL,
				is_deleted boolean NOT NULL
			);
			CREATE INDEX parcel_tracking_alias_id ON parcel_tracking(alias, parcel_tracking_id);
		`,
		20: `ALTER TABLE parcel_tracking
			ADD COLUMN nick text;`,
	})
}
