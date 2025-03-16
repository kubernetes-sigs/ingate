package main

import (
	"fmt"
	"github/kubernetes-sigs/ingate/cmd/ingate/root"
	"os"
)

func main() {
	if err := root.GetRootCommand().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
