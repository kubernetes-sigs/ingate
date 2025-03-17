package main

import (
	"fmt"
	"os"

	"github.com/kubernetes-sigs/ingate/cmd/ingate/root"
)

func main() {
	if err := root.GetRootCommand().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
