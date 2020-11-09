package main

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/osm/irc"
	"github.com/osm/smhi"
)

const SMHI_FORECAST_CURRENT_SELECT_SQL = `SELECT
	id,
	timestamp,
	inserted_at,
	updated_at,
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
	wind_speed
FROM
	smhi_forecast
WHERE
	name = $1 AND
	timestamp >= ?
ORDER BY timestamp
LIMIT 1`

const SMHI_FORECAST_INSERT_SQL = `INSERT OR REPLACE INTO smhi_forecast (
	id,
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
	wind_speed
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);`

func init() {
	rand.Seed(time.Now().UnixNano())
}

// initSMHIDefaults sets default values for all settings.
func (b *bot) initSMHIDefaults() {
	if b.IRC.SMHILanguage == "" {
		b.IRC.SMHILanguage = "en-US"
	}
	if b.IRC.SMHICmdWeather == "" {
		b.IRC.SMHICmdWeather = "!smhi"
	}
	if b.IRC.SMHIMsgWeatherError == "" {
		b.IRC.SMHIMsgWeatherError = "unable to find forecast"
	}
	if b.IRC.SMHIMsgWeather == "" {
		b.IRC.SMHIMsgWeather = "<weather_symbol_description>, <air_temperature> C"
	}
}

// smhiGetForecasts runs once every hour and fetches new forecasts for all the
// locations that is defined in the config. The forecast is saved to the
// database. We'll always wipe
func (b *bot) smhiGetForecasts() {
	var fc *smhi.PointForecast
	var err error

	for {
		// Iterate over the locations and fetch a forecast for the given
		// coordinates.
		for name, coord := range b.IRC.SMHIForecastLocations {
			b.logger.Printf("smhiGetForecasts: fetching forecasts for %s", name)
			if fc, err = smhi.GetPointForecast(coord.Longitude, coord.Latitude); err != nil {
				b.logger.Printf("smhiGetForecasts: %v", err)
				continue
			}
			b.logger.Printf("smhiGetForecasts: got forecasts for %s", name)

			// Iterate over the time series, which includes the actual
			// forecast data.
			for _, ts := range fc.TimeSeries {
				stmt, err := b.prepare(SMHI_FORECAST_INSERT_SQL)
				if err != nil {
					b.logger.Printf("smhiGetForecasts: %v", err)
					b.privmsg(b.DB.Err)
					continue
				}
				defer stmt.Close()

				smhiTimestamp := ts.Timestamp.In(b.timezone).Format("2006-01-02T15:04:05.999")
				b.logger.Printf("smhiGetForecasts: inserting forecasts for %s, %s", name, smhiTimestamp)
				_, err = stmt.Exec(
					fmt.Sprintf("%s-%s",
						smhiTimestamp,
						name,
					),
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
				)
				if err != nil {
					b.logger.Printf("smhiGetForecasts: %v", err)
					b.privmsg(b.DB.Err)
					continue
				}

				// Since the bot might be running on a slow
				// machine and we are using a SQLite which
				// doesn't handle a lot of inserts very well
				// (at least not on my RPI4), we'll sleep a
				// while for each insert so that we can catch
				// up.
				time.Sleep(time.Duration(rand.Intn(5)) * time.Second)
			}
		}

		// Let's sleep for an hour before we fetch new forecasts.
		time.Sleep(1 * time.Hour)
	}
}

// smhiCommandHandler handles the commands issued from the IRC channel.
func (b *bot) smhiCommandHandler(m *irc.Message) {
	a := b.parseAction(m).(*privmsgAction)

	if !a.validChannel {
		return
	}

	if a.cmd != b.IRC.SMHICmdWeather {
		return
	}

	if b.shouldIgnore(m) {
		return
	}

	if len(a.args) != 1 {
		return
	}
	name := a.args[0]

	var id string
	var timestamp string
	var insertedAt string
	var updatedAt string
	var airPressure string
	var airTemperature string
	var horizontalVisibility string
	var maximumPrecipitationIntensity string
	var meanPrecipitationIntensity string
	var meanValueOfHighLevelCloudCover string
	var meanValueOfLowLevelCloudCover string
	var meanValueOfMediumLevelCloudCover string
	var meanValueOfTotalCloudCover string
	var medianPrecipitationIntensity string
	var minimumPrecipitationIntensity string
	var percentOfPrecipitationInFrozenForm string
	var precipitationCategory string
	var precipitationCategoryDescription string
	var relativeHumidity string
	var thunderProbability string
	var weatherSymbol string
	var weatherSymbolDescription string
	var windDirection string
	var windGustSpeed string
	var windSpeed string
	err := b.queryRow(SMHI_FORECAST_CURRENT_SELECT_SQL, name, newTimestamp()).Scan(
		&id,
		&timestamp,
		&insertedAt,
		&updatedAt,
		&airPressure,
		&airTemperature,
		&horizontalVisibility,
		&maximumPrecipitationIntensity,
		&meanPrecipitationIntensity,
		&meanValueOfHighLevelCloudCover,
		&meanValueOfLowLevelCloudCover,
		&meanValueOfMediumLevelCloudCover,
		&meanValueOfTotalCloudCover,
		&medianPrecipitationIntensity,
		&minimumPrecipitationIntensity,
		&percentOfPrecipitationInFrozenForm,
		&precipitationCategory,
		&precipitationCategoryDescription,
		&relativeHumidity,
		&thunderProbability,
		&weatherSymbol,
		&weatherSymbolDescription,
		&windDirection,
		&windGustSpeed,
		&windSpeed,
	)
	if err != nil {
		b.privmsg(b.IRC.SMHIMsgWeatherError)
		return
	}

	b.privmsgph(b.IRC.SMHIMsgWeather, map[string]string{
		"<id>":                                      id,
		"<timestamp>":                               timestamp,
		"<inserted_at>":                             insertedAt,
		"<updated_at>":                              updatedAt,
		"<name>":                                    name,
		"<air_pressure>":                            airPressure,
		"<air_temperature>":                         airTemperature,
		"<horizontal_visibility>":                   horizontalVisibility,
		"<maximum_precipitation_intensity>":         maximumPrecipitationIntensity,
		"<mean_precipitation_intensity>":            meanPrecipitationIntensity,
		"<mean_value_of_high_level_cloud_cover>":    meanValueOfHighLevelCloudCover,
		"<mean_value_of_low_level_cloud_cover>":     meanValueOfLowLevelCloudCover,
		"<mean_value_of_medium_level_cloud_cover>":  meanValueOfMediumLevelCloudCover,
		"<mean_value_of_total_cloud_cover>":         meanValueOfTotalCloudCover,
		"<median_precipitation_intensity>":          medianPrecipitationIntensity,
		"<minimum_precipitation_intensity>":         minimumPrecipitationIntensity,
		"<percent_of_precipitation_in_frozen_form>": percentOfPrecipitationInFrozenForm,
		"<precipitation_category>":                  precipitationCategory,
		"<precipitation_category_description>":      precipitationCategoryDescription,
		"<relative_humidity>":                       relativeHumidity,
		"<thunder_probability>":                     thunderProbability,
		"<weather_symbol>":                          weatherSymbol,
		"<weather_symbol_description>":              weatherSymbolDescription,
		"<wind_direction>":                          windDirection,
		"<wind_gust_speed>":                         windGustSpeed,
		"<wind_speed>":                              windSpeed,
	})
}
