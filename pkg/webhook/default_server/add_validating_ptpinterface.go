/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2022 Wind River Systems, Inc. */

package defaultserver

import (
	"fmt"

	"github.com/wind-river/cloud-platform-deployment-manager/pkg/webhook/default_server/ptpinterface/validating"
)

func init() {
	for k, v := range validating.Builders {
		_, found := builderMap[k]
		if found {
			log.V(1).Info(fmt.Sprintf(
				"conflicting webhook builder names in builder map: %v", k))
		}
		builderMap[k] = v
	}
	for k, v := range validating.HandlerMap {
		_, found := HandlerMap[k]
		if found {
			log.V(1).Info(fmt.Sprintf(
				"conflicting webhook builder names in handler map: %v", k))
		}
		_, found = builderMap[k]
		if !found {
			log.V(1).Info(fmt.Sprintf(
				"can't find webhook builder name %q in builder map", k))
			continue
		}
		HandlerMap[k] = v
	}
}
