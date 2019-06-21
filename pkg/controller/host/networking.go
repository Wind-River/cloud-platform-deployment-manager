/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019 Wind River Systems, Inc. */

package host

import (
	"fmt"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/addresses"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/interfaceDataNetworks"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/interfaceNetworks"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/interfaces"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/routes"
	perrors "github.com/pkg/errors"
	starlingxv1beta1 "github.com/wind-river/titanium-deployment-manager/pkg/apis/starlingx/v1beta1"
	utils "github.com/wind-river/titanium-deployment-manager/pkg/common"
	"github.com/wind-river/titanium-deployment-manager/pkg/controller/common"
	"github.com/wind-river/titanium-deployment-manager/pkg/manager"
	v1info "github.com/wind-river/titanium-deployment-manager/pkg/platform"
	"strings"
)

// interfaceIsStaleOrChanged is a utility function which determines whether
// a system interface name and a profile interface name refer to the same
// interface.  The purpose of this is to determine whether resources configured
// to use the old interface need to be removed and then re-added once the new
// interface has been setup.  If the interface will not be deleted and then
// later re-added during reconciliation then there is no need to delete the
// higher level resource.  For example, if a specific address is first
// configured on an vlan named "foo" with vid 11, but then later is to be moved
// to a vlan named "foo" with vid 12 we need to delete the address, delete the
// vlan, re-add the vlan with the right vid and then re-add the address.
func interfaceIsStaleOrChanged(oldName string, newName *string, profile *starlingxv1beta1.HostProfileSpec, host *v1info.HostInfo) bool {
	iface, found := host.FindInterfaceByName(oldName)
	if !found {
		// This is really an error, but for all practical purposes, returning
		// true here will force the resource to be deleted and re-added which
		// will hopefully resolve any data conflicts.
		return true
	}

	data := profile.Interfaces

	switch iface.Type {
	case interfaces.IFTypeEthernet:
		if portname, found := host.FindInterfacePortName(iface.ID); found {
			for _, e := range data.Ethernet {
				if newName != nil && e.Name != *newName {
					continue
				}

				if e.Port.Name == portname {
					// There is a matching ethernet in the pending configuration
					// that is based on the same ethernet interface therefore
					// this interface will still exist after the configuration
					// is applied.
					return false
				}
			}
		}

		// No equivalent interface was found.  It may now be of a different type
		return true

	case interfaces.IFTypeVLAN:
		for _, v := range data.VLAN {
			if newName != nil && v.Name != *newName {
				continue
			}
			if iface.VID != nil && v.VID == *iface.VID {
				// Repeat the same process on the lower interface to
				// determine if the vlan is going to move in which case
				// the resources over it must be deleted and re-added.
				return interfaceIsStaleOrChanged(iface.Uses[0], &v.Lower, profile, host)
			}
		}

		// No equivalent interface was found.  It may now be of a different type
		return true

	case interfaces.IFTypeAE:
		// Bond interfaces technically do not need to be deleted and re-added
		// since all of their attributes can be changed on the fly, but since
		// their name can change it makes it tricky to determine if resources
		// added over them need to be moved.  So, the best we can do here is to
		// look at the member interfaces and if there are any members in common
		// then we assume that the interface is the same otherwise a new one
		// will created.
		for _, b := range data.Bond {
			if newName != nil && b.Name != *newName {
				continue
			}
			for _, u := range iface.Uses {
				for _, m := range b.Members {
					if !interfaceIsStaleOrChanged(u, &m, profile, host) {
						return false
					}
				}
			}
		}

		// No equivalent interface was found.  It may now be of a different type
		return true

	case interfaces.IFTypeVirtual:
		// These are never deleted or renamed.
		return false

	default:
		log.Info(fmt.Sprintf("unexpected interface type: %s", iface.Type))
		return true
	}
}

func (r *ReconcileHost) ReconcileStaleRoutes(client *gophercloud.ServiceClient, instance *starlingxv1beta1.Host, profile *starlingxv1beta1.HostProfileSpec, host *v1info.HostInfo) error {
	updated := false

	if r.IsReconcilerEnabled(manager.Route) == false {
		return nil
	}

	for _, route := range host.Routes {
		var newName *string = nil

		for _, x := range profile.Routes {
			if x.Interface == route.InterfaceName &&
				strings.EqualFold(x.Network, route.Network) &&
				x.Prefix == route.Prefix &&
				x.Gateway == route.Gateway &&
				(x.Metric == nil || *x.Metric == route.Metric) {
				// Routes cannot be updated so all fields must match
				// otherwise re-provisioning is required.
				newName = &x.Interface
				break
			}
		}

		if newName == nil || interfaceIsStaleOrChanged(route.InterfaceName, newName, profile, host) {
			log.Info("deleting route", "uuid", route.ID)

			err := routes.Delete(client, route.ID).ExtractErr()
			if err != nil {
				err = perrors.Wrapf(err, "failed to delete route %s", route.ID)
				return err
			}

			r.NormalEvent(instance, common.ResourceDeleted,
				"stale route '%s/%d' has been deleted", route.Network, route.Prefix)

			updated = true
		}
	}

	if updated {
		results, err := routes.ListRoutes(client, host.ID)
		if err != nil {
			err = perrors.Wrapf(err, "failed to refresh routes on hostid %s", host.ID)
			return err
		}

		host.Routes = results
	}

	return nil
}

func (r *ReconcileHost) ReconcileStaleAddresses(client *gophercloud.ServiceClient, instance *starlingxv1beta1.Host, profile *starlingxv1beta1.HostProfileSpec, host *v1info.HostInfo) error {
	updated := false

	if r.IsReconcilerEnabled(manager.Address) == false {
		return nil
	}

	for _, addr := range host.Addresses {
		var newName *string = nil

		if addr.PoolUUID != nil {
			// Automatically assigned addresses should be ignored.  The system
			// will remove them when the pool is removed from the interface.
			continue
		}

		for _, x := range profile.Addresses {
			if strings.EqualFold(x.Address, addr.Address) && x.Prefix == addr.Prefix {
				// Addresses cannot be updated so unless both the address
				// and prefix match we consider it to require
				// re-provisioning.
				newName = &x.Interface
				break
			}
		}

		if newName == nil || interfaceIsStaleOrChanged(addr.InterfaceName, newName, profile, host) {
			log.Info("deleting address", "uuid", addr.ID)

			err := addresses.Delete(client, addr.ID).ExtractErr()
			if err != nil {
				err = perrors.Wrapf(err, "failed to delete address %s", addr.ID)
				return err
			}

			r.NormalEvent(instance, common.ResourceDeleted,
				"stale address '%s/%d' has been deleted", addr.Address, addr.Prefix)

			updated = true
		}
	}

	if updated {
		results, err := addresses.ListAddresses(client, host.ID)
		if err != nil {
			err = perrors.Wrapf(err, "failed to refresh addresses on hostid %s", host.ID)
			return err
		}

		host.Addresses = results
	}

	return nil
}

// ReconcileStaleInterfaces will examine the current set of system interfaces
// and determine if any of them need to be deleted.  An interface needs to be
// deleted if:
//   A) it no longer exists in the list of interfaces to be configured
//   B) it still exists as a VLAN, but has a different vlan-id value
//   C) it still exists as a Bond, but has no members in common with the system
//      interface.
func (r *ReconcileHost) ReconcileStaleInterfaces(client *gophercloud.ServiceClient, instance *starlingxv1beta1.Host, profile *starlingxv1beta1.HostProfileSpec, host *v1info.HostInfo) error {
	updated := false

	if r.IsReconcilerEnabled(manager.Interface) == false {
		return nil
	}

	for _, ifInfo := range host.Interfaces {
		if ifInfo.Type == interfaces.IFTypeEthernet || ifInfo.Type == interfaces.IFTypeVirtual {
			continue
		}

		if interfaceIsStaleOrChanged(ifInfo.Name, nil, profile, host) {
			// This interface either no longer exists or needs to be
			// re-provisioned.
			log.Info("deleting interface", "uuid", ifInfo.ID)

			err := interfaces.Delete(client, ifInfo.ID).ExtractErr()
			if err != nil {
				err = perrors.Wrapf(err, "failed to delete interface %s", ifInfo.ID)
				return err
			}

			r.NormalEvent(instance, common.ResourceDeleted,
				"stale interface %q has been deleted", ifInfo.Name)

			updated = true
		}
	}

	if updated {
		results, err := interfaces.ListInterfaces(client, host.ID)
		if err != nil {
			err = perrors.Wrapf(err, "failed to refresh interfaces on hostid %s", host.ID)
			return err
		}

		host.Interfaces = results
	}

	return nil
}

// hasIPv4StaticAddresses is a utility function which determines if an interface
// has any configured static addresses.
func hasIPv4StaticAddresses(info starlingxv1beta1.CommonInterfaceInfo, profile *starlingxv1beta1.HostProfileSpec) bool {
	for _, addrInfo := range profile.Addresses {
		if utils.IsIPv4(addrInfo.Address) {
			if addrInfo.Interface == info.Name {
				return true
			}
		}
	}
	return false
}

// hasIPv6StaticAddresses is a utility function which determines if an interface
// has any configured static addresses.
func hasIPv6StaticAddresses(info starlingxv1beta1.CommonInterfaceInfo, profile *starlingxv1beta1.HostProfileSpec) bool {
	for _, addrInfo := range profile.Addresses {
		if utils.IsIPv4(addrInfo.Address) {
			if addrInfo.Interface == info.Name {
				return true
			}
		}
	}
	return false
}

// hasIPv4DynamicAddresses is a utility function which determines if an
// interface has any configured networks that are dynamic which would
// necessitate enabling dynamic addressing on an interface.
// TODO(alegacy): this is currently only intended to be used data interfaces
//  because it assumes that the presence of a pool is enough to determine if
//  it is dynamic rather than looking for a network to determine if that network
//  is dynamic or not (which is currently not possible for networks used for
//  data interfaces).
func hasIPv4DynamicAddresses(info starlingxv1beta1.CommonInterfaceInfo, host *v1info.HostInfo) (*string, bool) {
	if info.PlatformNetworks == nil {
		return nil, false
	}

	for _, networkName := range *info.PlatformNetworks {
		pool := host.FindAddressPoolByName(networkName)
		if pool != nil {
			if utils.IsIPv4(pool.Network) {
				return &pool.ID, true
			}
		}
	}

	return nil, false
}

// hasIPv6DynamicAddresses is a utility function which determines if an
// interface has any configured networks that are dynamic which would
// necessitate enabling dynamic addressing on an interface.
// TODO(alegacy): this is currently only intended to be used data interfaces
//  because it assumes that the presence of a pool is enough to determine if
//  it is dynamic rather than looking for a network to determine if that network
//  is dynamic or not (which is currently not possible for networks used for
//  data interfaces).
func hasIPv6DynamicAddresses(info starlingxv1beta1.CommonInterfaceInfo, host *v1info.HostInfo) (*string, bool) {
	if info.PlatformNetworks == nil {
		return nil, false
	}

	for _, networkName := range *info.PlatformNetworks {
		pool := host.FindAddressPoolByName(networkName)
		if pool != nil {
			if utils.IsIPv6(pool.Network) {
				return &pool.ID, true
			}
		}
	}

	return nil, false
}

// getInterfaceIPv4Address is a utility function which determines what address
// mode settings should be applied to an interface.
func getInterfaceIPv4Addressing(info starlingxv1beta1.CommonInterfaceInfo, profile *starlingxv1beta1.HostProfileSpec, host *v1info.HostInfo) (mode string, pool *string) {
	var ok bool

	if hasIPv4StaticAddresses(info, profile) {
		mode = interfaces.AddressModeStatic
	} else if pool, ok = hasIPv4DynamicAddresses(info, host); ok {
		mode = interfaces.AddressModePool
	} else {
		mode = interfaces.AddressModeDisabled
	}

	return mode, pool
}

// getInterfaceIPv6Address is a utility function which determines what address
// mode settings should be applied to an interface.
func getInterfaceIPv6Addressing(info starlingxv1beta1.CommonInterfaceInfo, profile *starlingxv1beta1.HostProfileSpec, host *v1info.HostInfo) (mode string, pool *string) {
	var ok bool

	if hasIPv6StaticAddresses(info, profile) {
		mode = interfaces.AddressModeStatic
	} else if pool, ok = hasIPv6DynamicAddresses(info, host); ok {
		mode = interfaces.AddressModePool
	} else {
		mode = interfaces.AddressModeDisabled
	}

	return mode, pool
}

// interfaceUpdateRequired is a utility function which determines whether the
// common interface attributes have changed and if so fills in the opts struct
// with the values that must be passed to the system API.
func interfaceUpdateRequired(info starlingxv1beta1.CommonInterfaceInfo, iface *interfaces.Interface, profile *starlingxv1beta1.HostProfileSpec, host *v1info.HostInfo) (opts interfaces.InterfaceOpts, result bool) {
	if iface.Type != interfaces.IFTypeVirtual {
		// Only allow name changes for non-virtual interfaces.  That is, never
		// allow "lo" to be renamed.
		if info.Name != iface.Name {
			opts.Name = &info.Name
			result = true
		}
	}

	if strings.EqualFold(info.Class, iface.Class) == false {
		if info.Class != interfaces.IFClassNone || iface.Class != "" {
			// Class is returned from the systemAPI as "" if it is not set, but
			// we need to set it to "none" in order to clear it.
			opts.Class = &info.Class
			result = true
		}
	}

	if info.MTU != nil && *info.MTU != iface.MTU {
		opts.MTU = info.MTU
		result = true
	}

	if info.Class == interfaces.IFClassData {
		// TODO(alegacy): We might need to remove this restriction and manage
		//  these attributes for other interface classes, but for now limit our
		//  handling of these for data interfaces only.

		mode, pool := getInterfaceIPv4Addressing(info, profile, host)
		if iface.IPv4Mode == nil && mode != interfaces.AddressModeDisabled ||
			iface.IPv4Mode != nil && mode != *iface.IPv4Mode {
			opts.IPv4Mode = &mode
			result = true
		}
		if pool == nil && iface.IPv4Pool != nil || pool != nil && iface.IPv4Pool == nil {
			opts.IPv4Pool = pool
			result = true
		} else if pool != nil && iface.IPv4Pool != nil && *pool != *iface.IPv4Pool {
			opts.IPv4Pool = pool
			result = true
		}

		mode, pool = getInterfaceIPv6Addressing(info, profile, host)
		if iface.IPv6Mode == nil && mode != interfaces.AddressModeDisabled ||
			iface.IPv6Mode != nil && mode != *iface.IPv6Mode {
			opts.IPv6Mode = &mode
			result = true
		}
		if pool == nil && iface.IPv6Pool != nil || pool != nil && iface.IPv6Pool == nil {
			opts.IPv6Pool = pool
			result = true
		} else if pool != nil && iface.IPv6Pool != nil && *pool != *iface.IPv6Pool {
			opts.IPv6Pool = pool
			result = true
		}
	}

	return opts, result
}

// ethernetUpdateRequired is a utility function which determines whether the
// ethernet specific interface attributes have changed and if so fills in the
// opts struct with the values that must be passed to the system API.
func ethernetUpdateRequired(ethInfo starlingxv1beta1.EthernetInfo, iface *interfaces.Interface, opts *interfaces.InterfaceOpts) (result bool) {
	if ethInfo.CommonInterfaceInfo.Class == interfaces.IFClassPCISRIOV {
		// Ensure that SRIOV VF count is up to date.
		if ethInfo.VFCount != nil {
			if iface.VFCount == nil {
				opts.VFCount = ethInfo.VFCount
				result = true
			} else if *ethInfo.VFCount != *iface.VFCount {
				opts.VFCount = ethInfo.VFCount
				result = true
			}
		} else if iface.VFCount != nil && *iface.VFCount != 0 {
			zero := 0
			opts.VFCount = &zero
			result = true
		}
	}

	return result
}

// ReconcileInterfaceNetworks implements a method to reconcile the list of
// networks on an interface against the configured set of networks.
func (r *ReconcileHost) ReconcileInterfaceNetworks(client *gophercloud.ServiceClient, instance *starlingxv1beta1.Host, info starlingxv1beta1.CommonInterfaceInfo, iface interfaces.Interface, host *v1info.HostInfo) (updated bool, err error) {
	if info.PlatformNetworks == nil {
		return updated, err
	}

	// Get the lists of current and configured networks on this interface
	current := host.BuildInterfaceNetworkList(iface)
	configured := *info.PlatformNetworks

	// Diff the lists to determine what changes need to be applied
	added, removed, _ := utils.ListDelta(current, configured)

	for _, name := range removed {
		if id, ok := host.FindInterfaceNetworkID(iface, name); ok {
			log.Info("deleting interface-network from interface", "ifname", iface.Name, "id", id)

			err := interfaceNetworks.Delete(client, id).ExtractErr()
			if err != nil {
				err = perrors.Wrapf(err, "failed to delete interface-network %q from iface %q", id, iface.Name)
				return updated, err
			}

			updated = true

			r.NormalEvent(instance, common.ResourceDeleted,
				"interface-network %q has been removed from %q", id, iface.Name)
		} else {
			msg := fmt.Sprintf("unable to find interface-network id for network %q on interface %q", name, iface.Name)
			return updated, starlingxv1beta1.NewMissingSystemResource(msg)
		}
	}

	for _, name := range added {
		if id, ok := host.FindNetworkID(name); ok {
			opts := interfaceNetworks.InterfaceNetworkOpts{
				InterfaceUUID: iface.ID,
				NetworkUUID:   id,
			}

			log.Info("creating an interface-network association", "ifname", iface.Name, "network", name)

			_, err := interfaceNetworks.Create(client, opts).Extract()
			if err != nil {
				err = perrors.Wrapf(err, "failed to create interface-network association: %s",
					common.FormatStruct(opts))
				return updated, err
			}

			updated = true

			r.NormalEvent(instance, common.ResourceCreated,
				"interface-network %q has been create on %q", name, iface.Name)
		}
	}

	return updated, err
}

// ReconcileInterfaceNetworks implements a method to reconcile the list of
// networks on an interface against the configured set of networks.
func (r *ReconcileHost) ReconcileInterfaceDataNetworks(client *gophercloud.ServiceClient, instance *starlingxv1beta1.Host, info starlingxv1beta1.CommonInterfaceInfo, iface interfaces.Interface, host *v1info.HostInfo) (updated bool, err error) {
	if info.DataNetworks == nil {
		return updated, err
	}

	// Get the lists of current and configured networks on this interface
	current := host.BuildInterfaceDataNetworkList(iface)
	configured := *info.DataNetworks

	// Diff the lists to determine what changes need to be applied
	added, removed, _ := utils.ListDelta(current, configured)

	for _, name := range removed {
		if id, ok := host.FindInterfaceDataNetworkID(iface, name); ok {
			log.Info("deleting interface-datanetwork from interface", "ifname", iface.Name, "id", id)

			err := interfaceDataNetworks.Delete(client, id).ExtractErr()
			if err != nil {
				err = perrors.Wrapf(err, "failed to delete interface-datanetwork %q from iface %q", id, iface.Name)
				return updated, err
			}

			updated = true

			r.NormalEvent(instance, common.ResourceDeleted,
				"interface-datanetwork %q has been removed from %q", id, iface.Name)
		} else {
			msg := fmt.Sprintf("unable to find interface-datanetwork id for network %q on interface %q", name, iface.Name)
			return updated, starlingxv1beta1.NewMissingSystemResource(msg)
		}
	}

	for _, name := range added {
		if id, ok := host.FindDataNetworkID(name); ok {
			opts := interfaceDataNetworks.InterfaceDataNetworkOpts{
				InterfaceUUID:   iface.ID,
				DataNetworkUUID: id,
			}

			log.Info("creating an interface-datanetwork association", "ifname", iface.Name, "network", name)

			_, err := interfaceDataNetworks.Create(client, opts).Extract()
			if err != nil {
				err = perrors.Wrapf(err, "failed to create interface-datanetwork association: %s",
					common.FormatStruct(opts))
				return updated, err
			}

			updated = true

			r.NormalEvent(instance, common.ResourceCreated,
				"interface-datanetwork %q has been create on %q", name, iface.Name)
		}
	}

	return updated, err
}

// ReconcileEthernetInterfaces will update system interfaces to align with the
// desired configuration.  It is assumed that the configuration will apply;
// meaning that prior to invoking this function stale interfaces and stale
// interface configurations have been resolved so that the intended list of
// ethernet interface configuration will apply cleanly here.
func (r *ReconcileHost) ReconcileEthernetInterfaces(client *gophercloud.ServiceClient, instance *starlingxv1beta1.Host, profile *starlingxv1beta1.HostProfileSpec, host *v1info.HostInfo) error {
	var iface *interfaces.Interface
	var ifuuid string
	var found bool

	if r.IsReconcilerEnabled(manager.Interface) == false {
		return nil
	}

	if profile.Interfaces == nil || len(profile.Interfaces.Ethernet) == 0 {
		return nil
	}

	updated := false
	for _, ethInfo := range profile.Interfaces.Ethernet.SortByNetworkCount() {
		// For each configured ethernet interface update the associated system
		// resource.
		if ethInfo.Name != interfaces.LoopbackInterfaceName {
			ifuuid, found = host.FindPortInterfaceUUID(ethInfo.Port.Name)
			if !found {
				msg := fmt.Sprintf("unable to find interface UUID for port: %s", ethInfo.Port.Name)
				return starlingxv1beta1.NewMissingSystemResource(msg)
			}

			iface, found = host.FindInterface(ifuuid)
			if !found {
				msg := fmt.Sprintf("unable to find interface: %s", ifuuid)
				return starlingxv1beta1.NewMissingSystemResource(msg)
			}
		} else {
			iface, found = host.FindInterfaceByName(interfaces.LoopbackInterfaceName)
			if !found {
				msg := fmt.Sprintf("unable to find loopback interface: %s",
					interfaces.LoopbackInterfaceName)
				return starlingxv1beta1.NewMissingSystemResource(msg)
			}

			ifuuid = iface.ID
		}

		opts, ok1 := interfaceUpdateRequired(ethInfo.CommonInterfaceInfo, iface, profile, host)
		if ok2 := ethernetUpdateRequired(ethInfo, iface, &opts); ok1 || ok2 {
			log.Info("updating interface", "uuid", ifuuid, "opts", opts)

			_, err := interfaces.Update(client, ifuuid, opts).Extract()
			if err != nil {
				err = perrors.Wrapf(err, "failed to update interface: %s, %s",
					ifuuid, common.FormatStruct(opts))
				return err
			}

			r.NormalEvent(instance, common.ResourceUpdated,
				"ethernet interface %q has been updated", ethInfo.Name)

			updated = true
		}

		networksUpdated, err := r.ReconcileInterfaceNetworks(client, instance, ethInfo.CommonInterfaceInfo, *iface, host)
		if err != nil {
			return err
		}

		dataNetworksUpdated, err := r.ReconcileInterfaceDataNetworks(client, instance, ethInfo.CommonInterfaceInfo, *iface, host)
		if err != nil {
			return err
		}

		updated = updated || networksUpdated || dataNetworksUpdated
	}

	if updated {
		// Interfaces have been updated so we need to refresh the list of interfaces
		// so that the next method that needs to act on the list will have the
		// updated view of the system.
		objects, err := interfaces.ListInterfaces(client, host.ID)
		if err != nil {
			err = perrors.Wrapf(err, "failed to refresh interfaces for hostid: %s", host.ID)
			return err
		}

		host.Interfaces = objects

		results, err := interfaceNetworks.ListInterfaceNetworks(client, host.ID)
		if err != nil {
			err = perrors.Wrapf(err, "failed to refresh interface-networks for hostid: %s", host.ID)
			return err
		}

		host.InterfaceNetworks = results
	}

	return nil
}

// bondUpdateRequired is a utility function which determines whether the
// bond specificc interface attributes have changed and if so fills in the opts
// struct with the values that must be passed to the system API.
func bondUpdateRequired(bond starlingxv1beta1.BondInfo, iface *interfaces.Interface, opts *interfaces.InterfaceOpts) (result bool) {
	if iface.AEMode != nil && strings.EqualFold(bond.Mode, *iface.AEMode) == false {
		opts.AEMode = &bond.Mode
		result = true
	}

	if bond.TransmitHashPolicy != nil {
		if iface.AETransmitHash != nil && strings.EqualFold(*bond.TransmitHashPolicy, *iface.AETransmitHash) == false {
			opts.AETransmitHash = bond.TransmitHashPolicy
			result = true
		}
	}

	if utils.ListChanged(bond.Members, iface.Uses) {
		// The system API handles "uses" inconsistently between create and
		// update.  It requires a different attribute name for both.  This
		// looks like it is because the system API uses an old copy of ironic
		// as its bases and the json patch code does not support lists.
		members := bond.Members.ToStringList()
		opts.UsesModify = &members
		result = true
	}

	return result
}

// commonInterfaceOptions is a utility to populate the interface options for
// all common interface attributes.
func commonInterfaceOptions(info starlingxv1beta1.CommonInterfaceInfo, profile *starlingxv1beta1.HostProfileSpec, host *v1info.HostInfo) interfaces.InterfaceOpts {
	opts := interfaces.InterfaceOpts{
		HostUUID: &host.ID,
		Name:     &info.Name,
		Class:    &info.Class,
		MTU:      info.MTU,
	}

	if info.Class == interfaces.IFClassData {
		// TODO(alegacy): We might need to remove this restriction and manage
		//  these attributes for other interface classes, but for now limit our
		//  handling of these for data interfaces only.

		mode, pool := getInterfaceIPv4Addressing(info, profile, host)
		opts.IPv4Mode = &mode
		opts.IPv4Pool = pool

		mode, pool = getInterfaceIPv6Addressing(info, profile, host)
		opts.IPv6Mode = &mode
		opts.IPv6Pool = pool
	}

	return opts
}

func (r *ReconcileHost) ReconcileBondInterfaces(client *gophercloud.ServiceClient, instance *starlingxv1beta1.Host, profile *starlingxv1beta1.HostProfileSpec, host *v1info.HostInfo) (err error) {
	var iface *interfaces.Interface

	if r.IsReconcilerEnabled(manager.Interface) == false {
		return nil
	}

	if profile.Interfaces == nil || len(profile.Interfaces.Bond) == 0 {
		return nil
	}

	updated := false
	for _, bondInfo := range profile.Interfaces.Bond {
		// For each configured bond interface create or update the related
		// system resource.
		ifuuid, found := host.FindBondInterfaceUUID(bondInfo.Members)
		if !found {
			// Create the interface
			opts := commonInterfaceOptions(bondInfo.CommonInterfaceInfo, profile, host)

			iftype := interfaces.IFTypeAE
			opts.Type = &iftype
			opts.AEMode = &bondInfo.Mode
			opts.AETransmitHash = bondInfo.TransmitHashPolicy

			members := bondInfo.Members.ToStringList()
			opts.Uses = &members

			log.Info("creating bond interface", "opts", opts)

			iface, err = interfaces.Create(client, opts).Extract()
			if err != nil {
				err = perrors.Wrapf(err, "failed to create bond interface: %s",
					common.FormatStruct(opts))
				return err
			}

			r.NormalEvent(instance, common.ResourceCreated,
				"bond interface %q has been created", bondInfo.Name)

			updated = true
		} else {
			// Update the interface
			iface, found = host.FindInterface(ifuuid)
			if !found {
				msg := fmt.Sprintf("failed to find interface: %s", ifuuid)
				return starlingxv1beta1.NewMissingSystemResource(msg)
			}

			opts, ok1 := interfaceUpdateRequired(bondInfo.CommonInterfaceInfo, iface, profile, host)
			if ok2 := bondUpdateRequired(bondInfo, iface, &opts); ok1 || ok2 {
				log.Info("updating bond interface", "uuid", ifuuid, "opts", opts)

				_, err := interfaces.Update(client, ifuuid, opts).Extract()
				if err != nil {
					err = perrors.Wrapf(err, "failed to update interface: %s, %s",
						ifuuid, common.FormatStruct(opts))
					return err
				}

				r.NormalEvent(instance, common.ResourceUpdated,
					"ethernet interface %q has been updated", bondInfo.Name)

				updated = true
			}
		}

		networksUpdated, err := r.ReconcileInterfaceNetworks(client, instance, bondInfo.CommonInterfaceInfo, *iface, host)
		if err != nil {
			return err
		}

		dataNetworksUpdated, err := r.ReconcileInterfaceDataNetworks(client, instance, bondInfo.CommonInterfaceInfo, *iface, host)
		if err != nil {
			return err
		}

		updated = updated || networksUpdated || dataNetworksUpdated
	}

	if updated {
		// Interfaces have been updated so we need to refresh the list of interfaces
		// so that the next method that needs to act on the list will have the
		// updated view of the system.
		objects, err := interfaces.ListInterfaces(client, host.ID)
		if err != nil {
			err = perrors.Wrapf(err, "failed to refresh interfaces for hostid: %s", host.ID)
			return err
		}

		host.Interfaces = objects

		results, err := interfaceNetworks.ListInterfaceNetworks(client, host.ID)
		if err != nil {
			err = perrors.Wrapf(err, "failed to refresh interface-networks for hostid: %s", host.ID)
			return err
		}

		host.InterfaceNetworks = results
	}

	return nil
}

func (r *ReconcileHost) ReconcileVLANInterfaces(client *gophercloud.ServiceClient, instance *starlingxv1beta1.Host, profile *starlingxv1beta1.HostProfileSpec, host *v1info.HostInfo) (err error) {
	var iface *interfaces.Interface

	if r.IsReconcilerEnabled(manager.Interface) == false {
		return nil
	}

	if profile.Interfaces == nil || len(profile.Interfaces.VLAN) == 0 {
		return nil
	}

	updated := false

	for _, vlanInfo := range profile.Interfaces.VLAN {
		// For each configured bond interface create or update the related
		// system resource.
		ifuuid, found := host.FindVLANInterfaceUUID(vlanInfo.VID)
		if !found {
			// Create the interface
			opts := commonInterfaceOptions(vlanInfo.CommonInterfaceInfo, profile, host)

			iftype := interfaces.IFTypeVLAN
			opts.Type = &iftype
			opts.VID = &vlanInfo.VID
			uses := []string{vlanInfo.Lower}
			opts.Uses = &uses

			log.Info("creating vlan interface", "opts", opts)

			iface, err = interfaces.Create(client, opts).Extract()
			if err != nil {
				err = perrors.Wrapf(err, "failed to create vlan interface: %s",
					common.FormatStruct(opts))
				return err
			}

			r.NormalEvent(instance, common.ResourceCreated,
				"vlan interface %q has been created", vlanInfo.Name)

			updated = true
		} else {
			// Update the interface
			iface, found = host.FindInterface(ifuuid)
			if !found {
				msg := fmt.Sprintf("failed to find interface: %s", ifuuid)
				return starlingxv1beta1.NewMissingSystemResource(msg)
			}

			if opts, ok := interfaceUpdateRequired(vlanInfo.CommonInterfaceInfo, iface, profile, host); ok {
				log.Info("updating vlan interface", "uuid", ifuuid, "opts", opts)

				_, err := interfaces.Update(client, ifuuid, opts).Extract()
				if err != nil {
					err = perrors.Wrapf(err, "failed to update interface: %s, %s",
						ifuuid, common.FormatStruct(opts))
					return err
				}

				r.NormalEvent(instance, common.ResourceUpdated,
					"vlan interface %q has been updated", vlanInfo.Name)

				updated = true
			}
		}

		networksUpdated, err := r.ReconcileInterfaceNetworks(client, instance, vlanInfo.CommonInterfaceInfo, *iface, host)
		if err != nil {
			return err
		}

		dataNetworksUpdated, err := r.ReconcileInterfaceDataNetworks(client, instance, vlanInfo.CommonInterfaceInfo, *iface, host)
		if err != nil {
			return err
		}

		updated = updated || networksUpdated || dataNetworksUpdated
	}

	if updated {
		// Interfaces have been updated so we need to refresh the list of interfaces
		// so that the next method that needs to act on the list will have the
		// updated view of the system.
		objects, err := interfaces.ListInterfaces(client, host.ID)
		if err != nil {
			err = perrors.Wrapf(err, "failed to refresh interfaces for hostid: %s",
				host.ID)
			return err
		}

		host.Interfaces = objects

		results, err := interfaceNetworks.ListInterfaceNetworks(client, host.ID)
		if err != nil {
			err = perrors.Wrapf(err, "failed to refresh interface-networks for hostid: %s", host.ID)
			return err
		}

		host.InterfaceNetworks = results
	}

	return nil
}

func (r *ReconcileHost) ReconcileAddresses(client *gophercloud.ServiceClient, instance *starlingxv1beta1.Host, profile *starlingxv1beta1.HostProfileSpec, host *v1info.HostInfo) error {
	updated := false

	if r.IsReconcilerEnabled(manager.Address) == false {
		return nil
	}

	for _, addrInfo := range profile.Addresses {
		_, found := host.FindAddressUUID(addrInfo.Interface, addrInfo.Address, addrInfo.Prefix)
		if found {
			continue
		}

		iface, found := host.FindInterfaceByName(addrInfo.Interface)
		if !found {
			msg := fmt.Sprintf("unable to find interface: %s", addrInfo.Interface)
			return starlingxv1beta1.NewMissingSystemResource(msg)
		}

		opts := addresses.AddressOpts{
			Address:       &addrInfo.Address,
			Prefix:        &addrInfo.Prefix,
			InterfaceUUID: &iface.ID,
		}

		log.Info("creating address", "opts", opts)

		_, err := addresses.Create(client, opts).Extract()
		if err != nil {
			err = perrors.Wrapf(err, "failed to create address: %s",
				common.FormatStruct(opts))
			return err
		}

		r.NormalEvent(instance, common.ResourceCreated,
			"address '%s/%d' has been created", addrInfo.Address, addrInfo.Prefix)

		updated = true
	}

	if updated {
		objects, err := addresses.ListAddresses(client, host.ID)
		if err != nil {
			err = perrors.Wrapf(err, "failed to refresh addresses for hostid: %s",
				host.ID)
			return err
		}

		host.Addresses = objects
	}

	return nil
}

func (r *ReconcileHost) ReconcileRoutes(client *gophercloud.ServiceClient, instance *starlingxv1beta1.Host, profile *starlingxv1beta1.HostProfileSpec, host *v1info.HostInfo) error {
	updated := false

	if r.IsReconcilerEnabled(manager.Route) == false {
		return nil
	}

	for _, routeInfo := range profile.Routes {
		_, found := host.FindRouteUUID(routeInfo.Interface, routeInfo.Network, routeInfo.Prefix)
		if found {
			continue
		}

		iface, found := host.FindInterfaceByName(routeInfo.Interface)
		if !found {
			msg := fmt.Sprintf("unable to find interface: %s", routeInfo.Interface)
			return starlingxv1beta1.NewMissingSystemResource(msg)
		}

		opts := routes.RouteOpts{
			Network:       &routeInfo.Network,
			Prefix:        &routeInfo.Prefix,
			Gateway:       &routeInfo.Gateway,
			InterfaceUUID: &iface.ID,
		}

		if routeInfo.Metric != nil {
			opts.Metric = routeInfo.Metric
		} else {
			metric := routes.DefaultMetric
			opts.Metric = &metric
		}

		log.Info("creating route", "opts", opts)

		_, err := routes.Create(client, opts).Extract()
		if err != nil {
			err = perrors.Wrapf(err, "failed to create route %s",
				common.FormatStruct(opts))
			return err
		}

		r.NormalEvent(instance, common.ResourceCreated,
			"route '%s/%d' via %q has been created",
			routeInfo.Network, routeInfo.Prefix, routeInfo.Gateway)

		updated = true
	}

	if updated {
		objects, err := routes.ListRoutes(client, host.ID)
		if err != nil {
			err = perrors.Wrapf(err, "failed to refresh routes for hostid: %s",
				host.ID)
			return err
		}

		host.Routes = objects
	}

	return nil
}

// ReconcileNetworking is responsible for reconciling the Memory configuration
// of a host resource.
func (r *ReconcileHost) ReconcileNetworking(client *gophercloud.ServiceClient, instance *starlingxv1beta1.Host, profile *starlingxv1beta1.HostProfileSpec, host *v1info.HostInfo) error {
	var err error

	if r.IsReconcilerEnabled(manager.Networking) == false {
		return nil
	}

	if profile.HasWorkerSubFunction() {
		// The system API only supports setting these attributes on nodes
		// that support the compute subfunction.

		// Remove stale routes or routes on addresses that will be updated.
		err := r.ReconcileStaleRoutes(client, instance, profile, host)
		if err != nil {
			return err
		}

		// Remove stale addresses or addresses on interfaces that will be
		// deleted and re-added.
		err = r.ReconcileStaleAddresses(client, instance, profile, host)
		if err != nil {
			return err
		}
	}

	// Remove stale vlans or bond interfaces that will be deleted
	err = r.ReconcileStaleInterfaces(client, instance, profile, host)
	if err != nil {
		return err
	}

	// Update ethernet interfaces
	err = r.ReconcileEthernetInterfaces(client, instance, profile, host)
	if err != nil {
		return err
	}

	// Update/Add bond interfaces
	err = r.ReconcileBondInterfaces(client, instance, profile, host)
	if err != nil {
		return err
	}

	// Update/Add vlan interfaces
	err = r.ReconcileVLANInterfaces(client, instance, profile, host)
	if err != nil {
		return err
	}

	if profile.HasWorkerSubFunction() {
		// The system API only supports setting these attributes on nodes
		// that support the compute subfunction.

		// Update/Add addresses
		err = r.ReconcileAddresses(client, instance, profile, host)
		if err != nil {
			return err
		}

		// Update/Add routes
		err = r.ReconcileRoutes(client, instance, profile, host)
		if err != nil {
			return err
		}
	}

	return nil
}
