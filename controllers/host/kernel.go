/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2023 Wind River Systems, Inc. */

package host

import (
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/kernel"
	perrors "github.com/pkg/errors"
	starlingxv1 "github.com/wind-river/cloud-platform-deployment-manager/api/v1"
	"github.com/wind-river/cloud-platform-deployment-manager/common"
	utils "github.com/wind-river/cloud-platform-deployment-manager/controllers/common"
	v1info "github.com/wind-river/cloud-platform-deployment-manager/platform"
)

// GetKernelUpdateOpts is a utility function which determines whether an update
// is required for the kernel configuration
// if an update is required it will return the kernel update opts
func GetKernelUpdateOpts(kernelResult *kernel.Kernel, profile *starlingxv1.HostProfileSpec) (opts kernel.KernelOpts, updateRequired bool) {
	opts.Kernel = nil
	updateRequired = false

	if *(profile.Kernel) != kernelResult.ProvisionedKernel {
		updateRequired = true
		opts.Kernel = profile.Kernel
	}

	return opts, updateRequired
}

// ReconcileKernel is responsible for reconciling the Memory configuration of a
// host resource.
func (r *HostReconciler) ReconcileKernel(client *gophercloud.ServiceClient, instance *starlingxv1.Host, profile *starlingxv1.HostProfileSpec, hostinfo *v1info.HostInfo) error {
	updated := false

	if profile.Kernel == nil || !common.IsReconcilerEnabled(common.Kernel) {
		return nil
	}

	// Retrieve the current Kernel configuration
	kernelResult, err := kernel.Get(client, hostinfo.ID).Extract()
	if err != nil {
		err = perrors.Wrapf(err, "failed to get kernel for host: %s", hostinfo.ID)
		return err
	}

	opts, updateRequired := GetKernelUpdateOpts(kernelResult, profile)
	if updateRequired {
		logHost.Info("updating kernel configuration", "opts", opts)

		_, err := kernel.Update(client, hostinfo.ID, opts).Extract()
		if err != nil {
			err = perrors.Wrapf(err, "failed to update kernel: %s, %s",
				hostinfo.ID, utils.FormatStruct(opts))
			return err
		}

		updated = true
	}

	if updated {
		r.NormalEvent(instance, utils.ResourceUpdated, "kernel updated")

		kernelResult, err := kernel.Get(client, hostinfo.ID).Extract()
		if err != nil {
			err = perrors.Wrap(err, "failed to refresh host kernel")
			return err
		}

		// update the hostinfo 'cache'
		hostinfo.Kernel = *kernelResult
	}

	return nil
}
