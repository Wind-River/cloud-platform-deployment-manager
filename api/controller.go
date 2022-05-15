/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019 Wind River Systems, Inc. */

package api

import (
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// AddToManagerFuncs is a list of functions to add all Controllers to the Manager
var AddToManagerControllersFuncs []func(manager.Manager) error

// AddToManager adds all Controllers to the Manager
func AddToManagerControllers(m manager.Manager) error {
	for _, f := range AddToManagerControllersFuncs {
		if err := f(m); err != nil {
			return err
		}
	}
	return nil
}
