package smhi

import (
	"time"
)

// PrecipitationCategory constants.
const (
	NoPrecipitation PrecipitationCategory = iota
	Snow
	SnowAndRain
	Rain
	Drizzle
	FreezingRain
	FreezingDrizzle
)

// WeatherSymbol constants.
const (
	ClearSky WeatherSymbol = iota + 1
	NearlyClearSky
	VariableCloudiness
	HalfclearSky
	CloudySky
	Overcast
	Fog
	LightRainShowers
	ModerateRainShowers
	HeavyRainShowers
	Thunderstorm
	LightSleetShowers
	ModerateSleetShowers
	HeavySleetShowers
	LightSnowShowers
	ModerateSnowShowers
	HeavySnowShowers
	LightRain
	ModerateRain
	HeavyRain
	Thunder
	LightSleet
	ModerateSleet
	HeavySleet
	LightSnowfall
	ModerateSnowfall
	HeavySnowfall
)

type PrecipitationCategory uint8

type Coordinate []float64

type WeatherSymbol uint8

type Geometry struct {
	Type        string
	Coordinates []Coordinate
}

// PointForecastAPI defines the data structure that is returned by the SMHI
// point forecast API
type PointForecastAPI struct {
	ApprovedTime  string
	ReferenceTime string
	Geometry      Geometry
	TimeSeries    []struct {
		ValidTime  string
		Parameters []struct {
			Name      string
			LevelType string
			Level     uint8
			Unit      string
			Values    []float64
		}
	}
}

// Forecast defines the structure that holds the converted TimeSeries data
// from the data returned by the SMHI point forecast API.
type Forecast struct {
	Hash                               string
	Timestamp                          time.Time
	AirPressure                        float64
	AirTemperature                     float64
	HorizontalVisibility               float64
	MaximumPrecipitationIntensity      float64
	MeanPrecipitationIntensity         float64
	MeanValueOfHighLevelCloudCover     uint8
	MeanValueOfLowLevelCloudCover      uint8
	MeanValueOfMediumLevelCloudCover   uint8
	MeanValueOfTotalCloudCover         uint8
	MedianPrecipitationIntensity       float64
	MinimumPrecipitationIntensity      float64
	PercentOfPrecipitationInFrozenForm int8
	PrecipitationCategory              PrecipitationCategory
	PrecipitationCategoryDescription   map[string]string
	RelativeHumidity                   uint8
	ThunderProbability                 uint8
	WeatherSymbol                      WeatherSymbol
	WeatherSymbolDescription           map[string]string
	WindDirection                      uint8
	WindGustSpeed                      float64
	WindSpeed                          float64
	WindSpeedDescription               map[string]string
}

// PointForecast holds the data for a complete PointForecast request.
type PointForecast struct {
	ApprovedTime  time.Time
	ReferenceTime time.Time
	Geometry      Geometry
	TimeSeries    []Forecast
}
