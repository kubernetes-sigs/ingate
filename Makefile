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


REPO_INFO ?= $(shell git config --get remote.origin.url)

COMMIT_SHA ?= git-$(shell git rev-parse --short HEAD)

BUILD_ID ?= "UNSET"

# REGISTRY is the image registry to use for build and push image targets.
REGISTRY ?= gcr.io/k8s-staging/ingate
# Name of the image
INGATE_IMAGE_NAME ?= ingate-controller
# IMAGE is the image URL for build and push image targets.
IMAGE ?= $(REGISTRY)/$(IMAGE_NAME)
BASE_IMAGE ?= $(shell cat versions/BASE_IMAGE)

## help: Show this help info.
.PHONY: help
help:
	@awk 'BEGIN {FS = ":.*##"; printf ""} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)


.PHONY: versions
versions: ## List out versions of Software being used to develop InGate
	echo "GOLANG: ${GOLANG}"
	echo "INGATE: ${INGATE_VERSION}"
	echo "BASE_IMAGE: ${BASE_IMAGE}"
	echo "Commit SHA: ${COMMIT_SHA}"
	echo "HOST_ARCH: ${ARCH}"


## All Make targets for docker build

.PHONY: docker.build
docker.build: docker.clean ## Build image for a particular arch.
	echo "Building docker ingate-controller ($(ARCH))..."
	docker build \
		$(PLATFORM_FLAG) $(PLATFORM) \
		--no-cache \
		--build-arg BASE_IMAGE="$(BASE_IMAGE)" \
		--build-arg VERSION="$(INGATE_VERSION)" \
		--build-arg TARGETARCH="$(ARCH)" \
		--build-arg COMMIT_SHA="$(COMMIT_SHA)" \
		--build-arg BUILD_ID="$(BUILD_ID)" \
		-t $(REGISTRY)/controller:$(INGATE_VERSION) images/ingate-controller

.PHONY: docker.clean
docker.clean: ## Removes local image
	echo "removing old image $(REGISTRY)/controller:$(INGATE_VERSION)"
	@docker rmi -f $(REGISTRY)/controller:$(INGATE_VERSION) || true

docker.registry: ## Starts a local docker registry for testing
	docker rm registry || true
	docker run -d -p 6000:5000 --name registry registry:2.7


## All Make targets for golang

# Where to place the golang built binarys
TARGETS_DIR := "./images/ingate-controller/bin/${ARCH}"

# Supported Platforms for building multiarch binaries.
PLATFORMS ?= darwin_amd64 darwin_arm64 linux_amd64 linux_arm64

GOPATH := $(shell go env GOPATH)
ifeq ($(origin GOBIN), undefined)
	GOBIN := $(GOPATH)/bin
endif

GOOS := $(shell go env GOOS)
ifeq ($(origin GOOS), undefined)
		GOOS := $(shell go env GOOS)
endif

VERSION_PACKAGE := github.com/kubernetes-sigs/ingate/internal/cmd/version

.PHONY: go.build
go.build: ## Build go binary for InGate
	CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(ARCH) go build -trimpath -ldflags="-buildid= -w -s \
  -X $(VERSION_PACKAGE).inGateVersion=$(INGATE_VERSION) \
  -X $(VERSION_PACKAGE).gitCommitID=$(COMMIT_SHA)" \
  -buildvcs=false \
  -o "$(TARGETS_DIR)/ingate" "$(PKG)/cmd/ingate"

.PHONY: go.clean
go.clean: ## Clean go building output files
	rm -rf $(TARGETS_DIR)

.PHONY: go.test.unit
go.test.unit: ## Run go unit tests
	go test -race ./...		

## All make targets for deploying a dev environment for InGate development

# Version of kubernetes to deploy on kind cluster
K8S_VERSION ?= $(shell cat versions/KUBERNETES_VERSIONS | head -n1)
# Name of kind cluster to deploy
KIND_CLUSTER_NAME := ingate-dev
# Gateway API Version to deploy on kind cluster
GW_VERSION ?= $(shell cat versions/GATEWAY_API)
# Gateway API channel to deploy on kind cluster See https://gateway-api.sigs.k8s.io/concepts/versioning/?h=chann#release-channels for more details 
GW_CHANNEL ?= standard

kind.all: kind.build go.build docker.build kind.load metallb.install gateway.install ## Start a Development environment for InGate

kind.build: kind.clean ## Build a kind cluster for testing
	kind create cluster --config tools/kind/config.yaml --name $(KIND_CLUSTER_NAME)   --image "kindest/node:$(K8S_VERSION)"

kind.clean: ## Deletes kind cluster 
	kind delete clusters $(KIND_CLUSTER_NAME) 

KIND_WORKERS=$(shell kind get nodes --name="${KIND_CLUSTER_NAME}" | grep worker | awk '{printf (NR>1?",":"") $1}')

kind.load: ## Load InGate Image onto kind cluster
	kind load docker-image --name="$(KIND_CLUSTER_NAME)" --nodes="$(KIND_WORKERS)" "$(REGISTRY)"/controller:"$(TAG)"

metallb.install: ## Install metalLB in kind cluster for Load Balancing 
	tools/scripts/install_metallb.sh

# example https://raw.githubusercontent.com/kubernetes-sigs/gateway-api/refs/heads/release-1.2/config/crd/standard/gateway.networking.k8s.io_gateways.yaml
gateway.install: ## Install Gateway API CRDs in cluster 
	kubectl apply -f https://raw.githubusercontent.com/kubernetes-sigs/gateway-api/$(GW_VERSION)/config/crd/$(GW_CHANNEL)/gateway.networking.k8s.io_gatewayclasses.yaml
	kubectl apply -f https://raw.githubusercontent.com/kubernetes-sigs/gateway-api/$(GW_VERSION)/config/crd/$(GW_CHANNEL)/gateway.networking.k8s.io_gateways.yaml
	kubectl apply -f https://raw.githubusercontent.com/kubernetes-sigs/gateway-api/$(GW_VERSION)/config/crd/$(GW_CHANNEL)/gateway.networking.k8s.io_httproutes.yaml
	kubectl apply -f https://raw.githubusercontent.com/kubernetes-sigs/gateway-api/$(GW_VERSION)/config/crd/$(GW_CHANNEL)/gateway.networking.k8s.io_referencegrants.yaml
	kubectl apply -f https://raw.githubusercontent.com/kubernetes-sigs/gateway-api/$(GW_VERSION)/config/crd/$(GW_CHANNEL)/gateway.networking.k8s.io_grpcroutes.yaml
