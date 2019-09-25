/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019 Wind River Systems, Inc. */

package testing

import (
	"fmt"
	"github.com/gophercloud/gophercloud/testhelper/client"
	"net/http"
	"testing"

	th "github.com/gophercloud/gophercloud/testhelper"
)

var license1 = "0123456789abcdefghijklmnopqrstuvwxyz"

const LicenseCreateResponse = `
{
  "success": "Success: new license installed",
  "error": null,
  "body": null
}
`

const LicenseGetResponse = `
{
  "content": "0123456789abcdefghijklmnopqrstuvwxyz",
  "error": null
}
`

func HandleLicenseCreationSuccessfully(t *testing.T, response string) {
	th.Mux.HandleFunc("/license/license_install", func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "POST")
		th.TestHeader(t, r, "X-Auth-Token", client.TokenID)
		th.TestMultipartRequest(t, r, `--f8cf998387ef2df13f3169e0aa11bd03c5e1487f1df63ddca5552f1bb854
Content-Disposition: form-data; name="file"; filename="license.lic"
Content-Type: application/octet-stream

0123456789abcdefghijklmnopqrstuvwxyz
--f8cf998387ef2df13f3169e0aa11bd03c5e1487f1df63ddca5552f1bb854--
`)

		w.WriteHeader(http.StatusAccepted)
		w.Header().Add("Content-Type", "application/json")
		fmt.Fprintf(w, response)
	})
}

func HandleLicenseGetSuccessfully(t *testing.T) {
	th.Mux.HandleFunc("/license/get_license_file", func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "GET")
		th.TestHeader(t, r, "X-Auth-Token", client.TokenID)
		w.Header().Add("Content-Type", "application/json")
		fmt.Fprintf(w, LicenseGetResponse)
	})
}
