/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2024-2026 Wind River Systems, Inc. */
package v1

import (
	"errors"
	"time"

	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/memory"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/physicalvolumes"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	starlingxv1 "github.com/wind-river/cloud-platform-deployment-manager/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("HostProfileWebhook", func() {

	Describe("ValidateMemoryFunction", func() {
		Context("When memory function is platform and size is not 4kb", func() {
			It("should return platform memory must be allocated from 4K pages error", func() {
				node := starlingxv1.MemoryNodeInfo{}
				function := starlingxv1.MemoryFunctionInfo{
					Function: memory.MemoryFunctionPlatform,
					PageSize: string(starlingxv1.PageSize2M),
				}
				err := validateMemoryFunction(node, function)
				msg := errors.New("platform memory must be allocated from 4K pages.")
				Expect(err).To(Equal(msg))
			})
		})
		Context("When memory function is not platform and size is 4kb", func() {
			It("should return 4K pages can only be reserved for platform memory error", func() {
				node := starlingxv1.MemoryNodeInfo{}
				function := starlingxv1.MemoryFunctionInfo{
					Function: "random func",
					PageSize: string(starlingxv1.PageSize4K),
				}
				err := validateMemoryFunction(node, function)
				msg := errors.New("4K pages can only be reserved for platform memory.")
				Expect(err).To(Equal(msg))
			})
		})
		Context("When memory function is  platform and size is 4kb", func() {
			It("should validate memory function successfully without error", func() {
				node := starlingxv1.MemoryNodeInfo{}
				function := starlingxv1.MemoryFunctionInfo{
					Function: memory.MemoryFunctionPlatform,
					PageSize: string(starlingxv1.PageSize4K),
				}
				err := validateMemoryFunction(node, function)
				Expect(err).ToNot(HaveOccurred())
			})
		})
	})

	Describe("ValidateProcessorInfo", func() {
		Context("When no duplicate processor entries are present", func() {
			It("should validate without error", func() {
				obj := &starlingxv1.HostProfile{
					Spec: starlingxv1.HostProfileSpec{
						Processors: starlingxv1.ProcessorNodeList{
							{
								Functions: starlingxv1.ProcessorFunctionList{
									{
										Function: "platform",
										Count:    1,
									},
								},
							},
						},
					},
				}
				err := validateProcessorInfo(obj)
				Expect(err).ToNot(HaveOccurred())
			})
		})

		Context("When there are duplicate processor entries", func() {
			It("should return duplicate processor entries are not allowed error", func() {
				obj := &starlingxv1.HostProfile{
					Spec: starlingxv1.HostProfileSpec{
						Processors: starlingxv1.ProcessorNodeList{
							{
								Functions: starlingxv1.ProcessorFunctionList{
									{
										Function: "platform",
										Count:    1,
									},
									{
										Function: "platform",
										Count:    2,
									},
								},
								Node: 5,
							},
						},
					},
				}
				msg := errors.New("duplicate processor entries are not allowed for node 5 function platform.")
				err := validateProcessorInfo(obj)
				Expect(err).To(Equal(msg))
			})
		})
	})

	Describe("ValidatePhysicalVolumeInfo", func() {
		Context("When partition size is nil ", func() {
			It("should return partition specifications must include a size attribute error", func() {
				obj := &starlingxv1.PhysicalVolumeInfo{
					Type: physicalvolumes.PVTypePartition,
				}
				err := validatePhysicalVolumeInfo(obj)
				msg := errors.New("partition specifications must include a 'size' attribute")
				Expect(err).To(Equal(msg))
			})
		})
		Context("When the volume type is not partition", func() {
			It("should succeed without error", func() {
				obj := &starlingxv1.PhysicalVolumeInfo{
					Type: "randomType",
				}
				err := validatePhysicalVolumeInfo(obj)
				Expect(err).ToNot(HaveOccurred())
			})
		})
	})

	Describe("ValidateMemoryInfo", func() {
		//TBD: when duplicate memory entries are present.
		Context("When no duplicate memory entries are present", func() {
			It("should validate memory info without error", func() {
				obj := &starlingxv1.HostProfile{
					Spec: starlingxv1.HostProfileSpec{
						Memory: starlingxv1.MemoryNodeList{
							{
								Node: 1,
								Functions: starlingxv1.MemoryFunctionList{
									{
										Function:  memory.MemoryFunctionPlatform,
										PageSize:  string(starlingxv1.PageSize4K),
										PageCount: 100,
									},
								},
							},
						},
					},
				}
				err := validateMemoryInfo(obj)
				Expect(err).ToNot(HaveOccurred())
			})
		})
		Context("When duplicate memory entries are present", func() {
			It("should return error when validating memory info", func() {
				obj := &starlingxv1.HostProfile{
					Spec: starlingxv1.HostProfileSpec{
						Memory: starlingxv1.MemoryNodeList{
							{
								Node: 1,
								Functions: starlingxv1.MemoryFunctionList{
									{
										Function:  memory.MemoryFunctionPlatform,
										PageSize:  string(starlingxv1.PageSize4K),
										PageCount: 100,
									},
									{
										Function:  memory.MemoryFunctionPlatform,
										PageSize:  string(starlingxv1.PageSize4K),
										PageCount: 150,
									},
								},
							},
						},
					},
				}
				err := validateMemoryInfo(obj)
				msg := errors.New("duplicate memory entries are not allowed for node 1 function platform pagesize 4KB.")
				Expect(err).To(Equal(msg))
			})
		})
	})
	Describe("ValidateVolumeGroupInfo", func() {
		Context("When the vloumeGroup info has partition with size attr", func() {
			It("should validate VolumeGroupInfo successfully without errors", func() {
				size := 1
				obj := &starlingxv1.VolumeGroupInfo{
					PhysicalVolumes: starlingxv1.PhysicalVolumeList{
						{
							Type: "disk",
							Path: "/a/b/c",
							Size: &size,
						},
						{
							Type: "partition",
							Path: "pathtodevice",
							Size: &size,
						},
					},
				}
				err := validateVolumeGroupInfo(obj)
				Expect(err).ToNot(HaveOccurred())
			})
		})
		Context("When the vloumeGroup info has partition without size attr", func() {
			It("should return partition specifications must include a size attribute error", func() {
				size := 1
				obj := &starlingxv1.VolumeGroupInfo{
					PhysicalVolumes: starlingxv1.PhysicalVolumeList{
						{
							Type: "disk",
							Path: "/a/b/c",
							Size: &size,
						},
						{
							Type: "partition",
							Path: "pathtodevice",
						},
					},
				}
				err := validateVolumeGroupInfo(obj)
				msg := errors.New("partition specifications must include a 'size' attribute")
				Expect(err).To(Equal(msg))
			})
		})
	})
	Describe("ValidateStorageInfo", func() {
		Context("When there is size attr present with partition type physcial vol", func() {
			It("should validate Storage Info successfully without error", func() {
				size := 1
				obj := &starlingxv1.HostProfile{
					Spec: starlingxv1.HostProfileSpec{
						Storage: &starlingxv1.ProfileStorageInfo{
							VolumeGroups: starlingxv1.VolumeGroupList{
								{
									Name: "VolGrpList",
									PhysicalVolumes: starlingxv1.PhysicalVolumeList{
										{
											Type: "disk",
											Path: "/a/b/c",
											Size: &size,
										},
										{
											Type: "partition",
											Path: "pathtodevice",
											Size: &size,
										},
									},
								},
							},
						},
					},
				}
				err := validateStorageInfo(obj)
				Expect(err).ToNot(HaveOccurred())
			})
		})
	})
	Describe("ValidateHostProfile", func() {
		Context("When the spec base is empty", func() {
			It("should return profile base name must not be empty error", func() {
				size := 1
				baseEmpty := ""
				obj := &starlingxv1.HostProfile{
					Spec: starlingxv1.HostProfileSpec{
						Base: &baseEmpty,
						Storage: &starlingxv1.ProfileStorageInfo{
							VolumeGroups: starlingxv1.VolumeGroupList{
								{
									Name: "VolGrpList",
									PhysicalVolumes: starlingxv1.PhysicalVolumeList{
										{
											Type: "disk",
											Path: "/a/b/c",
											Size: &size,
										},
										{
											Type: "partition",
											Path: "pathtodevice",
											Size: &size,
										},
									},
								},
							},
						},
					},
				}
				err := validateHostProfile(obj)
				msg := errors.New("profile base name must not be empty")
				Expect(err).To(Equal(msg))
			})
		})
		Context("When the spec base is empty", func() {
			It("should return profile base name must not be empty error", func() {
				baseEmpty := ""
				obj := &starlingxv1.HostProfile{
					Spec: starlingxv1.HostProfileSpec{
						Base: &baseEmpty,
					},
				}
				err := validateHostProfile(obj)
				msg := errors.New("profile base name must not be empty")
				Expect(err).To(Equal(msg))
			})
		})
		Context("When the spec base is non-empty", func() {
			It("should validate Host Profile successfully without any error", func() {
				size := 1
				base := "base"
				obj := &starlingxv1.HostProfile{
					Spec: starlingxv1.HostProfileSpec{
						Base: &base,
						Processors: starlingxv1.ProcessorNodeList{
							{
								Functions: starlingxv1.ProcessorFunctionList{
									{
										Function: "platform",
										Count:    1,
									},
									{
										Function: "application",
										Count:    2,
									},
								},
								Node: 5,
							},
						},
						Memory: starlingxv1.MemoryNodeList{
							{
								Node: 1,
								Functions: starlingxv1.MemoryFunctionList{
									{
										Function:  memory.MemoryFunctionPlatform,
										PageSize:  string(starlingxv1.PageSize4K),
										PageCount: 100,
									},
									{
										Function:  "vm",
										PageSize:  string(starlingxv1.PageSize2M),
										PageCount: 150,
									},
								},
							},
						},
						Storage: &starlingxv1.ProfileStorageInfo{
							VolumeGroups: starlingxv1.VolumeGroupList{
								{
									Name: "VolGrpList",
									PhysicalVolumes: starlingxv1.PhysicalVolumeList{
										{
											Type: "disk",
											Path: "/a/b/c",
											Size: &size,
										},
										{
											Type: "partition",
											Path: "pathtodevice",
											Size: &size,
										},
									},
								},
							},
						},
					},
				}
				err := validateHostProfile(obj)
				Expect(err).ToNot(HaveOccurred())
			})
		})
	})
})

var _ = Describe("HostProfileWebhook integration", func() {

	const (
		timeout  = time.Second * 30
		interval = time.Millisecond * 500
	)

	Context("with valid console formats", func() {
		It("should accept or reject based on console validation", func() {
			key := types.NamespacedName{Namespace: "default"}

			tests := []struct {
				name    string
				console string
				wantErr bool
			}{
				{name: "empty", console: "", wantErr: false},
				{name: "serial-simple", console: "ttyS0", wantErr: false},
				{name: "serial-missing-baud", console: "ttyS0,", wantErr: true},
				{name: "serial-with-baud", console: "ttyS0,115200", wantErr: false},
				{name: "serial-with-baud-and-parity", console: "ttyS0,115200n8", wantErr: false},
				{name: "serial-double-digit-with-baud", console: "ttyS01,115200", wantErr: false},
				{name: "serial-invalid", console: "tty", wantErr: true},
				{name: "serial-invalid-numbers", console: "1111", wantErr: true},
				{name: "graphical-simple", console: "tty0", wantErr: false},
				{name: "graphical-double-digit", console: "tty01", wantErr: false},
				{name: "parallel-simple", console: "lp0", wantErr: false},
				{name: "parallel-double-digit", console: "lp01", wantErr: false},
				{name: "usb-simple", console: "ttyUSB0", wantErr: false},
				{name: "usb-missing-baud", console: "ttyUSB0,", wantErr: true},
				{name: "usb-with-baud", console: "ttyUSB0,115200", wantErr: false},
				{name: "usb-with-baud-and-parity", console: "ttyUSB0,115200n8", wantErr: false},
				{name: "usb-invalid", console: "usb0", wantErr: true},
			}
			for _, tt := range tests {
				created := &starlingxv1.HostProfile{
					ObjectMeta: metav1.ObjectMeta{Name: tt.name, Namespace: "default"},
					Spec: starlingxv1.HostProfileSpec{
						ProfileBaseAttributes: starlingxv1.ProfileBaseAttributes{
							Console: &tt.console,
						},
					},
				}
				if tt.wantErr {
					Expect(k8sClient.Create(ctx, created)).To(Not(Succeed()))
				} else {
					Expect(k8sClient.Create(ctx, created)).To(Succeed())
					key.Name = tt.name
					fetched := &starlingxv1.HostProfile{}
					Eventually(func() bool {
						return k8sClient.Get(ctx, key, fetched) == nil
					}, timeout, interval).Should(BeTrue())
					Expect(fetched.Spec.Console).To(Equal(created.Spec.Console))
				}
			}
		})
	})
})
