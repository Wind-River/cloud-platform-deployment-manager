/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019 Wind River Systems, Inc. */

package testing

import (
	"fmt"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/interfaces"
	"github.com/gophercloud/gophercloud/testhelper/client"
	"net/http"
	"testing"

	th "github.com/gophercloud/gophercloud/testhelper"
)

var (
	IPv4Modes     = [3]string{interfaces.AddressModeStatic, interfaces.AddressModePool, interfaces.AddressModeDisabled}
	IPv4Pool      = "dbfa4c2e-4526-4aaf-b07b-a3da7aeb6c26"
	IPv6Modes     = [3]string{interfaces.AddressModeStatic, interfaces.AddressModePool, interfaces.AddressModeDisabled}
	IPv6Pool      = "934d8341-5114-46d2-9560-7c47618892c7"
	VFCounts      = [3]int{0, 1, 2}
	InterfaceHerp = interfaces.Interface{
		ID:             "7499f727-e19b-4e9b-a571-5919bad20dc3",
		Name:           "Herp",
		Type:           interfaces.IFTypeEthernet,
		Class:          interfaces.IFClassPlatform,
		NetworkType:    "",
		MTU:            1500,
		VID:            nil,
		IPv4Mode:       &IPv4Modes[0],
		IPv4Pool:       nil,
		IPv6Mode:       &IPv6Modes[0],
		IPv6Pool:       nil,
		Networks:       nil,
		DataNetworks:   nil,
		AEMode:         nil,
		AETransmitHash: nil,
		VFCount:        &VFCounts[0],
		Uses:           []string{"data0"},
		Users:          []string{},
	}
	InterfaceDerp = interfaces.Interface{
		ID:             "a5965fee-dc60-40dc-a234-edf87f1f9380",
		Name:           "Derp",
		Type:           interfaces.IFTypeVirtual,
		Class:          interfaces.IFClassData,
		NetworkType:    "",
		MTU:            1400,
		VID:            nil,
		IPv4Mode:       &IPv4Modes[1],
		IPv4Pool:       &IPv4Pool,
		IPv6Mode:       &IPv6Modes[1],
		IPv6Pool:       &IPv6Pool,
		Networks:       nil,
		DataNetworks:   nil,
		AEMode:         nil,
		AETransmitHash: nil,
		VFCount:        &VFCounts[1],
		Uses:           []string{},
		Users:          []string{},
	}
	InterfaceMerp = interfaces.Interface{
		ID:             "336132b4-27a1-4b1a-bb05-7f17fe290a66",
		Name:           "Merp",
		Type:           interfaces.IFTypeVLAN,
		Class:          interfaces.IFClassPCISRIOV,
		NetworkType:    "",
		MTU:            1500,
		VID:            nil,
		IPv4Mode:       &IPv4Modes[2],
		IPv4Pool:       nil,
		IPv6Mode:       &IPv6Modes[2],
		IPv6Pool:       nil,
		Networks:       nil,
		DataNetworks:   nil,
		AEMode:         nil,
		AETransmitHash: nil,
		VFCount:        &VFCounts[2],
		Uses:           []string{},
		Users:          []string{},
	}
)

const InterfaceListBody = `
{
  "iinterfaces": [
    {
      "aemode": null,
      "forihostid": 2,
      "ifclass": "platform",
      "ifname": "Herp",
      "iftype": "ethernet",
      "ihost_uuid": "f73dda8e-be3c-4704-ad1e-ed99e44b846e",
      "imac": "08:00:27:25:6a:20",
      "imtu": 1500,
      "ipv4_mode": "static",
      "ipv6_mode": "static",
      "links": [
        {
          "href": "http://192.168.204.2:6385/v1/iinterfaces/67f0631b-4616-4308-af36-820633f6a70e",
          "rel": "self"
        },
        {
          "href": "http://192.168.204.2:6385/iinterfaces/67f0631b-4616-4308-af36-820633f6a70e",
          "rel": "bookmark"
        }
      ],
      "schedpolicy": null,
      "sriov_numvfs": 0,
      "sriov_vf_driver": null,
      "txhashpolicy": null,
      "used_by": [],
      "uses": [
        "data0"
      ],
      "uuid": "7499f727-e19b-4e9b-a571-5919bad20dc3",
      "vlan_id": null
    },
    {
      "aemode": null,
      "forihostid": 2,
      "ifclass": "data",
      "ifname": "Derp",
      "iftype": "virtual",
      "ihost_uuid": "f73dda8e-be3c-4704-ad1e-ed99e44b846e",
      "imac": "08:00:27:af:1f:fe",
      "imtu": 1400,
      "ipv4_mode": "pool",
      "ipv4_pool": "dbfa4c2e-4526-4aaf-b07b-a3da7aeb6c26",
      "ipv6_mode": "pool",
      "ipv6_pool": "934d8341-5114-46d2-9560-7c47618892c7",
      "links": [
        {
          "href": "http://192.168.204.2:6385/v1/iinterfaces/94c19af1-c912-410e-bcdf-f8d54e666a7b",
          "rel": "self"
        },
        {
          "href": "http://192.168.204.2:6385/iinterfaces/94c19af1-c912-410e-bcdf-f8d54e666a7b",
          "rel": "bookmark"
        }
      ],
      "schedpolicy": null,
      "sriov_numvfs": 1,
      "sriov_vf_driver": null,
      "txhashpolicy": null,
      "used_by": [],
      "uses": [],
      "uuid": "a5965fee-dc60-40dc-a234-edf87f1f9380",
      "vlan_id": null
    },
    {
      "aemode": null,
      "forihostid": 2,
      "ifclass": "pci-sriov",
      "ifname": "Merp",
      "iftype": "vlan",
      "ihost_uuid": "f73dda8e-be3c-4704-ad1e-ed99e44b846e",
      "imac": "08:00:27:d8:21:f7",
      "imtu": 1500,
      "ipv4_mode": "disabled",
      "ipv6_mode": "disabled",
      "links": [
        {
          "href": "http://192.168.204.2:6385/v1/iinterfaces/b383f735-929f-48fd-b717-62399523cc63",
          "rel": "self"
        },
        {
          "href": "http://192.168.204.2:6385/iinterfaces/b383f735-929f-48fd-b717-62399523cc63",
          "rel": "bookmark"
        }
      ],
      "schedpolicy": null,
      "sriov_numvfs": 2,
      "sriov_vf_driver": null,
      "txhashpolicy": null,
      "used_by": [],
      "uses": [],
      "uuid": "336132b4-27a1-4b1a-bb05-7f17fe290a66",
      "vlan_id": null
    }
  ]
}
`

const InterfaceSingleBody = `
{
  "aemode": null,
  "created_at": "2019-08-08T15:30:34.495808+00:00",
  "forihostid": 2,
  "ifcapabilities": {},
  "ifclass": "data",
  "ifname": "Derp",
  "iftype": "virtual",
  "ihost_uuid": "f73dda8e-be3c-4704-ad1e-ed99e44b846e",
  "imac": "08:00:27:25:6a:20",
  "imtu": 1400,
  "ipv4_mode": "pool",
  "ipv4_pool": "dbfa4c2e-4526-4aaf-b07b-a3da7aeb6c26",
  "ipv6_mode": "pool",
  "ipv6_pool": "934d8341-5114-46d2-9560-7c47618892c7",
  "links": [
    {
      "href": "http://192.168.204.2:6385/v1/iinterfaces/67f0631b-4616-4308-af36-820633f6a70e",
      "rel": "self"
    },
    {
      "href": "http://192.168.204.2:6385/iinterfaces/67f0631b-4616-4308-af36-820633f6a70e",
      "rel": "bookmark"
    }
  ],
  "ports": [
    {
      "href": "http://192.168.204.2:6385/v1/iinterfaces/67f0631b-4616-4308-af36-820633f6a70e/ports",
      "rel": "self"
    },
    {
      "href": "http://192.168.204.2:6385/iinterfaces/67f0631b-4616-4308-af36-820633f6a70e/ports",
      "rel": "bookmark"
    }
  ],
  "schedpolicy": null,
  "sriov_numvfs": 1,
  "sriov_vf_driver": null,
  "txhashpolicy": null,
  "updated_at": "2019-08-08T15:30:59.199077+00:00",
  "used_by": [],
  "uses": [],
  "uuid": "a5965fee-dc60-40dc-a234-edf87f1f9380",
  "vlan_id": null
}
`

func HandleInterfaceListSuccessfully(t *testing.T) {
	th.Mux.HandleFunc("/ihosts/f73dda8e-be3c-4704-ad1e-ed99e44b846e/iinterfaces", func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "GET")
		th.TestHeader(t, r, "X-Auth-Token", client.TokenID)
		w.Header().Add("Content-Type", "application/json")
		fmt.Fprintf(w, InterfaceListBody)
	})
}

func HandleInterfaceGetSuccessfully(t *testing.T) {
	th.Mux.HandleFunc("/iinterfaces/a5965fee-dc60-40dc-a234-edf87f1f9380", func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "GET")
		th.TestHeader(t, r, "X-Auth-Token", client.TokenID)
		w.Header().Add("Content-Type", "application/json")
		fmt.Fprintf(w, InterfaceSingleBody)
	})
}

func HandleInterfaceUpdateSuccessfully(t *testing.T) {
	th.Mux.HandleFunc("/iinterfaces/a5965fee-dc60-40dc-a234-edf87f1f9380", func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "PATCH")
		th.TestHeader(t, r, "X-Auth-Token", client.TokenID)
		th.TestHeader(t, r, "Accept", "application/json")
		th.TestHeader(t, r, "Content-Type", "application/json")
		th.TestJSONRequest(t, r, `[ { "op": "replace", "path": "/ifname", "value": "new-name" } ]`)
		fmt.Fprintf(w, InterfaceSingleBody)
	})
}

func HandleInterfaceDeletionSuccessfully(t *testing.T) {
	th.Mux.HandleFunc("/iinterfaces/a5965fee-dc60-40dc-a234-edf87f1f9380", func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "DELETE")
		th.TestHeader(t, r, "X-Auth-Token", client.TokenID)

		w.WriteHeader(http.StatusNoContent)
	})
}

func HandleInterfaceCreationSuccessfully(t *testing.T, response string) {
	th.Mux.HandleFunc("/iinterfaces", func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "POST")
		th.TestHeader(t, r, "X-Auth-Token", client.TokenID)
		th.TestJSONRequest(t, r, `{
          "ifclass": "data",
          "ifname": "Derp",
          "iftype": "virtual",
          "ihost_uuid": "f73dda8e-be3c-4704-ad1e-ed99e44b846e",
          "imtu": 1400,
          "ipv4_mode": "pool",
          "ipv4_pool": "dbfa4c2e-4526-4aaf-b07b-a3da7aeb6c26",
          "ipv6_mode": "pool",
          "ipv6_pool": "934d8341-5114-46d2-9560-7c47618892c7",
          "sriov_numvfs": 1,
          "uses": [],
          "usesmodify": []
        }`)

		w.WriteHeader(http.StatusAccepted)
		w.Header().Add("Content-Type", "application/json")
		fmt.Fprintf(w, response)
	})
}
