/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019 Wind River Systems, Inc. */

package testing

import (
	"fmt"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/addresses"
	th "github.com/gophercloud/gophercloud/testhelper"
	"github.com/gophercloud/gophercloud/testhelper/client"
	"net/http"
	"testing"
)

var (
	poolUUID    = "5a74726d-5e8a-4396-8ae1-f19779fbcf4f"
	AddressHerp = addresses.Address{
		ID:            "2441bf93-6109-4635-ad04-4674ee523d86",
		Address:       "1.2.3.4",
		Prefix:        28,
		InterfaceName: "lo",
		InterfaceUUID: "62b00dc7-0549-4418-84a4-117c1f74b8d4",
		EnableDAD:     false,
		PoolUUID:      &poolUUID,
	}
	AddressDerp = addresses.Address{
		ID:            "37fe7dd1-51dd-4579-8904-1fdc71f4dbd1",
		Address:       "4.3.2.1",
		Prefix:        24,
		InterfaceName: "eth0",
		InterfaceUUID: "62b00dc7-0549-4418-84a4-117c1f74b8d4",
		EnableDAD:     true,
		PoolUUID:      nil,
	}
)

const AddressListBody = `
{
  "addresses": [
    {
      "address": "1.2.3.4",
      "enable_dad": false,
      "forihostid": 1,
      "ifname": "lo",
      "interface_uuid": "62b00dc7-0549-4418-84a4-117c1f74b8d4",
      "pool_uuid": "5a74726d-5e8a-4396-8ae1-f19779fbcf4f",
      "prefix": 28,
      "uuid": "2441bf93-6109-4635-ad04-4674ee523d86"
    },
    {
      "address": "4.3.2.1",
      "enable_dad": true,
      "forihostid": 1,
      "ifname": "eth0",
      "interface_uuid": "62b00dc7-0549-4418-84a4-117c1f74b8d4",
      "pool_uuid": null,
      "prefix": 24,
      "uuid": "37fe7dd1-51dd-4579-8904-1fdc71f4dbd1"
    }
  ]
}
`

const AddressSingleBody = `
{
    "address": "4.3.2.1",
    "enable_dad": true,
    "forihostid": 1,
    "ifname": "eth0",
    "interface_uuid": "62b00dc7-0549-4418-84a4-117c1f74b8d4",
    "pool_uuid": null,
    "prefix": 24,
    "uuid": "37fe7dd1-51dd-4579-8904-1fdc71f4dbd1"
}
`

func HandleAddressListSuccessfully(t *testing.T) {
	th.Mux.HandleFunc("/ihosts/f757b5c7-89ab-4d93-bfd7-a97780ec2c1e/addresses", func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "GET")
		th.TestHeader(t, r, "X-Auth-Token", client.TokenID)
		w.Header().Add("Content-Type", "application/json")
		fmt.Fprintf(w, AddressListBody)
	})
}

func HandleAddressGetSuccessfully(t *testing.T) {
	th.Mux.HandleFunc("/addresses/37fe7dd1-51dd-4579-8904-1fdc71f4dbd1", func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "GET")
		th.TestHeader(t, r, "X-Auth-Token", client.TokenID)
		th.TestHeader(t, r, "Accept", "application/json")

		// fmt.Fprintf(w, PatchAddressBody)
		fmt.Fprintf(w, AddressSingleBody)
	})
}

func HandleAddressDeletionSuccessfully(t *testing.T) {
	th.Mux.HandleFunc("/addresses/37fe7dd1-51dd-4579-8904-1fdc71f4dbd1", func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "DELETE")
		th.TestHeader(t, r, "X-Auth-Token", client.TokenID)

		w.WriteHeader(http.StatusNoContent)
	})
}

func HandleAddressCreationSuccessfully(t *testing.T, response string) {
	th.Mux.HandleFunc("/addresses", func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "POST")
		th.TestHeader(t, r, "X-Auth-Token", client.TokenID)
		th.TestJSONRequest(t, r, `{
          "address": "4.3.2.1",
          "interface_uuid": "62b00dc7-0549-4418-84a4-117c1f74b8d4",
          "prefix": 24
        }`)

		w.WriteHeader(http.StatusAccepted)
		w.Header().Add("Content-Type", "application/json")
		fmt.Fprintf(w, response)
	})
}
