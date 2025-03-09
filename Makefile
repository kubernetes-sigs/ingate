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

help:  ## Display this help
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z0-9_-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

.PHONY: live-docs
live-docs: ## Build and launch a local copy of the documentation website in http://localhost:8000
	@docker build ${PLATFORM_FLAG} ${PLATFORM} \
                  		--no-cache \
                  		 -t ingress-nginx-docs .github/actions/mkdocs
	@docker run ${PLATFORM_FLAG} ${PLATFORM} --rm -it \
		-p 8000:8000 \
		-v ${PWD}:/docs \
		--entrypoint /bin/bash   \
		ingress-nginx-docs \
		-c "pip install -r /docs/docs/requirements.txt && mkdocs serve --dev-addr=0.0.0.0:8000"
