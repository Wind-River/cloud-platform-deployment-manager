/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2024 Wind River Systems, Inc. */
package v1

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func GetAddrPool(ip_family string) *AddressPool {
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

	return &AddressPool{
		Spec: AddressPoolSpec{
			Subnet:             subnet,
			FloatingAddress:    &floating_address,
			Controller0Address: &controller0_address,
			Controller1Address: &controller1_address,
			Prefix:             24,
			Gateway:            &gateway,
			Allocation: AllocationInfo{
				Order:  &allocation_order,
				Ranges: []AllocationRange{{Start: range_start, End: range_end}},
			},
		},
	}
}

var _ = Describe("addresspool_webhook functions", func() {

	Describe("validateAddressPool function is tested", func() {
		Context("IPv4 Data is correct", func() {
			It("validates succesfully without error", func() {
				r := GetAddrPool("ipv4")
				err := r.validateAddressPool()
				Expect(err).To(BeNil())
			})
		})

		Context("IPv6 Data is correct", func() {
			It("validates succesfully without error", func() {
				r := GetAddrPool("ipv6")
				err := r.validateAddressPool()
				Expect(err).To(BeNil())
			})
		})

		Context("Subnet is not valid IPv4 / IPv6", func() {
			It("validation fails with error", func() {
				r := GetAddrPool("ipv4")
				invalid_ip := "not-valid-ipv4-ipv6"
				r.Spec.Subnet = invalid_ip
				err := r.validateAddressPool()
				Expect(err).ShouldNot(BeNil())
			})
		})

		Context("Gateway is not valid IPv4 / IPv6", func() {
			It("validation fails with error", func() {
				r := GetAddrPool("ipv4")
				invalid_ip := "not-valid-ipv4-ipv6"
				r.Spec.Gateway = &invalid_ip
				err := r.validateAddressPool()
				Expect(err).ShouldNot(BeNil())
			})
		})

		Context("FloatingAddress is not valid IPv4 / IPv6", func() {
			It("validation fails with error", func() {
				r := GetAddrPool("ipv4")
				invalid_ip := "not-valid-ipv4-ipv6"
				r.Spec.FloatingAddress = &invalid_ip
				err := r.validateAddressPool()
				Expect(err).ShouldNot(BeNil())
			})
		})

		Context("Controller0Address is not valid IPv4 / IPv6", func() {
			It("validation fails with error", func() {
				r := GetAddrPool("ipv4")
				invalid_ip := "not-valid-ipv4-ipv6"
				r.Spec.Controller0Address = &invalid_ip
				err := r.validateAddressPool()
				Expect(err).ShouldNot(BeNil())
			})
		})

		Context("Controller1Address is not valid IPv4 / IPv6", func() {
			It("validation fails with error", func() {
				r := GetAddrPool("ipv4")
				invalid_ip := "not-valid-ipv4-ipv6"
				r.Spec.Controller1Address = &invalid_ip
				err := r.validateAddressPool()
				Expect(err).ShouldNot(BeNil())
			})
		})

		Context("Allocation Range Start is not valid IPv4 / IPv6", func() {
			It("validation fails with error", func() {
				r := GetAddrPool("ipv4")
				invalid_ip := "not-valid-ipv4-ipv6"
				r.Spec.Allocation.Ranges[0].Start = invalid_ip
				err := r.validateAddressPool()
				Expect(err).ShouldNot(BeNil())
			})
		})

		Context("Allocation Range End is not valid IPv4 / IPv6", func() {
			It("validation fails with error", func() {
				r := GetAddrPool("ipv4")
				invalid_ip := "not-valid-ipv4-ipv6"
				r.Spec.Allocation.Ranges[0].End = invalid_ip
				err := r.validateAddressPool()
				Expect(err).ShouldNot(BeNil())
			})
		})

		Context("Gateway is not same IP family as subnet", func() {
			It("validation fails with error", func() {
				r := GetAddrPool("ipv4")
				invalid_ip := "fcff:1:2::3"
				r.Spec.Gateway = &invalid_ip
				err := r.validateAddressPool()
				Expect(err).ShouldNot(BeNil())
			})
		})

		Context("FloatingAddress is not same IP family as subnet", func() {
			It("validation fails with error", func() {
				r := GetAddrPool("ipv4")
				invalid_ip := "fcff:1:2::3"
				r.Spec.FloatingAddress = &invalid_ip
				err := r.validateAddressPool()
				Expect(err).ShouldNot(BeNil())
			})
		})

		Context("Controller0Address is not same IP family as subnet", func() {
			It("validation fails with error", func() {
				r := GetAddrPool("ipv4")
				invalid_ip := "fcff:1:2::3"
				r.Spec.Controller0Address = &invalid_ip
				err := r.validateAddressPool()
				Expect(err).ShouldNot(BeNil())
			})
		})

		Context("Controller1Address is not same IP family as subnet", func() {
			It("validation fails with error", func() {
				r := GetAddrPool("ipv4")
				invalid_ip := "fcff:1:2::3"
				r.Spec.Controller1Address = &invalid_ip
				err := r.validateAddressPool()
				Expect(err).ShouldNot(BeNil())
			})
		})

		Context("Allocation Range Start & End is not from same IP family", func() {
			It("validation fails with error", func() {
				r := GetAddrPool("ipv4")
				invalid_ip := "fcff:1:2::3"
				r.Spec.Allocation.Ranges[0].End = invalid_ip
				err := r.validateAddressPool()
				Expect(err).ShouldNot(BeNil())
			})
		})

		Context("Allocation Range Start & End is not from same IP family as subnet", func() {
			It("validation fails with error", func() {
				r := GetAddrPool("ipv4")
				invalid_ip := "fcff:1:2::3"
				r.Spec.Allocation.Ranges[0].Start = invalid_ip
				r.Spec.Allocation.Ranges[0].End = invalid_ip
				err := r.validateAddressPool()
				Expect(err).ShouldNot(BeNil())
			})
		})

		Context("Invalid Prefix For IP Family", func() {
			It("validation fails with error", func() {
				r := GetAddrPool("ipv4")
				r.Spec.Prefix = 33
				err := r.validateAddressPool()
				Expect(err).ShouldNot(BeNil())
			})
		})
	})
})
