#!/bin/bash

# shellcheck disable=SC1072
# shellcheck disable=SC1073

: ${OPERATOR_DIR:=operator}

IMAGE="nixery.dev/shell/jq/git/curl/kustomize/gnutar/gzip"

if [[ -z "$VERSION" ]]; then
    VERSION=$(docker run --rm $IMAGE bash -c "
        curl -s https://api.github.com/repos/storageos/operator/releases |
        jq '.[]|select(.draft==false)' |
        jq -r .tag_name |
        head -1
    ")
fi

docker run --rm -v $OPERATOR_DIR:/output $IMAGE bash -c "
    git clone https://github.com/storageos/operator.git &&
    cd operator &&
    git reset --hard $VERSION &>/dev/null &&
    (cd config/manager ; kustomize edit set image controller=storageos/operator:$VERSION) &&
    (cd config ; tar -czvf /output/storageos-operator-manifests.tar.gz .)
"