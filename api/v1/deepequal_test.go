/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2025 Wind River Systems, Inc. */

package v1

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("deepequal utils", func() {

	Describe("deepequal PtpInstanceSpec", func() {
		Context("with map of unordered lists", func() {
			It("should be equal successfully", func() {
				params := map[string][]string{
					"global":   []string{"paramA=A", "paramB=B"},
					"sectionA": []string{"table_id=1", "UDPv4=1.2.3.4"},
				}

				spec_1 := PtpInstanceSpec{
					Service:            "ptp4l",
					InstanceParameters: params,
				}
				// same deep copy should be deep equal
				spec_2 := new(PtpInstanceSpec)
				spec_1.DeepCopyInto(spec_2)
				Expect(spec_1.DeepEqual(spec_2)).To(BeTrue())

				// modify, now deep not equal
				spec_2.InstanceParameters["global"] = []string{"paramA=B", "paramB=B"}
				Expect(spec_1.DeepEqual(spec_2)).To(BeFalse())

				// shuffle, same unordered list, be deep equal
				spec_2.InstanceParameters["global"] = []string{"paramB=B", "paramA=A"}
				Expect(spec_1.DeepEqual(spec_2)).To(BeTrue())

				// unequal length, deep not equal
				spec_2.InstanceParameters["global"] = []string{"paramB=B", "paramA=A", "test=test"}
				Expect(spec_1.DeepEqual(spec_2)).To(BeFalse())

				// unequal section, not equal
				spec_2.InstanceParameters["global"] = []string{"paramB=B", "paramA=A"}
				spec_2.InstanceParameters["sectionB"] = []string{"table_id=2"}
				Expect(spec_1.DeepEqual(spec_2)).To(BeFalse())

			})
		})
	})
})
