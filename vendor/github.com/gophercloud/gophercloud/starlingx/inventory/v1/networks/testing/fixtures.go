/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019 Wind River Systems, Inc. */

package testing

import (
	"fmt"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/networks"
	"github.com/gophercloud/gophercloud/testhelper/client"
	"net/http"
	"testing"

	th "github.com/gophercloud/gophercloud/testhelper"
)

var (
	NetworkHerp = networks.Network{
		UUID:      "f30e3a13-addf-4a8e-8854-43c8e477d10d",
		ID:        1,
		Name:      "Herp",
		Type:      "mgmt",
		Dynamic:   true,
		PoolUUID:  "5a74726d-5e8a-4396-8ae1-f19779fbcf4f",
		CreatedAt: "",
		UpdatedAt: nil,
	}
	NetworkDerp = networks.Network{
		UUID:      "f757b5c7-89ab-4d93-bfd7-a97780ec2c1e",
		ID:        2,
		Name:      "Derp",
		Type:      "oam",
		Dynamic:   false,
		PoolUUID:  "c7ac5a0c-606b-4fe0-9065-28a8c8fb78cc",
		CreatedAt: "",
		UpdatedAt: nil,
	}
)

const NetworkListBody = `
{
    "networks": [
        {
          "dynamic": true,
          "id": 1,
          "name": "Herp",
          "pool_uuid": "5a74726d-5e8a-4396-8ae1-f19779fbcf4f",
          "type": "mgmt",
          "uuid": "f30e3a13-addf-4a8e-8854-43c8e477d10d"
        },
        {
          "dynamic": false,
          "id": 2,
          "name": "Derp",
          "pool_uuid": "c7ac5a0c-606b-4fe0-9065-28a8c8fb78cc",
          "type": "oam",
          "uuid": "f757b5c7-89ab-4d93-bfd7-a97780ec2c1e"
        }
    ]
}
`

const SingleNetworkBody = `
{
    "dynamic": false,
    "id": 2,
    "name": "Derp",
    "pool_uuid": "c7ac5a0c-606b-4fe0-9065-28a8c8fb78cc",
    "type": "oam",
    "uuid": "f757b5c7-89ab-4d93-bfd7-a97780ec2c1e"
}
`

func HandleNetworkListSuccessfully(t *testing.T) {
	th.Mux.HandleFunc("/networks", func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "GET")
		th.TestHeader(t, r, "X-Auth-Token", client.TokenID)
		w.Header().Add("Content-Type", "application/json")
		fmt.Fprintf(w, NetworkListBody)
	})
}

func HandleNetworkGetSuccessfully(t *testing.T) {
	th.Mux.HandleFunc("/networks/f757b5c7-89ab-4d93-bfd7-a97780ec2c1e", func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "GET")
		th.TestHeader(t, r, "X-Auth-Token", client.TokenID)
		w.Header().Add("Content-Type", "application/json")
		fmt.Fprintf(w, SingleNetworkBody)
	})
}

func HandleNetworkUpdateSuccessfully(t *testing.T) {
	th.Mux.HandleFunc("/networks/f757b5c7-89ab-4d93-bfd7-a97780ec2c1e", func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "PATCH")
		th.TestHeader(t, r, "X-Auth-Token", client.TokenID)
		th.TestHeader(t, r, "Accept", "application/json")
		th.TestHeader(t, r, "Content-Type", "application/json")
		th.TestJSONRequest(t, r, `[ { "op": "replace", "path": "/name", "value": "new-name" } ]`)
		fmt.Fprintf(w, SingleNetworkBody)
	})
}

func HandleNetworkDeletionSuccessfully(t *testing.T) {
	th.Mux.HandleFunc("/networks/f757b5c7-89ab-4d93-bfd7-a97780ec2c1e", func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "DELETE")
		th.TestHeader(t, r, "X-Auth-Token", client.TokenID)

		w.WriteHeader(http.StatusNoContent)
	})
}

func HandleNetworkCreationSuccessfully(t *testing.T, response string) {
	th.Mux.HandleFunc("/networks", func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "POST")
		th.TestHeader(t, r, "X-Auth-Token", client.TokenID)
		th.TestJSONRequest(t, r, `{
          "dynamic": false,
          "name": "Derp",
          "pool_uuid": "c7ac5a0c-606b-4fe0-9065-28a8c8fb78cc",
          "type": "oam"
        }`)

		w.WriteHeader(http.StatusAccepted)
		w.Header().Add("Content-Type", "application/json")
		fmt.Fprintf(w, response)
	})
}
