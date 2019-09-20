/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019 Wind River Systems, Inc. */

package testing

import (
	"fmt"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/ptp"
	th "github.com/gophercloud/gophercloud/testhelper"
	"github.com/gophercloud/gophercloud/testhelper/client"
	"net/http"
	"testing"
)

var (
	updatedAt = "2019-08-10T14:40:00.136864+00:00"
	PTPHerp   = ptp.PTP{
		ID:        "3d9e9e37-117f-4e04-8141-2e467a6dd3ea",
		Mode:      "hardware",
		Transport: "l2",
		Mechanism: "e2e",
		SystemID:  "5af5f7e5-1eea-4e76-b539-ac552e132e47",
		CreatedAt: "2019-08-07T14:40:00.136864+00:00",
		UpdatedAt: &updatedAt,
	}
	PTPDerp = ptp.PTP{
		ID:        "d87feed9-e351-40fc-8356-7bf6a59750ea",
		Mode:      "software",
		Transport: "l1",
		Mechanism: "e2e",
		SystemID:  "8f4b1965-fede-4cbd-baca-7e7a8b00c529",
		CreatedAt: "2019-08-07T14:40:00.136864+00:00",
		UpdatedAt: nil,
	}
)

const PTPListBody = `
{
  "ptps": [
    {
      "created_at": "2019-08-07T14:40:00.136864+00:00",
      "isystem_uuid": "5af5f7e5-1eea-4e76-b539-ac552e132e47",
      "links": [
        {
          "href": "http://192.168.204.2:6385/v1/ptps/3d9e9e37-117f-4e04-8141-2e467a6dd3ea",
          "rel": "self"
        },
        {
          "href": "http://192.168.204.2:6385/ptps/3d9e9e37-117f-4e04-8141-2e467a6dd3ea",
          "rel": "bookmark"
        }
      ],
      "mechanism": "e2e",
      "mode": "hardware",
      "transport": "l2",
      "updated_at": "2019-08-10T14:40:00.136864+00:00",
      "uuid": "3d9e9e37-117f-4e04-8141-2e467a6dd3ea"
    },
    {
      "created_at": "2019-08-07T14:40:00.136864+00:00",
      "isystem_uuid": "8f4b1965-fede-4cbd-baca-7e7a8b00c529",
      "links": [
        {
          "href": "http://192.168.204.2:6385/v1/ptps/3d9e9e37-117f-4e04-8141-2e467a6dd3ea",
          "rel": "self"
        },
        {
          "href": "http://192.168.204.2:6385/ptps/3d9e9e37-117f-4e04-8141-2e467a6dd3ea",
          "rel": "bookmark"
        }
      ],
      "mechanism": "e2e",
      "mode": "software",
      "transport": "l1",
      "updated_at": null,
      "uuid": "d87feed9-e351-40fc-8356-7bf6a59750ea"
    }
  ]
}
`

const PTPSingleBody = `
{
    "created_at": "2019-08-07T14:40:00.136864+00:00",
    "isystem_uuid": "8f4b1965-fede-4cbd-baca-7e7a8b00c529",
    "links": [
        {
          "href": "http://192.168.204.2:6385/v1/ptps/3d9e9e37-117f-4e04-8141-2e467a6dd3ea",
          "rel": "self"
        },
        {
          "href": "http://192.168.204.2:6385/ptps/3d9e9e37-117f-4e04-8141-2e467a6dd3ea",
          "rel": "bookmark"
        }
      ],
    "mechanism": "e2e",
    "mode": "software",
    "transport": "l1",
    "updated_at": null,
    "uuid": "d87feed9-e351-40fc-8356-7bf6a59750ea"
}
`

func HandlePTPListSuccessfully(t *testing.T) {
	th.Mux.HandleFunc("/ptp", func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "GET")
		th.TestHeader(t, r, "X-Auth-Token", client.TokenID)
		w.Header().Add("Content-Type", "application/json")
		fmt.Fprintf(w, PTPListBody)
	})
}

func HandlePTPUpdateSuccessfully(t *testing.T) {
	th.Mux.HandleFunc("/ptp/d87feed9-e351-40fc-8356-7bf6a59750ea", func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "PATCH")
		th.TestHeader(t, r, "X-Auth-Token", client.TokenID)
		th.TestHeader(t, r, "Accept", "application/json")
		th.TestHeader(t, r, "Content-Type", "application/json")
		th.TestJSONRequest(t, r, `[ { "op": "replace", "path": "/transport", "value": "l2" } ]`)

		fmt.Fprintf(w, PTPSingleBody)
	})
}

func HandlePTPGetSuccessfully(t *testing.T) {
	th.Mux.HandleFunc("/ptp/d87feed9-e351-40fc-8356-7bf6a59750ea", func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "GET")
		th.TestHeader(t, r, "X-Auth-Token", client.TokenID)
		th.TestHeader(t, r, "Accept", "application/json")

		fmt.Fprintf(w, PTPSingleBody)
	})
}
