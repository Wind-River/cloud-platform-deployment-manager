/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2024 Wind River Systems, Inc. */

package host

import (
	"context"
	"fmt"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/addresspools"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/networkAddressPools"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/networks"
	perrors "github.com/pkg/errors"
	starlingxv1 "github.com/wind-river/cloud-platform-deployment-manager/api/v1"
	utils "github.com/wind-river/cloud-platform-deployment-manager/common"
	"github.com/wind-river/cloud-platform-deployment-manager/controllers/common"
	cloudManager "github.com/wind-river/cloud-platform-deployment-manager/controllers/manager"
	v1info "github.com/wind-river/cloud-platform-deployment-manager/platform"
	"strings"
)

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

// GetSystemAddrPool returns gophercloud AddressPool object based on AddressPool spec.
func GetSystemAddrPool(client *gophercloud.ServiceClient, addrpool_instance *starlingxv1.AddressPool) (*addresspools.AddressPool, error) {
	var found_addrpool *addresspools.AddressPool
	addrpool_list, err := addresspools.ListAddressPools(client)
	if err != nil {
		logHost.Error(err, "failed to fetch addresspools from system")
		return nil, err
	}
	// Preferably fetch the addresspool using UUID.
	if addrpool_instance.Status.ID != nil {
		found_addrpool = utils.GetSystemAddrPoolByUUID(addrpool_list, *addrpool_instance.Status.ID)
		if found_addrpool != nil {
			return found_addrpool, nil
		}
	}

	found_addrpool = utils.GetSystemAddrPoolByName(addrpool_list, addrpool_instance.Name)

	return found_addrpool, nil
}

// GetSystemNetwork returns gophercloud Network object based on PlatformNetwork spec.
func GetSystemNetwork(client *gophercloud.ServiceClient, network_instance *starlingxv1.PlatformNetwork) (*networks.Network, error) {
	var found_network *networks.Network
	network_list, err := networks.ListNetworks(client)
	if err != nil {
		logHost.Error(err, "failed to fetch networks from system")
		return nil, err
	}

	// Preferably fetch the network using UUID.
	if network_instance.Status.ID != nil {
		found_network = utils.GetSystemNetworkByUUID(network_list, *network_instance.Status.ID)
		if found_network != nil {
			return found_network, nil
		}
	}

	found_network = utils.GetSystemNetworkByName(network_list, network_instance.Name)

	return found_network, nil
}

// ValidateAddressPool validates the addresspool spec specific to the network it will be associated with.
// This is different from validations done in addresspool webhook which is more primitive in nature.
// Result of this validation determines if at all reconciliation request has to be requeued.
func (r *HostReconciler) ValidateAddressPool(
	network_instance *starlingxv1.PlatformNetwork,
	addrpool_instance *starlingxv1.AddressPool,
	system_info *cloudManager.SystemInfo) bool {

	spec := addrpool_instance.Spec
	if network_instance.Spec.Type == cloudManager.OAMNetworkType {
		if system_info.SystemType == cloudManager.SystemTypeAllInOne &&
			system_info.SystemMode == cloudManager.SystemModeSimplex {

			if spec.FloatingAddress == nil ||
				spec.Gateway == nil {

				msg := "The 'floatingAddress' and 'gateway' are mandatory parameters for oam address pool in AIO-SX."
				logHost.Info(msg)
				return false
			}
		} else {
			// Multinode system
			if spec.FloatingAddress == nil ||
				spec.Gateway == nil ||
				spec.Controller0Address == nil ||
				spec.Controller1Address == nil {

				msg := fmt.Sprintf(
					"The %s are mandatory parameters for oam address pool in multinode setup.",
					"'floatingAddress', 'gateway', 'controller0Address' and 'controller1Address'")
				logHost.Info(msg)
				return false
			}
		}
	} else if network_instance.Spec.Type == cloudManager.MgmtNetworkType ||
		network_instance.Spec.Type == cloudManager.ClusterHostNetworkType ||
		network_instance.Spec.Type == cloudManager.PXEBootNetworkType {

		if spec.FloatingAddress == nil ||
			spec.Controller0Address == nil ||
			spec.Controller1Address == nil {

			msg := fmt.Sprintf(
				"The %s are mandatory parameters for %s address pools.",
				"'floatingAddress', 'controller0Address' and 'controller1Address'",
				"management, cluster-host and pxeboot")
			logHost.Info(msg)
			return false
		}

		if network_instance.Spec.Type == cloudManager.PXEBootNetworkType && utils.IsIPv6(addrpool_instance.Spec.Subnet) {
			log_msg := fmt.Sprintf(
				"Network of type pxeboot only supports pool of family IPv4. AddressPool '%s' will not be reconciled.",
				addrpool_instance.Name)
			logHost.Info(log_msg)
			return false
		}
	}

	return true
}

// IsNetworkUpdateRequired determines if platform network update is required
// by comparing it with applied PlatformNetwork spec.
func (r *HostReconciler) IsNetworkUpdateRequired(network_instance *starlingxv1.PlatformNetwork, current_network *networks.Network, primary_address_pool *addresspools.AddressPool) (opts networks.NetworkOpts, result bool, uuid string) {
	var delta strings.Builder

	spec := network_instance.Spec

	if current_network == nil || (network_instance.Name != current_network.Name) {
		opts.Name = &network_instance.Name
		delta.WriteString(fmt.Sprintf("\t+Name: %s\n", *opts.Name))
		result = true
	}

	if current_network == nil || (spec.Type != current_network.Type) {
		opts.Type = &spec.Type
		delta.WriteString(fmt.Sprintf("\t+Type: %s\n", *opts.Type))
		result = true
	}

	if current_network == nil || (spec.Dynamic != current_network.Dynamic) {
		opts.Dynamic = &spec.Dynamic
		delta.WriteString(fmt.Sprintf("\t+Dynamic: %v\n", *opts.Dynamic))
		result = true
	}

	if current_network == nil {
		// This is required while creating the network only and
		// internally managed by the system during reconfiguration.
		opts.PoolUUID = &primary_address_pool.ID
		delta.WriteString(fmt.Sprintf("\t+PoolUUID: %v\n", *opts.PoolUUID))
	} else {
		uuid = current_network.UUID
	}

	deltaString := delta.String()
	if deltaString != "" {
		deltaString = "\n" + strings.TrimSuffix(deltaString, "\n")
		logHost.V(2).Info(fmt.Sprintf("delta configuration:%s\n", deltaString))
	}

	network_instance.Status.Delta = deltaString
	err := r.Client.Status().Update(context.TODO(), network_instance)
	if err != nil {
		logHost.Error(err, fmt.Sprintf("failed to update '%s' platform network delta", network_instance.Name))
	}

	return opts, result, uuid
}

// IsAddrPoolUpdateRequired determines if addresspool update is required
// by comparing it with applied AddressPool spec.
func (r *HostReconciler) IsAddrPoolUpdateRequired(network_instance *starlingxv1.PlatformNetwork, addrpool_instance *starlingxv1.AddressPool, current_addrpool *addresspools.AddressPool) (opts addresspools.AddressPoolOpts, result bool, uuid string) {
	var delta strings.Builder

	if current_addrpool == nil || (addrpool_instance.Name != current_addrpool.Name) {
		opts.Name = &addrpool_instance.Name
		delta.WriteString(fmt.Sprintf("\t+Name: %s\n", *opts.Name))
		result = true
	}

	spec := addrpool_instance.Spec

	if current_addrpool == nil || !utils.IsIPAddressSame(spec.Subnet, current_addrpool.Network) {
		opts.Network = &spec.Subnet
		delta.WriteString(fmt.Sprintf("\t+Network: %s\n", *opts.Network))
		result = true
	}

	if current_addrpool == nil || spec.Prefix != current_addrpool.Prefix {
		opts.Prefix = &spec.Prefix
		delta.WriteString(fmt.Sprintf("\t+Prefix: %d\n", *opts.Prefix))
		result = true
	}

	if (current_addrpool == nil && spec.FloatingAddress != nil) ||
		(spec.FloatingAddress != nil && !utils.IsIPAddressSame(*spec.FloatingAddress, current_addrpool.FloatingAddress)) {
		opts.FloatingAddress = spec.FloatingAddress
		delta.WriteString(fmt.Sprintf("\t+Floating Address: %s\n", *opts.FloatingAddress))
		result = true
	} else if spec.FloatingAddress == nil && current_addrpool != nil && current_addrpool.FloatingAddress != "" {
		opts.FloatingAddress = spec.FloatingAddress
		delta.WriteString(fmt.Sprintf("\t-Floating Address: %s\n", current_addrpool.FloatingAddress))
		result = true
	}

	if (current_addrpool == nil && spec.Controller0Address != nil) ||
		(spec.Controller0Address != nil && !utils.IsIPAddressSame(*spec.Controller0Address, current_addrpool.Controller0Address)) {
		opts.Controller0Address = spec.Controller0Address
		delta.WriteString(fmt.Sprintf("\t+Controller0 Address: %s\n", *opts.Controller0Address))
		result = true
	} else if spec.Controller0Address == nil && current_addrpool != nil && current_addrpool.Controller0Address != "" {
		opts.Controller0Address = spec.Controller0Address
		delta.WriteString(fmt.Sprintf("\t-Controller0 Address: %s\n", current_addrpool.Controller0Address))
		result = true
	}

	if (current_addrpool == nil && spec.Controller1Address != nil) ||
		(spec.Controller1Address != nil && !utils.IsIPAddressSame(*spec.Controller1Address, current_addrpool.Controller1Address)) {
		opts.Controller1Address = spec.Controller1Address
		delta.WriteString(fmt.Sprintf("\t+Controller1 Address: %s\n", *opts.Controller1Address))
		result = true
	} else if spec.Controller1Address == nil && current_addrpool != nil && current_addrpool.Controller1Address != "" {
		opts.Controller1Address = spec.Controller1Address
		delta.WriteString(fmt.Sprintf("\t-Controller1 Address: %s\n", current_addrpool.Controller1Address))
		result = true
	}

	if network_instance.Spec.Type != networks.NetworkTypeOther {
		// TODO(alegacy): There is a sysinv bug in how the gateway address
		//  gets registered in the database.  It doesn't have a "name" and
		//  so causes an exception when a related route is added.
		if (current_addrpool == nil && spec.Gateway != nil) ||
			(spec.Gateway != nil && current_addrpool.Gateway == nil) ||
			(spec.Gateway != nil && !utils.IsIPAddressSame(*spec.Gateway, *current_addrpool.Gateway)) {
			opts.Gateway = spec.Gateway
			delta.WriteString(fmt.Sprintf("\t+Gateway: %s\n", *opts.Gateway))
			result = true
		} else if spec.Gateway == nil && current_addrpool != nil && current_addrpool.Gateway != nil {
			opts.Gateway = spec.Gateway
			delta.WriteString(fmt.Sprintf("\t-Gateway Address: %s\n", *current_addrpool.Gateway))
			result = true
		}
	}

	if (current_addrpool == nil && spec.Allocation.Order != nil) ||
		(current_addrpool != nil && spec.Allocation.Order != nil && *spec.Allocation.Order != current_addrpool.Order) {
		opts.Order = spec.Allocation.Order
		delta.WriteString(fmt.Sprintf("\t+Order: %s\n", *opts.Order))
		result = true
	}

	if len(spec.Allocation.Ranges) > 0 {
		ranges := makeRangeArray(spec.Allocation.Ranges)
		if current_addrpool == nil || !compareRangeArrays(ranges, current_addrpool.Ranges) {
			opts.Ranges = &ranges
			delta.WriteString(fmt.Sprintf("\t+Ranges: %s\n", *opts.Ranges))
			result = true
		}
	}

	if current_addrpool != nil {
		uuid = current_addrpool.ID
	}

	deltaString := delta.String()
	if deltaString != "" {
		deltaString = "\n" + strings.TrimSuffix(deltaString, "\n")
		logHost.V(2).Info(fmt.Sprintf("delta configuration:%s\n", deltaString))
	}

	addrpool_instance.Status.Delta = deltaString
	err := r.Client.Status().Update(context.TODO(), addrpool_instance)
	if err != nil {
		logHost.Error(err, fmt.Sprintf("failed to update '%s' addresspool delta", addrpool_instance.Name))
	}

	return opts, result, uuid
}

// ReconcileAddrPoolResource function reconciles address pool as per applied
// AddressPool spec.
func (r *HostReconciler) ReconcileAddrPoolResource(client *gophercloud.ServiceClient, network_instance *starlingxv1.PlatformNetwork, addrpool_instance *starlingxv1.AddressPool, system_info *cloudManager.SystemInfo) (error, *bool, *bool) {

	system_addrpool, err := GetSystemAddrPool(client, addrpool_instance)
	if err != nil {
		return err, nil, nil
	}

	r.UpdateAddrPoolUUID(addrpool_instance, system_addrpool)

	opts, update_required, uuid := r.IsAddrPoolUpdateRequired(network_instance, addrpool_instance, system_addrpool)

	err, reconcile_expected := r.ReconcilePlatformNetworkExpected(client, network_instance, addrpool_instance, update_required, uuid)
	if err != nil {
		return err, nil, nil
	}

	validation_result := r.ValidateAddressPool(network_instance, addrpool_instance, system_info)

	if reconcile_expected && validation_result && update_required {
		err := r.CreateOrUpdateAddrPools(client, opts, uuid, addrpool_instance)
		if err == nil {
			// Make sure network UUID is synchronized
			system_addrpool, err = GetSystemAddrPool(client, addrpool_instance)
			if err != nil {
				return err, nil, nil
			}
			r.UpdateAddrPoolUUID(addrpool_instance, system_addrpool)
		}
		return err, &reconcile_expected, &validation_result
	} else if !validation_result {
		// These errors are to be corrected by the user.
		// No use requeuing the request until user corrects it.

		// Validation applies for addresspools to be created in the
		// context of network but not for addresspool that already exists
		// as per spec.
		validation_result = validation_result || !update_required
		return nil, &reconcile_expected, &validation_result
	} else if update_required {
		msg := fmt.Sprintf(
			"There is delta between applied spec and system for addresspool '%s'",
			addrpool_instance.Name)
		logHost.Info(msg)
		err := perrors.New(msg)
		return err, &reconcile_expected, &validation_result
	}

	return nil, &reconcile_expected, &validation_result

}

// Synchronize AddressPoolStatus.ID with correct UUID
// of the addresspool as reported by the system.
func (r *HostReconciler) UpdateAddrPoolUUID(addrpool_instance *starlingxv1.AddressPool, system_addrpool *addresspools.AddressPool) {
	update_required := false
	if system_addrpool != nil {
		if addrpool_instance.Status.ID == nil {
			update_required = true
			addrpool_instance.Status.ID = &system_addrpool.ID
		} else {
			// Update stray UUID however this may have been caused.
			if *addrpool_instance.Status.ID != system_addrpool.ID {
				update_required = true
				addrpool_instance.Status.ID = &system_addrpool.ID
			}
		}
	}

	if update_required {
		err := r.Client.Status().Update(context.TODO(), addrpool_instance)
		if err != nil {
			// Logging the error should be enough, failure to update addrpool instance
			// UUID should not block rest of the reconciliation since we
			// always fallback to Name based addrpool instance lookup in case
			// UUID is not updated / not valid.
			logHost.Error(err, fmt.Sprintf("failed to update '%s' addresspool UUID", addrpool_instance.Name))
		}
	}
}

// Synchronize PlatformNetworkStatus.ID with correct UUID
// of the network as reported by the system.
func (r *HostReconciler) UpdateNetworkUUID(network_instance *starlingxv1.PlatformNetwork, system_network *networks.Network) {
	update_required := false
	if system_network != nil {
		if network_instance.Status.ID == nil {
			update_required = true
			network_instance.Status.ID = &system_network.UUID
		} else {
			// Update stray UUID however this may have been caused.
			if *network_instance.Status.ID != system_network.UUID {
				update_required = true
				network_instance.Status.ID = &system_network.UUID
			}
		}
	}

	if update_required {
		err := r.Client.Status().Update(context.TODO(), network_instance)
		if err != nil {
			// Logging the error should be enough, failure to update network
			// UUID should not block rest of the reconciliation since we
			// always fallback to Name based platform network lookup in case
			// UUID is not updated / not valid.
			logHost.Error(err, fmt.Sprintf("failed to update '%s' platform network UUID", network_instance.Name))
		}
	}
}

// IsReconfiguration function helps determine if the spec the user is trying to apply
// will really end up reconfiguring the network or not (ie. modifying current address pools).
func (r *HostReconciler) IsReconfiguration(client *gophercloud.ServiceClient, network_instance *starlingxv1.PlatformNetwork, addrpool_instance *starlingxv1.AddressPool) (error, bool) {
	err, system_network, system_network_addrpools, addrpool_list := r.GetAllNetworkAddressPoolData(
		client, network_instance)
	if err != nil {
		return err, false
	}

	if system_network != nil {
		network_addrpool, associated_addrpool := GetAssociatedNetworkAddrPool(
			system_network,
			addrpool_instance,
			system_network_addrpools,
			addrpool_list)

		if network_addrpool != nil && associated_addrpool != nil {
			// There exists an associated addresspool from same IP family.
			// This is an attempt to reconfigure the platform network.
			return nil, true
		}
	}

	return nil, false
}

// ReconcileOtherPlatformNetworksExpected returns true if the network or the address pool
// has to be newly configured for network types other than oam / admin / mgmt.
func (r *HostReconciler) ReconcileOtherPlatformNetworksExpected(client *gophercloud.ServiceClient,
	network_instance *starlingxv1.PlatformNetwork,
	addrpool_instance *starlingxv1.AddressPool,
	uuid string) (error, bool) {

	if network_instance.Spec.Type == cloudManager.OtherNetworkType {
		// The network type "other" indicates only addresspool is
		// going to be reconciled, hence, it's always reconcilable.
		return nil, true
	}

	if addrpool_instance != nil {
		err, is_reconfig := r.IsReconfiguration(client, network_instance, addrpool_instance)
		if err != nil {
			return err, false
		}
		if !is_reconfig {
			return nil, true
		}
	} else {
		// Since addrpool_instance is nil the ReconcileOtherPlatformNetworksExpected
		// call is implied for networks.
		// If UUID is empty, it means network has to be created, hence return true.
		if uuid == "" {
			return nil, true
		}
	}

	return nil, false
}

// ReconcilePlatformNetworkExpected is a very important function that really controls the reconciliation
// behaviour of network and associated addresspools. Note that parameters 'update_required'
// and 'uuid' refers to address pool update_required and address pool uuid when called from
// ReconcileAddrPoolResource function.
func (r *HostReconciler) ReconcilePlatformNetworkExpected(client *gophercloud.ServiceClient, network_instance *starlingxv1.PlatformNetwork, addrpool_instance *starlingxv1.AddressPool, update_required bool, uuid string) (error, bool) {
	if network_instance.Status.DeploymentScope == cloudManager.ScopeBootstrap {
		switch network_instance.Spec.Type {
		case cloudManager.OAMNetworkType,
			cloudManager.MgmtNetworkType,
			cloudManager.AdminNetworkType:
			// Block both fresh configuration / reconfiguration of networks / addrpools
			// such as oam / mgmt / admin in day-1.
			return nil, false
		default:
			// Allow fresh configuration of networks / addrpools other than
			// oam / mgmt / admin in day-1 but not reconfiguration.
			// Reconciliation of addresspools under "other" network type is
			// always allowed.
			return r.ReconcileOtherPlatformNetworksExpected(client, network_instance, addrpool_instance, uuid)
		}
	} else if network_instance.Status.DeploymentScope == cloudManager.ScopePrincipal {
		switch network_instance.Spec.Type {
		case cloudManager.OAMNetworkType,
			cloudManager.MgmtNetworkType,
			cloudManager.AdminNetworkType:
			// Allow both fresh configuration / reconfiguration of networks / addrpools
			// such as oam / mgmt / admin in day-2.
			return nil, true
		default:
			// Allow fresh configuration of networks / addrpools other than
			// oam / mgmt / admin in day-2 but not reconfiguration.
			// Reconciliation of addresspools under "other" network type is
			// always allowed.
			return r.ReconcileOtherPlatformNetworksExpected(client, network_instance, addrpool_instance, uuid)
		}
	}

	// Unless explicitly specified that reconciliation is allowed
	// for given instances of platform network and address pools
	// return false.
	return nil, false
}

// CreateOrUpdateNetworks creates or updates networks on the system.
func (r *HostReconciler) CreateOrUpdateNetworks(client *gophercloud.ServiceClient, opts networks.NetworkOpts, uuid string, network_instance *starlingxv1.PlatformNetwork) error {
	if uuid == "" {
		_, err := networks.Create(client, opts).Extract()
		if err != nil {
			logHost.Error(err, fmt.Sprintf("failed to create platform network: %s", common.FormatStruct(opts)))
			return err
		}

		r.ReconcilerEventLogger.NormalEvent(network_instance, common.ResourceCreated,
			fmt.Sprintf("platform network '%s' has been created", network_instance.Name))
	} else {
		_, err := networks.Update(client, uuid, opts).Extract()
		if err != nil {
			logHost.Error(err, fmt.Sprintf("failed to update platform network: %s", common.FormatStruct(opts)))
			return err
		}

		r.ReconcilerEventLogger.NormalEvent(network_instance, common.ResourceUpdated,
			fmt.Sprintf("platform network '%s' has been updated", network_instance.Name))
	}

	return nil
}

// CreateOrUpdateAddrPools creates or updates addresspools on the system.
func (r *HostReconciler) CreateOrUpdateAddrPools(client *gophercloud.ServiceClient, opts addresspools.AddressPoolOpts, uuid string, addrpool_instance *starlingxv1.AddressPool) error {
	if uuid == "" {
		_, err := addresspools.Create(client, opts).Extract()
		if err != nil {
			logHost.Error(err, fmt.Sprintf("failed to create addresspool: %s", common.FormatStruct(opts)))
			return err
		}

		r.ReconcilerEventLogger.NormalEvent(addrpool_instance, common.ResourceCreated,
			fmt.Sprintf("addresspool '%s' has been created", addrpool_instance.Name))
	} else {
		_, err := addresspools.Update(client, uuid, opts).Extract()
		if err != nil {
			logHost.Error(err, fmt.Sprintf("failed to update addresspool: %s", common.FormatStruct(opts)))
			return err
		}

		r.ReconcilerEventLogger.NormalEvent(addrpool_instance, common.ResourceUpdated,
			fmt.Sprintf("addresspool '%s' has been updated", addrpool_instance.Name))
	}

	return nil
}

// GetAssociatedNetworkAddrPool returns network-addrpool object associated with
// given AddressPool instance (based on the addresspool family).
func GetAssociatedNetworkAddrPool(
	system_network *networks.Network,
	addrpool_instance *starlingxv1.AddressPool,
	system_network_addrpools []networkAddressPools.NetworkAddressPool,
	addrpool_list []addresspools.AddressPool) (*networkAddressPools.NetworkAddressPool, *addresspools.AddressPool) {

	for _, network_addrpool := range system_network_addrpools {
		if network_addrpool.NetworkUUID == system_network.UUID {
			addrpool := utils.GetSystemAddrPoolByUUID(addrpool_list, network_addrpool.AddressPoolUUID)
			if addrpool != nil {
				if utils.IsIPv4(addrpool.Network) == utils.IsIPv4(addrpool_instance.Spec.Subnet) {
					// If the addresspool is from same network family
					// return it as associated network-addresspool object.
					// A network can have at most two network-addresspools,
					// one from each network family ie. IPv4 & IPv6.
					return &network_addrpool, addrpool
				}
			}
		}
	}

	return nil, nil
}

// GetAllNetworkAddressPoolData returns all the network-addrpool objects
// configured on the system.
func (r *HostReconciler) GetAllNetworkAddressPoolData(
	client *gophercloud.ServiceClient,
	network_instance *starlingxv1.PlatformNetwork) (
	error,
	*networks.Network,
	[]networkAddressPools.NetworkAddressPool,
	[]addresspools.AddressPool) {

	system_network, err := GetSystemNetwork(client, network_instance)
	if err != nil {
		return err, nil, nil, nil
	}

	system_network_addrpools, err := networkAddressPools.ListNetworkAddressPools(client)
	if err != nil {
		logHost.Error(err, "failed to fetch network-addresspools from system")
		return err, nil, nil, nil
	}

	addrpool_list, err := addresspools.ListAddressPools(client)
	if err != nil {
		logHost.Error(err, "failed to fetch addresspools from system")
		return err, nil, nil, nil
	}

	return nil, system_network, system_network_addrpools, addrpool_list

}

// UpdateNetworkAddrPools determines and (re)creates network-addrpool objects
// on the system depending upon associated address pools mentioned in the
// PlatformNetwork spec.
func (r *HostReconciler) UpdateNetworkAddrPools(client *gophercloud.ServiceClient, network_instance *starlingxv1.PlatformNetwork, addrpool_instance *starlingxv1.AddressPool) error {

	err, system_network, system_network_addrpools, addrpool_list := r.GetAllNetworkAddressPoolData(
		client, network_instance)
	if err != nil {
		return err
	}

	if system_network == nil {
		// No point in continuing if there is no network already
		return nil
	}

	system_addrpool, err := GetSystemAddrPool(client, addrpool_instance)
	if err != nil {
		return err
	} else if system_addrpool == nil {
		// No point in continuing if there is no addresspool already
		return nil
	}

	network_addrpool, associated_addrpool := GetAssociatedNetworkAddrPool(
		system_network,
		addrpool_instance,
		system_network_addrpools,
		addrpool_list)

	if network_addrpool != nil && associated_addrpool != nil {
		if associated_addrpool.ID != system_addrpool.ID {
			// Delete the associated network addrpool since it's not
			// linked to same address pool as the address pool spec.
			err := networkAddressPools.Delete(client, network_addrpool.UUID).ExtractErr()
			if err != nil {
				logHost.Error(err, "failed to delete associated network-addresspool")
				return err
			} else {
				log_msg := fmt.Sprintf(
					"Deleted network-addresspool object %s - %s",
					network_addrpool.NetworkName,
					network_addrpool.AddressPoolName)

				r.ReconcilerEventLogger.NormalEvent(network_instance, common.ResourceDeleted,
					log_msg)
			}
		} else {
			// No action required there is already network-addresspool
			// association created for given network and addresspool
			log_msg := fmt.Sprintf(
				"Found network-addresspool with %s - %s. No need to delete/recreate network-addresspool association.",
				network_addrpool.NetworkName,
				network_addrpool.AddressPoolName)

			logHost.V(2).Info(log_msg)

			return nil
		}
	}

	opts := networkAddressPools.NetworkAddressPoolOpts{}
	opts.NetworkUUID = &system_network.UUID
	opts.AddressPoolUUID = &system_addrpool.ID

	_, err = networkAddressPools.Create(client, opts).Extract()

	if err == nil {
		msg := fmt.Sprintf("Created new network-addrpool association %s - %s", system_network.Name, system_addrpool.Name)

		r.ReconcilerEventLogger.NormalEvent(network_instance, common.ResourceCreated,
			msg)
	} else {
		logHost.Error(err, "there was an error creating new network-addresspool.")
	}

	return err

}

// UpdateNetworkReconciliationStatus updates the reconciliation status of
// PlatformNetwork spec.
func (r *HostReconciler) UpdateNetworkReconciliationStatus(
	network_instance *starlingxv1.PlatformNetwork,
	is_reconciled bool,
	reconcile_expected bool) error {

	oldInSync := network_instance.Status.InSync

	if network_instance.Status.DeploymentScope == cloudManager.ScopeBootstrap {
		if !reconcile_expected {
			// Prevents raising alarm if configuration of given network type
			// is unsupported in day-1 and system is out-of-sync.
			// Insync will serve as reconciliation indicator in this case.
			network_instance.Status.Reconciled = true
		} else {
			network_instance.Status.Reconciled = is_reconciled
		}
		network_instance.Status.InSync = is_reconciled
	} else {
		network_instance.Status.InSync = is_reconciled
		network_instance.Status.Reconciled = is_reconciled
	}

	err := r.Client.Status().Update(context.TODO(), network_instance)
	if err != nil {
		logHost.Error(err, fmt.Sprintf("failed to update '%s' platform network status", network_instance.Name))
		return err
	}

	if oldInSync != network_instance.Status.InSync {
		r.ReconcilerEventLogger.NormalEvent(network_instance, common.ResourceUpdated,
			"%s network's synchronization has changed to: %t", network_instance.Name, network_instance.Status.InSync)
	}

	return nil
}

// UpdateAddrPoolReconciliationStatus updates reconciliation status of the
// AddressPool spec.
func (r *HostReconciler) UpdateAddrPoolReconciliationStatus(
	network_instance *starlingxv1.PlatformNetwork,
	addrpool_instance *starlingxv1.AddressPool,
	is_reconciled bool,
	reconcile_expected bool) error {

	oldInSync := addrpool_instance.Status.InSync

	// AddressPool doesn't have deploymentScope by design.
	// It inherits deploymentScope of associated network.
	if network_instance.Status.DeploymentScope == cloudManager.ScopeBootstrap {
		if !reconcile_expected {
			// Prevents raising alarm if configuration of given network type
			// is unsupported in day-1 and system is out-of-sync.
			// Insync will serve as reconciliation indicator in this case.
			addrpool_instance.Status.Reconciled = true
		} else {
			addrpool_instance.Status.Reconciled = is_reconciled
		}
		addrpool_instance.Status.InSync = is_reconciled
	} else if network_instance.Status.DeploymentScope == cloudManager.ScopePrincipal {
		addrpool_instance.Status.InSync = is_reconciled
		addrpool_instance.Status.Reconciled = is_reconciled
	}

	err := r.Client.Status().Update(context.TODO(), addrpool_instance)
	if err != nil {
		logHost.Error(err, fmt.Sprintf("failed to update '%s' addresspool status", addrpool_instance.Name))
		return err
	}

	if oldInSync != addrpool_instance.Status.InSync {
		r.ReconcilerEventLogger.NormalEvent(addrpool_instance, common.ResourceUpdated,
			"%s addresspool's synchronization has changed to: %t", addrpool_instance.Name, addrpool_instance.Status.InSync)
	}

	return nil
}

// ReconcileNetworkResource reconciles a PlatformNetwork resource as per spec.
func (r *HostReconciler) ReconcileNetworkResource(client *gophercloud.ServiceClient, network_instance *starlingxv1.PlatformNetwork) (error, *bool) {

	system_network, err := GetSystemNetwork(client, network_instance)
	if err != nil {
		return err, nil
	}

	addrpool_instances, fetch_errs := r.GetAddressPoolsFromPlatformNetwork(network_instance.Spec.AssociatedAddressPools,
		network_instance.Namespace)
	if len(fetch_errs) > 0 {
		// There were errors while fetching associated addresspool instances.
		// Reconciliation request will be requeued.
		return fetch_errs[0], nil
	}

	primary_address_pool, err := GetSystemAddrPool(client, addrpool_instances[0])
	if err != nil {
		return err, nil
	} else if system_network == nil && primary_address_pool == nil {
		return common.NewPlatformNetworkReconciliationError("Cannot create a network without primary address pool."), nil
	}

	r.UpdateNetworkUUID(network_instance, system_network)

	opts, update_required, uuid := r.IsNetworkUpdateRequired(network_instance, system_network, primary_address_pool)

	err, reconcile_expected := r.ReconcilePlatformNetworkExpected(client, network_instance, nil, update_required, uuid)
	if err != nil {
		return err, nil
	}

	if reconcile_expected && update_required {
		err := r.CreateOrUpdateNetworks(client, opts, uuid, network_instance)
		if err == nil {
			// Make sure network UUID is synchronized
			system_network, err = GetSystemNetwork(client, network_instance)
			if err != nil {
				return err, nil
			}
			r.UpdateNetworkUUID(network_instance, system_network)
		}
		return err, &reconcile_expected
	} else if update_required {
		err_msg := fmt.Sprintf(
			"There is delta between applied spec and system for platform network '%s'",
			network_instance.Name)
		err := perrors.New(err_msg)
		return err, &reconcile_expected
	}

	return nil, &reconcile_expected

}

// ReconcileNetworkAndAddressPools function implements the actual behavior to
// reconcile network and addresspool objects based on specs applied.
func (r *HostReconciler) ReconcileNetworkAndAddressPools(
	client *gophercloud.ServiceClient,
	network_instance *starlingxv1.PlatformNetwork,
	addrpool_instance *starlingxv1.AddressPool,
	system_info *cloudManager.SystemInfo) error {

	err, addrpool_reconcile_expected, validation_result := r.ReconcileAddrPoolResource(client, network_instance, addrpool_instance, system_info)
	if err != nil && addrpool_reconcile_expected == nil {
		// Some other error occured not related to reconciliation.
		// Eg. error listing networks by querying the system.
		// Request will be requeued.
		return err
	}

	addrpool_is_reconciled := err == nil && *validation_result

	err_status := r.UpdateAddrPoolReconciliationStatus(
		network_instance,
		addrpool_instance,
		addrpool_is_reconciled,
		*addrpool_reconcile_expected)

	if *addrpool_reconcile_expected && err != nil {
		//Reconciliation request will be requeued
		return err
	} else if err_status != nil {
		//Reconciliation request will be requeued
		return err_status
	} else if !*validation_result {
		// If addresspool reconciliation is expected and applied spec is not valid
		// error will be logged and reconciliation request should not be requeued
		// since it will only be futile.
		// An example would be pxeboot network type with IPv6 addresspool - this
		// never succeeds.
		return nil
	}

	if network_instance.Spec.Type == cloudManager.OtherNetworkType {
		// The network type "other" does not exist in real sense.
		// It's serves as an indicator to DM that only addresspool
		// needs to be reconciled and not network / network-addrpools.
		// Just set reconciled / insync to true for the sake of consistency.
		err_status = r.UpdateNetworkReconciliationStatus(
			network_instance,
			true,
			true)

		if err_status != nil {
			//Reconciliation request will be requeued
			return err_status
		}

		return nil
	}

	// It's important to note that UUID of first addresspool in the list of "AssociatedAddressPools"
	// will be used during creation of network and this will become the primary IP pool of the network.
	err, network_reconcile_expected := r.ReconcileNetworkResource(client, network_instance)
	if err != nil && network_reconcile_expected == nil {
		// Some other error occured not related to reconciliation.
		// Eg. error listing networks by querying the system.
		// Request will be requeued.
		return err
	}

	network_is_reconciled := err == nil
	err_status = r.UpdateNetworkReconciliationStatus(
		network_instance,
		network_is_reconciled,
		*network_reconcile_expected)

	if *network_reconcile_expected && err != nil {
		//Reconciliation request will be requeued
		return err
	} else if err_status != nil {
		//Reconciliation request will be requeued
		return err_status
	}

	// Update network-addresspool only if network and addresspools have been reconciled.
	if *addrpool_reconcile_expected && addrpool_is_reconciled && network_is_reconciled {
		logHost.V(2).Info(
			fmt.Sprintf("Updating network-addresspool association for network '%s' and addrpool '%s'",
				network_instance.Name, addrpool_instance.Name))
		update_err := r.UpdateNetworkAddrPools(client, network_instance, addrpool_instance)
		if update_err != nil {
			return update_err
		}
	}

	return nil
}

// ReconcilePlatformNetworkBootstrap reconciles PlatformNetwork and AddressPool specs
// in the context of deploymentScope bootstrap (Day-1 operation).
func (r *HostReconciler) ReconcilePlatformNetworkBootstrap(
	client *gophercloud.ServiceClient,
	host_instance *starlingxv1.Host,
	network_instance *starlingxv1.PlatformNetwork,
	addrpool_instance *starlingxv1.AddressPool,
	system_info *cloudManager.SystemInfo) error {

	return r.ReconcileNetworkAndAddressPools(client, network_instance, addrpool_instance, system_info)

}

// ReconcileMgmtPrincipalSimplex reconciles PlatformNetwork and AddressPool specs
// of mgmt network type in the context of deploymentScope principal (Day-2 operation)
// on AIO-SX system.
func (r *HostReconciler) ReconcileMgmtPrincipalSimplex(
	client *gophercloud.ServiceClient,
	host_instance *starlingxv1.Host,
	host *v1info.HostInfo,
	network_instance *starlingxv1.PlatformNetwork,
	addrpool_instance *starlingxv1.AddressPool,
	system_info *cloudManager.SystemInfo) error {

	if host.IsLockedDisabled() {
		return r.ReconcileNetworkAndAddressPools(client, network_instance, addrpool_instance, system_info)
	} else {
		return r.LockHostRequest(host_instance, host.ID, host.Personality, "platform networks")
	}
}

// ReconcilePlatformNetworkPrincipalSimplex reconciles PlatformNetwork and AddressPool specs
// in the context of deploymentScope principal (Day-2 operation) on AIO-SX system.
func (r *HostReconciler) ReconcilePlatformNetworkPrincipalSimplex(
	client *gophercloud.ServiceClient,
	host_instance *starlingxv1.Host,
	host *v1info.HostInfo,
	network_instance *starlingxv1.PlatformNetwork,
	addrpool_instance *starlingxv1.AddressPool,
	system_info *cloudManager.SystemInfo) error {

	switch network_instance.Spec.Type {
	case cloudManager.OAMNetworkType,
		cloudManager.MgmtNetworkType,
		cloudManager.AdminNetworkType:

		system_addrpool, err := GetSystemAddrPool(client, addrpool_instance)
		if err != nil {
			return err
		}

		_, update_required, _ := r.IsAddrPoolUpdateRequired(network_instance, addrpool_instance, system_addrpool)

		validation_result := r.ValidateAddressPool(network_instance, addrpool_instance, system_info)

		if update_required && validation_result {

			// This is valid mgmt network reconfiguration / fresh configuration and
			// spec is clearly not in sync with the system and hence it should be
			// allowed for reconciliation.

			if network_instance.Spec.Type == cloudManager.MgmtNetworkType {
				return r.ReconcileMgmtPrincipalSimplex(
					client,
					host_instance,
					host,
					network_instance,
					addrpool_instance,
					system_info)
			}

		}

		return r.ReconcileNetworkAndAddressPools(client, network_instance, addrpool_instance, system_info)

	default:
		return r.ReconcileNetworkAndAddressPools(client, network_instance, addrpool_instance, system_info)
	}

}

// ReconcilePlatformNetworkPrincipal reconciles PlatformNetwork and AddressPool specs
// in the context of deploymentScope principal (Day-2 operation).
func (r *HostReconciler) ReconcilePlatformNetworkPrincipal(
	client *gophercloud.ServiceClient,
	host_instance *starlingxv1.Host,
	host *v1info.HostInfo,
	network_instance *starlingxv1.PlatformNetwork,
	addrpool_instance *starlingxv1.AddressPool,
	system_info *cloudManager.SystemInfo) error {

	if system_info.SystemType == cloudManager.SystemTypeAllInOne &&
		system_info.SystemMode == cloudManager.SystemModeSimplex {

		return r.ReconcilePlatformNetworkPrincipalSimplex(client, host_instance, host, network_instance, addrpool_instance, system_info)

	}

	return nil

}

// ReconcilePlatformNetworkAndAddrPoolResource is a function that reconciles a
// particular PlatformNetwork and associated AddressPool specs.
func (r *HostReconciler) ReconcilePlatformNetworkAndAddrPoolResource(
	client *gophercloud.ServiceClient,
	host_instance *starlingxv1.Host,
	host *v1info.HostInfo,
	network_instance *starlingxv1.PlatformNetwork,
	addrpool_instance *starlingxv1.AddressPool,
	system_info *cloudManager.SystemInfo) error {

	if network_instance.Status.DeploymentScope == cloudManager.ScopeBootstrap {

		return r.ReconcilePlatformNetworkBootstrap(client, host_instance, network_instance, addrpool_instance, system_info)

	} else if network_instance.Status.DeploymentScope == cloudManager.ScopePrincipal {

		return r.ReconcilePlatformNetworkPrincipal(client, host_instance, host, network_instance, addrpool_instance, system_info)

	}

	return nil

}

// ReconcilePlatformNetworks is a function that reconciles PlatformNetwork specs and
// associated AddressPool specs applied by the user.
func (r *HostReconciler) ReconcilePlatformNetworks(client *gophercloud.ServiceClient, instance *starlingxv1.Host, profile *starlingxv1.HostProfileSpec, host *v1info.HostInfo, system_info *cloudManager.SystemInfo) []error {
	var errs []error
	if !utils.IsReconcilerEnabled(utils.HostPlatformNetwork) {
		return nil
	}

	platform_network_instances, fetch_errs := r.ListPlatformNetworks(instance.Namespace)
	errs = append(errs, fetch_errs...)

	for _, platform_network_instance := range platform_network_instances {
		addrpool_instances, fetch_errs := r.GetAddressPoolsFromPlatformNetwork(platform_network_instance.Spec.AssociatedAddressPools,
			instance.Namespace)
		errs = append(errs, fetch_errs...)

		for _, addrpool_instance := range addrpool_instances {
			err := r.ReconcilePlatformNetworkAndAddrPoolResource(client, instance, host, platform_network_instance, addrpool_instance, system_info)
			if err != nil {
				errs = append(errs, err)

				cause := perrors.Cause(err)
				if _, ok := cause.(cloudManager.WaitForMonitor); ok {
					// Perform no further reconciliations until host reaches
					// expected state.
					return errs
				}
			}
		}
	}

	return errs
}
