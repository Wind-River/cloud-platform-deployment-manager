/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019-2024 Wind River Systems, Inc. */

package common

import (
	"reflect"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Common utils", func() {
	Describe("IsIPv4 utility", func() {
		Context("with address data", func() {
			It("should determin IPv4 address", func() {
				type args struct {
					address string
				}
				tests := []struct {
					name string
					args args
					want bool
				}{
					{name: "any",
						args: args{address: "0.0.0.0"},
						want: true},
					{name: "normal",
						args: args{address: "1.2.3.4"},
						want: true},
					{name: "ipv6-to-ipv4-mapped",
						args: args{address: "::ffff:1.2.3.4"},
						want: true},
					{name: "invalid-fqdn",
						args: args{address: "a.b.c.d"},
						want: false},
					{name: "invalid-too-many-octets",
						args: args{address: "1.2.3.4.5"},
						want: false},
					{name: "invalid-ipv6",
						args: args{address: "fd00::1"},
						want: false},
				}
				for _, tt := range tests {
					got := IsIPv4(tt.args.address)
					Expect(got).To(Equal(tt.want))
				}
			})
		})
	})

	Describe("IsIPv6 utility", func() {
		Context("with address data", func() {
			It("should determin IPv6 address", func() {
				type args struct {
					address string
				}
				tests := []struct {
					name string
					args args
					want bool
				}{
					{name: "any",
						args: args{address: "::"},
						want: true},
					{name: "normal",
						args: args{address: "fd00::1:2:3:4"},
						want: true},
					{name: "ipv6-to-ipv4-mapped",
						args: args{address: "::ffff:1.2.3.4"},
						want: false},
					{name: "invalid-fqdn",
						args: args{address: "a.b.c.d"},
						want: false},
					{name: "invalid-too-many-octets",
						args: args{address: "a:b:c:d:e:f:g:h:i"},
						want: false},
					{name: "invalid-too-many-expansions",
						args: args{address: "fd00::1::2"},
						want: false},
					{name: "invalid-ipv6",
						args: args{address: "fd00::asdf"},
						want: false},
				}
				for _, tt := range tests {
					got := IsIPv6(tt.args.address)
					Expect(got).To(Equal(tt.want))
				}
			})
		})
	})

	Describe("ListDelta utility", func() {
		Context("with two lists", func() {
			It("should calculate the difference", func() {
				empty := make([]string, 0)
				type args struct {
					a []string
					b []string
				}
				tests := []struct {
					name        string
					args        args
					wantAdded   []string
					wantRemoved []string
					wantSame    []string
				}{
					{name: "add_first",
						args: args{a: []string{},
							b: []string{"1"}},
						wantAdded:   []string{"1"},
						wantRemoved: empty,
						wantSame:    empty},
					{name: "add_second",
						args: args{a: []string{"2"},
							b: []string{"1", "2"}},
						wantAdded:   []string{"1"},
						wantRemoved: empty,
						wantSame:    []string{"2"}},
					{name: "add_multiple",
						args: args{a: []string{"2"},
							b: []string{"1", "2", "3", "4"}},
						wantAdded:   []string{"1", "3", "4"},
						wantRemoved: empty,
						wantSame:    []string{"2"}},
					{name: "remove_last",
						args: args{a: []string{"1"},
							b: []string{}},
						wantAdded:   empty,
						wantRemoved: []string{"1"},
						wantSame:    empty},
					{name: "remove_one",
						args: args{a: []string{"1", "2"},
							b: []string{"2"}},
						wantAdded:   empty,
						wantRemoved: []string{"1"},
						wantSame:    []string{"2"}},
					{name: "remove_multiple",
						args: args{a: []string{"1", "2", "3", "4"},
							b: []string{"2"}},
						wantAdded:   empty,
						wantRemoved: []string{"1", "3", "4"},
						wantSame:    []string{"2"}},
					{name: "identical",
						args: args{a: []string{"1"},
							b: []string{"1"}},
						wantAdded:   empty,
						wantRemoved: empty,
						wantSame:    []string{"1"}},
				}

				for _, tt := range tests {
					gotAdded, gotRemoved, gotSame := ListDelta(tt.args.a, tt.args.b)
					Expect(reflect.DeepEqual(gotAdded, tt.wantAdded)).To(BeTrue())
					Expect(reflect.DeepEqual(gotRemoved, tt.wantRemoved)).To(BeTrue())
					Expect(reflect.DeepEqual(gotSame, tt.wantSame)).To(BeTrue())
				}
			})
		})
	})

	Describe("ListChanged utility", func() {
		Context("with two string arrays", func() {
			It("should identify changed", func() {
				type args struct {
					a []string
					b []string
				}
				tests := []struct {
					name string
					args args
					want bool
				}{
					{name: "compare_with_empty",
						args: args{a: []string{},
							b: []string{"1"}},
						want: true},
					{name: "compare_with_non_empty",
						args: args{a: []string{"2"},
							b: []string{"1", "2"}},
						want: true},
					{name: "compare_to_empty",
						args: args{a: []string{"1"},
							b: []string{}},
						want: true},
					{name: "compare_to_non_empty",
						args: args{a: []string{"1", "2"},
							b: []string{"2"}},
						want: true},
					{name: "identical",
						args: args{a: []string{"1"},
							b: []string{"1"}},
						want: false},
				}
				for _, tt := range tests {
					got := ListChanged(tt.args.a, tt.args.b)
					Expect(got).To(Equal(tt.want))
				}
			})
		})
	})

	Describe("ListIntersect utility", func() {
		Context("with two string arrays", func() {
			It("should identify commonality", func() {
				type args struct {
					a []string
					b []string
				}
				tests := []struct {
					name  string
					args  args
					want  []string
					want1 bool
				}{
					{name: "same",
						args: args{a: []string{"a", "b"},
							b: []string{"b", "a"}},
						want:  []string{"a", "b"},
						want1: true},
					{name: "empty",
						args: args{a: []string{},
							b: []string{}},
						want:  []string{},
						want1: false},
					{name: "intersect",
						args: args{a: []string{"a", "b"},
							b: []string{"b", "c"}},
						want:  []string{"b"},
						want1: true},
					{name: "no-intersect",
						args: args{a: []string{"a", "b"},
							b: []string{"c", "d"}},
						want:  []string{},
						want1: false},
				}
				for _, tt := range tests {
					got, got1 := ListIntersect(tt.args.a, tt.args.b)
					Expect(reflect.DeepEqual(got, tt.want)).To(BeTrue())
					Expect(got1).To(Equal(tt.want1))
				}
			})
		})
	})

	Describe("ComparePartitionPaths utility", func() {
		Context("with two string", func() {
			It("should identify disk portion matched", func() {
				type args struct {
					a string
					b string
				}
				tests := []struct {
					name string
					args args
					want bool
				}{
					{name: "same-disks",
						args: args{a: "/dev/disk/by-path/pci-0000:00:1f.2-ata-5.0",
							b: "/dev/disk/by-path/pci-0000:00:1f.2-ata-5.0"},
						want: true},
					{name: "different-disks",
						args: args{a: "/dev/disk/by-path/pci-0000:00:1f.2-ata-5.0",
							b: "/dev/disk/by-path/pci-0000:00:1f.3-ata-6.0"},
						want: false},
					{name: "same-disk-partitions",
						args: args{a: "/dev/disk/by-path/pci-0000:00:1f.2-ata-5.0-part1",
							b: "/dev/disk/by-path/pci-0000:00:1f.2-ata-5.0-part1"},
						want: true},
					{name: "different-disks-partitions",
						args: args{a: "/dev/disk/by-path/pci-0000:00:1f.2-ata-5.0-part1",
							b: "/dev/disk/by-path/pci-0000:00:1f.3-ata-6.0-part1"},
						want: false},
					{name: "different-partitions",
						args: args{a: "/dev/disk/by-path/pci-0000:00:1f.2-ata-5.0-part1",
							b: "/dev/disk/by-path/pci-0000:00:1f.2-ata-5.0-part2"},
						want: true},
				}
				for _, tt := range tests {
					got := ComparePartitionPaths(tt.args.a, tt.args.b)
					Expect(got).To(Equal(tt.want))
				}
			})
		})
	})

	Describe("ContainsString utility", func() {
		Context("with two string", func() {
			It("should identify contained", func() {
				type args struct {
					slice []string
					s     string
				}
				tests := []struct {
					name string
					args args
					want bool
				}{
					{name: "included",
						args: args{slice: []string{"abc", "def"},
							s: "abc"},
						want: true},
					{name: "not-included",
						args: args{slice: []string{"abc", "def"},
							s: "xyz"},
						want: false},
					{name: "substring-not-included",
						args: args{slice: []string{"abc", "def"},
							s: "a"},
						want: false},
				}
				for _, tt := range tests {
					got := ContainsString(tt.args.slice, tt.args.s)
					Expect(got).To(Equal(tt.want))
				}
			})
		})
	})

	Describe("RemoveString utility", func() {
		Context("with string and list", func() {
			It("should remove string from list", func() {
				type args struct {
					slice []string
					s     string
				}
				tests := []struct {
					name       string
					args       args
					wantResult []string
				}{
					{name: "included",
						args: args{slice: []string{"abc", "def"},
							s: "abc"},
						wantResult: []string{"def"}},
					{name: "not-included",
						args: args{slice: []string{"abc", "def"},
							s: "xyz"},
						wantResult: []string{"abc", "def"}},
					{name: "substring-not-included",
						args: args{slice: []string{"abc", "def"},
							s: "a"},
						wantResult: []string{"abc", "def"}},
				}
				for _, tt := range tests {
					gotResult := RemoveString(tt.args.slice, tt.args.s)
					Expect(reflect.DeepEqual(gotResult, tt.wantResult)).To(BeTrue())
				}
			})
		})
	})

	Describe("DedupeSlice utility", func() {
		Context("with a slice with duplicates", func() {
			It("should remove the string duplicates", func() {
				stringTests := []struct {
					name       string
					given      []string
					wantResult []string
				}{
					{name: "one string duplicate",
						given:      []string{"foo0", "foo1", "foo1", "foo2", "foo3"},
						wantResult: []string{"foo0", "foo1", "foo2", "foo3"},
					},
					{name: "two string duplicates",
						given:      []string{"foo0", "foo1", "foo1", "foo2", "foo2"},
						wantResult: []string{"foo0", "foo1", "foo2"},
					},
				}
				for _, tt := range stringTests {
					gotResult := DedupeSlice(tt.given)
					Expect(reflect.DeepEqual(gotResult, tt.wantResult)).To(BeTrue())
				}
			})
			It("should remove the int duplicates", func() {
				intTests := []struct {
					name       string
					given      []int
					wantResult []int
				}{
					{name: "one int duplicate",
						given:      []int{101, 202, 303, 404, 303},
						wantResult: []int{101, 202, 303, 404},
					},
					{name: "two int duplicates",
						given:      []int{101, 101, 202, 303, 404, 101},
						wantResult: []int{101, 202, 303, 404},
					},
				}
				for _, tt := range intTests {
					gotResult := DedupeSlice(tt.given)
					Expect(reflect.DeepEqual(gotResult, tt.wantResult)).To(BeTrue())
				}
			})
		})
	})
})
