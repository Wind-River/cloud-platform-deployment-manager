/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019 Wind River Systems, Inc. */

package controller

import (
	"github.com/wind-river/cloud-platform-deployment-manager/pkg/controller/datanetwork"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, datanetwork.Add)
}
