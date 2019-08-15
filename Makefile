# SPDX-License-Identifier: Apache-2.0
# Copyright(c) 2019 Wind River Systems, Inc.

# Image URL to use all building/pushing image targets
DEFAULT_IMG ?= titanium/deployment-manager
EXAMPLES ?= ${HOME}/tmp/titanium-deployment-manager/examples

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
all: test manager tools helm-package docker-build

# Publish all artifacts
publish: helm-publish docker-push

# Run tests
test: generate fmt vet manifests helm-lint
	go test ./pkg/... ./cmd/... -coverprofile cover.out

# Build manager binary
manager: generate fmt vet
	go build -gcflags "${GOBUILD_GCFLAGS}" -o bin/manager github.com/wind-river/titanium-deployment-manager/cmd/manager

# Build manager binary
tools: generate fmt vet
	go build -gcflags "${GOBUILD_GCFLAGS}" -o bin/deploy github.com/wind-river/titanium-deployment-manager/cmd/deploy

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

# Check helm chart validity
helm-lint: manifests
	helm lint helm/titanium-deployment-manager

# Create helm chart package
helm-package: helm-lint
	helm package helm/titanium-deployment-manager --destination docs/charts

# Update the helm repo
helm-publish: helm-package
	helm repo index docs/charts

# Generate some example deployment configurations
examples:
	mkdir -p ${EXAMPLES}
	kustomize build examples/standard/default  > ${EXAMPLES}/standard.yaml
	kustomize build examples/standard/vxlan > ${EXAMPLES}/standard-vxlan.yaml
	kustomize build examples/standard/https > ${EXAMPLES}/standard-https.yaml
	kustomize build examples/standard/bond > ${EXAMPLES}/standard-bond.yaml
	kustomize build examples/storage/default  > ${EXAMPLES}/storage.yaml
	kustomize build examples/aio-sx/default > ${EXAMPLES}/aio-sx.yaml
	kustomize build examples/aio-sx/vxlan > ${EXAMPLES}/aio-sx-vxlan.yaml
	kustomize build examples/aio-sx/https > ${EXAMPLES}/aio-sx-https.yaml
	kustomize build examples/aio-dx/default > ${EXAMPLES}/aio-dx.yaml
	kustomize build examples/aio-dx/vxlan > ${EXAMPLES}/aio-dx-vxlan.yaml
	kustomize build examples/aio-dx/https > ${EXAMPLES}/aio-dx-https.yaml
