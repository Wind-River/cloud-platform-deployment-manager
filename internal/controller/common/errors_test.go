/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2024-2025 Wind River Systems, Inc. */

package common

import (
	"errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Test Build utilities:", func() {
	It("Test NewSystemDependency", func() {
		msg := "message"
		want := ErrSystemDependency{BaseError{msg}}
		got := NewSystemDependency(msg)
		Expect(got).To(Equal(want))
	})

	It("Test NewMissingKubernetesResource", func() {
		msg := "message"
		want := ErrMissingKubernetesResource{BaseError{msg}}
		got := NewMissingKubernetesResource(msg)
		Expect(got).To(Equal(want))
	})

	It("Test NewResourceStatusDependency", func() {
		msg := "message"
		want := ErrResourceStatusDependency{BaseError{msg}}
		got := NewResourceStatusDependency(msg)
		Expect(got).To(Equal(want))
	})

	It("Test NewResourceConfigurationDependency", func() {
		msg := "message"
		want := ErrResourceConfigurationDependency{BaseError{msg}}
		got := NewResourceConfigurationDependency(msg)
		Expect(got).To(Equal(want))
	})

	It("Test NewUserDataError", func() {
		msg := "message"
		want := ErrUserDataError{BaseError{msg}}
		got := NewUserDataError(msg)
		Expect(got).To(Equal(want))
	})

	It("Test NewValidationError", func() {
		msg := "message"
		want := ValidationError{BaseError{msg}}
		got := NewValidationError(msg)
		Expect(got).To(Equal(want))
	})

	It("Test NewHTTPSClientRequired", func() {
		msg := "message"
		want := HTTPSClientRequired{BaseError{msg}}
		got := NewHTTPSClientRequired(msg)
		Expect(got).To(Equal(want))
	})

	It("Test NewChangeAfterInSync", func() {
		msg := "message"
		want := ChangeAfterReconciled{BaseError{msg}}
		got := NewChangeAfterInSync(msg)
		Expect(got).To(Equal(want))
	})

	It("Test NewUnlockError with simple message", func() {
		expectedMessage := "Failed to unlock controller-0: Kernel upgrade in progress"
		want := ErrUnlockError{BaseError{expectedMessage}}
		reason := errors.New("Kernel upgrade in progress")
		got := NewUnlockError("controller-0", reason)
		Expect(got).To(Equal(want))
	})

	It("Test NewUnlockError extract fault string reason", func() {
		want := ErrUnlockError{BaseError{"Failed to unlock controller-0: Kernel upgrade in progress"}}
		reason := errors.New(`Some kind of error codes, faultstring\\: \\\\\"Kernel upgrade in progress\\\", more error codes here`)
		got := NewUnlockError("controller-0", reason)
		Expect(got).To(Equal(want))
	})

	It("Test Error", func() {
		msg := "message"
		baseErr := BaseError{msg}
		want := msg
		got := baseErr.Error()
		Expect(got).To(Equal(want))
	})
})
