# StorageOS kubectl plugin

[![Go Report Card](https://goreportcard.com/badge/github.com/storageos/kubectl-storageos)](https://goreportcard.com/report/github.com/storageos/kubectl-storageos)
[![e2e test](https://github.com/storageos/kubectl-storageos/actions/workflows/kuttl-e2e-test-1.23.yaml/badge.svg)](https://github.com/storageos/kubectl-storageos/actions/workflows/kuttl-e2e-test-1.23.yaml)
[![CodeQL](https://github.com/storageos/kubectl-storageos/actions/workflows/codeql-analysis.yml/badge.svg)](https://github.com/storageos/kubectl-storageos/actions/workflows/codeql-analysis.yml)
[![GitHub Super-Linter](https://github.com/storageos/kubectl-storageos/workflows/Lint%20Code%20Base/badge.svg)](https://github.com/marketplace/actions/super-linter)
[![Active](http://img.shields.io/badge/Status-Active-green.svg)](https://github.com/storageos/kubectl-storageos)
[![PR's Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg?style=flat)](https://github.com/storageos/kubectl-storageos/pulls)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](https://opensource.org/licenses/MIT)

Administrating tool for your StorageOS clusters.

## Installation

### Pre-Requisites

* Go 1.18+

### Download released version

```bash
curl -sSLo kubectl-storageos.tar.gz \
    https://github.com/storageos/kubectl-storageos/releases/download/v1.2.0/kubectl-storageos_1.2.0_linux_amd64.tar.gz \
    && tar -xf kubectl-storageos.tar.gz \
    && chmod +x kubectl-storageos \
    && sudo mv kubectl-storageos /usr/local/bin/ \
    && rm kubectl-storageos.tar.gz
```

### Build from source

```bash
git clone https://github.com/storageos/kubectl-storageos.git
cd kubectl-storageos
make build
cp ./bin/kubectl-storageos /usr/local/bin # directory on your $PATH
```

## Usage

### Install the latest version of StorageOS on your kubernetes cluster

```bash
kubectl storageos install
```

### Install an [ETCD Cluster](https://github.com/storageos/etcd-cluster-operator) and the latest version of StorageOS
**Warning**: This installation of ETCD is *not* production ready.

```bash
kubectl storageos install --include-etcd
```

### Uninstall StorageOS from your kubernetes cluster

```bash
kubectl storageos uninstall
```

### Uninstall both StorageOS and ETCD from your kubernetes cluster

> The following process **will not** remove data stored in disk by StorageOS.
> If Etcd is removed, StorageOS Volumes won't be recoverable, but if the Etcd
> cluster is kept intact, the volumes and their data will be available after a
> reinstall.

```bash
kubectl storageos uninstall --include-etcd
```

The ETCD uninstall process refers only to an ETCD cluster installed by the StorageOS ETCD Cluster Operator.

**Note**: The StorageOS ETCD Cluster Operator is a fork of the [Improbable Engineering ETCD Cluster Operator](https://github.com/improbable-eng/etcd-cluster-operator). As such, an instance of the latter operator running on the user's Kubernetes cluster can also be uninstalled by this command.

### Upgrade StorageOS to the latest version

```bash
kubectl storageos upgrade
```

The **upgrade** commands uninstalls your existing StorageOS cluster and installs the latest StorageOS cluster.

### Preflight checks

```bash
kubectl storageos preflight
```

A preflight check is a set of validations that can be run to ensure that a cluster meets the requirements to run StorageOS.

## Config file

Flags can also be passed to the **install**, **uninstall** and **upgrade** commands via the kubectl storageos config file like so:

```bash
kubectl storageos install --config-path=/path/to/config
```

This command expects to find a config file named "**kubectl-storageos-config.yaml**"

The **upgrade** command reads the `uninstall` and `install` settings in the config spec to perform the upgrade.
The following is an example of a config file that might be used for an upgrade with custom namespaces:

```yaml
apiVersion: storageos.com/v1
kind: KubectlStorageOSConfig
metadata:
  name: kubectlstorageosconfig-sample
spec:
  install:
    storageOSOperatorNamespace: storageos-operator-new
    storageOSClusterNamespace: storageos-cluster-new
  uninstall:
    storageOSOperatorNamespace: storageos-operator-old
```

For an example config file, see `config/samples/_v1_kubectlstorageosconfig.yaml`.

## Enable TLS

### Install ETCD and StorageOS with TLS enabled

```bash
kubectl storageos install --include-etcd --etcd-tls-enabled
```

### Install StorageOS and connect to an existing TLS enabled ETCD cluster

```bash
kubectl storageos install --etcd-tls-enabled
```

In order for StorageOS to connect to ETCD over HTTPS, a k8s secret must exist in the `storageos-cluster-namespace` with valid ETCD credentials.
This secret can be created like so:

```lang-none
kubectl create secret generic <etcd-secret-name> -n <storageos-cluster-namespace> \
    --from-file=etcd-client-ca.crt=path/to/ca.crt \
    --from-file=etcd-client.crt=path/to/tls.crt \
    --from-file=etcd-client.key=path/to/tls.key
```
**Note:** The default `etcd-secret-name` is `storageos-etcd-secret`. Should you name your secret differently, you must pass the name to the install command via `--etcd-secret-name`

## Recovery

Before **uninstall** and **upgrade** commands are executed, a number of manifests relative to the existing StorageOS cluster are written locally to disk in order for the user to manually recover the cluster should an error occur.

These manifests can be located at `$HOME/.kube/storageos`.
