/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2024 Wind River Systems, Inc. */
package controllers

import (
	"encoding/json"
	"fmt"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/addresspools"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/networks"
	th "github.com/gophercloud/gophercloud/testhelper"
	starlingxv1 "github.com/wind-river/cloud-platform-deployment-manager/api/v1"
	cloudManager "github.com/wind-river/cloud-platform-deployment-manager/controllers/manager"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"net/http"
)

const AddrPoolListBody = `
{
    "addrpools": [
        {
            "gateway_address": null,
            "network": "192.168.204.0",
            "name": "management",
            "ranges": [
                [
                    "192.168.204.2",
                    "192.168.204.50"
                ]
            ],
			"floating_address": "192.168.204.2",
			"controller0_address": "192.168.204.3",
			"controller1_address": "192.168.204.4",
            "prefix": 24,
            "order": "random",
            "uuid": "aa277c8e-7421-4721-ae6a-347771fe4fa6"
        },
		{
            "gateway_address": "10.10.10.1",
            "network": "10.10.10.0",
            "name": "oam",
            "ranges": [
                [
                    "10.10.10.2",
                    "10.10.10.254"
                ]
            ],
			"floating_address": "10.10.10.2",
			"controller0_address": "10.10.10.3",
			"controller1_address": "10.10.10.4",
            "prefix": 24,
            "order": "random",
            "uuid": "384c6eb3-d48b-486e-8151-7dcecd3779df"
        },
		{
            "gateway_address": "192.168.208.1",
            "network": "192.168.208.0",
            "name": "admin",
            "ranges": [
                [
                    "192.168.208.2",
                    "192.168.208.50"
                ]
            ],
			"floating_address": "192.168.208.2",
			"controller0_address": "192.168.208.3",
			"controller1_address": "192.168.208.4",
            "prefix": 24,
            "order": "random",
            "uuid": "be2eb19c-4b47-88ec-82c5-6b29097cf439"
        },
        {
            "gateway_address": null,
            "network": "169.254.202.0",
            "name": "pxeboot",
            "ranges": [
                [
                    "169.254.202.1",
                    "169.254.202.254"
                ]
            ],
			"floating_address": "169.254.202.2",
			"controller0_address": "169.254.202.3",
			"controller1_address": "169.254.202.4",
            "prefix": 24,
            "order": "random",
            "uuid": "28f8fabb-43df-4458-a256-d9195e2b669e"
        }
    ]
}
`

const NetworkListBody = `
{
    "networks": [
        {
			"dynamic": false,
			"id": 1,
			"name": "admin",
			"pool_uuid": "be2eb19c-4b47-88ec-82c5-6b29097cf439",
			"type": "admin",
			"uuid": "c434c909-f2eb-4a4e-87f1-525cbe9b1ec2"
        },
        {
			"dynamic": true,
			"id": 2,
			"name": "mgmt",
			"pool_uuid": "aa277c8e-7421-4721-ae6a-347771fe4fa6",
			"type": "mgmt",
			"uuid": "a48a7b6d-9cfa-24a4-8d48-f0e25d35984a"
        },
		{
			"dynamic": false,
			"id": 3,
			"name": "oam",
			"pool_uuid": "384c6eb3-d48b-486e-8151-7dcecd3779df",
			"type": "oam",
			"uuid": "32665423-d48b-486e-8151-7dcecd3779df"
		},
		{
			"dynamic": true,
			"id": 4,
			"name": "pxeboot",
			"pool_uuid": "28f8fabb-43df-4458-a256-d9195e2b669e",
			"type": "pxeboot",
			"uuid": "0bebc4ef-e8e4-1248-b9d5-8694a79f58cc"
		}
    ]
}
`

const OAMNetworkListBody = `
{
	"iextoams": [
		{
			"uuid": "32665423-d48b-486e-8151-7dcecd3779df",
			"oam_subnet": "10.10.10.0/24",
			"oam_gateway_ip": "10.10.10.1",
			"oam_floating_ip": "10.10.10.2",
			"oam_c0_ip": "10.10.10.3",
			"oam_c1_ip": "10.10.10.4",
			"oam_start_ip": "10.10.10.2",
			"oam_end_ip": "10.10.10.254",
			"region_config": false,
			"isystem_uuid": "607671a2-15a7-4f97-9297-c4e1804cde12",
			"links": [
				{
					"href": "http://192.168.204.2:6385/v1/iextoams/32665423-d48b-486e-8151-7dcecd3779df",
					"rel": "self"
				}, {
					"href": "http://192.168.204.2:6385/iextoams/32665423-d48b-486e-8151-7dcecd3779df",
					"rel": "bookmark"
				}
			],
			"created_at": "2023-11-28T13:10:53.200531+00:00",
			"updated_at": null
		}
	]
}
`

const SingleSystemBody = `
{
	"isystems": [
		{
			"system_mode": "simplex",
			"created_at": "2019-08-07T14:32:41.617713+00:00",
			"links": [
				{
					"href": "http://192.168.204.2:6385/v1/isystems/5af5f7e5-1eea-4e76-b539-ac552e132e47",
					"rel": "self"
				},
				{
					"href": "http://192.168.204.2:6385/isystems/5af5f7e5-1eea-4e76-b539-ac552e132e47",
					"rel": "bookmark"
				}
			],
			"security_feature": "spectre_meltdown_v1",
			"description": "Test System",
			"software_version": "19.01",
			"service_project_name": "services",
			"updated_at": "2019-08-07T14:45:50.822509+00:00",
			"distributed_cloud_role": null,
			"location": "vbox",
			"capabilities": {
				"sdn_enabled": false,
				"shared_services": "[]",
				"bm_region": "External",
				"vswitch_type": "none",
				"https_enabled": false,
				"region_config": false
			},
			"name": "Herp",
			"contact": "info@windriver.com",
			"system_type": "All-in-one",
			"timezone": "UTC",
			"region_name": "RegionOne",
			"uuid": "5af5f7e5-1eea-4e76-b539-ac552e132e47"
		}
    ]
}
`

const HostsListBody = `
{
  "ihosts": [
    {
      "action": "none",
      "administrative": "unlocked",
      "apparmor": "disabled",
      "hw_settle": "0",
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
      "subfunction_oper": "enabled",
      "subfunctions": "controller,worker",
      "target_load": "19.01",
      "task": null,
      "tboot": "false",
      "ttys_dcd": null,
      "updated_at": "2019-08-07T15:01:23.348321+00:00",
      "uptime": 3490,
      "uuid": "d99637e9-5451-45c6-98f4-f18968e43e91",
      "vim_progress_status": null,
      "max_cpu_mhz_configured": "1800"
    }
  ]
}
`

const DummyAddressPoolUpdateResponse = `
{
	"gateway_address": null,
	"network": "169.254.202.0",
	"name": "dummy",
	"ranges": [
		[
			"169.254.202.1",
			"169.254.202.254"
		]
	],
	"floating_address": "169.254.202.2",
	"controller0_address": "169.254.202.3",
	"controller1_address": "169.254.202.4",
	"prefix": 24,
	"order": "random",
	"uuid": "123914e3-36e4-41a8-a702-d9f6e54d7140"
}
`

const DummyNetworkUpdateResponse = `
{
    "dynamic": false,
    "id": 2,
    "name": "dummy",
    "pool_uuid": "c7ac5a0c-606b-4fe0-9065-28a8c8fb78cc",
    "type": "oam",
    "uuid": "f757b5c7-89ab-4d93-bfd7-a97780ec2c1e"
}
`

const DummyOAMUpdateResponse = `
{
    "uuid": "727bd796-070f-40c2-8b9b-7ed674fd0fe7",
	"oam_subnet": "10.10.20.0/24",
	"oam_gateway_ip": null,
	"oam_floating_ip": "10.10.20.5",
	"oam_c0_ip": "10.10.20.3",
	"oam_c1_ip": "10.10.20.4"
}
`

var HostsListBodyResponse string
var SingleSystemBodyResponse string

func HandleAddressPoolRequests(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		w.Header().Add("Content-Type", "application/json")
		fmt.Fprint(w, AddrPoolListBody)
	case http.MethodPost:
		w.Header().Add("Content-Type", "application/json")
		fmt.Fprint(w, DummyAddressPoolUpdateResponse)
	case http.MethodPatch:
		w.Header().Add("Content-Type", "application/json")
		fmt.Fprint(w, DummyAddressPoolUpdateResponse)
	case http.MethodDelete:
		w.WriteHeader(http.StatusNoContent)
	default:
		http.Error(w, `{"error": "Method not allowed"}`, http.StatusMethodNotAllowed)
	}
}

func AddressPoolAPIS() {
	th.Mux.HandleFunc("/addrpools", HandleAddressPoolRequests)
}

func HandleNetworkRequests(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		w.Header().Add("Content-Type", "application/json")
		fmt.Fprint(w, NetworkListBody)
	case http.MethodPost:
		w.Header().Add("Content-Type", "application/json")
		fmt.Fprint(w, DummyNetworkUpdateResponse)
	case http.MethodPatch:
		w.Header().Add("Content-Type", "application/json")
		fmt.Fprint(w, DummyNetworkUpdateResponse)
	case http.MethodDelete:
		w.WriteHeader(http.StatusNoContent)
	default:
		http.Error(w, `{"error": "Method not allowed"}`, http.StatusMethodNotAllowed)
	}
}

func NetworkAPIS() {
	th.Mux.HandleFunc("/networks", HandleNetworkRequests)
}

func HandleOAMNetworkRequests(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		w.Header().Add("Content-Type", "application/json")
		fmt.Fprint(w, OAMNetworkListBody)
	case http.MethodPatch:
		w.Header().Add("Content-Type", "application/json")
		fmt.Fprint(w, DummyOAMUpdateResponse)
	case http.MethodDelete:
		w.WriteHeader(http.StatusNoContent)
	default:
		http.Error(w, `{"error": "Method not allowed"}`, http.StatusMethodNotAllowed)
	}
}

func OAMNetworkAPIS() {
	th.Mux.HandleFunc("/iextoam", HandleOAMNetworkRequests)
}

func HandleSystemRequests(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "application/json")
	switch r.Method {
	case http.MethodGet:
		fmt.Fprint(w, SingleSystemBodyResponse)
	default:
		http.Error(w, `{"error": "Method not allowed"}`, http.StatusMethodNotAllowed)
	}
}

func SystemAPIS() {
	th.Mux.HandleFunc("/isystems", HandleSystemRequests)
}

func HandleHostRequests(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "application/json")
	switch r.Method {
	case http.MethodGet:
		fmt.Fprint(w, HostsListBodyResponse)
	default:
		http.Error(w, `{"error": "Method not allowed"}`, http.StatusMethodNotAllowed)
	}
}

func HostAPIS() {
	th.Mux.HandleFunc("/ihosts", HandleHostRequests)
}

func GetPlatformNetworksFromFixtures(namespace string) map[string]*starlingxv1.PlatformNetwork {
	PlatformNetworks := make(map[string]*starlingxv1.PlatformNetwork)

	var Networks struct {
		NetworkList []networks.Network `json:"networks"`
	}
	var AddressPools struct {
		AddressPoolList []addresspools.AddressPool `json:"addrpools"`
	}

	_ = json.Unmarshal([]byte(NetworkListBody), &Networks)
	_ = json.Unmarshal([]byte(AddrPoolListBody), &AddressPools)

	for _, network := range Networks.NetworkList {
		allocation_order := networks.AllocationOrderDynamic
		if !network.Dynamic {
			allocation_order = networks.AllocationOrderStatic
		}

		for _, addrpool := range AddressPools.AddressPoolList {
			if network.PoolUUID == addrpool.ID {
				PlatformNetworks[network.Type] = &starlingxv1.PlatformNetwork{
					ObjectMeta: metav1.ObjectMeta{
						Name:      network.Name,
						Namespace: namespace,
					},
					Spec: starlingxv1.PlatformNetworkSpec{
						Type:    network.Type,
						Subnet:  addrpool.Network,
						Prefix:  addrpool.Prefix,
						Gateway: addrpool.Gateway,
						Allocation: starlingxv1.AllocationInfo{
							Type:   allocation_order,
							Order:  &addrpool.Order,
							Ranges: []starlingxv1.AllocationRange{{Start: addrpool.Ranges[0][0], End: addrpool.Ranges[0][1]}},
						},
						FloatingAddress:    addrpool.FloatingAddress,
						Controller0Address: addrpool.Controller0Address,
						Controller1Address: addrpool.Controller1Address,
					},
				}
			}
		}
	}

	return PlatformNetworks
}

func StartPlatformNetworkAPIHandlers() {
	HostsListBodyResponse = HostsListBody
	SingleSystemBodyResponse = SingleSystemBody
	AddressPoolAPIS()
	NetworkAPIS()
	OAMNetworkAPIS()
	SystemAPIS()
	HostAPIS()

	var Networks struct {
		NetworkList []networks.Network `json:"networks"`
	}
	_ = json.Unmarshal([]byte(NetworkListBody), &Networks)

	for _, network := range Networks.NetworkList {
		if network.Type == cloudManager.OAMNetworkType {
			th.Mux.HandleFunc("/iextoam/"+network.UUID, HandleOAMNetworkRequests)
		} else {
			th.Mux.HandleFunc("/networks/"+network.UUID, HandleNetworkRequests)
		}
		th.Mux.HandleFunc("/addrpools/"+network.PoolUUID, HandleAddressPoolRequests)
	}
}
