/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2024-2025 Wind River Systems, Inc. */
package v1

import (
	"errors"

	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/memory"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/physicalvolumes"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	starlingxv1 "github.com/wind-river/cloud-platform-deployment-manager/api/v1"
)

var _ = Describe("hostProfile_webhook functions", func() {

	Describe("validateMemoryFunction function is tested", func() {
		Context("When memory function is platform and size is not 4kb", func() {
			It("Gives the platform memory must be allocated from 4K pages error", func() {
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
			It("4K pages can only be reserved for platform memory error is thrown", func() {
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
			It("Validation of memory function is succesfull without error", func() {
				node := starlingxv1.MemoryNodeInfo{}
				function := starlingxv1.MemoryFunctionInfo{
					Function: memory.MemoryFunctionPlatform,
					PageSize: string(starlingxv1.PageSize4K),
				}
				err := validateMemoryFunction(node, function)
				Expect(err).To(BeNil())
			})
		})
	})

	Describe("validateProcessorInfo function is tested", func() {
		Context("When no duplicate processor entries are present", func() {
			It("validates without throwing error", func() {
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
				Expect(err).To(BeNil())
			})
		})

		Context("When there are duplicate processor entries", func() {
			It("Throws the duplicate processor entries are not allowed error", func() {
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

	Describe("validatePhysicalVolumeInfo function is tested", func() {
		Context("When partition size is nil ", func() {
			It("Throws partition specifications must include a 'size' attribute error", func() {
				obj := &starlingxv1.PhysicalVolumeInfo{
					Type: physicalvolumes.PVTypePartition,
				}
				err := validatePhysicalVolumeInfo(obj)
				msg := errors.New("partition specifications must include a 'size' attribute")
				Expect(err).To(Equal(msg))
			})
		})
		Context("When the volume type is not partition", func() {
			It("Successful with no error", func() {
				obj := &starlingxv1.PhysicalVolumeInfo{
					Type: "randomType",
				}
				err := validatePhysicalVolumeInfo(obj)
				Expect(err).To(BeNil())
			})
		})
	})

	Describe("validateMemoryInfo function is tested", func() {
		//TBD: when duplicate memory entries are present.
		Context("When no duplicate memory entries are present", func() {
			It("Validates the memory info without throwing any error", func() {
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
				Expect(err).To(BeNil())
			})
		})
		Context("When duplicate memory entries are present", func() {
			It("Validates the memory info and throws error", func() {
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
	Describe("validateVolumeGroupInfo function is tested", func() {
		Context("When the vloumeGroup info has partition with size attr", func() {
			It("Successfully validates the VolumeGroupInfo without errors", func() {
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
				Expect(err).To(BeNil())
			})
		})
		Context("When the vloumeGroup info has partition without size attr", func() {
			It("Throws the partition specifications must include a 'size' attribute error", func() {
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
	Describe("validateStorageInfo function is tested", func() {
		Context("When there is size attr present with partition type physcial vol", func() {
			It("Succesfully validates Storage Info without error", func() {
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
				Expect(err).To(BeNil())
			})
		})
	})
	Describe("validateHostProfile function is tested", func() {
		Context("When the spec base is empty", func() {
			It("Throws profile base name must not be empty error", func() {
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
			It("Throws profile base name must not be empty error", func() {
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
			It("Successfully validates Host Profile without any error", func() {
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
				Expect(err).To(BeNil())
			})
		})
	})
})
