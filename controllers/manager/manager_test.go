/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019-2022, 2025 Wind River Systems, Inc. */

package manager

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Manager utils", func() {
	Describe("getNextCount utility", func() {
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
