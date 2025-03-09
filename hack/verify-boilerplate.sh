#!/usr/bin/env bash

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

# Exit on error.
set -o errexit -o nounset -o pipefail

# Get root.
root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)"

# Define script.
script="${root}/hack/verify_boilerplate.py"

# Check script.
if [[ ! -f "${script}" ]]
then
  # Define version and URL.
  version="v0.2.5"
  url="https://raw.githubusercontent.com/kubernetes/repo-infra/${version}/hack/verify_boilerplate.py"

  # Download script.
  curl --silent --show-error --fail "${url}" --location --output "${script}"

  # Make executable.
  chmod +x "${script}"
fi

# Run script.
"${script}" --boilerplate-dir "${root}/hack/boilerplate" --rootdir "${root}"
