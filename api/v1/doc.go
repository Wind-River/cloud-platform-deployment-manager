/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019-2022 Wind River Systems, Inc. */

// Package v1 contains API Schema definitions for the StarlingX v1 API
// group.
//
// The schema definitions contained within are based on the StarlingX API
// definitions which are documented at the following location:
//
// https://docs.starlingx.io/api-ref/stx-config/index.html
//
// The API documentation contained within this package is intended to provide
// additional information related directly to the usage of the Deployment
// Manager.  There is only minimal information about the nature of each
// attribute to provide the reader with some context.  For a more thorough
// explanation the API definition at the aforementioned URL should be
// referenced.
// +kubebuilder:object:generate=true
// +deepequal-gen=package
// +k8s:conversion-gen=github.com/wind-river/cloud-platform-deployment-manager/pkg/apis/starlingx
// +k8s:defaulter-gen=TypeMeta
// +groupName=starlingx.windriver.com
package v1
