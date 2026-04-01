/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2024-2026 Wind River Systems, Inc. */
package host

import (
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/memory"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	starlingxv1 "github.com/wind-river/cloud-platform-deployment-manager/api/v1"
)

var _ = Describe("Memory utils", func() {
	Describe("vswitchCountMemoryByFunction", func() {
		Context("when processor does not match node", func() {
			It("should return zero", func() {
				memories := []memory.Memory{{ID: "234", Processor: 1}}
				got, err := vswitchCountMemoryByFunction(memories, 2, starlingxv1.PageSize1G)
				Expect(err).ToNot(HaveOccurred())
				Expect(got).To(Equal(0))
			})
		})

		Context("when hugepages required is nil", func() {
			It("should return the count", func() {
				memories := []memory.Memory{{
					ID: "234", Processor: 2,
					VSwitchHugepagesSize:     2,
					VSwitchHugepagesCount:    1,
					VSwitchHugepagesRequired: nil,
				}}
				got, err := vswitchCountMemoryByFunction(memories, 2, starlingxv1.PageSize2M)
				Expect(err).ToNot(HaveOccurred())
				Expect(got).To(Equal(1))
			})
		})

		Context("when hugepages required is set", func() {
			It("should return the required value", func() {
				required := 2
				memories := []memory.Memory{{
					ID: "345", Processor: 2,
					VSwitchHugepagesSize:     2,
					VSwitchHugepagesRequired: &required,
					VSwitchHugepagesCount:    5,
				}}
				got, err := vswitchCountMemoryByFunction(memories, 2, starlingxv1.PageSize2M)
				Expect(err).ToNot(HaveOccurred())
				Expect(got).To(Equal(2))
			})
		})
	})

	Describe("vmCountMemoryByFunction", func() {
		Context("when using 2M pages with pending value", func() {
			It("should return the pending count", func() {
				pending := 10
				memories := []memory.Memory{{
					Processor:            0,
					VM2MHugepagesCount:   5,
					VM2MHugepagesPending: &pending,
				}}
				got, err := vmCountMemoryByFunction(memories, 0, starlingxv1.PageSize2M)
				Expect(err).ToNot(HaveOccurred())
				Expect(got).To(Equal(10))
			})
		})

		Context("when using 2M pages without pending", func() {
			It("should return the current count", func() {
				memories := []memory.Memory{{
					Processor:          0,
					VM2MHugepagesCount: 5,
				}}
				got, err := vmCountMemoryByFunction(memories, 0, starlingxv1.PageSize2M)
				Expect(err).ToNot(HaveOccurred())
				Expect(got).To(Equal(5))
			})
		})

		Context("when using 1G pages with pending value", func() {
			It("should return the pending count", func() {
				pending := 4
				memories := []memory.Memory{{
					Processor:            0,
					VM1GHugepagesCount:   2,
					VM1GHugepagesPending: &pending,
				}}
				got, err := vmCountMemoryByFunction(memories, 0, starlingxv1.PageSize1G)
				Expect(err).ToNot(HaveOccurred())
				Expect(got).To(Equal(4))
			})
		})

		Context("when using 1G pages without pending", func() {
			It("should return the current count", func() {
				memories := []memory.Memory{{
					Processor:          0,
					VM1GHugepagesCount: 2,
				}}
				got, err := vmCountMemoryByFunction(memories, 0, starlingxv1.PageSize1G)
				Expect(err).ToNot(HaveOccurred())
				Expect(got).To(Equal(2))
			})
		})
	})

	Describe("platformCountMemoryByFunction", func() {
		Context("when using 4K pages", func() {
			It("should convert MiB platform memory to 4K page count", func() {
				memories := []memory.Memory{{
					Processor: 0,
					Platform:  1024, // 1024 MiB
				}}
				got, err := platformCountMemoryByFunction(memories, 0, starlingxv1.PageSize4K)
				Expect(err).ToNot(HaveOccurred())
				// 1024 MiB * 1048576 / 4096 = 262144
				Expect(got).To(Equal(262144))
			})
		})

		Context("when using non-4K pages", func() {
			It("should return zero", func() {
				memories := []memory.Memory{{
					Processor: 0,
					Platform:  1024,
				}}
				got, err := platformCountMemoryByFunction(memories, 0, starlingxv1.PageSize2M)
				Expect(err).ToNot(HaveOccurred())
				Expect(got).To(Equal(0))
			})
		})
	})

	Describe("memoryCountByFunction", func() {
		var memories []memory.Memory

		BeforeEach(func() {
			memories = []memory.Memory{{Processor: 0, Platform: 512, VM2MHugepagesCount: 3}}
		})

		It("should dispatch to vswitch", func() {
			_, err := memoryCountByFunction(memories, 0, memory.MemoryFunctionVSwitch, starlingxv1.PageSize2M)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should dispatch to vm", func() {
			got, err := memoryCountByFunction(memories, 0, memory.MemoryFunctionVM, starlingxv1.PageSize2M)
			Expect(err).ToNot(HaveOccurred())
			Expect(got).To(Equal(3))
		})

		It("should dispatch to platform", func() {
			_, err := memoryCountByFunction(memories, 0, memory.MemoryFunctionPlatform, starlingxv1.PageSize4K)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should return error for unsupported function", func() {
			_, err := memoryCountByFunction(memories, 0, "unknown", starlingxv1.PageSize2M)
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("memoryUpdateRequired", func() {
		Context("when counts match", func() {
			It("should return false", func() {
				f := starlingxv1.MemoryFunctionInfo{
					Function:  memory.MemoryFunctionVM,
					PageSize:  string(starlingxv1.PageSize2M),
					PageCount: 10,
				}
				_, result := memoryUpdateRequired(f, 10)
				Expect(result).To(BeFalse())
			})
		})

		Context("when VM 1G pages differ", func() {
			It("should return true with VMHugepages1G set", func() {
				f := starlingxv1.MemoryFunctionInfo{
					Function:  memory.MemoryFunctionVM,
					PageSize:  string(starlingxv1.PageSize1G),
					PageCount: 4,
				}
				opts, result := memoryUpdateRequired(f, 2)
				Expect(result).To(BeTrue())
				Expect(*opts.VMHugepages1G).To(Equal(4))
			})
		})

		Context("when VM 2M pages differ", func() {
			It("should return true with VMHugepages2M set", func() {
				f := starlingxv1.MemoryFunctionInfo{
					Function:  memory.MemoryFunctionVM,
					PageSize:  string(starlingxv1.PageSize2M),
					PageCount: 100,
				}
				opts, result := memoryUpdateRequired(f, 50)
				Expect(result).To(BeTrue())
				Expect(*opts.VMHugepages2M).To(Equal(100))
			})
		})

		Context("when vswitch 1G pages differ", func() {
			It("should return true with VSwitchHugepages set", func() {
				f := starlingxv1.MemoryFunctionInfo{
					Function:  memory.MemoryFunctionVSwitch,
					PageSize:  string(starlingxv1.PageSize1G),
					PageCount: 2,
				}
				opts, result := memoryUpdateRequired(f, 1)
				Expect(result).To(BeTrue())
				Expect(*opts.VSwitchHugepages).To(Equal(2))
				Expect(*opts.VSwitchHugepageSize).To(Equal(1024))
			})
		})

		Context("when vswitch 2M pages differ", func() {
			It("should return true with size 2", func() {
				f := starlingxv1.MemoryFunctionInfo{
					Function:  memory.MemoryFunctionVSwitch,
					PageSize:  string(starlingxv1.PageSize2M),
					PageCount: 8,
				}
				opts, result := memoryUpdateRequired(f, 4)
				Expect(result).To(BeTrue())
				Expect(*opts.VSwitchHugepageSize).To(Equal(2))
			})
		})

		Context("when platform pages differ", func() {
			It("should return true with Platform set in MiB", func() {
				f := starlingxv1.MemoryFunctionInfo{
					Function:  memory.MemoryFunctionPlatform,
					PageSize:  string(starlingxv1.PageSize4K),
					PageCount: 262144, // 1024 MiB worth of 4K pages
				}
				opts, result := memoryUpdateRequired(f, 131072)
				Expect(result).To(BeTrue())
				// 262144 * 4096 / 1048576 = 1024
				Expect(*opts.Platform).To(Equal(1024))
			})
		})
	})
})
