/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2024-2026 Wind River Systems, Inc. */

package common

import (
	"errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
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

	It("should create a NewUnlockError with simple message", func() {
		expectedMessage := "Failed to unlock controller-0: Kernel upgrade in progress"
		want := ErrUnlockError{BaseError{expectedMessage}}
		reason := errors.New("Kernel upgrade in progress")
		got := NewUnlockError("controller-0", reason)
		Expect(got).To(Equal(want))
	})

	It("should extract fault string reason from NewUnlockError", func() {
		want := ErrUnlockError{BaseError{"Failed to unlock controller-0: Kernel upgrade in progress"}}
		reason := errors.New(`Some kind of error codes, faultstring\\: \\\\\"Kernel upgrade in progress\\\", more error codes here`)
		got := NewUnlockError("controller-0", reason)
		Expect(got).To(Equal(want))
	})

	It("should return the error message", func() {
		msg := "message"
		baseErr := BaseError{msg}
		want := msg
		got := baseErr.Error()
		Expect(got).To(Equal(want))
	})
})
