// /* SPDX-License-Identifier: Apache-2.0 */
// /* Copyright(c) 2019-2022 Wind River Systems, Inc. */
package host

import (
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/memory"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	starlingxv1 "github.com/wind-river/cloud-platform-deployment-manager/api/v1"
)

var _ = Describe("Memory utils", func() {
	Context("swtich count memory", func() {
		var pagesize starlingxv1.PageSize = "PageSize1G"
		node := 2

		memories := []memory.Memory{{
			ID:        "234",
			Processor: 1,
		}}
		want := 0
		got, err := vswitchCountMemoryByFunction(memories, node, pagesize)

		Expect(got).To(Equal(want))
		Expect(err).To(BeNil())

	})
	Context("swtich count memory", func() {
		var pagesize starlingxv1.PageSize = "PageSize2M"
		node := 2

		memories := []memory.Memory{{
			ID:                       "234",
			Processor:                2,
			VSwitchHugepagesSize:     0,
			VSwitchHugepagesCount:    1,
			VSwitchHugepagesRequired: nil,
		}}
		want := 1
		got, err := vswitchCountMemoryByFunction(memories, node, pagesize)
		Expect(got).To(Equal(want))
		Expect(err).To(BeNil())

	})

	Context("swtich count memory", func() {
		var pagesize starlingxv1.PageSize = "PageSize2M"
		node := 2
		required := 2

		memories := []memory.Memory{{
			ID:                       "345",
			Processor:                2,
			VSwitchHugepagesSize:     0,
			VSwitchHugepagesRequired: &required,
			VSwitchHugepagesCount:    2,
		}}
		want := 2
		got, err := vswitchCountMemoryByFunction(memories, node, pagesize)
		Expect(got).To(Equal(want))
		Expect(err).To(BeNil())

	})
})
