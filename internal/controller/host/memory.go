/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019, 2022, 2026 Wind River Systems, Inc. */

package host

import (
	"fmt"

	"github.com/alecthomas/units"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/memory"
	perrors "github.com/pkg/errors"
	starlingxv1 "github.com/wind-river/cloud-platform-deployment-manager/api/v1"
	"github.com/wind-river/cloud-platform-deployment-manager/common"
	utils "github.com/wind-river/cloud-platform-deployment-manager/internal/controller/common"
	v1info "github.com/wind-river/cloud-platform-deployment-manager/platform"
)

// vswitchCountMemoryByFunction returns the number of pages of a particular size
// that the vswitch function is using on a given processor node/socket.
func vswitchCountMemoryByFunction(memories []memory.Memory, node int, pagesize starlingxv1.PageSize) (int, error) {
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
func vmCountMemoryByFunction(memories []memory.Memory, node int, pagesize starlingxv1.PageSize) (int, error) {
	count := 0

	for _, mem := range memories {
		if mem.Processor != node {
			continue
		}

		switch pagesize {
		case starlingxv1.PageSize2M:
			if mem.VM2MHugepagesPending == nil {
				count += mem.VM2MHugepagesCount
			} else {
				count += *mem.VM2MHugepagesPending
			}

		case starlingxv1.PageSize1G:
			if mem.VM1GHugepagesPending == nil {
				count += mem.VM1GHugepagesCount
			} else {
				count += *mem.VM1GHugepagesPending
			}
		}
	}

	return count, nil
}

// platformMiBToPageCount converts a platform memory reservation expressed in
// whole MiB (as tracked by the system API in imemorys.platform_reserved_mib)
// into the equivalent number of pages of the given page size.  Platform memory
// is only ever expressed in 4K pages, so pagesize is always PageSize4K here;
// it is kept as a parameter to mirror the other memory conversion helpers.
//
// The system API only stores platform memory in whole MiB, so this conversion
// is lossy for any page count that is not a whole number of MiB (i.e. not a
// multiple of 256 4K pages).  Both the readback of the current configuration
// and the comparison of the desired configuration must use this same
// calculation, otherwise the current and desired page counts are expressed in
// different granularities and the reconciler reports a difference that can
// never converge.
func platformMiBToPageCount(mib int, pagesize starlingxv1.PageSize) int {
	return (mib * int(units.Mebibyte)) / pagesize.Bytes()
}

// platformPageCountToMiB converts a platform memory page count into the whole
// MiB value accepted by the system API.  This is the inverse of
// platformMiBToPageCount and truncates to whole MiB, matching the granularity
// of the system API.  As with platformMiBToPageCount, platform memory is only
// ever expressed in 4K pages, so pagesize is always PageSize4K.
func platformPageCountToMiB(pageCount int, pagesize starlingxv1.PageSize) int {
	return (pageCount * pagesize.Bytes()) / int(units.Mebibyte)
}

// alignPlatformMemory rounds the page count of a platform memory function down
// to whole MiB, the granularity the system API can store, by running it
// through the same MiB round-trip used to read the current value back.  It is
// a no-op for any other function.  Both the update decision
// (memoryUpdateRequired) and the profile comparison
// (NormalizePlatformMemoryForComparison) use it so the desired value is always
// expressed in the same granularity as the current one.
func alignPlatformMemory(f *starlingxv1.MemoryFunctionInfo) {
	if f.Function != memory.MemoryFunctionPlatform {
		return
	}
	pageSize := starlingxv1.PageSize(f.PageSize)
	f.PageCount = platformMiBToPageCount(platformPageCountToMiB(f.PageCount, pageSize), pageSize)
}

// platformCountMemoryByFunction returns the number of pages of a particular
// size that the platform function is using on a given processor node/socket.
func platformCountMemoryByFunction(memories []memory.Memory, node int, pagesize starlingxv1.PageSize) (int, error) {
	count := 0

	for _, mem := range memories {
		if mem.Processor != node {
			continue
		}

		if pagesize == starlingxv1.PageSize4K {
			// Return the equivalent number of 4K pages that are currently
			// configured on the host.  The size reported by the API is in MiB
			// so convert it to the equivalent page count.
			count += platformMiBToPageCount(mem.Platform, pagesize)
		}
	}

	return count, nil
}

// memoryCountByFunction counts the number of pages of a particular size that a
// specific function is using on a processor node/socket.
func memoryCountByFunction(data []memory.Memory, node int, function string, pagesize starlingxv1.PageSize) (int, error) {
	switch function {
	case memory.MemoryFunctionVSwitch:
		return vswitchCountMemoryByFunction(data, node, pagesize)
	case memory.MemoryFunctionVM:
		return vmCountMemoryByFunction(data, node, pagesize)
	case memory.MemoryFunctionPlatform:
		return platformCountMemoryByFunction(data, node, pagesize)
	}

	msg := fmt.Sprintf("unsupported memory function: %s", function)
	return 0, utils.NewUserDataError(msg)
}

// memoryUpdateRequired is a utility function which determines whether an
// update is required to adjust the memory configuration
func memoryUpdateRequired(f starlingxv1.MemoryFunctionInfo, count int) (opts memory.MemoryOpts, result bool) {
	// Platform memory is tracked by the system API in whole MiB, so the
	// current count (derived from the stored MiB value) can only ever match a
	// whole-MiB desired count.  Align the desired count the same way before
	// comparing, otherwise a page count that is not a whole number of MiB
	// would request an update on every reconcile and never converge.
	alignPlatformMemory(&f)

	if count != f.PageCount {
		opts.Function = f.Function

		switch f.Function {
		case memory.MemoryFunctionVM:
			if f.PageSize == string(starlingxv1.PageSize1G) {
				opts.VMHugepages1G = &f.PageCount

			} else if f.PageSize == string(starlingxv1.PageSize2M) {
				opts.VMHugepages2M = &f.PageCount
			}

		case memory.MemoryFunctionVSwitch:
			if f.PageSize == string(starlingxv1.PageSize1G) {
				opts.VSwitchHugepages = &f.PageCount
				hpSize := 1024
				opts.VSwitchHugepageSize = &hpSize

			} else if f.PageSize == string(starlingxv1.PageSize2M) {
				opts.VSwitchHugepages = &f.PageCount
				hpSize := 2
				opts.VSwitchHugepageSize = &hpSize
			}

		case memory.MemoryFunctionPlatform:
			size := platformPageCountToMiB(f.PageCount, starlingxv1.PageSize(f.PageSize))
			opts.Platform = &size
		}

		result = true
	}

	return opts, result
}

// ReconcileMemory is responsible for reconciling the Memory configuration of a
// host resource.
func (r *HostReconciler) ReconcileMemory(client *gophercloud.ServiceClient, instance *starlingxv1.Host, profile *starlingxv1.HostProfileSpec, host *v1info.HostInfo) error {
	if len(profile.Memory) == 0 || !common.IsReconcilerEnabled(common.Memory) {
		return nil
	}

	// Retrieve the current CPU configuration
	objects, err := memory.ListMemory(client, host.ID)
	if err != nil {
		err = perrors.Wrapf(err, "failed to list memory for: %s", host.ID)
		return err
	}

	memories, err := getMemoryOpts(profile, objects, host)
	if err != nil {
		return err
	}

	for _, m := range memories {
		logHost.Info("updating memory configuration", "opts", m.Opts)
		if _, err := memory.Update(client, m.Mem.ID, m.Opts).Extract(); err != nil {
			err = perrors.Wrapf(err, "failed to update memory: %s, %s",
				host.ID, utils.FormatStruct(m.Opts))
			return err
		}
	}

	if len(memories) > 0 {
		r.NormalEvent(instance, utils.ResourceUpdated,
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

type memoryWithOpts struct {
	Mem  *memory.Memory
	Opts memory.MemoryOpts
}

// getMemoryOpts is a utility function to get the memories settings based on the host's
// personality or subfunction. If the host is a worker, it will allow and add all the
// kinds of memory configuartion, but if the host is a controller, it will allow
// only plataform memory configuration.
func getMemoryOpts(
	profile *starlingxv1.HostProfileSpec,
	memories []memory.Memory,
	host *v1info.HostInfo,
) (memWithOptsList []memoryWithOpts, err error) {

	isWorker := profile.HasWorkerSubFunction()
	for _, nodeInfo := range profile.Memory {
		// For each NUMA node configuration
		mem := host.FindMemory(nodeInfo.Node)
		if mem == nil {
			msg := fmt.Sprintf("failed to find memory resource for node %d", nodeInfo.Node)
			return memWithOptsList, starlingxv1.NewMissingSystemResource(msg)
		}

		for _, f := range nodeInfo.Functions {
			if !isWorker && f.Function != memory.MemoryFunctionPlatform {
				msg := fmt.Sprintf("Ignoring memory of function %s as it's only allowed for worker nodes", f.Function)
				logHost.Info(msg)
				continue
			}

			// For each function within a NUMA node configuration
			pageSize := starlingxv1.PageSize(f.PageSize)
			count, err := memoryCountByFunction(memories, nodeInfo.Node, f.Function, pageSize)
			if err != nil {
				return memWithOptsList, err
			}

			if opts, ok := memoryUpdateRequired(f, count); ok {
				memWithOptsList = append(memWithOptsList, memoryWithOpts{Mem: mem, Opts: opts})
			}
		}
	}

	return memWithOptsList, err
}

// NormalizePlatformMemoryForComparison rounds the platform 4K page counts in
// the profile down to whole MiB, the granularity the system API can store.
// Without this, a desired page count that is not a whole number of MiB never
// matches the value read back from the system, so the generic profile
// comparison reports a false delta and the host never reaches in-sync.
// This mirrors the MiB round-trip done in memoryUpdateRequired.
func NormalizePlatformMemoryForComparison(profile *starlingxv1.HostProfileSpec) {
	if profile == nil {
		return
	}
	for n := range profile.Memory {
		for i := range profile.Memory[n].Functions {
			alignPlatformMemory(&profile.Memory[n].Functions[i])
		}
	}
}
