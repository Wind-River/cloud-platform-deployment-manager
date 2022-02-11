/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2022 Wind River Systems, Inc. */

package controller

import (
	"github.com/wind-river/cloud-platform-deployment-manager/pkg/controller/ptpinstance"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, ptpinstance.Add)
}
