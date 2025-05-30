package cmd

import (
	"flag"
	"github.com/kubernetes-sigs/ingate/internal/controlplane"
	"github.com/spf13/cobra"
	"k8s.io/klog/v2"
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

	// Initialize klog flags
	klog.InitFlags(nil)

	// Add klog flags to Cobra
	cmd.PersistentFlags().AddGoFlagSet(flag.CommandLine)

	return cmd
}
