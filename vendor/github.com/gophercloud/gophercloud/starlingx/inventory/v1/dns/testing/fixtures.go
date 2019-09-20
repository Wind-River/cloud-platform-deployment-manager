/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019 Wind River Systems, Inc. */

package testing

import (
	"fmt"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/dns"
	th "github.com/gophercloud/gophercloud/testhelper"
	"github.com/gophercloud/gophercloud/testhelper/client"
	"net/http"
	"testing"
)

var (
	updatedAt = "2019-08-07T14:42:24.885906+00:00"
	DNSHerp   = dns.DNS{
		ID:          "4a667701-fcca-448e-9cdb-c02df956920d",
		Nameservers: "128.224.144.130,8.8.8.8",
		SystemID:    "5af5f7e5-1eea-4e76-b539-ac552e132e47",
		CreatedAt:   "2019-08-07T14:40:00.082258+00:00",
		UpdatedAt:   &updatedAt,
	}
	DNSDerp = dns.DNS{
		ID:          "e60b7d12-7585-486e-9c27-3d16e0daba09",
		Nameservers: "8.8.8.8",
		SystemID:    "0cf998b0-3ab8-4661-8103-1b9f13fe10bc",
		CreatedAt:   "2019-08-07T14:40:00.082258+00:00",
		UpdatedAt:   nil,
	}
)

const DNSListBody = `
{
    "idnss": [
        {
          "created_at": "2019-08-07T14:40:00.082258+00:00",
          "isystem_uuid": "5af5f7e5-1eea-4e76-b539-ac552e132e47",
          "links": [
            {
              "href": "http://192.168.204.2:6385/v1/idnss/4a667701-fcca-448e-9cdb-c02df956920d",
              "rel": "self"
            },
            {
              "href": "http://192.168.204.2:6385/idnss/4a667701-fcca-448e-9cdb-c02df956920d",
              "rel": "bookmark"
            }
          ],
          "nameservers": "128.224.144.130,8.8.8.8",
          "updated_at": "2019-08-07T14:42:24.885906+00:00",
          "uuid": "4a667701-fcca-448e-9cdb-c02df956920d"
        },
        {
          "created_at": "2019-08-07T14:40:00.082258+00:00",
          "isystem_uuid": "0cf998b0-3ab8-4661-8103-1b9f13fe10bc",
          "links": [
            {
              "href": "http://192.168.204.2:6385/v1/idnss/4a667701-fcca-448e-9cdb-c02df956920d",
              "rel": "self"
            },
            {
              "href": "http://192.168.204.2:6385/idnss/4a667701-fcca-448e-9cdb-c02df956920d",
              "rel": "bookmark"
            }
          ],
          "nameservers": "8.8.8.8",
          "updated_at": null,
          "uuid": "e60b7d12-7585-486e-9c27-3d16e0daba09"
        }
    ]
}
`

const SingleDNSBody = `
{
      "created_at": "2019-08-07T14:40:00.082258+00:00",
      "isystem_uuid": "0cf998b0-3ab8-4661-8103-1b9f13fe10bc",
      "links": [
        {
          "href": "http://192.168.204.2:6385/v1/idnss/4a667701-fcca-448e-9cdb-c02df956920d",
          "rel": "self"
        },
        {
          "href": "http://192.168.204.2:6385/idnss/4a667701-fcca-448e-9cdb-c02df956920d",
          "rel": "bookmark"
        }
      ],
      "nameservers": "8.8.8.8",
      "updated_at": null,
      "uuid": "e60b7d12-7585-486e-9c27-3d16e0daba09"
}
`

func HandleDNSListSuccessfully(t *testing.T) {
	th.Mux.HandleFunc("/idns", func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "GET")
		th.TestHeader(t, r, "X-Auth-Token", client.TokenID)
		w.Header().Add("Content-Type", "application/json")
		fmt.Fprintf(w, DNSListBody)
	})
}

func HandleDNSGetSuccessfully(t *testing.T) {
	th.Mux.HandleFunc("/idns/e60b7d12-7585-486e-9c27-3d16e0daba09", func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "GET")
		th.TestHeader(t, r, "X-Auth-Token", client.TokenID)
		th.TestHeader(t, r, "Accept", "application/json")

		fmt.Fprintf(w, SingleDNSBody)
	})
}

func HandleDNSUpdateSuccessfully(t *testing.T) {
	th.Mux.HandleFunc("/idns/e60b7d12-7585-486e-9c27-3d16e0daba09", func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "PATCH")
		th.TestHeader(t, r, "X-Auth-Token", client.TokenID)
		th.TestHeader(t, r, "Accept", "application/json")
		th.TestHeader(t, r, "Content-Type", "application/json")
		th.TestJSONRequest(t, r, `[ { "op": "replace", "path": "/nameservers", "value": "128.224.144.130" } ]`)

		fmt.Fprintf(w, SingleDNSBody)
	})
}
