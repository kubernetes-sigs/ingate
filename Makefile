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
# Alpine version for controller container
ALPINE=$(shell cat versions/ALPINE)
# REV is the short git sha of latest commit.
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
IMAGE ?= ${REGISTRY}/${IMAGE_NAME}


## help: Show this help info.
.PHONY: help
help:
	@awk 'BEGIN {FS = ":.*##"; printf ""} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)


.PHONY: versions
versions: ## List out versions of Software being used to develop InGate
	echo "GOLANG: ${GOLANG}"
	echo "INGATE: ${INGATE_VERSION}"
	echo "ALPINE: ${ALPINE}"
	echo "Commit SHA: ${COMMIT_SHA}"
	echo "HOST_ARCH: ${ARCH}"


## All Make targets for docker build


.PHONY: docker.image
docker.image: clean-image ## Build image for a particular arch.
	echo "Building docker ingate-controller ($(ARCH))..."
	docker build \
		${PLATFORM_FLAG} ${PLATFORM} \
		--no-cache \
		--build-arg BASE_IMAGE="$(BASE_IMAGE)" \
		--build-arg VERSION="$(TAG)" \
		--build-arg TARGETARCH="$(ARCH)" \
		--build-arg COMMIT_SHA="$(COMMIT_SHA)" \
		--build-arg BUILD_ID="$(BUILD_ID)" \
		-t $(REGISTRY)/controller:$(TAG) image/ingate-controller

.PHONY: clean-image
clean-image: ## Removes local image
	echo "removing old image $(REGISTRY)/controller:$(TAG)"
	@docker rmi -f $(REGISTRY)/controller:$(TAG) || true


## All Make targets for golang

VERSION_PACKAGE := github.com/kubernetes-sigs/ingate/internal/cmd/version

TARGETS_DIR := "images/ingate-controller/bin/${ARCH}"

PLATFORMS ?= amd64 arm arm64

.PHONY: go.build
go.build: ## Build go binary for InGate
	CGO_ENABLED=0 GOOS=$(OS) GOARCH=$(ARCH) go build -trimpath -ldflags="-buildid= -w -s \
  -X ${PKG}/version.RELEASE=${TAG} \
  -X ${PKG}/version.COMMIT=${COMMIT_SHA} \
  -X ${PKG}/version.REPO=${REPO_INFO}" \
  -buildvcs=false \
  -o "${TARGETS_DIR}/ingate" "${PKG}/cmd/ingate"

.PHONY: go.test.unit
go.test.unit: ## Run go unit tests
	go test -race ./...		