/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2026 Wind River Systems, Inc. */

package common

import (
	"errors"

	"github.com/gophercloud/gophercloud"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	perrors "github.com/pkg/errors"
)

var _ = Describe("API error utilities", func() {
	body := []byte(`{"error_message": "{\"debuginfo\": null, \"faultcode\": \"Client\", \"faultstring\": \"Rejected: host not ready\"}"}`)
	resp := func(code int) gophercloud.ErrUnexpectedResponseCode {
		return gophercloud.ErrUnexpectedResponseCode{
			Method: "PATCH", URL: "http://sysinv/v1/ihosts/1", Actual: code, Body: body}
	}

	DescribeTable("should extract the response from every gophercloud response error type",
		func(in error, code int) {
			Expect(IsAPIResponseError(in)).To(BeTrue())
			Expect(APIErrorStatusCode(in)).To(Equal(code))
			Expect(APIErrorReason(in)).To(Equal("Rejected: host not ready"))
		},
		Entry("ErrUnexpectedResponseCode", resp(422), 422),
		Entry("*ErrUnexpectedResponseCode", func() error { r := resp(502); return &r }(), 502),
		Entry("ErrDefault400", gophercloud.ErrDefault400{ErrUnexpectedResponseCode: resp(400)}, 400),
		Entry("ErrDefault401", gophercloud.ErrDefault401{ErrUnexpectedResponseCode: resp(401)}, 401),
		Entry("ErrDefault403", gophercloud.ErrDefault403{ErrUnexpectedResponseCode: resp(403)}, 403),
		Entry("ErrDefault404", gophercloud.ErrDefault404{ErrUnexpectedResponseCode: resp(404)}, 404),
		Entry("ErrDefault405", gophercloud.ErrDefault405{ErrUnexpectedResponseCode: resp(405)}, 405),
		Entry("ErrDefault408", gophercloud.ErrDefault408{ErrUnexpectedResponseCode: resp(408)}, 408),
		Entry("ErrDefault409", gophercloud.ErrDefault409{ErrUnexpectedResponseCode: resp(409)}, 409),
		Entry("ErrDefault429", gophercloud.ErrDefault429{ErrUnexpectedResponseCode: resp(429)}, 429),
		Entry("ErrDefault500", gophercloud.ErrDefault500{ErrUnexpectedResponseCode: resp(500)}, 500),
		Entry("ErrDefault503", gophercloud.ErrDefault503{ErrUnexpectedResponseCode: resp(503)}, 503),
		Entry("wrapped ErrDefault409",
			perrors.Wrap(gophercloud.ErrDefault409{ErrUnexpectedResponseCode: resp(409)}, "failed to create route"), 409),
	)

	DescribeTable("should not treat other errors as response errors",
		func(in error) {
			Expect(IsAPIResponseError(in)).To(BeFalse())
			Expect(APIErrorStatusCode(in)).To(Equal(0))
		},
		Entry("nil", nil),
		Entry("plain error", errors.New("boom")),
		Entry("nil *ErrUnexpectedResponseCode", (*gophercloud.ErrUnexpectedResponseCode)(nil)),
		Entry("gophercloud error without a response", gophercloud.ErrTimeOut{}),
		Entry("re-authentication failure", &gophercloud.ErrUnableToReauthenticate{}),
	)
})
