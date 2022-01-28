.PHONY: e2e

# Image URL to use all building/pushing image targets
IMG ?= storageos/kubectl-storageos:test

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

KUBECTL_STOS_VERSION ?= v1.1.0

# Generate kuttl e2e tests for the following storageos/kind-node versions
# TEST_KIND_NODES is not intended to be updated manually.
# Please edit LATEST_KIND_NODE instead and run 'make update-kind-nodes'.
TEST_KIND_NODES ?= 1.18.6,1.19.0,1.20.5,1.21.0,1.22.3

LATEST_KIND_NODE ?= 1.22.3
REPO ?= kubectl-storageos

# Setting SHELL to bash allows bash commands to be executed by recipes.
# This is a requirement for 'setup-envtest.sh' in the test target.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

# Produce CRDs that work back to Kubernetes 1.11 (no version conversion)
CRD_OPTIONS ?= "crd:trivialVersions=true,preserveUnknownFields=false"

LDF_FLAGS = -X github.com/storageos/kubectl-storageos/pkg/version.PluginVersion=

BUILDFLAGS = -tags "exclude_graphdriver_btrfs exclude_graphdriver_devicemapper"

all: build

##@ General

# The help target prints out all targets with their descriptions organized
# beneath their categories. The categories are represented by '##@' and the
# target descriptions by '##'. The awk commands is responsible for reading the
# entire set of makefiles included in this invocation, looking for lines of the
# file as xyz: ## something, and then pretty-format the target and help. Then,
# if there's a line with ##@ something, that gets pretty-printed as a category.
# More info on the usage of ANSI control characters for terminal formatting:
# https://en.wikipedia.org/wiki/ANSI_escape_code#SGR_parameters
# More info on the awk command:
# http://linuxcommand.org/lc3_adv_awk.php

help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

generate: controller-gen ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

fmt: ## Run go fmt against code.
	go fmt ./...

vet: ## Run go vet against code.
	go vet ${BUILDFLAGS} ./...

ENVTEST_ASSETS_DIR=$(shell pwd)/testbin
test: fmt vet generate ## Run tests.
	mkdir -p ${ENVTEST_ASSETS_DIR}
	test -f ${ENVTEST_ASSETS_DIR}/setup-envtest.sh || curl -sSLo ${ENVTEST_ASSETS_DIR}/setup-envtest.sh https://raw.githubusercontent.com/kubernetes-sigs/controller-runtime/v0.7.2/hack/setup-envtest.sh
	source ${ENVTEST_ASSETS_DIR}/setup-envtest.sh; fetch_envtest_tools $(ENVTEST_ASSETS_DIR); setup_envtest_env $(ENVTEST_ASSETS_DIR); go test -v ${BUILDFLAGS} github.com/storageos/kubectl-storageos/...

e2e: ## Run e2e tests against latest supported k8s cluster.  
	kubectl-kuttl test --config e2e/kuttl/installer-1.22.yaml
	kubectl-kuttl test --config e2e/kuttl/upgrade-1.22.yaml

##@ Build

build: test ## test and build manager binary.
	make _build

_build: ## Build manager binary.
	go build ${BUILDFLAGS} -ldflags "$(LDF_FLAGS)$(KUBECTL_STOS_VERSION)" -o bin/kubectl-storageos github.com/storageos/kubectl-storageos

_build-pre: ## Build manager binary.
	go build ${BUILDFLAGS} -ldflags "$(LDF_FLAGS)$(KUBECTL_STOS_VERSION) -X github.com/storageos/kubectl-storageos/pkg/version.EnableUnofficialRelease=true" -o bin/kubectl-storageos github.com/storageos/kubectl-storageos

run: fmt vet generate ## Run a controller from your host.
	go run ./main.go

update-kind-nodes: 
	LATEST_KIND_NODE=$(LATEST_KIND_NODE) ./hack/update-kind-nodes.sh

generate-tests: ## Generate kuttl e2e tests
	TEST_KIND_NODES=$(TEST_KIND_NODES) REPO=$(REPO) ./hack/generate-tests.sh

##@ Deployment
manifests: controller-gen ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
		$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=manager-role webhook paths="./..." output:crd:artifacts:config=config/crd/bases

install: kustomize ## Install CRDs into the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/crd | kubectl apply -f -

uninstall: kustomize ## Uninstall CRDs from the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/crd | kubectl delete -f -

deploy: kustomize ## Deploy controller to the K8s cluster specified in ~/.kube/config.
	cd config/manager && $(KUSTOMIZE) edit set image controller=${IMG}
	$(KUSTOMIZE) build config/default | kubectl apply -f -

undeploy: ## Undeploy controller from the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/default | kubectl delete -f -

deps: controller-gen kustomize ## Download all dependencies if necessary.

CONTROLLER_GEN = $(shell pwd)/bin/controller-gen
controller-gen: ## Download controller-gen locally if necessary.
	$(call go-get-tool,$(CONTROLLER_GEN),sigs.k8s.io/controller-tools/cmd/controller-gen@v0.4.1)

KUSTOMIZE = $(shell pwd)/bin/kustomize
kustomize: ## Download kustomize locally if necessary.
	$(call go-get-tool,$(KUSTOMIZE),sigs.k8s.io/kustomize/kustomize/v3@v3.8.7)

# go-get-tool will 'go get' any package $2 and install it to $1.
PROJECT_DIR := $(shell dirname $(abspath $(lastword $(MAKEFILE_LIST))))
define go-get-tool
@[ -f $(1) ] || { \
set -e ;\
TMP_DIR=$$(mktemp -d) ;\
cd $$TMP_DIR ;\
go mod init tmp ;\
echo "Downloading $(2)" ;\
GOBIN=$(PROJECT_DIR)/bin go get $(2) ;\
rm -rf $$TMP_DIR ;\
}
endef
