/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2024 Wind River Systems, Inc. */
package v1

import (
	"errors"

	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/memory"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/physicalvolumes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("hostProfile_webhook functions", func() {

	Describe("validateMemoryFunction function is tested", func() {
		Context("When memory function is platform and size is not 4kb", func() {
			It("Gives the platform memory must be allocated from 4K pages error", func() {
				node := MemoryNodeInfo{}
				function := MemoryFunctionInfo{
					Function: memory.MemoryFunctionPlatform,
					PageSize: string(PageSize2M),
				}
				err := validateMemoryFunction(node, function)
				msg := errors.New("platform memory must be allocated from 4K pages.")
				Expect(err).To(Equal(msg))
			})
		})
		Context("When memory function is not platform and size is 4kb", func() {
			It("4K pages can only be reserved for platform memory error is thrown", func() {
				node := MemoryNodeInfo{}
				function := MemoryFunctionInfo{
					Function: "random func",
					PageSize: string(PageSize4K),
				}
				err := validateMemoryFunction(node, function)
				msg := errors.New("4K pages can only be reserved for platform memory.")
				Expect(err).To(Equal(msg))
			})
		})
		Context("When memory function is  platform and size is 4kb", func() {
			It("Validation of memory function is succesfull without error", func() {
				node := MemoryNodeInfo{}
				function := MemoryFunctionInfo{
					Function: memory.MemoryFunctionPlatform,
					PageSize: string(PageSize4K),
				}
				err := validateMemoryFunction(node, function)
				Expect(err).To(BeNil())
			})
		})
	})

	Describe("validateProcessorInfo function is tested", func() {
		Context("When no duplicate processor entries are present", func() {
			It("validates without throwing error", func() {
				obj := &HostProfile{
					Spec: HostProfileSpec{
						Processors: ProcessorNodeList{
							{
								Functions: ProcessorFunctionList{
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
				obj := &HostProfile{
					Spec: HostProfileSpec{
						Processors: ProcessorNodeList{
							{
								Functions: ProcessorFunctionList{
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
				obj := &PhysicalVolumeInfo{
					Type: physicalvolumes.PVTypePartition,
				}
				err := validatePhysicalVolumeInfo(obj)
				msg := errors.New("partition specifications must include a 'size' attribute")
				Expect(err).To(Equal(msg))
			})
		})
		Context("When the volume type is not partition", func() {
			It("Successful with no error", func() {
				obj := &PhysicalVolumeInfo{
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
				obj := &HostProfile{
					Spec: HostProfileSpec{
						Memory: MemoryNodeList{
							{
								Node: 1,
								Functions: MemoryFunctionList{
									{
										Function:  memory.MemoryFunctionPlatform,
										PageSize:  string(PageSize4K),
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
				obj := &HostProfile{
					Spec: HostProfileSpec{
						Memory: MemoryNodeList{
							{
								Node: 1,
								Functions: MemoryFunctionList{
									{
										Function:  memory.MemoryFunctionPlatform,
										PageSize:  string(PageSize4K),
										PageCount: 100,
									},
									{
										Function:  memory.MemoryFunctionPlatform,
										PageSize:  string(PageSize4K),
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
				obj := &VolumeGroupInfo{
					PhysicalVolumes: PhysicalVolumeList{
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
				obj := &VolumeGroupInfo{
					PhysicalVolumes: PhysicalVolumeList{
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
				obj := &HostProfile{
					Spec: HostProfileSpec{
						Storage: &ProfileStorageInfo{
							VolumeGroups: &VolumeGroupList{
								{
									Name: "VolGrpList",
									PhysicalVolumes: PhysicalVolumeList{
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
				obj := &HostProfile{
					Spec: HostProfileSpec{
						Base: &baseEmpty,
						Storage: &ProfileStorageInfo{
							VolumeGroups: &VolumeGroupList{
								{
									Name: "VolGrpList",
									PhysicalVolumes: PhysicalVolumeList{
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
				err := obj.validateHostProfile()
				msg := errors.New("profile base name must not be empty")
				Expect(err).To(Equal(msg))
			})
		})
		Context("When the spec base is empty", func() {
			It("Throws profile base name must not be empty error", func() {
				baseEmpty := ""
				obj := &HostProfile{
					Spec: HostProfileSpec{
						Base: &baseEmpty,
					},
				}
				err := obj.validateHostProfile()
				msg := errors.New("profile base name must not be empty")
				Expect(err).To(Equal(msg))
			})
		})
		Context("When the spec base is non-empty", func() {
			It("Successfully validates Host Profile without any error", func() {
				size := 1
				base := "base"
				obj := &HostProfile{
					Spec: HostProfileSpec{
						Base: &base,
						Processors: ProcessorNodeList{
							{
								Functions: ProcessorFunctionList{
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
						Memory: MemoryNodeList{
							{
								Node: 1,
								Functions: MemoryFunctionList{
									{
										Function:  memory.MemoryFunctionPlatform,
										PageSize:  string(PageSize4K),
										PageCount: 100,
									},
									{
										Function:  "vm",
										PageSize:  string(PageSize2M),
										PageCount: 150,
									},
								},
							},
						},
						Storage: &ProfileStorageInfo{
							VolumeGroups: &VolumeGroupList{
								{
									Name: "VolGrpList",
									PhysicalVolumes: PhysicalVolumeList{
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
				err := obj.validateHostProfile()
				Expect(err).To(BeNil())
			})
		})
	})
})
