/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019-2023 Wind River Systems, Inc. */

package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/go-logr/logr"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/addresspools"
	hosts "github.com/gophercloud/gophercloud/starlingx/inventory/v1/hosts"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/networks"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/oamNetworks"
	system "github.com/gophercloud/gophercloud/starlingx/inventory/v1/system"
	perrors "github.com/pkg/errors"
	starlingxv1 "github.com/wind-river/cloud-platform-deployment-manager/api/v1"
	utils "github.com/wind-river/cloud-platform-deployment-manager/common"
	"github.com/wind-river/cloud-platform-deployment-manager/controllers/common"
	cloudManager "github.com/wind-river/cloud-platform-deployment-manager/controllers/manager"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var logPlatformNetwork = log.Log.WithName("controller").WithName("platformnetwork")

const PlatformNetworkControllerName = "platformnetwork-controller"

const PlatformNetworkFinalizerName = "platformnetwork.finalizers.windriver.com"

const (
	// Defines commonly used network allocation type values.
	AllocationTypeDynamic = "dynamic"
)

var _ reconcile.Reconciler = &PlatformNetworkReconciler{}

// PlatformNetworkReconciler reconciles a PlatformNetwork object
type PlatformNetworkReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
	cloudManager.CloudManager
	common.ReconcilerErrorHandler
	common.ReconcilerEventLogger
}

// makeRangeArray converts an array of range structs to an array of arrays where
// the inner array contains two elements.  The first element is the range start
// address and the second element is the range end address.  This is to align
// with the system API formatting which represents a pair as an array of two
// elements.
func makeRangeArray(ranges []starlingxv1.AllocationRange) [][]string {
	result := make([][]string, len(ranges))
	for index, r := range ranges {
		result[index] = []string{r.Start, r.End}
	}

	return result
}

// compareRangeArrays compares two range arrays and returns true if they are
// equal.
func compareRangeArrays(x, y [][]string) bool {
	if len(x) != len(y) {
		return false
	}

	count := 0
	for _, o := range x {
		for _, i := range y {
			if strings.EqualFold(o[0], i[0]) && strings.EqualFold(o[1], i[1]) {
				count++
			}
		}
	}

	return len(x) == count
}

// oamUpdateRequired determines whether a oam network resource must
// be updated to align with the stored value.  Only the updated fields are
// include in the request options to minimum churn and to ease debugging.
func oamUpdateRequired(instance *starlingxv1.PlatformNetwork, p *oamNetworks.OAMNetwork, r *PlatformNetworkReconciler) (opts oamNetworks.OAMNetworkOpts, result bool) {
	var delta strings.Builder

	spec := instance.Spec
	instance_subnet := fmt.Sprintf("%s/%d", spec.Subnet, spec.Prefix)
	if instance_subnet != p.OAMSubnet {
		opts.OAMSubnet = &instance_subnet
		delta.WriteString(fmt.Sprintf("\t+Subnet: %s\n", *opts.OAMSubnet))
		result = true
	}

	if instance.Spec.Type != networks.NetworkTypeOther {
		// TODO(alegacy): There is a sysinv bug in how the gateway address
		//  gets registered in the database.  It doesn't have a "name" and
		//  so causes an exception when a related route is added.
		if spec.Gateway != nil && (p.OAMGatewayIP == nil || !strings.EqualFold(*spec.Gateway, *p.OAMGatewayIP)) {
			opts.OAMGatewayIP = spec.Gateway
			delta.WriteString(fmt.Sprintf("\t+Gateway: %s\n", *opts.OAMGatewayIP))
			result = true
		}
	}

	if spec.FloatingAddress != "" && spec.FloatingAddress != p.OAMFloatingIP {
		opts.OAMFloatingIP = &spec.FloatingAddress
		delta.WriteString(fmt.Sprintf("\t+Floating Address: %s\n", *opts.OAMFloatingIP))
		result = true
	}

	if spec.Controller0Address != "" && spec.Controller0Address != p.OAMC0IP {
		opts.OAMC0IP = &spec.Controller0Address
		delta.WriteString(fmt.Sprintf("\t+Controller0 Address: %s\n", *opts.OAMC0IP))
		result = true
	}

	if spec.Controller1Address != "" && spec.Controller1Address != p.OAMC1IP {
		opts.OAMC1IP = &spec.Controller1Address
		delta.WriteString(fmt.Sprintf("\t+Controller1 Address: %s\n", *opts.OAMC1IP))
		result = true
	}

	tempRange := [][]string{{p.OAMStartIP, p.OAMEndIP}}
	if len(spec.Allocation.Ranges) > 0 {
		ranges := makeRangeArray(spec.Allocation.Ranges)
		if !compareRangeArrays(ranges, tempRange) {
			opts.OAMStartIP = &ranges[0][0]
			opts.OAMEndIP = &ranges[0][1]
			delta.WriteString(fmt.Sprintf("\t+Start IP: %s\n", *opts.OAMStartIP))
			delta.WriteString(fmt.Sprintf("\t+End IP: %s\n", *opts.OAMEndIP))
			result = true
		}
	}
	deltaString := delta.String()
	if deltaString != "" {
		deltaString = "\n" + strings.TrimSuffix(deltaString, "\n")
		logPlatformNetwork.Info(fmt.Sprintf("delta configuration:%s\n", deltaString))
	}

	instance.Status.Delta = deltaString
	err := r.Client.Status().Update(context.TODO(), instance)
	if err != nil {
		logPlatformNetwork.Info(fmt.Sprintf("failed to update oam status:  %s\n", err))
	}
	return opts, result
}

// poolUpdateRequired determines whether a system address pool resource must
// be updated to align with the stored value.  Only the updated fields are
// include in the request options to minimum churn and to ease debugging.
func poolUpdateRequired(instance *starlingxv1.PlatformNetwork, p *addresspools.AddressPool, r *PlatformNetworkReconciler) (opts addresspools.AddressPoolOpts, result bool) {
	var delta strings.Builder
	// The address pool name for network type mgmt has to be 'management'
	// and cannot be anything else. GetAddrPoolNameByNetworkType ensures
	// pool name is as per the requirement. This is the limitation on sysinv.
	poolName := r.GetAddrPoolNameByNetworkType(instance.Spec.Type, instance.Name)
	if p.Name != poolName {
		opts.Name = &poolName
		delta.WriteString(fmt.Sprintf("\t+Name: %s\n", *opts.Name))
		result = true
	}

	spec := instance.Spec
	if !strings.EqualFold(spec.Subnet, p.Network) {
		opts.Network = &spec.Subnet
		delta.WriteString(fmt.Sprintf("\t+Network: %s\n", *opts.Network))
		result = true
	}

	if spec.Prefix != p.Prefix {
		opts.Prefix = &spec.Prefix
		delta.WriteString(fmt.Sprintf("\t+Prefix: %d\n", *opts.Prefix))
		result = true
	}

	if spec.FloatingAddress != "" && spec.FloatingAddress != p.FloatingAddress {
		opts.FloatingAddress = &spec.FloatingAddress
		delta.WriteString(fmt.Sprintf("\t+Floating Address: %s\n", *opts.FloatingAddress))
		result = true
	}

	if spec.Controller0Address != "" && spec.Controller0Address != p.Controller0Address {
		opts.Controller0Address = &spec.Controller0Address
		delta.WriteString(fmt.Sprintf("\t+Controller0 Address: %s\n", *opts.Controller0Address))
		result = true
	}

	if spec.Controller1Address != "" && spec.Controller1Address != p.Controller1Address {
		opts.Controller1Address = &spec.Controller1Address
		delta.WriteString(fmt.Sprintf("\t+Controller1 Address: %s\n", *opts.Controller1Address))
		result = true
	}

	if instance.Spec.Type != networks.NetworkTypeOther {
		// TODO(alegacy): There is a sysinv bug in how the gateway address
		//  gets registered in the database.  It doesn't have a "name" and
		//  so causes an exception when a related route is added.
		if spec.Gateway != nil && (p.Gateway == nil || !strings.EqualFold(*spec.Gateway, *p.Gateway)) {
			opts.Gateway = spec.Gateway
			delta.WriteString(fmt.Sprintf("\t+Gateway: %s\n", *opts.Gateway))
			result = true
		}
	}

	if spec.Allocation.Order != nil && *spec.Allocation.Order != p.Order {
		opts.Order = spec.Allocation.Order
		delta.WriteString(fmt.Sprintf("\t+Order: %s\n", *opts.Order))
		result = true
	}

	if len(spec.Allocation.Ranges) > 0 {
		ranges := makeRangeArray(spec.Allocation.Ranges)
		if !compareRangeArrays(ranges, p.Ranges) {
			opts.Ranges = &ranges
			delta.WriteString(fmt.Sprintf("\t+Ranges: %s\n", *opts.Ranges))
			result = true
		}
	}
	deltaString := delta.String()
	if deltaString != "" {
		deltaString = "\n" + strings.TrimSuffix(deltaString, "\n")
		logPlatformNetwork.V(2).Info(fmt.Sprintf("delta configuration:%s\n", deltaString))
	}

	instance.Status.Delta = deltaString
	err := r.Client.Status().Update(context.TODO(), instance)
	if err != nil {
		logPlatformNetwork.Info(fmt.Sprintf("failed to update status:  %s\n", err))
	}

	return opts, result
}

// Address pool which will be used for network of mgmt type should use the name 'management'
func (r *PlatformNetworkReconciler) GetAddrPoolNameByNetworkType(network_type string, pool_name string) string {
	if network_type == cloudManager.MgmtNetworkType {
		return cloudManager.MgmtAddrPoolName
	} else {
		return pool_name
	}
}

// ReconcileNew is a method which handles reconciling a new data resource and
// creates the corresponding system resource thru the system API.
func (r *PlatformNetworkReconciler) ReconcileNewAddressPool(client *gophercloud.ServiceClient, instance *starlingxv1.PlatformNetwork) (*addresspools.AddressPool, error) {
	if instance.Status.Reconciled && r.StopAfterInSync() {
		// Do not process any further changes once we have reached a
		// synchronized state unless there is an annotation on the resource.
		if _, present := instance.Annotations[cloudManager.ReconcileAfterInSync]; !present {
			msg := common.NoProvisioningAfterReconciled
			r.ReconcilerEventLogger.NormalEvent(instance, common.ResourceUpdated, msg)
			return nil, common.NewChangeAfterInSync(msg)
		} else {
			logPlatformNetwork.Info(common.ProvisioningAllowedAfterReconciled)
		}
	}

	poolName := r.GetAddrPoolNameByNetworkType(instance.Spec.Type, instance.Name)

	opts := addresspools.AddressPoolOpts{
		Name:    &poolName,
		Network: &instance.Spec.Subnet,
		Prefix:  &instance.Spec.Prefix,
		Order:   instance.Spec.Allocation.Order,
	}

	if instance.Spec.FloatingAddress != "" {
		opts.FloatingAddress = &instance.Spec.FloatingAddress
	}

	if instance.Spec.Controller0Address != "" {
		opts.Controller0Address = &instance.Spec.Controller0Address
	}

	if instance.Spec.Controller1Address != "" {
		opts.Controller1Address = &instance.Spec.Controller1Address
	}

	if instance.Spec.Type != networks.NetworkTypeOther {
		// TODO(alegacy): There is a sysinv bug in how the gateway address
		//  gets registered in the database.  It doesn't have a "name" and
		//  so causes an exception when a related route is added.  So to
		// avoid the issue only set the gateway for all other network types.
		opts.Gateway = instance.Spec.Gateway
	}

	if len(instance.Spec.Allocation.Ranges) > 0 {
		ranges := makeRangeArray(instance.Spec.Allocation.Ranges)
		opts.Ranges = &ranges
	}

	logPlatformNetwork.Info("creating address pool", "opts", opts)

	pool, err := addresspools.Create(client, opts).Extract()
	if err != nil {
		err = perrors.Wrapf(err, "failed to create pool: %s", common.FormatStruct(opts))
		return nil, err
	}

	r.ReconcilerEventLogger.NormalEvent(instance, common.ResourceCreated,
		"address pool has been created")

	return pool, nil
}

// ReconcileUpdated is a method which handles reconciling an existing data
// resource and updates the corresponding system resource thru the system API to
// match the desired state of the resource.
func (r *PlatformNetworkReconciler) ReconcileUpdatedOAMNetwork(client *gophercloud.ServiceClient, instance *starlingxv1.PlatformNetwork, oam *oamNetworks.OAMNetwork) error {
	if opts, ok := oamUpdateRequired(instance, oam, r); ok {
		if instance.Status.Reconciled && r.StopAfterInSync() {
			// Do not process any further changes once we have reached a
			// synchronized state unless there is an annotation on the resource.
			if _, present := instance.Annotations[cloudManager.ReconcileAfterInSync]; !present {
				msg := common.NoChangesAfterReconciled
				r.ReconcilerEventLogger.NormalEvent(instance, common.ResourceUpdated, msg)
				return common.NewChangeAfterInSync(msg)
			} else {
				logPlatformNetwork.Info(common.ChangedAllowedAfterReconciled)
			}
		}

		if instance.Status.DeploymentScope != cloudManager.ScopePrincipal {
			r.ReconcilerEventLogger.WarningEvent(instance, common.ResourceDependency,
				"unable to update OAM Network with deploymentScope = bootstrap")
			return nil
		}

		// Update existing oam network
		logPlatformNetwork.Info("updating oam network", "uuid", oam.UUID, "opts", opts)

		result, err := oamNetworks.Update(client, oam.UUID, opts).Extract()
		if err != nil {
			err = perrors.Wrapf(err, "failed to update oam network: %+v", opts)
			return err
		}

		*oam = *result

		r.ReconcilerEventLogger.NormalEvent(instance, common.ResourceUpdated,
			"oam network has been updated")

	}

	return nil
}

func (r *PlatformNetworkReconciler) ReconcileUpdatedAddressPool(client *gophercloud.ServiceClient, instance *starlingxv1.PlatformNetwork, pool *addresspools.AddressPool) error {
	if opts, ok := poolUpdateRequired(instance, pool, r); ok {
		if instance.Status.Reconciled && r.StopAfterInSync() {
			// Do not process any further changes once we have reached a
			// synchronized state unless there is an annotation on the resource.
			if _, present := instance.Annotations[cloudManager.ReconcileAfterInSync]; !present {
				msg := common.NoChangesAfterReconciled
				r.ReconcilerEventLogger.NormalEvent(instance, common.ResourceUpdated, msg)
				return common.NewChangeAfterInSync(msg)
			} else {
				logPlatformNetwork.Info(common.ChangedAllowedAfterReconciled)
			}
		}

		// Update existing pool
		logPlatformNetwork.Info("updating address pool", "uuid", pool.ID, "opts", opts)

		result, err := addresspools.Update(client, pool.ID, opts).Extract()
		if err != nil {
			err = perrors.Wrapf(err, "failed to update pool: %+v", opts)
			return err
		}

		*pool = *result

		r.ReconcilerEventLogger.NormalEvent(instance, common.ResourceUpdated,
			"address pool has been updated")

	}

	return nil
}

// DeleteAddressPool is a method which deletes a given address pool using
// the address pool API.
func (r *PlatformNetworkReconciler) DeleteAddressPool(client *gophercloud.ServiceClient, instance *starlingxv1.PlatformNetwork, pool *addresspools.AddressPool) error {
	if pool != nil {
		// Unless it was already deleted go ahead and attempt to delete it.
		err := addresspools.Delete(client, pool.ID).ExtractErr()
		if err != nil {
			if code, ok := err.(gophercloud.ErrUnexpectedResponseCode); !ok {
				err = perrors.Wrap(err, "failed to delete address pool")
				return err
			} else if code.Actual == 409 {
				logPlatformNetwork.Info("address pool is still in use; deleting local resource anyway")
				// NOTE: there is no way to block the kubernetes delete beyond
				//  delaying it a little while we delete an external resource so
				//  since we can't fail this then log it and allow it to
				//  continue for now.
			} else {
				err = perrors.Wrap(err, "unexpected response code on address pool delete")
				return err
			}
		}

		r.ReconcilerEventLogger.NormalEvent(instance, common.ResourceDeleted, "address pool has been deleted")
	}

	return nil
}

// FindExistingAddressPool attempts to re-use the existing resource referenced
// by the ID value stored in the status or to find another resource with a
// matching name.
func (r *PlatformNetworkReconciler) FindExistingAddressPool(client *gophercloud.ServiceClient, instance *starlingxv1.PlatformNetwork) (pool *addresspools.AddressPool, err error) {
	id := instance.Status.PoolUUID
	if id != nil {
		// This network was previously provisioned.
		pool, err = addresspools.Get(client, *id).Extract()
		if err != nil {
			if _, ok := err.(gophercloud.ErrDefault404); !ok {
				err = perrors.Wrapf(err, "failed to get: %s", *id)
				return nil, err
			}

			// The resource may have been deleted by the system or operator
			// therefore continue and attempt to recreate it.
			logPlatformNetwork.Info("resource no longer exists", "id", *id)
			return nil, nil
		}

	} else {
		// This network needs to be provisioned if it doesn't already exist.
		addr_pools, err := addresspools.ListAddressPools(client)
		if err != nil {
			err = perrors.Wrap(err, "failed to list pools")
			return nil, err
		}

		results, err := networks.ListNetworks(client)
		if err != nil {
			err = perrors.Wrap(err, "failed to list networks")
			return nil, err
		}

		for _, nwk := range results {
			if nwk.Type == instance.Spec.Type {
				for _, p := range addr_pools {
					if p.ID == nwk.PoolUUID {
						pool = &p
						break
					}
				}
				break
			}
		}
	}

	return pool, err
}

// ReconcileAddressPool determines whether the stored address pool instance
// needs to be created or updated in the system.  This is done independently of
// the network resource since at the system level these are two independent
// resources.
func (r *PlatformNetworkReconciler) ReconcileAddressPool(client *gophercloud.ServiceClient, instance *starlingxv1.PlatformNetwork) error {
	var pool *addresspools.AddressPool
	var err error
	pool, err = r.FindExistingAddressPool(client, instance)

	if err != nil {
		return err
	}

	if !instance.DeletionTimestamp.IsZero() {
		if utils.ContainsString(instance.ObjectMeta.Finalizers, PlatformNetworkFinalizerName) {
			return r.DeleteAddressPool(client, instance, pool)
		}

	} else {
		if pool == nil {
			pool, err = r.ReconcileNewAddressPool(client, instance)
		} else if instance.Spec.Type == cloudManager.MgmtNetworkType || instance.Spec.Type == cloudManager.AdminNetworkType {
			// For network types mgmt and admin we cannot modify existing pool,
			// we have to delete and recreate it instead.
			if _, ok := poolUpdateRequired(instance, pool, r); ok {
				err = r.DeleteAddressPool(client, instance, pool)
				if err == nil {
					pool, err = r.ReconcileNewAddressPool(client, instance)
				}
			}
		} else if instance.Spec.Type == cloudManager.OAMNetworkType {
			oamNetworkList, err := oamNetworks.ListNetworks(client)
			if err != nil {
				err = perrors.Wrapf(err, "failed to get oam network")
				return err
			}
			oamNetwork := &oamNetworkList[0]
			err = r.ReconcileUpdatedOAMNetwork(client, instance, oamNetwork)

			if err != nil {
				return err
			}
		} else {
			err = r.ReconcileUpdatedAddressPool(client, instance, pool)
		}

		if err == nil && pool != nil {
			// Update the resource status to link it to the system object.
			if instance.Status.PoolUUID == nil || *instance.Status.PoolUUID != pool.ID {
				instance.Status.PoolUUID = &pool.ID
			}
		}
	}

	return err
}

// plNetworkUpdateRequired determines whether a system network resource must be
// updated to align with the stored value. Only the updated fields are
// include in the request options to minimum churn and to ease debugging.
func plNetworkUpdateRequired(instance *starlingxv1.PlatformNetwork, n *networks.Network) (opts networks.NetworkOpts, result bool) {
	if instance.Name != n.Name {
		opts.Name = &instance.Name
		result = true
	}

	spec := instance.Spec
	if spec.Type != n.Type {
		opts.Type = &spec.Type
		result = true
	}

	dynamic := bool(spec.Allocation.Type == AllocationTypeDynamic)
	if dynamic != n.Dynamic {
		opts.Dynamic = &dynamic
		result = true
	}

	status := instance.Status
	if status.PoolUUID != nil && *status.PoolUUID != n.PoolUUID {
		opts.PoolUUID = status.PoolUUID
		result = true
	}

	return opts, result
}

// networkResourceRequired determines whether the network instance requires both
// an address pool resource and a network resource or whether it requires only
// an address pool resource. Networks that are used purely to instantiate an
// address pool do not require an actual network resource to be created.  In
// the future we will hopefully change the system API to require this or even
// change it so that address pools are not explicitly provisioned, but for now
// we simply manage this detail here to make the UX better.
func networkResourceRequired(instance *starlingxv1.PlatformNetwork) bool {
	switch instance.Spec.Type {
	case networks.NetworkTypeOther:
		return false
	case cloudManager.OAMNetworkType:
		// return false because both network and address pool reconfiguration are
		// handled by OAM API which is handled by ReconcileUpdatedOAMNetwork.
		return false
	default:
		return true
	}
}

// ReconcileNewNetwork is a method which handles reconciling a new data resource
// and creates the corresponding system resource thru the system API.
func (r *PlatformNetworkReconciler) ReconcileNewNetwork(client *gophercloud.ServiceClient, instance *starlingxv1.PlatformNetwork) (*networks.Network, error) {
	if instance.Status.Reconciled && r.StopAfterInSync() {
		// Do not process any further changes once we have reached a
		// synchronized state unless there is an annotation on the resource.
		if _, present := instance.Annotations[cloudManager.ReconcileAfterInSync]; !present {
			msg := common.NoChangesAfterReconciled
			r.ReconcilerEventLogger.NormalEvent(instance, common.ResourceUpdated, msg)
			return nil, common.NewChangeAfterInSync(msg)
		} else {
			logPlatformNetwork.Info(common.ChangedAllowedAfterReconciled)
		}
	}

	dynamic := bool(instance.Spec.Allocation.Type == AllocationTypeDynamic)

	opts := networks.NetworkOpts{
		Name:     &instance.Name,
		Type:     &instance.Spec.Type,
		Dynamic:  &dynamic,
		PoolUUID: instance.Status.PoolUUID,
	}

	logPlatformNetwork.Info("creating platform network", "opts", opts)

	network, err := networks.Create(client, opts).Extract()
	if err != nil {
		err = perrors.Wrapf(err, "failed to create: %s", common.FormatStruct(opts))
		return nil, err
	}

	r.ReconcilerEventLogger.NormalEvent(instance, common.ResourceCreated,
		"platform network has been created")

	return network, nil
}

// ReconcileUpdated is a method which handles reconciling an existing data
// resource and updates the corresponding system resource thru the system API to
// match the desired state of the resource.
func (r *PlatformNetworkReconciler) ReconcileUpdatedNetwork(client *gophercloud.ServiceClient, instance *starlingxv1.PlatformNetwork, network *networks.Network) error {
	if opts, ok := plNetworkUpdateRequired(instance, network); ok {
		if instance.Status.Reconciled && r.StopAfterInSync() {
			// Do not process any further changes once we have reached a
			// synchronized state unless there is an annotation on the resource.
			if _, present := instance.Annotations[cloudManager.ReconcileAfterInSync]; !present {
				msg := common.NoChangesAfterReconciled
				r.ReconcilerEventLogger.NormalEvent(instance, common.ResourceUpdated, msg)
				return common.NewChangeAfterInSync(msg)
			} else {
				logPlatformNetwork.Info(common.ChangedAllowedAfterReconciled)
			}
		}

		logPlatformNetwork.Info("updating platform network", "uuid", network.UUID, "opts", opts)

		result, err := networks.Update(client, network.UUID, opts).Extract()
		if err != nil {
			err = perrors.Wrapf(err, "failed to update: %s, %s",
				network.UUID, common.FormatStruct(opts))
			return err
		}

		*network = *result

		r.ReconcilerEventLogger.NormalEvent(instance, common.ResourceUpdated,
			"platform network has been updated")
	}

	return nil
}

// ReconcileNew is a method which handles reconciling a new data resource and
// creates the corresponding system resource thru the system API.
func (r *PlatformNetworkReconciler) ReconciledDeletedNetwork(client *gophercloud.ServiceClient, instance *starlingxv1.PlatformNetwork, network *networks.Network) error {
	if network != nil {
		// Unless it was already deleted go ahead and attempt to delete it.
		err := networks.Delete(client, network.UUID).ExtractErr()
		if err != nil {
			err = perrors.Wrap(err, "failed to delete network pool")
			return err
		}

		r.ReconcilerEventLogger.NormalEvent(instance, common.ResourceDeleted, "network has been deleted")
	}

	return nil
}

// FindExistingNetwork attempts to re-use the existing resource referenced
// by the ID value stored in the status or to find another resource with a
// matching name.
func (r *PlatformNetworkReconciler) FindExistingNetwork(client *gophercloud.ServiceClient, instance *starlingxv1.PlatformNetwork) (network *networks.Network, err error) {
	id := instance.Status.ID
	if id != nil {
		// This network was previously provisioned.
		network, err = networks.Get(client, *id).Extract()
		if err != nil {
			if _, ok := err.(gophercloud.ErrDefault404); !ok {
				err = perrors.Wrapf(err, "failed to get: %s", *id)
				return nil, err
			}

			// The resource may have been deleted by the system or operator
			// therefore continue and attempt to recreate it.
			logPlatformNetwork.Info("resource no longer exists", "id", *id)
			return nil, nil
		}

	} else {
		// This network needs to be provisioned if it doesn't already exist.
		results, err := networks.ListNetworks(client)
		if err != nil {
			err = perrors.Wrap(err, "failed to list networks")
			return nil, err
		}

		for _, p := range results {
			if p.Name == instance.Name {
				network = &p
				break
			}
		}
	}

	return network, err
}

// ReconcileNetwork determines whether the stored network instance is needs to
// be created or updated in the system.  This is done independently of the
// address pool resource since at the system level these are two independent
// resources.
func (r *PlatformNetworkReconciler) ReconcileNetwork(client *gophercloud.ServiceClient, instance *starlingxv1.PlatformNetwork) error {
	network, err := r.FindExistingNetwork(client, instance)
	if err != nil {
		return err
	}

	if !instance.DeletionTimestamp.IsZero() {
		if utils.ContainsString(instance.ObjectMeta.Finalizers, PlatformNetworkFinalizerName) {
			return r.ReconciledDeletedNetwork(client, instance, network)
		}

	} else {
		if network == nil {
			network, err = r.ReconcileNewNetwork(client, instance)
		} else {
			err = r.ReconcileUpdatedNetwork(client, instance, network)
		}

		if err == nil && network != nil {
			// Update the resource status to link it to the system object.
			if instance.Status.ID == nil || *instance.Status.ID != network.UUID {
				instance.Status.ID = &network.UUID
			}
		}
	}

	return err
}

// statusUpdateRequired is a utility method to determine if the status needs
// to be updated at the API.
func (r *PlatformNetworkReconciler) statusUpdateRequired(instance *starlingxv1.PlatformNetwork, status *starlingxv1.PlatformNetworkStatus) bool {
	return !instance.Status.DeepEqual(status)
}

// Determine if the system type is All-in-one simplex, return the Host object if so.
func (r *PlatformNetworkReconciler) AIOSXHost(client *gophercloud.ServiceClient) (*hosts.Host, bool, error) {
	all_hosts, err := hosts.ListHosts(client)
	if err != nil {
		return nil, false, common.NewSystemDependency("Failed to get host list")
	}

	if len(all_hosts) == 1 {
		for _, host := range all_hosts {
			if host.Personality == cloudManager.PersonalityController {
				default_system, err := system.GetDefaultSystem(client)
				if err != nil {
					return nil, false, common.NewSystemDependency("Failed to get default system")
				}

				if default_system.SystemMode == fmt.Sprintf("%s", cloudManager.SystemModeSimplex) && strings.ToLower(default_system.SystemType) == fmt.Sprintf("%s", cloudManager.SystemTypeAllInOne) {
					return &host, true, nil
				}
			}
		}
	}

	return nil, false, nil
}

func (r *PlatformNetworkReconciler) AddressPoolUpdateRequired(client *gophercloud.ServiceClient, instance *starlingxv1.PlatformNetwork) (bool, error) {
	var pool *addresspools.AddressPool
	var err error
	pool, err = r.FindExistingAddressPool(client, instance)
	if err != nil {
		return false, err
	} else if pool == nil {
		return true, nil
	}

	if _, ok := poolUpdateRequired(instance, pool, r); ok {
		return true, nil
	}
	return false, nil
}

func (r *PlatformNetworkReconciler) NetworkUpdateRequired(client *gophercloud.ServiceClient, instance *starlingxv1.PlatformNetwork) (bool, error) {
	if instance.Spec.Type == cloudManager.OAMNetworkType {
		oamNetworkList, err := oamNetworks.ListNetworks(client)
		if err != nil {
			return false, err
		} else if len(oamNetworkList) == 0 {
			return true, nil
		} else {
			oamNetwork := &oamNetworkList[0]
			if _, ok := oamUpdateRequired(instance, oamNetwork, r); ok {
				return true, nil
			}
		}
	} else {
		network, err := r.FindExistingNetwork(client, instance)
		if err != nil {
			return false, err
		} else if network == nil {
			return true, nil
		}
		if _, ok := plNetworkUpdateRequired(instance, network); ok {
			return true, nil
		}
	}

	return false, nil
}

func (r *PlatformNetworkReconciler) PlatformNetworkUpdateRequired(client *gophercloud.ServiceClient, instance *starlingxv1.PlatformNetwork) (bool, error) {
	pool_update_required, err := r.AddressPoolUpdateRequired(client, instance)
	if err != nil {
		err = common.NewSystemDependency("There was an error retrieving the address pool.")
		return false, err
	}

	nwk_update_required, err := r.NetworkUpdateRequired(client, instance)
	if err != nil {
		err = common.NewSystemDependency("There was an error retrieving the network.")
		return false, err
	}

	return pool_update_required || nwk_update_required, err
}

func (r *PlatformNetworkReconciler) UpdateInsyncStatus(client *gophercloud.ServiceClient, instance *starlingxv1.PlatformNetwork, err error) (bool, error) {
	inSync := err == nil
	if inSync != instance.Status.InSync || inSync != instance.Status.Reconciled {
		r.ReconcilerEventLogger.NormalEvent(instance, common.ResourceUpdated, "synchronization has changed to: %t", inSync)
		instance.Status.InSync = inSync
		instance.Status.Reconciled = inSync
		err2 := r.Client.Status().Update(context.TODO(), instance)
		if err2 != nil {
			logPlatformNetwork.Error(err2, "failed to update platform network status")
			return inSync, err2
		}
	}
	return inSync, nil
}

func (r *PlatformNetworkReconciler) TriggerHostNetworkReconciliation(inSync bool, nwk_type string, host *hosts.Host, strategy string) bool {
	if inSync {
		logPlatformNetwork.Info(fmt.Sprintf("PlatformNetwork reconciliation successful for %s. Triggering host reconciliation.", nwk_type))
		host_strategy := cloudManager.HostStrategyInfo{
			StrategyRequired: strategy,
			Host:             *host,
		}
		_ = r.CloudManager.SendHostReconciliationTrigger(host_strategy)
		return true
	}
	return false
}

// This method will try to reconcile both address pool as well as the network
// and update the instance status accordingly.
func (r *PlatformNetworkReconciler) ReconcileAddressPoolAndNetwork(client *gophercloud.ServiceClient, instance *starlingxv1.PlatformNetwork) (inSync bool, err error) {
	err = r.ReconcileAddressPool(client, instance)
	if err == nil {
		if networkResourceRequired(instance) {
			err = r.ReconcileNetwork(client, instance)
		}
	}

	inSync, err2 := r.UpdateInsyncStatus(client, instance, err)
	if err2 != nil {
		return inSync, err2
	}

	return inSync, err
}

// ReconcileResource interacts with the system API in order to reconcile the
// state of a data network with the state stored in the k8s database.
func (r *PlatformNetworkReconciler) ReconcileResource(client *gophercloud.ServiceClient, instance *starlingxv1.PlatformNetwork) (err error) {
	oldStatus := instance.Status.DeepCopy()

	if instance.DeletionTimestamp.IsZero() {
		update_pn, err := r.PlatformNetworkUpdateRequired(client, instance)
		if err != nil {
			return err
		}
		if instance.Status.DeploymentScope == cloudManager.ScopeBootstrap {
			inSync := !update_pn
			if instance.Spec.Type == cloudManager.MgmtNetworkType || instance.Spec.Type == cloudManager.OAMNetworkType || instance.Spec.Type == cloudManager.AdminNetworkType {
				// Just check current config vs profile for networks of type mgmt, oam and admin
				// Update instance.Status.InSync to false if there is a difference and vice-versa.
				// instance.Status.Reconciled will always be set to true to avoid raising alarms in Day-1
				// for above mentioned network types.
				if instance.Status.InSync != inSync {
					r.ReconcilerEventLogger.NormalEvent(instance, common.ResourceUpdated, "synchronization has changed to: %t", inSync)
				}
				instance.Status.InSync = inSync
				instance.Status.Reconciled = true
				err2 := r.Client.Status().Update(context.TODO(), instance)
				if err2 != nil {
					logPlatformNetwork.Error(err2, "failed to update platform network status")
				}
				return err2
			} else {
				// Try to reconcile rest of the networks and update the status accordingly.
				_, err = r.ReconcileAddressPoolAndNetwork(client, instance)
				if err != nil {
					return err
				}
			}
		} else if instance.Status.DeploymentScope == cloudManager.ScopePrincipal {
			// if update_pn {

			if !(instance.Status.InSync && instance.Status.Reconciled) {
				host, is_aiosx, err := r.AIOSXHost(client)
				if err != nil {
					return err
				}
				if is_aiosx {
					logPlatformNetwork.Info("Proceeding with network reconfiguration on AIO-SX")
					if instance.Spec.Type == cloudManager.MgmtNetworkType {

						if host.AdministrativeState == hosts.AdminLocked && host.OperationalStatus == hosts.OperDisabled {

							inSync, err := r.ReconcileAddressPoolAndNetwork(client, instance)
							if err != nil {
								return err
							}

							// Trigger network reconciliation from host controller due to deletion of
							// interface-network-assignment caused by management address pool deletion.
							nwk_reconcile_triggered := r.TriggerHostNetworkReconciliation(inSync, instance.Name, host, cloudManager.StrategyUnlockRequired)
							if !nwk_reconcile_triggered {
								return common.NewResourceConfigurationDependency("there was an error reconciling network configuration. will try again.")
							}

							return nil

						} else {
							host_strategy := cloudManager.HostStrategyInfo{
								StrategyRequired: cloudManager.StrategyLockRequired,
								Host:             *host,
							}
							_ = r.CloudManager.SendHostStrategyUpdate(host_strategy)
							return common.NewResourceConfigurationDependency("waiting for system to be in locked state before reconfiguring network")
						}
					} else if instance.Spec.Type == cloudManager.OAMNetworkType {

						_, err := r.ReconcileAddressPoolAndNetwork(client, instance)
						if err != nil {
							return err
						}

						return nil

					} else {

						inSync, err := r.ReconcileAddressPoolAndNetwork(client, instance)
						if err != nil {
							return err
						}

						// Trigger network reconciliation from host controller due to deletion of
						// interface-network-assignment caused by admin address pool deletion.
						if instance.Spec.Type == cloudManager.AdminNetworkType {
							nwk_reconcile_triggered := r.TriggerHostNetworkReconciliation(inSync, instance.Name, host, cloudManager.StrategyNotRequired)
							if !nwk_reconcile_triggered {
								return common.NewResourceConfigurationDependency("there was an error reconciling network configuration. will try again.")
							}
						}

						return nil
					}
				} else {

					// handle for non AIO-SX system types
					if instance.Spec.Type == cloudManager.MgmtNetworkType {
						// Just compare current config vs profile, update instance.Status.InSync
						// and instance.Status.Reconciled accordingly.
						inSync := !update_pn
						instance.Status.InSync = inSync
						instance.Status.Reconciled = inSync
						err2 := r.Client.Status().Update(context.TODO(), instance)
						if err2 != nil {
							logPlatformNetwork.Error(err2, "failed to update platform network status")
						}
						return err2
					} else {
						// Try to reconcile rest of the networks and update the status accordingly.
						_, err = r.ReconcileAddressPoolAndNetwork(client, instance)
						if err != nil {
							return err
						}
					}

					return nil
				}
			}

			// }
		}

		if r.statusUpdateRequired(instance, oldStatus) {
			err2 := r.Client.Status().Update(context.TODO(), instance)
			if err2 != nil {
				logPlatformNetwork.Error(err2, "failed to update platform network status")
				return err2
			}
		}

	} else {
		// Reverse the order of operations for deletes since there is a built-in
		// dependency between the two.
		if networkResourceRequired(instance) {
			err = r.ReconcileNetwork(client, instance)
		} else {
			err = nil
		}

		if err == nil {
			err = r.ReconcileAddressPool(client, instance)
		}

		if err == nil {
			// Remove the finalizer so the kubernetes delete operation can
			// continue.
			instance.ObjectMeta.Finalizers = utils.RemoveString(instance.ObjectMeta.Finalizers, PlatformNetworkFinalizerName)
			if err := r.Client.Update(context.Background(), instance); err != nil {
				return err
			}
		}

	}

	return err
}

// StopAfterInSync determines whether the reconciler should continue processing
// change requests after the configuration has been reconciled a first time.
func (r *PlatformNetworkReconciler) StopAfterInSync() bool {
	// If the option is not found or the option was specified in a form other
	// than a bool then assume the safest default value possible.
	return utils.GetReconcilerOptionBool(utils.PlatformNetwork, utils.StopAfterInSync, true)
}

// Obtain deploymentScope value from configuration
// Taking this value from annotation in instacne
// (It seems Client.Get does not update Status value from configuration)
// "bootstrap" if "bootstrap" in configuration or deploymentScope not specified
// "principal" if "principal" in configuration
func (r *PlatformNetworkReconciler) GetScopeConfig(instance *starlingxv1.PlatformNetwork) (scope string, err error) {
	// Set default value for deployment scope
	deploymentScope := cloudManager.ScopeBootstrap
	// Set DeploymentScope from configuration
	annotation := instance.GetObjectMeta().GetAnnotations()
	if annotation != nil {
		config, ok := annotation["kubectl.kubernetes.io/last-applied-configuration"]
		if ok {
			status_config := &starlingxv1.PlatformNetwork{}
			err := json.Unmarshal([]byte(config), &status_config)
			if err == nil {
				if status_config.Status.DeploymentScope != "" {
					lowerCaseScope := strings.ToLower(status_config.Status.DeploymentScope)
					switch lowerCaseScope {
					case cloudManager.ScopeBootstrap:
						deploymentScope = cloudManager.ScopeBootstrap
					case cloudManager.ScopePrincipal:
						deploymentScope = cloudManager.ScopePrincipal
					default:
						err = fmt.Errorf("Unsupported DeploymentScope: %s",
							status_config.Status.DeploymentScope)
						return deploymentScope, err
					}
				}
			} else {
				err = perrors.Wrapf(err, "failed to Unmarshal annotaion last-applied-configuration")
				return deploymentScope, err
			}
		}
	}
	return deploymentScope, nil
}

// Update deploymentScope and ReconcileAfterInSync in instance
// ReconcileAfterInSync value will be:
// "true"  if deploymentScope is "principal" because it is day 2 operation (update configuration)
// "false" if deploymentScope is "bootstrap"
// Then reflect these values to cluster object
func (r *PlatformNetworkReconciler) UpdateConfigStatus(instance *starlingxv1.PlatformNetwork) (err error) {
	err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
		err := r.Client.Get(context.TODO(), types.NamespacedName{
			Name:      instance.Name,
			Namespace: instance.Namespace,
		}, instance)
		if err != nil {
			return err
		}
		deploymentScope, err := r.GetScopeConfig(instance)
		if err != nil {
			return err
		}
		logPlatformNetwork.V(2).Info("deploymentScope in configuration", "deploymentScope", deploymentScope)

		// Put ReconcileAfterInSync values depends on scope
		// "true"  if scope is "principal" because it is day 2 operation (update configuration)
		// "false" if scope is "bootstrap" or None
		afterInSync, ok := instance.Annotations[cloudManager.ReconcileAfterInSync]
		if deploymentScope == cloudManager.ScopePrincipal {
			if !ok || afterInSync != "true" {
				instance.Annotations[cloudManager.ReconcileAfterInSync] = "true"
			}
		} else {
			if ok && afterInSync == "true" {
				delete(instance.Annotations, cloudManager.ReconcileAfterInSync)
			}
		}
		return r.Client.Update(context.TODO(), instance)
	})
	if err != nil {
		err = perrors.Wrapf(err, "failed to update profile annotation ReconcileAfterInSync")
		return err
	}

	err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
		err := r.Client.Get(context.TODO(), types.NamespacedName{
			Name:      instance.Name,
			Namespace: instance.Namespace,
		}, instance)
		if err != nil {
			return err
		}

		// Update scope status
		deploymentScope, err := r.GetScopeConfig(instance)
		if err != nil {
			return err
		}
		logPlatformNetwork.V(2).Info("deploymentScope in configuration", "deploymentScope", deploymentScope)
		instance.Status.DeploymentScope = deploymentScope

		// Set default value for StrategyRequired
		if instance.Status.StrategyRequired == "" {
			instance.Status.StrategyRequired = cloudManager.StrategyNotRequired
		}

		// Check if the configuration is updated
		if instance.Status.ObservedGeneration != instance.ObjectMeta.Generation {
			if instance.Status.ObservedGeneration == 0 &&
				instance.Status.Reconciled {
				// Case: DM upgrade in reconciled node
				instance.Status.ConfigurationUpdated = false
			} else {
				// Case: Fresh install or Day-2 operation
				instance.Status.ConfigurationUpdated = true
				if instance.Status.DeploymentScope == cloudManager.ScopePrincipal {
					instance.Status.Reconciled = false
					// Update strategy required status for strategy monitor
					r.CloudManager.UpdateConfigVersion()
				}
			}
			instance.Status.ObservedGeneration = instance.ObjectMeta.Generation
			// Reset strategy when new configuration is applied
			instance.Status.StrategyRequired = cloudManager.StrategyNotRequired
		}

		return r.Client.Status().Update(context.TODO(), instance)
	})

	if err != nil {
		err = perrors.Wrapf(err, "failed to update status: %s",
			common.FormatStruct(instance.Status))
		return err
	}

	return nil
}

// Reconcile reads that state of the cluster for a PlatformNetwork object and makes changes based on the state read
// +kubebuilder:rbac:groups=starlingx.windriver.com,resources=platformnetworks,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=starlingx.windriver.com,resources=platformnetworks/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=starlingx.windriver.com,resources=platformnetworks/finalizers,verbs=update
func (r *PlatformNetworkReconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	// To reduce the repitition of adding the resource name to every log we
	// replace the logger with one that includes the resource name and then
	// restore it at the end of the reconcile function.

	savedLog := logPlatformNetwork
	logPlatformNetwork = logPlatformNetwork.WithName(request.NamespacedName.String())
	defer func() { logPlatformNetwork = savedLog }()

	// Fetch the DataNetwork instance
	instance := &starlingxv1.PlatformNetwork{}
	err := r.Client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Object not found, return.  Created objects are automatically
			// garbage collected. For additional cleanup logic use finalizers.
			return reconcile.Result{}, nil
		}

		logPlatformNetwork.Error(err, "unable to read object: %v", request)
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	// Update scope from configuration
	logPlatformNetwork.V(2).Info("before UpdateConfigStatus", "instance", instance)
	err = r.UpdateConfigStatus(instance)
	if err != nil {
		logPlatformNetwork.Error(err, "unable to update scope")
		return reconcile.Result{}, err
	}
	logPlatformNetwork.V(2).Info("after UpdateConfigStatus", "instance", instance)

	if instance.DeletionTimestamp.IsZero() {
		// Ensure that the object has a finalizer setup as a pre-delete hook so
		// that we can delete any system resources that we previously added.
		if !utils.ContainsString(instance.ObjectMeta.Finalizers, PlatformNetworkFinalizerName) {
			instance.ObjectMeta.Finalizers = append(instance.ObjectMeta.Finalizers, PlatformNetworkFinalizerName)
			if err := r.Client.Update(context.Background(), instance); err != nil {
				return reconcile.Result{}, err
			}

			// Might as well return immediately as the update is going to cause
			// another reconcile event for this resource and we don't want to
			// access the system API more than necessary.
			return reconcile.Result{}, nil
		}
	}

	if !utils.IsReconcilerEnabled(utils.PlatformNetwork) {
		return reconcile.Result{}, nil
	}

	platformClient := r.CloudManager.GetPlatformClient(request.Namespace)
	if platformClient == nil {
		// The client has not been authenticated by the system controller so
		// wait.
		r.ReconcilerEventLogger.WarningEvent(instance, common.ResourceDependency,
			"waiting for platform client creation")
		return common.RetryMissingClient, nil
	}

	if !r.CloudManager.GetSystemReady(request.Namespace) {
		r.ReconcilerEventLogger.WarningEvent(instance, common.ResourceDependency,
			"waiting for system reconciliation")
		return common.RetrySystemNotReady, nil
	}

	err = r.ReconcileResource(platformClient, instance)
	if err != nil {
		return r.ReconcilerErrorHandler.HandleReconcilerError(request, err)
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *PlatformNetworkReconciler) SetupWithManager(mgr ctrl.Manager) error {
	tMgr := cloudManager.GetInstance(mgr)
	r.Client = mgr.GetClient()
	r.Scheme = mgr.GetScheme()
	r.CloudManager = tMgr
	r.ReconcilerErrorHandler = &common.ErrorHandler{
		CloudManager: tMgr,
		Logger:       logPlatformNetwork}
	r.ReconcilerEventLogger = &common.EventLogger{
		EventRecorder: mgr.GetEventRecorderFor(PlatformNetworkControllerName),
		Logger:        logPlatformNetwork}
	return ctrl.NewControllerManagedBy(mgr).
		For(&starlingxv1.PlatformNetwork{}).
		Complete(r)
}
