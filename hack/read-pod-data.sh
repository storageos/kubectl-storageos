#!/bin/bash -e

# shellcheck disable=SC2086
# shellcheck disable=SC2223

: "${1?= required}" # pod name
: "${2?= required}" # path to read from
: "${3?= required}" # expected data read


echo "Checking $1 for data..."

existing_data=$(/usr/local/bin/kubectl exec -i "$1" -- bash -c "cat $2")
existing_data="${existing_data%"${existing_data##*[![:space:]]}"}"  

if [ "$existing_data" = "$3" ]; then
	echo "Success, data discovered: $existing_data"
else
	echo "Error: data not found"
	exit 1
fi
