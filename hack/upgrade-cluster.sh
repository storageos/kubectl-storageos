#!/bin/bash -e

# shellcheck disable=SC2086
# shellcheck disable=SC2223

: ${VERSION:=develop}
: ${UPGRADE:=}
: ${EXTRA_FLAGS:=--skip-stos-cluster --wait --stack-trace}

cleanup() {
    echo "Cleaning up..."
    rm -rf storageos-operator.yaml
}

for img in storageos/operator:$VERSION storageos/portal-manager:$VERSION storageos/api-manager:$VERSION soegarots/node:$VERSION; do
    docker pull -q $img
done

operatorImage=$(docker inspect --format='{{index .RepoDigests 0}}' storageos/operator:$VERSION)
portalImage=$(docker inspect --format='{{index .RepoDigests 0}}' storageos/portal-manager:$VERSION)
apiManagerImage=$(docker inspect --format='{{index .RepoDigests 0}}' storageos/api-manager:$VERSION)
nodeImage=$(docker inspect --format='{{index .RepoDigests 0}}' soegarots/node:$VERSION)

docker run --rm storageos/operator-manifests:$VERSION | \
    sed "s|storageos/operator:${VERSION}|${operatorImage}|" | \
    sed "s|RELATED_IMAGE_API_MANAGER.*|RELATED_IMAGE_API_MANAGER: ${apiManagerImage}|" | \
    sed "s|RELATED_IMAGE_PORTAL_MANAGER.*|RELATED_IMAGE_PORTAL_MANAGER: ${portalImage}|" | \
    sed "s|RELATED_IMAGE_STORAGEOS_NODE.*|RELATED_IMAGE_STORAGEOS_NODE: ${nodeImage}|" > \
    storageos-operator.yaml

grep -e "\simage:\s" storageos-operator.yaml
grep "RELATED_IMAGE_" storageos-operator.yaml

trap cleanup EXIT

if [[ -z "$UPGRADE" ]]; then
    ./bin/kubectl-storageos install --include-etcd --stos-operator-yaml storageos-operator.yaml $EXTRA_FLAGS
else
    ./bin/kubectl-storageos upgrade --skip-namespace-deletion yes --stos-operator-yaml storageos-operator.yaml $EXTRA_FLAGS
fi
    