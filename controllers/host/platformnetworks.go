/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2024 Wind River Systems, Inc. */

package host

import (
	"context"
	"github.com/gophercloud/gophercloud"
	perrors "github.com/pkg/errors"
	starlingxv1 "github.com/wind-river/cloud-platform-deployment-manager/api/v1"
	utils "github.com/wind-river/cloud-platform-deployment-manager/common"
	v1info "github.com/wind-river/cloud-platform-deployment-manager/platform"
	"k8s.io/apimachinery/pkg/types"
	kubeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// TODO(sriram-gn): All platform network reconciliation workflows will be implemented here.
// Currently we are just setting status of Reconciled / InSync of every platform network
// and address pool instances to true just to acknowledge these objects are accepted.
func (r *HostReconciler) ReconcilePlatformNetworks(client *gophercloud.ServiceClient, instance *starlingxv1.Host, profile *starlingxv1.HostProfileSpec, host *v1info.HostInfo) error {
	if !utils.IsReconcilerEnabled(utils.HostPlatformNetwork) {
		return nil
	}

	addrpools := &starlingxv1.AddressPoolList{}
	platform_networks := &starlingxv1.PlatformNetworkList{}
	opts := kubeclient.ListOptions{}
	opts.Namespace = instance.Namespace
	err := r.List(context.TODO(), addrpools, &opts)
	if err != nil {
		err = perrors.Wrap(err, "failed to list address pools")
		return err
	}

	err = r.List(context.TODO(), platform_networks, &opts)
	if err != nil {
		err = perrors.Wrap(err, "failed to list platform networks")
		return err
	}

	// TODO(sriram-gn): Remove this after implementing actual reconciliation workflows.
	for _, addrpool := range addrpools.Items {
		addrpool_instance := &starlingxv1.AddressPool{}
		addrpool_namespace := types.NamespacedName{Namespace: addrpool.ObjectMeta.Namespace, Name: addrpool.ObjectMeta.Name}
		err := r.Client.Get(context.TODO(), addrpool_namespace, addrpool_instance)
		if err != nil {
			logHost.Error(err, "Failed to get addrpool resource from namespace")
		}
		addrpool_instance.Status.InSync = true
		addrpool_instance.Status.Reconciled = true
		err = r.Client.Status().Update(context.TODO(), addrpool_instance)
		if err != nil {
			logHost.Error(err, "Failed to update addrpool status")
			return err
		}
	}

	// TODO(sriram-gn): Remove this after implementing actual reconciliation workflows.
	for _, platform_network := range platform_networks.Items {
		platform_network_instance := &starlingxv1.PlatformNetwork{}
		platform_network_namespace := types.NamespacedName{Namespace: platform_network.ObjectMeta.Namespace, Name: platform_network.ObjectMeta.Name}
		err := r.Client.Get(context.TODO(), platform_network_namespace, platform_network_instance)
		if err != nil {
			logHost.Error(err, "Failed to get platform network resource from namespace")
		}
		platform_network_instance.Status.InSync = true
		platform_network_instance.Status.Reconciled = true
		err = r.Client.Status().Update(context.TODO(), platform_network_instance)
		if err != nil {
			logHost.Error(err, "Failed to update platform_network_instance status")
			return err
		}
	}

	return nil
}
