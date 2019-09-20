/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019 Wind River Systems, Inc. */

package testing

import (
	"fmt"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/cephmonitors"
	"github.com/gophercloud/gophercloud/testhelper/client"
	"net/http"
	"testing"

	th "github.com/gophercloud/gophercloud/testhelper"
)

var (
	CephMonitorHerp = cephmonitors.CephMonitor{
		ID:         "c8663bf4-43b0-43f6-8f74-33e3f8cf257c",
		HostUUID:   "d99637e9-5451-45c6-98f4-f18968e43e91",
		Hostname:   "controller-0",
		State:      "configured",
		Task:       nil,
		Size:       20,
		DevicePath: nil,
		CreatedAt:  "2019-08-07T14:42:54.350432+00:00",
		UpdatedAt:  nil,
	}
	CephMonitorDerp = cephmonitors.CephMonitor{
		ID:         "79c25f45-9e3d-42bf-ab94-7c6b193934b7",
		HostUUID:   "d99637e9-5451-45c6-98f4-f18968e43e91",
		Hostname:   "controller-1",
		State:      "configured",
		Task:       nil,
		Size:       30,
		DevicePath: nil,
		CreatedAt:  "2019-08-08T15:10:19.905984+00:00",
		UpdatedAt:  nil,
	}
	CephMonitorMerp = cephmonitors.CephMonitor{
		ID:         "f692ab74-3429-4588-83cb-ed8d8ed0f275",
		HostUUID:   "66b62c51-974b-4bcc-b273-e8365833157e",
		Hostname:   "compute-0",
		State:      "unconfigured",
		Task:       nil,
		Size:       10,
		DevicePath: nil,
		CreatedAt:  "2019-08-09T15:10:19.905984+00:00",
		UpdatedAt:  nil,
	}
)

const CephMonListBody = `
{
  "ceph_mon": [
    {
      "ceph_mon_gib": 20,
      "created_at": "2019-08-07T14:42:54.350432+00:00",
      "device_path": null,
      "hostname": "controller-0",
      "ihost_uuid": "d99637e9-5451-45c6-98f4-f18968e43e91",
      "links": [
        {
          "href": "http://192.168.204.2:6385/v1/ceph_mon/c8663bf4-43b0-43f6-8f74-33e3f8cf257c",
          "rel": "self"
        },
        {
          "href": "http://192.168.204.2:6385/ceph_mon/c8663bf4-43b0-43f6-8f74-33e3f8cf257c",
          "rel": "bookmark"
        }
      ],
      "state": "configured",
      "task": null,
      "updated_at": null,
      "uuid": "c8663bf4-43b0-43f6-8f74-33e3f8cf257c"
    },
    {
      "ceph_mon_gib": 30,
      "created_at": "2019-08-08T15:10:19.905984+00:00",
      "device_path": null,
      "hostname": "controller-1",
      "ihost_uuid": "d99637e9-5451-45c6-98f4-f18968e43e91",
      "links": [
        {
          "href": "http://192.168.204.2:6385/v1/ceph_mon/79c25f45-9e3d-42bf-ab94-7c6b193934b7",
          "rel": "self"
        },
        {
          "href": "http://192.168.204.2:6385/ceph_mon/79c25f45-9e3d-42bf-ab94-7c6b193934b7",
          "rel": "bookmark"
        }
      ],
      "state": "configured",
      "task": null,
      "updated_at": null,
      "uuid": "79c25f45-9e3d-42bf-ab94-7c6b193934b7"
    },
    {
      "ceph_mon_gib": 10,
      "created_at": "2019-08-09T15:10:19.905984+00:00",
      "device_path": null,
      "hostname": "compute-0",
      "ihost_uuid": "66b62c51-974b-4bcc-b273-e8365833157e",
      "links": [
        {
          "href": "http://192.168.204.2:6385/v1/ceph_mon/c8663bf4-43b0-43f6-8f74-33e3f8cf257c",
          "rel": "self"
        },
        {
          "href": "http://192.168.204.2:6385/ceph_mon/c8663bf4-43b0-43f6-8f74-33e3f8cf257c",
          "rel": "bookmark"
        }
      ],
      "state": "unconfigured",
      "task": null,
      "updated_at": null,
      "uuid": "f692ab74-3429-4588-83cb-ed8d8ed0f275"
    }
  ]
}
`

const CephMonSingleBody = `
    {
      "ceph_mon_gib": 30,
      "created_at": "2019-08-08T15:10:19.905984+00:00",
      "device_path": null,
      "hostname": "controller-1",
      "ihost_uuid": "d99637e9-5451-45c6-98f4-f18968e43e91",
      "links": [
        {
          "href": "http://192.168.204.2:6385/v1/ceph_mon/79c25f45-9e3d-42bf-ab94-7c6b193934b7",
          "rel": "self"
        },
        {
          "href": "http://192.168.204.2:6385/ceph_mon/79c25f45-9e3d-42bf-ab94-7c6b193934b7",
          "rel": "bookmark"
        }
      ],
      "state": "configured",
      "task": null,
      "updated_at": null,
      "uuid": "79c25f45-9e3d-42bf-ab94-7c6b193934b7"
    }
`

func HandleCephMonListSuccessfully(t *testing.T) {
	th.Mux.HandleFunc("/ceph_mon", func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "GET")
		th.TestHeader(t, r, "X-Auth-Token", client.TokenID)
		w.Header().Add("Content-Type", "application/json")
		fmt.Fprintf(w, CephMonListBody)
	})
}

func HandleCephMonUpdateSuccessfully(t *testing.T) {
	th.Mux.HandleFunc("/ceph_mon/79c25f45-9e3d-42bf-ab94-7c6b193934b7", func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "PATCH")
		th.TestHeader(t, r, "X-Auth-Token", client.TokenID)
		th.TestHeader(t, r, "Accept", "application/json")
		th.TestHeader(t, r, "Content-Type", "application/json")
		th.TestJSONRequest(t, r, `[ { "op": "replace", "path": "/ceph_mon_gib", "value": 35 } ]`)
		fmt.Fprintf(w, CephMonSingleBody)
	})
}

func HandleCephMonDeletionSuccessfully(t *testing.T) {
	th.Mux.HandleFunc("/ceph_mon/79c25f45-9e3d-42bf-ab94-7c6b193934b7", func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "DELETE")
		th.TestHeader(t, r, "X-Auth-Token", client.TokenID)

		w.WriteHeader(http.StatusNoContent)
	})
}

func HandleCephMonCreationSuccessfully(t *testing.T, response string) {
	th.Mux.HandleFunc("/ceph_mon", func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "POST")
		th.TestHeader(t, r, "X-Auth-Token", client.TokenID)
		th.TestJSONRequest(t, r, `{
          "ceph_mon_gib": 30,
          "ihost_uuid": "d99637e9-5451-45c6-98f4-f18968e43e91"
        }`)

		w.WriteHeader(http.StatusAccepted)
		w.Header().Add("Content-Type", "application/json")
		fmt.Fprintf(w, response)
	})
}
