/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019 Wind River Systems, Inc. */

package testing

import (
	"fmt"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/clusters"
	"github.com/gophercloud/gophercloud/testhelper/client"
	"net/http"
	"testing"

	th "github.com/gophercloud/gophercloud/testhelper"
)

var (
	clusterID   = "a6fafcc2-4f6a-4bcf-9491-dff7b4da21cc"
	ClusterHerp = clusters.Cluster{
		ID:              "1696d0a2-aeb4-428e-90f9-52bd0a1a4412",
		Name:            "Herp",
		Type:            "ceph",
		ClusterID:       &clusterID,
		DeploymentModel: "controller-nodes",
	}
	ClusterDerp = clusters.Cluster{
		ID:              "27caf12f-19af-4cd3-bc41-d4467ec80e39",
		Name:            "Derp",
		Type:            "test",
		ClusterID:       nil,
		DeploymentModel: "worker-nodes",
	}
)

const ClusterListBody = `
{
  "clusters": [
    {
      "cluster_uuid": "a6fafcc2-4f6a-4bcf-9491-dff7b4da21cc",
      "deployment_model": "controller-nodes",
      "links": [
        {
          "href": "http://192.168.204.2:6385/v1/clusters/1696d0a2-aeb4-428e-90f9-52bd0a1a4412",
          "rel": "self"
        },
        {
          "href": "http://192.168.204.2:6385/clusters/1696d0a2-aeb4-428e-90f9-52bd0a1a4412",
          "rel": "bookmark"
        }
      ],
      "name": "Herp",
      "type": "ceph",
      "uuid": "1696d0a2-aeb4-428e-90f9-52bd0a1a4412"
    },
    {
      "cluster_uuid": null,
      "deployment_model": "worker-nodes",
      "links": [
        {
          "href": "http://192.168.204.2:6385/v1/clusters/1696d0a2-aeb4-428e-90f9-52bd0a1a4412",
          "rel": "self"
        },
        {
          "href": "http://192.168.204.2:6385/clusters/1696d0a2-aeb4-428e-90f9-52bd0a1a4412",
          "rel": "bookmark"
        }
      ],
      "name": "Derp",
      "type": "test",
      "uuid": "27caf12f-19af-4cd3-bc41-d4467ec80e39"
    }
  ]
}
`

const SingleClusterBody = `
{
      "cluster_uuid": null,
      "deployment_model": "worker-nodes",
      "links": [
        {
          "href": "http://192.168.204.2:6385/v1/clusters/1696d0a2-aeb4-428e-90f9-52bd0a1a4412",
          "rel": "self"
        },
        {
          "href": "http://192.168.204.2:6385/clusters/1696d0a2-aeb4-428e-90f9-52bd0a1a4412",
          "rel": "bookmark"
        }
      ],
      "name": "Derp",
      "type": "test",
      "uuid": "27caf12f-19af-4cd3-bc41-d4467ec80e39"
    }
`

func HandleClusterListSuccessfully(t *testing.T) {
	th.Mux.HandleFunc("/clusters", func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "GET")
		th.TestHeader(t, r, "X-Auth-Token", client.TokenID)
		w.Header().Add("Content-Type", "application/json")
		fmt.Fprintf(w, ClusterListBody)
	})
}

func HandleSystemGetSuccessfully(t *testing.T) {
	th.Mux.HandleFunc("/clusters/27caf12f-19af-4cd3-bc41-d4467ec80e39", func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "GET")
		th.TestHeader(t, r, "X-Auth-Token", client.TokenID)
		th.TestHeader(t, r, "Accept", "application/json")

		fmt.Fprintf(w, SingleClusterBody)
	})
}
