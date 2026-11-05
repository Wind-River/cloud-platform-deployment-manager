/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2026 Wind River Systems, Inc. */

package common

import (
	"errors"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/starlingx"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	perrors "github.com/pkg/errors"
)

var _ = Describe("API error utilities", func() {
	body := []byte(`{"error_message": "{\"debuginfo\": null, \"faultcode\": \"Client\", \"faultstring\": \"Rejected: host not ready\"}"}`)

	// fault builds the StarlingX FaultError that gophercloud returns for a
	// rejected (non 2xx) StarlingX request, regardless of the status code.
	fault := func(code int) error {
		return starlingx.FaultError{
			ErrUnexpectedResponseCode: gophercloud.ErrUnexpectedResponseCode{
				Method: "PATCH", URL: "http://sysinv/v1/ihosts/1", Actual: code, Body: body,
			},
		}
	}

	DescribeTable("should extract the response from a rejected request, wrapped or not",
		func(in error, code int) {
			Expect(IsAPIResponseError(in)).To(BeTrue())
			Expect(APIErrorStatusCode(in)).To(Equal(code))
			Expect(APIErrorReason(in)).To(Equal("Rejected: host not ready"))
		},
		Entry("400", fault(400), 400),
		Entry("403", fault(403), 403),
		Entry("404", fault(404), 404),
		Entry("405", fault(405), 405),
		Entry("408", fault(408), 408),
		Entry("409", fault(409), 409),
		Entry("422", fault(422), 422),
		Entry("429", fault(429), 429),
		Entry("500", fault(500), 500),
		Entry("503", fault(503), 503),
		Entry("wrapped 409",
			perrors.Wrap(fault(409), "failed to create route"), 409),
	)

	It("should report the wrap annotation as the operation", func() {
		in := perrors.Wrap(fault(409), "failed to create route")
		Expect(APIErrorOperation(in)).To(Equal("failed to create route"))
	})

	It("should report no operation for an unwrapped response error", func() {
		Expect(APIErrorOperation(fault(400))).To(Equal(""))
	})

	DescribeTable("should not treat other errors as response errors",
		func(in error) {
			Expect(IsAPIResponseError(in)).To(BeFalse())
			Expect(APIErrorStatusCode(in)).To(Equal(0))
		},
		Entry("nil", nil),
		Entry("plain error", errors.New("boom")),
		Entry("gophercloud error without a response", gophercloud.ErrTimeOut{}),
		Entry("re-authentication failure", &gophercloud.ErrUnableToReauthenticate{}),
	)

	It("should return the plain error string as the reason for a non-response error", func() {
		Expect(APIErrorReason(errors.New("boom"))).To(Equal("boom"))
		Expect(APIErrorReason(nil)).To(Equal(""))
	})
})
