/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019 Wind River Systems, Inc. */

package testing

import (
	"fmt"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/disks"
	"github.com/gophercloud/gophercloud/testhelper/client"
	"net/http"
	"testing"

	th "github.com/gophercloud/gophercloud/testhelper"
)

var (
	serialIDs = "VB3fea9502-68eb7238"
	storIDs   = "df71b759-999e-42be-95ce-6fafa9ef3d17"
	updated   = [2]string{"2019-08-08T15:09:26.897705+00:00", "2019-08-08T15:09:25.446203+00:00"}
	DiskHerp  = disks.Disk{
		ID:               "b45be1ad-585b-466b-accf-b1fdca885457",
		DevicePath:       "/dev/disk/by-path/pci-0000:00:0d.0-ata-1.0",
		DeviceNode:       "/dev/sda",
		DeviceType:       "SSD",
		DeviceWWN:        nil,
		DeviceID:         "ata-VBOX_HARDDISK_VB3fea9502-68eb7238",
		DeviceNumber:     2048,
		Size:             245760,
		AvailableSpace:   27624,
		PhysicalVolumeID: nil,
		SerialID:         &serialIDs,
		StorID:           &storIDs,
		Capabilities:     disks.Capabilities{},
		RPM:              "N/A",
		CreatedAt:        "2019-08-07T14:42:44.871550+00:00",
		UpdatedAt:        &updated[0],
	}
	DiskDerp = disks.Disk{
		ID:               "c8e0a268-dd2e-4001-ba8c-21875325d01c",
		DevicePath:       "/dev/disk/by-path/pci-0000:00:0d.0-ata-2.0",
		DeviceNode:       "/dev/sdb",
		DeviceType:       "HDD",
		DeviceWWN:        nil,
		DeviceID:         "ata-VBOX_HARDDISK_VB7baf1d93-c5bb3149",
		DeviceNumber:     2064,
		Size:             245760,
		AvailableSpace:   0,
		PhysicalVolumeID: nil,
		SerialID:         nil,
		StorID:           nil,
		Capabilities:     disks.Capabilities{},
		RPM:              "N/A",
		CreatedAt:        "2019-08-07T14:42:44.908233+00:00",
		UpdatedAt:        &updated[1],
	}
)

const DisksListBody = `
{
  "idisks": [
    {
      "available_mib": 27624,
      "capabilities": {
        "model_num": "VBOX HARDDISK",
        "stor_function": "rootfs"
      },
      "created_at": "2019-08-07T14:42:44.871550+00:00",
      "device_id": "ata-VBOX_HARDDISK_VB3fea9502-68eb7238",
      "device_node": "/dev/sda",
      "device_num": 2048,
      "device_path": "/dev/disk/by-path/pci-0000:00:0d.0-ata-1.0",
      "device_type": "SSD",
      "device_wwn": null,
      "ihost_uuid": "d99637e9-5451-45c6-98f4-f18968e43e91",
      "ipv_uuid": null,
      "istor_uuid": "df71b759-999e-42be-95ce-6fafa9ef3d17",
      "links": [
        {
          "href": "http://192.168.204.2:6385/v1/idisks/b45be1ad-585b-466b-accf-b1fdca885457",
          "rel": "self"
        },
        {
          "href": "http://192.168.204.2:6385/idisks/b45be1ad-585b-466b-accf-b1fdca885457",
          "rel": "bookmark"
        }
      ],
      "rpm": "N/A",
      "serial_id": "VB3fea9502-68eb7238",
      "size_mib": 245760,
      "updated_at": "2019-08-08T15:09:26.897705+00:00",
      "uuid": "b45be1ad-585b-466b-accf-b1fdca885457"
    },
    {
      "available_mib": 0,
      "capabilities": {
        "model_num": "VBOX HARDDISK"
      },
      "created_at": "2019-08-07T14:42:44.908233+00:00",
      "device_id": "ata-VBOX_HARDDISK_VB7baf1d93-c5bb3149",
      "device_node": "/dev/sdb",
      "device_num": 2064,
      "device_path": "/dev/disk/by-path/pci-0000:00:0d.0-ata-2.0",
      "device_type": "HDD",
      "device_wwn": null,
      "ihost_uuid": "d99637e9-5451-45c6-98f4-f18968e43e91",
      "ipv_uuid": null,
      "istor_uuid": null,
      "links": [
        {
          "href": "http://192.168.204.2:6385/v1/idisks/c8e0a268-dd2e-4001-ba8c-21875325d01c",
          "rel": "self"
        },
        {
          "href": "http://192.168.204.2:6385/idisks/c8e0a268-dd2e-4001-ba8c-21875325d01c",
          "rel": "bookmark"
        }
      ],
      "rpm": "N/A",
      "serial_id": null,
      "size_mib": 245760,
      "updated_at": "2019-08-08T15:09:25.446203+00:00",
      "uuid": "c8e0a268-dd2e-4001-ba8c-21875325d01c"
    }
  ]
}
`

const DisksSingleBody = `
    {
      "available_mib": 0,
      "capabilities": {
        "model_num": "VBOX HARDDISK"
      },
      "created_at": "2019-08-07T14:42:44.908233+00:00",
      "device_id": "ata-VBOX_HARDDISK_VB7baf1d93-c5bb3149",
      "device_node": "/dev/sdb",
      "device_num": 2064,
      "device_path": "/dev/disk/by-path/pci-0000:00:0d.0-ata-2.0",
      "device_type": "HDD",
      "device_wwn": null,
      "ihost_uuid": "d99637e9-5451-45c6-98f4-f18968e43e91",
      "ipv_uuid": null,
      "istor_uuid": null,
      "links": [
        {
          "href": "http://192.168.204.2:6385/v1/idisks/c8e0a268-dd2e-4001-ba8c-21875325d01c",
          "rel": "self"
        },
        {
          "href": "http://192.168.204.2:6385/idisks/c8e0a268-dd2e-4001-ba8c-21875325d01c",
          "rel": "bookmark"
        }
      ],
      "rpm": "N/A",
      "serial_id": null,
      "size_mib": 245760,
      "updated_at": "2019-08-08T15:09:25.446203+00:00",
      "uuid": "c8e0a268-dd2e-4001-ba8c-21875325d01c"
    }
`

func HandleDisksListSuccessfully(t *testing.T) {
	th.Mux.HandleFunc("/ihosts/f73dda8e-be3c-4704-ad1e-ed99e44b846e/idisks", func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "GET")
		th.TestHeader(t, r, "X-Auth-Token", client.TokenID)
		w.Header().Add("Content-Type", "application/json")
		fmt.Fprintf(w, DisksListBody)
	})
}

func HandleDiskGetSuccessfully(t *testing.T) {
	th.Mux.HandleFunc("/idisks/c8e0a268-dd2e-4001-ba8c-21875325d01c", func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "GET")
		th.TestHeader(t, r, "X-Auth-Token", client.TokenID)
		w.Header().Add("Content-Type", "application/json")
		fmt.Fprintf(w, DisksSingleBody)
	})
}
