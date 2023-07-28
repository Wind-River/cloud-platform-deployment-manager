/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019-2022 Wind River Systems, Inc. */

package platform

import (
	"strconv"
	"strings"

	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/hostFilesystems"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/licenses"
	"github.com/pkg/errors"
	utils "github.com/wind-river/cloud-platform-deployment-manager/common"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/addresses"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/addresspools"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/cephmonitors"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/certificates"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/clusters"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/controllerFilesystems"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/cpus"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/datanetworks"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/disks"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/dns"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/drbd"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/hosts"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/interfaceDataNetworks"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/interfaceNetworks"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/interfaces"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/labels"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/memory"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/networks"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/ntp"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/osds"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/partitions"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/physicalvolumes"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/ports"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/ptp"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/ptpinstances"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/ptpinterfaces"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/routes"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/serviceparameters"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/storagebackends"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/storagetiers"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/system"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/volumegroups"
)

// HostInfo defines the system resources that are collected thru the system API.
// Since various methods that deal with specific resources often needs related
// information from other resource those pieces of information are being
// aggregated into a single type to facilitate passing the data around and
// minimizing the number of API calls required.
type HostInfo struct {
	hosts.Host
	Labels                []labels.Label
	CPU                   []cpus.CPU
	Memory                []memory.Memory
	Monitors              []cephmonitors.CephMonitor
	Networks              []networks.Network
	DataNetworks          []datanetworks.DataNetwork
	InterfaceNetworks     []interfaceNetworks.InterfaceNetwork
	InterfaceDataNetworks []interfaceDataNetworks.InterfaceDataNetwork
	Pools                 []addresspools.AddressPool
	Ports                 []ports.Port
	Interfaces            []interfaces.Interface
	Addresses             []addresses.Address
	Routes                []routes.Route
	Disks                 []disks.Disk
	Partitions            []partitions.DiskPartition
	VolumeGroups          []volumegroups.VolumeGroup
	PhysicalVolumes       []physicalvolumes.PhysicalVolume
	OSDs                  []osds.OSD
	Clusters              []clusters.Cluster
	StorageTiers          map[string]*storagetiers.StorageTier
	FileSystems           []hostFilesystems.FileSystem
	PTPInstances          []ptpinstances.PTPInstance
	PTPInterfaces         []ptpinterfaces.PTPInterface
}

type SystemInfo struct {
	system.System
	DRBD              *drbd.DRBD
	DNS               *dns.DNS
	NTP               *ntp.NTP
	PTP               *ptp.PTP
	Certificates      []certificates.Certificate
	ServiceParameters []serviceparameters.ServiceParameter
	StorageBackends   []storagebackends.StorageBackend
	FileSystems       []controllerFilesystems.FileSystem
	License           *licenses.License
}

func (in *SystemInfo) PopulateSystemInfo(client *gophercloud.ServiceClient) error {
	var err error

	result, err := system.GetDefaultSystem(client)
	if err != nil {
		err = errors.Wrapf(err, "failed to get system")
		return err
	}
	in.System = *result

	in.DRBD, err = drbd.GetDefaultDRBD(client)
	if err != nil {
		err = errors.Wrap(err, "failed to get DRBD info")
		return err
	}

	in.DNS, err = dns.GetSystemDNS(client, result.ID)
	if err != nil {
		err = errors.Wrap(err, "failed to get DNS info")
		return err
	}

	in.NTP, err = ntp.GetSystemNTP(client, result.ID)
	if err != nil {
		err = errors.Wrap(err, "failed to get NTP info")
		return err
	}

	in.PTP, err = ptp.GetSystemPTP(client, result.ID)
	if err != nil {
		err = errors.Wrap(err, "failed to get PTP info")
		return err
	}

	// TODO(alegacy): The system API does not provide a differentiation of
	//  of certificates by system id therefore we take the entire list.
	in.Certificates, err = certificates.ListCertificates(client)
	if err != nil {
		err = errors.Wrap(err, "failed to get certificate list")
		return err
	}

	in.ServiceParameters, err = serviceparameters.ListServiceParameters(client)
	if err != nil {
		err = errors.Wrap(err, "failed to get service parameters")
		return err
	}

	in.StorageBackends, err = storagebackends.ListBackends(client)
	if err != nil {
		err = errors.Wrap(err, "failed to get storagebackends")
		return err
	}

	in.FileSystems, err = controllerFilesystems.ListFileSystems(client)
	if err != nil {
		err = errors.Wrap(err, "failed to get filesystem list")
		return err
	}

	in.License, err = licenses.Get(client).Extract()
	if err != nil {
		if !strings.Contains(err.Error(), "License file not found") {
			err = errors.Wrap(err, "failed to get license list")
			return err
		}
	}

	return nil
}

// getSystemPartitions augments the list of partitions that were retrieved using
// the List API with additional partitions that were created by the system
// and are not visible from the List API.
func (in *HostInfo) PopulateSystemPartitions(client *gophercloud.ServiceClient) error {
	for _, pv := range in.PhysicalVolumes {
		if pv.Type != physicalvolumes.PVTypePartition {
			continue
		}

		if _, ok := in.FindPartition(pv.DeviceUUID); !ok {
			// System partitions are not reported by the List API so they need
			// to be retrieved individually.
			partition, err := partitions.Get(client, pv.DeviceUUID).Extract()
			if err != nil {
				err = errors.Wrapf(err, "failed to lookup system partition: %s", pv.DeviceUUID)
				return err
			}

			in.Partitions = append(in.Partitions, *partition)
		}
	}

	return nil
}

// getSystemPartitions augments the list of partitions that were retrieved using
// the List API with additional partitions that were created by the system
// and are not visible from the List API.
func (in *HostInfo) PopulateStorageTiers(client *gophercloud.ServiceClient) error {
	tiersByCluster := make(map[string]*storagetiers.StorageTier)
	results, err := clusters.ListClusters(client)
	if err != nil {
		err = errors.Wrap(err, "failed to list system storage clusters")
		return err
	}

	for _, c := range results {
		tiers, err := storagetiers.ListTiers(client, c.ID)
		if err != nil {
			err = errors.Wrapf(err, "failed to list storage tiers for cluster: %s", c.ID)
			return err
		}

		for _, t := range tiers {
			if t.Name == storagetiers.StorageTierName {
				tiersByCluster[c.Name] = &t
			}
		}
	}

	if len(tiersByCluster) > 0 {
		in.StorageTiers = tiersByCluster
	} else {
		in.StorageTiers = nil
	}

	return nil
}

// getHostInfo is a utility function which build all host attributes and
// stores them into a single structure that acts as a cache of data that can
// be passed around and re-used rather than having to re-read data that is
// required in multiple functions.
func (in *HostInfo) PopulateHostInfo(client *gophercloud.ServiceClient, hostid string) error {
	var err error

	result, err := hosts.Get(client, hostid).Extract()
	if err != nil {
		err = errors.Wrapf(err, "failed to get host %s", hostid)
		return err
	}
	in.Host = *result

	in.Labels, err = labels.ListLabels(client, hostid)
	if err != nil {
		err = errors.Wrapf(err, "failed to list labels for host %s", hostid)
		return err
	}

	in.CPU, err = cpus.ListCPUs(client, hostid)
	if err != nil {
		err = errors.Wrapf(err, "failed to list CPU for host %s", hostid)
		return err
	}

	in.Memory, err = memory.ListMemory(client, hostid)
	if err != nil {
		err = errors.Wrapf(err, "failed to list memory for host %s", hostid)
		return err
	}

	in.Monitors, err = cephmonitors.ListCephMonitors(client)
	if err != nil {
		err = errors.Wrapf(err, "failed to list Ceph monitors for host %s", hostid)
		return err
	}

	in.Networks, err = networks.ListNetworks(client)
	if err != nil {
		err = errors.Wrapf(err, "failed to list networks for host %s", hostid)
		return err
	}

	in.DataNetworks, err = datanetworks.ListDataNetworks(client)
	if err != nil {
		err = errors.Wrapf(err, "failed to list data networks for host %s", hostid)
		return err
	}

	in.InterfaceNetworks, err = interfaceNetworks.ListInterfaceNetworks(client, hostid)
	if err != nil {
		err = errors.Wrapf(err, "failed to list interface networks for host %s", hostid)
		return err
	}

	in.InterfaceDataNetworks, err = interfaceDataNetworks.ListInterfaceDataNetworks(client, hostid)
	if err != nil {
		err = errors.Wrapf(err, "failed to list interface data networks for host %s", hostid)
		return err
	}

	in.Pools, err = addresspools.ListAddressPools(client)
	if err != nil {
		err = errors.Wrapf(err, "failed to list address pools for host %s", hostid)
		return err
	}

	in.Ports, err = ports.ListPorts(client, hostid)
	if err != nil {
		err = errors.Wrapf(err, "failed to list ports for host %s", hostid)
		return err
	}

	in.Interfaces, err = interfaces.ListInterfaces(client, hostid)
	if err != nil {
		err = errors.Wrapf(err, "failed to list interfaces for host %s", hostid)
		return err
	}

	in.Addresses, err = addresses.ListAddresses(client, hostid)
	if err != nil {
		err = errors.Wrapf(err, "failed to list addresses for host %s", hostid)
		return err
	}

	in.Routes, err = routes.ListRoutes(client, hostid)
	if err != nil {
		err = errors.Wrapf(err, "failed to list routes for host %s", hostid)
		return err
	}

	in.Disks, err = disks.ListDisks(client, hostid)
	if err != nil {
		err = errors.Wrapf(err, "failed to list disks for host %s", hostid)
		return err
	}

	in.Partitions, err = partitions.ListPartitions(client, hostid)
	if err != nil {
		err = errors.Wrapf(err, "failed to list partitions for host %s", hostid)
		return err
	}

	in.VolumeGroups, err = volumegroups.ListVolumeGroups(client, hostid)
	if err != nil {
		err = errors.Wrapf(err, "failed to list volume groups for host %s", hostid)
		return err
	}

	in.PhysicalVolumes, err = physicalvolumes.ListPhysicalVolumes(client, hostid)
	if err != nil {
		err = errors.Wrapf(err, "failed to list physical volumes for host %s", hostid)
		return err
	}

	in.OSDs, err = osds.ListOSDs(client, hostid)
	if err != nil {
		err = errors.Wrapf(err, "failed to list OSDs for host %s", hostid)
		return err
	}

	in.Clusters, err = clusters.ListClusters(client)
	if err != nil {
		err = errors.Wrapf(err, "failed to list clusters for host %s", hostid)
		return err
	}

	in.PTPInstances, err = ptpinstances.ListHostPTPInstances(client, hostid)
	if err != nil {
		err = errors.Wrapf(err, "failed to list PTP instances for host %s", hostid)
		return err
	}

	in.PTPInterfaces, err = ptpinterfaces.ListHostPTPInterfaces(client, hostid)
	if err != nil {
		err = errors.Wrapf(err, "failed to list PTP interfaces for host %s", hostid)
		return err
	}

	// TODO(alegacy):  the system API needs to be changed to either show all
	//  system created resources or to not show them at all.
	err = in.PopulateSystemPartitions(client)
	if err != nil {
		return err
	}

	// Refresh the list of storage tiers since they will be needed when adding
	// the OSDs.
	err = in.PopulateStorageTiers(client)
	if err != nil {
		return err
	}

	in.FileSystems, err = hostFilesystems.ListFileSystems(client, hostid)
	if err != nil {
		err = errors.Wrapf(err, "failed to list filesystems for host %s", hostid)
		return err
	}

	return nil
}

// findPortInterfaceUUID is a utility function which accepts a port name and
// attempts to find the interface UUID which represents the interface associated
// to the port.
func (in *HostInfo) FindPortInterfaceUUID(portname string) (string, bool) {
	// TODO(alegacy): consider storing ports as a map indexed by name to
	//  eliminate excessive looping
	for _, p := range in.Ports {
		if p.Name == portname {
			return p.InterfaceID, true
		}
	}
	return "", false
}

// findInterface is a utility function which accepts a host interface UUID value
// and returns a reference to a host interface from the system API.
func (in *HostInfo) FindInterface(ifuuid string) (*interfaces.Interface, bool) {
	for _, i := range in.Interfaces {
		if i.ID == ifuuid {
			return &i, true
		}
	}
	return nil, false
}

// findInterfaceByName is a utility function which finds a system interface
// object by its unique name.
func (in *HostInfo) FindInterfaceByName(name string) (*interfaces.Interface, bool) {
	for _, i := range in.Interfaces {
		if i.Name == name {
			return &i, true
		}
	}
	return nil, false
}

// findVLANInterfaceUUID is a utility function to find a VLAN interface in the
// list of interfaces returned by the systemAPI.
func (in *HostInfo) FindVLANInterfaceUUID(vid int) (string, bool) {
	for _, i := range in.Interfaces {
		if i.Type != interfaces.IFTypeVLAN {
			continue
		}

		if i.VID != nil && *i.VID == vid {
			return i.ID, true
		}
	}
	return "", false
}

// findVFInterfaceUUID is a utility function to find a VF interface in the
// list of interfaces returned by the systemAPI.
func (in *HostInfo) FindVFInterfaceUUID(name string) (string, bool) {
	for _, i := range in.Interfaces {
		if i.Type != interfaces.IFTypeVF {
			continue
		}
		if i.Name == name {
			return i.ID, true
		}
	}
	return "", false
}

// findBondInterfaceUUID is a utility function to find a bond interface in the
// list of interfaces returned by the systemAPI. Because a user may rename a
// bond we try to identify it by its members rather than its name.  This may not
// handle all cases so may need to be revisited later.
func (in *HostInfo) FindBondInterfaceUUID(members []string) (string, bool) {
	for _, i := range in.Interfaces {
		if i.Type != interfaces.IFTypeAE {
			continue
		}

		if _, ok := utils.ListIntersect(members, i.Uses); ok {
			return i.ID, true
		}
	}
	return "", false
}

// findInterfacePortName is a utility function which searches the list of
// ports and returns the name of the port that is associated to the interface
// ID value specified.
func (in *HostInfo) FindInterfacePortName(id string) (string, bool) {
	for _, p := range in.Ports {
		if p.InterfaceID == id {
			return p.Name, true
		}
	}
	return "", false
}

// findAddressUUID is a utility function which finds a system address object
// by its unique attributes.
func (in *HostInfo) FindAddressUUID(ifname string, address string, prefix int) (*addresses.Address, bool) {
	for _, addr := range in.Addresses {
		if addr.InterfaceName == ifname {
			if strings.EqualFold(addr.Address, address) {
				if addr.Prefix == prefix {
					return &addr, true
				}
			}
		}
	}
	return nil, false
}

// findRouteUUID is a utility function which finds a system route object
// by its unique attributes.
func (in *HostInfo) FindRouteUUID(ifname string, network string, prefix int) (*routes.Route, bool) {
	for _, r := range in.Routes {
		if r.InterfaceName == ifname {
			if strings.EqualFold(r.Network, network) {
				if r.Prefix == prefix {
					return &r, true
				}
			}
		}
	}
	return nil, false
}

// findVolumeGroup is a utility function that attempts to find a system volume
// group by name.
func (in *HostInfo) FindVolumeGroup(name string) (*volumegroups.VolumeGroup, bool) {
	for _, vg := range in.VolumeGroups {
		if vg.Name == name {
			return &vg, true
		}
	}

	return nil, false
}

// findPartition is a utility function that attempts to find a system partition
// by its unique uuid value.
func (in *HostInfo) FindPartition(uuid string) (*partitions.DiskPartition, bool) {
	for _, p := range in.Partitions {
		if p.ID == uuid {
			return &p, true
		}
	}

	return nil, false
}

// FindOSDByPath is a utility function that attempts to find an OSD by
// its absolute path.
func (in *HostInfo) FindOSDByPath(path string) (*osds.OSD, bool) {
	for _, o := range in.OSDs {
		if disk, ok := in.FindDiskByPath(path); ok {
			if disk.ID == o.DiskID {
				return &o, true
			}
		}
	}

	return nil, false
}

// FindClusterNameByTier does a reverse lookup in the cluster tier map and
// returns the name of the cluster to which the tier id is associated.
func (in *HostInfo) FindClusterNameByTier(id string) (string, bool) {
	for k, v := range in.StorageTiers {
		if v.ID == id {
			return k, true
		}
	}
	return "", false
}

func (in *HostInfo) FindClusterByName(name string) *clusters.Cluster {
	for _, c := range in.Clusters {
		if c.Name == name {
			return &c
		}
	}
	return nil
}

// FindDisk is a utility function that attempts to find a system disk by
// its unique uuid value.
func (in *HostInfo) FindDisk(id string) (*disks.Disk, bool) {
	for _, d := range in.Disks {
		if d.ID == id {
			return &d, true
		}
	}

	return nil, false
}

// FindDiskByPath is a utility function that attempts to find a system disk by
// its absolute device path.
func (in *HostInfo) FindDiskByPath(path string) (*disks.Disk, bool) {
	for _, d := range in.Disks {
		if utils.ComparePartitionPaths(d.DevicePath, path) {
			// Allow the match to succeed even if a partition path was used
			// rather than a disk path.
			return &d, true
		}
	}

	return nil, false
}

// FindDiskByNode is a utility function that attempts to find a system disk by
// its device node.
func (in *HostInfo) FindDiskByNode(path string) (*disks.Disk, bool) {
	for _, d := range in.Disks {
		if d.DeviceNode == path {
			return &d, true
		}
	}

	return nil, false
}

// FindPartitionByPath is a utility function that attempts to find a disk
// partition by its absolute device path.  Size is expected in Gibibytes.
func (in *HostInfo) FindPartitionByPath(path string, size int, physicalVolumeName string) (*partitions.DiskPartition, bool) {
	volume, _ := in.FindPhysicalVolume(physicalVolumeName, physicalvolumes.PVTypePartition, path, size)
	if volume != nil {
		// First look for a partition that is associated to the named volume
		// group.  It is, at least theoretically, possible for two partitions
		// of the same size to be on the same disk, but be used for different
		// volume groups so we need to differentiate between different groups
		// when searching.
		for _, p := range in.Partitions {
			if p.PhysicalVolumeID != nil && *p.PhysicalVolumeID == volume.ID {
				if utils.ComparePartitionPaths(path, p.DevicePath) {
					if size == p.Gibibytes() {
						return &p, true
					}
				}
			}
		}
	}

	// Otherwise, look for a partition that is not yet associated to any
	// volume group but matches the path and size specified.
	for _, p := range in.Partitions {
		if p.PhysicalVolumeID != nil {
			continue
		}
		if utils.ComparePartitionPaths(path, p.DevicePath) {
			if size == p.Gibibytes() {
				return &p, true
			}
		}
	}

	return nil, false
}

// FindPhysicalVolume is a utility function that attempts to find a system
// partition matching the criteria specified.  Size is expected in Gibibytes
func (in *HostInfo) FindPhysicalVolume(groupName string, typ string, path string, size int) (*physicalvolumes.PhysicalVolume, bool) {
	for _, v := range in.PhysicalVolumes {
		if v.VolumeGroupName != groupName {
			continue
		}

		if v.Type != typ {
			continue
		}

		if v.Type == physicalvolumes.PVTypeDisk {
			if v.DevicePath == path {
				return &v, true
			}

		} else if utils.ComparePartitionPaths(v.DevicePath, path) {
			// Because the user cannot guess as to what the partition number
			// will be ahead of time the path may be specified as a disk path
			// and so we use the disk path, rather than the full partition path,
			// to match any partitions on the same disk.

			// There is a discrepancy between how the size is reported on the
			// physical volume and the underlying partition.  It is not
			// guaranteed that they are exactly the same.  Not sure if that is
			// a bug in system inventory or if it is related to overhead
			// accounting but we need to ensure that we use the same value as
			// in parsePartitionInfo otherwise we may think that we need to
			// create a new partition.
			if p, ok := in.FindPartition(v.DeviceUUID); ok {
				if p.Gibibytes() == size {
					return &v, true
				}
			}
		}
	}

	return nil, false
}

// FindMemory is a utility function which finds a memory resource for a given
// processor node.
func (in *HostInfo) FindMemory(node int) (*memory.Memory, bool) {
	for _, m := range in.Memory {
		if m.Processor == node {
			return &m, true
		}
	}
	return nil, false
}

// BuildNetworkIDList is a utility function which takes a set of network names
// and produces a list of network id values.  Some networks as specified in the
// configuration yaml are only represented as address pools in the system so
// they are ignored here.
func (in *HostInfo) BuildNetworkIDList(nets []string) []string {
	result := make([]string, 0)
	for _, n := range nets {
		// TODO(alegacy): consider storing networks as a map indexed by name to
		//  eliminate excessive looping.
		for _, x := range in.Networks {
			if x.Name == n {
				result = append(result, strconv.Itoa(x.ID))
			}
		}
	}
	return result
}

// FindNetworkID is a utility method to find the network ID value for a given
// network name.
func (in *HostInfo) FindNetworkID(name string) (string, bool) {
	for _, net := range in.Networks {
		if net.Name == name {
			return net.UUID, true
		}
	}

	return "", false
}

// FindDataNetworkID is a utility method to find the data network ID value for
// a given data network name.
func (in *HostInfo) FindDataNetworkID(name string) (string, bool) {
	for _, net := range in.DataNetworks {
		if net.Name == name {
			return net.ID, true
		}
	}

	return "", false
}

// FindInterfaceNetworkID is a utility method to find the interface-network ID
// value for a given interface and network name.
func (in *HostInfo) FindInterfaceNetworkID(iface interfaces.Interface, network string) (string, bool) {
	for _, association := range in.InterfaceNetworks {
		if association.NetworkName == network {
			return association.UUID, true
		}
	}

	return "", false
}

// FindInterfaceDataNetworkID is a utility method to find the
// interface-datanetwork ID value for a given interface and data network name.
func (in *HostInfo) FindInterfaceDataNetworkID(iface interfaces.Interface, datanetwork string) (string, bool) {
	for _, association := range in.InterfaceDataNetworks {
		if association.DataNetworkName == datanetwork {
			return association.UUID, true
		}
	}

	return "", false
}

// BuildInterfaceNetworkList is a utility function that takes a builds a list
// of network names based on the specific interface ID and the host's list of
// interface-to-network associations.
func (in *HostInfo) BuildInterfaceNetworkList(iface interfaces.Interface) []string {
	result := make([]string, 0)

	for _, association := range in.InterfaceNetworks {
		if association.InterfaceUUID == iface.ID {
			result = append(result, association.NetworkName)
		}
	}

	return result
}

// BuildInterfaceDataNetworkList is a utility function takes a builds a list
// // of network names based on the specific interface ID and the host's list of
// // interface-to-datanetwork associations.
func (in *HostInfo) BuildInterfaceDataNetworkList(iface interfaces.Interface) []string {
	result := make([]string, 0)

	for _, association := range in.InterfaceDataNetworks {
		if association.InterfaceUUID == iface.ID {
			result = append(result, association.DataNetworkName)
		}
	}

	return result
}

// FindPTPInterfaceNameByInterface is a utility function to search for the name
// of a PTP interface by the interface info.
func (in *HostInfo) FindPTPInterfaceNameByInterface(iface interfaces.Interface) []string {
	result := make([]string, 0)

	// Interface name is formatted as "hostname/ifname" in PTPinterfaces,
	// eg. "controller-0/data0"
	interfaceStr := in.Host.Hostname + "/" + iface.Name

	if len(in.PTPInterfaces) > 0 {
		for _, singlePTPInterface := range in.PTPInterfaces {
			for _, singleInterface := range singlePTPInterface.InterfaceNames {
				if interfaceStr == singleInterface {
					// Note: we currently only allow one PTP interface to be
					// assigned to an interface, so this result will be
					// assigned once.
					result = append(result, singlePTPInterface.Name)
				}
			}
		}
	}

	return result
}

// BuildPTPInstanceList is a utility function to iterate through
// all PTP instances associated with the host and return a list
// of PTP instance names.
func (in *HostInfo) BuildPTPInstanceList() []string {
	result := make([]string, 0)

	for _, ptpInstance := range in.PTPInstances {
		result = append(result, ptpInstance.Name)
	}

	return result
}

// FindLabel is a utility function which searchs the current list of host
// labels and finds the first entry that matches the specified key.
func (in *HostInfo) FindLabel(key string) (string, bool) {
	for _, l := range in.Labels {
		if l.Key == key {
			return l.Value, true
		}
	}

	return "", false
}

// CountCPUByFunction examines the list of CPU instance returned by the system
// API and counts the number of cores that match the node and function specified
// by the caller
func (in *HostInfo) CountCPUByFunction(node int, function string) int {
	count := 0
	for _, c := range in.CPU {
		if c.Thread != 0 {
			// Processor configurations are always done on a physical core
			// basis so do not include hyper-thread cores.
			continue
		}

		// Only consider thread=0 because we do not consider hyper-thread
		// cores as part of cpu allocations.
		if strings.EqualFold(c.Function, function) && c.Processor == node {
			count++
		}
	}

	return count
}

// FindAddressPoolByName is a utility function which examines the list of system
// address pools and returns a reference to a pool which matches the network
// name provided.
func (in *HostInfo) FindAddressPoolByName(name string) *addresspools.AddressPool {
	for _, p := range in.Pools {
		if p.Name == name {
			return &p
		}
	}

	return nil
}

// FindAddressPoolByName is a utility function which examines the list of system
// address pools and returns a reference to a pool which matches the network
// name provided.
func (in *HostInfo) FindAddressPool(id string) *addresspools.AddressPool {
	for _, p := range in.Pools {
		if p.ID == id {
			return &p
		}
	}

	return nil
}

// IsSystemAddress determines if an address was added to the system
// automatically or whether it was added manually.  The determination is based
// on whether the address is associated to an address pool.
func (in *HostInfo) IsSystemAddress(address *addresses.Address) bool {
	return address.PoolUUID != nil
}

// IsStorageDeploymentModel determines where storage nodes are expected to be
// deployed.
func (in *HostInfo) IsStorageDeploymentModel() bool {
	for _, x := range in.Clusters {
		if x.DeploymentModel == clusters.DeploymentModelStorage {
			return true
		}
	}

	return false
}
