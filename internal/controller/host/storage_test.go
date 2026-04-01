/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2026 Wind River Systems, Inc. */
package host

import (
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/osds"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	starlingxv1 "github.com/wind-river/cloud-platform-deployment-manager/api/v1"
)

var _ = Describe("osdUpdateRequired", func() {
	Context("when journal is nil in spec", func() {
		It("should return false", func() {
			osdInfo := &starlingxv1.OSDInfo{Function: "osd", Path: "/dev/sda"}
			osd := &osds.OSD{}
			_, result := osdUpdateRequired(osdInfo, osd)
			Expect(result).To(BeFalse())
		})
	})

	Context("when journal is set and osd has no location", func() {
		It("should return true with journal location and size", func() {
			osdInfo := &starlingxv1.OSDInfo{
				Function: "osd",
				Path:     "/dev/sda",
				Journal:  &starlingxv1.JournalInfo{Location: "/dev/sdb", Size: 1},
			}
			osd := &osds.OSD{}
			opts, result := osdUpdateRequired(osdInfo, osd)
			Expect(result).To(BeTrue())
			Expect(*opts.JournalLocation).To(Equal("/dev/sdb"))
			Expect(*opts.JournalSize).To(Equal(1))
		})
	})

	Context("when journal sizes differ", func() {
		It("should return true with updated size", func() {
			osdInfo := &starlingxv1.OSDInfo{
				Function: "osd",
				Path:     "/dev/sda",
				Journal:  &starlingxv1.JournalInfo{Location: "/dev/sdb", Size: 2},
			}
			loc := "/dev/sdb"
			sizeMiB := 1024 // 1 GiB in MiB
			osd := &osds.OSD{
				JournalInfo: osds.JournalInfo{
					Location: &loc,
					Size:     &sizeMiB,
				},
			}
			opts, result := osdUpdateRequired(osdInfo, osd)
			Expect(result).To(BeTrue())
			Expect(*opts.JournalSize).To(Equal(2))
		})
	})

	Context("when journal sizes match", func() {
		It("should return false", func() {
			osdInfo := &starlingxv1.OSDInfo{
				Function: "osd",
				Path:     "/dev/sda",
				Journal:  &starlingxv1.JournalInfo{Location: "/dev/sdb", Size: 1},
			}
			loc := "/dev/sdb"
			sizeMiB := 1024 // 1 GiB = 1024 MiB, Gibibytes() = 1024/1024 = 1
			osd := &osds.OSD{
				JournalInfo: osds.JournalInfo{
					Location: &loc,
					Size:     &sizeMiB,
				},
			}
			_, result := osdUpdateRequired(osdInfo, osd)
			Expect(result).To(BeFalse())
		})
	})
})
