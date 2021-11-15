#!/bin/bash

# shellcheck disable=SC2086
# shellcheck disable=SC2223

ID=$(docker ps -aqf "name=kind-control-plane")

docker exec -i $ID rm -f /var/lib/storageos/config.json 
