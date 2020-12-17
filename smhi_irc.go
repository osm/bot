package main

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/osm/irc"
)

const SMHI_FORECAST_CURRENT_SELECT_SQL_SQLITE = `SELECT
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
	wind_speed,
	wind_speed_description
FROM
	smhi_forecast
WHERE
	name = $1
	AND substr(timestamp, 0, 11) = $2
ORDER BY timestamp`

const SMHI_FORECAST_CURRENT_SELECT_SQL_POSTGRES = `SELECT
	id,
	timestamp,
	inserted_at,
	updated_at,
	round(air_pressure, 1),
	round(air_temperature, 1),
	round(horizontal_visibility, 1),
	round(maximum_precipitation_intensity, 1),
	mean_precipitation_intensity,
	mean_value_of_high_level_cloud_cover,
	mean_value_of_low_level_cloud_cover,
	mean_value_of_medium_level_cloud_cover,
	mean_value_of_total_cloud_cover,
	round(median_precipitation_intensity, 1),
	round(minimum_precipitation_intensity, 1),
	percent_of_precipitation_in_frozen_form,
	precipitation_category,
	precipitation_category_description,
	relative_humidity,
	thunder_probability,
	weather_symbol,
	weather_symbol_description,
	wind_direction,
	round(wind_gust_speed, 1),
	round(wind_speed, 1),
	wind_speed_description
FROM
	smhi_forecast
WHERE
	name = $1
	AND substr(timestamp::text, 0, 11) = $2
ORDER BY timestamp`

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

// smhiForecastCmdRegexp extracts dates and time and splits them up into
// groups.
const smhiForecastCmdRegexpSpace = `( )?`
const smhiForecastCmdRegexpNick = `([0-9a-zA-ZåäöÅÄÖ_\-\*,]+)`
const smhiForecastCmdRegexpDate = `((tomorrow|imorgon)|(\d\d\d\d-(0?[1-9]|1[0-2])-(0?[1-9]|[12][0-9]|3[01])))?`
const smhiForecastCmdRegexpTime = `((0[0-9]|1[0-9]|2[0-3])((:|.)([0-9]|[0-5][0-9]))?)?`

var smhiForecastCmdRegexp = regexp.MustCompile(fmt.Sprintf(`^%s%s%s%s%s$`,
	smhiForecastCmdRegexpNick,
	smhiForecastCmdRegexpSpace,
	smhiForecastCmdRegexpDate,
	smhiForecastCmdRegexpSpace,
	smhiForecastCmdRegexpTime,
))

// smhiCommandHandler handles the commands issued from the IRC channel.
func (b *bot) smhiCommandHandler(m *irc.Message) {
	a := b.parseAction(m).(*privmsgAction)

	// Not our channel, return.
	if !a.validChannel {
		return
	}

	// Not a SMHI command, return.
	if a.cmd != b.IRC.SMHICmdWeather {
		return
	}

	// Use should be ignored, return.
	if b.shouldIgnore(m) {
		return
	}

	// Not enough args, return.
	if len(a.args) < 1 {
		return
	}

	// Not enough args, return.
	parts := smhiForecastCmdRegexp.FindStringSubmatch(strings.Join(a.args, " "))
	if len(parts) == 0 {
		b.privmsg(b.IRC.SMHIMsgWeatherError)
		return
	}

	// Split nicks on ,.
	nicks := strings.Split(parts[1], ",")

	// The * is a special character which means that were printing info
	// on all locations we're fetching data for.
	printAll := false
	if len(nicks) == 1 && nicks[0] == "*" {
		printAll = true
		for n := range b.IRC.SMHIForecastLocations {
			nicks = append(nicks, n)
		}
	}

	// If we've got a match we use the submitted value, otherwise fallback
	// to todays date.
	var d string
	if len(parts[4]) > 0 {
		d = newDateWithDuration(time.Hour * 24)
	} else if len(parts[5]) > 0 {
		d = parts[5]
	} else {
		d = newDate()
	}

	// If we've got a match we use the submitted value, otherwise fallback
	// to the current time.
	var h int
	if len(parts[10]) > 0 {
		h = stringToInt(parts[10])
	} else {
		h = stringToInt(newHour())
	}

	// Iterate over the nicks.
	for _, n := range nicks {
		// The nick wasn't found, continue.
		data, hasName := b.IRC.SMHIForecastLocations[n]
		if !hasName {
			continue
		}

		// Don't print aliases when we're fetching all records.
		if printAll && len(data.Alias) > 0 {
			continue
		}

		// If the forecast has an alias set, we'll use that instead of the
		// provided name.
		var name string
		if len(data.Alias) > 0 {
			name = data.Alias
		} else {
			name = n
		}

		selectQuery := SMHI_FORECAST_CURRENT_SELECT_SQL_SQLITE
		if b.DB.Engine == "postgres" {
			selectQuery = SMHI_FORECAST_CURRENT_SELECT_SQL_POSTGRES
		}

		// Execute the query and return the results.
		rows, err := b.query(selectQuery, name, d)
		if err != nil {
			b.privmsg(b.IRC.SMHIMsgWeatherError)
			return
		}
		defer rows.Close()

		// Append forecasts for each returned row.
		var forecasts []smhiForecast
		for rows.Next() {
			var fc smhiForecast
			err = rows.Scan(
				&fc.Id,
				&fc.Timestamp,
				&fc.InsertedAt,
				&fc.UpdatedAt,
				&fc.AirPressure,
				&fc.AirTemperature,
				&fc.HorizontalVisibility,
				&fc.MaximumPrecipitationIntensity,
				&fc.MeanPrecipitationIntensity,
				&fc.MeanValueOfHighLevelCloudCover,
				&fc.MeanValueOfLowLevelCloudCover,
				&fc.MeanValueOfMediumLevelCloudCover,
				&fc.MeanValueOfTotalCloudCover,
				&fc.MedianPrecipitationIntensity,
				&fc.MinimumPrecipitationIntensity,
				&fc.PercentOfPrecipitationInFrozenForm,
				&fc.PrecipitationCategory,
				&fc.PrecipitationCategoryDescription,
				&fc.RelativeHumidity,
				&fc.ThunderProbability,
				&fc.WeatherSymbol,
				&fc.WeatherSymbolDescription,
				&fc.WindDirection,
				&fc.WindGustSpeed,
				&fc.WindSpeed,
				&fc.WindSpeedDescription,
			)
			forecasts = append(forecasts, fc)
		}

		// No forecasts found, return early.
		if len(forecasts) == 0 {
			b.privmsg(b.IRC.SMHIMsgWeatherError)
			return
		}

		// Set the index to the hour we are searching for, this assumes that
		// the forecast has 24 entries, one for each hour.
		idx := h

		// If we have less than 24 forecasts for the given date we'll try to
		// find the hour that is closest to what we search for.
		if len(forecasts) < 24 {
			var l []int
			for _, f := range forecasts {
				l = append(l, stringToInt(f.Timestamp[11:13]))
			}

			idx = sort.SearchInts(l, h)
			if idx == -1 {
				idx += 1
			} else if idx == len(l) {
				idx -= 1
			}
		}

		// Use the forecast specified by the idx.
		var fc *smhiForecast = &forecasts[idx]

		// Send the message.
		b.privmsgph(b.IRC.SMHIMsgWeather, map[string]string{
			"<id>":                                   fc.Id,
			"<timestamp>":                            fc.Timestamp,
			"<date>":                                 fc.Timestamp[0:10],
			"<time>":                                 fc.Timestamp[11:16],
			"<inserted_at>":                          fc.InsertedAt,
			"<updated_at>":                           fc.UpdatedAt,
			"<nick>":                                 n,
			"<name>":                                 name,
			"<air_pressure>":                         fc.AirPressure,
			"<air_temperature>":                      fmtNumber(fc.AirTemperature, b.IRC.SMHILanguage),
			"<horizontal_visibility>":                fc.HorizontalVisibility,
			"<maximum_precipitation_intensity>":      fc.MaximumPrecipitationIntensity,
			"<mean_precipitation_intensity>":         fc.MeanPrecipitationIntensity,
			"<mean_value_of_high_level_cloud_cover>": fc.MeanValueOfHighLevelCloudCover,
			"<mean_value_of_low_level_cloud_cover>":  fc.MeanValueOfLowLevelCloudCover,
			"<mean_value_of_medium_level_cloud_cover>":  fc.MeanValueOfMediumLevelCloudCover,
			"<mean_value_of_total_cloud_cover>":         fc.MeanValueOfTotalCloudCover,
			"<median_precipitation_intensity>":          fc.MedianPrecipitationIntensity,
			"<minimum_precipitation_intensity>":         fc.MinimumPrecipitationIntensity,
			"<percent_of_precipitation_in_frozen_form>": fc.PercentOfPrecipitationInFrozenForm,
			"<precipitation_category>":                  fc.PrecipitationCategory,
			"<precipitation_category_description>":      fc.PrecipitationCategoryDescription,
			"<relative_humidity>":                       fc.RelativeHumidity,
			"<thunder_probability>":                     fc.ThunderProbability,
			"<weather_symbol>":                          fc.WeatherSymbol,
			"<weather_symbol_description>":              fc.WeatherSymbolDescription,
			"<wind_direction>":                          fc.WindDirection,
			"<wind_gust_speed>":                         fc.WindGustSpeed,
			"<wind_speed>":                              fmtNumber(fc.WindSpeed, b.IRC.SMHILanguage),
			"<wind_speed_description>":                  fc.WindSpeedDescription,
		})
	}
}

// smhiForecast contains all the values that are read from the database.
type smhiForecast struct {
	Id                                 string
	Timestamp                          string
	InsertedAt                         string
	UpdatedAt                          string
	AirPressure                        string
	AirTemperature                     string
	HorizontalVisibility               string
	MaximumPrecipitationIntensity      string
	MeanPrecipitationIntensity         string
	MeanValueOfHighLevelCloudCover     string
	MeanValueOfLowLevelCloudCover      string
	MeanValueOfMediumLevelCloudCover   string
	MeanValueOfTotalCloudCover         string
	MedianPrecipitationIntensity       string
	MinimumPrecipitationIntensity      string
	PercentOfPrecipitationInFrozenForm string
	PrecipitationCategory              string
	PrecipitationCategoryDescription   string
	RelativeHumidity                   string
	ThunderProbability                 string
	WeatherSymbol                      string
	WeatherSymbolDescription           string
	WindDirection                      string
	WindGustSpeed                      string
	WindSpeed                          string
	WindSpeedDescription               string
}
