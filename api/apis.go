/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019 Wind River Systems, Inc. */

// Generate deepcopy for apis
// Increase verbosity (-v 5) to help troubleshoot errors
//go:generate go run ../../vendor/k8s.io/code-generator/cmd/deepcopy-gen/main.go -v 1 5 -O zz_generated.deepcopy -i ./... -h ../../hack/boilerplate.go.txt
//go:generate go run ../../vendor/github.com/wind-river/deepequal-gen/main.go -v 1 -O zz_generated.deepequal -i ./... -h ../../hack/boilerplate.go.txt

// Package apis contains Kubernetes API groups.
package api

import (
	"k8s.io/apimachinery/pkg/runtime"
)

// AddToSchemes may be used to add all resources defined in the project to a Scheme
var AddToSchemes runtime.SchemeBuilder

// AddToSchemeApi adds all Resources to the Scheme
func AddToSchemeApi(s *runtime.Scheme) error {
	return AddToSchemes.AddToScheme(s)
}
