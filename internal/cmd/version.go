package cmd

import (
	"github.com/kubernetes-sigs/ingate/internal/cmd/version"
	"github.com/spf13/cobra"
)

func GetVersionCommand() *cobra.Command {

	cmd := &cobra.Command{
		Use:     "version",
		Aliases: []string{"versions", "v"},
		Short:   "Show versions",
		RunE: func(cmd *cobra.Command, args []string) error {
			return version.Print()
		},
	}

	return cmd
}
