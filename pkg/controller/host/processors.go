/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019 Wind River Systems, Inc. */

package host

import (
	"strconv"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/cpus"
	perrors "github.com/pkg/errors"
	starlingxv1 "github.com/wind-river/cloud-platform-deployment-manager/pkg/apis/starlingx/v1"
	"github.com/wind-river/cloud-platform-deployment-manager/pkg/config"
	"github.com/wind-river/cloud-platform-deployment-manager/pkg/controller/common"
	v1info "github.com/wind-river/cloud-platform-deployment-manager/pkg/platform"
)

// GetCPUUpdateOpts is to get and combine the array of CPUOpts to update.
// The processors need to be updated in one shot as there's validation logic that
// may block the attempt to update every core individually.
func (r *ReconcileHost) GetCPUUpdateOpts(profile *starlingxv1.HostProfileSpec, host *v1info.HostInfo) ([]cpus.CPUOpts, bool) {
	updateRequired := false
	opts := make([]cpus.CPUOpts, 0)
	optsByFunction := make(map[string]*[]map[string]int)

	for _, nodeInfo := range profile.Processors {
		// For each NUMA node configuration

		for _, f := range nodeInfo.Functions {
			// For each function within a NUMA node configuration

			count := host.CountCPUByFunction(nodeInfo.Node, f.Function)
			if count != f.Count {
				updateRequired = true

				if socketListCache, ok := optsByFunction[f.Function]; ok {
					*socketListCache = append(*socketListCache, map[string]int{strconv.Itoa(nodeInfo.Node): f.Count})
				} else {
					optsByFunction[f.Function] = &[]map[string]int{{strconv.Itoa(nodeInfo.Node): f.Count}}
				}
			}
		}
	}

	if updateRequired {
		var cpuOpts cpus.CPUOpts
		for key, value := range optsByFunction {
			cpuOpts = cpus.CPUOpts{Function: key, Sockets: *value}
			opts = append(opts, cpuOpts)
		}
	}

	return opts, updateRequired
}

// ReconcileProcessors is responsible for reconciling the CPU configuration of a
// host resource.
func (r *ReconcileHost) ReconcileProcessors(client *gophercloud.ServiceClient, instance *starlingxv1.Host, profile *starlingxv1.HostProfileSpec, host *v1info.HostInfo) error {
	updated := false

	if len(profile.Processors) == 0 || !config.IsReconcilerEnabled(config.Processor) {
		return nil
	}

	opts, updateRequired := r.GetCPUUpdateOpts(profile, host)

	if updateRequired {
		log.Info("updating CPU configuration", "opts", opts)

		_, err := cpus.Update(client, host.ID, opts).Extract()
		if err != nil {
			err = perrors.Wrapf(err, "failed to update processors: %s, %s",
				host.ID, common.FormatStruct(opts))
			return err
		}

		updated = true
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
