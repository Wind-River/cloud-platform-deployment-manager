/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019-2023 Wind River Systems, Inc. */

package build

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/cpus"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/hosts"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/interfaces"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/memory"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/volumegroups"
	v1 "github.com/wind-river/cloud-platform-deployment-manager/api/v1"
	utils "github.com/wind-river/cloud-platform-deployment-manager/common"
)

// HostFilter defines an interface from which concrete host filters can
// be defined.  The purpose of a host filter is to look at a given profile
// and remove any fields that are relevant to a single host versus being
// relevant to multiple hosts.  Those fields should be moved to the host
// overrides attributes.
type HostFilter interface {
	Filter(profile *v1.HostProfile, host *v1.Host, deployment *Deployment) error
}

// Controller0Filter defines a host filter which is responsible for changing
// the provisioning mode of the controller-0 nodes from static to dynamic since
// we never statically provisioning controller-0 as it is always pre-populated.
type Controller0Filter struct {
}

func NewController0Filter() *Controller0Filter {
	return &Controller0Filter{}
}

func (in *Controller0Filter) Filter(profile *v1.HostProfile, host *v1.Host, deployment *Deployment) error {
	if host.Name == hosts.Controller0 {
		// Controller0 must always be dynamic since it is expected to
		// already be present.  Set this in the overrides rather than in the
		// profile to minimize the number of profiles required.
		dynamic := v1.ProvioningModeDynamic
		host.Spec.Overrides.ProvisioningMode = &dynamic
		if host.Spec.Overrides.BootMAC != nil {
			host.Spec.Match = &v1.MatchInfo{
				BootMAC: host.Spec.Overrides.BootMAC,
			}
			host.Spec.Overrides.BootMAC = nil
		}
	}

	return nil
}

// LocationFilter defines a host filter which is responsible for moving
// the location attribute (if set) from a host profile to the host overrides
// attributes.  Since locations are usually specified to identify specific
// hosts rather than the larger data center then it usually belongs at the
// host level.
type LocationFilter struct {
}

func NewLocationFilter() *LocationFilter {
	return &LocationFilter{}
}

func (in *LocationFilter) Filter(profile *v1.HostProfile, host *v1.Host, deployment *Deployment) error {
	if profile.Spec.Location != nil {
		host.Spec.Overrides.Location = profile.Spec.Location
		profile.Spec.Location = nil
	}

	return nil
}

// AddressFilter defines a host filter which is responsible for moving
// IP addresses from a host profile to the host overrides attributes.  Since IP
// addresses are always host specific there is no need to create a unique
// profile just for this one attribute.
type AddressFilter struct {
}

func NewAddressFilter() *AddressFilter {
	return &AddressFilter{}
}

func (in *AddressFilter) Filter(profile *v1.HostProfile, host *v1.Host, deployment *Deployment) error {
	// There are certain profile attributes that are most certainly host
	// specific so move them from the profile to the host overrides.
	if profile.Spec.Addresses != nil {
		host.Spec.Overrides.Addresses = profile.Spec.Addresses
		profile.Spec.Addresses = nil
	}

	return nil
}

// BMAddressFilter defines a host filter which is responsible for moving
// a BM address from a host profile to the host overrides attributes.  Since IP
// addresses are always host specific there is no need to create a unique
// profile just for this one attribute.
type BMAddressFilter struct {
}

func NewBMAddressFilter() *BMAddressFilter {
	return &BMAddressFilter{}
}

func (in *BMAddressFilter) Filter(profile *v1.HostProfile, host *v1.Host, deployment *Deployment) error {
	if profile.Spec.BoardManagement != nil && profile.Spec.BoardManagement.Address != nil {
		// If there is a BM address then it is going to be host specific
		// so move the attribute, but leave the credentials in the profile since
		// we assume that all hosts share the same credentials
		host.Spec.Overrides.BoardManagement = &v1.BMInfo{
			Address: profile.Spec.BoardManagement.Address,
		}
		profile.Spec.BoardManagement.Address = nil
	}

	return nil
}

// StorageMonitorFilter defines a host filter which is responsible for moving
// storage monitors from a host profile to the host overrides attribute. since
// it is only expected to be present on a single node and we do not want to
// prevent sharing the same profile across multiple nodes.
type StorageMonitorFilter struct {
}

func NewStorageMonitorFilter() *StorageMonitorFilter {
	return &StorageMonitorFilter{}
}

func (in *StorageMonitorFilter) Filter(profile *v1.HostProfile, host *v1.Host, deployment *Deployment) error {
	storage := profile.Spec.Storage
	if storage != nil && storage.Monitor != nil {
		if host.Spec.Overrides.Storage == nil {
			host.Spec.Overrides.Storage = &v1.ProfileStorageInfo{}
		}
		host.Spec.Overrides.Storage.Monitor = storage.Monitor

		profile.Spec.Storage.Monitor = nil
		if storage.VolumeGroups == nil && storage.OSDs == nil && storage.FileSystems == nil {
			profile.Spec.Storage = nil
		}
	}

	return nil
}

// LoopbackInterfaceFilter defines a host filter which is responsible for moving
// any loopback interface specification from the profile to the host overrides.
// The loopback interface is only expected on controller-0 nodes therefore
// rather than create a unique profile because of this one interface we prefer
// to move it to the host specific overrides and leave the profile as generic.
type LoopbackInterfaceFilter struct {
}

func NewLoopbackInterfaceFilter() *LoopbackInterfaceFilter {
	return &LoopbackInterfaceFilter{}
}

func (in *LoopbackInterfaceFilter) Filter(profile *v1.HostProfile, host *v1.Host, deployment *Deployment) error {
	var loopbackInfo v1.EthernetInfo

	if profile.Spec.Interfaces == nil {
		return nil
	}
	profileInterfaces := profile.Spec.Interfaces

	result := make([]v1.EthernetInfo, 0)
	for _, ethInfo := range profileInterfaces.Ethernet {
		if ethInfo.Name != interfaces.LoopbackInterfaceName {
			result = append(result, ethInfo)
		} else {
			loopbackInfo = ethInfo
		}
	}

	if len(profileInterfaces.Ethernet) != len(result) {
		profileInterfaces.Ethernet = result
		hostInterfaces := host.Spec.Overrides.Interfaces
		if hostInterfaces == nil {
			hostInterfaces = &v1.InterfaceInfo{}
		}
		hostInterfaces.Ethernet = append(hostInterfaces.Ethernet, loopbackInfo)
		host.Spec.Overrides.Interfaces = hostInterfaces
	}

	return nil
}

// ProfileFilter defines an interface from which concrete profile filters can
// be defined.  The purpose of a profile filter is to look at a given profile
// and remove any fields that are not necessary or relevant to the deployment
// being generated.
type ProfileFilter interface {
	Filter(profile *v1.HostProfile, deployment *Deployment) error
	Reset()
}

// InterfaceUnusedFilter defines a profile filter which looks at the list of
// interfaces present on a profile and removes any unused ethernet interfaces.
// For instance, an ethernet interface is considered unused if it has no
// 'class' set and it not used as a dependency to some other interface.
type InterfaceUnusedFilter struct {
}

func NewInterfaceUnusedFilter() *InterfaceUnusedFilter {
	return &InterfaceUnusedFilter{}
}

func (in *InterfaceUnusedFilter) Reset() {
	// Nothing to do
}

func (in *InterfaceUnusedFilter) Filter(profile *v1.HostProfile, deployment *Deployment) error {
	if profile.Spec.Interfaces == nil {
		return nil
	}

	info := profile.Spec.Interfaces

	result := v1.EthernetList{}
	for _, e := range info.Ethernet {
		if e.Class != interfaces.IFClassNone || isInterfaceInUse(e.Name, info) {
			result = append(result, e)
		}
	}

	if len(result) != 0 {
		info.Ethernet = result
	} else {
		info.Ethernet = nil
	}

	return nil
}

// MemoryClearAllFilter defines a special memory filter that removes all memory
// configurations.  This is useful to avoid slight memory discrepancies between
// when we know that the system defaults have never been updated.
type MemoryClearAllFilter struct {
}

func NewMemoryClearAllFilter() *MemoryClearAllFilter {
	return &MemoryClearAllFilter{}
}

func (in *MemoryClearAllFilter) Reset() {
	// Nothing to do
}

func (in *MemoryClearAllFilter) Filter(profile *v1.HostProfile, deployment *Deployment) error {
	profile.Spec.Memory = nil
	return nil
}

// MemoryDefaultsFilter defines a profile filter which looks at the list of
// memory functions present and removes any that appear to be system defaults.
type MemoryDefaultsFilter struct {
}

func NewMemoryDefaultsFilter() *MemoryDefaultsFilter {
	return &MemoryDefaultsFilter{}
}

func (in *MemoryDefaultsFilter) Reset() {
	// Nothing to do
}

func (in *MemoryDefaultsFilter) Filter(profile *v1.HostProfile, deployment *Deployment) error {
	nodes := v1.MemoryNodeList{}
	for _, node := range profile.Spec.Memory {
		functions := v1.MemoryFunctionList{}
		for _, function := range node.Functions {
			if function.PageCount == 0 {
				// Unallocated functions do not need to be captured.
				continue
			} else if strings.EqualFold(function.Function, memory.MemoryFunctionVSwitch) && function.PageCount == 1 {
				// If there is only a single VSwitch page then that is likely
				// the system default
				continue
			}

			// TODO(alegacy): are there any other system defaults that we could
			//  identify and remove?  Platform memory is likely unchanged but
			//  there are no guarantees.

			functions = append(functions, function)
		}

		if len(functions) != 0 {
			node.Functions = functions
			nodes = append(nodes, node)
		} else {
			node.Functions = nil
		}
	}

	if len(nodes) != 0 {
		profile.Spec.Memory = nodes
	} else {
		profile.Spec.Memory = nil
	}

	return nil
}

// ProcessorDefaultsFilter defines a profile filter which look at the list of processor
// functions present and removes any that appear to be system defaults.
type ProcessorDefaultsFilter struct {
}

func NewProcessorDefaultsFilter() *ProcessorDefaultsFilter {
	return &ProcessorDefaultsFilter{}
}

func (in *ProcessorDefaultsFilter) Reset() {
	// Nothing to do
}

func (in *ProcessorDefaultsFilter) Filter(profile *v1.HostProfile, deployment *Deployment) error {
	nodes := v1.ProcessorNodeList{}
	for _, node := range profile.Spec.Processors {
		functions := v1.ProcessorFunctionList{}
		for _, function := range node.Functions {
			if strings.EqualFold(function.Function, cpus.CPUFunctionPlatform) {
				switch *profile.Spec.Personality {
				case hosts.PersonalityController:
					if function.Count == 2 {
						// Controller nodes default to 2 platform cores.
						continue
					}
				case hosts.PersonalityWorker:
					if function.Count == 1 {
						// Worker nodes default to 1 platform core.
						continue
					}
				}

			}

			// For all other CPU functions we allow any value because it is
			// a bit tricky to guess at the defaults since the system behaves
			// differently based on the vswitch type, the system mode, and the
			// system type.

			functions = append(functions, function)
		}

		if len(functions) != 0 {
			node.Functions = functions
			nodes = append(nodes, node)
		} else {
			node.Functions = nil
		}
	}

	if len(nodes) != 0 {
		profile.Spec.Processors = nodes
	} else {
		profile.Spec.Processors = nil
	}

	return nil
}

// ProcessorClearAllFilter defines a special Processor filter that removes all
// Processor configurations.  This is useful to avoid cases where it is
// difficult to determine if the current processor configuration is a system
// default or not.
type ProcessorClearAllFilter struct {
}

func NewProcessorClearAllFilter() *ProcessorClearAllFilter {
	return &ProcessorClearAllFilter{}
}

func (in *ProcessorClearAllFilter) Reset() {
	// Nothing to do
}

func (in *ProcessorClearAllFilter) Filter(profile *v1.HostProfile, deployment *Deployment) error {
	profile.Spec.Processors = nil
	return nil
}

// VolumeGroupFilter defines a profile filter which looks at the list of
// volume groups present and removes any that are included in the specified
// blacklist.
type VolumeGroupFilter struct {
	Blacklist []string
}

func NewVolumeGroupFilter(blacklist []string) *VolumeGroupFilter {
	return &VolumeGroupFilter{Blacklist: blacklist}
}

func NewVolumeGroupSystemFilter() *VolumeGroupFilter {
	return NewVolumeGroupFilter(volumegroups.SystemDefinedVolumeGroups)
}

func (in *VolumeGroupFilter) Reset() {
	// Nothing to do
}

func (in *VolumeGroupFilter) Filter(profile *v1.HostProfile, deployment *Deployment) error {
	if profile.Spec.Storage == nil || profile.Spec.Storage.VolumeGroups == nil {
		return nil
	}

	storage := profile.Spec.Storage
	groups := v1.VolumeGroupList{}
	for _, vg := range *storage.VolumeGroups {
		if utils.ContainsString(in.Blacklist, vg.Name) {
			// This group is blacklisted so skip it.
			continue
		}

		groups = append(groups, vg)
	}

	if len(groups) != 0 {
		list := v1.VolumeGroupList(groups)
		storage.VolumeGroups = &list
	} else {
		storage.VolumeGroups = nil
	}

	return nil
}

// InterfaceNamingFilter defines a profile filter that normalizes interface
// names across host profiles so that there is a better chance of reducing
// the number of profiles that exist in the system definition.
type InterfaceNamingFilter struct {
	updates map[string]string
}

func NewInterfaceNamingFilter() *InterfaceNamingFilter {
	return &InterfaceNamingFilter{}
}

const (
	pxebootNetwork = "pxeboot"
	pxebootIface   = "pxeboot0"
	mgmtNetwork    = "mgmt"
	mgmtIface      = "mgmt0"
	clusterNetwork = "cluster-host"
	clusterIface   = "cluster0"
	oamNetwork     = "oam"
	oamIface       = "oam0"
)

func (in *InterfaceNamingFilter) CheckInterface(info *v1.CommonInterfaceInfo) {
	if info.Name == interfaces.LoopbackInterfaceName {
		// Never rename the Loopback interface
		return
	} else if info.PlatformNetworks == nil {
		return
	}

	networks := v1.PlatformNetworkItemListToStrings(*info.PlatformNetworks)

	if utils.ContainsString(networks, pxebootNetwork) {
		if info.Name != pxebootIface {
			in.updates[info.Name] = pxebootIface
			info.Name = pxebootIface
		}
	} else if utils.ContainsString(networks, mgmtNetwork) {
		if info.Name != mgmtIface {
			in.updates[info.Name] = mgmtIface
			info.Name = mgmtIface
		}
	} else if utils.ContainsString(networks, clusterNetwork) {
		if info.Name != clusterNetwork {
			in.updates[info.Name] = clusterIface
			info.Name = clusterIface
		}
	} else if utils.ContainsString(networks, oamNetwork) {
		if info.Name != oamNetwork {
			in.updates[info.Name] = oamIface
			info.Name = oamIface
		}
	}
}

func (in *InterfaceNamingFilter) Reset() {
	in.updates = make(map[string]string)
}

func (in *InterfaceNamingFilter) Filter(profile *v1.HostProfile, deployment *Deployment) error {
	if profile.Spec.Interfaces == nil {
		return nil
	}

	ethernet := profile.Spec.Interfaces.Ethernet
	for idx := range ethernet {
		in.CheckInterface(&ethernet[idx].CommonInterfaceInfo)
	}

	bonds := profile.Spec.Interfaces.Bond
	for idx := range bonds {
		in.CheckInterface(&bonds[idx].CommonInterfaceInfo)
	}

	vlans := profile.Spec.Interfaces.VLAN
	for idx := range vlans {
		in.CheckInterface(&vlans[idx].CommonInterfaceInfo)

		// Update any references to lower interface names that may have changed.
		if newName, ok := in.updates[vlans[idx].Lower]; ok {
			vlans[idx].Lower = newName
		}
	}

	vfs := profile.Spec.Interfaces.VF
	for idx := range vfs {
		in.CheckInterface(&vfs[idx].CommonInterfaceInfo)

		// Update any references to lower interface names that may have changed.
		if newName, ok := in.updates[vfs[idx].Lower]; ok {
			vfs[idx].Lower = newName
		}
	}

	// Update any address references to interface names that may have changed.
	for _, a := range profile.Spec.Addresses {
		if newName, ok := in.updates[a.Interface]; ok {
			a.Interface = newName
		}
	}

	// Update any route references to interface names that may have changed.
	for _, r := range profile.Spec.Routes {
		if newName, ok := in.updates[r.Interface]; ok {
			r.Interface = newName
		}
	}

	return nil
}

// InterfaceMTUFilter defines a profile filter which attempts to find
// discrepancies with MTU values across different nodes.  This is a two-pass
// filter meaning it must be run on all profiles twice so that the highwater
// mark can be applied to all profiles.
type InterfaceMTUFilter struct {
	highwatermarks map[string]int
}

func NewInterfaceMTUFilter() *InterfaceMTUFilter {
	return &InterfaceMTUFilter{highwatermarks: make(map[string]int)}
}

func (in *InterfaceMTUFilter) Reset() {
	// Nothing to do
}

func (in *InterfaceMTUFilter) CheckMTU(info *v1.CommonInterfaceInfo) {
	value := interfaces.DefaultMTU
	if info.MTU != nil {
		value = *info.MTU
	}

	if info.PlatformNetworks == nil {
		return
	}

	networks := v1.PlatformNetworkItemListToStrings(*info.PlatformNetworks)
	for _, network := range networks {
		if max, ok := in.highwatermarks[network]; ok {
			if value > max {
				in.highwatermarks[network] = value
			} else if max != interfaces.DefaultMTU {
				info.MTU = &max
			}
		} else {
			in.highwatermarks[network] = value
		}
	}
}

func (in *InterfaceMTUFilter) CheckMemberMTU(info *v1.BondInfo, ethernet v1.EthernetList) {
	for idx := range ethernet {
		if utils.ContainsString(info.Members, ethernet[idx].Name) {
			ethernet[idx].MTU = info.MTU
		}
	}
}

func (in *InterfaceMTUFilter) Filter(profile *v1.HostProfile, deployment *Deployment) error {
	ethernet := profile.Spec.Interfaces.Ethernet
	for idx := range ethernet {
		in.CheckMTU(&ethernet[idx].CommonInterfaceInfo)
	}

	bonds := profile.Spec.Interfaces.Bond
	for idx := range bonds {
		in.CheckMTU(&bonds[idx].CommonInterfaceInfo)
		in.CheckMemberMTU(&bonds[idx], ethernet)
	}

	vlans := profile.Spec.Interfaces.VLAN
	for idx := range vlans {
		in.CheckMTU(&vlans[idx].CommonInterfaceInfo)
	}

	vfs := profile.Spec.Interfaces.VF
	for idx := range vfs {
		in.CheckMTU(&vfs[idx].CommonInterfaceInfo)
	}

	return nil
}

// ConsoleNameFilter defines a profile filter a that attempts to normalize the
// console attributes on hosts.  Console specifications seem to be consistently
// setting the parity and stop bits so if they are missing we attempt to apply
// the Linux default values.  The flow control value is left off since it does
// not appear to be used much.  Empty values are omitted since the API does
// not accept them.
type ConsoleNameFilter struct {
	regex *regexp.Regexp
}

func NewConsoleNameFilter() *ConsoleNameFilter {
	return &ConsoleNameFilter{regex: regexp.MustCompile(`ttyS[0-9]+,\d+$`)}
}

func (in *ConsoleNameFilter) Reset() {
	// Nothing to do
}

func (in *ConsoleNameFilter) Filter(profile *v1.HostProfile, deployment *Deployment) error {
	if profile.Spec.Console != nil {
		if in.regex.MatchString(*profile.Spec.Console) {
			console := fmt.Sprintf("%sn8", *profile.Spec.Console)
			profile.Spec.Console = &console
		} else if *profile.Spec.Console == "" {
			profile.Spec.Console = nil
		}
	}

	return nil
}

// InterfaceDefaultsFilter defines a profile filter that removes default values
// from interfaces in an effort to minimize the number of fields defined on
// interfaces.
type InterfaceDefaultsFilter struct {
	updates map[string]string
}

func NewInterfaceDefaultsFilter() *InterfaceDefaultsFilter {
	return &InterfaceDefaultsFilter{}
}

func (in *InterfaceDefaultsFilter) Reset() {
	in.updates = make(map[string]string)
}

func (in *InterfaceDefaultsFilter) CheckInterface(info *v1.CommonInterfaceInfo) {
	if info.MTU != nil && *info.MTU == interfaces.DefaultMTU {
		info.MTU = nil
	}

	// NOTE(alegacy): for now we allow all other values because excluding either
	// the platform or data networks causes issues for AE data members since
	// the collected defaults may have a non-empty list and if we apply a
	// configuration over top with a nil network list then the original default
	// values will be used which will cause issues at configuration time.
}

func (in *InterfaceDefaultsFilter) Filter(profile *v1.HostProfile, deployment *Deployment) error {
	if profile.Spec.Interfaces == nil {
		return nil
	}

	ethernet := profile.Spec.Interfaces.Ethernet
	for idx := range ethernet {
		in.CheckInterface(&ethernet[idx].CommonInterfaceInfo)
	}

	bonds := profile.Spec.Interfaces.Bond
	for idx := range bonds {
		in.CheckInterface(&bonds[idx].CommonInterfaceInfo)
	}

	vlans := profile.Spec.Interfaces.VLAN
	for idx := range vlans {
		in.CheckInterface(&vlans[idx].CommonInterfaceInfo)
	}

	vfs := profile.Spec.Interfaces.VF
	for idx := range vfs {
		in.CheckInterface(&vfs[idx].CommonInterfaceInfo)
	}

	return nil
}

// SystemFilter defines an interface from which concrete system filters can
// be defined.  The purpose of a system filter is to remove attributes that
// may not be needed for a runtime configuration or to align values to
// end user requirements.
type SystemFilter interface {
	Filter(system *v1.System, deployment *Deployment) error
}

// CACertificateFilter defines a system filter that removes trusted CA
// certificates from the configuration under the assumption that they were added
// at bootstrap time rather than as a post install step.  This is being done
// to remove the trusted ssl_ca certificates that get installed at boostrap
// time to allow images from custom docker registries to be loaded.  Since
// those certificates are added at bootstrap time there is no need to also
// add them to the deployment config since it only adds an extra step for
// end users.  Since we do not delete certificates at reconcile time (for now)
// there is no need to have these included.
type CACertificateFilter struct {
}

func NewCACertificateFilter() *CACertificateFilter {
	return &CACertificateFilter{}
}

func (in *CACertificateFilter) Filter(system *v1.System, deployment *Deployment) error {
	if system.Spec.Certificates == nil {
		return nil
	}

	result := make([]v1.CertificateInfo, 0)
	for _, c := range *system.Spec.Certificates {
		if c.Type == v1.PlatformCACertificate || c.Type == v1.OpenstackCACertificate {
			continue
		}

		result = append(result, c)
	}

	if len(result) > 0 {
		list := v1.CertificateList(result)
		system.Spec.Certificates = &list
	} else {
		system.Spec.Certificates = nil
	}

	return nil
}

type ServiceParameterFilter struct {
}

func NewServiceParametersSystemFilter() *ServiceParameterFilter {
	return &ServiceParameterFilter{}
}

func IsDefaultServiceParameter(sp *v1.ServiceParameterInfo) bool {
	return true
}

func (in *ServiceParameterFilter) Filter(system *v1.System, deployment *Deployment) error {
	if system.Spec.ServiceParameters == nil {
		return nil
	}

	result := make([]v1.ServiceParameterInfo, 0)
	for _, sp := range *system.Spec.ServiceParameters {
		// currently this skips everything
		if IsDefaultServiceParameter(&sp) {
			continue
		}
		result = append(result, sp)
	}

	if len(result) > 0 {
		list := v1.ServiceParameterList(result)
		system.Spec.ServiceParameters = &list
	} else {
		system.Spec.ServiceParameters = nil
	}

	return nil
}

// InterfaceRemoveUuidFilter defines a profile and host filter that removes
// uuid values from interfaces.
type InterfaceRemoveUuidFilter struct {
	updates map[string]string
}

func NewInterfaceRemoveUuidFilter() *InterfaceRemoveUuidFilter {
	return &InterfaceRemoveUuidFilter{}
}

func (in *InterfaceRemoveUuidFilter) Reset() {
	in.updates = make(map[string]string)
}

func (in *InterfaceRemoveUuidFilter) CheckInterface(info *v1.CommonInterfaceInfo) {
	info.UUID = ""
}

func (in *InterfaceRemoveUuidFilter) Filter(profile *v1.HostProfile, host *v1.Host, deployment *Deployment) error {
	// Check in profile
	if profile.Spec.Interfaces == nil {
		return nil
	}

	ethernet := profile.Spec.Interfaces.Ethernet
	for idx := range ethernet {
		in.CheckInterface(&ethernet[idx].CommonInterfaceInfo)
	}

	bonds := profile.Spec.Interfaces.Bond
	for idx := range bonds {
		in.CheckInterface(&bonds[idx].CommonInterfaceInfo)
	}

	vlans := profile.Spec.Interfaces.VLAN
	for idx := range vlans {
		in.CheckInterface(&vlans[idx].CommonInterfaceInfo)
	}

	vfs := profile.Spec.Interfaces.VF
	for idx := range vfs {
		in.CheckInterface(&vfs[idx].CommonInterfaceInfo)
	}

	// Check in host override
	if host.Spec.Overrides.Interfaces == nil {
		return nil
	}

	ethernet = host.Spec.Overrides.Interfaces.Ethernet
	for idx := range ethernet {
		in.CheckInterface(&ethernet[idx].CommonInterfaceInfo)
	}

	bonds = host.Spec.Overrides.Interfaces.Bond
	for idx := range bonds {
		in.CheckInterface(&bonds[idx].CommonInterfaceInfo)
	}

	vlans = host.Spec.Overrides.Interfaces.VLAN
	for idx := range vlans {
		in.CheckInterface(&vlans[idx].CommonInterfaceInfo)
	}

	vfs = host.Spec.Overrides.Interfaces.VF
	for idx := range vfs {
		in.CheckInterface(&vfs[idx].CommonInterfaceInfo)
	}
	return nil
}
