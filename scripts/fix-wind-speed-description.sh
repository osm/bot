#!/usr/bin/env bash

if [ -z "$1" ]; then
	echo "usage: $0 <db file>"
	exit 1
fi

echo "select
	id,
	wind_speed
from smhi_forecast
where wind_speed_description is null" | sqlite3 "$1" >tmp

get_wind_speed_description() {
	if (($(echo "$1 <= 0.2" | bc -l))); then
		echo -n "Stiltje"
	elif (($(echo "$1 >= 0.3 && $1 <= 1.5" | bc -l))); then
		echo -n "Nästan stiltje"
	elif (($(echo "$1 >= 1.6 && $1 <= 3.3" | bc -l))); then
		echo -n "Lätt bris"
	elif (($(echo "$1 >= 3.4 && $1 <= 5.4" | bc -l))); then
		echo -n "God bris"
	elif (($(echo "$1 >= 5.5 && $1 <= 7.9" | bc -l))); then
		echo -n "Frisk bris"
	elif (($(echo "$1 >= 8.0 && $1 <= 10.7" | bc -l))); then
		echo -n "Styv bris"
	elif (($(echo "$1 >= 10.8 && $1 <= 13.8" | bc -l))); then
		echo -n "Hård bris"
	elif (($(echo "$1 >= 13.9 && $1 <= 17.1" | bc -l))); then
		echo -n "Styv kuling"
	elif (($(echo "$1 >= 17.2 && $1 <= 20.7" | bc -l))); then
		echo -n "Hård kuling"
	elif (($(echo "$1 >= 20.8 && $1 <= 24.4" | bc -l))); then
		echo -n "Halv storm"
	elif (($(echo "$1 >= 24.5 && $1 <= 28.4" | bc -l))); then
		echo -n "Storm"
	elif (($(echo "$1 >= 28.5 && $1 <= 32.6" | bc -l))); then
		echo -n "Svår storm"
	else
		echo -n "Orkan"
	fi
}

for row in $(cat ./tmp); do
	id=$(echo $row | cut -d'|' -f1)
	wind_speed=$(echo $row | cut -d'|' -f2-)
	wind_speed_description=$(get_wind_speed_description $wind_speed)
	echo "update smhi_forecast set wind_speed_description = '$wind_speed_description' where id = '$id'" | sqlite3 $1
done
