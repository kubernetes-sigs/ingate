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
