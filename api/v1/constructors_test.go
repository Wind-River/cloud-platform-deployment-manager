/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019-2023 Wind River Systems, Inc. */

package v1

import (
	"reflect"
	"strings"

	"github.com/alecthomas/units"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/addresses"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/cpus"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/disks"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/labels"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/routes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	common "github.com/wind-river/cloud-platform-deployment-manager/common"
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

	Describe("Test IsDefaultServiceParameter", func() {
		Context("When filtering default service parameters", func() {
			It("should return true", func() {
				tests := common.DefaultParameters
				for _, test := range tests {
					singleServiceParameter := ServiceParameterInfo{
						Service:   test.Service,
						Section:   test.Section,
						ParamName: test.ParamName,
					}
					got := IsDefaultServiceParameter(&singleServiceParameter)
					Expect(got).To(BeTrue())
				}
			})
		})
		Context("When filtering not default service parameters", func() {
			It("should return false", func() {
				singleServiceParameter := ServiceParameterInfo{
					Service:   "foo",
					Section:   "bar",
					ParamName: "foobar",
				}
				got := IsDefaultServiceParameter(&singleServiceParameter)
				Expect(got).To(BeFalse())
			})
		})
	})

	Describe("Test Megabytes", func() {
		Context("When the pagesize is PageSize1G", func() {
			var test PageSize = PageSize1G
			It("should return 1024", func() {
				want := 1024
				got := test.Megabytes()
				Expect(reflect.DeepEqual(got, want)).To(BeTrue())

			})
		})
		Context("When the pagesize is PageSize2M", func() {
			It("should return 2", func() {
				test := PageSize(PageSize2M)
				want := 2
				got := test.Megabytes()
				Expect(reflect.DeepEqual(got, want)).To(BeTrue())

			})
		})
		Context("When the pagesize is other than PageSize2M and PageSize1G", func() {
			It("should return 0", func() {
				test := PageSize("5MB")
				want := 0
				got := test.Megabytes()
				Expect(reflect.DeepEqual(got, want)).To(BeTrue())

			})
		})
	})

	Describe("Test Bytes", func() {
		Context("When the pagesize is PageSize1G", func() {
			var test PageSize = PageSize1G
			It("should return int(units.Gibibyte)", func() {
				want := int(units.Gibibyte)
				got := test.Bytes()
				Expect(reflect.DeepEqual(got, want)).To(BeTrue())

			})
		})
		Context("When the pagesize is PageSize2M", func() {
			It("should return 2 * int(units.Mebibyte)", func() {
				test := PageSize(PageSize2M)
				want := 2 * int(units.Mebibyte)
				got := test.Bytes()
				Expect(reflect.DeepEqual(got, want)).To(BeTrue())

			})
		})
		Context("When the pagesize is PageSize4K", func() {
			It("should return 4 * int(units.KiB)", func() {
				test := PageSize(PageSize4K)
				want := 4 * int(units.KiB)
				got := test.Bytes()
				Expect(reflect.DeepEqual(got, want)).To(BeTrue())

			})
		})
		Context("When the pagesize is other than PageSize2M,PageSize1G and PageSize4K", func() {
			It("should return 0", func() {
				test := PageSize("5MB")
				want := 0
				got := test.Bytes()
				Expect(reflect.DeepEqual(got, want)).To(BeTrue())

			})
		})
	})

	Describe("Test parseLabelInfo", func() {
		Context("When host labels are present", func() {
			It("should check if profile labels are same as host lables)", func() {
				profile := &HostProfileSpec{}
				host := platform.HostInfo{}
				hostLabels := make([]labels.Label, 2)
				hostLabels[0].Key = "label1"
				hostLabels[0].Value = "label1"
				hostLabels[1].Key = "label2"
				hostLabels[1].Value = "label2"
				host.Labels = hostLabels
				err := parseLabelInfo(profile, host)
				Expect(err).To(BeNil())
				Expect(len(profile.Labels)).To(Equal(len(hostLabels)))

				profileLabels1 := make([]labels.Label, len(profile.Labels))
				profileLabels2 := make([]labels.Label, len(profile.Labels))

				profileLabels1[0].Key = "label1"
				profileLabels1[0].Value = "label1"
				profileLabels1[1].Key = "label2"
				profileLabels1[1].Value = "label2"

				profileLabels2[0].Key = "label2"
				profileLabels2[0].Value = "label2"
				profileLabels2[1].Key = "label1"
				profileLabels2[1].Value = "label1"

				profileLabels := make([]labels.Label, len(profile.Labels))

				itr := 0
				for j, v := range profile.Labels {
					profileLabels[itr].Key = j
					profileLabels[itr].Value = v
					itr++
				}
				flag := false
				if (profileLabels[0] == profileLabels1[0] && profileLabels[1] == profileLabels1[1]) || (profileLabels[0] == profileLabels2[0] && profileLabels[1] == profileLabels2[1]) {
					flag = true
				}
				Expect(flag).To(Equal(true))
			})
		})
	})

	Describe("Test parseAddressInfo", func() {
		Context("When host address is not a  systemAddress", func() {
			It("should check if profile address is same as host address)", func() {
				profile := &HostProfileSpec{}
				host := platform.HostInfo{}
				host.Addresses = make([]addresses.Address, 1)

				host.Addresses[0] = addresses.Address{
					InterfaceName: "InterfaceName",
					Address:       "Address",
					Prefix:        1,
				}
				err := parseAddressInfo(profile, host)
				profileAddresses := make([]addresses.Address, len(profile.Addresses))

				itr := 0
				for _, v := range profile.Addresses {
					profileAddresses[itr].Address = v.Address
					profileAddresses[itr].InterfaceName = v.Interface
					profileAddresses[itr].Prefix = v.Prefix
					itr++
				}
				Expect(err).To(BeNil())
				Expect(profileAddresses).To(Equal(host.Addresses))
			})
		})
		Context("When host address is  a systemAddress", func() {
			It("should check if profile address is nil)", func() {
				profile := &HostProfileSpec{}
				host := platform.HostInfo{}
				poolID := "poolID"
				address := addresses.Address{
					InterfaceName: "InterfaceName",
					Address:       "Address",
					Prefix:        1,
					PoolUUID:      &poolID,
				}
				host.Addresses = make([]addresses.Address, 1)
				host.Addresses[0] = address
				err := parseAddressInfo(profile, host)
				Expect(err).To(BeNil())
				Expect(profile.Addresses).To(BeNil())
			})
		})
	})

	Describe("Test parseRouteInfo", func() {
		Context("When host route data is present in thre host", func() {
			It("should check if profile routes are same as host routes)", func() {
				profile := &HostProfileSpec{}
				host := platform.HostInfo{}
				host.Routes = make([]routes.Route, 1)
				hostRoutes := routes.Route{
					InterfaceName: "InterfaceName",
					Network:       "oam",
					Prefix:        1,
					Gateway:       "Gateway",
					Metric:        1,
				}
				host.Routes[0] = hostRoutes
				err := parseRouteInfo(profile, host)
				profileRoutes := make([]routes.Route, len(profile.Routes))

				itr := 0
				for _, v := range profile.Routes {
					profileRoutes[itr].Network = v.Network
					profileRoutes[itr].InterfaceName = v.Interface
					profileRoutes[itr].Prefix = v.Prefix
					profileRoutes[itr].Gateway = v.Gateway
					profileRoutes[itr].Metric = *v.Metric
					itr++
				}
				Expect(err).To(BeNil())
				Expect(profileRoutes).To(Equal(host.Routes))
			})
		})
	})

	Describe("Test parseProcessorInfo", func() {
		Context("When host route data is present in thre host", func() {
			It("should check if profile routes are same as host routes)", func() {
				profile := &HostProfileSpec{}

				hostCpus := make([]cpus.CPU, 1)
				hostCpus[0].Function = cpus.CPUFunctionPlatform
				hostCpus[0].Processor = 0
				hostCpus[0].Thread = 0
				h := platform.HostInfo{
					CPU: hostCpus,
				}
				want := make([]ProcessorInfo, 1)
				data := ProcessorFunctionInfo{
					Function: strings.ToLower(hostCpus[0].Function),
					Count:    1,
				}
				want[0].Node = 0
				want[0].Functions = make([]ProcessorFunctionInfo, 0)
				want[0].Functions = append(want[0].Functions, data)
				list := ProcessorNodeList(want)
				err := parseProcessorInfo(profile, h)
				Expect(err).To(BeNil())
				Expect(profile.Processors).To(Equal(list))
			})
		})
	})
})
