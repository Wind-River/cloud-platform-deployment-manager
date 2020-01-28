/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2020 Wind River Systems, Inc. */

package testing

import (
	"fmt"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/serviceparameters"
	"github.com/gophercloud/gophercloud/testhelper/client"
	"net/http"
	"testing"

	th "github.com/gophercloud/gophercloud/testhelper"
)

var (
	herpUUID             = "85b3757d-4168-43c8-b921-5231de1955c7"
	ServiceParameterHerp = serviceparameters.ServiceParameter{
		ID:         herpUUID,
		Service:    "horizon",
		Section:    "auth",
		ParamName:  "lockout_retries",
		ParamValue: "3",
	}
	derpUUID             = "5fce00bc-ea24-41d3-9d3b-3061b58349a8"
	ServiceParameterDerp = serviceparameters.ServiceParameter{
		ID:         derpUUID,
		Service:    "horizon",
		Section:    "auth",
		ParamName:  "lockout_seconds",
		ParamValue: "300",
	}
	merpUUID             = "c7c97ce7-fb70-4197-9600-f0d20ab7f8a3"
	ServiceParameterMerp = serviceparameters.ServiceParameter{
		ID:         merpUUID,
		Service:    "radosgw",
		Section:    "config",
		ParamName:  "fs_size_mb",
		ParamValue: "25",
	}
	berpUUID             = "c515850b-413a-453c-b9ea-a946664a021b"
	ServiceParameterBerp = serviceparameters.ServiceParameter{
		ID:         berpUUID,
		Service:    "http",
		Section:    "config",
		ParamName:  "http_port",
		ParamValue: "8080",
	}

	bbqUUID             = "75b3757d-4168-43c8-b921-5231de1955c9"
	bbqResource         = "bbq::brickets::charcoal::mode"
	ServiceParameterBBQ = serviceparameters.ServiceParameter{
		ID:         "75b3757d-4168-43c8-b921-5231de1955c9",
		Service:    "bbq",
		Section:    "brickets",
		ParamName:  "charcoal",
		ParamValue: "enabled",
		Resource:   &bbqResource,
	}
	ServiceParameterBBQUpdated = serviceparameters.ServiceParameter{
		ID:         ServiceParameterBBQ.ID,
		Service:    ServiceParameterBBQ.Service,
		Section:    ServiceParameterBBQ.Section,
		ParamName:  ServiceParameterBBQ.ParamName,
		ParamValue: "disabled",
		Resource:   &bbqResource,
	}
)

const ServiceParameterListBody = `
{
  "parameters": [
    {
      "resource": null,
      "uuid": "85b3757d-4168-43c8-b921-5231de1955c7",
      "service": "horizon",
      "section": "auth",
      "links": [
        {
          "href": "http://192.168.204.1:6385/v1/parameters/85b3757d-4168-43c8-b921-5231de1955c7",
          "rel": "self"
        },
        {
          "href": "http://192.168.204.1:6385/parameters/85b3757d-4168-43c8-b921-5231de1955c7",
           "rel": "bookmark"
        }
      ],
      "value": "3",
      "name": "lockout_retries",
      "personality": null
    },
    {
      "resource": null,
      "uuid": "5fce00bc-ea24-41d3-9d3b-3061b58349a8",
      "service": "horizon",
      "section": "auth",
      "links": [
        {
          "href": "http://192.168.204.1:6385/v1/parameters/5fce00bc-ea24-41d3-9d3b-3061b58349a8",
          "rel": "self"
        },
        {
          "href": "http://192.168.204.1:6385/parameters/5fce00bc-ea24-41d3-9d3b-3061b58349a8",
          "rel": "bookmark"
        }
      ],
      "value": "300",
      "name": "lockout_seconds",
      "personality": null
    },
    {
      "resource": null,
      "uuid": "c7c97ce7-fb70-4197-9600-f0d20ab7f8a3",
      "service": "radosgw",
      "section": "config",
      "links": [
        {
          "href": "http://192.168.204.1:6385/v1/parameters/c7c97ce7-fb70-4197-9600-f0d20ab7f8a3",
          "rel": "self"
        },
        {
          "href": "http://192.168.204.1:6385/parameters/c7c97ce7-fb70-4197-9600-f0d20ab7f8a3",
          "rel": "bookmark"
        }
      ],
      "value": "25",
      "name": "fs_size_mb",
      "personality": null
    },
    {
      "resource": null,
      "uuid": "c515850b-413a-453c-b9ea-a946664a021b",
      "service": "http",
      "section": "config",
      "links": [
        {
          "href": "http://192.168.204.1:6385/v1/parameters/c515850b-413a-453c-b9ea-a946664a021b",
          "rel": "self"
        },
        {
          "href": "http://192.168.204.1:6385/parameters/c515850b-413a-453c-b9ea-a946664a021b",
          "rel": "bookmark"
        }
      ],
      "value": "8080",
      "name": "http_port",
      "personality": null
    }
  ]
}
`
const SingleServiceParameterBody = `
{
      "resource": null,
      "uuid": "c515850b-413a-453c-b9ea-a946664a021b",
      "service": "http",
      "section": "config",
      "links": [
        {
          "href": "http://192.168.204.1:6385/v1/parameters/c515850b-413a-453c-b9ea-a946664a021b",
          "rel": "self"
        },
        {
          "href": "http://192.168.204.1:6385/parameters/c515850b-413a-453c-b9ea-a946664a021b",
          "rel": "bookmark"
        }
      ],
      "value": "8080",
      "name": "http_port",
      "personality": null
}
`

/* Create mocked based on bbqUUID */
const ServiceParameterCreateBody = `
{
    "uuid": "75b3757d-4168-43c8-b921-5231de1955c9",
    "service": "bbq",
    "section": "brickets",
    "links": [
    {
      "href": "http://192.168.204.1:6385/v1/parameters/75b3757d-4168-43c8-b921-5231de1955c9",
      "rel": "self"
    },
    {
      "href": "http://192.168.204.1:6385/parameters/75b3757d-4168-43c8-b921-5231de1955c9",
      "rel": "bookmark"
    }
  ],
  "value": "enabled",
  "name": "charcoal",
  "resource": "bbq::brickets::charcoal::mode",
  "personality": null
}
`

/* Update mocked based on bbqUUID */
const ServiceParameterUpdateBody = `
{
    "uuid": "75b3757d-4168-43c8-b921-5231de1955c9",
    "service": "bbq",
    "section": "brickets",
    "links": [
    {
      "href": "http://192.168.204.1:6385/v1/parameters/75b3757d-4168-43c8-b921-5231de1955c9",
      "rel": "self"
    },
    {
      "href": "http://192.168.204.1:6385/parameters/75b3757d-4168-43c8-b921-5231de1955c9",
      "rel": "bookmark"
    }
  ],
  "value": "disabled",
  "name": "charcoal",
  "resource": "bbq::brickets::charcoal::mode",
  "personality": null
}
`

func HandleServiceParameterListSuccessfully(t *testing.T) {
	th.Mux.HandleFunc("/service_parameter", func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "GET")
		th.TestHeader(t, r, "X-Auth-Token", client.TokenID)
		w.Header().Add("Content-Type", "application/json")
		fmt.Fprintf(w, ServiceParameterListBody)
	})
}

/* This mock function returns the GET for the ServiceParameter for berpUUID */
func HandleServiceParameterGetSuccessfully(t *testing.T) {
	th.Mux.HandleFunc("/service_parameter/"+berpUUID, func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "GET")
		th.TestHeader(t, r, "X-Auth-Token", client.TokenID)
		w.Header().Add("Content-Type", "application/json")
		fmt.Fprintf(w, SingleServiceParameterBody)
	})
}

/* This mock function returns the PATCH for the ServiceParameter for bbqUUID
This example is ONLY changing the 'value' to 'disabled' */
func HandleServiceParameterUpdateSuccessfully(t *testing.T) {
	th.Mux.HandleFunc("/service_parameter/"+bbqUUID, func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "PATCH")
		th.TestHeader(t, r, "X-Auth-Token", client.TokenID)
		th.TestHeader(t, r, "Accept", "application/json")
		th.TestHeader(t, r, "Content-Type", "application/json")
		th.TestJSONRequest(t, r, `[ { "op": "replace", "path": "/value", "value": "disabled" } ]`)
		fmt.Fprintf(w, ServiceParameterUpdateBody)
	})
}

func HandleServiceParameterDeletionSuccessfully(t *testing.T) {
	th.Mux.HandleFunc("/service_parameter/a5965fee-dc60-40dc-a234-edf87f1f9380", func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "DELETE")
		th.TestHeader(t, r, "X-Auth-Token", client.TokenID)

		w.WriteHeader(http.StatusNoContent)
	})
}

/* This mock function returns the POST for the ServiceParameter returning bbq object */
func HandleServiceParameterCreationSuccessfully(t *testing.T, response string) {
	th.Mux.HandleFunc("/service_parameter", func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "POST")
		th.TestHeader(t, r, "X-Auth-Token", client.TokenID)
		th.TestJSONRequest(t, r, `{
          		"parameters": {
            			"charcoal": "enabled"
          		},
          		"section": "brickets",
          		"service": "bbq",
          		"resource": "bbq::brickets::charcoal::mode"
        	}`)

		w.WriteHeader(http.StatusAccepted)
		w.Header().Add("Content-Type", "application/json")
		fmt.Fprintf(w, response)
	})
}
