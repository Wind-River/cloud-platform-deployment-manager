# SPDX-License-Identifier: Apache-2.0
# Copyright(c) 2019-2022 Wind River Systems, Inc.

# The Helm package command is not capable of figuring out if a package actually
# needs to be re-built therefore this Makefile will only invoke that command
# if it determines that any packaged files have changed.  This behaviour
# can be overridden with this variable.
HELM_FORCE ?= 0

# Image URL to use all building/pushing image targets
DEFAULT_IMG ?= wind-river/cloud-platform-deployment-manager
BUILDER_IMG ?= ${DEFAULT_IMG}-builder:latest

HELM_CLIENT_VER := $(shell helm version --client --short 2>/dev/null | awk '{print $$NF}' | sed 's/^v//')
HELM_CLIENT_VER_REL := $(shell echo ${HELM_CLIENT_VER} | awk -F. '{print $$1}')
HELM_CLIENT_VER_MAJ := $(shell echo ${HELM_CLIENT_VER} | awk -F. '{print $$2}')

# Parameters for deploy tool
GIT_HEAD := $(shell git rev-list -1 HEAD)
GIT_LAST_TAG := $(shell git describe --abbrev=0 --tags)
GIT_BRANCH := $(shell git rev-parse --abbrev-ref HEAD)
GIT_PATCH := ""

APP_MODULE := "github.com/wind-river/cloud-platform-deployment-manager"
DEPLOY_LDFLAGS := -X ${APP_MODULE}/cmd/deploy/cmd.GitLastTag=${GIT_LAST_TAG}
DEPLOY_LDFLAGS += -X ${APP_MODULE}/cmd/deploy/cmd.GitPatch=${GIT_PATCH}
DEPLOY_LDFLAGS += -X ${APP_MODULE}/cmd/deploy/cmd.GitHead=${GIT_HEAD}
DEPLOY_LDFLAGS += -X ${APP_MODULE}/cmd/deploy/cmd.GitBranch=${GIT_BRANCH}

# ENVTEST_K8S_VERSION refers to the version of kubebuilder assets to be downloaded by envtest binary.
ENVTEST_K8S_VERSION = 1.23

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

# Setting SHELL to bash allows bash commands to be executed by recipes.
# This is a requirement for 'setup-envtest.sh' in the test target.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

ifeq (${DEBUG}, yes)
	DOCKER_TARGET = debug
	GOBUILD_GCFLAGS = all=-N -l
	IMG ?= ${DEFAULT_IMG}:debug
else
	DOCKER_TARGET = production
	GOBUILD_GCFLAGS = ""
	IMG ?= ${DEFAULT_IMG}:latest
endif

# Helm manifest for CRDs
HELM_CRDS=helm/wind-river-cloud-platform-deployment-manager/templates/crds.yaml

# DeepCopy auto generated file
DEEPCOPY_GEN_FILE=./api/v1/zz_generated.deepcopy.go

.PHONY: all
all: helm-ver-check test build tools helm-package docker-build examples

# Publish all artifacts
publish: helm-package docker-push

##@ General

##@ Development

.PHONY: manifests
manifests: controller-gen ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
	$(CONTROLLER_GEN) rbac:roleName=manager-role crd webhook paths="./..." output:crd:artifacts:config=config/crd/bases

.PHONY: generate
generate: controller-gen deepequal-gen ## Generate code containing DeepCopy, DeepEqual, DeepCopyInto, and DeepCopyObject method implementations.
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."
	## Replace array to rarray type
	## This is fix for https://github.com/kubernetes-sigs/controller-tools/issues/586
	## For example, []PlatformNetworkItem -> PlatformNetworkItemList
	##
	sed -i 's#\[\]PlatformNetworkItem#PlatformNetworkItemList#g' $(DEEPCOPY_GEN_FILE)
	sed -i 's#\[\]DataNetworkItem#DataNetworkItemList#g' $(DEEPCOPY_GEN_FILE)
	sed -i 's#\[\]PtpInterfaceItem#PtpInterfaceItemList#g' $(DEEPCOPY_GEN_FILE)
	sed -i 's#\[\]OSDInfo#OSDList#g' $(DEEPCOPY_GEN_FILE)
	sed -i 's#\[\]VolumeGroupInfo#VolumeGroupList#g' $(DEEPCOPY_GEN_FILE)
	sed -i 's#\[\]FileSystemInfo#FileSystemList#g' $(DEEPCOPY_GEN_FILE)
	sed -i 's#\[\]DNSServer#DNSServerList#g' $(DEEPCOPY_GEN_FILE)
	sed -i 's#\[\]NTPServer#NTPServerList#g' $(DEEPCOPY_GEN_FILE)
	sed -i 's#\[\]CertificateInfo#CertificateList#g' $(DEEPCOPY_GEN_FILE)
	sed -i 's#\[\]ServiceParameterInfo#ServiceParameterList#g' $(DEEPCOPY_GEN_FILE)
	sed -i 's#\[\]StorageBackend#StorageBackendList#g' $(DEEPCOPY_GEN_FILE)
	sed -i 's#\[\]ControllerFileSystemInfo#ControllerFileSystemList#g' $(DEEPCOPY_GEN_FILE)
	$(DEEPEQUAL_GEN) -v 1 -o ${PWD} -O zz_generated.deepequal -i ./api/v1 -h ./hack/boilerplate.go.txt  --gen-package-path ./api/v1

.PHONY: fmt
fmt: ## Run go fmt against code.
	go fmt ./...

golangci: golangci-lint ## Run the golangci-lint static analysis
	$(GOLANGCI_LINT) run ./api/...
	$(GOLANGCI_LINT) run ./controllers/...

.PHONY: vet
vet: golangci ## Run go vet against code.
	go vet ./...

.PHONY: test
test: manifests generate fmt vet envtest ## Run tests.
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) -p path)" go test ./... -coverprofile cover.out

##@ Build

.PHONY: build
build: generate fmt vet ## Build manager binary.
	go build -gcflags "${GOBUILD_GCFLAGS}" -o bin/manager main.go

.PHONY: tools
tools: generate fmt vet ## Build deploy binary.
	go build -ldflags "${DEPLOY_LDFLAGS}" -gcflags "${GOBUILD_GCFLAGS}" -o bin/deploy cmd/deploy/main.go

# Run against the configured Kubernetes cluster in ~/.kube/config
.PHONY: run
run: build
ifeq ($(DEBUG),yes)
	dlv --listen=:30000 --headless=true --api-version=2 --accept-multiclient exec bin/manager
else
	bin/manager
endif

.PHONY: docker-build
docker-build: test ## Build docker image with the manager.
	docker build . -t ${IMG} --target ${DOCKER_TARGET} --build-arg "GOBUILD_GCFLAGS=${GOBUILD_GCFLAGS}"

.PHONY: docker-push
docker-push: ## Push docker image with the manager.
	docker push ${IMG}

##@ Deployment

ifndef ignore-not-found
  ignore-not-found = false
endif

.PHONY: install
install: manifests kustomize ## Install CRDs into the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/crd | kubectl apply -f -

# .PHONY: uninstall
# uninstall: manifests kustomize ## Uninstall CRDs from the K8s cluster specified in ~/.kube/config. Call with ignore-not-found=true to ignore resource not found errors during deletion.
# 	$(KUSTOMIZE) build config/crd | kubectl delete --ignore-not-found=$(ignore-not-found) -f -

# .PHONY: deploy
# deploy: manifests kustomize ## Deploy controller to the K8s cluster specified in ~/.kube/config.
# 	cd config/manager && $(KUSTOMIZE) edit set image controller=${IMG}
# 	$(KUSTOMIZE) build config/default | kubectl apply -f -

# .PHONY: undeploy
# undeploy: ## Undeploy controller from the K8s cluster specified in ~/.kube/config. Call with ignore-not-found=true to ignore resource not found errors during deletion.
# 	$(KUSTOMIZE) build config/default | kubectl delete --ignore-not-found=$(ignore-not-found) -f -

##@ Build Dependencies

## Location to install dependencies to
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

## Tool Binaries
KUSTOMIZE ?= $(LOCALBIN)/kustomize
CONTROLLER_GEN ?= $(LOCALBIN)/controller-gen
ENVTEST ?= $(LOCALBIN)/setup-envtest
DEEPEQUAL_GEN ?= $(LOCALBIN)/deepequal-gen
GOLANGCI_LINT ?= $(LOCALBIN)/golangci-lint

## Tool Versions
KUSTOMIZE_VERSION ?= v3.8.7
CONTROLLER_TOOLS_VERSION ?= v0.8.0
GOLANGCI_LINT_VERSION ?= v1.49.0

KUSTOMIZE_INSTALL_SCRIPT ?= "https://raw.githubusercontent.com/kubernetes-sigs/kustomize/master/hack/install_kustomize.sh"
GOLANGCI_INSTALL_SCRIPT ?= "https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh"

.PHONY: kustomize
kustomize: $(KUSTOMIZE) ## Download kustomize locally if necessary.
$(KUSTOMIZE): $(LOCALBIN)
ifeq ("$(wildcard $(KUSTOMIZE))","")
	curl -s $(KUSTOMIZE_INSTALL_SCRIPT) | bash -s -- $(subst v,,$(KUSTOMIZE_VERSION)) $(LOCALBIN)
endif

.PHONY: controller-gen
controller-gen: $(CONTROLLER_GEN) ## Download controller-gen locally if necessary.
$(CONTROLLER_GEN): $(LOCALBIN)
	GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-tools/cmd/controller-gen@$(CONTROLLER_TOOLS_VERSION)

.PHONY: envtest
envtest: $(ENVTEST) ## Download envtest-setup locally if necessary.
$(ENVTEST): $(LOCALBIN)
	GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest

.PHONY: deepequal-gen
deepequal-gen: $(DEEPEQUAL_GEN) ## Download deepequal-gen locally if necessary.
$(DEEPEQUAL_GEN): $(LOCALBIN)
	GOBIN=$(LOCALBIN) go install github.com/wind-river/deepequal-gen@latest

.PHONY: golangci-lint
golangci-lint: $(GOLANGCI_LINT) ## Download golangci-lint locally if necessary.
$(GOLANGCI_LINT): $(LOCALBIN)
	curl -sSfL $(GOLANGCI_INSTALL_SCRIPT) | sh -s -- -b $(LOCALBIN) $(GOLANGCI_LINT_VERSION)

# Build the builder image
builder-build:
	docker build . -t ${BUILDER_IMG} -f Dockerfile.builder

builder-run: builder-build
	docker run -v /var/run/docker.sock:/var/run/docker.sock \
		-v ${PWD}:/go/src/github.com/wind-river/cloud-platform-deployment-manager \
		--rm ${BUILDER_IMG}

# Check minimum helm version
helm-ver-check:
	@if [[ ${HELM_CLIENT_VER_REL} < 2 || ( ${HELM_CLIENT_VER_REL} == 2 && ${HELM_CLIENT_VER_MAJ} < 16 ) ]]; then
		@echo "Minimum required helm client version is v2.16. Installed version is ${HELM_CLIENT_VER}"
		@/bin/false
	@fi

.PHONY:
helm_crds: manifests kustomize ## Generate CRDs for DM helm charts
	if [[ $$(comm -12 <(git diff-index --name-only HEAD | sort -u) <(find config/crd | sort -u) | wc -l) -ne 0 || ${HELM_FORCE} -ne 0 ]]; then
		$(KUSTOMIZE) build config/crd > ${HELM_CRDS}
		sed -i 's#cert-manager.io/inject-ca-from: $$(CERTIFICATE_NAMESPACE)/$$(CERTIFICATE_NAME)#cert-manager.io/inject-ca-from: {{ .Values.namespace }}/{{ .Values.namespace }}-serving-cert#g' ${HELM_CRDS}
		sed -i 's#namespace: system#namespace: {{ .Values.namespace }}#g' ${HELM_CRDS}
		sed -i 's#name: webhook-service#name: {{ .Values.namespace }}-webhook-service#g' ${HELM_CRDS}
	fi

# Check helm chart validity
helm-lint: manifests helm_crds
	helm lint helm/wind-river-cloud-platform-deployment-manager

# Create helm chart package
.ONESHELL:
SHELL = /bin/bash
helm-package: helm-ver-check helm-lint
	git update-index -q --ignore-submodules --refresh
	if [[ $$(comm -12 <(git diff-index --name-only HEAD | sort -u) <(find helm/wind-river-cloud-platform-deployment-manager config | sort -u) | wc -l) -ne 0 || ${HELM_FORCE} -ne 0 ]]; then
		helm package helm/wind-river-cloud-platform-deployment-manager --destination docs/charts;
		helm repo index docs/charts;
	fi

# Generate some example deployment configurations
.PHONY: examples
examples: kustomize
	$(KUSTOMIZE) build examples/standard/default > examples/standard.yaml
	$(KUSTOMIZE) build examples/standard/vxlan > examples/standard-vxlan.yaml
	$(KUSTOMIZE) build examples/standard/https > examples/standard-https.yaml
	$(KUSTOMIZE) build examples/standard/https-with-cert-manager > examples/standard-https-with-cert-manager.yaml
	$(KUSTOMIZE) build examples/standard/bond > examples/standard-bond.yaml
	$(KUSTOMIZE) build examples/standard/per-instance-ptp > examples/standard-per-instance-ptp.yaml
	$(KUSTOMIZE) build examples/standard/app-armor > examples/standard-app-armor.yaml
	$(KUSTOMIZE) build examples/standard/hw-settle > examples/standard-hw-settle.yaml
	$(KUSTOMIZE) build examples/storage/default > examples/storage.yaml
	$(KUSTOMIZE) build examples/aio-sx/default > examples/aio-sx.yaml
	$(KUSTOMIZE) build examples/aio-sx/vxlan > examples/aio-sx-vxlan.yaml
	$(KUSTOMIZE) build examples/aio-sx/https > examples/aio-sx-https.yaml
	$(KUSTOMIZE) build examples/aio-sx/https-with-cert-manager > examples/aio-sx-https-with-cert-manager.yaml
	$(KUSTOMIZE) build examples/aio-sx/single-nic > examples/aio-sx-single-nic.yaml
	$(KUSTOMIZE) build examples/aio-sx/vf-rate-limit > examples/aio-sx-vf-rate-limit.yaml
	$(KUSTOMIZE) build examples/aio-sx/geo-location > examples/aio-sx-geo-location.yaml
	$(KUSTOMIZE) build examples/aio-sx/app-armor > examples/aio-sx-app-armor.yaml
	$(KUSTOMIZE) build examples/aio-sx/hw-settle > examples/aio-sx-hw-settle.yaml
	$(KUSTOMIZE) build examples/aio-dx/default > examples/aio-dx.yaml
	$(KUSTOMIZE) build examples/aio-dx/vxlan > examples/aio-dx-vxlan.yaml
	$(KUSTOMIZE) build examples/aio-dx/https > examples/aio-dx-https.yaml
	$(KUSTOMIZE) build examples/aio-dx/https-with-cert-manager > examples/aio-dx-https-with-cert-manager.yaml
