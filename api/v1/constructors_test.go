/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019-2023 Wind River Systems, Inc. */

package v1

import (
	"reflect"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/disks"
	"github.com/wind-river/cloud-platform-deployment-manager/platform"
)

var _ = Describe("Constructor utils for kind", func() {
	Describe("stripPartitionNumber utility", func() {
		Context("with partition and disk data", func() {
			It("should removes the \"-partNNN\" successfully", func() {
				type args struct {
					path string
				}
				tests := []struct {
					name string
					args args
					want string
				}{
					{
						name: "partition",
						args: args{"/dev/disk/by-path/pci-0000:00:0d.0-ata-1.0-part2"},
						want: "/dev/disk/by-path/pci-0000:00:0d.0-ata-1.0",
					},
					{
						name: "disk",
						args: args{"/dev/disk/by-path/pci-0000:00:0d.0-ata-1.0"},
						want: "/dev/disk/by-path/pci-0000:00:0d.0-ata-1.0",
					},
				}
				for _, tt := range tests {
					got := stripPartitionNumber(tt.args.path)
					Expect(got).To(Equal(tt.want))
				}
			})
		})
	})

	Describe("FixDevicePath utility", func() {
		Context("with path data", func() {
			It("should convert it to the newer format", func() {
				type args struct {
					path string
					host platform.HostInfo
				}
				host1 := platform.HostInfo{
					Disks: []disks.Disk{
						{DeviceNode: "/dev/sdaa",
							DevicePath: "/dev/disk/by-path/pci-0000:00:14.0-usb-0:1:1.0-scsi-0:0:0:0"},
						{DeviceNode: "/dev/sda",
							DevicePath: "/dev/disk/by-path/pci-0000:00:15.0-usb-0:1:1.0-scsi-0:0:0:0"},
						{DeviceNode: "/dev/sdbb",
							DevicePath: "/dev/disk/by-path/pci-0000:00:16.0-usb-0:1:1.0-scsi-0:0:0:0"},
						{DeviceNode: "/dev/sdb",
							DevicePath: "/dev/disk/by-path/pci-0000:00:17.0-usb-0:1:1.0-scsi-0:0:0:0"},
						{DeviceNode: "/dev/mapper/mpatha",
							DevicePath: "/dev/disk/by-id/pci-0000:00:18.0-usb-0:1:1.0-scsi-0:0:0:0"},
					},
				}
				tests := []struct {
					name string
					args args
					want string
				}{
					{name: "short-form-no-overlap",
						args: args{path: "sda", host: host1},
						want: "/dev/disk/by-path/pci-0000:00:15.0-usb-0:1:1.0-scsi-0:0:0:0"},
					{name: "short-form",
						args: args{path: "sdaa", host: host1},
						want: "/dev/disk/by-path/pci-0000:00:14.0-usb-0:1:1.0-scsi-0:0:0:0"},
					{name: "long-form-no-overlap",
						args: args{path: "/dev/sdb", host: host1},
						want: "/dev/disk/by-path/pci-0000:00:17.0-usb-0:1:1.0-scsi-0:0:0:0"},
					{name: "long-form",
						args: args{path: "/dev/sdbb", host: host1},
						want: "/dev/disk/by-path/pci-0000:00:16.0-usb-0:1:1.0-scsi-0:0:0:0"},
					{name: "already-in-new-format",
						args: args{path: "/dev/disk/by-path/pci-0000:00:16.0-usb-0:1:1.0-scsi-0:0:0:0", host: host1},
						want: "/dev/disk/by-path/pci-0000:00:16.0-usb-0:1:1.0-scsi-0:0:0:0"},
					{name: "not-found",
						args: args{path: "/dev/sdc", host: host1},
						want: "/dev/sdc"},
					{name: "imcomplete-mapper-not-found",
						args: args{path: "mpatha", host: host1},
						want: "mpatha"},
					{name: "mapper-form",
						args: args{path: "/dev/mapper/mpatha", host: host1},
						want: "/dev/disk/by-id/pci-0000:00:18.0-usb-0:1:1.0-scsi-0:0:0:0"},
					{name: "path-form",
						args: args{path: "/dev/disk/by-id/pci-0000:00:18.0-usb-0:1:1.0-scsi-0:0:0:0", host: host1},
						want: "/dev/disk/by-id/pci-0000:00:18.0-usb-0:1:1.0-scsi-0:0:0:0"},
					{name: "imcomplete-path-form-not-found",
						args: args{path: "pci-0000:00:18.0-usb-0:1:1.0-scsi-0:0:0:0", host: host1},
						want: "pci-0000:00:18.0-usb-0:1:1.0-scsi-0:0:0:0"},
				}
				for _, tt := range tests {
					got := FixDevicePath(tt.args.path, tt.args.host)
					Expect(reflect.DeepEqual(got, tt.want)).To(BeTrue())
				}
			})
		})
	})
})
