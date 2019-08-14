/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019 Wind River Systems, Inc. */

package host

import (
	"fmt"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/cephmonitors"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/clusters"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/hostFilesystems"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/hosts"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/osds"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/partitions"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/physicalvolumes"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/storagetiers"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/volumegroups"
	perrors "github.com/pkg/errors"
	starlingxv1beta1 "github.com/wind-river/titanium-deployment-manager/pkg/apis/starlingx/v1beta1"
	"github.com/wind-river/titanium-deployment-manager/pkg/config"
	"github.com/wind-river/titanium-deployment-manager/pkg/controller/common"
	titaniumManager "github.com/wind-river/titanium-deployment-manager/pkg/manager"
	v1info "github.com/wind-river/titanium-deployment-manager/pkg/platform"
	"strings"
)

// ReconcileMonitor is responsible for reconciling the Ceph storage monitor
// configuration of a compute host resource.
func (r *ReconcileHost) ReconcileMonitor(client *gophercloud.ServiceClient, instance *starlingxv1beta1.Host, profile *starlingxv1beta1.HostProfileSpec, host *v1info.HostInfo) error {

	if !config.IsReconcilerEnabled(config.StorageMonitor) {
		return nil
	}

	if profile.Personality == nil || *profile.Personality != hosts.PersonalityWorker {
		// The monitors on the controllers are handled automatically.
		return nil
	}

	monitors, err := cephmonitors.ListCephMonitors(client)
	if err != nil {
		err = perrors.Wrap(err, "failed to list monitors on host")
		return err
	}

	if profile.Storage.Monitor == nil {
		// Delete any existing monitors
		for _, monitor := range monitors {
			if monitor.HostUUID == host.ID {
				// TODO(alegacy): The system API currently does not support deleting a
				//  monitor directly.  The entire node needs to be deleted and re-added.
				log.Info("stale Ceph monitor detected;  Deleting monitors is not supported")
			}
		}

	} else {
		storage := profile.Storage

		found := false
		for _, monitor := range monitors {
			if monitor.HostUUID == host.ID {
				found = true

				if storage.Monitor.Size != nil && *storage.Monitor.Size != monitor.Size {
					opts := cephmonitors.CephMonitorOpts{
						Size: storage.Monitor.Size,
					}

					log.Info("updating Ceph monitor", "opts", opts)

					_, err := cephmonitors.Update(client, host.ID, opts).Extract()
					if err != nil {
						err = perrors.Wrap(err, "failed to update Ceph monitor")
						return err
					}

					r.NormalEvent(instance, common.ResourceCreated,
						"ceph monitor has been updated")
				}
			}
		}

		if !found {
			// Add a new monitor for this host.
			opts := cephmonitors.CephMonitorOpts{
				HostUUID: &host.ID,
				Size:     profile.Storage.Monitor.Size,
			}

			log.Info("adding Ceph monitor", "opts", opts)

			_, err := cephmonitors.Create(client, opts).Extract()
			if err != nil {
				err = perrors.Wrap(err, "failed to create Ceph monitor")
				return err
			}

			r.NormalEvent(instance, common.ResourceCreated,
				"ceph monitor has been created")
		}
	}

	return nil
}

// ReconcileMonitor is responsible for reconciling the disk partitions
// configuration on a host.
func (r *ReconcileHost) ReconcilePartitions(client *gophercloud.ServiceClient, instance *starlingxv1beta1.Host, profile *starlingxv1beta1.HostProfileSpec, host *v1info.HostInfo, group starlingxv1beta1.VolumeGroupInfo) error {
	updated := false

	if !config.IsReconcilerEnabled(config.Partition) {
		return nil
	}

	for _, pvInfo := range group.PhysicalVolumes {
		if pvInfo.Type != physicalvolumes.PVTypePartition || pvInfo.Size == nil {
			// Ignore disks, and since validation ensures that partition sizes
			// are not nil ignore those as well.
			continue
		}

		size := 0
		if pvInfo.Size != nil {
			size = *pvInfo.Size
		}

		if _, ok := host.FindPartitionByPath(pvInfo.Path, size, group.Name); ok {
			// A matching partition already exists.
			continue
		}

		// Lookup the disk and use its ID to create the partition
		disk, ok := host.FindDiskByPath(pvInfo.Path)
		if !ok {
			msg := fmt.Sprintf("failed to find disk for path %s", pvInfo.Path)
			return starlingxv1beta1.NewMissingSystemResource(msg)
		}

		// Create a new partition for this physical volume.
		typeName := partitions.PartitionTypeLVM
		typeGUID := partitions.PartitionTypeMap[typeName]
		opts := partitions.DiskPartitionOpts{
			HostID:   host.ID,
			DiskID:   disk.ID,
			TypeName: &typeName,
			TypeGUID: &typeGUID,
		}

		if pvInfo.Size != nil {
			opts.Size = *pvInfo.Size
		}

		log.Info("creating partition", "opts", opts)

		partition, err := partitions.Create(client, opts).Extract()
		if err != nil {
			err = perrors.Wrapf(err, "failed to create new partition: %s",
				common.FormatStruct(opts))
			return err
		}

		r.NormalEvent(instance, common.ResourceCreated,
			"partition %q has been created", partition.DevicePath)

		updated = true
	}

	if updated {
		result, err := partitions.ListPartitions(client, host.ID)
		if err != nil {
			err = perrors.Wrap(err, "failed to refresh partitions on host")
			return err
		}

		host.Partitions = result

		// TODO(alegacy):  the system API needs to be changed to either show all
		//  system created resources or to not show them at all.
		//  See: https://bugs.launchpad.net/bugs/1823739
		err = host.PopulateSystemPartitions(client)
		if err != nil {
			return err
		}
	}

	for _, p := range host.Partitions {
		switch p.Status {
		case partitions.StatusDeleting, partitions.StatusModifying, partitions.StatusCreating:
			m := NewPartitionStateMonitor(instance, host.ID)
			msg := "waiting for partitions to transition to ready state"
			return r.StartMonitor(m, msg)
		}
	}

	return nil
}

// ReconcilePhysicalVolumes is responsible for reconciling the physical volume
// configuration on a host.
func (r *ReconcileHost) ReconcilePhysicalVolumes(client *gophercloud.ServiceClient, instance *starlingxv1beta1.Host, profile *starlingxv1beta1.HostProfileSpec, host *v1info.HostInfo, group starlingxv1beta1.VolumeGroupInfo) error {
	if !config.IsReconcilerEnabled(config.PhysicalVolume) {
		return nil
	}

	vg, ok := host.FindVolumeGroup(group.Name)
	if !ok {
		// The LVG was created by the caller so this should never happen.
		msg := fmt.Sprintf("unable to find volume group %s", group.Name)
		return starlingxv1beta1.NewMissingSystemResource(msg)
	}

	// Make sure that all required partitions exist.
	err := r.ReconcilePartitions(client, instance, profile, host, group)
	if err != nil {
		return err
	}

	updated := false

	for _, pvInfo := range group.PhysicalVolumes {
		var deviceID string

		size := 0
		if pvInfo.Size != nil {
			size = *pvInfo.Size
		}

		if _, ok := host.FindPhysicalVolume(group.Name, pvInfo.Type, pvInfo.Path, size); ok {
			// Already exists.  No work required.
			continue
		}

		// Otherwise, we need to create a new one but first we need to find the
		// device to which it will be associated.
		if pvInfo.Type == physicalvolumes.PVTypePartition {
			if partition, ok := host.FindPartitionByPath(pvInfo.Path, size, group.Name); ok {
				deviceID = partition.ID
			}
		} else {
			if disk, ok := host.FindDiskByPath(pvInfo.Path); ok {
				deviceID = disk.ID
			}
		}

		if deviceID == "" {
			msg := fmt.Sprintf("failed to find physical volume device: %s(%s)", pvInfo.Path, pvInfo.Type)
			return starlingxv1beta1.NewMissingSystemResource(msg)
		}

		// Create the new physical volume.
		opts := physicalvolumes.PhysicalVolumeOpts{
			HostID:        host.ID,
			DeviceID:      deviceID,
			VolumeGroupID: vg.ID,
			Type:          pvInfo.Type,
		}

		log.Info("creating physical volume", "opts", opts)

		_, err := physicalvolumes.Create(client, opts).Extract()
		if err != nil {
			err = perrors.Wrap(err, "failed to create physical volume")
			return err
		}

		r.NormalEvent(instance, common.ResourceCreated,
			"physical volume '%s(%s)' has been created", pvInfo.Path, pvInfo.Type)

		updated = true
	}

	if updated {
		result, err := physicalvolumes.ListPhysicalVolumes(client, host.ID)
		if err != nil {
			err = perrors.Wrap(err, "failed to refresh physical volume list")
			return err
		}

		host.PhysicalVolumes = result
	}

	return nil
}

// ReconcileVolumeGroups is responsible for reconciling the volume group
// configuration of a host resource.
func (r *ReconcileHost) ReconcileVolumeGroups(client *gophercloud.ServiceClient, instance *starlingxv1beta1.Host, profile *starlingxv1beta1.HostProfileSpec, host *v1info.HostInfo) error {
	updated := false

	if profile.Storage.VolumeGroups == nil {
		return nil
	}

	if !config.IsReconcilerEnabled(config.VolumeGroup) {
		return nil
	}

	for _, vgInfo := range *profile.Storage.VolumeGroups {
		var ok bool

		if _, ok = host.FindVolumeGroup(vgInfo.Name); !ok {
			// Create a new volume group.
			var capabilitiesPtr *volumegroups.CapabilitiesOpts
			opts := volumegroups.VolumeGroupOpts{
				HostID: &host.ID,
				Name:   &vgInfo.Name,
			}

			capabilities := volumegroups.CapabilitiesOpts{}

			if vgInfo.ConcurrentDiskOperations != nil {
				capabilities.ConcurrentDiskOperations = vgInfo.ConcurrentDiskOperations
				capabilitiesPtr = &capabilities
			}

			if vgInfo.LVMType != nil {
				capabilities.LVMType = vgInfo.LVMType
				capabilitiesPtr = &capabilities
			}

			opts.Capabilities = capabilitiesPtr

			log.Info("creating Volume Group", "opts", opts)

			_, err := volumegroups.Create(client, opts).Extract()
			if err != nil {
				err = perrors.Wrapf(err, "failed to create volume group, %s",
					common.FormatStruct(opts))
				return err
			}

			r.NormalEvent(instance, common.ResourceCreated,
				"volume Group %q has been created", vgInfo.Name)

			updated = true
		}
	}

	if updated {
		result, err := volumegroups.ListVolumeGroups(client, host.ID)
		if err != nil {
			err = perrors.Wrap(err, "failed to refresh volume groups")
			return err
		}

		host.VolumeGroups = result
	}

	for _, vgInfo := range *profile.Storage.VolumeGroups {
		// Reconcile the state of each physical volume on this group.
		err := r.ReconcilePhysicalVolumes(client, instance, profile, host, vgInfo)
		if err != nil {
			return err
		}
	}

	return nil
}

func osdUpdateRequired(osdInfo *starlingxv1beta1.OSDInfo, osd *osds.OSD) (opts osds.OSDOpts, result bool) {
	if osdInfo.Journal != nil {
		if osd.JournalInfo.Location == nil {
			// No journal existed previously, so add it now.
			size := osdInfo.Journal.Size
			opts.JournalLocation = &osdInfo.Journal.Location
			opts.JournalSize = &size
			result = true

		} else if osd.JournalInfo.Gibibytes() != osdInfo.Journal.Size {
			// The sizes do not match so update it.
			size := osdInfo.Journal.Size
			opts.JournalSize = &size
			result = true
		}
	}

	return opts, result
}

// ReconcileStaleOSDs is responsible for removing any OSD resources that are
// either no longer in the configured list or their function or journal has
// changed.
func (r *ReconcileHost) ReconcileStaleOSDs(client *gophercloud.ServiceClient, instance *starlingxv1beta1.Host, profile *starlingxv1beta1.HostProfileSpec, host *v1info.HostInfo) error {
	present := make(map[string]bool)
	updated := make(map[string]bool)

	if profile.Storage.OSDs == nil {
		return nil
	}

	if !config.IsReconcilerEnabled(config.OSD) {
		return nil
	}

	for _, osdInfo := range *profile.Storage.OSDs {
		if osd, ok := host.FindOSDByPath(osdInfo.Path); ok {
			present[osd.ID] = true

			if osd.Function != osdInfo.Function {
				// The system API does not support changing the function on
				// an OSD so delete it so that it can be re-added.
				updated[osd.ID] = true
			} else if osdInfo.Journal == nil && osd.JournalInfo.Location != nil {
				if *osd.JournalInfo.Location != osd.ID {
					// The system API does not support removing the journal so
					// delete this OSD so that it can be re-added.
					updated[osd.ID] = true
				}
			}
		}
	}

	changes := false
	for _, osd := range host.OSDs {
		// Delete stale OSDs
		if !present[osd.ID] || updated[osd.ID] {
			log.Info("deleting stale or updated OSD", "opts", osd)

			err := osds.Delete(client, osd.ID).ExtractErr()
			if err != nil {
				err = perrors.Wrapf(err, "failed to delete OSD: %s",
					common.FormatStruct(osd))
				return err
			}

			r.NormalEvent(instance, common.ResourceDeleted,
				"osd %q deleted", osd.ID)

			changes = true
		}
	}

	if changes {
		result, err := osds.ListOSDs(client, host.ID)
		if err != nil {
			err = perrors.Wrap(err, "failed to refresh OSD list for host")
			return err
		}

		host.OSDs = result
	}

	return nil
}

// OSDProvisioningState determines at what time the system permits OSD resources
// to be added to a host.
func (r *ReconcileHost) OSDProvisioningState(namespace string, personality string) RequiredState {
	switch r.GetSystemType(namespace) {
	case titaniumManager.SystemTypeAllInOne:
		// OSDs are allowed at any time on AIO systems.
		return RequiredStateAny
	case titaniumManager.SystemTypeStandard:
		if strings.EqualFold(personality, hosts.PersonalityStorage) {
			// OSDs are only allowed while locked/disabled for storage nodes.
			return RequiredStateDisabled
		}
		// On standard systems, OSDs must be added to controllers while enabled
		return RequiredStateEnabled
	}
	return RequiredStateNone
}

// OSDProvisioningAllowed is a utility function which determines whether OSD
// provisioning is allowed based on the node type, the current cluster
// deployment model, and the current state of the controllers.
func (r *ReconcileHost) OSDProvisioningAllowed(instance *starlingxv1beta1.Host, osdInfo starlingxv1beta1.OSDInfo, tierUUID *string, host *v1info.HostInfo) error {
	clusterName := osdInfo.GetClusterName()

	cluster := host.FindClusterByName(clusterName)
	if cluster == nil {
		// The cluster has not yet been created so wait and retry
		msg := fmt.Sprintf("waiting for the %q cluster to be created before allowing OSDs",
			clusterName)
		m := NewClusterPresenceMonitor(instance, clusterName)
		return r.StartMonitor(m, msg)
	}

	if cluster.DeploymentModel == clusters.DeploymentModelUndefined {
		// The cluster does not yet support OSD provisioning
		msg := "waiting for storage deployment model to be defined before allowing OSDs"
		m := NewClusterDeploymentModelMonitor(instance, cluster.ID)
		return r.StartMonitor(m, msg)

	} else if cluster.DeploymentModel == clusters.DeploymentModelStorage ||
		cluster.DeploymentModel == clusters.DeploymentModelController {
		if r.GetSystemType(instance.Namespace) == titaniumManager.SystemTypeStandard {
			if !r.MonitorsEnabled(hosts.OSDMinimumMonitorCount) {
				msg := fmt.Sprintf("waiting for %d monitor(s) to be enabled before allowing OSDs",
					hosts.OSDMinimumMonitorCount)
				m := NewStorageMonitorCountMonitor(instance, hosts.OSDMinimumMonitorCount)
				return r.StartMonitor(m, msg)
			}
		}
	}

	if tierUUID == nil {
		// The storage tier has not yet been allocated so wait and retry.
		msg := fmt.Sprintf("waiting for the %q %s tier to be created",
			clusterName, storagetiers.StorageTierName)
		m := NewStorageTierMonitor(instance, cluster.ID, storagetiers.StorageTierName)
		return r.StartMonitor(m, msg)
	}

	return nil
}

// buildOSDOpts is a utility function to contructs OSD request parameters
// suitable for use in the system API.
func buildOSDOpts(host *v1info.HostInfo, osdInfo starlingxv1beta1.OSDInfo) (osds.OSDOpts, error) {
	disk, _ := host.FindDiskByPath(osdInfo.Path)
	if disk == nil {
		msg := fmt.Sprintf("unable to find disk for path: %s", osdInfo.Path)
		return osds.OSDOpts{}, starlingxv1beta1.NewMissingSystemResource(msg)
	}

	opts := osds.OSDOpts{
		HostID:   &host.ID,
		DiskID:   &disk.ID,
		Function: &osdInfo.Function,
	}

	if osdInfo.Journal != nil {
		journal, _ := host.FindOSDByPath(osdInfo.Journal.Location)
		if journal == nil {
			msg := fmt.Sprintf("unable to find journal OSD with path: %s",
				osdInfo.Journal.Location)
			return osds.OSDOpts{}, starlingxv1beta1.NewMissingSystemResource(msg)

		} else if journal.Function != osds.FunctionOSD {
			msg := fmt.Sprintf("OSD on disk %s is not a Journal OSD", journal.DiskID)
			return osds.OSDOpts{}, common.NewUserDataError(msg)
		}

		size := osdInfo.Journal.Size
		opts.JournalLocation = &journal.ID
		opts.JournalSize = &size
	}

	tier := host.StorageTiers[osdInfo.GetClusterName()]
	if tier != nil {
		opts.TierUUID = &tier.ID
	}

	return opts, nil
}

// ReconcileOSDsByType is responsible for reconciling the storage OSD
// configuration of a host resource for a specific type of OSD function.
func (r *ReconcileHost) ReconcileOSDsByType(client *gophercloud.ServiceClient, instance *starlingxv1beta1.Host, profile *starlingxv1beta1.HostProfileSpec, host *v1info.HostInfo, function string) error {
	updated := false

	for _, osdInfo := range *profile.Storage.OSDs {
		if osdInfo.Function != function {
			continue
		}

		if osd, ok := host.FindOSDByPath(osdInfo.Path); ok {
			if opts, ok := osdUpdateRequired(&osdInfo, osd); ok {
				// Update the OSD
				log.Info("updating OSD", "uuid", osd.ID, "opts", opts)

				_, err := osds.Update(client, osd.ID, opts).Extract()
				if err != nil {
					err = perrors.Wrapf(err, "failed to update OSD: %s, %s",
						osd.ID, common.FormatStruct(opts))
					return err
				}

				r.NormalEvent(instance, common.ResourceUpdated,
					"OSD %q has been updated", osdInfo.Path)

				updated = true
			}

		} else {
			opts, err := buildOSDOpts(host, osdInfo)
			if err != nil {
				return err
			}

			err = r.OSDProvisioningAllowed(instance, osdInfo, opts.TierUUID, host)
			if err != nil {
				return err
			}

			log.Info("creating OSD", "opts", opts)

			_, err = osds.Create(client, opts).Extract()
			if err != nil {
				err = perrors.Wrap(err, "failed to create OSD")
				return err
			}

			r.NormalEvent(instance, common.ResourceCreated,
				"OSD %q has been created", osdInfo.Path)

			updated = true
		}
	}

	if updated {
		result, err := osds.ListOSDs(client, host.ID)
		if err != nil {
			err = perrors.Wrap(err, "failed to refresh OSD list for host")
			return err
		}

		host.OSDs = result
	}

	return nil
}

// ReconcileOSDs is responsible for reconciling the storage OSD configuration
// of a host resource.
func (r *ReconcileHost) ReconcileOSDs(client *gophercloud.ServiceClient, instance *starlingxv1beta1.Host, profile *starlingxv1beta1.HostProfileSpec, host *v1info.HostInfo) error {

	if profile.Storage.OSDs == nil {
		return nil
	}

	if !config.IsReconcilerEnabled(config.OSD) {
		return nil
	}

	if len(*profile.Storage.OSDs) == 0 {
		return nil
	}

	// Journal OSDs must be added before regular OSDs since regular OSDs must
	// reference Journal OSDs by UUID.
	for _, f := range []string{osds.FunctionJournal, osds.FunctionOSD} {
		err := r.ReconcileOSDsByType(client, instance, profile, host, f)
		if err != nil {
			return err
		}
	}

	return nil
}

// ReconcileFileSystems is responsible for reconciling the storage file system
// configuration of a host resource.
func (r *ReconcileHost) ReconcileFileSystems(client *gophercloud.ServiceClient, instance *starlingxv1beta1.Host, profile *starlingxv1beta1.HostProfileSpec, host *v1info.HostInfo) error {

	if profile.Storage.FileSystems == nil {
		return nil
	}

	if !config.IsReconcilerEnabled(config.FileSystems) {
		return nil
	}

	if len(*profile.Storage.FileSystems) == 0 {
		return nil
	}

	if !host.IsUnlockedAvailable() {
		msg := "waiting for host to reach available state"
		m := NewUnlockedAvailableHostMonitor(instance, host.ID)
		return r.StartMonitor(m, msg)
	}

	updates := make([]hostFilesystems.FileSystemOpts, 0)
	for _, fsInfo := range *profile.Storage.FileSystems {
		found := false
		for _, fs := range host.FileSystems {
			if fs.Name != fsInfo.Name {
				continue
			}

			found = true
			if fsInfo.Size > fs.Size {
				// Update the system resource with the new size.
				opts := hostFilesystems.FileSystemOpts{
					Name: fsInfo.Name,
					Size: fsInfo.Size,
				}

				updates = append(updates, opts)
			}
		}

		if !found {
			msg := fmt.Sprintf("unknown host filesystem %q", fsInfo.Name)
			return starlingxv1beta1.NewMissingSystemResource(msg)
		}
	}

	if len(updates) > 0 {
		log.Info("updating host filesystem sizes", "opts", updates)

		err := hostFilesystems.Update(client, host.ID, updates).ExtractErr()
		if err != nil {
			err = perrors.Wrapf(err, "failed to update filesystems sizes")
			return err
		}

		r.NormalEvent(instance, common.ResourceUpdated, "filesystem sizes have been updated")
	}

	return nil
}

// ReconcileStorage is responsible for reconciling the Storage configuration of
// a host resource.
func (r *ReconcileHost) ReconcileStorage(client *gophercloud.ServiceClient, instance *starlingxv1beta1.Host, profile *starlingxv1beta1.HostProfileSpec, host *v1info.HostInfo) error {
	if !config.IsReconcilerEnabled(config.Storage) {
		return nil
	}

	if profile.Storage == nil {
		return nil
	}

	// TODO(alegacy): For now, we only support adding OSDs, volume groups and
	//  associated partitions.  It is possible, but cumbersome, to make changes
	//  to the configuration so until there is a real need we are only going to
	//  handle the initial provisioning case.

	err := r.ReconcileMonitor(client, instance, profile, host)
	if err != nil {
		return err
	}

	err = r.ReconcileVolumeGroups(client, instance, profile, host)
	if err != nil {
		return err
	}

	err = r.ReconcileStaleOSDs(client, instance, profile, host)
	if err != nil {
		return err
	}

	switch r.OSDProvisioningState(instance.Namespace, host.Personality) {
	case RequiredStateDisabled, RequiredStateAny:
		err = r.ReconcileOSDs(client, instance, profile, host)
		if err != nil {
			return err
		}
	}

	return nil
}
