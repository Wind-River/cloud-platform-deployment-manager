/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019 Wind River Systems, Inc. */

package testing

import (
	"fmt"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/certificates"
	"github.com/gophercloud/gophercloud/testhelper/client"
	"net/http"
	"testing"

	th "github.com/gophercloud/gophercloud/testhelper"
)

var (
	CertificateHerp = certificates.Certificate{
		ID:        "2a8fc6dc-a056-496e-8b08-65b0ae34b36f",
		Type:      "ssl_ca",
		Signature: "ssl_ca_10886226602156394257",
		CreatedAt: "",
		UpdatedAt: nil,
	}
	CertificateDerp = certificates.Certificate{
		ID:        "28710c00-d534-49b8-badd-30334d72de5c",
		Type:      "test",
		Signature: "test_10886226602156394257",
		CreatedAt: "",
		UpdatedAt: nil,
	}
)

const CertificateListBody = `
{
  "certificates": [
    {
      "certtype": "ssl_ca",
      "details": null,
      "expiry_date": "2021-06-05T20:28:20+00:00",
      "issuer": null,
      "signature": "ssl_ca_10886226602156394257",
      "start_date": "2018-08-16T20:28:20+00:00",
      "uuid": "2a8fc6dc-a056-496e-8b08-65b0ae34b36f"
    },
    {
      "certtype": "test",
      "details": null,
      "expiry_date": "2021-06-05T20:28:20+00:00",
      "issuer": null,
      "signature": "test_10886226602156394257",
      "start_date": "2018-08-16T20:28:20+00:00",
      "uuid": "28710c00-d534-49b8-badd-30334d72de5c"
    }
  ]
}
`

const CertificateSingleBody = `
{
  "certtype": "ssl_ca",
  "details": null,
  "expiry_date": "2021-06-05T20:28:20+00:00",
  "issuer": null,
  "signature": "ssl_ca_10886226602156394257",
  "start_date": "2018-08-16T20:28:20+00:00",
  "uuid": "2a8fc6dc-a056-496e-8b08-65b0ae34b36f"
}
`

const CertificateCreateResponse = `
{
  "certificates": {
  "certtype": "ssl_ca",
  "details": null,
  "expiry_date": "2021-06-05T20:28:20+00:00",
  "issuer": null,
  "signature": "ssl_ca_10886226602156394257",
  "start_date": "2018-08-16T20:28:20+00:00",
  "uuid": "2a8fc6dc-a056-496e-8b08-65b0ae34b36f"
  },
  "error": null,
  "body": null
}
`

func HandleCertificateListSuccessfully(t *testing.T) {
	th.Mux.HandleFunc("/certificate", func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "GET")
		th.TestHeader(t, r, "X-Auth-Token", client.TokenID)
		w.Header().Add("Content-Type", "application/json")
		fmt.Fprintf(w, CertificateListBody)
	})
}

func HandleCertificateGetSuccessfully(t *testing.T) {
	th.Mux.HandleFunc("/certificate/f757b5c7-89ab-4d93-bfd7-a97780ec2c1e", func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "GET")
		th.TestHeader(t, r, "X-Auth-Token", client.TokenID)
		w.Header().Add("Content-Type", "application/json")
		fmt.Fprintf(w, CertificateSingleBody)
	})
}

func HandleCertificateCreationSuccessfully(t *testing.T, response string) {
	th.Mux.HandleFunc("/certificate/certificate_install", func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "POST")
		th.TestHeader(t, r, "X-Auth-Token", client.TokenID)
		th.TestMultipartRequest(t, r, `--f8cf998387ef2df13f3169e0aa11bd03c5e1487f1df63ddca5552f1bb854
Content-Disposition: form-data; name="mode"

ssl_ca
--f8cf998387ef2df13f3169e0aa11bd03c5e1487f1df63ddca5552f1bb854
Content-Disposition: form-data; name="file"; filename="certificate.pem"
Content-Type: application/octet-stream

foobar
--f8cf998387ef2df13f3169e0aa11bd03c5e1487f1df63ddca5552f1bb854--
`)

		w.WriteHeader(http.StatusAccepted)
		w.Header().Add("Content-Type", "application/json")
		fmt.Fprintf(w, response)
	})
}
