/*
Copyright 2025 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package version

import (
	"fmt"
	"io"
	"runtime"
)

type Version struct {
	InGateVersion string `json:"ingateVersion"`
	GitCommitID   string `json:"gitCommitID"`
	GolangVersion string `json:"golangVersion"`
}

func GetVersion() Version {
	return Version{
		InGateVersion: inGateVersion,
		GitCommitID:   gitCommitID,
		GolangVersion: runtime.Version(),
	}
}

var (
	inGateVersion string
	gitCommitID   string
)

func Print(w io.Writer) error {
	ver := GetVersion()

	_, _ = fmt.Fprintf(w, "INGATE_VERSION: %s\n", ver.InGateVersion)
	_, _ = fmt.Fprintf(w, "GIT_COMMIT_ID: %s\n", ver.GitCommitID)
	_, _ = fmt.Fprintf(w, "GOLANG_VERSION: %s\n", ver.GolangVersion)

	return nil
}
