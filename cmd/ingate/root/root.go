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

package root

import (
	"github.com/kubernetes-sigs/ingate/internal/cmd"

	"github.com/spf13/cobra"
)

func GetRootCommand() *cobra.Command {
	c := &cobra.Command{
		Use:   "ingate",
		Short: "InGate Gateway and Ingress Controller",
		Long:  "InGate is a kubernetes controller for deploying and managing Gateway and Ingress resources",
	}

	c.AddCommand(cmd.GetVersionCommand())
	c.AddCommand(cmd.StartControlPlaneCommand())
	return c
}
