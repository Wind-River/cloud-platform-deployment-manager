/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019-2022 Wind River Systems, Inc. */

package host

import (
	"strconv"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/cpus"
	perrors "github.com/pkg/errors"
	starlingxv1 "github.com/wind-river/cloud-platform-deployment-manager/api/v1"
	com "github.com/wind-river/cloud-platform-deployment-manager/common"
	"github.com/wind-river/cloud-platform-deployment-manager/controllers/common"
	v1info "github.com/wind-river/cloud-platform-deployment-manager/platform"
)

// ReconcileProcessors is responsible for reconciling the CPU configuration of a
// host resource.
func (r *HostReconciler) ReconcileProcessors(client *gophercloud.ServiceClient, instance *starlingxv1.Host, profile *starlingxv1.HostProfileSpec, host *v1info.HostInfo) error {
	updated := false

	if len(profile.Processors) == 0 || !com.IsReconcilerEnabled(com.Processor) {
		return nil
	}

	for _, nodeInfo := range profile.Processors {
		// For each NUMA node configuration

		for _, f := range nodeInfo.Functions {
			// For each function within a NUMA node configuration

			count := host.CountCPUByFunction(nodeInfo.Node, f.Function)
			if count != f.Count {
				opts := []cpus.CPUOpts{{
					Function: f.Function,
					Sockets:  []map[string]int{{strconv.Itoa(nodeInfo.Node): f.Count}},
				}}

				logHost.Info("updating CPU configuration", "opts", opts)

				_, err := cpus.Update(client, host.ID, opts).Extract()
				if err != nil {
					err = perrors.Wrapf(err, "failed to update processors: %s, %s",
						host.ID, common.FormatStruct(opts))
					return err
				}

				updated = true
			}
		}
	}

	if updated {
		r.NormalEvent(instance, common.ResourceUpdated,
			"cpu allocations have been updated")

		results, err := cpus.ListCPUs(client, host.ID)
		if err != nil {
			err = perrors.Wrap(err, "failed to refresh host CPU list")
			return err
		}

		host.CPU = results
	}

	return nil
}
