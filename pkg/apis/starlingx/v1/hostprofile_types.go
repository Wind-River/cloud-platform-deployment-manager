/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019 Wind River Systems, Inc. */

package v1

import (
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/clusters"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/hosts"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +deepequal-gen:unordered-array=true
type StringList []string

// ToStringList converts from this type alias to an actual array of strings
func (in StringList) ToStringList() []string {
	result := make([]string, 0)
	for _, n := range in {
		result = append(result, n)
	}
	return result
}

// BMPasswordInfo defines attributes specific to password based
// authentication.
type BMPasswordInfo struct {
	// Secret defines the name of the secret which contains the username and
	// password for the board management
	// controller.
	Secret string `json:"secret"`
}

// DeepEqual overrides the code generated DeepEqual method because the
// credential information built from the running configuration never includes
// the password since the system API does not provide it.  Therefore when
// the BMPasswordInfo is setup dynamically we put a dummy value in the Secret
// name which will likely never match what is in the desired configuration so
// there is no point in comparing it.
func (in *BMPasswordInfo) DeepEqual(other *BMPasswordInfo) bool {
	// TODO(alegacy): A side effect of not being able to compare the password
	//  based credential information is that we also do not compare the username
	//  so we will never reconcile it unless some other attribute also changed.
	//  That is fine for now since we are only supporting the initial
	//  configuration and not subsequent modifications.
	return true
}

// BMCredentials defines attributes specific to each authentication
// type.
// +deepequal-gen:ignore-nil-fields=true
type BMCredentials struct {
	// Password defines the attributes specific to password based
	// authentication.
	// +optional
	Password *BMPasswordInfo `json:"password,omitempty"`
}

// +deepequal-gen:ignore-nil-fields=true
type BMInfo struct {
	// Type defines the board management controller type.  This is left as
	// optional so that the address can be overridden on a per-host basis
	// without worrying about overwriting the type or credentials.
	// +kubebuilder:validation:Enum=bmc
	// +optional
	Type *string `json:"type,omitempty"`

	// Address defines the IP address or hostname of the board management
	// interface.  An address is specific to a host therefore this should only
	// be set if the profile is only going to be used to configure a single
	// host; otherwise it should be set as a per-host override.
	// +optional
	Address *string `json:"address,omitempty"`

	// Credentials defines the authentication credentials for the board
	// management interface.  This is left as optional so that the address can
	// be overridden on a per-host basis without worrying about overwriting the
	// type or credentials.
	// +optional
	Credentials *BMCredentials `json:"credentials,omitempty"`
}

// ProcessorFunctionInfo defines the number of cores to assign to a
// specific function.
type ProcessorFunctionInfo struct {
	// Function defines the function for which to allocate a number of cores.
	// +kubebuilder:validation:Enum=platform,shared,vswitch,isolated,applications
	Function string `json:"function"`

	// Count defines the number of cores to allocate to a specific function.
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=64
	Count int `json:"count"`
}

// ProcessorFunctionList defines a type to represent a slice of processor
// function objects.
// +deepequal-gen:unordered-array=true
type ProcessorFunctionList []ProcessorFunctionInfo

// ProcessorInfo defines the processor core allocations for a
// specific NUMA socket/node.
type ProcessorInfo struct {
	// Node defines the NUMA node number for which to allocate a number of
	// functions.
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=7
	Node int `json:"node"`

	// Functions defines a list of function specific allocations for the given
	// NUMA socket/node.
	Functions ProcessorFunctionList `json:"functions"`
}

// ProcessorNodeList defines a type to represent a slice of processor infos
// +deepequal-gen:unordered-array=true
type ProcessorNodeList []ProcessorInfo

// MemoryFunctionInfo defines the amount of memory to assign to a
// specific function.
type MemoryFunctionInfo struct {
	// Function defines the function for which to allocate a number of cores.
	// +kubebuilder:validation:Enum=platform,vm,vswitch
	Function string `json:"function"`

	// PageSize defines the size of individual memory pages to be allocated to
	// a specific function.  For platform
	// allocations the 4KB page size is the only valid choice.
	// +kubebuilder:validation:Enum=4KB,2MB,1GB
	PageSize string `json:"pageSize"`

	// PageCount defines the number of pages to allocate to a specific function.
	PageCount int `json:"pageCount"`
}

// MemoryFunctionList defines a type to represent a slice of memory function
// objects.
// +deepequal-gen:unordered-array=true
type MemoryFunctionList []MemoryFunctionInfo

// MemoryNodeInfo defines the memory allocations for a specific NUMA
// node/socket.
type MemoryNodeInfo struct {
	// Node defines the NUMA node number for which to allocate a number of
	// functions.
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=7
	Node int `json:"node"`

	// Functions defines a list of function specific allocations for the given
	// NUMA socket/node.
	Functions MemoryFunctionList `json:"functions"`
}

// MemoryNodeList defines a type to represent a slice of memory node objects.
// +deepequal-gen:unordered-array=true
type MemoryNodeList []MemoryNodeInfo

// JournalInfo defines attributes of an OSD journal device.
type JournalInfo struct {
	// Location defines	the OSD device path to be used as the Journal OSD for
	// this logical device.
	// +kubebuilder:validation:MaxLength=255
	Location string `json:"location"`

	// Size defines the size of the OSD journal in gibibytes.
	// +kubebuilder:validation:Minimum=1
	Size int `json:"size"`
}

// OSDInfo defines attributes specific to a single OSD device.
// +deepequal-gen:ignore-nil-fields=true
type OSDInfo struct {
	// Function defines the function to be assigned to the OSD device.
	// +kubebuilder:validation:Enum=osd,journal
	Function string `json:"function"`

	// Path defines the disk device path to use as backing for the OSD device.
	// +kubebuilder:validation:MaxLength=4095
	// +kubebuilder:validation:Pattern=^/dev/.+$
	Path string `json:"path"`

	// ClusterName defines the storage cluster to which the OSD device should
	// be assigned.  By default this is the "ceph_cluster".
	// +kubebuilder:validation:MaxLength=255
	// +optional
	ClusterName *string `json:"cluster,omitempty"`

	// Journal defines another OSD device to be used as the journal for this
	// OSD device.
	// +optional
	Journal *JournalInfo `json:"journal,omitempty"`
}

// OSDList defines a type to represent a slice of OSD objects.
// +deepequal-gen:unordered-array=true
type OSDList []OSDInfo

// GetClusterName returns the configured cluster name or the default if it
// wasn't specified.
// TODO(alegacy): this could be done with a defaulting webhook but it seems like
//  overkill for so few cases where a default is necessary.
func (in *OSDInfo) GetClusterName() string {
	if in.ClusterName == nil {
		return clusters.CephClusterName
	}
	return *in.ClusterName
}

// PhysicalVolumeInfo defines attributes of a physical volume.
// +deepequal-gen:ignore-nil-fields=true
type PhysicalVolumeInfo struct {
	// Type defines the type of physical volume.
	// +kubebuilder:validation:Enum=disk,partition
	Type string `json:"type"`

	// Path defines the device path backing the physical volume.  If 'Type' is
	// set as disk then this attribute refers to the absolute path of a disk
	// device.  If 'Type' is set as partition then it refers to the device path
	// of the disk onto which this partition will be created.
	// +kubebuilder:validation:MaxLength=255
	Path string `json:"path"`

	// Size defines the size of the disk partition in gibibytes.  This should be
	// omitted if the path refers to a disk.
	// +kubebuilder:validation:Minimum=1
	// +optional
	Size *int `json:"size,omitempty"`
}

// PhysicalVolumeList defines a type to represent a slice of physical volumes
// +deepequal-gen:unordered-array=true
type PhysicalVolumeList []PhysicalVolumeInfo

// VolumeGroupInfo defines the attributes specific to a single
// volume group.
// +deepequal-gen:ignore-nil-fields=true
type VolumeGroupInfo struct {
	// SystemName defines the name of the logical volume group
	// +kubebuilder:validation:MaxLength=255
	// +kubebuilder:validation:Pattern=^[a-zA-Z0-9\-_]+$
	Name string `json:"name"`

	// ConcurrentDiskOperations defines the number of concurrent disk operations
	// permitted.
	// +kubebuilder:validation:Minimum=1
	// +optional
	ConcurrentDiskOperations *int `json:"concurrentDiskOperations,omitempty"`

	// LVMType defines the provisioning type for volumes defines with 'Type'
	// set to 'lvm'.
	// +kubebuilder:validation:Enum=thin,thick
	// +optional
	LVMType *string `json:"lvmType,omitempty"`

	// PhysicalVolumes defines the list of volumes to be created on the host.
	PhysicalVolumes PhysicalVolumeList `json:"physicalVolumes"`
}

// VolumeGroupList defines a type to represent a slice of volume groups
// +deepequal-gen:unordered-array=true
type VolumeGroupList []VolumeGroupInfo

// MonitorInfo defines the monitor attributes used to
// configure a Ceph storage monitor on a node.
// +deepequal-gen:ignore-nil-fields=true
type MonitorInfo struct {
	// Size represents the storage allocated to the monitor in gibibytes
	// +kubebuilder:validation:Minimum=20
	// +kubebuilder:validation:Maximum=40
	// +optional
	Size *int `json:"size,omitempty"`
}

// FileSystemInfo defines the attributes of a single host filesystem resource.
type FileSystemInfo struct {
	// Name defines the system defined name of the filesystem resource.  Each
	// filesystem name may only be applicable to a subset of host personalities.
	// Refer to StarlingX documentation for more information.
	// +kubebuilder:validation:Enum=backup,docker,scratch,kubelet
	Name string `json:"name"`

	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:ExclusiveMinimum=false
	Size int `json:"size"`
}

// FileSystemList defines a type to represent a slice of host filesystem
// resources.
// +deepequal-gen:unordered-array=true
type FileSystemList []FileSystemInfo

// ProfileStorageInfo defines the storage specific attributes for the host.
// +deepequal-gen:ignore-nil-fields=true
type ProfileStorageInfo struct {
	// Monitor defines whether a Ceph storage monitor should be enabled on a
	// node.
	// +optional
	Monitor *MonitorInfo `json:"monitor,omitempty"`

	// OSDs defines the list of OSD devices to be created on the host.  This is
	// only applicable to storage related nodes.
	// +optional
	OSDs *OSDList `json:"osds,omitempty"`

	// VolumeGroups defines the list of volume groups to be created on the host.
	// +optional
	VolumeGroups *VolumeGroupList `json:"volumeGroups,omitempty"`

	// FileSystems defines the list of file systems to be defined on the host.
	// +optional
	FileSystems *FileSystemList `json:"filesystems,omitempty"`
}

// EthernetPortInfo defines the attributes specific to a single
// Ethernet port.
type EthernetPortInfo struct {
	// SystemName defines the device name of the Ethernet port.
	// +kubebuilder:validation:MaxLength=255
	// +kubebuilder:validation:Pattern=^[a-zA-Z0-9\-_]+$
	Name string `json:"name"`
}

// CommonInterfaceInfo defines the attributes common to all interface
// types.  They are defined once, here,
// and inlined within each of the different interface type structures.
// +deepequal-gen:ignore-nil-fields=true
type CommonInterfaceInfo struct {
	// SystemName defines the name of the interface to be configured.
	// +kubebuilder:validation:MaxLength=255
	// +kubebuilder:validation:Pattern=^[a-zA-Z0-9\-_\.]+$
	Name string `json:"name"`

	// Class defines the intended usage of this interface by the system.
	// +kubebuilder:validation:Enum=platform,data,pci-sriov,pci-passthrough,none
	Class string `json:"class"`

	// MTU defines the maximum transmit unit for this interface.
	// +kubebuilder:validation:Minimum=576
	// +kubebuilder:validation:Maximum=9216
	// +optional
	MTU *int `json:"mtu,omitempty"`

	// PlatformNetworks defines the list of platform networks to be configured
	// against this interface.
	// +kubebuilder:validation:MaxLength=255
	// +kubebuilder:validation:Pattern=^[a-zA-Z0-9\-_]+$
	// +optional
	PlatformNetworks *StringList `json:"platformNetworks,omitempty"`

	// DataNetworks defines the list of data networks to be configured against
	// this interface.
	// +kubebuilder:validation:MaxLength=255
	// +kubebuilder:validation:Pattern=^[a-zA-Z0-9\-_]+$
	// +optional
	DataNetworks *StringList `json:"dataNetworks,omitempty"`
}

// EthernetInfo defines the attributes specific to a single
// Ethernet interface.
// +deepequal-gen:ignore-nil-fields=true
type EthernetInfo struct {
	// CommonInterfaceInfo defines attributes common to all interface
	// types.
	CommonInterfaceInfo `json:",inline"`

	// VFCount defines the number of SRIOV VF interfaces to be allocated.  Only
	// applicable if the interface class is set to "pci-sriov".
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=128
	// +optional
	VFCount *int `json:"vfCount,omitempty"`

	// VFDriver defines the device driver to be associated with each individual
	// SRIOV VF interface allocated.  Only applicable if the interface class is
	// set to "pci-sriov".
	VFDriver *string `json:"vfDriver,omitempty"`

	// Port defines the attributes identifying the underlying port which defines
	// this Ethernet interface.
	Port EthernetPortInfo `json:"port"`
}

// EthernetList defines a type to represent a slice of ethernet interfaces.
// +deepequal-gen:unordered-array=true
type EthernetList []EthernetInfo

// VLANInfo defines the attributes specific to a single VLAN
// interface.
type VLANInfo struct {
	// CommonInterfaceInfo defines attributes common to all interface
	// types.
	CommonInterfaceInfo `json:",inline"`

	// Lower defines the interface name over which this VLAN interface is to be
	// configured.
	// +kubebuilder:validation:Pattern=^[a-zA-Z0-9\-_\.]+$
	Lower string `json:"lower"`

	// VID defines the VLAN ID value to be assigned to this VLAN interface.
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=4095
	VID int `json:"vid"`
}

// VLANList defines a type to represent a slice of VLAN interfaces.
// +deepequal-gen:unordered-array=true
type VLANList []VLANInfo

// BondInfo defines the attributes specific to a single Bond
// interface.
type BondInfo struct {
	// CommonInterfaceInfo defines attributes common to all interface
	// types.
	CommonInterfaceInfo `json:",inline"`

	// Members defines the list of interfaces which, together, make up the Bond
	// interface.
	// +kubebuilder:validation:MaxLength=255
	// +kubebuilder:validation:Pattern=^[a-zA-Z0-9\-_\.]+$
	Members StringList `json:"members"`

	// Mode defines the Bond interface aggregation mode.
	// +kubebuilder:validation:Enum=balanced,active-standby,802.3ad
	Mode string `json:"mode"`

	// TransmitHashPolicy defines the transmit interface selection policy for
	// the Bond interface.  Only applicable for 802.3ad and balanced modes.
	// +kubebuilder:validation:Enum=layer2,layer2+3
	// +optional
	TransmitHashPolicy *string `json:"transmitHashPolicy,omitempty"`
}

// BondList defines a type to represent a slice of Bond interfaces.
// +deepequal-gen:unordered-array=true
type BondList []BondInfo

// InterfaceInfo defines the attributes specific to a single
// interface.
type InterfaceInfo struct {
	// Ethernet defines the list of ethernet interfaces to be configured on a
	// host.
	Ethernet EthernetList `json:"ethernet,omitempty" patchStrategy:"merge" patchMergeKey:"name"`

	// VLAN defines the list of VLAN interfaces to be configured on a host.
	VLAN VLANList `json:"vlan,omitempty"`

	// Bond defines the list of Bond interfaces to be configured on a host.
	Bond BondList `json:"bond,omitempty"`
}

// AddressInfo defines the attributes specific to a single address.
type AddressInfo struct {
	// Interface is a reference to the interface name against which to configure
	// the address.
	// +kubebuilder:validation:MaxLength=255
	// +kubebuilder:validation:Pattern=^[a-zA-Z0-9\-_\.]+$
	Interface string `json:"interface"`

	// Address defines the IPv4 or IPv6 address value.
	Address string `json:"address"`

	// Prefix defines the IP address network prefix length.
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=128
	Prefix int `json:"prefix"`
}

// AddressList defines a type to represent a slice of addresses.
// +deepequal-gen:unordered-array=true
type AddressList []AddressInfo

// RouteInfo defines the attributes specific to a single route.
// +deepequal-gen:ignore-nil-fields=true
type RouteInfo struct {
	// Interface is a reference to the interface name against which to configure
	// the route.
	// +kubebuilder:validation:MaxLength=255
	// +kubebuilder:validation:Pattern=^[a-zA-Z0-9\-_\.]+$
	Interface string `json:"interface"`

	// Subnet defines the destination network address subnet.
	Network string `json:"subnet"`

	// Prefix defines the destination network address prefix length.
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=128
	Prefix int `json:"prefix"`

	// Gateway defines the next hop gateway IP address.
	Gateway string `json:"gateway"`

	// Metric defines the route preference metric for this route.
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=255
	// +optional
	Metric *int `json:"metric,omitempty"`
}

// RouteList defines a type to represent a slice of routes.
// +deepequal-gen:unordered-array=true
type RouteList []RouteInfo

// IsKeyEqual compares two processor info array elements and determines if they
// refer to the same instance.  All other attributes will be merged during
// profile merging.
func (in ProcessorInfo) IsKeyEqual(x ProcessorInfo) bool {
	return in.Node == x.Node
}

// IsKeyEqual compares two processor function array elements and determines if
// they refer to the same instance.  All other attributes will be merged during
// profile merging.
func (in ProcessorFunctionInfo) IsKeyEqual(x ProcessorFunctionInfo) bool {
	return in.Function == x.Function
}

// IsKeyEqual compares two memory info array elements and determines if they
// refer to the same instance.  All other attributes will be merged during
// profile merging.
func (in MemoryNodeInfo) IsKeyEqual(x MemoryNodeInfo) bool {
	return in.Node == x.Node
}

// IsKeyEqual compares two memory function array elements and determines if they
// refer to the same instance.  All other attributes will be merged during
// profile merging.
func (in MemoryFunctionInfo) IsKeyEqual(x MemoryFunctionInfo) bool {
	return in.Function == x.Function && in.PageSize == x.PageSize
}

// IsKeyEqual compares two storage OSD info array elements and determines if
// they refer to the same instance.  All other attributes will be merged during
// profile merging.
func (in OSDInfo) IsKeyEqual(x OSDInfo) bool {
	return in.Path == x.Path
}

// IsKeyEqual compares two storage volume array elements and determines if they
// refer to the same instance.  All other attributes will be merged during
// profile merging.
func (in VolumeGroupInfo) IsKeyEqual(x VolumeGroupInfo) bool {
	return in.Name == x.Name
}

// IsKeyEqual compares two storage file system array elements and determines if
// they refer to the same instance.  All other attributes will be merged during
// profile merging.
func (in FileSystemInfo) IsKeyEqual(x FileSystemInfo) bool {
	return in.Name == x.Name
}

// IsKeyEqual compares two ethernet interface array elements and determines if
// they refer to the same instance.  All other attributes will be merged during
// profile merging.
func (in EthernetInfo) IsKeyEqual(x EthernetInfo) bool {
	// Ethernet interfaces can be renamed but only a single interface can refer
	// to a unique port name
	return in.Port.Name == x.Port.Name
}

// IsKeyEqual compares two VLAN interface array elements and determines if they
// refer to the same instance.  All other attributes will be merged during
// profile merging.
func (in VLANInfo) IsKeyEqual(x VLANInfo) bool {
	return in.Name == x.Name && in.Class == x.Class
}

// IsKeyEqual compares two Bond interface array elements and determines if they
// refer to the same instance.  All other attributes will be merged during
// profile merging.
func (in BondInfo) IsKeyEqual(x BondInfo) bool {
	return in.Name == x.Name && in.Class == x.Class
}

// IsKeyEqual compares two interface address array elements and determines if
// they refer to the same instance.  All other attributes will be merged during
// profile merging.
func (in AddressInfo) IsKeyEqual(x AddressInfo) bool {
	// Addresses should be unique on a node therefore if an address is moved
	// to a different interface then merging two profiles should keep the
	// address info but merge the interface name.
	return in.Address == x.Address
}

// IsKeyEqual compares two interface route array elements and determines if
// they refer to the same instance.  All other attributes will be merged during
// profile merging.
func (in RouteInfo) IsKeyEqual(x RouteInfo) bool {
	// Routes can be duplicated on a node but must be unique on an interface
	// therefore to allow a routes metric or gateway to be changed we match
	// on interface + network + prefix.
	if in.Interface == x.Interface {
		if in.Network == x.Network {
			if in.Prefix == x.Prefix {
				return true
			}
		}
	}
	return false
}

// +deepequal-gen:ignore-nil-fields=true
type ProfileBaseAttributes struct {
	// Personality defines the role to be assigned to the host
	// +kubebuilder:validation:Enum=controller,worker,storage,controller-worker
	// +optional
	Personality *string `json:"personality,omitempty"`

	// AdministrativeState defines the desired administrative state of the host
	// +kubebuilder:validation:Enum=locked,unlocked
	// +optional
	AdministrativeState *string `json:"administrativeState,omitempty"`

	// SubFunctionList defines the set of subfunctions to be provisioned on the
	// node at time of initial provisioning.
	// +kubebuilder:validation:Enum=controller,worker,storage,lowlatency
	// +optional
	SubFunctions StringList `json:"subfunctions,omitempty"`

	// Location defines the physical location of the host in the data centre.
	// +optional
	Location *string `json:"location,omitempty"`

	// Labels defines the set of labels to be applied to the kubernetes node
	// resources that is running on this host.
	Labels map[string]string `json:"labels,omitempty"`

	// InstallOutput defines the install output method.  The graphical mode is
	// only suitable when the console attribute is set to a graphical terminal.
	// The text mode can be used with both serial and graphical console
	// configurations.
	// +kubebuilder:validation:Enum=text,graphical
	// +optional
	InstallOutput *string `json:"installOutput,omitempty"`

	// Console defines the installation output device.
	// +kubebuilder:validation:Pattern=^(|tty[0-9]+|ttyS[0-9]+(,\d+([a-zA-Z0-9]+)?)?|ttyUSB[0-9]+(,\d+([a-zA-Z0-9]+))?|lp[0-9]+)$
	// +optional
	Console *string `json:"console,omitempty"`

	// BootDevice defines the absolute device path of the device to be used for
	// installation.
	// +kubebuilder:validation:Pattern=^/dev/.+$
	// +kubebuilder:validation:MaxLength=4095
	// +optional
	BootDevice *string `json:"bootDevice,omitempty"`

	// PowerOn defines the initial power state of the node if static
	// provisioning is being used.
	// +optional
	PowerOn *bool `json:"powerOn,omitempty"`

	// ProvisioningMode defines whether a host is provisioned dynamically when
	// it appears in system host inventory or whether it is provisioned
	// statically and powered up explicitly.  Statically provisioned hosts
	// require that the user supply a boot MAC address, board management IP
	// address, and a management IP address if the management network is
	// configured for static address assignment.
	// +kubebuilder:validation:Enum=static,dynamic
	// +optional
	ProvisioningMode *string `json:"provisioningMode,omitempty"`

	// BootMAC defines the MAC address that a host uses to perform the initial
	// software installation.  This is only applicable for statically
	// provisioned hosts and should be set on each hosts via the overrides
	// attributes.
	// +kubebuilder:validation:Pattern=^([0-9a-fA-Z]{2}[:-]){5}([0-9a-fA-Z]{2})$
	// +optional
	BootMAC *string `json:"bootMAC,omitempty"`

	// RootDevice defines the absolute device path of the device to be used as
	// the root file system.
	// +kubebuilder:validation:Pattern=^/dev/.+$
	// +kubebuilder:validation:MaxLength=4095
	// +optional
	RootDevice *string `json:"rootDevice,omitempty"`

	// ClockSynchronization defines the clock synchronization source of the host
	// resource.
	// +kubebuilder:validation:Enum=ntp,ptp
	// +optional
	ClockSynchronization *string `json:"clockSynchronization,omitempty"`
}

// HostProfileSpec defines the desired state of HostProfile
type HostProfileSpec struct {
	// Base defines the name of another HostProfile from which to inherit
	// attributes.  HostProfiles can be structured in a hierarchy so that many
	// HostProfiles can inherit generic attributes from a parent HostProfile.
	// This hierarchy can be defined in multiple layers; with lower layers
	// overriding attributes set in higher layers.
	//
	// At configuration time, before a Host is configured, the hierarchy of
	// HostProfile resources is flattened to produce a single composite profile
	// that represents the final attributes as they are overridden down the
	// HostProfile hierarchy.
	//
	// Once the HostProfile hierarchy is flattened to a composite profile.  The
	// Deployment Manager will further refine the profile to create a final
	// HostProfile which serves as the final configuration for the Host
	// resource.  To create the final HostProfile, the Deployment Manages merges
	// the composite profile with the initial default host attributes, and then
	// merges the individual host overrides into that result.  The process
	// can be illustrated as follows:
	//
	//
	//         Host Defaults      +---------------------+
	//            |                                      \
	//            |                                       \
	//         Base Profile        +                       \
	//            |                 \                       \
	//           ...                  + Composite Profile ----+  Final Profile
	//            |                 /                        /
	//      Personality Profile(s) +                        /
	//            |                                        /
	//            |                                       /
	//           Host                                    /
	//            |                                     /
	//            |                                    /
	//         Host Overrides       +-----------------+
	//
	//
	// Merging two HostProfileSpec resources consists of merging the attributes
	// of a higher precedence profile into the attributes of a lower precedences
	// profile.  The rules for merging attributes are as follows.
	//
	//   1) A nil pointer is always overwritten by a non-nil pointer.
	//
	//   2) Two non-nil pointers are merged together according to the underlying
	//      type.
	//
	//   2a) If the type pointed to is a primitive type (e.g., int, bool,
	//       string, etc) then the higher precedence value is used).
	//
	//   2b) If the type pointed to is a structure then this same merge
	//       procedure is repeated recursively on each field of the structure
	//       with these same rules applying to each field.
	//
	//   2c) If the type pointed to is a slice/array then rule (3) is used.
	//
	//   2d) If the type pointed to is a map then higher precedence value is
	//       used and the entire map is overwritten.
	//
	//   3) Two slices are merged together using the following sub-rules.
	//
	//   3a) If the elements of slices define the KeyEqual() method then an
	//       attempt is made to try to merge equivalent element using this same
	//       merge strategy.  Elements from the higher precedence list that do
	//       not have an equivalent in the lower precedence list are appended to
	//       the list.  Elements appearing in the lower precedence list but not
	//       in the higher precedence list are kept intact.
	//
	//   3b) If the elements of the slices do not define the KeyEqual() method
	//       then they are simply concatenated together.
	//
	//   3c) An empty slice is handled as a special case that deletes the
	//       contents of the lower precedence slice.  Do not confuse an empty
	//       slice with a nil slice pointer.
	//
	// +optional
	Base *string `json:"base,omitempty"`

	// ProfileBaseAttributes defines the node level base attributes.  They are
	// grouped together to take advantage of the code generated DeepEqual
	// method to facilitate comparisons.
	ProfileBaseAttributes `json:",inline"`

	// BoardManagement defines the attributes specific to the board management
	// controller configuration.
	// +optional
	BoardManagement *BMInfo `json:"boardManagement,omitempty"`

	// Processors defines the core allocations for each function across all NUMA
	// sockets/nodes.
	Processors ProcessorNodeList `json:"processors,omitempty"`

	// Memory defines the memory allocations for each function across all NUMA
	// sockets/nodes.
	Memory MemoryNodeList `json:"memory,omitempty"`

	// Storage defines the storage attributes for the host
	// +optional
	Storage *ProfileStorageInfo `json:"storage,omitempty"`

	// Interfaces defines the list of interfaces to be configured against this
	// host.
	// +optional
	Interfaces *InterfaceInfo `json:"interfaces,omitempty"`

	// Addresses defines the list of addresses to be configured against this
	// host.  Addresses are specific to a single host therefore they should only
	// be specified if this profile is only going to be used to configure a
	// single
	// host.
	Addresses AddressList `json:"addresses,omitempty"`

	// Routes defines the list of routes to be configured against this host.
	// Routes require that the target interface be configured with a suitable
	// address (e.g., one that allows reachability to next hop device(s))
	// therefore the host must be configured with valid addresses or configured
	// to for automatic address assignment from a platform network.
	Routes RouteList `json:"routes,omitempty"`
}

// HasWorkerSubfunction is a utility function that returns true if a profile
// is configured to require the compute subfunction.
func (in *HostProfileSpec) HasWorkerSubFunction() bool {
	if in.Personality != nil && *in.Personality == hosts.PersonalityWorker {
		return true
	}

	for _, s := range in.SubFunctions {
		if s == hosts.SubFunctionWorker {
			return true
		}
	}

	return false
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// HostProfile defines the attributes that represent the host level
// attributes of a StarlingX system.  This is represents the bulk of the
// system API attributes and is the most complex part of the schema definition.
// Refer the full list of API documentation here:
//
//   https://docs.starlingx.io/api-ref/stx-config/index.html
//
// +k8s:openapi-gen=true
// +deepequal-gen=false
// +kubebuilder:printcolumn:name="base",type="string",JSONPath=".spec.base",description="The parent host profile."
type HostProfile struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec HostProfileSpec `json:"spec,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// HostProfileList contains a list of HostProfile
// +deepequal-gen=false
type HostProfileList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []HostProfile `json:"items"`
}

func init() {
	SchemeBuilder.Register(&HostProfile{}, &HostProfileList{})
}
