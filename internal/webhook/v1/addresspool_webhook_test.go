/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2024-2026 Wind River Systems, Inc. */
package v1

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	starlingxv1 "github.com/wind-river/cloud-platform-deployment-manager/api/v1"
)

func GetAddrPool(ip_family string) *starlingxv1.AddressPool {
	subnet := "192.168.204.0"
	floating_address := "192.168.204.2"
	controller0_address := "192.168.204.3"
	controller1_address := "192.168.204.4"
	gateway := "192.168.204.1"
	range_start := "192.168.204.2"
	range_end := "192.168.204.254"
	if ip_family == "ipv6" {
		subnet = "fcff:1:2:3::0"
		floating_address = "fcff:1:2:3::2"
		controller0_address = "fcff:1:2:3::3"
		controller1_address = "fcff:1:2:3::4"
		gateway = "fcff:1:2:3::1"
		range_start = "fcff:1:2:3::1"
		range_end = "fcff:1:2:3::1000"
	}
	allocation_order := "random"

	return &starlingxv1.AddressPool{
		Spec: starlingxv1.AddressPoolSpec{
			Subnet:             subnet,
			FloatingAddress:    &floating_address,
			Controller0Address: &controller0_address,
			Controller1Address: &controller1_address,
			Prefix:             24,
			Gateway:            &gateway,
			Allocation: starlingxv1.AllocationInfo{
				Order:  &allocation_order,
				Ranges: []starlingxv1.AllocationRange{{Start: range_start, End: range_end}},
			},
		},
	}
}

var _ = Describe("AddressPoolWebhook", func() {

	Describe("ValidateAddressPool", func() {
		Context("when IPv4 data is correct", func() {
			It("should validate successfully without error", func() {
				r := GetAddrPool("ipv4")
				err := validateAddressPool(r)
				Expect(err).ToNot(HaveOccurred())
			})
		})

		Context("when IPv6 data is correct", func() {
			It("should validate successfully without error", func() {
				r := GetAddrPool("ipv6")
				err := validateAddressPool(r)
				Expect(err).ToNot(HaveOccurred())
			})
		})

		Context("when subnet is not valid IPv4 or IPv6", func() {
			It("should fail validation with error", func() {
				r := GetAddrPool("ipv4")
				invalid_ip := "not-valid-ipv4-ipv6"
				r.Spec.Subnet = invalid_ip
				err := validateAddressPool(r)
				Expect(err).Should(HaveOccurred())
			})
		})

		Context("when gateway is not valid IPv4 or IPv6", func() {
			It("should fail validation with error", func() {
				r := GetAddrPool("ipv4")
				invalid_ip := "not-valid-ipv4-ipv6"
				r.Spec.Gateway = &invalid_ip
				err := validateAddressPool(r)
				Expect(err).Should(HaveOccurred())
			})
		})

		Context("when FloatingAddress is not valid IPv4 or IPv6", func() {
			It("should fail validation with error", func() {
				r := GetAddrPool("ipv4")
				invalid_ip := "not-valid-ipv4-ipv6"
				r.Spec.FloatingAddress = &invalid_ip
				err := validateAddressPool(r)
				Expect(err).Should(HaveOccurred())
			})
		})

		Context("when Controller0Address is not valid IPv4 or IPv6", func() {
			It("should fail validation with error", func() {
				r := GetAddrPool("ipv4")
				invalid_ip := "not-valid-ipv4-ipv6"
				r.Spec.Controller0Address = &invalid_ip
				err := validateAddressPool(r)
				Expect(err).Should(HaveOccurred())
			})
		})

		Context("when Controller1Address is not valid IPv4 or IPv6", func() {
			It("should fail validation with error", func() {
				r := GetAddrPool("ipv4")
				invalid_ip := "not-valid-ipv4-ipv6"
				r.Spec.Controller1Address = &invalid_ip
				err := validateAddressPool(r)
				Expect(err).Should(HaveOccurred())
			})
		})

		Context("when Allocation Range Start is not valid IPv4 or IPv6", func() {
			It("should fail validation with error", func() {
				r := GetAddrPool("ipv4")
				invalid_ip := "not-valid-ipv4-ipv6"
				r.Spec.Allocation.Ranges[0].Start = invalid_ip
				err := validateAddressPool(r)
				Expect(err).Should(HaveOccurred())
			})
		})

		Context("when Allocation Range End is not valid IPv4 or IPv6", func() {
			It("should fail validation with error", func() {
				r := GetAddrPool("ipv4")
				invalid_ip := "not-valid-ipv4-ipv6"
				r.Spec.Allocation.Ranges[0].End = invalid_ip
				err := validateAddressPool(r)
				Expect(err).Should(HaveOccurred())
			})
		})

		Context("when gateway is not same IP family as subnet", func() {
			It("should fail validation with error", func() {
				r := GetAddrPool("ipv4")
				invalid_ip := "fcff:1:2::3"
				r.Spec.Gateway = &invalid_ip
				err := validateAddressPool(r)
				Expect(err).Should(HaveOccurred())
			})
		})

		Context("when FloatingAddress is not same IP family as subnet", func() {
			It("should fail validation with error", func() {
				r := GetAddrPool("ipv4")
				invalid_ip := "fcff:1:2::3"
				r.Spec.FloatingAddress = &invalid_ip
				err := validateAddressPool(r)
				Expect(err).Should(HaveOccurred())
			})
		})

		Context("when Controller0Address is not same IP family as subnet", func() {
			It("should fail validation with error", func() {
				r := GetAddrPool("ipv4")
				invalid_ip := "fcff:1:2::3"
				r.Spec.Controller0Address = &invalid_ip
				err := validateAddressPool(r)
				Expect(err).Should(HaveOccurred())
			})
		})

		Context("when Controller1Address is not same IP family as subnet", func() {
			It("should fail validation with error", func() {
				r := GetAddrPool("ipv4")
				invalid_ip := "fcff:1:2::3"
				r.Spec.Controller1Address = &invalid_ip
				err := validateAddressPool(r)
				Expect(err).Should(HaveOccurred())
			})
		})

		Context("when Allocation Range Start and End is not from same IP family", func() {
			It("should fail validation with error", func() {
				r := GetAddrPool("ipv4")
				invalid_ip := "fcff:1:2::3"
				r.Spec.Allocation.Ranges[0].End = invalid_ip
				err := validateAddressPool(r)
				Expect(err).Should(HaveOccurred())
			})
		})

		Context("when Allocation Range Start and End is not from same IP family as subnet", func() {
			It("should fail validation with error", func() {
				r := GetAddrPool("ipv4")
				invalid_ip := "fcff:1:2::3"
				r.Spec.Allocation.Ranges[0].Start = invalid_ip
				r.Spec.Allocation.Ranges[0].End = invalid_ip
				err := validateAddressPool(r)
				Expect(err).Should(HaveOccurred())
			})
		})

		Context("when prefix is invalid for IP family", func() {
			It("should fail validation with error", func() {
				r := GetAddrPool("ipv4")
				r.Spec.Prefix = 33
				err := validateAddressPool(r)
				Expect(err).Should(HaveOccurred())
			})
		})
	})
})

var _ = Describe("AddressPoolWebhook wrappers", func() {
	Context("when calling validators directly", func() {
		It("should accept ValidateCreate", func() {
			v := &AddressPoolCustomValidator{}
			_, err := v.ValidateCreate(ctx, GetAddrPool("ipv4"))
			Expect(err).ToNot(HaveOccurred())
		})
		It("should accept ValidateUpdate", func() {
			v := &AddressPoolCustomValidator{}
			_, err := v.ValidateUpdate(ctx, GetAddrPool("ipv4"), GetAddrPool("ipv4"))
			Expect(err).ToNot(HaveOccurred())
		})
		It("should accept ValidateDelete", func() {
			v := &AddressPoolCustomValidator{}
			_, err := v.ValidateDelete(ctx, GetAddrPool("ipv4"))
			Expect(err).ToNot(HaveOccurred())
		})
	})
})
