/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019-2023 Wind River Systems, Inc. */

package v1

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/alecthomas/units"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/addresspools"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/certificates"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/controllerFilesystems"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/cpus"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/datanetworks"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/hostFilesystems"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/hosts"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/interfaces"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/licenses"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/memory"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/networks"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/physicalvolumes"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/ptpinstances"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/ptpinterfaces"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/serviceparameters"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/storagebackends"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/volumegroups"
	v1info "github.com/wind-river/cloud-platform-deployment-manager/platform"
	v1 "k8s.io/api/core/v1"
	v1types "k8s.io/apimachinery/pkg/apis/meta/v1"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var log = logf.Log.WithName("controller").WithName("host")

const (
	ControllerToolsLabel   = "controller-tools.k8s.io"
	ControllerToolsVersion = "1.0"
)

const (
	// Secret map key names.
	SecretUsernameKey = "username"
	SecretPasswordKey = "password"
)

// group defines the current in use API group.
const Group = "starlingx.windriver.com"

// version defines the curent in use API version.
const Version = "v1"

const APIVersion = Group + "/" + Version

// Defines the current list of resource kinds.
const (
	KindHost            = "Host"
	KindHostProfile     = "HostProfile"
	KindPlatformNetwork = "PlatformNetwork"
	KindDataNetwork     = "DataNetwork"
	KindSystem          = "System"
	KindPTPInstance     = "PtpInstance"
	KindPTPInterface    = "PtpInterface"
)

type PageSize string

// Defines the accepted hugepage memory page sizes.
const (
	PageSize4K PageSize = "4KB"
	PageSize2M PageSize = "2MB"
	PageSize1G PageSize = "1GB"
)

// Bytes returns the page size in bytes.
func (v PageSize) Bytes() int {
	switch v {
	case PageSize1G:
		return int(units.Gibibyte)
	case PageSize2M:
		return 2 * int(units.Mebibyte)
	case PageSize4K:
		return 4 * int(units.KiB)
	}

	// This is never expected to happen so no error is returned.
	return 0
}

// Gibibytes returns the page size in megabytes.
func (v PageSize) Megabytes() int {
	switch v {
	case PageSize1G:
		return 1024
	case PageSize2M:
		return 2
	}

	// This is never expected to happen so no error is returned.
	return 0
}

// Defines the valid host provisioning modes
const (
	ProvioningModeStatic  = "static"
	ProvioningModeDynamic = "dynamic"
)

// Defines the default Secret name used for tracking license files.
const SystemDefaultLicenseName = "system-license"

// ErrMissingSystemResource defines an error to be used when reporting that
// an operation is unable to find a required system resource from the
// system API.  This error is not intended for kubernetes resources that are
// missing.  For those use ErrMissingKubernetesResource
type ErrMissingSystemResource struct {
	message string
}

// Error returns the message associated with an error of this type.
func (in ErrMissingSystemResource) Error() string {
	return in.message
}

// NewMissingSystemResource defines a constructor for the
// ErrMissingSystemResource error type.
func NewMissingSystemResource(msg string) error {
	return ErrMissingSystemResource{msg}
}

// stripPartitionNumber is a utility function that removes the "-partNNN" suffix
// from the partition device path.
func stripPartitionNumber(path string) string {
	re := regexp.MustCompile("-part[0-9]*")
	return re.ReplaceAllString(path, "")
}

// parseLabelInfo is a utility which parses the label data as it is presented
// by the system API and stores the data in the form required by a profile spec.
func parseLabelInfo(profile *HostProfileSpec, host v1info.HostInfo) error {
	result := make(map[string]string)
	for _, l := range host.Labels {
		result[l.Key] = l.Value
	}

	if len(result) > 0 {
		profile.Labels = result
	}

	return nil
}

// parseProcessorInfo is a utility which parses the CPU data as it is presented
// by the system API and stores the data in the form required by a profile spec.
func parseProcessorInfo(profile *HostProfileSpec, host v1info.HostInfo) error {
	var result []ProcessorInfo

	// First, organize the data by node and function.
	nodes := make(map[int]map[string]int)
	for _, c := range host.CPU {
		if c.Thread != 0 {
			// Processor configurations are always done on a physical core
			// basis so do not include hyper-thread cores.
			continue
		}

		function := strings.ToLower(c.Function)
		if function == cpus.CPUFunctionApplication {
			// These cannot be configured.  They are simply a placeholder
			// for those CPUs that are not allocated for any other function.
			continue
		}

		if f, ok := nodes[c.Processor]; !ok {
			nodes[c.Processor] = map[string]int{function: 1}
			if profile.HasWorkerSubFunction() && function != cpus.CPUFunctionVSwitch && c.Processor == 0 {
				// Always add the vswitch function since if it is set to 0
				// it won't show up in the list and will be missing from the
				// profile that we create.
				nodes[c.Processor][cpus.CPUFunctionVSwitch] = 0
			}
		} else {
			f[function] = f[function] + 1
		}
	}

	// Second, prepare the final data by converting the maps to arrays.
	for key := range nodes {
		node := ProcessorInfo{
			Node: key,
		}

		for function, count := range nodes[key] {
			data := ProcessorFunctionInfo{
				Function: strings.ToLower(function),
				Count:    count,
			}
			node.Functions = append(node.Functions, data)
		}

		result = append(result, node)
	}

	profile.Processors = result

	return nil
}

// parseMemoryInfo is a utility which parses the memory data as it is presented
// by the system API and stores the data in the form required by a profile spec.
func parseMemoryInfo(profile *HostProfileSpec, host v1info.HostInfo) error {
	var result []MemoryNodeInfo

	for _, m := range host.Memory {
		info := MemoryNodeInfo{
			Node: m.Processor,
		}

		// Platform memory allocations
		platform := MemoryFunctionInfo{
			Function:  memory.MemoryFunctionPlatform,
			PageSize:  string(PageSize4K),
			PageCount: (m.Platform * int(units.Mebibyte)) / PageSize4K.Bytes(),
		}
		info.Functions = append(info.Functions, platform)

		if profile.HasWorkerSubFunction() {
			// VSwitch memory allocations
			vswitch := MemoryFunctionInfo{
				Function:  memory.MemoryFunctionVSwitch,
				PageCount: m.VSwitchHugepagesCount,
			}
			if m.VSwitchHugepagesSize == PageSize2M.Megabytes() {
				vswitch.PageSize = string(PageSize2M)
			} else if m.VSwitchHugepagesSize == PageSize1G.Megabytes() {
				vswitch.PageSize = string(PageSize1G)
			} else {
				vswitch.PageSize = string(PageSize4K)
			}
			info.Functions = append(info.Functions, vswitch)

			// VM memory allocations
			vm2m := MemoryFunctionInfo{
				Function: memory.MemoryFunctionVM,
				PageSize: string(PageSize2M),
			}
			if m.VM2MHugepagesPending == nil {
				vm2m.PageCount = m.VM2MHugepagesCount
			} else {
				vm2m.PageCount = *m.VM2MHugepagesPending
			}

			if m.VSwitchHugepagesSize == PageSize2M.Megabytes() {
				// The system API does not properly report the 2M pages that are
				// exclusively reserved for VM use.  If vswitch is also using
				// 2M pages then its total is lumped in with the VM total so
				// we need to separate them.
				// TODO(alegacy): This needs to be fixed whenever the system
				//  api reports unique values.
				if vm2m.PageCount >= vswitch.PageCount {
					// On initial provisioning the memory does not seem to be
					// accounted for properly so only do this if it does not
					// result in a negative error.
					vm2m.PageCount -= vswitch.PageCount
				}
			}

			info.Functions = append(info.Functions, vm2m)

			vm1g := MemoryFunctionInfo{
				Function: memory.MemoryFunctionVM,
				PageSize: string(PageSize1G),
			}
			if m.VM1GHugepagesPending == nil {
				vm1g.PageCount = m.VM1GHugepagesCount
			} else {
				vm1g.PageCount = *m.VM1GHugepagesPending
			}
			info.Functions = append(info.Functions, vm1g)
		}

		result = append(result, info)
	}

	profile.Memory = result

	return nil
}

// parseInterfaceInfo is a utility which parses the interface data as it is
// presented by the system API and stores the data in the form required by a
// profile spec.
func parseInterfaceInfo(profile *HostProfileSpec, host v1info.HostInfo) error {
	result := InterfaceInfo{}
	ethernets := make([]EthernetInfo, 0)
	bonds := make([]BondInfo, 0)
	vlans := make([]VLANInfo, 0)
	vfs := make([]VFInfo, 0)

	for _, iface := range host.Interfaces {
		data := CommonInterfaceInfo{
			Name:  iface.Name,
			Class: iface.Class,
		}

		mtu := iface.MTU
		data.MTU = &mtu

		if iface.Class == "" {
			data.Class = interfaces.IFClassNone
		}

		nets := host.BuildInterfaceNetworkList(iface)

		if iface.IPv4Pool != nil {
			// TODO(alegacy): platform networks of type "other" exist to map
			//  address pools to data interfaces.  This is (hopefully) a
			//  temporary measure until we can support these as actual networks.
			pool := host.FindAddressPool(*iface.IPv4Pool)
			if pool != nil {
				nets = append(nets, pool.Name)
			}
		}

		if iface.IPv6Pool != nil {
			// TODO(alegacy): platform networks of type "other" exist to map
			//  address pools to data interfaces.  This is (hopefully) a
			//  temporary measure until we can support these as actual networks.
			pool := host.FindAddressPool(*iface.IPv6Pool)
			if pool != nil {
				nets = append(nets, pool.Name)
			}
		}

		netList := StringsToPlatformNetworkItemList(nets)
		data.PlatformNetworks = &netList

		dataNets := host.BuildInterfaceDataNetworkList(iface)

		dataNetList := StringsToDataNetworkItemList(dataNets)
		data.DataNetworks = &dataNetList

		data.PTPRole = iface.PTPRole

		dataPtpInterfaceList := host.FindPTPInterfaceNameByInterface(iface)
		dataPtpInterfaces := StringsToPtpInterfaceItemList(dataPtpInterfaceList)
		data.PtpInterfaces = &dataPtpInterfaces

		switch iface.Type {
		case interfaces.IFTypeEthernet:
			var ethernet EthernetInfo
			if len(iface.Uses) > 0 {
				ethernet = EthernetInfo{
					Lower: iface.Uses[0],
					Port:  EthernetPortInfo{Name: "dummy"}}
			} else {
				portname, found := host.FindInterfacePortName(iface.ID)
				if !found {
					msg := fmt.Sprintf("unable to find port name for interface id %s", iface.ID)
					return NewMissingSystemResource(msg)
				}
				ethernet = EthernetInfo{
					Port: EthernetPortInfo{
						Name: portname}}
			}

			ethernet.CommonInterfaceInfo = data

			if strings.EqualFold(iface.Class, interfaces.IFClassPCISRIOV) {
				ethernet.VFCount = iface.VFCount
				ethernet.VFDriver = iface.VFDriver
			}

			ethernets = append(ethernets, ethernet)

		case interfaces.IFTypeVLAN:
			vlan := VLANInfo{
				VID:   *iface.VID,
				Lower: iface.Uses[0]}
			vlan.CommonInterfaceInfo = data
			vlans = append(vlans, vlan)

		case interfaces.IFTypeAE:
			bond := BondInfo{
				Mode:               *iface.AEMode,
				TransmitHashPolicy: iface.AETransmitHash,
				PrimaryReselect:    iface.AEPrimReselect,
				Members:            iface.Uses}
			bond.CommonInterfaceInfo = data
			bonds = append(bonds, bond)

		case interfaces.IFTypeVirtual:
			// Virtual interfaces are only used on AIO-SX systems so manage
			// them as an Ethernet interface for simplicity sake.
			ethernet := EthernetInfo{
				Port: EthernetPortInfo{
					Name: data.Name}}
			ethernet.CommonInterfaceInfo = data
			ethernets = append(ethernets, ethernet)

		case interfaces.IFTypeVF:

			vf := VFInfo{
				VFCount:   *iface.VFCount,
				Lower:     iface.Uses[0],
				VFDriver:  iface.VFDriver,
				MaxTxRate: iface.MaxTxRate}
			vf.CommonInterfaceInfo = data
			vfs = append(vfs, vf)
		}
	}

	if len(ethernets) > 0 {
		result.Ethernet = ethernets
	}

	if len(vlans) > 0 {
		result.VLAN = vlans
	}

	if len(bonds) > 0 {
		result.Bond = bonds
	}

	if len(vfs) > 0 {
		result.VF = vfs
	}

	profile.Interfaces = &result

	return nil
}

// parseAddressInfo is a utility which parses the address data as it is
// presented by the system API and stores the data in the form required by a
// profile spec.
func parseAddressInfo(profile *HostProfileSpec, host v1info.HostInfo) error {
	result := make([]AddressInfo, 0)

	for _, a := range host.Addresses {
		if host.IsSystemAddress(&a) {
			// ignore these because they appear after creating or modifying
			// interfaces and that makes it difficult to compare the current
			// config to the desired profile because it always looks like we
			// need to deal with a difference in the address list.
			continue
		}

		address := AddressInfo{
			Interface: a.InterfaceName,
			Address:   a.Address,
			Prefix:    a.Prefix,
		}
		result = append(result, address)
	}

	if len(result) > 0 {
		profile.Addresses = result
	}

	return nil
}

// parseRouteInfo is a utility which parses the route data as it is presented
// by the system API and stores the data in the form required by a profile spec.
func parseRouteInfo(profile *HostProfileSpec, host v1info.HostInfo) error {
	result := make([]RouteInfo, len(host.Routes))

	for i, r := range host.Routes {
		metric := r.Metric
		route := RouteInfo{
			Interface: r.InterfaceName,
			Network:   r.Network,
			Prefix:    r.Prefix,
			Gateway:   r.Gateway,
			Metric:    &metric,
		}
		result[i] = route
	}

	if len(result) > 0 {
		profile.Routes = result
	}

	return nil
}

// parsePhysicalVolumeInfo is a utility which parses the physical volume data as
// it is presented by the system API and stores the data in the form required by
// a profile spec.
func parsePhysicalVolumeInfo(group *VolumeGroupInfo, vg *volumegroups.VolumeGroup, host v1info.HostInfo) error {
	result := make([]PhysicalVolumeInfo, 0)

	for _, pv := range host.PhysicalVolumes {
		if pv.VolumeGroupID != vg.ID {
			continue
		}

		physicalVolume := PhysicalVolumeInfo{
			Type: pv.Type,
			Path: pv.DevicePath,
		}

		if pv.Type == physicalvolumes.PVTypePartition {
			if partition, ok := host.FindPartition(pv.DeviceUUID); ok {
				size := partition.Gibibytes()
				physicalVolume.Size = &size
				physicalVolume.Path = stripPartitionNumber(partition.DevicePath)
			} else {
				msg := fmt.Sprintf("failed to lookup partition %s", pv.DeviceUUID)
				return NewMissingSystemResource(msg)
			}
		}

		result = append(result, physicalVolume)
	}

	group.PhysicalVolumes = result

	return nil
}

// parsePartitionInfo is a utility which parses the partition data as it is
// presented by the system API and stores the data in the form required by a
// profile spec.
func parseVolumeGroupInfo(profile *HostProfileSpec, host v1info.HostInfo) error {
	result := make([]VolumeGroupInfo, len(host.VolumeGroups))

	for i, vg := range host.VolumeGroups {
		group := VolumeGroupInfo{
			Name: vg.Name,
		}

		if value := vg.Capabilities.LVMType; value != nil {
			lvmType := *value
			group.LVMType = &lvmType
		}

		err := parsePhysicalVolumeInfo(&group, &vg, host)
		if err != nil {
			return err
		}

		result[i] = group
	}

	if len(result) > 0 {
		list := VolumeGroupList(result)
		profile.Storage.VolumeGroups = &list
	}

	return nil
}

// parseOSDInfo is a utility which parses the OSD data as it is presented by the
// system API and stores the data in the form required by a profile spec.
func parseOSDInfo(profile *HostProfileSpec, host v1info.HostInfo) error {
	result := make([]OSDInfo, 0)

	for _, o := range host.OSDs {
		osd := OSDInfo{
			Function: o.Function,
		}

		clusterName, found := host.FindClusterNameByTier(o.TierUUID)
		if found {
			osd.ClusterName = &clusterName
		}

		disk, _ := host.FindDisk(o.DiskID)
		if disk == nil {
			log.Info("unable to find disk for OSD", "uuid", o.ID)
			continue // skip
		}

		osd.Path = disk.DevicePath

		if o.JournalInfo.Location != nil && *o.JournalInfo.Location != o.ID {
			// If the journal points to a separate OSD then use that information
			// to populate the profile info; otherwise if the journal is
			// pointing to itself then there is no need to save that in the
			// profile because that is system generated.
			if o.JournalInfo.Path != nil {
				path := stripPartitionNumber(*o.JournalInfo.Path)
				journal := JournalInfo{
					Location: path,
					Size:     o.JournalInfo.Gibibytes(),
				}
				osd.Journal = &journal
			} else {
				log.Info("unexpected nil OSD journal path", "uuid", o.ID)
			}
		}

		result = append(result, osd)
	}

	if len(result) > 0 {
		list := OSDList(result)
		profile.Storage.OSDs = &list
	}

	return nil
}

// parseMonitorInfo is a utility which parses the Ceph Monitor data as it is
// presented by the system API and stores the data in the form required by a
// profile spec.
func parseMonitorInfo(profile *HostProfileSpec, host v1info.HostInfo) error {
	for _, m := range host.Monitors {
		if m.Hostname == host.Hostname {
			size := m.Size
			profile.Storage.Monitor = &MonitorInfo{
				Size: &size,
			}

			return nil
		}
	}

	return nil
}

func parseHostFileSystemInfo(spec *HostProfileSpec, fileSystems []hostFilesystems.FileSystem) error {
	result := make([]FileSystemInfo, 0)

	for _, fs := range fileSystems {
		info := FileSystemInfo{
			Name: fs.Name,
			Size: fs.Size,
		}

		result = append(result, info)
	}

	list := FileSystemList(result)
	spec.Storage.FileSystems = &list

	return nil
}

// parseStorageInfo is a utility which parses the storage data as it is
// presented by the system API and stores the data in the form required by a
// profile spec.
func parseStorageInfo(profile *HostProfileSpec, host v1info.HostInfo) error {
	var err error

	storage := ProfileStorageInfo{}
	profile.Storage = &storage

	if host.Personality == hosts.PersonalityWorker {
		// The monitors on the controllers are handled automatically so to avoid
		// creating differences between the controller profiles and current
		// configurations just avoid adding these to the dynamic profiles.
		err = parseMonitorInfo(profile, host)
		if err != nil {
			return err

		}
	}

	// Fill-in partition attributes
	err = parseVolumeGroupInfo(profile, host)
	if err != nil {
		return err
	}

	// Fill-in partition attributes
	err = parseOSDInfo(profile, host)
	if err != nil {
		return err
	}

	// Fill-in filesystem attributes
	err = parseHostFileSystemInfo(profile, host.FileSystems)
	if err != nil {
		return err
	}

	if storage.OSDs == nil && storage.VolumeGroups == nil && storage.FileSystems == nil {
		profile.Storage = nil
	}

	return nil
}

// autoGenerateBMSecretName is utility which generates a host specific secret
// name to refer to the board management credentials for that specific host.
// NOTE: for now we are only going to generate a single secret for all nodes.
// If customization is required the user can manually clone what they need.
func autoGenerateBMSecretName() string {
	return "bmc-secret"
}

// parseBoardManagementInfo is a utility which parses the board management data
// as it is presented by the system API and stores the data in the form required
// by a profile spec.  Since the credentials are only partially presented by
// the API they are not stored in the profile.
func parseBoardManagementInfo(profile *HostProfileSpec, host v1info.HostInfo) error {
	if host.BMType != nil {
		info := BMInfo{
			Type: host.BMType,
		}

		if host.BMAddress != nil {
			info.Address = host.BMAddress
		}

		if host.BMUsername != nil {
			info.Credentials = &BMCredentials{
				Password: &BMPasswordInfo{
					Secret: autoGenerateBMSecretName()},
			}
		}

		profile.BoardManagement = &info
	} else {
		bmType := "none"
		info := BMInfo{
			Type:        &bmType,
			Address:     nil,
			Credentials: nil,
		}

		profile.BoardManagement = &info
	}

	return nil
}

func NewNamespace(name string) (*v1.Namespace, error) {
	namespace := v1.Namespace{
		TypeMeta: v1types.TypeMeta{
			APIVersion: "v1",
			Kind:       "Namespace",
		},
		ObjectMeta: v1types.ObjectMeta{
			Name: name,
		},
	}

	return &namespace, nil
}

// FixDevicePath is a utility function that take a legacy formatted device
// path (e.g., sda or /dev/sda) and convert it to the newer format which is
// more explicit
// (e.g., /dev/disk/by-path/pci-0000:00:14.0-usb-0:1:1.0-scsi-0:0:0:0).
func FixDevicePath(path string, host v1info.HostInfo) string {
	// device path starts from /dev/disk/*
	formPath := regexp.MustCompile(`(?s)^/dev/disk/.*`)
	// e.g. sda
	shortFormNode := regexp.MustCompile(`(?s)^\w+$`)

	var searchPath string
	if formPath.MatchString(path) {
		// Find the device path
		return path
	} else if shortFormNode.MatchString(path) {
		// Append /dev/ to the short format to get full format of device node
		searchPath = fmt.Sprintf("/dev/%s", path)
	} else {
		// For the rest formats, likely is a full format of a device node
		searchPath = path
	}

	if disk, ok := host.FindDiskByNode(searchPath); ok {
		return disk.DevicePath
	}

	// No alternative found
	return path
}

const zeroMAC = "00:00:00:00:00:00"

// BuildHostProfile takes the current set of host attributes and builds a
// fake host profile that can be used as a reference for the current settings
// applied to the host.
func NewHostProfileSpec(host v1info.HostInfo) (*HostProfileSpec, error) {
	var err error

	spec := HostProfileSpec{}

	// Fill-in the basic attributes
	spec.Personality = &host.Personality
	subfunctionStrings := strings.Split(host.SubFunctions, ",")
	subfunctions := make([]SubFunction, 0)
	if len(subfunctionStrings) > 0 {
		for _, subfunctionString := range subfunctionStrings {
			subfunctions = append(subfunctions, SubFunctionFromString(subfunctionString))
		}
	}
	spec.SubFunctions = subfunctions
	spec.AdministrativeState = &host.AdministrativeState
	if host.BootMAC != zeroMAC {
		// During initial configuration the first controller has a zero MAC
		// address set as its boot MAC address.  Storing that value in the
		// defaults causes a conflict once the real MAC is setup in the system
		// therefore we continuously try to set it back to the zero MAC but
		// the system rejects it.
		spec.BootMAC = &host.BootMAC
	}
	spec.Console = &host.Console
	spec.InstallOutput = &host.InstallOutput
	if host.HwSettle != "" {
		spec.HwSettle = &host.HwSettle
	}
	if host.AppArmor != "" {
		spec.AppArmor = &host.AppArmor
	}
	if host.MaxCPUMhzConfigured != "" {
		spec.MaxCPUMhzConfigured = &host.MaxCPUMhzConfigured
	}
	if host.Location.Name != nil && *host.Location.Name != "" {
		spec.Location = host.Location.Name
	}

	bootDevice := FixDevicePath(host.BootDevice, host)
	spec.BootDevice = &bootDevice
	rootDevice := FixDevicePath(host.RootDevice, host)
	spec.RootDevice = &rootDevice
	clock := *host.ClockSynchronization
	clockCopied := clock[0:] // Copy ClockSynchronization value
	spec.ClockSynchronization = &clockCopied
	ptpInstances := host.BuildPTPInstanceList()
	ptpInstanceList := StringsToPtpInstanceItemList(ptpInstances)
	spec.PtpInstances = ptpInstanceList

	// Assume that the board is powered on unless there is a clear indication
	// that it is not.
	powerState := true
	if host.AvailabilityStatus == hosts.AvailPowerOff {
		if host.Task == nil || *host.Task != hosts.TaskPoweringOn {
			powerState = false
		}
	} else if host.Task != nil && *host.Task == hosts.TaskPoweringOff {
		powerState = false
	}
	spec.PowerOn = &powerState

	err = parseBoardManagementInfo(&spec, host)
	if err != nil {
		return nil, err
	}

	err = parseLabelInfo(&spec, host)
	if err != nil {
		return nil, err
	}

	if spec.HasWorkerSubFunction() {
		// Fill-in CPU attributes
		err := parseProcessorInfo(&spec, host)
		if err != nil {
			return nil, err
		}

		// Fill-in Memory attributes
		err = parseMemoryInfo(&spec, host)
		if err != nil {
			return nil, err
		}
	}

	// Fill-in Interface attributes
	err = parseInterfaceInfo(&spec, host)
	if err != nil {
		return nil, err
	}

	// Fill-in Address attributes
	err = parseAddressInfo(&spec, host)
	if err != nil {
		return nil, err
	}

	// Fill-in Route attributes
	err = parseRouteInfo(&spec, host)
	if err != nil {
		return nil, err
	}

	// Fill-in Storage attributes
	err = parseStorageInfo(&spec, host)
	if err != nil {
		return nil, err
	}

	return &spec, nil
}

func NewHostProfile(name string, namespace string, hostInfo v1info.HostInfo) (*HostProfile, error) {
	name = fmt.Sprintf("%s-profile", name)
	profile := HostProfile{
		TypeMeta: v1types.TypeMeta{
			APIVersion: APIVersion,
			Kind:       KindHostProfile,
		},
		ObjectMeta: v1types.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				ControllerToolsLabel: ControllerToolsVersion,
			},
		},
	}

	spec, err := NewHostProfileSpec(hostInfo)
	if err != nil {
		return nil, err
	}

	spec.DeepCopyInto(&profile.Spec)

	return &profile, nil
}

func autoGenerateCertName(certType string, certIndex int) string {
	// Kubernetes does not accept underscores in resource names.
	certType = strings.Replace(certType, "_", "-", -1)
	return fmt.Sprintf("%s-cert-secret-%d", certType, certIndex)
}

func parseCertificateInfo(spec *SystemSpec, certificates []certificates.Certificate) error {
	result := make([]CertificateInfo, 0)

	for index, c := range certificates {
		cert := CertificateInfo{
			Type: c.Type,
			// Use a fixed naming so that we can document how we auto-generate
			// a full system description for cloning purposes
			Secret:    autoGenerateCertName(c.Type, index),
			Signature: c.Signature,
		}

		result = append(result, cert)
	}

	list := CertificateList(result)
	spec.Certificates = &list

	return nil
}

func parseServiceParameterInfo(spec *SystemSpec, serviceParams []serviceparameters.ServiceParameter) error {
	result := make([]ServiceParameterInfo, 0)

	for _, sp := range serviceParams {
		info := ServiceParameterInfo{
			Service:     sp.Service,
			Section:     sp.Section,
			ParamName:   sp.ParamName,
			ParamValue:  sp.ParamValue,
			Resource:    sp.Resource,
			Personality: sp.Personality,
		}

		result = append(result, info)
	}

	list := ServiceParameterList(result)
	spec.ServiceParameters = &list

	return nil
}

func parseFileSystemInfo(spec *SystemSpec, fileSystems []controllerFilesystems.FileSystem) error {
	result := make([]ControllerFileSystemInfo, 0)

	for _, fs := range fileSystems {
		info := ControllerFileSystemInfo{
			Name: fs.Name,
			Size: fs.Size,
		}

		result = append(result, info)
	}

	if spec.Storage == nil {
		spec.Storage = &SystemStorageInfo{}
	}

	list := ControllerFileSystemList(result)
	spec.Storage.FileSystems = &list

	return nil
}

func parseStorageBackendInfo(spec *SystemSpec, storageBackends []storagebackends.StorageBackend) error {
	result := make([]StorageBackend, 0)

	for _, sb := range storageBackends {
		rep, _ := strconv.Atoi(sb.Capabilities.Replication)
		info := StorageBackend{
			Name:              sb.Name,
			Type:              sb.Backend,
			Network:           &sb.Network,
			ReplicationFactor: &rep,
		}
		result = append(result, info)
	}

	if spec.Storage == nil {
		spec.Storage = &SystemStorageInfo{}
	}

	list := StorageBackendList(result)
	spec.Storage.Backends = &list

	return nil
}

func parseLicenseInfo(spec *SystemSpec, license *licenses.License) error {
	if license != nil {
		// Populate a Secret name reference but for now don't bother trying
		// to setup the actual Secret with the license data.  That may be
		// necessary some day in order to properly compare the desired config
		// with the current config.
		spec.License = &LicenseInfo{Secret: SystemDefaultLicenseName}
	}

	return nil
}

func NewSystemStatus(systemInfo v1info.SystemInfo) (*SystemStatus, error) {
	status := SystemStatus{}

	if systemInfo.SystemType != "" {
		status.SystemType = systemInfo.SystemType
	}

	if systemInfo.SystemMode != "" {
		status.SystemMode = systemInfo.SystemMode
	}

	return &status, nil
}

func NewSystemSpec(systemInfo v1info.SystemInfo) (*SystemSpec, error) {
	spec := SystemSpec{}

	// Fill-in the basic attributes
	if systemInfo.Location != "" {
		spec.Location = &systemInfo.Location
	}

	if systemInfo.Description != "" {
		spec.Description = &systemInfo.Description
	}

	if systemInfo.Contact != "" {
		spec.Contact = &systemInfo.Contact
	}

	if systemInfo.Latitude != "" {
		spec.Latitude = &systemInfo.Latitude
	}

	if systemInfo.Longitude != "" {
		spec.Longitude = &systemInfo.Longitude
	}

	spec.VSwitchType = &systemInfo.Capabilities.VSwitchType

	if systemInfo.DRBD != nil {
		spec.Storage = &SystemStorageInfo{
			DRBD: &DRBDConfiguration{
				LinkUtilization: systemInfo.DRBD.LinkUtilization,
			},
		}
	}

	if systemInfo.DNS != nil {
		if systemInfo.DNS.Nameservers != "" {
			nameservers := StringsToDNSServerList(strings.Split(systemInfo.DNS.Nameservers, ","))
			spec.DNSServers = &nameservers
		} else {
			empty := StringsToDNSServerList(make([]string, 0))
			spec.DNSServers = &empty
		}
	}

	if systemInfo.NTP != nil {
		if systemInfo.NTP.NTPServers != "" {
			nameservers := StringsToNTPServerList(strings.Split(systemInfo.NTP.NTPServers, ","))
			spec.NTPServers = &nameservers
		} else {
			empty := StringsToNTPServerList(make([]string, 0))
			spec.NTPServers = &empty
		}
	}

	if systemInfo.PTP != nil {
		spec.PTP = &PTPInfo{
			Mode:      &systemInfo.PTP.Mode,
			Transport: &systemInfo.PTP.Transport,
			Mechanism: &systemInfo.PTP.Mechanism,
		}
	}

	if len(systemInfo.Certificates) > 0 {
		err := parseCertificateInfo(&spec, systemInfo.Certificates)
		if err != nil {
			return nil, err
		}
	}

	if len(systemInfo.ServiceParameters) > 0 {
		err := parseServiceParameterInfo(&spec, systemInfo.ServiceParameters)
		if err != nil {
			return nil, err
		}
	}

	if len(systemInfo.FileSystems) > 0 {
		err := parseFileSystemInfo(&spec, systemInfo.FileSystems)
		if err != nil {
			return nil, err
		}
	}

	if len(systemInfo.StorageBackends) > 0 {
		err := parseStorageBackendInfo(&spec, systemInfo.StorageBackends)
		if err != nil {
			return nil, err
		}
	}

	if systemInfo.License != nil {
		err := parseLicenseInfo(&spec, systemInfo.License)
		if err != nil {
			return nil, err
		}
	}

	return &spec, nil
}

func NewSystem(namespace string, name string, systemInfo v1info.SystemInfo) (*System, error) {
	system := System{
		TypeMeta: v1types.TypeMeta{
			APIVersion: APIVersion,
			Kind:       KindSystem,
		},
		ObjectMeta: v1types.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				ControllerToolsLabel: ControllerToolsVersion,
			},
		},
	}

	spec, err := NewSystemSpec(systemInfo)
	if err != nil {
		return nil, err
	}

	spec.DeepCopyInto(&system.Spec)

	status, err := NewSystemStatus(systemInfo)
	if err != nil {
		return nil, err
	}

	status.DeepCopyInto(&system.Status)

	return &system, nil
}

func NewBMSecret(name string, namespace string, username string) (*v1.Secret, error) {
	// It is not possible to reconstruct the password info from a running
	// system so scaffold it and allow the user to fill in the blanks.
	fakePassword := []byte("")

	secret := v1.Secret{
		TypeMeta: v1types.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: v1types.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Type: v1.SecretTypeBasicAuth,
		Data: map[string][]byte{
			v1.BasicAuthUsernameKey: []byte(username),
			v1.BasicAuthPasswordKey: fakePassword,
		},
	}

	return &secret, nil
}

func NewLicenseSecret(name string, namespace string, content string) (*v1.Secret, error) {
	secret := v1.Secret{
		TypeMeta: v1types.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: v1types.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Type: v1.SecretTypeOpaque,
		Data: map[string][]byte{
			SecretLicenseContentKey: []byte(content),
		},
	}

	return &secret, nil
}

func NewCertificateSecret(name string, namespace string) (*v1.Secret, error) {
	// It is not possible to reconstruct the certificate info from a running
	// system so scaffold it and allow the user to fill in the blanks.
	fakeInput := []byte("")

	secret := v1.Secret{
		TypeMeta: v1types.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: v1types.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Type: v1.SecretTypeTLS,
		Data: map[string][]byte{
			v1.TLSCertKey:              fakeInput,
			v1.TLSPrivateKeyKey:        fakeInput,
			v1.ServiceAccountRootCAKey: fakeInput,
		},
	}

	return &secret, nil
}

func NewHostSpec(hostInfo v1info.HostInfo) (*HostSpec, error) {
	spec := HostSpec{}

	// Fill-in the basic attributes
	hostname := hostInfo.Hostname
	if hostname == "" {
		hostname = hostInfo.ID
	}

	spec.Profile = hostname
	spec.Overrides = &HostProfileSpec{
		ProfileBaseAttributes: ProfileBaseAttributes{
			// Assume that hosts will all be statically provisioned for now.
			BootMAC: &hostInfo.BootMAC},
	}

	return &spec, nil
}

func NewHost(name string, namespace string, hostInfo v1info.HostInfo) (*Host, error) {
	host := Host{
		TypeMeta: v1types.TypeMeta{
			APIVersion: APIVersion,
			Kind:       KindHost,
		},
		ObjectMeta: v1types.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				ControllerToolsLabel: ControllerToolsVersion,
			},
		},
	}

	spec, err := NewHostSpec(hostInfo)
	if err != nil {
		return nil, err
	}

	spec.DeepCopyInto(&host.Spec)

	return &host, nil
}

func NewDataNetworkSpec(net datanetworks.DataNetwork) (*DataNetworkSpec, error) {
	spec := DataNetworkSpec{
		Type: net.Type,
	}

	if net.MTU != datanetworks.DefaultMTU {
		spec.MTU = &net.MTU
	}

	if net.Description != "" {
		spec.Description = &net.Description
	}

	if net.Type == datanetworks.TypeVxLAN {
		spec.VxLAN = &VxLANInfo{
			EndpointMode:  net.Mode,
			UDPPortNumber: net.UDPPortNumber,
			TTL:           net.TTL,
		}

		if net.Mode != nil && *net.Mode == datanetworks.EndpointModeDynamic {
			spec.VxLAN.MulticastGroup = net.MulticastGroup
		}
	}

	return &spec, nil
}

func NewDataNetwork(name string, namespace string, net datanetworks.DataNetwork) (*DataNetwork, error) {
	dataNetwork := DataNetwork{
		TypeMeta: v1types.TypeMeta{
			APIVersion: APIVersion,
			Kind:       KindDataNetwork,
		},
		ObjectMeta: v1types.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				ControllerToolsLabel: ControllerToolsVersion,
			},
		},
	}

	spec, err := NewDataNetworkSpec(net)
	if err != nil {
		return nil, err
	}

	spec.DeepCopyInto(&dataNetwork.Spec)

	return &dataNetwork, nil
}

func NewPtpInstanceSpec(inst ptpinstances.PTPInstance) (*PtpInstanceSpec, error) {
	spec := PtpInstanceSpec{
		Service: inst.Service,
	}

	if inst.Parameters != nil && len(inst.Parameters) > 0 {
		spec.InstanceParameters = inst.Parameters
	} else {
		spec.InstanceParameters = nil
	}

	return &spec, nil
}

func NewPTPInstance(name string, namespace string, inst ptpinstances.PTPInstance) (*PtpInstance, error) {
	ptpInstance := PtpInstance{
		TypeMeta: v1types.TypeMeta{
			APIVersion: APIVersion,
			Kind:       KindPTPInstance,
		},
		ObjectMeta: v1types.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				ControllerToolsLabel: ControllerToolsVersion,
			},
		},
	}

	spec, err := NewPtpInstanceSpec(inst)
	if err != nil {
		return nil, err
	}

	spec.DeepCopyInto(&ptpInstance.Spec)

	return &ptpInstance, nil
}

func NewPtpInterfaceSpec(PTPint ptpinterfaces.PTPInterface) (*PtpInterfaceSpec, error) {
	spec := PtpInterfaceSpec{
		PtpInstance: PTPint.PTPInstanceName,
	}

	if PTPint.Parameters != nil && len(PTPint.Parameters) > 0 {
		spec.InterfaceParameters = PTPint.Parameters
	} else {
		spec.InterfaceParameters = nil
	}

	return &spec, nil
}

func NewPTPInterface(name string, namespace string, PTPint ptpinterfaces.PTPInterface) (*PtpInterface, error) {
	ptpInterface := PtpInterface{
		TypeMeta: v1types.TypeMeta{
			APIVersion: APIVersion,
			Kind:       KindPTPInterface,
		},
		ObjectMeta: v1types.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				ControllerToolsLabel: ControllerToolsVersion,
			},
		},
	}

	spec, err := NewPtpInterfaceSpec(PTPint)
	if err != nil {
		return nil, err
	}

	spec.DeepCopyInto(&ptpInterface.Spec)

	return &ptpInterface, nil
}

func NewPlatformNetworkSpec(pool addresspools.AddressPool, network_type string) (*PlatformNetworkSpec, error) {
	spec := PlatformNetworkSpec{
		Type:    network_type,
		Subnet:  pool.Network,
		Prefix:  pool.Prefix,
		Gateway: pool.Gateway,
		Allocation: AllocationInfo{
			Type:  networks.AllocationOrderDynamic,
			Order: &pool.Order,
		},
	}

	ranges := make([]AllocationRange, 0)
	for _, r := range pool.Ranges {
		obj := AllocationRange{
			Start: r[0],
			End:   r[1],
		}
		ranges = append(ranges, obj)
	}

	spec.Allocation.Ranges = ranges

	return &spec, nil
}

func NewPlatformNetwork(name string, namespace string, pool addresspools.AddressPool, network_type string) (*PlatformNetwork, error) {
	platformNetwork := PlatformNetwork{
		TypeMeta: v1types.TypeMeta{
			APIVersion: APIVersion,
			Kind:       KindPlatformNetwork,
		},
		ObjectMeta: v1types.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				ControllerToolsLabel: ControllerToolsVersion,
			},
		},
	}

	spec, err := NewPlatformNetworkSpec(pool, network_type)
	if err != nil {
		return nil, err
	}

	spec.DeepCopyInto(&platformNetwork.Spec)

	return &platformNetwork, nil
}
