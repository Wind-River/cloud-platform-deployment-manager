/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019-2023 Wind River Systems, Inc. */

package host

import (
	"context"
	"fmt"
	"net"
	"sort"
	"strings"

	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/hosts"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/interfaces"
	"github.com/imdario/mergo"
	perrors "github.com/pkg/errors"
	starlingxv1 "github.com/wind-river/cloud-platform-deployment-manager/api/v1"
	"github.com/wind-river/cloud-platform-deployment-manager/controllers/common"
	v1info "github.com/wind-river/cloud-platform-deployment-manager/platform"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
)

// MergeProfiles invokes the mergo.Merge API with our desired modifiers.
func MergeProfiles(a, b *starlingxv1.HostProfileSpec) (*starlingxv1.HostProfileSpec, error) {
	t := common.DefaultMergeTransformer
	err := mergo.Merge(a, b, mergo.WithOverride, mergo.WithTransformers(t))
	if err != nil {
		err = perrors.Wrap(err, "mergo.Merge failed to merge profiles")
		return nil, err
	}

	return a, nil
}

// FixProfileAttributes makes some adjustments to profile attributes
func FixProfileAttributes(a, b, c *starlingxv1.HostProfileSpec, hostInfo *v1info.HostInfo) {
	// To compare the BootMAC's we need to lowercase the values
	if a.BootMAC != nil {
		lowerDefaultsBootMAC := strings.ToLower(*a.BootMAC)
		a.BootMAC = &lowerDefaultsBootMAC
	}
	if b.BootMAC != nil {
		lowerProfileBootMAC := strings.ToLower(*b.BootMAC)
		b.BootMAC = &lowerProfileBootMAC
	}

	// To compare the interface Members we need to sort them
	if b.Interfaces.Bond != nil {
		for _, bondInfo := range b.Interfaces.Bond {
			if bondInfo.Members != nil {
				sort.Strings(bondInfo.Members)
			}
		}
	}
	if c.Interfaces.Bond != nil {
		for _, bondInfo := range c.Interfaces.Bond {
			if bondInfo.Members != nil {
				sort.Strings(bondInfo.Members)
			}
		}
	}
	FixProfileDevicePath(a, hostInfo)
}

// FixProfileDevicePath is to fix the device path if it is offered as device node
func FixProfileDevicePath(a *starlingxv1.HostProfileSpec, hostInfo *v1info.HostInfo) {
	if a.RootDevice != nil {
		rootDevice := starlingxv1.FixDevicePath(*a.RootDevice, *hostInfo)
		a.RootDevice = &rootDevice
	}

	if a.BootDevice != nil {
		bootDevice := starlingxv1.FixDevicePath(*a.BootDevice, *hostInfo)
		a.BootDevice = &bootDevice
	}

	if a.Storage.OSDs != nil {
		FixOSDDevicePath(a, hostInfo)
	}

	if a.Storage.VolumeGroups != nil {
		FixVolumeGroupPath(a, hostInfo)
	}
}

// FixOSDDevicePath is to fix device path in the profile's OSD spec
func FixOSDDevicePath(a *starlingxv1.HostProfileSpec, hostInfo *v1info.HostInfo) {

	result := make([]starlingxv1.OSDInfo, 0)
	for _, o := range *a.Storage.OSDs {
		found := false
		o.Path = starlingxv1.FixDevicePath(o.Path, *hostInfo)

		// Check if the element exists with different format(short device node,
		// full deveice node, device path etc.) in the result list, should
		// not add duplication the profile.
		if len(result) > 0 {
			for _, elem := range result {
				found = common.CompareStructs(o, elem)
				if found {
					break
				}
			}
		}

		if !found {
			result = append(result, o)
		}
	}
	list := starlingxv1.OSDList(result)
	a.Storage.OSDs = &list
}

// FixVolumeGroupPath is to fix device path in the profile's VolumeGroup spec
func FixVolumeGroupPath(a *starlingxv1.HostProfileSpec, hostInfo *v1info.HostInfo) {

	result := make([]starlingxv1.VolumeGroupInfo, 0)
	for _, vg := range *a.Storage.VolumeGroups {
		pvs := FixPhysicalVolumesPath(&vg.PhysicalVolumes, hostInfo)
		vgInfo := starlingxv1.VolumeGroupInfo{
			Name:            vg.Name,
			LVMType:         vg.LVMType,
			PhysicalVolumes: pvs,
		}

		result = append(result, vgInfo)
	}
	list := starlingxv1.VolumeGroupList(result)
	a.Storage.VolumeGroups = &list
}

// FixPhysicalVolumesPath is to fix device path in the profile's physical volume spec
func FixPhysicalVolumesPath(a *starlingxv1.PhysicalVolumeList, hostInfo *v1info.HostInfo) starlingxv1.PhysicalVolumeList {

	list := make([]starlingxv1.PhysicalVolumeInfo, 0)
	for _, pv := range *a {
		found := false
		pvPath := starlingxv1.FixDevicePath(pv.Path, *hostInfo)
		pvInfo := starlingxv1.PhysicalVolumeInfo{
			Type: pv.Type,
			Path: pvPath,
			Size: pv.Size,
		}

		// Check if the element exists with different format(short device node,
		// full deveice node, device path etc.) in the result list, should
		// not add duplication the profile.
		if len(list) > 0 {
			for _, elem := range list {
				found = common.CompareStructs(pvInfo, elem)
				if found {
					break
				}
			}
		}

		if !found {
			list = append(list, pvInfo)
		}
	}
	result := starlingxv1.PhysicalVolumeList(list)
	return result
}

// GetHostProfile retrieves a HostProfileSpec from the kubernetes API
func (r *HostReconciler) GetHostProfile(namespace, profile string) (*starlingxv1.HostProfileSpec, error) {
	instance := &starlingxv1.HostProfile{}
	name := types.NamespacedName{Namespace: namespace, Name: profile}

	err := r.Get(context.TODO(), name, instance)
	if err != nil {
		if !errors.IsNotFound(err) {
			err = perrors.Wrapf(err, "failed to get profile: %s", name)
			return nil, err
		} else {
			msg := fmt.Sprintf("host profile %q not present", name)
			return nil, common.NewResourceConfigurationDependency(msg)
		}
	}

	return &instance.Spec, nil
}

// DeleteHostProfile deletes a HostProfile from the kubernetes API
func (r *HostReconciler) DeleteHostProfile(namespace, profile string) error {
	instance := &starlingxv1.HostProfile{}
	name := types.NamespacedName{Namespace: namespace, Name: profile}

	err := r.Get(context.TODO(), name, instance)
	if !errors.IsNotFound(err) {
		err = perrors.Wrapf(err, "failed to get profile: %s", name)
		return err

	} else if err != nil {
		err = r.Delete(context.TODO(), instance)
		if err != nil {
			err = perrors.Wrapf(err, "failed to delete profile: %s", name)
			return err
		}
	}

	return nil
}

// mergeProfileChain merges the profile attributes from each profile in the
// inheritance chain.  This is done recursively and fields set in lower profiles
// take precedence over its parent/base profile attributes.  Arrays are handled
// by looking for equivalent entries in the base profile attribute and replacing
// their values.  Array entries that are not found in the base profile are
// added to the array.
func (r *HostReconciler) mergeProfileChain(namespace string, current *starlingxv1.HostProfileSpec, visited map[string]bool) (*starlingxv1.HostProfileSpec, error) {
	if current.Base != nil {
		if value, ok := visited[*current.Base]; ok && value {
			msg := fmt.Sprintf("profile loop detected at: %s", *current.Base)
			return nil, common.NewValidationError(msg)
		}

		parent, err := r.GetHostProfile(namespace, *current.Base)
		if err != nil {
			return nil, err
		}

		parent, err = r.mergeProfileChain(namespace, parent, visited)
		if err != nil {
			return nil, err
		}

		return MergeProfiles(parent, current)
	}

	defaultCopy := DefaultHostProfile.DeepCopy()
	return MergeProfiles(defaultCopy, current)
}

// BuildCompositeProfile combines the default profile, the profile inheritance
// chain, and host specific overrides to form a final composite profile that
// will be applied to the host at configuration time.
func (r *HostReconciler) BuildCompositeProfile(host *starlingxv1.Host) (*starlingxv1.HostProfileSpec, error) {
	// Start with the explicit profile attached to the host.
	profile, err := r.GetHostProfile(host.Namespace, host.Spec.Profile)
	if err != nil {
		return nil, err
	}

	// Initialize map to track which profiles have already been visited so
	// that we can catch loops.
	visited := make(map[string]bool)

	// Traverse the list of profiles until the root profile is found.
	// Attributes from lower profiles (those closest to the host level) are
	// merged into the higher level profile.
	composite, err := r.mergeProfileChain(host.Namespace, profile, visited)
	if err != nil {
		return composite, err
	}

	// Finally, if the user had provided any per-host overrides then apply
	// over the composite profile.
	if host.Spec.Overrides != nil {
		// Merge the host overrides into the composite profile
		composite, err = MergeProfiles(composite, host.Spec.Overrides)
		if err != nil {
			return composite, err
		}
	}

	if composite.Interfaces != nil && len(composite.Interfaces.Ethernet) == 0 {
		// In some cases it is necessary to set the "ethernet" attribute to
		// an empty array in order to override the list of interfaces from a
		// parent profile, but we never want to override the system defaults
		// which will be applied later so reset this value to nil so that
		// the values from the defaults will be taken.
		composite.Interfaces.Ethernet = nil
	}

	// Remove leading zeros from each IP address in the composite profile
	addressList := make([]starlingxv1.AddressInfo, 0)
	for _, addr := range composite.Addresses {
		address := starlingxv1.AddressInfo{
			Interface: addr.Interface,
			Address:   net.ParseIP(addr.Address).String(),
			Prefix:    addr.Prefix,
		}
		addressList = append(addressList, address)
	}
	if len(addressList) > 0 {
		composite.Addresses = addressList
	}

	routeList := make([]starlingxv1.RouteInfo, 0)
	for _, rt := range composite.Routes {
		route := starlingxv1.RouteInfo{
			Interface: rt.Interface,
			Network:   net.ParseIP(rt.Network).String(),
			Prefix:    rt.Prefix,
			Gateway:   net.ParseIP(rt.Gateway).String(),
			Metric:    rt.Metric,
		}
		routeList = append(routeList, route)
	}
	if len(routeList) > 0 {
		composite.Routes = routeList
	}

	return composite, nil
}

// validateProfileUniqueInterfaces ensures that interface names are unique.  The
// system API will check for this on its own but guaranteeing that the interface
// data is as clean as possible helps simplify some of the coding choice in
// the interface reconciliation code.
func (r *HostReconciler) validateProfileUniqueInterfaces(host *starlingxv1.Host, profile *starlingxv1.HostProfileSpec) error {
	present := make(map[string]bool)

	for _, e := range profile.Interfaces.Ethernet {
		if _, ok := present[e.Name]; ok {
			msg := fmt.Sprintf("interfaces names must be unique; Ethernet %s is a duplicate.", e.Name)
			return common.NewValidationError(msg)
		}
	}

	for _, e := range profile.Interfaces.Bond {
		if _, ok := present[e.Name]; ok {
			msg := fmt.Sprintf("interfaces names must be unique; Bond %s is a duplicate.", e.Name)
			return common.NewValidationError(msg)
		}
	}

	for _, e := range profile.Interfaces.VLAN {
		if _, ok := present[e.Name]; ok {
			msg := fmt.Sprintf("interfaces names must be unique; VLAN %s is a duplicate.", e.Name)
			return common.NewValidationError(msg)
		}
	}

	for _, e := range profile.Interfaces.VF {
		if _, ok := present[e.Name]; ok {
			msg := fmt.Sprintf("interfaces names must be unique; VF %s is a duplicate.", e.Name)
			return common.NewValidationError(msg)
		}
	}

	return nil
}

// validateLoopbackInterface validates that if a loopback interface is specified
// that it references a port with the same name.  This is to ensure that the
// interface reconciliation code can make some assumptions about the naming
// strategy and therefore be simplified.
func (r *HostReconciler) validateLoopbackInterface(host *starlingxv1.Host, profile *starlingxv1.HostProfileSpec) error {
	for _, e := range profile.Interfaces.Ethernet {
		if e.Name == interfaces.LoopbackInterfaceName || e.Port.Name == interfaces.LoopbackInterfaceName {
			if e.Name != e.Port.Name {
				msg := "the virtual loopback interface must reference a port with the same name"
				return common.NewValidationError(msg)
			}
		}
	}
	return nil
}

// validateProfileInterfaces does minimal validation over the list of
// interfaces to be configured.
func (r *HostReconciler) validateProfileInterfaces(host *starlingxv1.Host, profile *starlingxv1.HostProfileSpec) error {
	if profile.Interfaces == nil {
		msg := "'interfaces' profile attribute is required for all hosts"
		return common.NewValidationError(msg)
	}

	err := r.validateProfileUniqueInterfaces(host, profile)
	if err != nil {
		return err
	}
	err = r.validateLoopbackInterface(host, profile)
	if err != nil {
		return err
	}
	return nil
}

// validateBoardManagement performs validation of the Board Management
// host attributes.
func (r *HostReconciler) validateBoardManagement(host *starlingxv1.Host, profile *starlingxv1.HostProfileSpec) error {
	if profile.BoardManagement == nil {
		return nil
	}

	bmInfo := profile.BoardManagement
	if bmInfo.Type == nil {
		msg := "Board Management 'type' is a required attribute"
		return common.NewValidationError(msg)
	}

	if *bmInfo.Type == "none" {
		return nil
	}

	if bmInfo.Credentials == nil {
		msg := "Board Management 'credentials' is a required attribute"
		return common.NewValidationError(msg)
	} else if bmInfo.Credentials.Password == nil {
		msg := "Board Management 'password' is a required attribute"
		return common.NewValidationError(msg)
	}

	if bmInfo.Address == nil {
		msg := "Board Management 'address' is a required attribute"
		return common.NewValidationError(msg)
	}

	return nil
}

// validateProfileAddresses performs validation of IPv4 and IPv6 addresses
// configured for the host
func (r *HostReconciler) validateProfileAddresses(host *starlingxv1.Host, profile *starlingxv1.HostProfileSpec) error {
	for _, addr := range profile.Addresses {
		if net.ParseIP(addr.Address) == nil {
			msg := "'address' profile attributes need to be in a valid IPv4 or IPv6 address format"
			return common.NewValidationError(msg)
		}
	}

	for _, rt := range profile.Routes {
		if net.ParseIP(rt.Network) == nil {
			msg := "'network' profile attributes need to be in a valid IPv4 or IPv6 address format"
			return common.NewValidationError(msg)
		}
		if net.ParseIP(rt.Gateway) == nil {
			msg := "'gateway' profile attributes need to be in a valid IPv4 or IPv6 address format"
			return common.NewValidationError(msg)
		}
	}

	return nil
}

// validateProfileSpec is a private method to validate the contents of a profile
// spec resource.
func (r *HostReconciler) validateProfileSpec(host *starlingxv1.Host, profile *starlingxv1.HostProfileSpec) error {
	if profile.Personality == nil {
		msg := "'personality' is a mandatory profile attribute"
		return common.NewValidationError(msg)
	}

	if !profile.HasWorkerSubFunction() {
		if profile.Processors != nil {
			msg := "'processor' profile attributes are only supported on nodes which include the worker subfunction"
			return common.NewValidationError(msg)
		}
	}

	if *profile.Personality != hosts.PersonalityWorker {
		if profile.Storage != nil && profile.Storage.Monitor != nil {
			msg := "'monitor' profile attributes are only permitted on worker nodes"
			return common.NewValidationError(msg)
		}
	}

	if profile.ProvisioningMode == nil {
		msg := "'provisioningMode' is a mandatory profile attribute"
		return common.NewValidationError(msg)
	}

	if *profile.ProvisioningMode == starlingxv1.ProvioningModeStatic {
		if profile.BootMAC == nil {
			msg := "'bootMAC' profile attribute is required for static provisioning"
			return common.NewValidationError(msg)
		}
	} else {
		if host.Spec.Match == nil {
			msg := "'match' host attribute is required for dynamic provisioning"
			return common.NewValidationError(msg)
		}
	}

	err := r.validateProfileInterfaces(host, profile)
	if err != nil {
		return err
	}

	err = r.validateBoardManagement(host, profile)
	if err != nil {
		return err
	}

	err = r.validateProfileAddresses(host, profile)
	if err != nil {
		return err
	}

	return nil
}

// ValidateProfile examines a composite profile and performs basic validation to
// ensure that all required attributes have been supplied.   This must be done
// at runtime rather than at schema validation time because most fields are
// marked as optional in the schema so that profile inheritance can be used to
// specify only subsets of attributes at each profile level (e.g., an interface
// profile does not need to set personality or administrative state, but some
// profile in the inheritance chain must).  Therefore each individual profile
// itself may not be valid but when attached to a host the full chain of
// profiles must produce a valid set of attributes.
func (r *HostReconciler) ValidateProfile(host *starlingxv1.Host, profile *starlingxv1.HostProfileSpec) error {
	err := r.validateProfileSpec(host, profile)
	if err != nil {
		return err
	}

	return nil
}
