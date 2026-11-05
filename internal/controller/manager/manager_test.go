/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019-2022, 2025-2026 Wind River Systems, Inc. */

package manager

import (
	"errors"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/starlingx"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Manager utils", func() {
	Describe("Function isVimAuthError", func() {
		fault := func(code int) error {
			return starlingx.FaultError{ErrUnexpectedResponseCode: gophercloud.ErrUnexpectedResponseCode{
				Method: "POST", URL: "http://vim/nfvi/v1", Actual: code}}
		}

		It("should detect a 403 returned as a FaultError (token expiry)", func() {
			Expect(isVimAuthError(fault(403))).To(BeTrue())
		})

		It("should detect a plain 401 that was not routed through the fault context", func() {
			Expect(isVimAuthError(gophercloud.ErrDefault401{})).To(BeTrue())
		})

		It("should not treat other errors as auth errors", func() {
			Expect(isVimAuthError(nil)).To(BeFalse())
			Expect(isVimAuthError(errors.New("boom"))).To(BeFalse())
			Expect(isVimAuthError(fault(409))).To(BeFalse())
		})
	})

	Describe("Function getNextCount", func() {
		Context("with a count value", func() {
			It("should return the next value", func() {
				type args struct {
					value string
				}
				tests := []struct {
					name string
					args args
					want string
				}{
					{name: "empty value",
						args: args{value: ""},
						want: "1",
					},
					{name: "integer value",
						args: args{value: "1"},
						want: "2"},
					{name: "alphanumeric value",
						args: args{value: "foobar"},
						want: "1"},
				}
				for _, tt := range tests {
					got := getNextCount(tt.args.value)
					Expect(got).To(Equal(tt.want))
				}
			})
		})
	})
})
