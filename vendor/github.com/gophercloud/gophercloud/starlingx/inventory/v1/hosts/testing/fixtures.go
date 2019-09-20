/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019 Wind River Systems, Inc. */

package testing

import (
	"fmt"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/hosts"
	"github.com/gophercloud/gophercloud/testhelper/client"
	"net/http"
	"testing"

	th "github.com/gophercloud/gophercloud/testhelper"
)

var (
	clockSynchronization = "ntp"
	invState             = "inventoried"
	task                 = "Unlocking"
	standbyController    = "Controller-Standby"
	activeController     = "Controller-Active"
	monitorFunction      = "monitor"
	vboxLocation         = "vbox"
	icewallLocation      = "The Ice Wall"
	ottawaLocation       = "Ottawa, Canada"
	HostHerp             = hosts.Host{
		ID:           "d99637e9-5451-45c6-98f4-f18968e43e91",
		Hostname:     "controller-0",
		Personality:  "controller",
		SubFunctions: "controller,worker",
		Capabilities: hosts.Capabilities{
			StorFunction: &monitorFunction,
			Personality:  &activeController,
		},
		Location:             hosts.Location{Name: &vboxLocation},
		InstallOutput:        "text",
		Console:              "tty0",
		BootMAC:              "08:08:08:08:08:08",
		BootIP:               "1.2.3.4",
		RootDevice:           "/dev/disk/by-path/pci-0000:00:0d.0-ata-1.0",
		BootDevice:           "/dev/disk/by-path/pci-0000:00:0d.0-ata-1.0",
		BMType:               nil,
		BMAddress:            nil,
		BMUsername:           nil,
		SerialNumber:         nil,
		AssetTag:             nil,
		ConfigurationStatus:  "Config out-of-date",
		ConfigurationApplied: "53296bf3-d205-4675-8c57-411c875c164e",
		ConfigurationTarget:  "2da20c19-2157-4231-b9b3-194175e7dad0",
		Task:                 nil,
		AdministrativeState:  "unlocked",
		OperationalStatus:    "enabled",
		AvailabilityStatus:   "online",
		InventoryState:       &invState,
		ClockSynchronization: &clockSynchronization,
	}
	HostDerp = hosts.Host{
		ID:           "f73dda8e-be3c-4704-ad1e-ed99e44b846e",
		Hostname:     "controller-1",
		Personality:  "controller",
		SubFunctions: "controller,worker",
		Capabilities: hosts.Capabilities{
			StorFunction: &monitorFunction,
			Personality:  &standbyController,
		},
		Location:             hosts.Location{Name: &icewallLocation},
		InstallOutput:        "graphic",
		Console:              "tty0",
		BootMAC:              "01:02:03:04:05:06",
		BootIP:               "4.3.2.1",
		RootDevice:           "/dev/disk/by-path/pci-0000:00:0d.0-ata-1.0",
		BootDevice:           "/dev/disk/by-path/pci-0000:00:0d.0-ata-1.0",
		BMType:               nil,
		BMAddress:            nil,
		BMUsername:           nil,
		SerialNumber:         nil,
		AssetTag:             nil,
		ConfigurationStatus:  "Config out-of-date",
		ConfigurationApplied: "2491d46f-d3ad-4460-981f-f86dc5a8bf6c",
		ConfigurationTarget:  "dd798e27-362e-49e7-9dc3-4e8ef7d0aa59",
		Task:                 &task,
		AdministrativeState:  "locked",
		OperationalStatus:    "disabled",
		AvailabilityStatus:   "online",
		InventoryState:       nil,
	}
	HostMerp = hosts.Host{
		ID:           "66b62c51-974b-4bcc-b273-e8365833157e",
		Hostname:     "compute-0",
		Personality:  "compute",
		SubFunctions: "worker",
		Capabilities: hosts.Capabilities{
			StorFunction: &monitorFunction,
			Personality:  nil,
		},
		Location:             hosts.Location{Name: &ottawaLocation},
		InstallOutput:        "text",
		Console:              "tty0",
		BootMAC:              "fe:00:27:af:22:96",
		BootIP:               "2.4.8.16",
		RootDevice:           "/dev/disk/by-path/pci-0000:00:0d.0-ata-1.0",
		BootDevice:           "/dev/disk/by-path/pci-0000:00:0d.0-ata-1.0",
		BMType:               nil,
		BMAddress:            nil,
		BMUsername:           nil,
		SerialNumber:         nil,
		AssetTag:             nil,
		ConfigurationStatus:  "",
		ConfigurationApplied: "",
		ConfigurationTarget:  "",
		Task:                 nil,
		AdministrativeState:  "locked",
		OperationalStatus:    "disabled",
		AvailabilityStatus:   "online",
		InventoryState:       nil,
	}
)

const HostsListBody = `
{
  "ihosts": [
    {
      "action": "none",
      "administrative": "unlocked",
      "availability": "online",
      "bm_ip": null,
      "bm_type": null,
      "bm_username": null,
      "boot_device": "/dev/disk/by-path/pci-0000:00:0d.0-ata-1.0",
      "capabilities": {
        "Personality": "Controller-Active",
        "stor_function": "monitor"
      },
      "clock_synchronization": "ntp",
      "config_applied": "53296bf3-d205-4675-8c57-411c875c164e",
      "config_status": "Config out-of-date",
      "config_target": "2da20c19-2157-4231-b9b3-194175e7dad0",
      "console": "tty0",
      "created_at": "2019-08-07T14:42:25.415277+00:00",
      "hostname": "controller-0",
      "id": 1,
      "ihost_action": null,
      "install_output": "text",
      "install_state": null,
      "install_state_info": null,
      "inv_state": "inventoried",
      "invprovision": "provisioning",
      "iprofile_uuid": null,
      "iscsi_initiator_name": null,
      "isystem_uuid": "5af5f7e5-1eea-4e76-b539-ac552e132e47",
      "links": [
        {
          "href": "http://192.168.204.2:6385/v1/ihosts/d99637e9-5451-45c6-98f4-f18968e43e91",
          "rel": "self"
        },
        {
          "href": "http://192.168.204.2:6385/ihosts/d99637e9-5451-45c6-98f4-f18968e43e91",
          "rel": "bookmark"
        }
      ],
      "location": {
        "locn": "vbox"
      },
      "mgmt_ip": "1.2.3.4",
      "mgmt_mac": "08:08:08:08:08:08",
      "mtce_info": null,
      "operational": "enabled",
      "peers": null,
      "personality": "controller",
      "reserved": "False",
      "rootfs_device": "/dev/disk/by-path/pci-0000:00:0d.0-ata-1.0",
      "serialid": null,
      "software_load": "19.01",
      "subfunction_avail": "online",
      "subfunction_oper": "disabled",
      "subfunctions": "controller,worker",
      "target_load": "19.01",
      "task": null,
      "tboot": "false",
      "ttys_dcd": null,
      "updated_at": "2019-08-07T15:01:23.348321+00:00",
      "uptime": 3490,
      "uuid": "d99637e9-5451-45c6-98f4-f18968e43e91",
      "vim_progress_status": null
    },
    {
      "action": "none",
      "administrative": "locked",
      "availability": "online",
      "bm_ip": null,
      "bm_type": null,
      "bm_username": null,
      "boot_device": "/dev/disk/by-path/pci-0000:00:0d.0-ata-1.0",
      "capabilities": {
        "Personality": "Controller-Standby",
        "stor_function": "monitor"
      },
      "clock_synchronization": null,
      "config_applied": "2491d46f-d3ad-4460-981f-f86dc5a8bf6c",
      "config_status": "Config out-of-date",
      "config_target": "dd798e27-362e-49e7-9dc3-4e8ef7d0aa59",
      "console": "tty0",
      "created_at": "2019-08-08T15:10:16.810867+00:00",
      "hostname": "controller-1",
      "id": 2,
      "ihost_action": "unlock",
      "install_output": "graphic",
      "install_state": "completed+",
      "install_state_info": null,
      "inv_state": null,
      "invprovision": "unprovisioned",
      "iprofile_uuid": null,
      "iscsi_initiator_name": null,
      "isystem_uuid": "5af5f7e5-1eea-4e76-b539-ac552e132e47",
      "links": [
        {
          "href": "http://192.168.204.2:6385/v1/ihosts/f73dda8e-be3c-4704-ad1e-ed99e44b846e",
          "rel": "self"
        },
        {
          "href": "http://192.168.204.2:6385/ihosts/f73dda8e-be3c-4704-ad1e-ed99e44b846e",
          "rel": "bookmark"
        }
      ],
      "location": {
        "locn": "The Ice Wall"
      },
      "mgmt_ip": "4.3.2.1",
      "mgmt_mac": "01:02:03:04:05:06",
      "mtce_info": null,
      "operational": "disabled",
      "peers": null,
      "personality": "controller",
      "reserved": "False",
      "rootfs_device": "/dev/disk/by-path/pci-0000:00:0d.0-ata-1.0",
      "serialid": null,
      "software_load": "19.01",
      "subfunction_avail": "online",
      "subfunction_oper": "disabled",
      "subfunctions": "controller,worker",
      "target_load": "19.01",
      "task": "Unlocking",
      "tboot": "false",
      "ttys_dcd": null,
      "updated_at": "2019-08-08T15:31:58.699163+00:00",
      "uptime": 149,
      "uuid": "f73dda8e-be3c-4704-ad1e-ed99e44b846e",
      "vim_progress_status": null
    },
    {
      "action": "none",
      "administrative": "locked",
      "availability": "online",
      "bm_ip": null,
      "bm_type": null,
      "bm_username": null,
      "boot_device": "/dev/disk/by-path/pci-0000:00:0d.0-ata-1.0",
      "capabilities": {
        "Personality": null,
        "stor_function": "monitor"
      },
      "clock_synchronization": null,
      "config_applied": null,
      "config_status": null,
      "config_target": null,
      "console": "tty0",
      "created_at": "2019-08-08T15:10:16.810867+00:00",
      "hostname": "compute-0",
      "id": 2,
      "ihost_action": "unlock",
      "install_output": "text",
      "install_state": "completed+",
      "install_state_info": null,
      "inv_state": null,
      "invprovision": "unprovisioned",
      "iprofile_uuid": null,
      "iscsi_initiator_name": null,
      "isystem_uuid": "5af5f7e5-1eea-4e76-b539-ac552e132e47",
      "links": [
        {
          "href": "http://192.168.204.2:6385/v1/ihosts/f73dda8e-be3c-4704-ad1e-ed99e44b846e",
          "rel": "self"
        },
        {
          "href": "http://192.168.204.2:6385/ihosts/f73dda8e-be3c-4704-ad1e-ed99e44b846e",
          "rel": "bookmark"
        }
      ],
      "location": {
        "locn": "Ottawa, Canada"
      },
      "mgmt_ip": "2.4.8.16",
      "mgmt_mac": "fe:00:27:af:22:96",
      "mtce_info": null,
      "operational": "disabled",
      "peers": null,
      "personality": "compute",
      "reserved": "False",
      "rootfs_device": "/dev/disk/by-path/pci-0000:00:0d.0-ata-1.0",
      "serialid": null,
      "software_load": "19.01",
      "subfunction_avail": "online",
      "subfunction_oper": "disabled",
      "subfunctions": "worker",
      "target_load": "19.01",
      "task": null,
      "tboot": "false",
      "ttys_dcd": null,
      "updated_at": "2019-08-08T15:31:58.699163+00:00",
      "uptime": 149,
      "uuid": "66b62c51-974b-4bcc-b273-e8365833157e",
      "vim_progress_status": null
    }
  ]
}
`

const SingleHostBody = `
{
      "action": "none",
      "administrative": "locked",
      "availability": "online",
      "bm_ip": null,
      "bm_type": null,
      "bm_username": null,
      "boot_device": "/dev/disk/by-path/pci-0000:00:0d.0-ata-1.0",
      "capabilities": {
        "Personality": "Controller-Standby",
        "stor_function": "monitor"
      },
      "clock_synchronization": null,
      "config_applied": "2491d46f-d3ad-4460-981f-f86dc5a8bf6c",
      "config_status": "Config out-of-date",
      "config_target": "dd798e27-362e-49e7-9dc3-4e8ef7d0aa59",
      "console": "tty0",
      "created_at": "2019-08-08T15:10:16.810867+00:00",
      "hostname": "controller-1",
      "id": 2,
      "ihost_action": "unlock",
      "install_output": "graphic",
      "install_state": "completed+",
      "install_state_info": null,
      "inv_state": null,
      "invprovision": "unprovisioned",
      "iprofile_uuid": null,
      "iscsi_initiator_name": null,
      "isystem_uuid": "5af5f7e5-1eea-4e76-b539-ac552e132e47",
      "links": [
        {
          "href": "http://192.168.204.2:6385/v1/ihosts/f73dda8e-be3c-4704-ad1e-ed99e44b846e",
          "rel": "self"
        },
        {
          "href": "http://192.168.204.2:6385/ihosts/f73dda8e-be3c-4704-ad1e-ed99e44b846e",
          "rel": "bookmark"
        }
      ],
      "location": {
        "locn": "The Ice Wall"
      },
      "mgmt_ip": "4.3.2.1",
      "mgmt_mac": "01:02:03:04:05:06",
      "mtce_info": null,
      "operational": "disabled",
      "peers": null,
      "personality": "controller",
      "reserved": "False",
      "rootfs_device": "/dev/disk/by-path/pci-0000:00:0d.0-ata-1.0",
      "serialid": null,
      "software_load": "19.01",
      "subfunction_avail": "online",
      "subfunction_oper": "disabled",
      "subfunctions": "controller,worker",
      "target_load": "19.01",
      "task": "Unlocking",
      "tboot": "false",
      "ttys_dcd": null,
      "updated_at": "2019-08-08T15:31:58.699163+00:00",
      "uptime": 149,
      "uuid": "f73dda8e-be3c-4704-ad1e-ed99e44b846e",
      "vim_progress_status": null
}
`

func HandleHostListSuccessfully(t *testing.T) {
	th.Mux.HandleFunc("/ihosts/", func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "GET")
		th.TestHeader(t, r, "X-Auth-Token", client.TokenID)
		w.Header().Add("Content-Type", "application/json")
		fmt.Fprintf(w, HostsListBody)
	})
}

func HandleHostGetSuccessfully(t *testing.T) {
	th.Mux.HandleFunc("/ihosts/f757b5c7-89ab-4d93-bfd7-a97780ec2c1e", func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "GET")
		th.TestHeader(t, r, "X-Auth-Token", client.TokenID)
		w.Header().Add("Content-Type", "application/json")
		fmt.Fprintf(w, SingleHostBody)
	})
}

func HandleHostUpdateSuccessfully(t *testing.T) {
	th.Mux.HandleFunc("/ihosts/f757b5c7-89ab-4d93-bfd7-a97780ec2c1e", func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "PATCH")
		th.TestHeader(t, r, "X-Auth-Token", client.TokenID)
		th.TestHeader(t, r, "Accept", "application/json")
		th.TestHeader(t, r, "Content-Type", "application/json")
		th.TestJSONRequest(t, r, `[ { "op": "replace", "path": "/hostname", "value": "new-name" } ]`)
		fmt.Fprintf(w, SingleHostBody)
	})
}

func HandleHostDeletionSuccessfully(t *testing.T) {
	th.Mux.HandleFunc("/ihosts/f757b5c7-89ab-4d93-bfd7-a97780ec2c1e", func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "DELETE")
		th.TestHeader(t, r, "X-Auth-Token", client.TokenID)

		w.WriteHeader(http.StatusNoContent)
	})
}

func HandleHostCreationSuccessfully(t *testing.T, response string) {
	th.Mux.HandleFunc("/ihosts", func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "POST")
		th.TestHeader(t, r, "X-Auth-Token", client.TokenID)
		th.TestJSONRequest(t, r, `{
          "console": "tty0",
          "hostname": "controller-1",
          "install_output": "graphic",
          "location": {
            "locn": "The Ice Wall"
          },
          "personality": "controller",
          "subfunctions": "controller,worker"
        }`)

		w.WriteHeader(http.StatusAccepted)
		w.Header().Add("Content-Type", "application/json")
		fmt.Fprintf(w, response)
	})
}
