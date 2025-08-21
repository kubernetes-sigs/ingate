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

package framework

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

// This file defines a common envtest constructor. The idea is that we don't
// keep repeating ourselves everytime we need an envtest to check reconciliation
// loops

func StartEnvTest(scheme *runtime.Scheme, k8sVersion, crdChannel string) (*envtest.Environment, *rest.Config, error) {
	// Get the local GatewayAPI CRDs. If this is failing you must execute `go mod tidy` first
	gwAPIPath, err := exec.Command("go", "list", "-m", "-f", "{{.Dir}}", "sigs.k8s.io/gateway-api").CombinedOutput()
	if err != nil {
		return nil, nil, fmt.Errorf("error finding local gwapi CRDs: %w", err)
	}

	if crdChannel == "" {
		crdChannel = "standard"
	}

	gwAPICRDPath := filepath.Join(strings.TrimSpace(string(gwAPIPath)), "config", "crd", crdChannel)
	dirStat, err := os.Stat(gwAPICRDPath)
	if err != nil {
		return nil, nil, fmt.Errorf("error checking CRD API dir %s for test: %w", gwAPICRDPath, err)
	}
	if !dirStat.IsDir() {
		return nil, nil, fmt.Errorf("%s is not a directory", gwAPICRDPath)
	}

	testEnv := &envtest.Environment{
		Scheme:                      scheme,
		ErrorIfCRDPathMissing:       true,
		DownloadBinaryAssets:        true,
		DownloadBinaryAssetsVersion: k8sVersion,
		CRDInstallOptions: envtest.CRDInstallOptions{
			Paths: []string{
				gwAPICRDPath,
			},
			CleanUpAfterUse: true,
		},
		AttachControlPlaneOutput: true,
	}

	restConfig, err := testEnv.Start()
	if err != nil {
		return nil, nil, fmt.Errorf("error starting envtest: %w", err)
	}

	return testEnv, restConfig, nil
}
