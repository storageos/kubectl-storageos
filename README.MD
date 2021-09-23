# StorageOS kubectl plugin

[![Go Report Card](https://goreportcard.com/badge/github.com/storageos/kubectl-storageos)](https://goreportcard.com/report/github.com/storageos/kubectl-storageos)
[![e2e test](https://github.com/storageos/kubectl-storageos/actions/workflows/kuttl-e2e-test.yaml/badge.svg)](https://github.com/storageos/kubectl-storageos/actions/workflows/kuttl-e2e-test.yaml)
[![CodeQL](https://github.com/storageos/kubectl-storageos/actions/workflows/codeql-analysis.yml/badge.svg)](https://github.com/storageos/kubectl-storageos/actions/workflows/codeql-analysis.yml)
[![Active](http://img.shields.io/badge/Status-Active-green.svg)](https://github.com/storageos/kubectl-storageos)
[![PR's Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg?style=flat)](https://github.com/storageos/kubectl-storageos/pulls)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](https://opensource.org/licenses/MIT)

This project is under development. Use it at your own risk, please.

The main goal of this project is to seemlessly enable StorageOS related administration tasks in a Kubernetes cluster.

## Installation

### Pre-Requisites

 * Go 1.16+

Since the project is under active development there isn't a release process or pre-compiled binaries yet.  
For now, to install the plugin you have to build your own version. 

```
# git clone https://github.com/storageos/kubectl-storageos.git
# cd kubectl-storageos
# make build
# cp ./bin/kubectl-storageos /usr/local/bin # Or any directory of your $PATH
```

## Usage

```
# kubectl storageos -h
StorageOS kubectl plugin

Usage:
  storageos [flags]
  storageos [command]

Available Commands:
  bundle      Generate a support bundle
  help        Help about any command
  install     Install StorageOS and (optionally) ETCD
  preflight   Test a k8s cluster for StorageOS pre-requisites
  uninstall   Uninstall StorageOS and (optionally) ETCD
  upgrade     Ugrade StorageOS

Flags:
      --as string                      Username to impersonate for the operation
      --as-group stringArray           Group to impersonate for the operation, this flag can be repeated to specify multiple groups.
      --cache-dir string               Default cache directory (default "/root/.kube/cache")
      --certificate-authority string   Path to a cert file for the certificate authority
      --client-certificate string      Path to a client certificate file for TLS
      --client-key string              Path to a client key file for TLS
      --cluster string                 The name of the kubeconfig cluster to use
      --context string                 The name of the kubeconfig context to use
  -h, --help                           help for storageos
      --insecure-skip-tls-verify       If true, the server's certificate will not be checked for validity. This will make your HTTPS connections insecure
      --kubeconfig string              Path to the kubeconfig file to use for CLI requests.
  -n, --namespace string               If present, the namespace scope for this CLI request
      --request-timeout string         The length of time to wait before giving up on a single server request. Non-zero values should contain a corresponding time unit (e.g. 1s, 2m, 3h). A value of zero means don't timeout requests. (default "0")
  -s, --server string                  The address and port of the Kubernetes API server
      --tls-server-name string         Server name to use for server certificate validation. If it is not provided, the hostname used to contact the server is used
      --token string                   Bearer token for authentication to the API server
      --user string                    The name of the kubeconfig user to use

Use "storageos [command] --help" for more information about a command.
```
## Commands
### Install

```
# kubectl storageos install -h
Install StorageOS and (optionally) ETCD

Usage:
  storageos install [flags]

Flags:
      --config-path string               path to look for kubectl-storageos-config.yaml
      --etcd-cluster-yaml string         etcd-cluster.yaml path or url
      --etcd-endpoints string            etcd endpoints
      --etcd-namespace string            namespace of etcd operator and cluster to be installed (default "storageos-etcd")
      --etcd-operator-yaml string        etcd-operator.yaml path or url
      --etcd-secret-name string          name of etcd secret in storageos cluster namespace (default "storageos-etcd-secret")
      --etcd-tls-enabled                 etcd cluster is TLS enabled
  -h, --help                             help for install
      --include-etcd                     install non-production etcd from github.com/storageos/etcd-cluster-operator
      --skip-etcd-endpoints-validation   skip validation of ETCD endpoints
      --storage-class string             name of storage class to be used by etcd cluster
      --stos-cluster-namespace string    namespace of storageos cluster to be installed (default "storageos")
      --stos-cluster-yaml string         storageos-cluster.yaml path or url
      --stos-operator-namespace string   namespace of storageos operator to be installed (default "storageos")
      --stos-operator-yaml string        storageos-operator.yaml path or url
      --version string                   version of storageos operator
      --wait                             wait for storagos cluster to enter running phase
```

#### Install the Latest Version of StorageOS on Your Kubernetes Cluster
`kubectl storageos install`


#### Install an [ETCD Cluster](https://github.com/storageos/etcd-cluster-operator) and the Latest Version of StorageOS
**Warning**: This installation of ETCD is *not* production ready.

`kubectl storageos install --include-etcd`



### Uninstall

```
# kubectl storageos uninstall -h
Uninstall StorageOS and (optionally) ETCD

Usage:
  storageos uninstall [flags]

Flags:
      --config-path string               path to look for kubectl-storageos-config.yaml
      --etcd-namespace string            namespace of etcd operator and cluster to be uninstalled (default "storageos-etcd")
  -h, --help                             help for uninstall
      --include-etcd                     uninstall etcd (only applicable to github.com/storageos/etcd-cluster-operator etcd cluster)
      --skip-namespace-deletion          leaving namespaces untouched
      --stos-cluster-namespace string    namespace of storageos cluster to be uninstalled (default "storageos")
      --stos-operator-namespace string   namespace of storageos operator to be uninstalled (default "storageos")
```
   
#### Uninstall StorageOS From Your Kubernetes Cluster

`kubectl storageos uninstall`

#### Uninstall StorageOS and ETCD From Your Kubernetes Cluster

> The following process **will not** remove data stored in disk by StorageOS.
> If Etcd is removed, StorageOS Volumes won't be recoverable, but if the Etcd
> cluster is kept intact, the volumes and their data will be available after a
> reinstall.

`kubectl storageos uninstall --include-etcd`

The ETCD uninstallation refers only to an ETCD cluster installed by the StorageOS ETCD Cluster Operator.

**Note**: The StorageOS ETCD Cluster Operator is a fork of the [Improbable Engineering ETCD Cluster Operator](https://github.com/improbable-eng/etcd-cluster-operator). As such, an instance of the latter operator running on the user's Kubernetes cluster can also be uninstalled by this command.


### Upgrade

```
kubectl storageos upgrade -h
Upgrade StorageOS operator and cluster version

Usage:
  storageos upgrade [flags]

Flags:
      --config-path string                         path to look for kubectl-storageos-config.yaml
      --etcd-endpoints string                      etcd endpoints
      --etcd-secret-name string                    name of etcd secret in storageos cluster namespace (default "storageos-etcd-secret")
      --etcd-tls-enabled                           etcd cluster is TLS enabled
  -h, --help                                       help for upgrade
      --install-stos-cluster-namespace string      namespace of storageos cluster to be installed (default "storageos")
      --install-stos-operator-namespace string     namespace of storageos operator to be installed (default "storageos")
      --skip-etcd-endpoints-validation             skip validation of ETCD endpoints
      --skip-namespace-deletion                    leaving namespaces untouched
      --stos-cluster-yaml string                   storageos-cluster.yaml path or url to be installed
      --stos-operator-yaml string                  storageos-operator.yaml path or url to be installed
      --uninstall-stos-cluster-namespace string    namespace of storageos cluster to be uninstalled (default "storageos")
      --uninstall-stos-operator-namespace string   namespace of storageos operator to be uninstalled (default "storageos")
      --version string                             version of storageos operator
      --wait                                       wait for storagos cluster to enter running phase
```

#### Upgrade StorageOS to the Latest Version

`kubectl storageos upgrade`

The **upgrade** commands uninstalls your existing StorageOS cluster and installs the latest StorageOS cluster.

### Preflight
```
# kubectl storageos preflight -h
A preflight check is a set of validations that can and should be run to ensure that a cluster meets the requirements to run StorageOS.

Usage:
  storageos preflight [url] [flags]

Flags:
      --as string                      Username to impersonate for the operation
      --as-group stringArray           Group to impersonate for the operation, this flag can be repeated to specify multiple groups.
      --cache-dir string               Default cache directory (default "/root/.kube/cache")
      --certificate-authority string   Path to a cert file for the certificate authority
      --client-certificate string      Path to a client certificate file for TLS
      --client-key string              Path to a client key file for TLS
      --cluster string                 The name of the kubeconfig cluster to use
      --collect-without-permissions    always run preflight checks even if some require permissions that preflight does not have
      --collector-image string         the full name of the collector image to use (default "storageos/troubleshoot:c77d9dc")
      --collector-pullpolicy string    the pull policy of the collector image
      --context string                 The name of the kubeconfig context to use
      --format string                  output format, one of human, json, yaml. only used when interactive is set to false (default "human")
  -h, --help                           help for preflight
      --insecure-skip-tls-verify       If true, the server's certificate will not be checked for validity. This will make your HTTPS connections insecure
      --interactive                    interactive preflights (default true)
      --kubeconfig string              Path to the kubeconfig file to use for CLI requests.
  -n, --namespace string               If present, the namespace scope for this CLI request
      --request-timeout string         The length of time to wait before giving up on a single server request. Non-zero values should contain a corresponding time unit (e.g. 1s, 2m, 3h). A value of zero means don't timeout requests. (default "0")
      --selector string                selector (label query) to filter remote collection nodes on.
  -s, --server string                  The address and port of the Kubernetes API server
      --since string                   force pod logs collectors to return logs newer than a relative duration like 5s, 2m, or 3h.
      --since-time string              force pod logs collectors to return logs after a specific date (RFC3339)
      --tls-server-name string         Server name to use for server certificate validation. If it is not provided, the hostname used to contact the server is used
      --token string                   Bearer token for authentication to the API server
      --user string                    The name of the kubeconfig user to use
```

## Config File

Flags can also be passed to the **install**, **uninstall** and **upgrade** commands via the kubectl storageos config file like so:

`kubectl storageos install --config-path=/path/to/config`

For this command to function correctly, the config file should be located and named like so: 

`/path/to/config/kubectl-storageos-config.yaml`

The **upgrade** command reads the `uninstall` and `install` settings in the config spec to perform the upgrade. 
The following is an example of a config file that might be used for an upgrade with custom namespaces:

```
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
    storageOSClusterNamespace: storageos-cluster-old
```

For an example config file, see:

`config/samples/_v1_kubectlstorageosconfig.yaml`

## TLS Enablement

### Install ETCD and StorageOS With TLS Enabled

`kubectl storageos install --include-etcd --etcd-tls-enabled`

### Install StorageOS and Connect to an Existing TLS Enabled ETCD Cluster

`kubectl storageos install --etcd-tls-enabled`

In order for StorageOS to connect to ETCD over HTTPS, a k8s secret must exist in the `storageos-cluster-namespace` with valid ETCD credentials. 
This secret can be created like so:

```
kubectl create secret generic <etcd-secret-name> -n <storageos-cluster-namespace> \
                --from-file=etcd-client-ca.crt=path/to/ca.crt \
                --from-file=etcd-client.crt=path/to/tls.crt \
                --from-file=etcd-client.key=path/to/tls.key
```
**Note:** The default `etcd-secret-name` is `storageos-etcd-secret`. Should you name your secret differently, you must pass the name to the install command via `--etcd-secret-name`

## Recovery

Before **uninstall** and **upgrade** commands are executed, a number of manifests relative to the existing StorageOS cluster are written locally to disk in order for the user to manually recover the cluster should an error occur.

These manifests can be located at `$HOME/.kube/storageos`.
