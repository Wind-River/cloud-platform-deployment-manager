/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019 Wind River Systems, Inc. */

package host

import (
	"fmt"
	"github.com/alecthomas/units"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/memory"
	perrors "github.com/pkg/errors"
	starlingxv1beta1 "github.com/wind-river/titanium-deployment-manager/pkg/apis/starlingx/v1beta1"
	"github.com/wind-river/titanium-deployment-manager/pkg/config"
	"github.com/wind-river/titanium-deployment-manager/pkg/controller/common"
	v1info "github.com/wind-river/titanium-deployment-manager/pkg/platform"
)

// vswitchCountMemoryByFunction returns the number of pages of a particular size
// that the vswitch function is using on a given processor node/socket.
func vswitchCountMemoryByFunction(memories []memory.Memory, node int, pagesize starlingxv1beta1.PageSize) (int, error) {
	count := 0

	for _, mem := range memories {
		if mem.Processor != node {
			continue
		}

		if pagesize.Megabytes() == mem.VSwitchHugepagesSize {
			if mem.VSwitchHugepagesRequired == nil {
				count += mem.VSwitchHugepagesCount
			} else {
				count += *mem.VSwitchHugepagesRequired
			}
		}
	}

	return count, nil
}

// vmCountMemoryByFunction returns the number of pages of a particular size
// that the VM function is using on a given processor node/socket.
func vmCountMemoryByFunction(memories []memory.Memory, node int, pagesize starlingxv1beta1.PageSize) (int, error) {
	count := 0

	for _, mem := range memories {
		if mem.Processor != node {
			continue
		}

		if pagesize == starlingxv1beta1.PageSize2M {
			if mem.VM2MHugepagesPending == nil {
				count += mem.VM2MHugepagesCount
			} else {
				count += *mem.VM2MHugepagesPending
			}

		} else if pagesize == starlingxv1beta1.PageSize1G {
			if mem.VM1GHugepagesPending == nil {
				count += mem.VM1GHugepagesCount
			} else {
				count += *mem.VM1GHugepagesPending
			}
		}
	}

	return count, nil
}

// platformCountMemoryByFunction returns the number of pages of a particular
// size that the platform function is using on a given processor node/socket.
func platformCountMemoryByFunction(memories []memory.Memory, node int, pagesize starlingxv1beta1.PageSize) (int, error) {
	count := 0

	for _, mem := range memories {
		if mem.Processor != node {
			continue
		}

		if pagesize == starlingxv1beta1.PageSize4K {
			// Return the equivalent number of 4K pages that are currently
			// configured on the host.  Size the returned by the API as MiB we
			// need to multiply by 1MB and divide by 4K.
			count += (mem.Platform * int(units.Mebibyte)) / 4096
		}
	}

	return count, nil
}

// memoryCountByFunction counts the number of pages of a particular size that a
// specific function is using on a processor node/socket.
func memoryCountByFunction(data []memory.Memory, node int, function string, pagesize starlingxv1beta1.PageSize) (int, error) {
	switch function {
	case memory.MemoryFunctionVSwitch:
		return vswitchCountMemoryByFunction(data, node, pagesize)
	case memory.MemoryFunctionVM:
		return vmCountMemoryByFunction(data, node, pagesize)
	case memory.MemoryFunctionPlatform:
		return platformCountMemoryByFunction(data, node, pagesize)
	}

	msg := fmt.Sprintf("unsupported memory function: %s", function)
	return 0, common.NewUserDataError(msg)
}

// memoryUpdateRequired is a utility function which determines whether an
// update is required to adjust the memory configuration
func memoryUpdateRequired(f starlingxv1beta1.MemoryFunctionInfo, count int) (opts memory.MemoryOpts, result bool) {

	if count != f.PageCount {
		pageSize := starlingxv1beta1.PageSize(f.PageSize)

		opts.Function = f.Function

		if f.Function == memory.MemoryFunctionVM {
			if f.PageSize == string(starlingxv1beta1.PageSize1G) {
				opts.VMHugepages1G = &f.PageCount

			} else if f.PageSize == string(starlingxv1beta1.PageSize2M) {
				opts.VMHugepages2M = &f.PageCount
			}

		} else if f.Function == memory.MemoryFunctionVSwitch {
			if f.PageSize == string(starlingxv1beta1.PageSize1G) {
				opts.VSwitchHugepages = &f.PageCount
				hpSize := 1024
				opts.VSwitchHugepageSize = &hpSize

			} else if f.PageSize == string(starlingxv1beta1.PageSize2M) {
				opts.VSwitchHugepages = &f.PageCount
				hpSize := 2
				opts.VSwitchHugepageSize = &hpSize
			}

		} else if f.Function == memory.MemoryFunctionPlatform {
			size := f.PageCount * pageSize.Bytes() / int(units.Mebibyte)
			opts.Platform = &size
		}

		result = true
	}

	return opts, result
}

// ReconcileMemory is responsible for reconciling the Memory configuration of a
// host resource.
func (r *ReconcileHost) ReconcileMemory(client *gophercloud.ServiceClient, instance *starlingxv1beta1.Host, profile *starlingxv1beta1.HostProfileSpec, host *v1info.HostInfo) error {
	updated := false

	if len(profile.Memory) == 0 || !config.IsReconcilerEnabled(config.Memory) {
		return nil
	}

	// Retrieve the current CPU configuration
	objects, err := memory.ListMemory(client, host.ID)
	if err != nil {
		err = perrors.Wrapf(err, "failed to list memory for: %s", host.ID)
		return err
	}

	for _, nodeInfo := range profile.Memory {
		// For each NUMA node configuration
		mem, ok := host.FindMemory(nodeInfo.Node)
		if !ok {
			msg := fmt.Sprintf("failed to find memory resource for node %d", nodeInfo.Node)
			return starlingxv1beta1.NewMissingSystemResource(msg)
		}

		for _, f := range nodeInfo.Functions {
			// For each function within a NUMA node configuration
			pageSize := starlingxv1beta1.PageSize(f.PageSize)
			count, err := memoryCountByFunction(objects, nodeInfo.Node, f.Function, pageSize)
			if err != nil {
				return err
			}

			if opts, ok := memoryUpdateRequired(f, count); ok {
				log.Info("updating memory configuration", "opts", opts)

				_, err := memory.Update(client, mem.ID, opts).Extract()
				if err != nil {
					err = perrors.Wrapf(err, "failed to update memory: %s, %s",
						host.ID, common.FormatStruct(opts))
					return err
				}

				updated = true
			}
		}
	}

	if updated {
		r.NormalEvent(instance, common.ResourceUpdated,
			"memory allocations have been updated")

		results, err := memory.ListMemory(client, host.ID)
		if err != nil {
			err = perrors.Wrap(err, "failed to refresh host memory list")
			return err
		}

		host.Memory = results
	}

	return nil
}
