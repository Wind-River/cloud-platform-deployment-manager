/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2022 Wind River Systems, Inc. */
package common

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	v1 "github.com/wind-river/cloud-platform-deployment-manager/api/v1"
)

func TestCommon(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Common Suite")
}

var _ = Describe("CompareStructs utils", func() {
	Describe("compare two structs", func() {
		Context("with same string covered from structs ", func() {
			It("should return true for PhysicalVolumeInfo", func() {
				size1 := 128
				size2 := 128
				structA := v1.PhysicalVolumeInfo{
					Type: "cgts",
					Path: "/dev/disk/by-path/pci-0000:00:1.0-usb-0:1:1.0-scsi-0:0:0:0",
					Size: &size1,
				}
				structB := v1.PhysicalVolumeInfo{
					Type: "cgts",
					Path: "/dev/disk/by-path/pci-0000:00:1.0-usb-0:1:1.0-scsi-0:0:0:0",
					Size: &size2,
				}
				equalPhysicalVolumeInfo := CompareStructs(structA, structB)
				Expect(equalPhysicalVolumeInfo).To(BeTrue())
			})
			It("should return false for PhysicalVolumeInfo", func() {
				size1 := 128
				size2 := 128
				structA := v1.PhysicalVolumeInfo{
					Type: "cgts",
					Path: "/dev/disk/by-path/pci-0000:00:1.0-usb-0:1:1.0-scsi-0:0:0:0",
					Size: &size1,
				}
				structB := v1.PhysicalVolumeInfo{
					Type: "cgts",
					Path: "/dev/disk/by-path/pci-0000:00:1.0-usb-0:1:2.0-scsi-0:0:0:0",
					Size: &size2,
				}
				equalPhysicalVolumeInfo := CompareStructs(structA, structB)
				Expect(equalPhysicalVolumeInfo).To(BeFalse())
			})
			It("should return true for OSDInfo", func() {
				cluster1 := "ceph_cluster"
				cluster2 := "ceph_cluster"
				structC := v1.OSDInfo{
					Function:    "osd",
					Path:        "/dev/disk/by-id/wwn-0x60002ac0000000000000000800029aaa",
					ClusterName: &cluster1,
					Journal:     nil,
				}
				structD := v1.OSDInfo{
					Function:    "osd",
					Path:        "/dev/disk/by-id/wwn-0x60002ac0000000000000000800029aaa",
					ClusterName: &cluster2,
					Journal:     nil,
				}
				equalOSDInfo := CompareStructs(structC, structD)
				Expect(equalOSDInfo).To(BeTrue())
			})
			It("should return false for OSDInfo", func() {
				cluster1 := "ceph_cluster"
				cluster2 := "ceph_cluster"
				structC := v1.OSDInfo{
					Function:    "osd",
					Path:        "/dev/disk/by-id/wwn-0x60002ac0000000000000000800029aaa",
					ClusterName: &cluster1,
					Journal:     nil,
				}
				structD := v1.OSDInfo{
					Function:    "osd",
					Path:        "/dev/disk/by-id/wwn-0x60002ac0000000000000000800029aab",
					ClusterName: &cluster2,
					Journal:     nil,
				}
				equalOSDInfo := CompareStructs(structC, structD)
				Expect(equalOSDInfo).To(BeFalse())
			})
		})
	})
})
