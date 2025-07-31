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

// Package version provides version information for InGate.
package version

import (
	// builtin
	"fmt"
	"io"
	"runtime"
)

// Version contains version information for InGate.
type Version struct {
	InGateVersion string `json:"ingateVersion"`
	GitCommitID   string `json:"gitCommitID"`
	GolangVersion string `json:"golangVersion"`
}

// GetVersion returns the current version information.
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

// Print writes version information to the provided writer.
func Print(w io.Writer) error {
	ver := GetVersion()

	if _, err := fmt.Fprintf(w, "INGATE_VERSION: %s\n", ver.InGateVersion); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "GIT_COMMIT_ID: %s\n", ver.GitCommitID); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "GOLANG_VERSION: %s\n", ver.GolangVersion); err != nil {
		return err
	}

	return nil
}
