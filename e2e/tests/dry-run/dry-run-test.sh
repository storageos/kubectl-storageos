#!/bin/bash -e

# shellcheck disable=SC2086
# shellcheck disable=SC2223

: "${LATEST_VERSION?= required}"

TMPDIR="$(mktemp -d)"
trap 'rm -rf -- "$TMPDIR"' EXIT

# Run install --dry-run command and move output to tmpdir
kubectl storageos install --dry-run --k8s-version=v1.22.0
cp -r storageos-dry-run $TMPDIR/ && rm -r storageos-dry-run
# Fetch operator-manifests and store in temp dir
docker run "storageos/operator-manifests:v${LATEST_VERSION}" > ${TMPDIR}/storageos-operator.yaml
# Compare storageos manifests generated by dry-run with test data
diff ${TMPDIR}/storageos-operator.yaml ${TMPDIR}/storageos-dry-run/storageos-operator.yaml
diff test-data/storageos-cluster.yaml ${TMPDIR}/storageos-dry-run/storageos-cluster.yaml

#  Run install --dry-run command and move output to tmpdir
kubectl storageos install --include-etcd --dry-run --etcd-storage-class=standard --k8s-version=v1.22.0
cp -r storageos-dry-run $TMPDIR/ && rm -r storageos-dry-run
# Fetch etcd-operator manifest 
wget -q https://github.com/storageos/etcd-cluster-operator/releases/download/v0.3.1/storageos-etcd-cluster-operator.yaml -P ${TMPDIR}/
# Compare etcd manifests generated by dry-run with test data
diff ${TMPDIR}/storageos-etcd-cluster-operator.yaml ${TMPDIR}/storageos-dry-run/etcd-operator.yaml
diff test-data/etcd-cluster.yaml ${TMPDIR}/storageos-dry-run/etcd-cluster.yaml

#  Run install --dry-run command and move output to tmpdir
kubectl storageos install --k8s-version=v1.22.0-gke.0 --dry-run
cp -r storageos-dry-run $TMPDIR/ && rm -r storageos-dry-run
# Compare resource-quota manifest generated by dry-run with test data
diff test-data/resource-quota.yaml ${TMPDIR}/storageos-dry-run/resource-quota.yaml

#TODO: For now, portal enablement is only available from 'develop' version of operator. As the dry-run capability
# fetches operator-manifests from release assets only, the portal configs are not testable until stos v2.6.0

#kubectl storageos install --enable-portal-manager --portal-api-url=www.test.com --portal-tenant-id=storageos --portal-client-id=storageos --portal-secret=storageos --dry-run
#cp -r storageos-dry-run $TMPDIR/ && rm -r storageos-dry-run
# Fetch portal-configmap
#wget -q https://github.com/storageos/kubectl-storageos/releases/download/v1.0.0/configmap-storageos-portal-manager.yaml -P ${TMPDIR}/
# Compare portal manifests generated by dry-run with test data
#diff ${TMPDIR}/configmap-storageos-portal-manager.yaml ${TMPDIR}/storageos-dry-run/storageos-portal-configmap.yaml
#diff test-data/storageos-portal-client.yaml ${TMPDIR}/storageos-dry-run/storageos-portal-client.yaml

echo "Dry run test completed succesfully!"
