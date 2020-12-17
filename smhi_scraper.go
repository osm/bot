package main

import (
	"fmt"
	"time"

	"github.com/osm/smhi"
)

const SMHI_FORECAST_HASH_SELECT_SQL = `SELECT
	hash
FROM smhi_forecast
WHERE
	name = $1 AND
	timestamp >= current_timestamp;`

const SMHI_FORECAST_INSERT_SQL_SQLITE = `INSERT OR REPLACE INTO smhi_forecast (
	id,
	hash,
	updated_at,
	timestamp,
	name,
	air_pressure,
	air_temperature,
	horizontal_visibility,
	maximum_precipitation_intensity,
	mean_precipitation_intensity,
	mean_value_of_high_level_cloud_cover,
	mean_value_of_low_level_cloud_cover,
	mean_value_of_medium_level_cloud_cover,
	mean_value_of_total_cloud_cover,
	median_precipitation_intensity,
	minimum_precipitation_intensity,
	percent_of_precipitation_in_frozen_form,
	precipitation_category,
	precipitation_category_description,
	relative_humidity,
	thunder_probability,
	weather_symbol,
	weather_symbol_description,
	wind_direction,
	wind_gust_speed,
	wind_speed,
	wind_speed_description
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);`

const SMHI_FORECAST_INSERT_SQL_POSTGRES = `INSERT INTO smhi_forecast (
	id,
	hash,
	updated_at,
	timestamp,
	name,
	air_pressure,
	air_temperature,
	horizontal_visibility,
	maximum_precipitation_intensity,
	mean_precipitation_intensity,
	mean_value_of_high_level_cloud_cover,
	mean_value_of_low_level_cloud_cover,
	mean_value_of_medium_level_cloud_cover,
	mean_value_of_total_cloud_cover,
	median_precipitation_intensity,
	minimum_precipitation_intensity,
	percent_of_precipitation_in_frozen_form,
	precipitation_category,
	precipitation_category_description,
	relative_humidity,
	thunder_probability,
	weather_symbol,
	weather_symbol_description,
	wind_direction,
	wind_gust_speed,
	wind_speed,
	wind_speed_description
) VALUES (
	$1,
	$2,
	$3,
	$4,
	$5,
	$6,
	$7,
	$8,
	$9,
	$10,
	$11,
	$12,
	$13,
	$14,
	$15,
	$16,
	$17,
	$18,
	$19,
	$20,
	$21,
	$22,
	$23,
	$24,
	$25,
	$26,
	$27
) ON CONFLICT (id)
DO UPDATE
SET
	id = EXCLUDED.id,
	hash = EXCLUDED.hash,
	updated_at = EXCLUDED.updated_at,
	timestamp = EXCLUDED.timestamp,
	name = EXCLUDED.name,
	air_pressure = EXCLUDED.air_pressure,
	air_temperature = EXCLUDED.air_temperature,
	horizontal_visibility = EXCLUDED.horizontal_visibility,
	maximum_precipitation_intensity = EXCLUDED.maximum_precipitation_intensity,
	mean_precipitation_intensity = EXCLUDED.mean_precipitation_intensity,
	mean_value_of_high_level_cloud_cover = EXCLUDED.mean_value_of_high_level_cloud_cover,
	mean_value_of_low_level_cloud_cover = EXCLUDED.mean_value_of_low_level_cloud_cover,
	mean_value_of_medium_level_cloud_cover = EXCLUDED.mean_value_of_medium_level_cloud_cover,
	mean_value_of_total_cloud_cover = EXCLUDED.mean_value_of_total_cloud_cover,
	median_precipitation_intensity = EXCLUDED.median_precipitation_intensity,
	minimum_precipitation_intensity = EXCLUDED.minimum_precipitation_intensity,
	percent_of_precipitation_in_frozen_form = EXCLUDED.percent_of_precipitation_in_frozen_form,
	precipitation_category = EXCLUDED.precipitation_category,
	precipitation_category_description = EXCLUDED.precipitation_category_description,
	relative_humidity = EXCLUDED.relative_humidity,
	thunder_probability = EXCLUDED.thunder_probability,
	weather_symbol = EXCLUDED.weather_symbol,
	weather_symbol_description = EXCLUDED.weather_symbol_description,
	wind_direction = EXCLUDED.wind_direction,
	wind_gust_speed = EXCLUDED.wind_gust_speed,
	wind_speed = EXCLUDED.wind_speed,
	wind_speed_description = EXCLUDED.wind_speed_description
`

// smhiGetForecasts runs once every hour and fetches new forecasts for all the
// locations that is defined in the config. The forecast is saved to the
// database. We'll always wipe
func (b *bot) smhiGetForecasts() {
	var fc *smhi.PointForecast

	for {
		// Iterate over the locations and fetch a forecast for the given
		// coordinates.
		for name, coord := range b.IRC.SMHIForecastLocations {
			// We don't fetch data for aliases.
			if len(coord.Alias) > 0 {
				continue
			}

			// But first of all, let's find all forecasts from now
			// on and in the future and construct a map of them
			// based by their hash. This will be used to determine
			// whether or not we need to update the entry when we
			// get new data from the SMHI API.
			rows, err := b.query(SMHI_FORECAST_HASH_SELECT_SQL, name)
			if err != nil {
				b.logger.Printf("smhiGetForecasts: %v", err)
				return
			}
			defer rows.Close()

			var forecasts map[string]bool = make(map[string]bool)
			for rows.Next() {
				var hash string
				rows.Scan(&hash)
				forecasts[hash] = true
			}

			b.logger.Printf("smhiGetForecasts: fetching forecasts for %s", name)
			if fc, err = smhi.GetPointForecast(coord.Longitude, coord.Latitude); err != nil {
				b.logger.Printf("smhiGetForecasts: %v", err)
				continue
			}
			b.logger.Printf("smhiGetForecasts: got forecasts for %s", name)

			// Iterate over the time series, which includes the actual
			// forecast data.
			for _, ts := range fc.TimeSeries {
				// Construct timestamp, id and hash.
				smhiTimestamp := ts.Timestamp.In(b.timezone).Format("2006-01-02T15:04:05.999")
				id := fmt.Sprintf("%s-%s",
					smhiTimestamp,
					name,
				)
				hash := fmt.Sprintf("%s|%s", id, ts.Hash)

				// The entry does alreayd exist in our
				// database, so we don't need to do anything.
				if inHash, _ := forecasts[hash]; inHash {
					continue
				}

				insertQuery := SMHI_FORECAST_INSERT_SQL_SQLITE
				if b.DB.Engine == "postgres" {
					insertQuery = SMHI_FORECAST_INSERT_SQL_POSTGRES
				}

				stmt, err := b.prepare(insertQuery)
				if err != nil {
					b.logger.Printf("smhiGetForecasts: %v", err)
					b.privmsg(b.DB.Err)
					continue
				}
				defer stmt.Close()

				b.logger.Printf("smhiGetForecasts: inserting forecasts for %s, %s", name, smhiTimestamp)
				_, err = stmt.Exec(
					id,
					hash,
					newTimestamp(),
					smhiTimestamp,
					name,
					ts.AirPressure,
					ts.AirTemperature,
					ts.HorizontalVisibility,
					ts.MaximumPrecipitationIntensity,
					ts.MeanPrecipitationIntensity,
					ts.MeanValueOfHighLevelCloudCover,
					ts.MeanValueOfLowLevelCloudCover,
					ts.MeanValueOfMediumLevelCloudCover,
					ts.MeanValueOfTotalCloudCover,
					ts.MedianPrecipitationIntensity,
					ts.MinimumPrecipitationIntensity,
					ts.PercentOfPrecipitationInFrozenForm,
					ts.PrecipitationCategory,
					ts.PrecipitationCategoryDescription[b.IRC.SMHILanguage],
					ts.RelativeHumidity,
					ts.ThunderProbability,
					ts.WeatherSymbol,
					ts.WeatherSymbolDescription[b.IRC.SMHILanguage],
					ts.WindDirection,
					ts.WindGustSpeed,
					ts.WindSpeed,
					ts.WindSpeedDescription[b.IRC.SMHILanguage],
				)
				if err != nil {
					b.logger.Printf("smhiGetForecasts: %v", err)
					b.privmsg(b.DB.Err)
					continue
				}
			}
		}

		// Let's sleep for an hour before we fetch new forecasts.
		time.Sleep(1 * time.Hour)
	}
}
