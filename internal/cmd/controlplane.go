package cmd

import (
	"github.com/kubernetes-sigs/ingate/internal/controlplane"
	"github.com/spf13/cobra"
)

func StartControlPlaneCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "start",
		Aliases: []string{"start", "s"},
		Short:   "Start InGate controller",
		RunE: func(cmd *cobra.Command, args []string) error {
			return controlplane.Start()
		},
	}

	return cmd
}
