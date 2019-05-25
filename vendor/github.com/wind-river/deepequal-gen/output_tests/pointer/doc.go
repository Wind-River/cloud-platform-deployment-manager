/*
SPDX-License-Identifier: Apache-2.0
Copyright 2017 The Kubernetes Authors.
*/

// +deepequal-gen=package

// This is a test package.
package pointer

type Ttest struct {
	Builtin *string
	Struct  *Ttest
}
