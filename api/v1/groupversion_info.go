/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2022 Wind River Systems, Inc. */

// Package v1 contains API Schema definitions for the starlingx v1 API group
// +kubebuilder:object:generate=true
// +groupName=starlingx.windriver.com
//
//go:generate ../../bin/deepequal-gen -v 1 -O zz_generated.deepequal -i ./... -h ../../hack/boilerplate.go.txt
package v1

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/scheme"
)

var (
	// GroupVersion is group version used to register these objects
	GroupVersion = schema.GroupVersion{Group: "starlingx.windriver.com", Version: "v1"}

	// SchemeBuilder is used to add go types to the GroupVersionKind scheme
	SchemeBuilder = &scheme.Builder{GroupVersion: GroupVersion}

	// AddToScheme adds the types in this group-version to the given scheme.
	AddToScheme = SchemeBuilder.AddToScheme
)
