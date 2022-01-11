#!/bin/bash -e

# shellcheck disable=SC2086
# shellcheck disable=SC2223

docker run storageos/operator-manifests:develop > storageos-operator.yaml
