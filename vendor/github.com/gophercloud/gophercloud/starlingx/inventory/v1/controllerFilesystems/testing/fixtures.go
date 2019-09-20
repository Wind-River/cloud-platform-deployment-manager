/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019 Wind River Systems, Inc. */

package testing

import (
	"fmt"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/controllerFilesystems"
	"github.com/gophercloud/gophercloud/testhelper/client"
	"net/http"
	"testing"

	th "github.com/gophercloud/gophercloud/testhelper"
)

var (
	FileSystemHerp = controllerFilesystems.FileSystem{
		ID:            "055ae646-355a-4a43-b2db-74c0ded7bb46",
		Name:          "Herp",
		State:         "",
		SystemUUID:    "6dc8ccb9-f687-40fa-9663-c0c286e65772",
		Replicated:    true,
		Size:          10,
		LogicalVolume: "cgcs-lv",
		CreatedAt:     "2019-08-07T14:42:25.273979+00:00",
		UpdatedAt:     nil,
	}
	FileSystemDerp = controllerFilesystems.FileSystem{
		ID:            "ff2e628d-23b2-4d73-b6b5-1c291ab6905a",
		Name:          "Derp",
		State:         "",
		SystemUUID:    "52a7c2b0-ee64-4090-9d6e-c60892128a05",
		Replicated:    false,
		Size:          1,
		LogicalVolume: "pgsql-lv",
		CreatedAt:     "2019-08-07T14:42:25.295207+00:00",
		UpdatedAt:     nil,
	}
	FileSystemMerp = controllerFilesystems.FileSystem{
		ID:            "299e1538-930b-4a7b-9ea7-7e9aabd56705",
		Name:          "Merp",
		State:         "",
		SystemUUID:    "cf9907ae-ea10-4a97-8974-0181001e9bb6",
		Replicated:    true,
		Size:          5,
		LogicalVolume: "etcd-lv",
		CreatedAt:     "2019-08-07T14:42:25.309300+00:00",
		UpdatedAt:     nil,
	}
)

const FileSystemListBody = `
{
  "controller_fs": [
    {
      "created_at": "2019-08-07T14:42:25.273979+00:00",
      "isystem_uuid": "6dc8ccb9-f687-40fa-9663-c0c286e65772",
      "links": [
        {
          "href": "http://192.168.204.2:6385/v1/controller_fs/055ae646-355a-4a43-b2db-74c0ded7bb46",
          "rel": "self"
        },
        {
          "href": "http://192.168.204.2:6385/controller_fs/055ae646-355a-4a43-b2db-74c0ded7bb46",
          "rel": "bookmark"
        }
      ],
      "logical_volume": "cgcs-lv",
      "name": "Herp",
      "replicated": true,
      "size": 10,
      "state": null,
      "updated_at": null,
      "uuid": "055ae646-355a-4a43-b2db-74c0ded7bb46"
    },
    {
      "created_at": "2019-08-07T14:42:25.295207+00:00",
      "isystem_uuid": "52a7c2b0-ee64-4090-9d6e-c60892128a05",
      "links": [
        {
          "href": "http://192.168.204.2:6385/v1/controller_fs/ff2e628d-23b2-4d73-b6b5-1c291ab6905a",
          "rel": "self"
        },
        {
          "href": "http://192.168.204.2:6385/controller_fs/ff2e628d-23b2-4d73-b6b5-1c291ab6905a",
          "rel": "bookmark"
        }
      ],
      "logical_volume": "pgsql-lv",
      "name": "Derp",
      "replicated": false,
      "size": 1,
      "state": null,
      "updated_at": null,
      "uuid": "ff2e628d-23b2-4d73-b6b5-1c291ab6905a"
    },
    {
      "created_at": "2019-08-07T14:42:25.309300+00:00",
      "isystem_uuid": "cf9907ae-ea10-4a97-8974-0181001e9bb6",
      "links": [
        {
          "href": "http://192.168.204.2:6385/v1/controller_fs/299e1538-930b-4a7b-9ea7-7e9aabd56705",
          "rel": "self"
        },
        {
          "href": "http://192.168.204.2:6385/controller_fs/299e1538-930b-4a7b-9ea7-7e9aabd56705",
          "rel": "bookmark"
        }
      ],
      "logical_volume": "etcd-lv",
      "name": "Merp",
      "replicated": true,
      "size": 5,
      "state": null,
      "updated_at": null,
      "uuid": "299e1538-930b-4a7b-9ea7-7e9aabd56705"
    }
  ]
}
`

const FileSystemSingleBody = `
{
      "created_at": "2019-08-07T14:42:25.295207+00:00",
      "isystem_uuid": "52a7c2b0-ee64-4090-9d6e-c60892128a05",
      "links": [
        {
          "href": "http://192.168.204.2:6385/v1/controller_fs/ff2e628d-23b2-4d73-b6b5-1c291ab6905a",
          "rel": "self"
        },
        {
          "href": "http://192.168.204.2:6385/controller_fs/ff2e628d-23b2-4d73-b6b5-1c291ab6905a",
          "rel": "bookmark"
        }
      ],
      "logical_volume": "pgsql-lv",
      "name": "Derp",
      "replicated": false,
      "size": 1,
      "state": null,
      "updated_at": null,
      "uuid": "ff2e628d-23b2-4d73-b6b5-1c291ab6905a"
}
`

func HandleFileSystemListSuccessfully(t *testing.T) {
	th.Mux.HandleFunc("/controller_fs", func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "GET")
		th.TestHeader(t, r, "X-Auth-Token", client.TokenID)
		w.Header().Add("Content-Type", "application/json")
		fmt.Fprintf(w, FileSystemListBody)
	})
}

func HandleFileSystemGetSuccessfully(t *testing.T) {
	th.Mux.HandleFunc("/controller_fs/ff2e628d-23b2-4d73-b6b5-1c291ab6905a", func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "GET")
		th.TestHeader(t, r, "X-Auth-Token", client.TokenID)
		w.Header().Add("Content-Type", "application/json")
		fmt.Fprintf(w, FileSystemSingleBody)
	})
}

func HandleFileSystemUpdateSuccessfully(t *testing.T) {
	th.Mux.HandleFunc("/controller_fs/ff2e628d-23b2-4d73-b6b5-1c291ab6905a", func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "PATCH")
		th.TestHeader(t, r, "X-Auth-Token", client.TokenID)
		th.TestHeader(t, r, "Accept", "application/json")
		th.TestHeader(t, r, "Content-Type", "application/json")
		th.TestJSONRequest(t, r, `[ { "op": "replace", "path": "/name", "value": "new-name" } ]`)
		fmt.Fprintf(w, FileSystemSingleBody)
	})
}
