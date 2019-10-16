/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019 Wind River Systems, Inc. */

package host

import (
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/cpus"
	perrors "github.com/pkg/errors"
	starlingxv1beta1 "github.com/wind-river/cloud-platform-deployment-manager/pkg/apis/starlingx/v1beta1"
	"github.com/wind-river/cloud-platform-deployment-manager/pkg/config"
	"github.com/wind-river/cloud-platform-deployment-manager/pkg/controller/common"
	v1info "github.com/wind-river/cloud-platform-deployment-manager/pkg/platform"
	"strconv"
)

// ReconcileProcessors is responsible for reconciling the CPU configuration of a
// host resource.
func (r *ReconcileHost) ReconcileProcessors(client *gophercloud.ServiceClient, instance *starlingxv1beta1.Host, profile *starlingxv1beta1.HostProfileSpec, host *v1info.HostInfo) error {
	updated := false

	if len(profile.Processors) == 0 || !config.IsReconcilerEnabled(config.Processor) {
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

				log.Info("updating CPU configuration", "opts", opts)

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
