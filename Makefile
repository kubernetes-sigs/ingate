# Copyright 2025 The Kubernetes Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# Add the following 'help' target to your Makefile
# And add help text after each target name starting with '\#\#'

.DEFAULT_GOAL:=help

.EXPORT_ALL_VARIABLES:

ifndef VERBOSE
.SILENT:
endif

# set default shell
SHELL=/bin/bash -o pipefail -o errexit

# Set Root Directory Path
ifeq ($(origin ROOT_DIR),undefined)
ROOT_DIR := $(abspath $(shell pwd -P))
endif

# Golang root package
PKG = github.com/kubernetes-sigs/ingate
# Ingate version building
INGATE_VERSION=$(shell cat versions/INGATE)
# Golang version to build controller and container
GOLANG=$(shell cat versions/GOLANG)

# HOST_ARCH is the architecture that the developer is using to build it
HOST_ARCH=$(shell which go >/dev/null 2>&1 && go env GOARCH)
ARCH ?= $(HOST_ARCH)
ifeq ($(ARCH),)
	$(error mandatory variable ARCH is empty, either set it when calling the command or make sure 'go env GOARCH' works)
endif

# Build information for the repo source
REPO_INFO ?= $(shell git config --get remote.origin.url)

# Build information for git commit 
COMMIT_SHA ?= git-$(shell git rev-parse --short HEAD)

# Build information for build id in cloud build
BUILD_ID ?= "UNSET"

# REGISTRY is the image registry to use for build and push image targets.
REGISTRY ?= gcr.io/k8s-staging/ingate
# Name of the image
INGATE_IMAGE_NAME ?= controller
# IMAGE is the image URL for build and push image targets.
IMAGE ?= $(REGISTRY)/$(IMAGE_NAME)
BASE_IMAGE ?= $(shell cat versions/BASE_IMAGE)


.PHONY: help 
help: ## help: Show this help info.
	@awk 'BEGIN {FS = ":.*##"; printf ""} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)


.PHONY: versions
versions: ## List out versions of Software being used to develop InGate
	echo "GOLANG: ${GOLANG}"
	echo "INGATE: ${INGATE_VERSION}"
	echo "BASE_IMAGE: ${BASE_IMAGE}"
	echo "Commit SHA: ${COMMIT_SHA}"
	echo "HOST_ARCH: ${ARCH}"


# All Make targets for docker build

# Name of the docker buildx builder for InGate
BUILDER ?= ingate

# Supported Platforms for building multiarch binaries.
PLATFORMS ?= linux/amd64,linux/arm,linux/arm64

.PHONY: docker.build
docker.build: ## Build a local docker container for InGate
	docker build \
		--platform=$(PLATFORMS) \
		--no-cache \
		--build-arg TARGET_ARCH=$(ARCH) \
		-t $(REGISTRY)/controller:$(INGATE_VERSION) \
		-f images/ingate-controller/Dockerfile.run images/ingate-controller

.PHONY: docker.builder
docker.builder: ## Create buildx for multi-platform builds
	docker buildx create --name $(BUILDER) --bootstrap --use || :
	docker buildx inspect $(BUILDER)

# Docker output, --push or --load 
OUTPUT ?=

.PHONY: docker.buildx
docker.buildx: docker.builder docker.clean ## Build Ingate Controller image for a particular arch.
	echo "Building docker $(REGISTRY)/${INGATE_IMAGE_NAME}:$(INGATE_VERSION) ($(ARCH))..."
	docker buildx build \
		--builder $(BUILDER) \
		--platform $(PLATFORMS) \
		--no-cache \
		--build-arg TARGET_ARCH=$(ARCH) \
		-t $(REGISTRY)/${INGATE_IMAGE_NAME}:$(INGATE_VERSION) \
		-f images/ingate-controller/Dockerfile.run images/ingate-controller \
		$(OUTPUT)

.PHONY: docker.push
docker.push: OUTPUT = --push 
docker.push: docker.build ## Push docker container to a $REGISTRY

.PHONY: docker.clean
docker.clean: ## Removes local image
	echo "removing old image $(REGISTRY)/controller:$(INGATE_VERSION)"
	docker rmi -f $(REGISTRY)/controller:$(INGATE_VERSION) || true

# All Make targets for golang

# Where to place the golang built binarys
TARGETS_DIR = "./images/ingate-controller/bin/${ARCH}"

# Where to get the version information 
VERSION_PACKAGE = github.com/kubernetes-sigs/ingate/internal/cmd/version

GOOS := $(shell go env GOOS)
ifeq ($(origin GOOS), undefined)
		GOOS := $(shell go env GOOS)
endif

.PHONY: go.build
go.build: go.clean ## Go build for ingate controller
	echo "Building ingate controller"
	docker run \
	--volume "${PWD}":/go/src/$(PKG) \
	-w /go/src/$(PKG) \
	-e CGO_ENABLED=0 \
	-e GOOS=$(GOOS)  \
	-e GOARCH=$(TARGETARCH) \
	golang:1.24.1-alpine3.21 \
	 go build -trimpath -ldflags="-buildid= -w -s \
	-X $(VERSION_PACKAGE).inGateVersion=$(INGATE_VERSION) \
	-X $(VERSION_PACKAGE).gitCommitID=$(COMMIT_SHA)" \
	-buildvcs=false \
	-o $(TARGETS_DIR)/ingate $(PKG)/cmd/ingate

.PHONY: go.clean
go.clean: ## Clean go building output files
	rm -rf $(TARGETS_DIR)

.PHONY: go.test.unit
go.test.unit: ## Run go unit tests
	docker run -e CGO_ENABLED=1 golang:1.24.1-alpine3.21 go test -race ./...		

# All make targets for deploying a dev environment for InGate development

# Version of kubernetes to deploy on kind cluster
K8S_VERSION ?= $(shell cat versions/KUBERNETES_VERSIONS | head -n1)
# Name of kind cluster to deploy
KIND_CLUSTER_NAME ?= ingate-dev
# Gateway API Version to deploy on kind cluster
GW_VERSION ?= $(shell cat versions/GATEWAY_API)
# Gateway API channel to deploy on kind cluster See https://gateway-api.sigs.k8s.io/concepts/versioning/?h=chann#release-channels for more details 
GW_CHANNEL ?= standard

.PHONY: kind.all
kind.all: kind.build go.build docker.build kind.load lb.install gateway.install ingate.deploy ## Start a Development environment for InGate

.PHONY: kind.build
kind.build: kind.clean ## Build a kind cluster for testing InGate development
	echo "Creating kind cluster ${KIND_CLUSTER_NAME}"
	kind create cluster --config tools/kind/config.yaml --name $(KIND_CLUSTER_NAME) --image "kindest/node:$(K8S_VERSION)"

.PHONY: kind.clean
kind.clean: ## Deletes InGate-dev kind cluster 
	echo "Deleting old kind cluster"
	kind delete clusters $(KIND_CLUSTER_NAME) 

.PHONY: kind.load
kind.load: ## Load InGate Image onto kind cluster
	echo "Loading Kind cluster with $(REGISTRY)/${INGATE_IMAGE_NAME}:$(INGATE_VERSION)"
	kind load docker-image --name="$(KIND_CLUSTER_NAME)" "$(REGISTRY)"/"${INGATE_IMAGE_NAME}":"$(INGATE_VERSION)"

# Using the kubernetes sig tool https://github.com/kubernetes-sigs/cloud-provider-kind
.PHONY: lb.install
lb.install: lb.clean ## Install Load Balancer in kind cluster for Gateway deployments
	echo "Deploying Cloud Controller manager to the kind cluster"
	docker run --rm --network kind --name kindccm --detach -v /var/run/docker.sock:/var/run/docker.sock registry.k8s.io/cloud-provider-kind/cloud-controller-manager:v0.6.0

.PHONY: lb.clean
lb.clean: ## Stops the Kind Cloud Container
	echo "Cleaning up old Cloud controller manager"
	docker stop kindccm && docker rm kindccm || true

# example https://raw.githubusercontent.com/kubernetes-sigs/gateway-api/refs/heads/release-1.2/config/crd/standard/gateway.networking.k8s.io_gateways.yaml
.PHONY: gateway.install
gateway.install: ## Install Gateway API CRDs in cluster 
	echo "Installing Gateway API CRDs version ${GW_VERSION} onto kind cluster ${KIND_CLUSTER_NAME}"
	kubectl apply -f https://raw.githubusercontent.com/kubernetes-sigs/gateway-api/$(GW_VERSION)/config/crd/$(GW_CHANNEL)/gateway.networking.k8s.io_gatewayclasses.yaml
	kubectl apply -f https://raw.githubusercontent.com/kubernetes-sigs/gateway-api/$(GW_VERSION)/config/crd/$(GW_CHANNEL)/gateway.networking.k8s.io_gateways.yaml
	kubectl apply -f https://raw.githubusercontent.com/kubernetes-sigs/gateway-api/$(GW_VERSION)/config/crd/$(GW_CHANNEL)/gateway.networking.k8s.io_httproutes.yaml

ingate.deploy:
	echo "Deploying Ingate Controller via helm"
	helm upgrade -i ingate charts/ingate --namespace=ingate --create-namespace --set global.registry="${REGISTRY}" --wait --debug

.PHONY: docs.build
docs.build: ## Build and launch a local copy of the documentation website in http://localhost:8000
	@docker build --no-cache -t ingate-docs -f tools/docs/Dockerfile .
	@docker run --rm -it \
		-p 8000:8000 \
		-v ${PWD}:/docs \
		--entrypoint /bin/bash   \
		ingate-docs \
		-c "mkdocs serve --dev-addr=0.0.0.0:8000"

.PHONY: misspell
misspell:  ## Check for spelling errors.
	@go install github.com/client9/misspell/cmd/misspell@latest
	misspell \
		-locale US \
		-error \
		cmd/* internal/* docs/* test/* charts/* README.md