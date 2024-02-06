/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019 Wind River Systems, Inc. */

package common

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Test Build utilities:", func() {
	Describe("Test NewSystemDependency", func() {
		msg := "message"
		want := ErrSystemDependency{BaseError{msg}}
		got := NewSystemDependency(msg)
		Expect(got).To(Equal(want))
	})
	Describe("Test NewMissingKubernetesResource", func() {
		msg := "message"
		want := ErrMissingKubernetesResource{BaseError{msg}}
		got := NewMissingKubernetesResource(msg)
		Expect(got).To(Equal(want))
	})
	Describe("Test NewResourceStatusDependency", func() {
		msg := "message"
		want := ErrResourceStatusDependency{BaseError{msg}}
		got := NewResourceStatusDependency(msg)
		Expect(got).To(Equal(want))
	})
	Describe("Test NewResourceConfigurationDependency", func() {
		msg := "message"
		want := ErrResourceConfigurationDependency{BaseError{msg}}
		got := NewResourceConfigurationDependency(msg)
		Expect(got).To(Equal(want))
	})
	Describe("Test NewUserDataError", func() {
		msg := "message"
		want := ErrUserDataError{BaseError{msg}}
		got := NewUserDataError(msg)
		Expect(got).To(Equal(want))
	})
	Describe("Test NewValidationError", func() {
		msg := "message"
		want := ValidationError{BaseError{msg}}
		got := NewValidationError(msg)
		Expect(got).To(Equal(want))
	})
	Describe("Test NewHTTPSClientRequired", func() {
		msg := "message"
		want := HTTPSClientRequired{BaseError{msg}}
		got := NewHTTPSClientRequired(msg)
		Expect(got).To(Equal(want))
	})
	Describe("Test NewChangeAfterInSync", func() {
		msg := "message"
		want := ChangeAfterReconciled{BaseError{msg}}
		got := NewChangeAfterInSync(msg)
		Expect(got).To(Equal(want))
	})
	Describe("Test Error", func() {
		msg := "message"
		baseErr := BaseError{msg}
		want := msg
		got := baseErr.Error()
		Expect(got).To(Equal(want))
	})
})
