/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019 Wind River Systems, Inc. */

package testing

import (
	"fmt"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/hostFilesystems"
	"github.com/gophercloud/gophercloud/testhelper/client"
	"net/http"
	"testing"

	th "github.com/gophercloud/gophercloud/testhelper"
)

var (
	FileSystemHerp = hostFilesystems.FileSystem{
		ID:            "3e181a1e-160d-4e24-b8e9-675ebbcaed52",
		Name:          "Herp",
		HostID:        "d99637e9-5451-45c6-98f4-f18968e43e91",
		Size:          8,
		LogicalVolume: "test-lv",
		CreatedAt:     "2019-08-07T14:42:48.796061+00:00",
		UpdatedAt:     nil,
	}
	FileSystemDerp = hostFilesystems.FileSystem{
		ID:            "1a43b9e1-6360-46c1-adbe-81987a732e94",
		Name:          "Derp",
		HostID:        "d99637e9-5451-45c6-98f4-f18968e43e91",
		Size:          40,
		LogicalVolume: "test-lv",
		CreatedAt:     "2019-08-07T14:42:48.808053+00:00",
		UpdatedAt:     nil,
	}
)

const FileSystemListBody = `
{
  "host_fs": [
    {
      "created_at": "2019-08-07T14:42:48.796061+00:00",
      "ihost_uuid": "d99637e9-5451-45c6-98f4-f18968e43e91",
      "links": [
        {
          "href": "http://192.168.204.2:6385/v1/host_fs/3e181a1e-160d-4e24-b8e9-675ebbcaed52",
          "rel": "self"
        },
        {
          "href": "http://192.168.204.2:6385/host_fs/3e181a1e-160d-4e24-b8e9-675ebbcaed52",
          "rel": "bookmark"
        }
      ],
      "logical_volume": "test-lv",
      "name": "Herp",
      "size": 8,
      "updated_at": null,
      "uuid": "3e181a1e-160d-4e24-b8e9-675ebbcaed52"
    },
    {
      "created_at": "2019-08-07T14:42:48.808053+00:00",
      "ihost_uuid": "d99637e9-5451-45c6-98f4-f18968e43e91",
      "links": [
        {
          "href": "http://192.168.204.2:6385/v1/host_fs/1a43b9e1-6360-46c1-adbe-81987a732e94",
          "rel": "self"
        },
        {
          "href": "http://192.168.204.2:6385/host_fs/1a43b9e1-6360-46c1-adbe-81987a732e94",
          "rel": "bookmark"
        }
      ],
      "logical_volume": "test-lv",
      "name": "Derp",
      "size": 40,
      "updated_at": null,
      "uuid": "1a43b9e1-6360-46c1-adbe-81987a732e94"
    }
  ]
}
`

const FileSystemSingleBody = `
{
      "created_at": "2019-08-07T14:42:48.808053+00:00",
      "ihost_uuid": "d99637e9-5451-45c6-98f4-f18968e43e91",
      "links": [
        {
          "href": "http://192.168.204.2:6385/v1/host_fs/1a43b9e1-6360-46c1-adbe-81987a732e94",
          "rel": "self"
        },
        {
          "href": "http://192.168.204.2:6385/host_fs/1a43b9e1-6360-46c1-adbe-81987a732e94",
          "rel": "bookmark"
        }
      ],
      "logical_volume": "test-lv",
      "name": "Derp",
      "size": 40,
      "updated_at": null,
      "uuid": "1a43b9e1-6360-46c1-adbe-81987a732e94"
}
`

func HandleFileSystemListSuccessfully(t *testing.T) {
	th.Mux.HandleFunc("/ihosts/d99637e9-5451-45c6-98f4-f18968e43e91/host_fs", func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "GET")
		th.TestHeader(t, r, "X-Auth-Token", client.TokenID)
		w.Header().Add("Content-Type", "application/json")
		fmt.Fprintf(w, FileSystemListBody)
	})
}

func HandleFileSystemGetSuccessfully(t *testing.T) {
	th.Mux.HandleFunc("/host_fs/1a43b9e1-6360-46c1-adbe-81987a732e94", func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "GET")
		th.TestHeader(t, r, "X-Auth-Token", client.TokenID)
		w.Header().Add("Content-Type", "application/json")
		fmt.Fprintf(w, FileSystemSingleBody)
	})
}

func HandleFileSystemUpdateSuccessfully(t *testing.T) {
	th.Mux.HandleFunc("/ihosts/d99637e9-5451-45c6-98f4-f18968e43e91/host_fs/update_many", func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "PUT")
		th.TestHeader(t, r, "X-Auth-Token", client.TokenID)
		th.TestHeader(t, r, "Accept", "application/json")
		th.TestHeader(t, r, "Content-Type", "application/json")
		th.TestJSONRequest(t, r, `[ [ { "op": "replace", "path": "/name", "value": "Derp"  }, { "op": "replace", "path": "/size", "value": 50 } ] ]`)
		fmt.Fprintf(w, FileSystemSingleBody)
	})
}
