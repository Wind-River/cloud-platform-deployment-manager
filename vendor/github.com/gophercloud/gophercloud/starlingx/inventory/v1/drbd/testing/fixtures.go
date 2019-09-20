/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019 Wind River Systems, Inc. */

package testing

import (
	"fmt"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/drbd"
	"github.com/gophercloud/gophercloud/testhelper/client"
	"net/http"
	"testing"

	th "github.com/gophercloud/gophercloud/testhelper"
)

var (
	DRBDHerp = drbd.DRBD{
		ID:              "90c13f86-49db-4a5a-ba6e-22016fe96223",
		LinkUtilization: 40,
		ParallelDevices: 1,
		RoundTripDelay:  0.2,
		SystemID:        "6dc8ccb9-f687-40fa-9663-c0c286e65772",
	}
	DRBDDerp = drbd.DRBD{
		ID:              "db71a663-1eb6-438d-8129-6c07b14297bf",
		LinkUtilization: 0,
		ParallelDevices: 0,
		RoundTripDelay:  0.3,
		SystemID:        "52a7c2b0-ee64-4090-9d6e-c60892128a05",
	}
	DRBDMerp = drbd.DRBD{
		ID:              "fca68026-cf50-4ab6-b1c5-59f3ebcfa3b9",
		LinkUtilization: 20,
		ParallelDevices: 2,
		RoundTripDelay:  0.1,
		SystemID:        "cf9907ae-ea10-4a97-8974-0181001e9bb6",
	}
)

const DRBDListBody = `
{
  "drbdconfigs": [
    {
      "created_at": "2019-08-07T14:40:00.110556+00:00",
      "isystem_uuid": "6dc8ccb9-f687-40fa-9663-c0c286e65772",
      "link_util": 40,
      "links": [
        {
          "href": "http://192.168.204.2:6385/v1/drbdconfigs/90c13f86-49db-4a5a-ba6e-22016fe96223",
          "rel": "self"
        },
        {
          "href": "http://192.168.204.2:6385/drbdconfigs/90c13f86-49db-4a5a-ba6e-22016fe96223",
          "rel": "bookmark"
        }
      ],
      "num_parallel": 1,
      "rtt_ms": 0.2,
      "updated_at": null,
      "uuid": "90c13f86-49db-4a5a-ba6e-22016fe96223"
    },
    {
      "created_at": "2019-08-07T14:40:00.110556+00:00",
      "isystem_uuid": "52a7c2b0-ee64-4090-9d6e-c60892128a05",
      "link_util": 0,
      "links": [
        {
          "href": "http://192.168.204.2:6385/v1/drbdconfigs/90c13f86-49db-4a5a-ba6e-22016fe96223",
          "rel": "self"
        },
        {
          "href": "http://192.168.204.2:6385/drbdconfigs/90c13f86-49db-4a5a-ba6e-22016fe96223",
          "rel": "bookmark"
        }
      ],
      "num_parallel": 0,
      "rtt_ms": 0.3,
      "updated_at": null,
      "uuid": "db71a663-1eb6-438d-8129-6c07b14297bf"
    },
    {
      "created_at": "2019-08-07T14:40:00.110556+00:00",
      "isystem_uuid": "cf9907ae-ea10-4a97-8974-0181001e9bb6",
      "link_util": 20,
      "links": [
        {
          "href": "http://192.168.204.2:6385/v1/drbdconfigs/90c13f86-49db-4a5a-ba6e-22016fe96223",
          "rel": "self"
        },
        {
          "href": "http://192.168.204.2:6385/drbdconfigs/90c13f86-49db-4a5a-ba6e-22016fe96223",
          "rel": "bookmark"
        }
      ],
      "num_parallel": 2,
      "rtt_ms": 0.1,
      "updated_at": null,
      "uuid": "fca68026-cf50-4ab6-b1c5-59f3ebcfa3b9"
    }
  ]
}
`

const DRBDSingleBody = `
{
      "created_at": "2019-08-07T14:40:00.110556+00:00",
      "isystem_uuid": "52a7c2b0-ee64-4090-9d6e-c60892128a05",
      "link_util": 0,
      "links": [
        {
          "href": "http://192.168.204.2:6385/v1/drbdconfigs/90c13f86-49db-4a5a-ba6e-22016fe96223",
          "rel": "self"
        },
        {
          "href": "http://192.168.204.2:6385/drbdconfigs/90c13f86-49db-4a5a-ba6e-22016fe96223",
          "rel": "bookmark"
        }
      ],
      "num_parallel": 0,
      "rtt_ms": 0.3,
      "updated_at": null,
      "uuid": "db71a663-1eb6-438d-8129-6c07b14297bf"
}
`

func HandleDRBDListSuccessfully(t *testing.T) {
	th.Mux.HandleFunc("/drbdconfig", func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "GET")
		th.TestHeader(t, r, "X-Auth-Token", client.TokenID)
		w.Header().Add("Content-Type", "application/json")
		fmt.Fprintf(w, DRBDListBody)
	})
}

func HandleDRBDGetSuccessfully(t *testing.T) {
	th.Mux.HandleFunc("/drbdconfig/90c13f86-49db-4a5a-ba6e-22016fe96223", func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "GET")
		th.TestHeader(t, r, "X-Auth-Token", client.TokenID)
		w.Header().Add("Content-Type", "application/json")
		fmt.Fprintf(w, DRBDSingleBody)
	})
}

func HandleDRBDUpdateSuccessfully(t *testing.T) {
	th.Mux.HandleFunc("/drbdconfig/90c13f86-49db-4a5a-ba6e-22016fe96223", func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "PATCH")
		th.TestHeader(t, r, "X-Auth-Token", client.TokenID)
		th.TestHeader(t, r, "Accept", "application/json")
		th.TestHeader(t, r, "Content-Type", "application/json")
		th.TestJSONRequest(t, r, `[ { "op": "replace", "path": "/link_util", "value": 60 } ]`)
		fmt.Fprintf(w, DRBDSingleBody)
	})
}
