/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2026 Wind River Systems, Inc. */
package host

import (
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/cpus"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/kernel"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	starlingxv1 "github.com/wind-river/cloud-platform-deployment-manager/api/v1"
	v1info "github.com/wind-river/cloud-platform-deployment-manager/platform"
)

var _ = Describe("GetKernelUpdateOpts", func() {
	Context("when kernel matches", func() {
		It("should return false", func() {
			k := "standard"
			profile := &starlingxv1.HostProfileSpec{
				ProfileBaseAttributes: starlingxv1.ProfileBaseAttributes{Kernel: &k},
			}
			current := &kernel.Kernel{ProvisionedKernel: "standard"}
			_, required := GetKernelUpdateOpts(current, profile)
			Expect(required).To(BeFalse())
		})
	})

	Context("when kernel differs", func() {
		It("should return true with updated kernel", func() {
			k := "lowlatency"
			profile := &starlingxv1.HostProfileSpec{
				ProfileBaseAttributes: starlingxv1.ProfileBaseAttributes{Kernel: &k},
			}
			current := &kernel.Kernel{ProvisionedKernel: "standard"}
			opts, required := GetKernelUpdateOpts(current, profile)
			Expect(required).To(BeTrue())
			Expect(*opts.Kernel).To(Equal("lowlatency"))
		})
	})
})

var _ = Describe("GetCPUUpdateOpts", func() {
	Context("when CPU counts match", func() {
		It("should return false", func() {
			r := &HostReconciler{}
			profile := &starlingxv1.HostProfileSpec{
				Processors: starlingxv1.ProcessorNodeList{
					{Node: 0, Functions: starlingxv1.ProcessorFunctionList{
						{Function: cpus.CPUFunctionPlatform, Count: 2},
					}},
				},
			}
			host := &v1info.HostInfo{
				CPU: []cpus.CPU{
					{Processor: 0, Function: cpus.CPUFunctionPlatform},
					{Processor: 0, Function: cpus.CPUFunctionPlatform},
				},
			}
			_, required := r.GetCPUUpdateOpts(profile, host)
			Expect(required).To(BeFalse())
		})
	})

	Context("when CPU counts differ", func() {
		It("should return true with opts", func() {
			r := &HostReconciler{}
			profile := &starlingxv1.HostProfileSpec{
				Processors: starlingxv1.ProcessorNodeList{
					{Node: 0, Functions: starlingxv1.ProcessorFunctionList{
						{Function: cpus.CPUFunctionPlatform, Count: 4},
					}},
				},
			}
			host := &v1info.HostInfo{
				CPU: []cpus.CPU{
					{Processor: 0, Function: cpus.CPUFunctionPlatform},
					{Processor: 0, Function: cpus.CPUFunctionPlatform},
				},
			}
			opts, required := r.GetCPUUpdateOpts(profile, host)
			Expect(required).To(BeTrue())
			Expect(opts).To(HaveLen(1))
			Expect(opts[0].Function).To(Equal(cpus.CPUFunctionPlatform))
		})
	})
})
