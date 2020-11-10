#!/bin/sh

if [ -z "$1" ]; then
	echo "usage: $0 <db file>"
	exit 1
fi

echo "select
	id,
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
	relative_humidity,
	thunder_probability,
	weather_symbol,
	wind_direction,
	wind_gust_speed,
	wind_speed
from smhi_forecast
where hash is null" | sqlite3 "$1" >tmp

for row in $(cat ./tmp); do
	id=$(echo $row | cut -d'|' -f1)
	hash=$(echo $row | cut -d'|' -f2-)
	echo "update smhi_forecast set hash = '$id|$hash' where id = '$id'" | sqlite3 "$1"
done
