package root

import (
	"github.com/kubernetes-sigs/ingate/internal/cmd"
	"github.com/spf13/cobra"
)

func GetRootCommand() *cobra.Command {
	c := &cobra.Command{
		Use:   "ingate",
		Short: "InGate Gateway and Ingress Controller",
		Long:  "InGate is a kubernetes contoller for deploy and managing Gateway and Ingress resources",
	}

	c.AddCommand(cmd.GetServerCommand())
	return c
}
