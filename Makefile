# SPDX-License-Identifier: Apache-2.0
# Copyright(c) 2019-2020 Wind River Systems, Inc.

# The Helm package command is not capable of figuring out if a package actually
# needs to be re-built therefore this Makefile will only invoke that command
# if it determines that any packaged files have changed.  This behaviour
# can be overridden with this variable.
HELM_FORCE ?= 0

# Image URL to use all building/pushing image targets
DEFAULT_IMG ?= wind-river/cloud-platform-deployment-manager
BUILDER_IMG ?= ${DEFAULT_IMG}-builder:latest

GIT_HEAD := $(shell git rev-list -1 HEAD)
GIT_LAST_TAG_COMMIT := $(shell git rev-list --tags --max-count=1)
GIT_LAST_TAG := $(shell git describe --tags $(GIT_LAST_TAG_COMMIT) )
GIT_BRANCH := $(shell git rev-parse --abbrev-ref HEAD)

DEPLOY_LDFLAGS := -X github.com/wind-river/cloud-platform-deployment-manager/cmd/deploy/cmd.GitLastTag=${GIT_LAST_TAG}
DEPLOY_LDFLAGS += -X github.com/wind-river/cloud-platform-deployment-manager/cmd/deploy/cmd.GitHead=${GIT_HEAD}
DEPLOY_LDFLAGS += -X github.com/wind-river/cloud-platform-deployment-manager/cmd/deploy/cmd.GitBranch=${GIT_BRANCH}

ifeq (${DEBUG}, yes)
	DOCKER_TARGET = debug
	GOBUILD_GCFLAGS = all=-N -l
	IMG ?= ${DEFAULT_IMG}:debug
else
	DOCKER_TARGET = production
	GOBUILD_GCFLAGS = ""
	IMG ?= ${DEFAULT_IMG}:latest
endif

.PHONY: examples

# Build all artifacts
all: test manager tools helm-package docker-build examples

# Publish all artifacts
publish: helm-package docker-push

# Run tests
test: generate fmt vet manifests helm-lint
	go test ./pkg/... ./cmd/... -coverprofile cover.out

# Build manager binary
manager: generate fmt vet
	go build -gcflags "${GOBUILD_GCFLAGS}" -o bin/manager github.com/wind-river/cloud-platform-deployment-manager/cmd/manager

# Build deploy binary
tools: generate fmt vet
	go build -ldflags "${DEPLOY_LDFLAGS}" -gcflags "${GOBUILD_GCFLAGS}" -o bin/deploy github.com/wind-river/cloud-platform-deployment-manager/cmd/deploy

# Run against the configured Kubernetes cluster in ~/.kube/config
run: manager
ifeq ($(DEBUG),yes)
	dlv --listen=:30000 --headless=true --api-version=2 --accept-multiclient exec bin/manager
else
	bin/manager
endif

# Install CRDs into a cluster
install: manifests
	kubectl apply -f config/crds

# Generate manifests e.g. CRD, RBAC etc.  The code generate RBAC files have a
# hardcoded namespace name that needs to be templated.
manifests:
	go run vendor/sigs.k8s.io/controller-tools/cmd/controller-gen/main.go all
	git grep -l "namespace: system" config/rbac | xargs -L1 sed -i 's#namespace: system#namespace: {{ .Values.namespace }}#g'

# Run go fmt against code
fmt:
	go fmt ./pkg/... ./cmd/...

# Run the golangci-lint static analysis
golangci:
	golangci-lint run ./pkg/...

# Run go vet against code
vet: golangci
	go vet ./pkg/... ./cmd/...

# Generate code
generate:
ifndef GOPATH
	$(error GOPATH not defined, please define GOPATH. Run "go help gopath" to learn more about GOPATH)
endif
	go generate ./pkg/... ./cmd/...

# Build the docker image
docker-build: test
	docker build . -t ${IMG} --target ${DOCKER_TARGET} --build-arg "GOBUILD_GCFLAGS=${GOBUILD_GCFLAGS}"

# Push the docker image
docker-push: docker-build
	docker push ${IMG}

# Build the builder image
builder-build:
	docker build . -t ${BUILDER_IMG} -f Dockerfile.builder

builder-run: builder-build
	docker run -v /var/run/docker.sock:/var/run/docker.sock \
		-v ${PWD}:/go/src/github.com/wind-river/cloud-platform-deployment-manager \
		--rm ${BUILDER_IMG}

# Check helm chart validity
helm-lint: manifests
	helm lint helm/wind-river-cloud-platform-deployment-manager

# Create helm chart package
.ONESHELL:
SHELL = /bin/bash
helm-package: helm-lint
	git update-index -q --ignore-submodules --refresh
	if [[ $$(comm -12 <(git diff-index --name-only HEAD | sort -u) <(find helm/wind-river-cloud-platform-deployment-manager config | sort -u) | wc -l) -ne 0 || ${HELM_FORCE} -ne 0 ]]; then
		helm package helm/wind-river-cloud-platform-deployment-manager --destination docs/charts;
		helm repo index docs/charts;
	fi

# Generate some example deployment configurations
examples:
	kustomize build examples/standard/default > examples/standard.yaml
	kustomize build examples/standard/vxlan > examples/standard-vxlan.yaml
	kustomize build examples/standard/https > examples/standard-https.yaml
	kustomize build examples/standard/bond > examples/standard-bond.yaml
	kustomize build examples/storage/default > examples/storage.yaml
	kustomize build examples/aio-sx/default > examples/aio-sx.yaml
	kustomize build examples/aio-sx/vxlan > examples/aio-sx-vxlan.yaml
	kustomize build examples/aio-sx/https > examples/aio-sx-https.yaml
	kustomize build examples/aio-dx/default > examples/aio-dx.yaml
	kustomize build examples/aio-dx/vxlan > examples/aio-dx-vxlan.yaml
	kustomize build examples/aio-dx/https > examples/aio-dx-https.yaml
