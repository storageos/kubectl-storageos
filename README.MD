# StorageOS kubectl plugin

[![Go Report Card](https://goreportcard.com/badge/github.com/storageos/kubectl-storageos)](https://goreportcard.com/report/github.com/storageos/kubectl-storageos)
[![e2e test](https://github.com/storageos/kubectl-storageos/actions/workflows/kuttl-e2e-test.yaml/badge.svg)](https://github.com/storageos/kubectl-storageos/actions/workflows/kuttl-e2e-test.yaml)
[![CodeQL](https://github.com/storageos/kubectl-storageos/actions/workflows/codeql-analysis.yml/badge.svg)](https://github.com/storageos/kubectl-storageos/actions/workflows/codeql-analysis.yml)
[![Active](http://img.shields.io/badge/Status-Active-green.svg)](https://github.com/storageos/kubectl-storageos)
[![PR's Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg?style=flat)](https://github.com/storageos/kubectl-storageos/pulls)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](https://opensource.org/licenses/MIT)

This project is under development, use it on your own risk please.

The main goal of this project is to give an easy integration with Kubernetes for StorageOS related administration tasks.

## Installation

### Pre-requisites

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
  install     Install StorageOS Cluster Operator

Flags:
      --as string                      Username to impersonate for the operation
      --as-group stringArray           Group to impersonate for the operation, this flag can be repeated to specify multiple groups.
      --cache-dir string               Default cache directory (default "/home/mhmxs/.kube/cache")
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

