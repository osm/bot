package main

import (
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/osm/irc"
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
	wind_speed,
	wind_speed_description
FROM
	smhi_forecast
WHERE
	name = ?
	AND substr(timestamp, 0, 11) = ?
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
var smhiForecastCmdRegexp = regexp.MustCompile(`^((tomorrow|imorgon)|(\d\d\d\d-(0?[1-9]|1[0-2])-(0?[1-9]|[12][0-9]|3[01])))?( )?((0[0-9]|1[0-9]|2[0-3])((:|.)([0-9]|[0-5][0-9]))?)?$`)

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

	// Possible commands.
	// !smhi osm
	// !smhi osm 10
	// !smhi osm 10:00
	// !smhi osm 2020-09-01 10
	// !smhi osm 2020-09-01 10:00

	// The first argument is always the name the forecast is associated
	// with.
	n := a.args[0]
	data, hasName := b.IRC.SMHIForecastLocations[n]
	if !hasName {
		b.privmsg(b.IRC.SMHIMsgWeatherError)
		return
	}

	// If the forecast has an alias set, we'll use that instead of the
	// provided name.
	var name string
	if len(data.Alias) > 0 {
		name = data.Alias
	} else {
		name = n
	}

	// Execute the regexp and extract the matches into groups.
	// If we can't match a date we'll use todays date as fallback value.
	// And if we can't extract a time from the arguments we'll fallback to
	// the current hour.
	parts := smhiForecastCmdRegexp.FindStringSubmatch(strings.Join(a.args[1:], " "))

	// We were unable to match anything with our regexp, so return early.
	if len(a.args) > 1 && len(parts) == 0 {
		b.privmsg(b.IRC.SMHIMsgWeatherError)
		return
	}

	// If we've got a match we use the submitted value, otherwise fallback
	// to todays date.
	var d string
	if len(parts[2]) > 0 {
		d = newDateWithDuration(time.Hour * 24)
	} else if len(parts[3]) > 0 {
		d = parts[3]
	} else {
		d = newDate()
	}

	// If we've got a match we use the submitted value, otherwise fallback
	// to the current time.
	var h int
	if len(parts[7]) > 0 {
		h = stringToInt(parts[7])
	} else {
		h = stringToInt(newHour())
	}

	// Execute the query and return the results.
	rows, err := b.query(SMHI_FORECAST_CURRENT_SELECT_SQL, name, d)
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
		"<name>":                                 name,
		"<air_pressure>":                         fc.AirPressure,
		"<air_temperature>":                      fc.AirTemperature,
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
		"<wind_speed>":                              fc.WindSpeed,
		"<wind_speed_description>":                  fc.WindSpeedDescription,
	})
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
