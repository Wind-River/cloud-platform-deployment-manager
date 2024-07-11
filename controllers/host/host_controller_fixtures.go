/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2024 Wind River Systems, Inc. */
package host

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/addresspools"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/networkAddressPools"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/networks"
	th "github.com/gophercloud/gophercloud/testhelper"
	starlingxv1 "github.com/wind-river/cloud-platform-deployment-manager/api/v1"
	utils "github.com/wind-river/cloud-platform-deployment-manager/common"
	cloudManager "github.com/wind-river/cloud-platform-deployment-manager/controllers/manager"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var AddrPoolListBody string
var NetworkListBody string
var NetworkAddressPoolListBody string

const AddrPoolListBodyNone = `{
	"addrpools": []
}`

const NetworkAddrPoolListBodyNone = `{
	"network_addresspools": []
}`

const NetworksListBodyNone = `{
	"networks": []
}`

const AddrPoolListBodyFull = `
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
		"network": "200.200.200.100",
		"name": "oam",
		"ranges": [
		  [
			"169.254.202.1",
			"169.254.202.254"
		  ]
		],
		"controller0_address": "200.200.200.3",
		"controller1_address": "200.200.200.4",
		"prefix": 24,
		"order": "random",
		"uuid": "384c6eb3-d48b-486e-8151-7dcecd3779df"
	  },
	  {
		"gateway_address": "200.200.200.1",
		"network": "200.200.200.100",
		"name": "admin",
		"ranges": [
		  [
			"169.254.202.1",
			"169.254.202.254"
		  ]
		],
		"floating_address": "200.200.200.2",
		"controller0_address": "200.200.200.3",
		"controller1_address": "200.200.200.4",
		"prefix": 24,
		"order": "random",
		"uuid": "be2eb19c-4b47-88ec-82c5-6b29097cf439"
	  },
	  {
		"gateway_address": null,
		"network": "200::200",
		"name": "pxeboot",
		"ranges": [
		  [
			"200::200:1",
			"200::200:254"
		  ]
		],
		"controller1_address": "200::200:4",
		"prefix": 24,
		"order": "random",
		"uuid": "28f8fabb-43df-4458-a256-d9195e2b669e",
		"allocation": {
			"ranges": [
			  {
				"start": "200::200:6",
				"end": "200::200:254"
			  }
			]
		  }
	  },
	  {
		"gateway_address": "200::200:1",
		"network": "200::200",
		"name": "oam-ipv6",
		"ranges": [
		  [
			"200::200:2",
			"200::200:254"
		  ]
		],
		"floating_address": "200::200:2",
		"controller0_address": "200::200:3",
		"controller1_address": "200::200:4",
		"prefix": 64,
		"order": "random",
		"uuid": "384c6eb3-d48b-486e-8151-7dcecd377666"
	  },
	  {
		"gateway_address": "200::200:1",
		"network": "200::200",
		"name": "admin-ipv6",
		"ranges": [
		  [
			"200::200:2",
			"200::200:50"
		  ]
		],
		"floating_address": "200::200:2",
		"controller0_address": "200::200:3",
		"controller1_address": "200::200:4",
		"prefix": 64,
		"order": "random",
		"uuid": "be2eb19c-4b47-88ec-82c5-6b29097cf666"
	  },
	  {
		"gateway_address": null,
		"network": "100::100",
		"name": "cluster-host-ipv6",
		"ranges": [
		  [
			"100::100:2",
			"100::100:254"
		  ]
		],
		"floating_address": "100::100:2",
		"controller0_address": "100::100:3",
		"controller1_address": "100::100:4",
		"prefix": 64,
		"order": "random",
		"uuid": "28f8fabb-43df-4458-a256-d9195e2b6666"
	  },
	  {
		"gateway_address": null,
		"network": "100.100.100.100",
		"name": "cluster-host",
		"ranges": [
		  [
			"100.100.100.2",
			"100.100.100.55"
		  ]
		],
		"floating_address": "100.100.100.2",
		"controller0_address": "100.100.100.3",
		"controller1_address": "100.100.100.4",
		"prefix": 64,
		"order": "random",
		"uuid": "28f8fabb-43df-4458-a256-d9195e2b6667"
	  }
	]
  }
`

const NetworkListBodyFull = `
{
    "networks": [
        {
			"dynamic": false,
			"id": 1,
			"name": "admin",
			"pool_uuid": "be2eb19c-4b47-88ec-82c5-6b29097cf666",
			"type": "admin",
			"uuid": "c434c909-f2eb-4a4e-87f1-525cbe9b1ec2",
			"primary_pool_family": "ipv6"
        },
        {
			"dynamic": true,
			"id": 2,
			"name": "mgmt",
			"pool_uuid": "aa277c8e-7421-4721-ae6a-347771fe4fa6",
			"type": "mgmt",
			"uuid": "a48a7b6d-9cfa-24a4-8d48-f0e25d35984a",
			"primary_pool_family": "ipv4"
        },
		{
			"dynamic": false,
			"id": 3,
			"name": "oam",
			"pool_uuid": "384c6eb3-d48b-486e-8151-7dcecd377666",
			"type": "oam",
			"uuid": "32665423-d48b-486e-8151-7dcecd3779df",
			"primary_pool_family": "ipv6"
		},
		{
			"dynamic": true,
			"id": 4,
			"name": "pxeboot",
			"pool_uuid": "28f8fabb-43df-4458-a256-d9195e2b669e",
			"type": "pxeboot",
			"uuid": "0bebc4ef-e8e4-1248-b9d5-8694a79f58cc",
			"primary_pool_family": "ipv6"
		},
		{
			"dynamic": true,
			"id": 4,
			"name": "cluster-host",
			"pool_uuid": "28f8fabb-43df-4458-a256-d9195e2b6667",
			"type": "cluster-host",
			"uuid": "0bebc4ef-e8e4-1248-b9d5-8694a79f58ce",
			"primary_pool_family": "ipv4"
		}
    ]
}
`

const NetworkAddressPoolListBodyFull = `
{
    "network_addresspools": [
		{
			"uuid": "11111111-a6e5-425e-9317-995da88d6694",
			"network_uuid": "c434c909-f2eb-4a4e-87f1-525cbe9b1ec2",
			"address_pool_uuid": "be2eb19c-4b47-88ec-82c5-6b29097cf666",
			"network_name": "admin",
			"address_pool_name": "admin-ipv6"
		},
		{
			"uuid": "11111111-2222-425e-9317-995da88d6694",
			"network_uuid": "c434c909-f2eb-4a4e-87f1-525cbe9b1ec2",
			"address_pool_uuid": "be2eb19c-4b47-88ec-82c5-6b29097cf439",
			"network_name": "admin",
			"address_pool_name": "admin"
		},
		{
            "uuid": "22222222-2222-425e-9317-995da88d6694",
            "network_uuid": "a48a7b6d-9cfa-24a4-8d48-f0e25d35984a",
            "address_pool_uuid": "aa277c8e-7421-4721-ae6a-347771fe4fa6",
            "network_name": "mgmt",
            "address_pool_name": "management"
        },
		{
			"uuid": "33333333-a6e5-425e-9317-995da88d6694",
			"network_uuid": "32665423-d48b-486e-8151-7dcecd3779df",
			"address_pool_uuid": "384c6eb3-d48b-486e-8151-7dcecd377666",
			"network_name": "oam",
			"address_pool_name": "oam-ipv6"
		},
		{
			"uuid": "33333333-2222-425e-9317-995da88d6694",
			"network_uuid": "32665423-d48b-486e-8151-7dcecd3779df",
			"address_pool_uuid": "384c6eb3-d48b-486e-8151-7dcecd3779df",
			"network_name": "oam",
			"address_pool_name": "oam"
		},
		{
			"uuid": "44444444-a6e5-425e-9317-995da88d6694",
			"network_uuid": "0bebc4ef-e8e4-1248-b9d5-8694a79f58cc",
			"address_pool_uuid": "28f8fabb-43df-4458-a256-d9195e2b669e",
			"network_name": "pxeboot",
			"address_pool_name": "pxeboot"
		},
		{
			"uuid": "55555555-a6e5-425e-9317-995da88d6694",
			"network_uuid": "0bebc4ef-e8e4-1248-b9d5-8694a79f58ce",
			"address_pool_uuid": "28f8fabb-43df-4458-a256-d9195e2b6666",
			"network_name": "cluster-host",
			"address_pool_name": "cluster-host-ipv6"
		},
		{
			"uuid": "55555555-a6e5-425e-9317-995da88d6695",
			"network_uuid": "0bebc4ef-e8e4-1248-b9d5-8694a79f58ce",
			"address_pool_uuid": "28f8fabb-43df-4458-a256-d9195e2b6667",
			"network_name": "cluster-host",
			"address_pool_name": "cluster-host"
		}
    ]
}
`

const NetworkAddressPoolClusterHostReconcile = `
{
    "network_addresspools": [
		{
			"uuid": "55555555-a6e5-425e-9317-995da88d6694",
			"network_uuid": "0bebc4ef-e8e4-1248-b9d5-8694a79f58ce",
			"address_pool_uuid": "28f8fabb-43df-4458-a256-d9195e2b6666",
			"network_name": "cluster-host",
			"address_pool_name": "cluster-host-ipv6"
		},
		{
			"uuid": "55555555-a6e5-425e-9317-995da88d6695",
			"network_uuid": "0bebc4ef-e8e4-1248-b9d5-8694a79f58ce",
			"address_pool_uuid": "28f8fabb-43df-4458-a256-d9195e2b6667",
			"network_name": "cluster-host",
			"address_pool_name": "cluster-host"
		},
		{
			"uuid": "33333333-a6e5-425e-9317-995da88d6694",
			"network_uuid": "32665423-d48b-486e-8151-7dcecd3779df",
			"address_pool_uuid": "384c6eb3-d48b-486e-8151-7dcecd377666",
			"network_name": "oam",
			"address_pool_name": "oam-ipv6"
		},
		{
			"uuid": "33333333-2222-425e-9317-995da88d6694",
			"network_uuid": "32665423-d48b-486e-8151-7dcecd3779df",
			"address_pool_uuid": "384c6eb3-d48b-486e-8151-7dcecd3779df",
			"network_name": "oam",
			"address_pool_name": "oam"
		}
	]
}
`

const AddressPoolClusterHostReconcile = `
{
	"addrpools": [
		{
			"gateway_address": null,
			"network": "44::44",
			"name": "cluster-host-ipv6",
			"ranges": [
			  [
				"100::100:2",
				"100::100:254"
			  ]
			],
			"floating_address": "100::100:2",
			"controller0_address": "100::100:3",
			"controller1_address": "100::100:4",
			"prefix": 64,
			"order": "random",
			"uuid": "28f8fabb-43df-4458-a256-d9195e2b6666"
		  },
		  {
			"gateway_address": null,
			"network": "44.44.44.44",
			"name": "cluster-host",
			"ranges": [
			  [
				"100.100.100.2",
				"100.100.100.55"
			  ]
			],
			"floating_address": "100.100.100.2",
			"controller0_address": "100.100.100.3",
			"controller1_address": "100.100.100.4",
			"prefix": 64,
			"order": "random",
			"uuid": "28f8fabb-43df-4458-a256-d9195e2b6667"
		  }
	]
}
`

const NetworkListClusterHostReconcile = `
{
    "networks": [
		{
			"dynamic": true,
			"id": 4,
			"name": "cluster-host",
			"pool_uuid": "28f8fabb-43df-4458-a256-d9195e2b6667",
			"type": "cluster-host",
			"uuid": "0bebc4ef-e8e4-1248-b9d5-8694a79f58ce",
			"primary_pool_family": "ipv4"
		},
		{
			"dynamic": false,
			"id": 3,
			"name": "oam",
			"pool_uuid": "384c6eb3-d48b-486e-8151-7dcecd377666",
			"type": "oam",
			"uuid": "32665423-d48b-486e-8151-7dcecd3779df",
			"primary_pool_family": "ipv6"
		}
    ]
}
`

const NetworkListWithoutDualStackOAM = `
{
    "networks": [
		{
			"dynamic": true,
			"id": 4,
			"name": "cluster-host",
			"pool_uuid": "28f8fabb-43df-4458-a256-d9195e2b6667",
			"type": "cluster-host",
			"uuid": "0bebc4ef-e8e4-1248-b9d5-8694a79f58ce",
			"primary_pool_family": "ipv4"
		}
    ]
}
`

const NetworkAddrPoolListWithoutDualStackOAM = `
{
    "network_addresspools": [
		{
			"uuid": "55555555-a6e5-425e-9317-995da88d6695",
			"network_uuid": "0bebc4ef-e8e4-1248-b9d5-8694a79f58ce",
			"address_pool_uuid": "28f8fabb-43df-4458-a256-d9195e2b6667",
			"network_name": "cluster-host",
			"address_pool_name": "cluster-host"
		}
	]
}
`

const DummyNetworkAddressPoolUpdateResponse = `
{
	"uuid": "55555555-a6e5-425e-9317-995da88d6695",
	"network_uuid": "0bebc4ef-e8e4-1248-b9d5-8694a79f58ce",
	"address_pool_uuid": "28f8fabb-43df-4458-a256-d9195e2b6667",
	"network_name": "cluster-host",
	"address_pool_name": "cluster-host"
}`

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
const HostBody = `
	{
		"uuid": "d99637e9-5451-45c6-98f4-f18968e43e91",
		"hostname": "controller-0",
		"personality": "Controller-Active",
		"subfunctions": "controller,worker",
		"capabilities": {
			"Personality": "Controller-Active",
			"stor_function": "monitor"
		  },
		  "location": {
			"locn": "vbox"
		  },
		  "install_output": "text",
		  "console": "tty0",
		  "mgmt_ip": "1.2.3.4",
          "mgmt_mac": "08:08:08:08:08:08",
		  "rootfs_device": "/dev/disk/by-path/pci-0000:00:0d.0-ata-1.0",
		  "boot_device": "/dev/disk/by-path/pci-0000:00:0d.0-ata-1.0",
		  "bm_ip": null,
		  "bm_type": null,
		  "bm_username": null,
		  "administrative": "unlocked",
		  "apparmor": "disabled",
		  "hw_settle": "0",
		  "availability": "available",
		  "max_cpu_mhz_configured": "1800",
		  "inv_state": "inventoried",
		  "operational": "enabled",
		  "clock_synchronization": "ntp",
		  "max_cpu_mhz_configured": "1800"
	}
	`

const HostsListBody = `
{
  "ihosts": [   
	{
		"uuid": "d99637e9-5451-45c6-98f4-f18968e43e91",
		"hostname": "controller-0",
		"personality": "Controller-Active",
		"subfunctions": "controller,worker",
		"capabilities": {
			"Personality": "Controller-Active",
			"stor_function": "monitor"
		  },
		  "location": {
			"locn": "vbox"
		  },
		  "install_output": "text",
		  "console": "tty0",
		  "mgmt_ip": "1.2.3.4",
          "mgmt_mac": "08:08:08:08:08:08",
		  "rootfs_device": "/dev/disk/by-path/pci-0000:00:0d.0-ata-1.0",
		  "boot_device": "/dev/disk/by-path/pci-0000:00:0d.0-ata-1.0",
		  "bm_ip": null,
		  "bm_type": null,
		  "bm_username": null,
		  "administrative": "unlocked",
		  "apparmor": "disabled",
		  "hw_settle": "0",
		  "availability": "available",
		  "max_cpu_mhz_configured": "1800",
		  "inv_state": "inventoried",
		  "operational": "enabled",
		  "clock_synchronization": "ntp",
		  "max_cpu_mhz_configured": "1800"
	}  
  ]
}
`

const DummyAddressPoolUpdateResponse = `
{
		"gateway_address": null,
		"network": "192.168.206.0",
		"name": "cluster-host",
		"ranges": [
			[
				"192.168.206.2",
				"192.168.206.55"
			]
		],
		"floating_address": "192.168.206.2",
		"controller0_address": "192.168.206.3",
		"controller1_address": "192.168.206.4",
		"prefix": 64,
		"order": "random",
		"uuid": "28f8fabb-43df-4458-a256-d9195e2b6667"
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

const DataNetworkListBody = `
{
    "datanetworks": [
        {
			"uuid": "c434c909-f2eb-4a4e-87f1-525cbe9b1ec2",
			"name": "admin",
			"type": "admin"
        }
	]
	}
	`

const InterfaceNetworkListBody = `
{
    "interface_networks": [
        {
			"id": 1,
			"uuid": "c434c909-f2eb-4a4e-87f1-525cbe9b1ec2"
        }
	]
}
`

const InterfaceDataNetworkListBody = `
{
    "interface_datanetworks": [
        {
			"id": 1,
			"DataNetworkUUID": "c434c909-f2eb-4a4e-87f1-525cbe9b1ec2",
			"InterfaceUUID": "c434c909-f2eb-4a4e-87f1-525cbe9b1ec2"
        }
	]
}
`

const KernelBodyResponse = `
{
	"ikernels": [
	{
		"id": "234
	}
	]
}
`

const Label = `
{
	"ilabels": [
	{
	"ID ": 1,
	"HostUUID": "f9d5aa8b-0346-4ee3-974e-8ced77f66ae4"
	}
	]
}
`

const CPU = `
{
"icpus": [
	{
		"id": 1,
		"processor": 2
	}
]
}
`

const Memory = `
{
	"imemorys": [
		{
			"id": 1,
			"processor": 2
		}
	]
}
`

const CephMonitor = `
{
	"ceph_mon": [
	{
	"ID ": 1,
	"Hostname": "controller-0",
	"HostUUID": "f9d5aa8b-0346-4ee3-974e-8ced77f66ae4"
	}
	]
}
`

const port = `
{
	"ethernet_ports": [
	{
	"ID": 1
	}
	]
}
`

const interfaceresponse = `
{
	"iinterfaces": [
	{
	"ID": 1
	}
	]
}
`

const address = `
{
"addresses": [
   {
    "ID": 1
   }
]
}
`

const route = `
{
	"routes": [
   {
	"ID": 1
   }
   ]
}
`

const disks = `
{
	"idisks": [
   {
	"ID": 1
   }
   ]
}
`

const partition = `
{
"partitions": [
   {
	"ID": 1
   }
   ]
}
`

const volumegroup = `
{
	"ilvgs": [
   {
	"ID": 1
   }
]	
}
`

const physicalvolume = `
{
	"ipvs": [
   {
	"ID": 1
   }
	]
}
`

const osd = `
{
	"istors": [
   {
	"ID": 1
   }
   ]
}
`

const cluster = `
{
	"clusters": [
   {
	"uuid": "1",
	"name": "cluster 1"
   }
   ]
}
`

const PTPInstance = `
{
	"ptp_instances": [
   {
	"ID": 1
   }
   ]
}`

const PTPInterface = `
{
	"ptp_interfaces": [
   {
	"ID": 1
   }
   ]
}`

const filesystem = `
{
	"host_fs": [
   {
	"ID": 1,
	"Hostname": "controller-0",
	"HostUUID": "f9d5aa8b-0346-4ee3-974e-8ced77f66ae4"
   }
   ]
}`

const storage_tiers = `
{
	"storage_tiers": [
		{
			"uuid": "1",
			"cluster_uuid": "d99637e9-5451-45c6-98f4-f18968e43e91"

		}
	]
}`

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
		// Read the body of the request
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Unable to read body", http.StatusBadRequest)
		} else {
			defer r.Body.Close()
			request_opts := &networks.NetworkOpts{}
			err := json.Unmarshal(body, request_opts)
			if err != nil {
				http.Error(w, "JSON decoding error", http.StatusInternalServerError)
			} else {
				if request_opts.PoolUUID == nil {
					http.Error(w, "Sorry, cannot create network without pool_uuid", http.StatusInternalServerError)
				} else if *request_opts.Type == "other" {
					http.Error(w, "Sorry, cannot create network of type other", http.StatusInternalServerError)
				} else {
					fmt.Fprint(w, DummyNetworkUpdateResponse)
				}
			}
		}
	case http.MethodPatch:
		w.Header().Add("Content-Type", "application/json")
		fmt.Fprint(w, DummyNetworkUpdateResponse)
	case http.MethodDelete:
		w.WriteHeader(http.StatusNoContent)
	default:
		http.Error(w, `{"error": "Method not allowed"}`, http.StatusMethodNotAllowed)
	}
}
func HandleDataNetworkRequests(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		w.Header().Add("Content-Type", "application/json")
		fmt.Fprint(w, DataNetworkListBody)
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
func HandleInterfaceNetworkRequests(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		w.Header().Add("Content-Type", "application/json")
		fmt.Fprint(w, InterfaceNetworkListBody)
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
func HandleInterfaceDataNetworkRequests(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		w.Header().Add("Content-Type", "application/json")
		fmt.Fprint(w, InterfaceDataNetworkListBody)
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
	th.Mux.HandleFunc("/datanetworks", HandleDataNetworkRequests)
	th.Mux.HandleFunc("/ihosts/d99637e9-5451-45c6-98f4-f18968e43e91/interface_networks", HandleInterfaceNetworkRequests)
	th.Mux.HandleFunc("/ihosts/d99637e9-5451-45c6-98f4-f18968e43e91/interface_datanetworks", HandleInterfaceDataNetworkRequests)
	th.Mux.HandleFunc("/network_addresspools", HandleNetworkAddressPoolRequests)
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
		fmt.Fprint(w, HostBody)
	default:
		http.Error(w, `{"error": "Method not allowed"}`, http.StatusMethodNotAllowed)
	}
}
func HandleListHostRequests(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "application/json")
	switch r.Method {
	case http.MethodGet:
		fmt.Fprint(w, HostsListBodyResponse)
	default:
		http.Error(w, `{"error": "Method not allowed"}`, http.StatusMethodNotAllowed)
	}
}
func HandleListFSRequests(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "application/json")
	switch r.Method {
	case http.MethodGet:
		fmt.Fprint(w, filesystem)
	default:
		http.Error(w, `{"error": "Method not allowed"}`, http.StatusMethodNotAllowed)
	}
}
func HandleKernelRequests(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "application/json")
	switch r.Method {
	case http.MethodGet:
		fmt.Fprint(w, KernelBodyResponse)
	default:
		http.Error(w, `{"error": "Method not allowed"}`, http.StatusMethodNotAllowed)
	}
}
func HandleCpuRequests(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "application/json")
	switch r.Method {
	case http.MethodGet:
		fmt.Fprint(w, CPU)
	default:
		http.Error(w, `{"error": "Method not allowed"}`, http.StatusMethodNotAllowed)
	}
}
func HandleLabelRequests(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "application/json")
	switch r.Method {
	case http.MethodGet:
		fmt.Fprint(w, Label)
	default:
		http.Error(w, `{"error": "Method not allowed"}`, http.StatusMethodNotAllowed)
	}
}
func HandleMemoryRequests(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "application/json")
	switch r.Method {
	case http.MethodGet:
		fmt.Fprint(w, Memory)
	default:
		http.Error(w, `{"error": "Method not allowed"}`, http.StatusMethodNotAllowed)
	}
}

func HandleCephMonitorsRequests(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "application/json")
	switch r.Method {
	case http.MethodGet:
		fmt.Fprint(w, CephMonitor)
	default:
		http.Error(w, `{"error": "Method not allowed"}`, http.StatusMethodNotAllowed)
	}
}
func HandlePortRequests(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "application/json")
	switch r.Method {
	case http.MethodGet:
		fmt.Fprint(w, port)
	default:
		http.Error(w, `{"error": "Method not allowed"}`, http.StatusMethodNotAllowed)
	}
}
func HandleInterfaceRequests(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		w.Header().Add("Content-Type", "application/json")
		fmt.Fprint(w, interfaceresponse)
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
func HandleAddressRequests(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		w.Header().Add("Content-Type", "application/json")
		fmt.Fprint(w, address)
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
func HandleRouteRequests(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		w.Header().Add("Content-Type", "application/json")
		fmt.Fprint(w, route)
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
func HandleDiskRequests(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "application/json")
	switch r.Method {
	case http.MethodGet:
		fmt.Fprint(w, disks)
	default:
		http.Error(w, `{"error": "Method not allowed"}`, http.StatusMethodNotAllowed)
	}
}
func HandlePartitionRequests(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		w.Header().Add("Content-Type", "application/json")
		fmt.Fprint(w, partition)
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
func HandleVGRequests(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		w.Header().Add("Content-Type", "application/json")
		fmt.Fprint(w, volumegroup)
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
func HandlePVRequests(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		w.Header().Add("Content-Type", "application/json")
		fmt.Fprint(w, physicalvolume)
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
func HandleOSDRequests(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		w.Header().Add("Content-Type", "application/json")
		fmt.Fprint(w, osd)
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
func HandleClusterRequests(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "application/json")
	switch r.Method {
	case http.MethodGet:
		fmt.Fprint(w, cluster)
	default:
		http.Error(w, `{"error": "Method not allowed"}`, http.StatusMethodNotAllowed)
	}
}
func HandleHostPTPInstanceRequests(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		w.Header().Add("Content-Type", "application/json")
		fmt.Fprint(w, PTPInstance)
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
func HandleHostPTPInterfaceRequests(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		w.Header().Add("Content-Type", "application/json")
		fmt.Fprint(w, PTPInterface)
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
func HandleStorageTierRequests(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		w.Header().Add("Content-Type", "application/json")
		fmt.Fprint(w, storage_tiers)
	case http.MethodPost:
		w.Header().Add("Content-Type", "application/json")
		fmt.Fprint(w, DummyNetworkUpdateResponse)
	case http.MethodPatch:
		w.Header().Add("Content-Type", "application/json")
		fmt.Fprint(w, DummyNetworkUpdateResponse)
	default:
		http.Error(w, `{"error": "Method not allowed"}`, http.StatusMethodNotAllowed)
	}
}
func HandleNetworkAddressPoolRequests(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		w.Header().Add("Content-Type", "application/json")
		fmt.Fprint(w, NetworkAddressPoolListBody)
	case http.MethodPost:
		w.Header().Add("Content-Type", "application/json")
		fmt.Fprint(w, DummyNetworkAddressPoolUpdateResponse)
	case http.MethodPatch:
		w.Header().Add("Content-Type", "application/json")
		fmt.Fprint(w, DummyNetworkUpdateResponse)
	case http.MethodDelete:
		w.WriteHeader(http.StatusNoContent)
	default:
		http.Error(w, `{"error": "Method not allowed"}`, http.StatusMethodNotAllowed)
	}
}

func HostAPIS() {
	th.Mux.HandleFunc("/ihosts/d99637e9-5451-45c6-98f4-f18968e43e91", HandleHostRequests)
	th.Mux.HandleFunc("/ihosts/", HandleListHostRequests)
	th.Mux.HandleFunc("/ihosts/d99637e9-5451-45c6-98f4-f18968e43e91/ikernels", HandleKernelRequests)
	th.Mux.HandleFunc("/ihosts/d99637e9-5451-45c6-98f4-f18968e43e91/ilabels", HandleLabelRequests)
	th.Mux.HandleFunc("/ihosts/d99637e9-5451-45c6-98f4-f18968e43e91/icpus", HandleCpuRequests)
	th.Mux.HandleFunc("/ihosts/d99637e9-5451-45c6-98f4-f18968e43e91/imemorys", HandleMemoryRequests)
	th.Mux.HandleFunc("/ceph_mon", HandleCephMonitorsRequests)
	th.Mux.HandleFunc("/ihosts/d99637e9-5451-45c6-98f4-f18968e43e91/host_fs", HandleListFSRequests)

}

func OtherAPIS() {
	th.Mux.HandleFunc("/ihosts/d99637e9-5451-45c6-98f4-f18968e43e91/ethernet_ports", HandlePortRequests)
	th.Mux.HandleFunc("/ihosts/d99637e9-5451-45c6-98f4-f18968e43e91/iinterfaces", HandleInterfaceRequests)
	th.Mux.HandleFunc("/ihosts/d99637e9-5451-45c6-98f4-f18968e43e91/addresses", HandleAddressRequests)
	th.Mux.HandleFunc("/ihosts/d99637e9-5451-45c6-98f4-f18968e43e91/routes", HandleRouteRequests)
	th.Mux.HandleFunc("/ihosts/d99637e9-5451-45c6-98f4-f18968e43e91/idisks", HandleDiskRequests)
	th.Mux.HandleFunc("/ihosts/d99637e9-5451-45c6-98f4-f18968e43e91/partitions", HandlePartitionRequests)
	th.Mux.HandleFunc("/ihosts/d99637e9-5451-45c6-98f4-f18968e43e91/ilvgs", HandleVGRequests)
	th.Mux.HandleFunc("/ihosts/d99637e9-5451-45c6-98f4-f18968e43e91/ipvs", HandlePVRequests)
	th.Mux.HandleFunc("/ihosts/d99637e9-5451-45c6-98f4-f18968e43e91/istors", HandleOSDRequests)
	th.Mux.HandleFunc("/clusters", HandleClusterRequests)
	th.Mux.HandleFunc("/ihosts/d99637e9-5451-45c6-98f4-f18968e43e91/ptp_instances", HandleHostPTPInstanceRequests)
	th.Mux.HandleFunc("/ihosts/d99637e9-5451-45c6-98f4-f18968e43e91/ptp_interfaces", HandleHostPTPInterfaceRequests)
	th.Mux.HandleFunc("/clusters/1/storage_tiers", HandleStorageTierRequests)
}

func GetPlatformNetworksFromFixtures(namespace string) (map[string]*starlingxv1.PlatformNetwork, map[string][]*starlingxv1.AddressPool) {
	PlatformNetworks := make(map[string]*starlingxv1.PlatformNetwork)
	AddressPoolInstances := make(map[string][]*starlingxv1.AddressPool)
	var Networks struct {
		NetworkList []networks.Network `json:"networks"`
	}
	var NetworkAddressPools struct {
		NetworkAddressPoolList []networkAddressPools.NetworkAddressPool `json:"network_addresspools"`
	}
	var AddressPools struct {
		AddressPoolList []addresspools.AddressPool `json:"addrpools"`
	}

	_ = json.Unmarshal([]byte(NetworkListBody), &Networks)
	_ = json.Unmarshal([]byte(NetworkAddressPoolListBody), &NetworkAddressPools)
	_ = json.Unmarshal([]byte(AddrPoolListBody), &AddressPools)

	for _, network_addr_pool := range NetworkAddressPools.NetworkAddressPoolList {
		network := utils.GetSystemNetworkByName(Networks.NetworkList, network_addr_pool.NetworkName)
		addrpool := utils.GetSystemAddrPoolByName(AddressPools.AddressPoolList, network_addr_pool.AddressPoolName)

		if _, ok := PlatformNetworks[network.Type]; ok {
			PlatformNetworks[network.Type].Spec.AssociatedAddressPools = append(PlatformNetworks[network.Type].Spec.AssociatedAddressPools,
				network_addr_pool.AddressPoolName)
		} else {
			PlatformNetworks[network.Type] = &starlingxv1.PlatformNetwork{
				ObjectMeta: metav1.ObjectMeta{
					Name:      network.Name,
					Namespace: namespace,
				},
				Spec: starlingxv1.PlatformNetworkSpec{
					Type:                   network.Type,
					Dynamic:                network.Dynamic,
					AssociatedAddressPools: []string{network_addr_pool.AddressPoolName},
				},
			}

		}

		addrpool_inst, err := starlingxv1.NewAddressPool(namespace, *addrpool)
		if err == nil {
			AddressPoolInstances[network.Type] = append(AddressPoolInstances[network.Type], addrpool_inst)
		}
	}

	return PlatformNetworks, AddressPoolInstances
}

func HostControllerAPIHandlers() {
	HostsListBodyResponse = HostsListBody
	SingleSystemBodyResponse = SingleSystemBody
	AddressPoolAPIS()
	NetworkAPIS()
	OAMNetworkAPIS()
	SystemAPIS()
	HostAPIS()
	OtherAPIS()

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
