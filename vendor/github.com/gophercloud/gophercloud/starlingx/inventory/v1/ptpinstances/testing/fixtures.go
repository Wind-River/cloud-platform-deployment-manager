/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2022 Wind River Systems, Inc. */

package testing

import (
	"fmt"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/ptpinstances"
	"github.com/gophercloud/gophercloud/testhelper/client"
	"net/http"
	"testing"
	th "github.com/gophercloud/gophercloud/testhelper"
)

var (
	controllerHostID         = "daadd444-d2f5-464c-9527-96fd34a05c16"
	herpUUID			     = "fa5defce-2546-4786-ae58-7bb08e2105fc"
	PTPInstanceHerp 	     = ptpinstances.PTPInstance{
		UUID: 	      herpUUID,
		ID:           2,
		Name:         "phc2sys1",
		Service:      "phc2sys",
		HostNames:    []string{},
		Parameters:   []string{},
		CreatedAt:    "2022-01-18T20:47:27.655974+00:00",
		UpdatedAt:    nil,
	}
	PTPInstanceDerp 	     = ptpinstances.PTPInstance{
		UUID:         "53041360-451f-49ea-8843-44fab16f6628",
		ID:           1,
		Name:         "ptp1",
		Service:      "ptp4l",
		HostNames:    []string{},
		Parameters:   []string{},
		CreatedAt:    "2022-01-18T17:56:43.012323+00:00",
		UpdatedAt:    nil,
	}
	PTPInstanceHerpUpdated   = ptpinstances.PTPInstance{
		UUID: 	      herpUUID,
		ID:           2,
		Name:         "phc2sys1",
		Service:      "phc2sys",
		HostNames:    []string{},
		Parameters:   []string{"domainNumber=24"},
		CreatedAt:    "2022-01-18T20:47:27.655974+00:00",
		UpdatedAt:    nil,
	}
	PTPInstanceHerpAddToHost = ptpinstances.PTPInstance{
		UUID: 	      herpUUID,
		ID:           2,
		Name:         "phc2sys1",
		Service:      "phc2sys",
		HostNames:    []string{"controller-0"},
		Parameters:   []string{},
		CreatedAt:    "2022-01-18T20:47:27.655974+00:00",
		UpdatedAt:    nil,
	}
)

const PTPInstanceListBody = `
{
	"ptp_instances": [
		{
			"uuid": "fa5defce-2546-4786-ae58-7bb08e2105fc", 
		 	"service": "phc2sys", 
		 	"created_at": "2022-01-18T20:47:27.655974+00:00",
		 	"updated_at": null,
		 	"capabilities": {},
		 	"hostnames": [],
		 	"parameters": [],
		 	"type": "ptp-instance",
		 	"id": 2,
		 	"name": "phc2sys1"
		}, 
		{
			"uuid": "53041360-451f-49ea-8843-44fab16f6628",
			"service": "ptp4l",
			"created_at": "2022-01-18T17:56:43.012323+00:00",
			"updated_at": null,
			"capabilities": {},
			"hostnames": [],
			"parameters": [],
			"type": "ptp-instance",
			"id": 1,
			"name": "ptp1"
		}
	]
}
`

const PTPInstanceSingleBody = `
{
	"uuid": "fa5defce-2546-4786-ae58-7bb08e2105fc",
	"service": "phc2sys",
	"created_at": "2022-01-18T20:47:27.655974+00:00",
	"updated_at": null,
	"capabilities": {},
	"hostnames": [],
	"parameters": [],
	"type": "ptp-instance",
	"id": 2,
	"name": "phc2sys1"
}
`

const AddPTPParametersBody = `
{
	"uuid": "fa5defce-2546-4786-ae58-7bb08e2105fc",
	"service": "phc2sys",
	"created_at": "2022-01-18T20:47:27.655974+00:00",
	"updated_at": null,
	"capabilities": {},
	"hostnames": [],
	"parameters": ["domainNumber=24"],
	"type": "ptp-instance",
	"id": 2,
	"name": "phc2sys1"
}
`

const HerpAddToHostBody = `
{
	"uuid": "fa5defce-2546-4786-ae58-7bb08e2105fc",
	"service": "phc2sys",
	"created_at": "2022-01-18T20:47:27.655974+00:00",
	"updated_at": null,
	"capabilities": {},
	"hostnames": ["controller-0"],
	"parameters": [],
	"type": "ptp-instance",
	"id": 2,
	"name": "phc2sys1"
}
`

func HandlePTPInstanceListSuccessfully(t *testing.T) {
	th.Mux.HandleFunc("/ptp_instances", func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "GET")
		th.TestHeader(t, r, "X-Auth-Token", client.TokenID)
		w.Header().Add("Content-Type", "application/json")
		fmt.Fprintf(w, PTPInstanceListBody)
	})
}

func HandleHostPTPInstanceListSuccessfully(t *testing.T) {
	th.Mux.HandleFunc("/ihosts/"+controllerHostID+"/ptp_instances", func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "GET")
		th.TestHeader(t, r, "X-Auth-Token", client.TokenID)
		w.Header().Add("Content-Type", "application/json")
		fmt.Fprintf(w, PTPInstanceListBody)
	})
}

func HandlePTPInstanceGetSuccessfully(t *testing.T) {
	th.Mux.HandleFunc("/ptp_instances/"+herpUUID, func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "GET")
		th.TestHeader(t, r, "X-Auth-Token", client.TokenID)
		th.TestHeader(t, r, "Accept", "application/json")

		fmt.Fprintf(w, PTPInstanceSingleBody)
	})
}

func HandlePTPInstanceDeletionSuccessfully(t *testing.T) {
	th.Mux.HandleFunc("/ptp_instances/"+herpUUID, func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "DELETE")
		th.TestHeader(t, r, "X-Auth-Token", client.TokenID)

		w.WriteHeader(http.StatusNoContent)
	})
}

func HandlePTPInstanceApplySuccessfully(t *testing.T) {
	th.Mux.HandleFunc("/ptp_instances/apply", func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "POST")
		th.TestHeader(t, r, "X-Auth-Token", client.TokenID)

		w.WriteHeader(http.StatusNoContent)
	})
}

func HandlePTPInstanceCreationSuccessfully(t *testing.T, response string) {
	th.Mux.HandleFunc("/ptp_instances", func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "POST")
		th.TestHeader(t, r, "X-Auth-Token", client.TokenID)
		th.TestJSONRequest(t, r, `{
          "name": "phc2sys1",
          "service": "phc2sys"
        }`)

		w.WriteHeader(http.StatusAccepted)
		w.Header().Add("Content-Type", "application/json")
		fmt.Fprintf(w, response)
	})
}

func HandleAddPTPParameterSuccessfully(t *testing.T) {
	th.Mux.HandleFunc("/ptp_instances/"+herpUUID, func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "PATCH")
		th.TestHeader(t, r, "X-Auth-Token", client.TokenID)
		th.TestHeader(t, r, "Accept", "application/json")
		th.TestHeader(t, r, "Content-Type", "application/json")
		th.TestJSONRequest(t, r, `[ { "op": "add", "path": "/ptp_parameters/-", "value": "domainNumber=24" } ]`)
		fmt.Fprintf(w, AddPTPParametersBody)
	})
}

func HandleRemovePTPParameterSuccessfully(t *testing.T) {
	th.Mux.HandleFunc("/ptp_instances/"+herpUUID, func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "PATCH")
		th.TestHeader(t, r, "X-Auth-Token", client.TokenID)
		th.TestHeader(t, r, "Accept", "application/json")
		th.TestHeader(t, r, "Content-Type", "application/json")
		th.TestJSONRequest(t, r, `[ { "op": "remove", "path": "/ptp_parameters/-", "value": "domainNumber=24" } ]`)
		fmt.Fprintf(w, PTPInstanceSingleBody)
	})
}

func HandleAddInstanceToHostSuccessfully(t *testing.T) {
	th.Mux.HandleFunc("/ihosts/"+controllerHostID, func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "PATCH")
		th.TestHeader(t, r, "X-Auth-Token", client.TokenID)
		th.TestHeader(t, r, "Accept", "application/json")
		th.TestHeader(t, r, "Content-Type", "application/json")
		th.TestJSONRequest(t, r, `[ { "op": "add", "path": "/ptp_instances/-", "value": 2 } ]`)
		fmt.Fprintf(w, HerpAddToHostBody)
	})
}

func HandleRemoveInstanceFromHostSuccessfully(t *testing.T) {
	th.Mux.HandleFunc("/ihosts/"+controllerHostID, func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "PATCH")
		th.TestHeader(t, r, "X-Auth-Token", client.TokenID)
		th.TestHeader(t, r, "Accept", "application/json")
		th.TestHeader(t, r, "Content-Type", "application/json")
		th.TestJSONRequest(t, r, `[ { "op": "remove", "path": "/ptp_instances/-", "value": 2 } ]`)
		fmt.Fprintf(w, PTPInstanceSingleBody)
	})
}
