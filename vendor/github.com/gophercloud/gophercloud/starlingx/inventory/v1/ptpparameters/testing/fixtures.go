/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2022 Wind River Systems, Inc. */

package testing

import (
	"fmt"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/ptpparameters"
	"github.com/gophercloud/gophercloud/testhelper/client"
	"net/http"
	"testing"
	th "github.com/gophercloud/gophercloud/testhelper"
)

var (
	PTPParameterHerp = ptpparameters.PTPParameter{
		Owners: []string{"424e80da-fdb0-4ddb-9f75-fa65d312d413"},
		Name: "domainNumber",
		Value: "24",
		UUID: "dd16b9c3-0fd2-491e-811d-df890a1524a1",
		CreatedAt: "2022-01-24T21:15:17.290128+00:00",
		UpdatedAt: nil,
	}
	PTPParameterDerp = ptpparameters.PTPParameter{
		Owners: []string{"82ef99c1-af38-432d-b5ac-ce6719ffc771"},
		Name: "masterOnly",
		Value: "0",
		UUID: "868e0ab8-2bc3-4d92-b736-de06bb8feb12",
		CreatedAt: "2022-01-24T21:50:27.567466+00:00",
		UpdatedAt: nil,
	}
)

const PTPParameterListBody = `
{
	"ptp_parameters": [
		{
			"owners": ["424e80da-fdb0-4ddb-9f75-fa65d312d413"],
			"name": "domainNumber",
			"created_at": "2022-01-24T21:15:17.290128+00:00",
			"updated_at": null,
			"value": "24",
			"uuid": "dd16b9c3-0fd2-491e-811d-df890a1524a1"
		}, 
		{
			"owners": ["82ef99c1-af38-432d-b5ac-ce6719ffc771"],
			"name": "masterOnly",
			"created_at": "2022-01-24T21:50:27.567466+00:00",
			"updated_at": null,
			"value": "0",
			"uuid": "868e0ab8-2bc3-4d92-b736-de06bb8feb12"
		}
	]
}
`

const PTPParameterSingleBody = `
{
	"owners": ["82ef99c1-af38-432d-b5ac-ce6719ffc771"],
	"uuid": "868e0ab8-2bc3-4d92-b736-de06bb8feb12",
	"created_at": "2022-01-24T21:50:27.567466+00:00",
	"updated_at": null,
	"value": "0",
	"id": 2,
	"name": "masterOnly"
}
`

func HandlePTPParameterListSuccessfully(t *testing.T) {
	th.Mux.HandleFunc("/ptp_parameters", func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "GET")
		th.TestHeader(t, r, "X-Auth-Token", client.TokenID)
		w.Header().Add("Content-Type", "application/json")
		fmt.Fprintf(w, PTPParameterListBody)
	})
}

func HandlePTPParameterGetSuccessfully(t *testing.T) {
	th.Mux.HandleFunc("/ptp_parameters/868e0ab8-2bc3-4d92-b736-de06bb8feb12", func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "GET")
		th.TestHeader(t, r, "X-Auth-Token", client.TokenID)
		th.TestHeader(t, r, "Accept", "application/json")

		fmt.Fprintf(w, PTPParameterSingleBody)
	})
}

func HandlePTPParameterDeletionSuccessfully(t *testing.T) {
	th.Mux.HandleFunc("/ptp_parameters/868e0ab8-2bc3-4d92-b736-de06bb8feb12", func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "DELETE")
		th.TestHeader(t, r, "X-Auth-Token", client.TokenID)

		w.WriteHeader(http.StatusNoContent)
	})
}

func HandlePTPParameterCreationSuccessfully(t *testing.T, response string) {
	th.Mux.HandleFunc("/ptp_parameters", func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "POST")
		th.TestHeader(t, r, "X-Auth-Token", client.TokenID)
		th.TestJSONRequest(t, r, `{
          "name": "masterOnly",
          "value": "0"
        }`)

		w.WriteHeader(http.StatusAccepted)
		w.Header().Add("Content-Type", "application/json")
		fmt.Fprintf(w, response)
	})
}
