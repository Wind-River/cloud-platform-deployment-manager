/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019-2022 Wind River Systems, Inc. */

package host

import (
	"fmt"
	"strings"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/addresses"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/interfaceDataNetworks"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/interfaceNetworks"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/interfaces"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/ptpinterfaces"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/routes"
	perrors "github.com/pkg/errors"
	starlingxv1 "github.com/wind-river/cloud-platform-deployment-manager/api/v1"
	utils "github.com/wind-river/cloud-platform-deployment-manager/common"
	"github.com/wind-river/cloud-platform-deployment-manager/controllers/common"
	v1info "github.com/wind-river/cloud-platform-deployment-manager/platform"
)

// findConfiguredBondInterface is a utility function that searches the current
// set of configured interfaces to determine whether current system interface
// still exists in the current configured interface list.  Determine whether
// a system bond interface is the same as a configured bond interface is a bit
// tricky because just about all of its attributes (including its members) can
// be modified without deleting and re-adding.  The best we can do here is to
// look at the member interfaces and if there are any members in common then we
// treat them as the same interface.
func findConfiguredBondInterface(profile *starlingxv1.HostProfileSpec, iface *interfaces.Interface, host *v1info.HostInfo) (*starlingxv1.CommonInterfaceInfo, bool) {
	for _, u := range iface.Uses {
		// Examine each member interface and attempt to find a configured
		// bond interface that has it in its list of members.
		if oldLower, ok := host.FindInterfaceByName(u); ok {
			// We found the interface object for the lower interface so
			// check to see if it is still configured.
			if newLower, found := findConfiguredInterface(oldLower, profile, host); found {
				// The interface is still configured so check to see
				// if its new name happens to be one of the members in one of
				// the configured bond interfaces.
				for _, b := range profile.Interfaces.Bond {
					if utils.ContainsString(b.Members, newLower.Name) {
						// The system and configured interface have at least 1
						// member in common so assume they refer to the same
						// interface.
						return &b.CommonInterfaceInfo, true
					}
				}
			}
		}
	}

	// No equivalent interface was found.  It may now be of a different type
	return nil, false
}

// findConfiguredVLANInterface is a utility function that searches the current
// set of configured interfaces to determine whether current system interface
// still exists in the current configured interface list.  To determine if the
// VLAN interface is still configured we need to match the VID as well as to
// determine if the lower interface is still the same.  Technically, we could
// return true if the VID matches a configured VLAN without considering the
// lower interface but to guarantee consistent processing of the interface,
// address, and route list the code in this module assumes that this function
// ensures that the lower interface is also the same.
func findConfiguredVLANInterface(profile *starlingxv1.HostProfileSpec, iface *interfaces.Interface, host *v1info.HostInfo) (*starlingxv1.CommonInterfaceInfo, bool) {
	// VLAN interfaces are more complicated since the name and vid can
	// remain the same but the interface can be moved over a different
	// lower interface.
	for _, v := range profile.Interfaces.VLAN {
		if iface.VID == nil || v.VID != *iface.VID {
			continue
		}

		// We found a matching VID so repeat the same process on the lower
		// interface to determine if the lower interface is still configured
		// as the same type of interface.  Only
		// return true if that search also succeeds.
		if oldLower, ok := host.FindInterfaceByName(iface.Uses[0]); ok {
			// We found the interface object for the lower interface so
			// check to see if it is still configured.
			if newLower, found := findConfiguredInterface(oldLower, profile, host); found {
				// The interface is still in the configured list so
				// return true as long as the name still matches; otherwise
				// that means that it is now pointing elsewhere.
				if v.Lower == newLower.Name {
					return &v.CommonInterfaceInfo, true
				}
			}
		}
	}

	// No equivalent interface was found.
	return nil, false
}

// findConfiguredVFInterface is a utility function that searches the current
// set of configured interfaces to determine whether current system interface
// still exists in the current configured interface list.  To determine if the
// SRIOV VF interface is still configured we must determine if the lower interface
// is still the same.
func findConfiguredVFInterface(profile *starlingxv1.HostProfileSpec, iface *interfaces.Interface, host *v1info.HostInfo) (*starlingxv1.CommonInterfaceInfo, bool) {
	for _, v := range profile.Interfaces.VF {
		if oldLower, ok := host.FindInterfaceByName(iface.Uses[0]); ok {
			// We found the interface object for the lower interface so
			// check to see if it is still configured.
			if newLower, found := findConfiguredInterface(oldLower, profile, host); found {
				// The interface is still in the configured list so
				// return true as long as the name still matches; otherwise
				// that means that it is now pointing elsewhere.
				if v.Lower == newLower.Name {
					return &v.CommonInterfaceInfo, true
				}
			}
		}
	}

	// No equivalent interface was found.
	return nil, false
}

// findConfiguredEthernetInterface is a utility function that searches the current
// set of configured interfaces to determine whether current system interface
// still exists in the current configured interface list.  Determining whether
// an Ethernet interface is configured is simple since the port names are fixed.
// We only need to ensure that system interface port name and the configured
// interface port names match.
func findConfiguredEthernetInterface(profile *starlingxv1.HostProfileSpec, iface *interfaces.Interface, host *v1info.HostInfo) (*starlingxv1.CommonInterfaceInfo, bool) {
	if portname, found := host.FindInterfacePortName(iface.ID); found {
		for _, e := range profile.Interfaces.Ethernet {
			if e.Port.Name == portname {
				return &e.CommonInterfaceInfo, true
			}
		}
	}

	// No equivalent interface was found.
	return nil, false
}

// findConfiguredVirtualInterface is a utility function that searches the current
// set of configured interfaces to determine whether current system interface
// still exists in the current configured interface list.  Determining whether
// an Virtual interface is configured is simple since the interface names are
// not allowed to change.
func findConfiguredVirtualInterface(profile *starlingxv1.HostProfileSpec, iface *interfaces.Interface) (*starlingxv1.CommonInterfaceInfo, bool) {
	for _, e := range profile.Interfaces.Ethernet {
		if e.Name == iface.Name {
			return &e.CommonInterfaceInfo, true
		}
	}

	// No equivalent interface was found.
	return nil, false
}

// findConfiguredInterface is a utility function that searches the current set
// of configured interfaces to determine whether current system interface still
// exists in the current configured interface list.
func findConfiguredInterface(iface *interfaces.Interface, profile *starlingxv1.HostProfileSpec, host *v1info.HostInfo) (*starlingxv1.CommonInterfaceInfo, bool) {
	switch iface.Type {
	case interfaces.IFTypeEthernet:
		return findConfiguredEthernetInterface(profile, iface, host)

	case interfaces.IFTypeVLAN:
		return findConfiguredVLANInterface(profile, iface, host)

	case interfaces.IFTypeAE:
		return findConfiguredBondInterface(profile, iface, host)

	case interfaces.IFTypeVirtual:
		// These are never deleted or renamed.
		return findConfiguredVirtualInterface(profile, iface)

	case interfaces.IFTypeVF:
		return findConfiguredVFInterface(profile, iface, host)

	default:
		logHost.Info(fmt.Sprintf("unexpected interface type: %s", iface.Type))
		return nil, false
	}

}

// findConfiguredRoute is a utility function that searches the current set of
// configured routes to determine whether current system route still exists in
// the current configured route list.
func findConfiguredRoute(route routes.Route, profile *starlingxv1.HostProfileSpec) (*starlingxv1.RouteInfo, bool) {
	for _, x := range profile.Routes {
		if x.Interface == route.InterfaceName &&
			strings.EqualFold(x.Network, route.Network) &&
			x.Prefix == route.Prefix &&
			strings.EqualFold(x.Gateway, route.Gateway) &&
			(x.Metric == nil || *x.Metric == route.Metric) {
			// Routes cannot be updated so all fields must match
			// otherwise re-provisioning is required.
			return &x, true
		}
	}

	return nil, false
}

// ReconcileStaleRoutes examines the current set of routes and deletes any
// routes that are stale or need to be re-provisioned.  A route needs to be
// deleted if:
//
//	A) The system route does not have an equivalent configured entry
//	B) The configured route has moved to a different underlying interface
//	C) The underlying interface needs to be deleted and re-added.
func (r *HostReconciler) ReconcileStaleRoutes(client *gophercloud.ServiceClient, instance *starlingxv1.Host, profile *starlingxv1.HostProfileSpec, host *v1info.HostInfo) error {
	updated := false

	if !utils.IsReconcilerEnabled(utils.Route) {
		return nil
	}

	for _, route := range host.Routes {
		remove := false

		if oldLower, found := host.FindInterfaceByName(route.InterfaceName); !found {
			// We could not identify the interface for this route.  This is
			// unexpected so force the deletion of the route so that we can
			// correct any latent issues.
			remove = true
		} else if newAddress, found := findConfiguredRoute(route, profile); !found {
			// We could not find an equivalent route in the current configured
			// list so remove this entry.
			remove = true
		} else {
			// We found an equivalent route but we need to make sure that it
			// is still configured over the same interface.  To do this we look
			// for a matching interface in the configured list for the route's
			// current lower interface.
			if newLower, found := findConfiguredInterface(oldLower, profile, host); !found {
				// The old lower is no longer configured so this route
				// definitely needs to be deleted.
				remove = true
			} else if newLower.Name != newAddress.Interface {
				// The old lower is still configured, but the new route is
				// configured over a completely different interface so it will
				// need to be moved.
				remove = true
			}
		}

		if remove {
			logHost.Info("deleting route", "uuid", route.ID)

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

// findConfiguredAddress is a utility function that searches the current set of
// configured addresses to determine whether current system address still exists
// in the current configured address list.
func findConfiguredAddress(addr addresses.Address, profile *starlingxv1.HostProfileSpec) (*starlingxv1.AddressInfo, bool) {
	for _, x := range profile.Addresses {
		if strings.EqualFold(x.Address, addr.Address) && x.Prefix == addr.Prefix {
			// Addresses cannot be updated so unless both the address
			// and prefix match we consider it to require
			// re-provisioning.
			return &x, true
		}
	}

	return nil, false
}

// ReconcileStalePTPInstance examines the current set of PTP instances and deletes
// any PTP instances that are stale or need to be re-provisioned.
func (r *HostReconciler) ReconcileStalePTPInterfaces(client *gophercloud.ServiceClient, instance *starlingxv1.Host, profile *starlingxv1.HostProfileSpec, host *v1info.HostInfo) error {

	if !utils.IsReconcilerEnabled(utils.Interface) {
		return nil
	}

	for _, iface := range host.Interfaces {
		// ReconcileStaleInterfaces is assumed to have been invoked before this
		// method therefore ignore any failures to find a configured interface
		// since they would have been already deleted in a previous step.

		if info, found := findConfiguredInterface(&iface, profile, host); found {
			if info.PtpInterfaces == nil {
				// The user did not specify any ptp interfaces (an empty list or
				// otherwise) so accept whatever is currently provisioned as
				// the desired list.
				continue
			}

			// Get the lists of current and configured networks on this interface
			current := host.FindPTPInterfaceNameByInterface(iface)

			configured := starlingxv1.PtpInterfaceItemListToStrings(*info.PtpInterfaces)

			// Diff the lists to determine if changes need to be applied
			_, removed, _ := utils.ListDelta(current, configured)

			for _, single := range removed {
				found, err := findPTPinterfaceByName(client, single)
				if err != nil {
					return err
				}

				opts := ptpinterfaces.PTPIntToIntOpt{
					PTPinterfaceID: &found.ID,
				}
				// Remove the stale PTP interface
				logHost.Info("deleting stale PTP interface from interface", "ifname", iface.Name)
				_, err = ptpinterfaces.RemovePTPIntFromInt(client, iface.ID, opts).Extract()
				if err != nil {
					err = perrors.Wrapf(err, "failed to remove stale PTP interface %q from iface %q",
						configured, iface.Name)
					return err
				}

				r.NormalEvent(instance, common.ResourceDeleted,
					"stale PTP interface %q has been removed from %q",
					configured, iface.Name)
			}
		}
	}
	return nil
}

// ReconcileStaleAddresses examines the current set of addresses and deletes
// any addresses that are stale or need to be re-provisioned.  An address needs
// to be deleted if:
//
//	A) The system address does not have an equivalent configured entry
//	B) The configured address has moved to a different underlying interface
//	C) The underlying interface needs to be deleted and re-added.
func (r *HostReconciler) ReconcileStaleAddresses(client *gophercloud.ServiceClient, instance *starlingxv1.Host, profile *starlingxv1.HostProfileSpec, host *v1info.HostInfo) error {
	updated := false

	if !utils.IsReconcilerEnabled(utils.Address) {
		return nil
	}

	for _, addr := range host.Addresses {
		if addr.PoolUUID != nil {
			// Automatically assigned addresses should be ignored.  The system
			// will remove them when the pool is removed from the interface.
			continue
		}

		remove := false

		if oldLower, found := host.FindInterfaceByName(addr.InterfaceName); !found {
			// We could not identify the interface for this address.  This is
			// unexpected so force the deletion of the address so that we can
			// correct any latent issues.
			remove = true
		} else if newAddress, found := findConfiguredAddress(addr, profile); !found {
			// We could not find an equivalent address in the current configured
			// list so remove this entry.
			remove = true
		} else {
			// We found an equivalent address but we need to make sure that it
			// is still configured over the same interface.  To do this we look
			// for a matching interface in the configured list for the address's
			// current lower interface.
			if newLower, found := findConfiguredInterface(oldLower, profile, host); !found {
				// The old lower is no longer configured so this address
				// definitely needs to be deleted.
				remove = true
			} else if newLower.Name != newAddress.Interface {
				// The old lower is still configured, but the new address is
				// configured over a completely different interface so it will
				// need to be moved.
				remove = true
			}
		}

		if remove {
			logHost.Info("deleting address", "uuid", addr.ID)

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
//
//	A) it no longer exists in the list of interfaces to be configured
//	B) it still exists as a VLAN, but has a different vlan-id value or lower
//	   interface
//	C) it still exists as a Bond, but has no members in common with the system
//	   interface.
//	D) it still exists as a VF, but has a different lower interface.
func (r *HostReconciler) ReconcileStaleInterfaces(client *gophercloud.ServiceClient, instance *starlingxv1.Host, profile *starlingxv1.HostProfileSpec, host *v1info.HostInfo) error {
	updated := false

	if !utils.IsReconcilerEnabled(utils.Interface) {
		return nil
	}

	for _, iface := range host.Interfaces {
		if iface.Type == interfaces.IFTypeEthernet || iface.Type == interfaces.IFTypeVirtual {
			continue
		}

		if _, found := findConfiguredInterface(&iface, profile, host); !found {
			// This interface either no longer exists or has changed in a way
			// that required re-provisioning.

			logHost.Info("deleting interface", "uuid", iface.ID)

			err := interfaces.Delete(client, iface.ID).ExtractErr()
			if err != nil {
				err = perrors.Wrapf(err, "failed to delete interface %s", iface.ID)
				return err
			}

			r.NormalEvent(instance, common.ResourceDeleted,
				"stale interface %q has been deleted", iface.Name)

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

// ReconcileStaleInterfaceNetworks will examine the current set of system
// interfaces and determine if any interface-network associations need to be
// removed from any interface.  This step is critical to proper reconciliation
// of interfaces because when moving a network from one interface to another
// it cannot be added to the new interface until it has been removed from the
// old network.  The only way to accomplish that is to execute this in two
// phases.  The first phase is handled by this method and deletes old
// associations.  The second phase is handled by Reconcile*Interfaces and adds
// missing associations.
func (r *HostReconciler) ReconcileStaleInterfaceNetworks(client *gophercloud.ServiceClient, instance *starlingxv1.Host, profile *starlingxv1.HostProfileSpec, host *v1info.HostInfo) error {
	updated := false

	if !utils.IsReconcilerEnabled(utils.Interface) {
		return nil
	}

	for _, iface := range host.Interfaces {
		// ReconcileStaleInterfaces is assumed to have been invoked before this
		// method therefore ignore any failures to find a configured interface
		// since they would have been already deleted in a previous step.

		if info, found := findConfiguredInterface(&iface, profile, host); found {
			if info.PlatformNetworks == nil {
				// The user did not specify any networks (an empty list or
				// otherwise) so accept whatever is currently provisioned as
				// the desired list.
				continue
			}

			// Get the lists of current and configured networks on this interface
			current := host.BuildInterfaceNetworkList(iface)
			configured := starlingxv1.PlatformNetworkItemListToStrings(*info.PlatformNetworks)

			// Diff the lists to determine what changes need to be applied
			_, removed, _ := utils.ListDelta(current, configured)

			for _, name := range removed {
				if id, ok := host.FindInterfaceNetworkID(iface, name); ok {
					logHost.Info("deleting stale interface-network from interface", "ifname",
						iface.Name, "id", id)

					err := interfaceNetworks.Delete(client, id).ExtractErr()
					if err != nil {
						err = perrors.Wrapf(err, "failed to delete stale interface-network %q from iface %q",
							id, iface.Name)
						return err
					}

					updated = true

					r.NormalEvent(instance, common.ResourceDeleted,
						"stale interface-network %q has been removed from %q",
						id, iface.Name)
				} else {
					msg := fmt.Sprintf("unable to find interface-network id for network %q on interface %q",
						name, iface.Name)
					return starlingxv1.NewMissingSystemResource(msg)
				}
			}
		}
	}

	if updated {
		results, err := interfaceNetworks.ListInterfaceNetworks(client, host.ID)
		if err != nil {
			err = perrors.Wrapf(err, "failed to refresh interface-networks on hostid %s",
				host.ID)
			return err
		}

		host.InterfaceNetworks = results
	}

	return nil
}

// ReconcileStaleInterfaceDataNetworks will examine the current set of system
// interfaces and determine if any interface-network associations need to be
// removed from any interface.  This step is critical to proper reconciliation
// of interfaces because when moving a network from one interface to another
// it cannot be added to the new interface until it has been removed from the
// old network.  The only way to accomplish that is to execute this in two
// phases.  The first phase is handled by this method and deletes old
// associations.  The second phase is handled by Reconcile*Interfaces and adds
// missing associations.
func (r *HostReconciler) ReconcileStaleInterfaceDataNetworks(client *gophercloud.ServiceClient, instance *starlingxv1.Host, profile *starlingxv1.HostProfileSpec, host *v1info.HostInfo) error {
	updated := false

	if !utils.IsReconcilerEnabled(utils.Interface) {
		return nil
	}

	for _, iface := range host.Interfaces {
		// ReconcileStaleInterfaces is assumed to have been invoked before this
		// method therefore ignore any failures to find a configured interface
		// since they would have been already deleted in a previous step.

		if info, found := findConfiguredInterface(&iface, profile, host); found {
			if info.DataNetworks == nil {
				// The user did not specify any networks (an empty list or
				// otherwise) so accept whatever is currently provisioned as
				// the desired list.
				continue
			}

			// Get the lists of current and configured networks on this interface
			current := host.BuildInterfaceDataNetworkList(iface)
			configured := starlingxv1.DataNetworkItemListToStrings(*info.DataNetworks)

			// Diff the lists to determine what changes need to be applied
			_, removed, _ := utils.ListDelta(current, configured)

			for _, name := range removed {
				if id, ok := host.FindInterfaceDataNetworkID(iface, name); ok {
					logHost.Info("deleting stale interface-network from interface",
						"ifname", iface.Name, "id", id)

					err := interfaceDataNetworks.Delete(client, id).ExtractErr()
					if err != nil {
						err = perrors.Wrapf(err, "failed to delete stale interface-datanetwork %q from iface %q",
							id, iface.Name)
						return err
					}

					updated = true

					r.NormalEvent(instance, common.ResourceDeleted,
						"stale interface-datanetwork %q has been removed from %q",
						id, iface.Name)
				} else {
					msg := fmt.Sprintf("unable to find interface-datanetwork id for network %q on interface %q",
						name, iface.Name)
					return starlingxv1.NewMissingSystemResource(msg)
				}
			}
		}
	}

	if updated {
		results, err := interfaceDataNetworks.ListInterfaceDataNetworks(client, host.ID)
		if err != nil {
			err = perrors.Wrapf(err, "failed to refresh interface-datanetworks on hostid %s",
				host.ID)
			return err
		}

		host.InterfaceDataNetworks = results
	}

	return nil
}

// hasIPv4StaticAddresses is a utility function which determines if an interface
// has any configured static addresses.
func hasIPv4StaticAddresses(info starlingxv1.CommonInterfaceInfo, profile *starlingxv1.HostProfileSpec) bool {
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
func hasIPv6StaticAddresses(info starlingxv1.CommonInterfaceInfo, profile *starlingxv1.HostProfileSpec) bool {
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
//
//	because it assumes that the presence of a pool is enough to determine if
//	it is dynamic rather than looking for a network to determine if that network
//	is dynamic or not (which is currently not possible for networks used for
//	data interfaces).
func hasIPv4DynamicAddresses(info starlingxv1.CommonInterfaceInfo, host *v1info.HostInfo) (*string, bool) {
	if info.PlatformNetworks == nil {
		return nil, false
	}

	networks := starlingxv1.PlatformNetworkItemListToStrings(*info.PlatformNetworks)
	for _, networkName := range networks {
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
//
//	because it assumes that the presence of a pool is enough to determine if
//	it is dynamic rather than looking for a network to determine if that network
//	is dynamic or not (which is currently not possible for networks used for
//	data interfaces).
func hasIPv6DynamicAddresses(info starlingxv1.CommonInterfaceInfo, host *v1info.HostInfo) (*string, bool) {
	if info.PlatformNetworks == nil {
		return nil, false
	}

	networks := starlingxv1.PlatformNetworkItemListToStrings(*info.PlatformNetworks)
	for _, networkName := range networks {
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
func getInterfaceIPv4Addressing(info starlingxv1.CommonInterfaceInfo, profile *starlingxv1.HostProfileSpec, host *v1info.HostInfo) (mode string, pool *string) {
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
func getInterfaceIPv6Addressing(info starlingxv1.CommonInterfaceInfo, profile *starlingxv1.HostProfileSpec, host *v1info.HostInfo) (mode string, pool *string) {
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
func interfaceUpdateRequired(info starlingxv1.CommonInterfaceInfo, iface *interfaces.Interface, profile *starlingxv1.HostProfileSpec, host *v1info.HostInfo) (opts interfaces.InterfaceOpts, result bool) {
	if iface.Type != interfaces.IFTypeVirtual {
		// Only allow name changes for non-virtual interfaces.  That is, never
		// allow "lo" to be renamed.
		if info.Name != iface.Name {
			opts.Name = &info.Name
			result = true
		}
	}

	if !strings.EqualFold(info.Class, iface.Class) {
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

	if (info.PTPRole == nil && iface.PTPRole != nil) || (info.PTPRole != nil && iface.PTPRole == nil) || (info.PTPRole != nil && *info.PTPRole != *iface.PTPRole) {
		opts.PTPRole = info.PTPRole
		result = true
	}

	if info.Class == interfaces.IFClassData || hasIPv4StaticAddresses(info, profile) || hasIPv6StaticAddresses(info, profile) {
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

// sriovUpdateRequired is a utility function which determines whether the
// SRIOV specific interface attributes have changed and if so fills in the
// opts struct with the values that must be passed to the system API.
func sriovUpdateRequired(ethInfo starlingxv1.EthernetInfo, iface *interfaces.Interface, profile *starlingxv1.HostProfileSpec, host *v1info.HostInfo) (opts interfaces.InterfaceOpts, result bool) {
	var ok bool

	if opts, ok = interfaceUpdateRequired(ethInfo.CommonInterfaceInfo, iface, profile, host); ok {
		result = true
	}

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

		// Ensure that SRIOV VF driver is up to date.  There is no way to set
		// this back to the default value of None so just ensure that it is
		// set to the desired value if one is requested.
		if ethInfo.VFDriver != nil {
			if iface.VFDriver == nil {
				opts.VFDriver = ethInfo.VFDriver
				result = true
			} else if *ethInfo.VFDriver != *iface.VFDriver {
				opts.VFDriver = ethInfo.VFDriver
				result = true
			}
		}
	}

	return opts, result
}

// ReconcileInterfaceNetworks implements a method to reconcile the list of
// networks on an interface against the configured set of networks.
func (r *HostReconciler) ReconcileInterfaceNetworks(client *gophercloud.ServiceClient, instance *starlingxv1.Host, info starlingxv1.CommonInterfaceInfo, iface interfaces.Interface, host *v1info.HostInfo) (updated bool, err error) {
	if info.PlatformNetworks == nil {
		return updated, err
	}

	// Get the lists of current and configured networks on this interface
	current := host.BuildInterfaceNetworkList(iface)
	configured := starlingxv1.PlatformNetworkItemListToStrings(*info.PlatformNetworks)

	// Diff the lists to determine what changes need to be applied
	added, removed, _ := utils.ListDelta(current, configured)

	for _, name := range removed {
		if id, ok := host.FindInterfaceNetworkID(iface, name); ok {
			logHost.Info("deleting interface-network from interface", "ifname", iface.Name, "id", id)

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
			return updated, starlingxv1.NewMissingSystemResource(msg)
		}
	}

	for _, name := range added {
		if id, ok := host.FindNetworkID(name); ok {
			opts := interfaceNetworks.InterfaceNetworkOpts{
				InterfaceUUID: iface.ID,
				NetworkUUID:   id,
			}

			logHost.Info("creating an interface-network association", "ifname", iface.Name, "network", name)

			_, err := interfaceNetworks.Create(client, opts).Extract()
			if err != nil {
				err = perrors.Wrapf(err, "failed to create interface-network association: %s",
					common.FormatStruct(opts))
				return updated, err
			}

			updated = true

			r.NormalEvent(instance, common.ResourceCreated,
				"interface-network %q has been created on %q", name, iface.Name)
		} else {
			// This is ignored because at the moment not all networks are
			// created equally.  Some networks are of type "other" and these
			// are address pools only so they do not show up in the list of
			// networks and are handled elsewhere.  Because of this we cannot
			// tell (easily) whether this condition is normal or an error.
			continue
		}
	}

	return updated, err
}

// findPTPInterfaceByName is to search for a PTP interface by its name,
// this instance may or may not associate with the current host/interface.
func findPTPinterfaceByName(client *gophercloud.ServiceClient, name string) (*ptpinterfaces.PTPInterface, error) {
	founds, err := ptpinterfaces.ListPTPInterfaces(client)
	if err != nil {
		return nil, err
	}
	for _, found := range founds {
		if found.Name == name {
			return &found, nil
		}
	}
	return nil, nil
}

// ReconcilePTPInterface implements a method to reconcile the PTP interface on
// an interface against the configured set of networks.
func (r *HostReconciler) ReconcilePTPInterface(client *gophercloud.ServiceClient, instance *starlingxv1.Host, info starlingxv1.CommonInterfaceInfo, iface interfaces.Interface, host *v1info.HostInfo) (updated bool, err error) {
	if info.PtpInterfaces == nil {
		return updated, nil
	}

	// Get the current and configured PTP interface on this interface
	current := host.FindPTPInterfaceNameByInterface(iface)
	configured := starlingxv1.PtpInterfaceItemListToStrings(*info.PtpInterfaces)

	// Diff the lists to determine if changes need to be applied
	added, removed, _ := utils.ListDelta(current, configured)

	if len(removed) > 0 {
		for _, single := range removed {
			found, err := findPTPinterfaceByName(client, single)
			if err != nil {
				return updated, err
			}
			opts := ptpinterfaces.PTPIntToIntOpt{
				PTPinterfaceID: &found.ID,
			}
			// Remove the PTP interface not expected
			logHost.Info("deleting stale PTP interface from interface", "ifname", iface.Name)
			_, err = ptpinterfaces.RemovePTPIntFromInt(client, iface.ID, opts).Extract()
			if err != nil {
				err = perrors.Wrapf(err, "failed to remove stale PTP interface %q from iface %q",
					configured, iface.Name)
				return updated, err
			}
			updated = true
			r.NormalEvent(instance, common.ResourceDeleted,
				"PTP interface %q has been removed from %q", single, iface.Name)
		}
	}

	if len(added) > 0 {
		for _, single := range added {
			found, err := findPTPinterfaceByName(client, single)
			if found == nil {
				return updated, common.NewResourceStatusDependency("PTP interface is not created, waiting for the creation")
			}
			if err != nil {
				return updated, err
			}
			opts := ptpinterfaces.PTPIntToIntOpt{
				PTPinterfaceID: &found.ID,
			}

			logHost.Info("adding PTP interface to interface", "ifname", iface.Name)

			_, err = ptpinterfaces.AddPTPIntToInt(client, iface.ID, opts).Extract()
			if err != nil {
				err = perrors.Wrapf(err, "failed to add PTP interface %q from iface %q",
					configured, iface.Name)
				return updated, err
			}

			updated = true
			r.NormalEvent(instance, common.ResourceCreated,
				"PTP interface %q has been added to %q", single, iface.Name)
		}
	}
	return updated, err
}

// ReconcileInterfaceDataNetworks implements a method to reconcile the list of
// networks on an interface against the configured set of networks.
func (r *HostReconciler) ReconcileInterfaceDataNetworks(client *gophercloud.ServiceClient, instance *starlingxv1.Host, info starlingxv1.CommonInterfaceInfo, iface interfaces.Interface, host *v1info.HostInfo) (updated bool, err error) {
	if info.DataNetworks == nil {
		return updated, err
	}

	// Get the lists of current and configured networks on this interface
	current := host.BuildInterfaceDataNetworkList(iface)
	configured := starlingxv1.DataNetworkItemListToStrings(*info.DataNetworks)

	// Diff the lists to determine what changes need to be applied
	added, removed, _ := utils.ListDelta(current, configured)

	for _, name := range removed {
		if id, ok := host.FindInterfaceDataNetworkID(iface, name); ok {
			logHost.Info("deleting interface-datanetwork from interface", "ifname", iface.Name, "id", id)

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
			return updated, starlingxv1.NewMissingSystemResource(msg)
		}
	}

	for _, name := range added {
		if id, ok := host.FindDataNetworkID(name); ok {
			opts := interfaceDataNetworks.InterfaceDataNetworkOpts{
				InterfaceUUID:   iface.ID,
				DataNetworkUUID: id,
			}

			logHost.Info("creating an interface-datanetwork association", "ifname", iface.Name, "network", name)

			_, err := interfaceDataNetworks.Create(client, opts).Extract()
			if err != nil {
				err = perrors.Wrapf(err, "failed to create interface-datanetwork association: %s",
					common.FormatStruct(opts))
				return updated, err
			}

			updated = true

			r.NormalEvent(instance, common.ResourceCreated,
				"interface-datanetwork %q has been created on %q", name, iface.Name)
		} else {
			msg := fmt.Sprintf("unable to find interface-datanetwork id for network %q", name)
			return updated, starlingxv1.NewMissingSystemResource(msg)
		}
	}

	return updated, err
}

// ReconcileEthernetInterfaces will update system interfaces to align with the
// desired configuration.  It is assumed that the configuration will apply;
// meaning that prior to invoking this function stale interfaces and stale
// interface configurations have been resolved so that the intended list of
// ethernet interface configuration will apply cleanly here.
func (r *HostReconciler) ReconcileEthernetInterfaces(client *gophercloud.ServiceClient, instance *starlingxv1.Host, profile *starlingxv1.HostProfileSpec, host *v1info.HostInfo) error {
	var iface *interfaces.Interface
	var ifuuid string
	var found bool

	if !utils.IsReconcilerEnabled(utils.Interface) {
		return nil
	}

	if profile.Interfaces == nil || len(profile.Interfaces.Ethernet) == 0 {
		return nil
	}

	updated := false
	networksUpdated := false
	dataNetworksUpdated := false
	ptpInterfaceUpdated := false

	for _, ethInfo := range profile.Interfaces.Ethernet {
		// SRIOV interfaces are deferred until later to avoid interfering with
		// platform interface configuration.
		if ethInfo.CommonInterfaceInfo.Class == interfaces.IFClassPCISRIOV {
			continue
		}

		// For each configured ethernet interface update the associated system
		// resource.
		if ethInfo.Lower != "" {
			// if this is a ethernet created on top of a ethernet
			_, found = host.FindInterfaceByName(ethInfo.Lower)
			if !found {
				msg := fmt.Sprintf("unable to find lower interface UUID for ethernet: %s", ethInfo.Name)
				return starlingxv1.NewMissingSystemResource(msg)
			}

			iface, found = host.FindInterfaceByName(ethInfo.Name)
			if !found {
				// Create the interface
				opts := commonInterfaceOptions(ethInfo.CommonInterfaceInfo, profile, host)

				iftype := interfaces.IFTypeEthernet
				opts.Type = &iftype
				uses := []string{ethInfo.Lower}
				opts.Uses = &uses

				logHost.Info("creating ethernet interface", "opts", opts)

				new_iface, err := interfaces.Create(client, opts).Extract()
				iface = new_iface
				if err != nil {
					err = perrors.Wrapf(err, "failed to create ethernet interface: %s",
						common.FormatStruct(opts))
					return err
				}

				r.NormalEvent(instance, common.ResourceCreated,
					"ethernet interface %q has been created", ethInfo.Name)

				updated = true
			} else {
				ifuuid = iface.ID
				if opts, ok := interfaceUpdateRequired(ethInfo.CommonInterfaceInfo, iface, profile, host); ok {
					logHost.Info("updating ethernet interface", "uuid", ifuuid, "opts", opts)

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
			}
		} else {
			if ethInfo.Name != interfaces.LoopbackInterfaceName {
				ifuuid, found = host.FindPortInterfaceUUID(ethInfo.Port.Name)
				if !found {
					msg := fmt.Sprintf("unable to find interface UUID for port: %s", ethInfo.Port.Name)
					return starlingxv1.NewMissingSystemResource(msg)
				}

				iface, found = host.FindInterface(ifuuid)
				if !found {
					msg := fmt.Sprintf("unable to find interface: %s", ifuuid)
					return starlingxv1.NewMissingSystemResource(msg)
				}
			} else {
				iface, found = host.FindInterfaceByName(interfaces.LoopbackInterfaceName)
				if !found {
					msg := fmt.Sprintf("unable to find loopback interface: %s",
						interfaces.LoopbackInterfaceName)
					return starlingxv1.NewMissingSystemResource(msg)
				}

				ifuuid = iface.ID
			}

			if opts, ok := interfaceUpdateRequired(ethInfo.CommonInterfaceInfo, iface, profile, host); ok {
				logHost.Info("updating ethernet interface", "uuid", ifuuid, "opts", opts)

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
		}

		result, err := r.ReconcileInterfaceNetworks(client, instance, ethInfo.CommonInterfaceInfo, *iface, host)
		if err != nil {
			return err
		}

		networksUpdated = networksUpdated || result

		result, err = r.ReconcileInterfaceDataNetworks(client, instance, ethInfo.CommonInterfaceInfo, *iface, host)
		if err != nil {
			return err
		}

		dataNetworksUpdated = dataNetworksUpdated || result

		result, err = r.ReconcilePTPInterface(client, instance, ethInfo.CommonInterfaceInfo, *iface, host)
		if err != nil {
			return err
		}

		ptpInterfaceUpdated = ptpInterfaceUpdated || result

		updated = updated || networksUpdated || dataNetworksUpdated || ptpInterfaceUpdated
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
	}

	if networksUpdated {
		// Interface network associations have been updated so we need to
		// refresh the list of interface-network associations.
		objects, err := interfaceNetworks.ListInterfaceNetworks(client, host.ID)
		if err != nil {
			err = perrors.Wrapf(err, "failed to refresh interface-networks for hostid: %s", host.ID)
			return err
		}

		host.InterfaceNetworks = objects
	}

	if dataNetworksUpdated {
		// Interface data network associations have been updated so we need to
		// refresh the list of interface-datanetwork associations.
		objects, err := interfaceDataNetworks.ListInterfaceDataNetworks(client, host.ID)
		if err != nil {
			err = perrors.Wrapf(err, "failed to refresh interface-datanetworks for hostid: %s", host.ID)
			return err
		}

		host.InterfaceDataNetworks = objects
	}

	if ptpInterfaceUpdated {
		// PTP interface associated with the interface have been updated so we
		// need to refresh the list of interface-datanetwork associations.
		objects, err := ptpinterfaces.ListHostPTPInterfaces(client, host.ID)
		if err != nil {
			err = perrors.Wrapf(err, "failed to refresh PTP interface for hostid: %s", host.ID)
			return err
		}

		host.PTPInterfaces = objects
	}

	return nil
}

// bondUpdateRequired is a utility function which determines whether the
// bond specificc interface attributes have changed and if so fills in the opts
// struct with the values that must be passed to the system API.
func bondUpdateRequired(bond starlingxv1.BondInfo, iface *interfaces.Interface, profile *starlingxv1.HostProfileSpec, host *v1info.HostInfo) (opts interfaces.InterfaceOpts, result bool) {
	var ok bool

	if opts, ok = interfaceUpdateRequired(bond.CommonInterfaceInfo, iface, profile, host); ok {
		result = true
	}

	if iface.AEMode != nil && !strings.EqualFold(bond.Mode, *iface.AEMode) {
		opts.AEMode = &bond.Mode
		result = true
	}

	if bond.TransmitHashPolicy != nil {
		if iface.AETransmitHash != nil && !strings.EqualFold(*bond.TransmitHashPolicy, *iface.AETransmitHash) {
			opts.AETransmitHash = bond.TransmitHashPolicy
			result = true
		}
	}

	if bond.PrimaryReselect != nil {
		if iface.AEPrimReselect != nil && !strings.EqualFold(*bond.PrimaryReselect, *iface.AEPrimReselect) {
			opts.AEPrimReselect = bond.PrimaryReselect
			result = true
		}
	}

	if utils.ListChanged(bond.Members, iface.Uses) {
		// The system API handles "uses" inconsistently between create and
		// update.  It requires a different attribute name for both.  This
		// looks like it is because the system API uses an old copy of ironic
		// as its bases and the json patch code does not support lists.
		opts.UsesModify = &bond.Members
		result = true
	}

	return opts, result
}

// commonInterfaceOptions is a utility to populate the interface options for
// all common interface attributes.
func commonInterfaceOptions(info starlingxv1.CommonInterfaceInfo, profile *starlingxv1.HostProfileSpec, host *v1info.HostInfo) interfaces.InterfaceOpts {
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

func (r *HostReconciler) ReconcileBondInterfaces(client *gophercloud.ServiceClient, instance *starlingxv1.Host, profile *starlingxv1.HostProfileSpec, host *v1info.HostInfo) (err error) {
	var iface *interfaces.Interface

	if !utils.IsReconcilerEnabled(utils.Interface) {
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
			opts.AEPrimReselect = bondInfo.PrimaryReselect

			opts.Uses = &bondInfo.Members

			logHost.Info("creating bond interface", "opts", opts)

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
				return starlingxv1.NewMissingSystemResource(msg)
			}

			if opts, ok := bondUpdateRequired(bondInfo, iface, profile, host); ok {
				logHost.Info("updating bond interface", "uuid", ifuuid, "opts", opts)

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

		ptpInterfaceUpdated, err := r.ReconcilePTPInterface(client, instance, bondInfo.CommonInterfaceInfo, *iface, host)
		if err != nil {
			return err
		}

		updated = updated || networksUpdated || dataNetworksUpdated || ptpInterfaceUpdated
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

func (r *HostReconciler) ReconcileVLANInterfaces(client *gophercloud.ServiceClient, instance *starlingxv1.Host, profile *starlingxv1.HostProfileSpec, host *v1info.HostInfo) (err error) {
	var iface *interfaces.Interface

	if !utils.IsReconcilerEnabled(utils.Interface) {
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

			logHost.Info("creating vlan interface", "opts", opts)

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
				return starlingxv1.NewMissingSystemResource(msg)
			}

			if opts, ok := interfaceUpdateRequired(vlanInfo.CommonInterfaceInfo, iface, profile, host); ok {
				logHost.Info("updating vlan interface", "uuid", ifuuid, "opts", opts)

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

		ptpInterfaceUpdated, err := r.ReconcilePTPInterface(client, instance, vlanInfo.CommonInterfaceInfo, *iface, host)
		if err != nil {
			return err
		}

		updated = updated || networksUpdated || dataNetworksUpdated || ptpInterfaceUpdated
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

		// Delete current defaults so that it will obtain the latest info
		logHost.Info("vlan updated. Remove defaults")
		instance.Status.Defaults = nil
	}

	return nil
}

// ReconcileSRIOVInterfaces will update system interfaces to align with the
// desired configuration.  It is assumed that the configuration will apply;
// meaning that prior to invoking this function stale interfaces and stale
// interface configurations have been resolved so that the intended list of
// ethernet interface configuration will apply cleanly here.
func (r *HostReconciler) ReconcileSRIOVInterfaces(client *gophercloud.ServiceClient, instance *starlingxv1.Host, profile *starlingxv1.HostProfileSpec, host *v1info.HostInfo) error {
	var iface *interfaces.Interface
	var ifuuid string
	var found bool

	if !utils.IsReconcilerEnabled(utils.Interface) {
		return nil
	}

	if profile.Interfaces == nil || len(profile.Interfaces.Ethernet) == 0 {
		return nil
	}

	updated := false
	networksUpdated := false
	dataNetworksUpdated := false
	ptpInterfaceUpdated := false

	for _, ethInfo := range profile.Interfaces.Ethernet {
		// Only processing SRIOV Ethernet interfaces
		if ethInfo.CommonInterfaceInfo.Class != interfaces.IFClassPCISRIOV {
			continue
		}

		ifuuid, found = host.FindPortInterfaceUUID(ethInfo.Port.Name)
		if !found {
			msg := fmt.Sprintf("unable to find interface UUID for port: %s", ethInfo.Port.Name)
			return starlingxv1.NewMissingSystemResource(msg)
		}

		iface, found = host.FindInterface(ifuuid)
		if !found {
			msg := fmt.Sprintf("unable to find interface: %s", ifuuid)
			return starlingxv1.NewMissingSystemResource(msg)
		}

		if opts, ok := sriovUpdateRequired(ethInfo, iface, profile, host); ok {
			logHost.Info("updating sriov ethernet interface", "uuid", ifuuid, "opts", opts)

			_, err := interfaces.Update(client, ifuuid, opts).Extract()
			if err != nil {
				err = perrors.Wrapf(err, "failed to update interface: %s, %s",
					ifuuid, common.FormatStruct(opts))
				return err
			}

			r.NormalEvent(instance, common.ResourceUpdated,
				"ethernet sriov interface %q has been updated", ethInfo.Name)

			updated = true
		}

		result, err := r.ReconcileInterfaceNetworks(client, instance, ethInfo.CommonInterfaceInfo, *iface, host)
		if err != nil {
			return err
		}

		networksUpdated = networksUpdated || result

		result, err = r.ReconcileInterfaceDataNetworks(client, instance, ethInfo.CommonInterfaceInfo, *iface, host)
		if err != nil {
			return err
		}

		dataNetworksUpdated = dataNetworksUpdated || result

		result, err = r.ReconcilePTPInterface(client, instance, ethInfo.CommonInterfaceInfo, *iface, host)
		if err != nil {
			return err
		}

		ptpInterfaceUpdated = ptpInterfaceUpdated || result

		updated = updated || networksUpdated || dataNetworksUpdated || ptpInterfaceUpdated
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
	}

	if networksUpdated {
		// Interface network associations have been updated so we need to
		// refresh the list of interface-network associations.
		objects, err := interfaceNetworks.ListInterfaceNetworks(client, host.ID)
		if err != nil {
			err = perrors.Wrapf(err, "failed to refresh interface-networks for hostid: %s", host.ID)
			return err
		}

		host.InterfaceNetworks = objects
	}

	if dataNetworksUpdated {
		// Interface data network associations have been updated so we need to
		// refresh the list of interface-datanetwork associations.
		objects, err := interfaceDataNetworks.ListInterfaceDataNetworks(client, host.ID)
		if err != nil {
			err = perrors.Wrapf(err, "failed to refresh interface-datanetworks for hostid: %s", host.ID)
			return err
		}

		host.InterfaceDataNetworks = objects
	}

	if ptpInterfaceUpdated {
		// PTP interface associated with the interface have been updated so we
		// need to refresh the list of interface-datanetwork associations.
		objects, err := ptpinterfaces.ListHostPTPInterfaces(client, host.ID)
		if err != nil {
			err = perrors.Wrapf(err, "failed to refresh PTP interface for hostid: %s", host.ID)
			return err
		}

		host.PTPInterfaces = objects
	}

	return nil
}

func (r *HostReconciler) ReconcileVFInterfaces(client *gophercloud.ServiceClient, instance *starlingxv1.Host, profile *starlingxv1.HostProfileSpec, host *v1info.HostInfo) (err error) {
	var iface *interfaces.Interface

	if !utils.IsReconcilerEnabled(utils.Interface) {
		return nil
	}

	if profile.Interfaces == nil || len(profile.Interfaces.VF) == 0 {
		return nil
	}

	updated := false

	for _, vfInfo := range profile.Interfaces.VF {
		// For each configured vf interface create or update the related
		// system resource.
		ifuuid, found := host.FindVFInterfaceUUID(vfInfo.Name)
		if !found {
			// Create the interface
			opts := commonInterfaceOptions(vfInfo.CommonInterfaceInfo, profile, host)

			iftype := interfaces.IFTypeVF
			opts.Type = &iftype
			opts.VFDriver = vfInfo.VFDriver
			opts.MaxTxRate = vfInfo.MaxTxRate
			opts.VFCount = &vfInfo.VFCount
			uses := []string{vfInfo.Lower}
			opts.Uses = &uses

			logHost.Info("creating sriov vf interface", "opts", opts)

			iface, err = interfaces.Create(client, opts).Extract()
			if err != nil {
				err = perrors.Wrapf(err, "failed to create sriov vf interface: %s",
					common.FormatStruct(opts))
				return err
			}

			r.NormalEvent(instance, common.ResourceCreated,
				"sriov vf interface %q has been created", vfInfo.Name)

			updated = true
		} else {
			// Update the interface
			iface, found = host.FindInterface(ifuuid)
			if !found {
				msg := fmt.Sprintf("failed to find interface: %s", ifuuid)
				return starlingxv1.NewMissingSystemResource(msg)
			}

			if opts, ok := interfaceUpdateRequired(vfInfo.CommonInterfaceInfo, iface, profile, host); ok {
				logHost.Info("updating sriov vf interface", "uuid", ifuuid, "opts", opts)

				_, err := interfaces.Update(client, ifuuid, opts).Extract()
				if err != nil {
					err = perrors.Wrapf(err, "failed to update interface: %s, %s",
						ifuuid, common.FormatStruct(opts))
					return err
				}

				r.NormalEvent(instance, common.ResourceUpdated,
					"sriovvf  interface %q has been updated", vfInfo.Name)

				updated = true
			}
		}

		networksUpdated, err := r.ReconcileInterfaceNetworks(client, instance, vfInfo.CommonInterfaceInfo, *iface, host)
		if err != nil {
			return err
		}

		dataNetworksUpdated, err := r.ReconcileInterfaceDataNetworks(client, instance, vfInfo.CommonInterfaceInfo, *iface, host)
		if err != nil {
			return err
		}

		ptpInterfaceUpdated, err := r.ReconcilePTPInterface(client, instance, vfInfo.CommonInterfaceInfo, *iface, host)
		if err != nil {
			return err
		}

		updated = updated || networksUpdated || dataNetworksUpdated || ptpInterfaceUpdated
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

		// Delete current defaults so that it will obtain the latest info
		logHost.Info("vf updated. Remove defaults")
		instance.Status.Defaults = nil
	}

	return nil
}

func (r *HostReconciler) ReconcileAddresses(client *gophercloud.ServiceClient, instance *starlingxv1.Host, profile *starlingxv1.HostProfileSpec, host *v1info.HostInfo) error {
	updated := false

	if !utils.IsReconcilerEnabled(utils.Address) {
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
			return starlingxv1.NewMissingSystemResource(msg)
		}

		opts := addresses.AddressOpts{
			Address:       &addrInfo.Address,
			Prefix:        &addrInfo.Prefix,
			InterfaceUUID: &iface.ID,
		}

		logHost.Info("creating address", "opts", opts)

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

func (r *HostReconciler) ReconcileRoutes(client *gophercloud.ServiceClient, instance *starlingxv1.Host, profile *starlingxv1.HostProfileSpec, host *v1info.HostInfo) error {
	updated := false

	if !utils.IsReconcilerEnabled(utils.Route) {
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
			return starlingxv1.NewMissingSystemResource(msg)
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

		logHost.Info("creating route", "opts", opts)

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
func (r *HostReconciler) ReconcileNetworking(client *gophercloud.ServiceClient, instance *starlingxv1.Host, profile *starlingxv1.HostProfileSpec, host *v1info.HostInfo) error {
	var err error

	if !utils.IsReconcilerEnabled(utils.Networking) {
		return nil
	}

	// Remove stale routes or routes on addresses that will be updated.
	err = r.ReconcileStaleRoutes(client, instance, profile, host)
	if err != nil {
		return err
	}

	// Remove stale addresses or addresses on interfaces that will be
	// deleted and re-added.
	err = r.ReconcileStaleAddresses(client, instance, profile, host)
	if err != nil {
		return err
	}

	// Remove stale vlans, bond/vf interfaces that will be deleted
	err = r.ReconcileStaleInterfaces(client, instance, profile, host)
	if err != nil {
		return err
	}

	// Remove stale interface-network associations
	err = r.ReconcileStaleInterfaceNetworks(client, instance, profile, host)
	if err != nil {
		return err
	}

	// Remove stale interface-network associations
	err = r.ReconcileStaleInterfaceDataNetworks(client, instance, profile, host)
	if err != nil {
		return err
	}

	// Remove stale PTP interface associations
	err = r.ReconcileStalePTPInterfaces(client, instance, profile, host)
	if err != nil {
		return err
	}

	// Update SRIOV interfaces
	err = r.ReconcileSRIOVInterfaces(client, instance, profile, host)
	if err != nil {
		return err
	}

	// Update/Add VF interfaces
	err = r.ReconcileVFInterfaces(client, instance, profile, host)
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

	return nil
}
