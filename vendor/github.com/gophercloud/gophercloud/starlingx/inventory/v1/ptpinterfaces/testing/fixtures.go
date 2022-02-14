/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2022 Wind River Systems, Inc. */

package testing

import (
	"fmt"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/ptpinterfaces"
	"github.com/gophercloud/gophercloud/testhelper/client"
	"net/http"
	"testing"
	th "github.com/gophercloud/gophercloud/testhelper"
)

var (
	controllerHostID        = "daadd444-d2f5-464c-9527-96fd34a05c16"
	interfaceUUID           = "f75b7395-df5f-422d-a404-db24d3a07cea"
	herpUUID                = "53041360-451f-49ea-8843-44fab16f6628"
	PTPInterfaceHerp        = ptpinterfaces.PTPInterface{
		PTPInstanceUUID: herpUUID,
		InterfaceNames:  []string{},
		UUID:            "b7d51ba0-35d7-4bab-9e27-a8b701587c54",
		ID:              3,
		PTPInstanceName: "ptp1",
		HostNames:       []string{},
		Parameters:      []string{},
		CreatedAt:       "2022-01-19T20:42:18.638033+00:00",
		UpdatedAt:       nil,
		Name:            "ptpint1",
	}
	PTPInterfaceDerp        = ptpinterfaces.PTPInterface{
		PTPInstanceUUID: "fa5defce-2546-4786-ae58-7bb08e2105fc",
		InterfaceNames:  []string{},
		UUID:            "45f0e417-26be-4ef7-b9c4-7283611a9c20",
		ID:              4,
		PTPInstanceName: "phc2sys1",
		HostNames:       []string{},
		Parameters:      []string{},
		CreatedAt:       "2022-01-20T18:17:38.095714+00:00",
		UpdatedAt:       nil,
		Name:            "ptpint2",
	}
	PTPInterfaceHerpUpdated = ptpinterfaces.PTPInterface{
		PTPInstanceUUID: herpUUID,
		InterfaceNames:  []string{},
		UUID:            "b7d51ba0-35d7-4bab-9e27-a8b701587c54",
		ID:              3,
		PTPInstanceName: "ptp1",
		HostNames:       []string{},
		Parameters:      []string{"masterOnly=0"},
		CreatedAt:       "2022-01-19T20:42:18.638033+00:00",
		UpdatedAt:       nil,
		Name:            "ptpint1",
	}
	PTPIntHerpAssignedToInt = ptpinterfaces.PTPInterface{
		PTPInstanceUUID: herpUUID,
		InterfaceNames:  []string{"controller-0/oam0"},
		UUID:            "b7d51ba0-35d7-4bab-9e27-a8b701587c54",
		ID:              3,
		PTPInstanceName: "ptp1",
		HostNames:       []string{"controller-0"},
		Parameters:      []string{},
		CreatedAt:       "2022-01-19T20:42:18.638033+00:00",
		UpdatedAt:       nil,
		Name:            "ptpint1",
	}
)

const PTPInterfaceListBody = `
{
	"ptp_interfaces": [
		{
			"ptp_instance_uuid": "53041360-451f-49ea-8843-44fab16f6628",
			"interface_names": [],
			"ptp_instance_id": 1,
			"uuid": "b7d51ba0-35d7-4bab-9e27-a8b701587c54",
			"parameters": [],
			"created_at": "2022-01-19T20:42:18.638033+00:00",
			"updated_at": null,
			"capabilities": {},
			"hostnames": [],
			"ptp_instance_name": "ptp1",
			"type": "ptp-interface",
			"id": 3,
			"name": "ptpint1"
		}, 
		{
			"ptp_instance_uuid": "fa5defce-2546-4786-ae58-7bb08e2105fc",
			"interface_names": [],
			"ptp_instance_id": 2,
			"uuid": "45f0e417-26be-4ef7-b9c4-7283611a9c20",
			"parameters": [],
			"created_at": "2022-01-20T18:17:38.095714+00:00",
			"updated_at": null,
			"capabilities": {},
			"hostnames": [],
			"ptp_instance_name": "phc2sys1",
			"type": "ptp-interface",
			"id": 4,
			"name": "ptpint2"
			}
		]
}
`

const PTPInterfaceSingleBody = `
{
	"ptp_instance_uuid": "53041360-451f-49ea-8843-44fab16f6628",
	"interface_names": [],
	"ptp_instance_id": 1,
	"uuid": "b7d51ba0-35d7-4bab-9e27-a8b701587c54",
	"parameters": [],
	"created_at": "2022-01-19T20:42:18.638033+00:00",
	"updated_at": null,
	"capabilities": {},
	"hostnames": [],
	"ptp_instance_name": "ptp1",
	"type": "ptp-interface",
	"id": 3,
	"name": "ptpint1"
}
`

const AddPTPParametersBody = `
{
	"ptp_instance_uuid": "53041360-451f-49ea-8843-44fab16f6628",
	"interface_names": [],
	"ptp_instance_id": 1,
	"uuid": "b7d51ba0-35d7-4bab-9e27-a8b701587c54",
	"parameters": ["masterOnly=0"],
	"created_at": "2022-01-19T20:42:18.638033+00:00",
	"updated_at": null,
	"capabilities": {},
	"hostnames": [],
	"ptp_instance_name": "ptp1",
	"type": "ptp-interface",
	"id": 3,
	"name": "ptpint1"
}
`

const PTPIntHerpAssignedToIntBody = `
{
	"ptp_instance_uuid": "53041360-451f-49ea-8843-44fab16f6628",
	"interface_names": ["controller-0/oam0"],
	"ptp_instance_id": 1,
	"uuid": "b7d51ba0-35d7-4bab-9e27-a8b701587c54",
	"parameters": [],
	"created_at": "2022-01-19T20:42:18.638033+00:00",
	"updated_at": null,
	"capabilities": {},
	"hostnames": ["controller-0"],
	"ptp_instance_name": "ptp1",
	"type": "ptp-interface",
	"id": 3,
	"name": "ptpint1"
}
`

func HandlePTPInterfaceListSuccessfully(t *testing.T) {
	th.Mux.HandleFunc("/ptp_interfaces", func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "GET")
		th.TestHeader(t, r, "X-Auth-Token", client.TokenID)
		w.Header().Add("Content-Type", "application/json")
		fmt.Fprintf(w, PTPInterfaceListBody)
	})
}

func HandlePTPInterfaceGetSuccessfully(t *testing.T) {
	th.Mux.HandleFunc("/ptp_interfaces/b7d51ba0-35d7-4bab-9e27-a8b701587c54", func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "GET")
		th.TestHeader(t, r, "X-Auth-Token", client.TokenID)
		th.TestHeader(t, r, "Accept", "application/json")

		fmt.Fprintf(w, PTPInterfaceSingleBody)
	})
}

func HandlePTPInterfaceDeletionSuccessfully(t *testing.T) {
	th.Mux.HandleFunc("/ptp_interfaces/b7d51ba0-35d7-4bab-9e27-a8b701587c54", func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "DELETE")
		th.TestHeader(t, r, "X-Auth-Token", client.TokenID)

		w.WriteHeader(http.StatusNoContent)
	})
}

func HandlePTPInterfaceCreationSuccessfully(t *testing.T, response string) {
	th.Mux.HandleFunc("/ptp_interfaces", func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "POST")
		th.TestHeader(t, r, "X-Auth-Token", client.TokenID)
		th.TestJSONRequest(t, r, `{
          "name": "ptpint1",
          "ptp_instance_uuid": "53041360-451f-49ea-8843-44fab16f6628"
        }`)

		w.WriteHeader(http.StatusAccepted)
		w.Header().Add("Content-Type", "application/json")
		fmt.Fprintf(w, response)
	})
}

func HandleAddPTPParameterSuccessfully(t *testing.T) {
	th.Mux.HandleFunc("/ptp_interfaces/"+herpUUID, func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "PATCH")
		th.TestHeader(t, r, "X-Auth-Token", client.TokenID)
		th.TestHeader(t, r, "Accept", "application/json")
		th.TestHeader(t, r, "Content-Type", "application/json")
		th.TestJSONRequest(t, r, `[ { "op": "add", "path": "/ptp_parameters/-", "value": "masterOnly=0" } ]`)
		fmt.Fprintf(w, AddPTPParametersBody)
	})
}

func HandleRemovePTPParameterSuccessfully(t *testing.T) {
	th.Mux.HandleFunc("/ptp_interfaces/"+herpUUID, func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "PATCH")
		th.TestHeader(t, r, "X-Auth-Token", client.TokenID)
		th.TestHeader(t, r, "Accept", "application/json")
		th.TestHeader(t, r, "Content-Type", "application/json")
		th.TestJSONRequest(t, r, `[ { "op": "remove", "path": "/ptp_parameters/-", "value": "masterOnly=0" } ]`)
		fmt.Fprintf(w, PTPInterfaceSingleBody)
	})
}

func HandleHostPTPInterfaceListSuccessfully(t *testing.T) {
	th.Mux.HandleFunc("/ihosts/"+controllerHostID+"/ptp_interfaces", func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "GET")
		th.TestHeader(t, r, "X-Auth-Token", client.TokenID)
		w.Header().Add("Content-Type", "application/json")
		fmt.Fprintf(w, PTPInterfaceListBody)
	})
}

func HandleIntPTPInterfaceListSuccessfully(t *testing.T) {
	th.Mux.HandleFunc("/iinterfaces/"+interfaceUUID+"/ptp_interfaces", func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "GET")
		th.TestHeader(t, r, "X-Auth-Token", client.TokenID)
		w.Header().Add("Content-Type", "application/json")
		fmt.Fprintf(w, PTPInterfaceListBody)
	})
}

func HandleAddPTPIntToIntSuccessfully(t *testing.T) {
	th.Mux.HandleFunc("/iinterfaces/"+interfaceUUID, func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "PATCH")
		th.TestHeader(t, r, "X-Auth-Token", client.TokenID)
		th.TestHeader(t, r, "Accept", "application/json")
		th.TestHeader(t, r, "Content-Type", "application/json")
		th.TestJSONRequest(t, r, `[ { "op": "add", "path": "/ptp_interfaces/-", "value": 3 } ]`)
		fmt.Fprintf(w, PTPIntHerpAssignedToIntBody)
	})
}

func HandleRemovePTPIntFromIntSuccessfully(t *testing.T) {
	th.Mux.HandleFunc("/iinterfaces/"+interfaceUUID, func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "PATCH")
		th.TestHeader(t, r, "X-Auth-Token", client.TokenID)
		th.TestHeader(t, r, "Accept", "application/json")
		th.TestHeader(t, r, "Content-Type", "application/json")
		th.TestJSONRequest(t, r, `[ { "op": "remove", "path": "/ptp_interfaces/-", "value": 3 } ]`)
		fmt.Fprintf(w, PTPInterfaceSingleBody)
	})
}
