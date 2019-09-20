/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019 Wind River Systems, Inc. */

package testing

import (
	"fmt"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/ntp"
	th "github.com/gophercloud/gophercloud/testhelper"
	"github.com/gophercloud/gophercloud/testhelper/client"
	"net/http"
	"testing"
)

var (
	updatedAt = "2019-08-07T15:01:15.254487+00:00"
	NTPHerp   = ntp.NTP{
		ID:         "4a667701-fcca-448e-9cdb-c02df956920d",
		NTPServers: "0.pool.ntp.org,1.pool.ntp.org,2.pool.ntp.org",
		SystemID:   "5af5f7e5-1eea-4e76-b539-ac552e132e47",
		CreatedAt:  "2019-08-07T14:40:00.095712+00:00",
		UpdatedAt:  &updatedAt,
	}
	NTPDerp = ntp.NTP{
		ID:         "92939488-6f53-4913-aa15-ce89162751c6",
		NTPServers: "0.pool.ntp.org",
		SystemID:   "11e0c25d-80ff-4cdb-b20d-eac604469a87",
		CreatedAt:  "2019-08-07T14:40:00.095712+00:00",
		UpdatedAt:  nil,
	}
)

const NTPListBody = `
{
  "intps": [
    {
      "created_at": "2019-08-07T14:40:00.095712+00:00",
      "isystem_uuid": "5af5f7e5-1eea-4e76-b539-ac552e132e47",
      "links": [
        {
          "href": "http://192.168.204.2:6385/v1/intps/4a667701-fcca-448e-9cdb-c02df956920d",
          "rel": "self"
        },
        {
          "href": "http://192.168.204.2:6385/intps/4a667701-fcca-448e-9cdb-c02df956920d",
          "rel": "bookmark"
        }
      ],
      "ntpservers": "0.pool.ntp.org,1.pool.ntp.org,2.pool.ntp.org",
      "updated_at": "2019-08-07T15:01:15.254487+00:00",
      "uuid": "4a667701-fcca-448e-9cdb-c02df956920d"
    },
    {
      "created_at": "2019-08-07T14:40:00.095712+00:00",
      "isystem_uuid": "11e0c25d-80ff-4cdb-b20d-eac604469a87",
      "links": [
        {
          "href": "http://192.168.204.2:6385/v1/intps/4a667701-fcca-448e-9cdb-c02df956920d",
          "rel": "self"
        },
        {
          "href": "http://192.168.204.2:6385/intps/4a667701-fcca-448e-9cdb-c02df956920d",
          "rel": "bookmark"
        }
      ],
      "ntpservers": "0.pool.ntp.org",
      "updated_at": null,
      "uuid": "92939488-6f53-4913-aa15-ce89162751c6"
    }
  ]
}
`

const NTPSingleBody = `
{
      "created_at": "2019-08-07T14:40:00.095712+00:00",
      "isystem_uuid": "11e0c25d-80ff-4cdb-b20d-eac604469a87",
      "links": [
        {
          "href": "http://192.168.204.2:6385/v1/intps/4a667701-fcca-448e-9cdb-c02df956920d",
          "rel": "self"
        },
        {
          "href": "http://192.168.204.2:6385/intps/4a667701-fcca-448e-9cdb-c02df956920d",
          "rel": "bookmark"
        }
      ],
      "ntpservers": "0.pool.ntp.org",
      "updated_at": null,
      "uuid": "92939488-6f53-4913-aa15-ce89162751c6"
    }
`

func HandleNTPListSuccessfully(t *testing.T) {
	th.Mux.HandleFunc("/intp", func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "GET")
		th.TestHeader(t, r, "X-Auth-Token", client.TokenID)
		w.Header().Add("Content-Type", "application/json")
		fmt.Fprintf(w, NTPListBody)
	})
}

func HandleNTPUpdateSuccessfully(t *testing.T) {
	th.Mux.HandleFunc("/intp/92939488-6f53-4913-aa15-ce89162751c6", func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "PATCH")
		th.TestHeader(t, r, "X-Auth-Token", client.TokenID)
		th.TestHeader(t, r, "Accept", "application/json")
		th.TestHeader(t, r, "Content-Type", "application/json")
		th.TestJSONRequest(t, r, `[ { "op": "replace", "path": "/ntpservers", "value": "1.pool.ntp.org" } ]`)

		fmt.Fprintf(w, NTPSingleBody)
	})
}

func HandleNTPGetSuccessfully(t *testing.T) {
	th.Mux.HandleFunc("/intp/92939488-6f53-4913-aa15-ce89162751c6", func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "GET")
		th.TestHeader(t, r, "X-Auth-Token", client.TokenID)
		th.TestHeader(t, r, "Accept", "application/json")

		fmt.Fprintf(w, NTPSingleBody)
	})
}
