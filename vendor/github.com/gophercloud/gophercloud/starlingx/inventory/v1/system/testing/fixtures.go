/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019 Wind River Systems, Inc. */

package testing

import (
	"fmt"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/system"
	th "github.com/gophercloud/gophercloud/testhelper"
	"github.com/gophercloud/gophercloud/testhelper/client"
	"net/http"
	"testing"
)

var (
	updatedString        = "2019-08-07T14:45:50.822509+00:00"
	sharedServicesString = "[]"
	SystemHerp           = system.System{
		ID:              "6dc8ccb9-f687-40fa-9663-c0c286e65772",
		Name:            "Herp",
		Description:     "Test System",
		Location:        "vbox",
		Contact:         "info@windriver.com",
		SystemMode:      "duplex",
		SystemType:      "All-in-one",
		SoftwareVersion: "19.01",
		RegionName:      "RegionOne",
		Capabilities: system.Capabilities{
			SDNEnabled:     false,
			SharedServices: &sharedServicesString,
			BMRegion:       "External",
			VSwitchType:    "none",
			HTTPSEnabled:   false,
			RegionConfig:   false,
		},
		CreatedAt: "2019-08-07T14:32:41.617713+00:00",
		UpdatedAt: &updatedString,
	}

	SystemDerp = system.System{
		ID:              "52a7c2b0-ee64-4090-9d6e-c60892128a05",
		Name:            "Derp",
		Description:     "Test System",
		Location:        "Ottawa",
		Contact:         "info@windriver.com",
		SystemMode:      "simplex",
		SystemType:      "All-in-one",
		SoftwareVersion: "19.01",
		RegionName:      "RegionTwo",
		Capabilities: system.Capabilities{
			SDNEnabled:     true,
			SharedServices: &sharedServicesString,
			BMRegion:       "External",
			VSwitchType:    "none",
			HTTPSEnabled:   true,
			RegionConfig:   true,
		},
		CreatedAt: "2019-08-07T14:32:41.617713+00:00",
		UpdatedAt: nil,
	}

	SystemMerp = system.System{
		ID:          "cf9907ae-ea10-4a97-8974-0181001e9bb6",
		Name:        "Merp",
		Description: "Test System",
		Location:    "The Ice Wall",
		Contact:     "info@windriver.com",
		// SystemMode: "duplex",
		// SystemType: "All-in-one",
		SoftwareVersion: "19.01",
		RegionName:      "RegionThree",
		Capabilities: system.Capabilities{
			SDNEnabled: true,
			// SharedServices: "[]",
			BMRegion: "Internal",
			// VSwitchType: "none",
			HTTPSEnabled: true,
			RegionConfig: false,
		},
		CreatedAt: "2019-08-07T14:45:50.822509+00:00",
		UpdatedAt: nil,
	}
)

const SystemListBody = `
{
	"isystems": [
		{
			"system_mode": "duplex",
			"created_at": "2019-08-07T14:32:41.617713+00:00",
			"links": [
				{
					"href": "http://192.168.204.2:6385/v1/isystems/6dc8ccb9-f687-40fa-9663-c0c286e65772",
					"rel": "self"
				},
				{
					"href": "http://192.168.204.2:6385/isystems/6dc8ccb9-f687-40fa-9663-c0c286e65772",
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
			"uuid": "6dc8ccb9-f687-40fa-9663-c0c286e65772"
		},
        {
			"system_mode": "simplex",
			"created_at": "2019-08-07T14:32:41.617713+00:00",
			"links": [
				{
					"href": "http://192.168.204.2:6385/v1/isystems/6dc8ccb9-f687-40fa-9663-c0c286e65772",
					"rel": "self"
				},
				{
					"href": "http://192.168.204.2:6385/isystems/6dc8ccb9-f687-40fa-9663-c0c286e65772",
					"rel": "bookmark"
				}
			],
			"security_feature": "spectre_meltdown_v1",
			"description": "Test System",
			"software_version": "19.01",
			"service_project_name": "services",
			"updated_at": null,
			"distributed_cloud_role": null,
			"location": "Ottawa",
			"capabilities": {
				"sdn_enabled": true,
				"shared_services": "[]",
				"bm_region": "External",
				"vswitch_type": "none",
				"https_enabled": true,
				"region_config": true
			},
			"name": "Derp",
			"contact": "info@windriver.com",
			"system_type": "All-in-one",
			"timezone": "UTC",
			"region_name": "RegionTwo",
			"uuid": "52a7c2b0-ee64-4090-9d6e-c60892128a05"
        },
        {
			"system_mode": "simplex",
			"created_at": "2019-08-07T14:32:41.617713+00:00",
			"links": [
				{
					"href": "http://192.168.204.2:6385/v1/isystems/6dc8ccb9-f687-40fa-9663-c0c286e65772",
					"rel": "self"
				},
				{
					"href": "http://192.168.204.2:6385/isystems/6dc8ccb9-f687-40fa-9663-c0c286e65772",
					"rel": "bookmark"
				}
			],
			"security_feature": "spectre_meltdown_v1",
			"description": "Test System",
			"software_version": "19.01",
			"service_project_name": "services",
			"updated_at": "",
			"distributed_cloud_role": null,
			"location": "Ottawa",
			"capabilities": {
				"sdn_enabled": false,
				"shared_services": "[]",
				"bm_region": "Internal",
				"vswitch_type": "none",
				"https_enabled": false,
				"region_config": false
			},
			"name": "Derp",
			"contact": "info@windriver.com",
			"system_type": "All-in-one",
			"timezone": "UTC",
			"region_name": "RegionTwo",
			"uuid": "52a7c2b0-ee64-4090-9d6e-c60892128a05"
        }
	]
}
`

const SingleSystemBody = `
{
	"isystems": [
		{
			"system_mode": "duplex",
			"created_at": "2019-08-07T14:32:41.617713+00:00",
			"links": [
				{
					"href": "http://192.168.204.2:6385/v1/isystems/6dc8ccb9-f687-40fa-9663-c0c286e65772",
					"rel": "self"
				},
				{
					"href": "http://192.168.204.2:6385/isystems/6dc8ccb9-f687-40fa-9663-c0c286e65772",
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
			"uuid": "6dc8ccb9-f687-40fa-9663-c0c286e65772"
		}
    ]
}
`

const PatchSystemBody = `
{
	"system_mode": "simplex",
	"created_at": "2019-08-07T14:32:41.617713+00:00",
	"links": [
		{
			"href": "http://192.168.204.2:6385/v1/isystems/6dc8ccb9-f687-40fa-9663-c0c286e65772",
			"rel": "self"
		},
		{
			"href": "http://192.168.204.2:6385/isystems/6dc8ccb9-f687-40fa-9663-c0c286e65772",
			"rel": "bookmark"
		}
	],
	"security_feature": "spectre_meltdown_v1",
	"description": "Test System",
	"software_version": "19.01",
	"service_project_name": "services",
	"updated_at": null,
	"distributed_cloud_role": null,
	"location": "Ottawa",
	"capabilities": {
		"sdn_enabled": true,
		"shared_services": "[]",
		"bm_region": "External",
		"vswitch_type": "none",
		"https_enabled": true,
		"region_config": true
	},
	"name": "Derp",
	"contact": "info@windriver.com",
	"system_type": "All-in-one",
	"timezone": "UTC",
	"region_name": "RegionTwo",
	"uuid": "52a7c2b0-ee64-4090-9d6e-c60892128a05"
}
`

func HandleSystemListSuccessfully(t *testing.T) {
	th.Mux.HandleFunc("/isystems", func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "GET")
		th.TestHeader(t, r, "X-Auth-Token", client.TokenID)
		w.Header().Add("Content-Type", "application/json")
		fmt.Fprintf(w, SystemListBody)
	})
}

func HandleSystemUpdateSuccessfully(t *testing.T) {
	th.Mux.HandleFunc("/isystems/6dc8ccb9-f687-40fa-9663-c0c286e65772", func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "PATCH")
		th.TestHeader(t, r, "X-Auth-Token", client.TokenID)
		th.TestHeader(t, r, "Accept", "application/json")
		th.TestHeader(t, r, "Content-Type", "application/json")
		th.TestJSONRequest(t, r, `[ { "op": "replace", "path": "/name", "value": "new-name" } ]`)

		fmt.Fprintf(w, PatchSystemBody)
	})
}

func HandleSystemGetSuccessfully(t *testing.T) {
	th.Mux.HandleFunc("/isystems/6dc8ccb9-f687-40fa-9663-c0c286e65772", func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "GET")
		th.TestHeader(t, r, "X-Auth-Token", client.TokenID)
		th.TestHeader(t, r, "Accept", "application/json")

		fmt.Fprintf(w, PatchSystemBody)
	})
}

func HandleSystemGetDefaultSuccesfully(t *testing.T) {
	th.Mux.HandleFunc("/isystems", func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "GET")
		th.TestHeader(t, r, "X-Auth-Token", client.TokenID)
		w.Header().Add("Content-Type", "application/json")
		fmt.Fprintf(w, SingleSystemBody)
	})
}
