/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2024-2026 Wind River Systems, Inc. */

package common

import (
	"encoding/json"
	"errors"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/starlingx"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	perrors "github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var _ = Describe("Test Build utilities:", func() {
	It("should create a NewSystemDependency", func() {
		msg := "message"
		want := ErrSystemDependency{BaseError{msg}}
		got := NewSystemDependency(msg)
		Expect(got).To(Equal(want))
	})

	It("should create a NewMissingKubernetesResource", func() {
		msg := "message"
		want := ErrMissingKubernetesResource{BaseError{msg}}
		got := NewMissingKubernetesResource(msg)
		Expect(got).To(Equal(want))
	})

	It("should create a NewResourceStatusDependency", func() {
		msg := "message"
		want := ErrResourceStatusDependency{BaseError{msg}}
		got := NewResourceStatusDependency(msg)
		Expect(got).To(Equal(want))
	})

	It("should create a NewResourceConfigurationDependency", func() {
		msg := "message"
		want := ErrResourceConfigurationDependency{BaseError{msg}}
		got := NewResourceConfigurationDependency(msg)
		Expect(got).To(Equal(want))
	})

	It("should create a NewUserDataError", func() {
		msg := "message"
		want := ErrUserDataError{BaseError{msg}}
		got := NewUserDataError(msg)
		Expect(got).To(Equal(want))
	})

	It("should create a NewValidationError", func() {
		msg := "message"
		want := ValidationError{BaseError{msg}}
		got := NewValidationError(msg)
		Expect(got).To(Equal(want))
	})

	It("should create a NewHTTPSClientRequired", func() {
		msg := "message"
		want := HTTPSClientRequired{BaseError{msg}}
		got := NewHTTPSClientRequired(msg)
		Expect(got).To(Equal(want))
	})

	It("should create a NewChangeAfterInSync", func() {
		msg := "message"
		want := ChangeAfterReconciled{BaseError{msg}}
		got := NewChangeAfterInSync(msg)
		Expect(got).To(Equal(want))
	})

	Describe("ErrSystemAPI", func() {
		// sysinvError mimics the error gophercloud returns for a rejected
		// StarlingX request: a starlingx.FaultError wrapping the response,
		// whose body is produced by sysinv-api's ParsableErrorMiddleware
		// wrapping a WSME fault.  The FaultError already extracts the
		// faultstring and strips any remote traceback.
		sysinvError := func(code int, faultcode, faultstring string) starlingx.FaultError {
			inner, _ := json.Marshal(map[string]interface{}{
				"faultcode": faultcode, "faultstring": faultstring, "debuginfo": nil})
			body, _ := json.Marshal(map[string]string{"error_message": string(inner)})
			return starlingx.FaultError{ErrUnexpectedResponseCode: gophercloud.ErrUnexpectedResponseCode{
				Method: "PATCH", URL: "http://sysinv/v1/ihosts/1", Actual: code, Body: body}}
		}

		It("should keep a simple message", func() {
			got := NewSystemAPIError("unlock host controller-0", errors.New("Kernel upgrade in progress"))
			Expect(got.Error()).To(Equal("failed to unlock host controller-0: Kernel upgrade in progress"))
			Expect(got.StatusCode).To(Equal(0))
		})

		It("should use the plain message of a non response error as the reason", func() {
			reason := errors.New("Kernel upgrade in progress")
			got := NewSystemAPIError("unlock host controller-0", reason)
			Expect(got.StatusCode).To(Equal(0))
			Expect(got.Reason).To(Equal("Kernel upgrade in progress"))
		})

		It("should report a missing response", func() {
			got := NewSystemAPIError("unlock host controller-0", nil)
			Expect(got.Reason).To(Equal("no valid response received from the system API"))
			Expect(got.Retry()).To(Equal(RetryTransientError))
		})

		It("should parse a sysinv rejection", func() {
			in := sysinvError(400, "Client", "Operation Rejected: host not ready. unlock.")
			got := NewSystemAPIError("unlock host controller-0", in)
			Expect(got.StatusCode).To(Equal(400))
			Expect(got.Reason).To(Equal("Operation Rejected: host not ready. unlock."))
			Expect(got.Error()).To(Equal(
				"failed to unlock host controller-0: Operation Rejected: host not ready. unlock. (HTTP 400)"))
			var original starlingx.FaultError
			Expect(errors.As(got, &original)).To(BeTrue())
			Expect(original.Actual).To(Equal(400))
		})

		It("should strip remote tracebacks from server errors", func() {
			in := sysinvError(500, "Server",
				"Failed to apply\nTraceback (most recent call last):\n  File \"x.py\"\nSysinvException: x\n")
			got := NewSystemAPIError("", perrors.Wrap(in, "failed to create route"))
			Expect(got.Reason).To(Equal("Failed to apply"))
			Expect(got.Operation).To(Equal("failed to create route"))
			Expect(got.Error()).To(Equal("failed to create route: Failed to apply (HTTP 500)"))
		})

		It("should derive the retry policy from the status code", func() {
			retry := func(code int) reconcile.Result {
				return NewSystemAPIError("", sysinvError(code, "Client", "x")).Retry()
			}
			Expect(retry(400)).To(Equal(RetryUserError))
			Expect(retry(404)).To(Equal(RetryUserError))
			Expect(retry(409)).To(Equal(RetryUserError))
			Expect(retry(408)).To(Equal(RetryTransientError))
			Expect(retry(429)).To(Equal(RetryTransientError))
			Expect(retry(500)).To(Equal(RetryServerError))
			Expect(retry(502)).To(Equal(RetryServerError))
		})

		It("should allow the retry policy to be overridden", func() {
			in := sysinvError(400, "Client", "x")
			Expect(NewSystemAPIError("unlock host controller-0", in).Retry()).To(Equal(RetryUserError))
			got := NewSystemAPIError("unlock host controller-0", in).WithRetry(RetryTransientError)
			Expect(got.Retry()).To(Equal(RetryTransientError))
		})

		It("should not wrap an existing ErrSystemAPI twice", func() {
			first := NewSystemAPIError("create route", sysinvError(409, "Client", "x"))
			got := NewSystemAPIError("", perrors.Wrap(first, "reconcile routes"))
			Expect(got.Operation).To(Equal("create route"))
			Expect(got.StatusCode).To(Equal(409))
		})
	})

	It("should return the error message", func() {
		msg := "message"
		baseErr := BaseError{msg}
		want := msg
		got := baseErr.Error()
		Expect(got).To(Equal(want))
	})
})
