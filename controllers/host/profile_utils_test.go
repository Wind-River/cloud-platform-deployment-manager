/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019-2022 Wind River Systems, Inc. */

package host

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"reflect"

	starlingxv1 "github.com/wind-river/cloud-platform-deployment-manager/api/v1"
	v1info "github.com/wind-river/cloud-platform-deployment-manager/platform"
)

var _ = Describe("Profile utils", func() {
	Describe("MergeProfiles utility", func() {
		Context("with profile spec data", func() {
			It("should merge successfully", func() {

				admin1 := "locked"
				location1 := "vbox"
				bmType1 := "bmc"
				bm1 := starlingxv1.BMInfo{
					Type: &bmType1,
				}
				personality2 := "controller"
				admin2 := "unlocked"
				type args struct {
					a *starlingxv1.HostProfileSpec
					b *starlingxv1.HostProfileSpec
				}
				size1 := 1
				osds1 := starlingxv1.OSDList{
					starlingxv1.OSDInfo{
						Function: "osd",
						Path:     "/dev/sda",
					},
				}
				vgs1 := starlingxv1.VolumeGroupList{
					starlingxv1.VolumeGroupInfo{
						Name: "nova-local",
						PhysicalVolumes: starlingxv1.PhysicalVolumeList{
							starlingxv1.PhysicalVolumeInfo{
								Type: "disk",
								Path: "/dev/sdb",
							},
						},
					},
				}
				fs1 := starlingxv1.FileSystemList{
					starlingxv1.FileSystemInfo{
						Name: "backup",
						Size: 10,
					},
				}
				osds2 := starlingxv1.OSDList{
					starlingxv1.OSDInfo{
						Function: "osd",
						Path:     "/dev/sda",
						Journal:  &starlingxv1.JournalInfo{Size: 10},
					},
					starlingxv1.OSDInfo{
						Function: "osd",
						Path:     "/dev/sdc",
					},
				}
				vgs2 := starlingxv1.VolumeGroupList{
					starlingxv1.VolumeGroupInfo{
						Name: "nova-local",
						PhysicalVolumes: starlingxv1.PhysicalVolumeList{
							starlingxv1.PhysicalVolumeInfo{
								Type: "disk",
								Path: "/dev/sdd",
							},
							starlingxv1.PhysicalVolumeInfo{
								Type: "partition",
								Path: "/dev/sde",
								Size: &size1,
							},
						},
					},
					starlingxv1.VolumeGroupInfo{
						Name: "spare",
						PhysicalVolumes: starlingxv1.PhysicalVolumeList{
							starlingxv1.PhysicalVolumeInfo{
								Type: "disk",
								Path: "/dev/sdb",
							},
						},
					},
				}
				fs2 := starlingxv1.FileSystemList{
					starlingxv1.FileSystemInfo{
						Name: "backup",
						Size: 20,
					},
					starlingxv1.FileSystemInfo{
						Name: "docker",
						Size: 5,
					},
				}
				vfcount1 := 1
				vfcount2 := 2
				vfcount3 := 3
				tests := []struct {
					name    string
					args    args
					want    *starlingxv1.HostProfileSpec
					wantErr bool
				}{
					{name: "basic",
						args: args{
							a: &starlingxv1.HostProfileSpec{
								ProfileBaseAttributes: starlingxv1.ProfileBaseAttributes{
									AdministrativeState: &admin1,
									Location:            &location1,
								},
								BoardManagement: &bm1,
							},
							b: &starlingxv1.HostProfileSpec{
								ProfileBaseAttributes: starlingxv1.ProfileBaseAttributes{
									AdministrativeState: &admin2,
									Personality:         &personality2,
								},
							},
						},
						want: &starlingxv1.HostProfileSpec{
							ProfileBaseAttributes: starlingxv1.ProfileBaseAttributes{
								AdministrativeState: &admin2,
								Personality:         &personality2,
								Location:            &location1,
							},
							BoardManagement: &bm1,
						},
						wantErr: false},
					{name: "interface-merge",
						args: args{a: &starlingxv1.HostProfileSpec{
							Interfaces: &starlingxv1.InterfaceInfo{
								Ethernet: []starlingxv1.EthernetInfo{
									{CommonInterfaceInfo: starlingxv1.CommonInterfaceInfo{Name: "eth0"}, Port: starlingxv1.EthernetPortInfo{Name: "eth0"}},
									{CommonInterfaceInfo: starlingxv1.CommonInterfaceInfo{Name: "eth1"}, Port: starlingxv1.EthernetPortInfo{Name: "eth1"}},
									{CommonInterfaceInfo: starlingxv1.CommonInterfaceInfo{Name: "sriov0"}, VFCount: &vfcount2, Port: starlingxv1.EthernetPortInfo{Name: "eth4"}}},
								VLAN: []starlingxv1.VLANInfo{
									{CommonInterfaceInfo: starlingxv1.CommonInterfaceInfo{Name: "vlan1"}, Lower: "eth0", VID: 1},
									{CommonInterfaceInfo: starlingxv1.CommonInterfaceInfo{Name: "vlan2"}, Lower: "eth1", VID: 2}},
								Bond: []starlingxv1.BondInfo{
									{CommonInterfaceInfo: starlingxv1.CommonInterfaceInfo{Name: "bond0"}, Mode: "balanced", Members: []string{"eth0", "eth1"}},
									{CommonInterfaceInfo: starlingxv1.CommonInterfaceInfo{Name: "bond1"}, Mode: "balanced", Members: []string{"eth2", "eth3"}}},
								VF: []starlingxv1.VFInfo{
									{CommonInterfaceInfo: starlingxv1.CommonInterfaceInfo{Name: "sriov1"}, VFCount: vfcount1, Lower: "sriov0"}},
							}},
							b: &starlingxv1.HostProfileSpec{
								Interfaces: &starlingxv1.InterfaceInfo{
									Ethernet: []starlingxv1.EthernetInfo{
										{CommonInterfaceInfo: starlingxv1.CommonInterfaceInfo{Name: "mgmt0"}, Port: starlingxv1.EthernetPortInfo{Name: "eth0"}},
										{CommonInterfaceInfo: starlingxv1.CommonInterfaceInfo{Name: "cluster0"}, Port: starlingxv1.EthernetPortInfo{Name: "eth2"}},
										{CommonInterfaceInfo: starlingxv1.CommonInterfaceInfo{Name: "sriov0"}, VFCount: &vfcount3, Port: starlingxv1.EthernetPortInfo{Name: "eth4"}}},
									VLAN: []starlingxv1.VLANInfo{
										{CommonInterfaceInfo: starlingxv1.CommonInterfaceInfo{Name: "vlan1"}, Lower: "mgmt0", VID: 1},
										{CommonInterfaceInfo: starlingxv1.CommonInterfaceInfo{Name: "vlan3"}, Lower: "cluster0", VID: 3}},
									Bond: []starlingxv1.BondInfo{
										{CommonInterfaceInfo: starlingxv1.CommonInterfaceInfo{Name: "bond0"}, Mode: "802.3ad", Members: []string{"eth10", "eth11"}},
										{CommonInterfaceInfo: starlingxv1.CommonInterfaceInfo{Name: "bond2"}, Mode: "802.3ad", Members: []string{"eth12", "eth13"}}},
									VF: []starlingxv1.VFInfo{
										{CommonInterfaceInfo: starlingxv1.CommonInterfaceInfo{Name: "sriov2"}, VFCount: vfcount1, Lower: "sriov0"}},
								},
							},
						},
						wantErr: false,
						want: &starlingxv1.HostProfileSpec{
							Interfaces: &starlingxv1.InterfaceInfo{
								Ethernet: []starlingxv1.EthernetInfo{
									{CommonInterfaceInfo: starlingxv1.CommonInterfaceInfo{Name: "mgmt0"}, Port: starlingxv1.EthernetPortInfo{Name: "eth0"}},
									{CommonInterfaceInfo: starlingxv1.CommonInterfaceInfo{Name: "eth1"}, Port: starlingxv1.EthernetPortInfo{Name: "eth1"}},
									{CommonInterfaceInfo: starlingxv1.CommonInterfaceInfo{Name: "sriov0"}, VFCount: &vfcount3, Port: starlingxv1.EthernetPortInfo{Name: "eth4"}},
									{CommonInterfaceInfo: starlingxv1.CommonInterfaceInfo{Name: "cluster0"}, Port: starlingxv1.EthernetPortInfo{Name: "eth2"}}},
								VLAN: []starlingxv1.VLANInfo{
									{CommonInterfaceInfo: starlingxv1.CommonInterfaceInfo{Name: "vlan1"}, Lower: "mgmt0", VID: 1},
									{CommonInterfaceInfo: starlingxv1.CommonInterfaceInfo{Name: "vlan2"}, Lower: "eth1", VID: 2},
									{CommonInterfaceInfo: starlingxv1.CommonInterfaceInfo{Name: "vlan3"}, Lower: "cluster0", VID: 3}},
								Bond: []starlingxv1.BondInfo{
									{CommonInterfaceInfo: starlingxv1.CommonInterfaceInfo{Name: "bond0"}, Mode: "802.3ad", Members: []string{"eth10", "eth11"}},
									{CommonInterfaceInfo: starlingxv1.CommonInterfaceInfo{Name: "bond1"}, Mode: "balanced", Members: []string{"eth2", "eth3"}},
									{CommonInterfaceInfo: starlingxv1.CommonInterfaceInfo{Name: "bond2"}, Mode: "802.3ad", Members: []string{"eth12", "eth13"}}},
								VF: []starlingxv1.VFInfo{
									{CommonInterfaceInfo: starlingxv1.CommonInterfaceInfo{Name: "sriov1"}, VFCount: vfcount1, Lower: "sriov0"},
									{CommonInterfaceInfo: starlingxv1.CommonInterfaceInfo{Name: "sriov2"}, VFCount: vfcount1, Lower: "sriov0"}},
							},
						},
					},
					{name: "storage-merge",
						args: args{
							a: &starlingxv1.HostProfileSpec{
								Storage: &starlingxv1.ProfileStorageInfo{
									Monitor:      &starlingxv1.MonitorInfo{Size: &size1},
									OSDs:         &osds1,
									VolumeGroups: &vgs1,
									FileSystems:  &fs1,
								},
							},
							b: &starlingxv1.HostProfileSpec{
								Storage: &starlingxv1.ProfileStorageInfo{
									Monitor:      &starlingxv1.MonitorInfo{Size: &size1},
									OSDs:         &osds2,
									VolumeGroups: &vgs2,
									FileSystems:  &fs2,
								},
							},
						},
						wantErr: false,
						want: &starlingxv1.HostProfileSpec{
							Storage: &starlingxv1.ProfileStorageInfo{
								Monitor:      &starlingxv1.MonitorInfo{Size: &size1},
								OSDs:         &osds2,
								VolumeGroups: &vgs2,
								FileSystems:  &fs2,
							},
						},
					},

					{name: "processor-merge",
						args: args{
							a: &starlingxv1.HostProfileSpec{
								Processors: starlingxv1.ProcessorNodeList{
									starlingxv1.ProcessorInfo{
										Node: 0,
										Functions: starlingxv1.ProcessorFunctionList{
											starlingxv1.ProcessorFunctionInfo{
												Function: "vswitch",
												Count:    1,
											},
										},
									},
								},
							},
							b: &starlingxv1.HostProfileSpec{
								Processors: starlingxv1.ProcessorNodeList{
									starlingxv1.ProcessorInfo{
										Node: 0,
										Functions: starlingxv1.ProcessorFunctionList{
											starlingxv1.ProcessorFunctionInfo{
												Function: "vswitch",
												Count:    2,
											},
											starlingxv1.ProcessorFunctionInfo{
												Function: "platform",
												Count:    1,
											},
										},
									},
									starlingxv1.ProcessorInfo{
										Node: 1,
										Functions: starlingxv1.ProcessorFunctionList{
											starlingxv1.ProcessorFunctionInfo{
												Function: "vswitch",
												Count:    4,
											},
											starlingxv1.ProcessorFunctionInfo{
												Function: "platform",
												Count:    2,
											},
										},
									},
								},
							},
						},
						wantErr: false,
						want: &starlingxv1.HostProfileSpec{
							Processors: starlingxv1.ProcessorNodeList{
								starlingxv1.ProcessorInfo{
									Node: 0,
									Functions: starlingxv1.ProcessorFunctionList{
										starlingxv1.ProcessorFunctionInfo{
											Function: "vswitch",
											Count:    2,
										},
										starlingxv1.ProcessorFunctionInfo{
											Function: "platform",
											Count:    1,
										},
									},
								},
								starlingxv1.ProcessorInfo{
									Node: 1,
									Functions: starlingxv1.ProcessorFunctionList{
										starlingxv1.ProcessorFunctionInfo{
											Function: "vswitch",
											Count:    4,
										},
										starlingxv1.ProcessorFunctionInfo{
											Function: "platform",
											Count:    2,
										},
									},
								},
							},
						},
					},
					{name: "memory-merge",
						args: args{
							a: &starlingxv1.HostProfileSpec{
								Memory: starlingxv1.MemoryNodeList{
									starlingxv1.MemoryNodeInfo{
										Node: 0,
										Functions: starlingxv1.MemoryFunctionList{
											starlingxv1.MemoryFunctionInfo{
												Function:  "vswitch",
												PageSize:  "2MB",
												PageCount: 16,
											},
											starlingxv1.MemoryFunctionInfo{
												Function:  "vswitch",
												PageSize:  "1GB",
												PageCount: 1,
											},
										},
									},
								},
							},
							b: &starlingxv1.HostProfileSpec{
								Memory: starlingxv1.MemoryNodeList{
									starlingxv1.MemoryNodeInfo{
										Node: 0,
										Functions: starlingxv1.MemoryFunctionList{
											starlingxv1.MemoryFunctionInfo{
												Function:  "vswitch",
												PageSize:  "2MB",
												PageCount: 128,
											},
											starlingxv1.MemoryFunctionInfo{
												Function:  "platform",
												PageSize:  "2MB",
												PageCount: 32,
											},
										},
									},
									starlingxv1.MemoryNodeInfo{
										Node: 1,
										Functions: starlingxv1.MemoryFunctionList{
											starlingxv1.MemoryFunctionInfo{
												Function:  "vswitch",
												PageSize:  "2MB",
												PageCount: 128,
											},
											starlingxv1.MemoryFunctionInfo{
												Function:  "platform",
												PageSize:  "2MB",
												PageCount: 32,
											},
										},
									},
								},
							},
						},
						wantErr: false,
						want: &starlingxv1.HostProfileSpec{
							Memory: starlingxv1.MemoryNodeList{
								starlingxv1.MemoryNodeInfo{
									Node: 0,
									Functions: starlingxv1.MemoryFunctionList{
										starlingxv1.MemoryFunctionInfo{
											Function:  "vswitch",
											PageSize:  "2MB",
											PageCount: 128,
										},
										starlingxv1.MemoryFunctionInfo{
											Function:  "vswitch",
											PageSize:  "1GB",
											PageCount: 1,
										},
										starlingxv1.MemoryFunctionInfo{
											Function:  "platform",
											PageSize:  "2MB",
											PageCount: 32,
										},
									},
								},
								starlingxv1.MemoryNodeInfo{
									Node: 1,
									Functions: starlingxv1.MemoryFunctionList{
										starlingxv1.MemoryFunctionInfo{
											Function:  "vswitch",
											PageSize:  "2MB",
											PageCount: 128,
										},
										starlingxv1.MemoryFunctionInfo{
											Function:  "platform",
											PageSize:  "2MB",
											PageCount: 32,
										},
									},
								},
							},
						},
					},
					{name: "addresses-merge",
						args: args{
							a: &starlingxv1.HostProfileSpec{
								Addresses: starlingxv1.AddressList{
									{Address: "1.2.3.4", Prefix: 24, Interface: "eth0"},
									{Address: "10.20.30.40", Prefix: 24, Interface: "eth1"},
								},
							},
							b: &starlingxv1.HostProfileSpec{
								Addresses: starlingxv1.AddressList{
									{Address: "10.20.30.40", Prefix: 32, Interface: "eth2"},
									{Address: "fd00::40", Prefix: 64, Interface: "eth1"},
								},
							},
						},
						wantErr: false,
						want: &starlingxv1.HostProfileSpec{
							Addresses: starlingxv1.AddressList{
								{Address: "1.2.3.4", Prefix: 24, Interface: "eth0"},
								{Address: "10.20.30.40", Prefix: 32, Interface: "eth2"},
								{Address: "fd00::40", Prefix: 64, Interface: "eth1"},
							},
						},
					},
					{name: "routes-merge",
						args: args{
							a: &starlingxv1.HostProfileSpec{
								Routes: starlingxv1.RouteList{
									{Network: "10.10.10.0", Prefix: 24, Gateway: "10.10.10.1", Interface: "eth0"},
									{Network: "172.16.0.0", Prefix: 16, Gateway: "172.16.0.1", Interface: "eth1"},
								},
							},
							b: &starlingxv1.HostProfileSpec{
								Routes: starlingxv1.RouteList{
									{Network: "10.10.10.0", Prefix: 24, Gateway: "10.10.10.2", Interface: "eth0"},
									{Network: "172.16.0.0", Prefix: 24, Gateway: "172.16.0.1", Interface: "eth2"},
									{Network: "fd00:1::", Prefix: 64, Gateway: "fd00:1::1", Interface: "eth1"},
								},
							},
						},
						wantErr: false,
						want: &starlingxv1.HostProfileSpec{
							Routes: starlingxv1.RouteList{
								{Network: "10.10.10.0", Prefix: 24, Gateway: "10.10.10.2", Interface: "eth0"},
								{Network: "172.16.0.0", Prefix: 16, Gateway: "172.16.0.1", Interface: "eth1"},
								{Network: "172.16.0.0", Prefix: 24, Gateway: "172.16.0.1", Interface: "eth2"},
								{Network: "fd00:1::", Prefix: 64, Gateway: "fd00:1::1", Interface: "eth1"},
							},
						},
					},
				}
				for _, tt := range tests {
					got, err := MergeProfiles(tt.args.a, tt.args.b)
					Expect(err).To(BeNil())
					Expect(reflect.DeepEqual(got, tt.want)).To(BeTrue())
					Expect(got).NotTo(BeNil())
					Expect(got.DeepEqual(tt.want)).To(BeTrue())
				}
			})
		})
	})

	Describe("FixKernelSubfunction", func() {
		Context("with profile spec data", func() {
			It("should fix the subfunctions successfully", func() {
				controllerPersonality := "controller"
				workerPersonality := "worker"
				standardKernel := "standard"
				lowlatencyKernel := "lowlatency"
				aioSubfuncs := []starlingxv1.SubFunction{"controller", "worker"}
				aiolowlatencySubfuncs := []starlingxv1.SubFunction{"controller", "worker", "lowlatency"}
				workerSubfuncs := []starlingxv1.SubFunction{"worker"}
				workerlowlatencySubfuncs := []starlingxv1.SubFunction{"worker", "lowlatency"}

				type args struct {
					spec *starlingxv1.HostProfileSpec
				}
				tests := []struct {
					name                string
					args                args
					nochange            bool
					updatedSubfunctions []starlingxv1.SubFunction
				}{
					// tests and spec data
					{
						name: "All-In-One Standard Kernel NoChange",
						args: args{
							spec: &starlingxv1.HostProfileSpec{
								ProfileBaseAttributes: starlingxv1.ProfileBaseAttributes{
									Personality:  &controllerPersonality,
									SubFunctions: aioSubfuncs,
									Kernel:       &standardKernel,
								},
							},
						},
						nochange:            true,
						updatedSubfunctions: aioSubfuncs,
					},
					{
						name: "All-In-One Lowlatency Kernel NoChange",
						args: args{
							spec: &starlingxv1.HostProfileSpec{
								ProfileBaseAttributes: starlingxv1.ProfileBaseAttributes{
									Personality:  &controllerPersonality,
									SubFunctions: aiolowlatencySubfuncs,
									Kernel:       &lowlatencyKernel,
								},
							},
						},
						nochange:            true,
						updatedSubfunctions: aiolowlatencySubfuncs,
					},
					{
						name: "Worker Standard Kernel NoChange",
						args: args{
							spec: &starlingxv1.HostProfileSpec{
								ProfileBaseAttributes: starlingxv1.ProfileBaseAttributes{
									Personality:  &workerPersonality,
									SubFunctions: workerSubfuncs,
									Kernel:       &standardKernel,
								},
							},
						},
						nochange:            true,
						updatedSubfunctions: workerSubfuncs,
					},
					{
						name: "Worker Lowlatency Kernel NoChange",
						args: args{
							spec: &starlingxv1.HostProfileSpec{
								ProfileBaseAttributes: starlingxv1.ProfileBaseAttributes{
									Personality:  &workerPersonality,
									SubFunctions: workerlowlatencySubfuncs,
									Kernel:       &lowlatencyKernel,
								},
							},
						},
						nochange:            true,
						updatedSubfunctions: workerlowlatencySubfuncs,
					},
					{
						name: "All-In-One Standard Kernel Remove Lowlatency",
						args: args{
							spec: &starlingxv1.HostProfileSpec{
								ProfileBaseAttributes: starlingxv1.ProfileBaseAttributes{
									Personality:  &controllerPersonality,
									SubFunctions: aiolowlatencySubfuncs,
									Kernel:       &standardKernel,
								},
							},
						},
						nochange:            true,
						updatedSubfunctions: aioSubfuncs,
					},
					{
						name: "All-In-One Standard Kernel Add Lowlatency",
						args: args{
							spec: &starlingxv1.HostProfileSpec{
								ProfileBaseAttributes: starlingxv1.ProfileBaseAttributes{
									Personality:  &controllerPersonality,
									SubFunctions: aioSubfuncs,
									Kernel:       &lowlatencyKernel,
								},
							},
						},
						nochange:            true,
						updatedSubfunctions: aiolowlatencySubfuncs,
					},
					{
						name: "Worker Standard Kernel Remove Lowlatency",
						args: args{
							spec: &starlingxv1.HostProfileSpec{
								ProfileBaseAttributes: starlingxv1.ProfileBaseAttributes{
									Personality:  &workerPersonality,
									SubFunctions: workerlowlatencySubfuncs,
									Kernel:       &standardKernel,
								},
							},
						},
						nochange:            true,
						updatedSubfunctions: workerSubfuncs,
					},
					{
						name: "Worker Standard Kernel Add Lowlatency",
						args: args{
							spec: &starlingxv1.HostProfileSpec{
								ProfileBaseAttributes: starlingxv1.ProfileBaseAttributes{
									Personality:  &workerPersonality,
									SubFunctions: workerSubfuncs,
									Kernel:       &lowlatencyKernel,
								},
							},
						},
						nochange:            true,
						updatedSubfunctions: workerlowlatencySubfuncs,
					},
				}
				for _, tt := range tests {
					specBefore := tt.args.spec
					FixKernelSubfunction(tt.args.spec)
					specAfter := tt.args.spec
					Expect(reflect.DeepEqual(specBefore, specAfter)).To(Equal(tt.nochange))
					Expect(reflect.DeepEqual(tt.updatedSubfunctions, specAfter.SubFunctions)).To(BeTrue())
				}
			})
		})
	})

	Describe("FixProfileAttributes", func() {
		Context("with base profile spec data", func() {
			It("should remove the base profile successfully", func() {
				type args struct {
					a    *starlingxv1.HostProfileSpec
					b    *starlingxv1.HostProfileSpec
					c    *starlingxv1.HostProfileSpec
					info *v1info.HostInfo
				}
				base1 := "base-profile"
				base2 := "base-profile"
				tests := []struct {
					name string
					args args
					want args
				}{
					{
						name: "Profile has Base profile",
						args: args{
							a: &starlingxv1.HostProfileSpec{
								Base: &base1,
							},
							b: &starlingxv1.HostProfileSpec{
								Base: &base2,
							},
							c:    &starlingxv1.HostProfileSpec{},
							info: &v1info.HostInfo{},
						},
						want: args{
							a:    &starlingxv1.HostProfileSpec{},
							b:    &starlingxv1.HostProfileSpec{},
							c:    &starlingxv1.HostProfileSpec{},
							info: &v1info.HostInfo{},
						},
					},
					{
						name: "Profile has no Base profile",
						args: args{
							a:    &starlingxv1.HostProfileSpec{},
							b:    &starlingxv1.HostProfileSpec{},
							c:    &starlingxv1.HostProfileSpec{},
							info: &v1info.HostInfo{},
						},
						want: args{
							a:    &starlingxv1.HostProfileSpec{},
							b:    &starlingxv1.HostProfileSpec{},
							c:    &starlingxv1.HostProfileSpec{},
							info: &v1info.HostInfo{},
						},
					},
				}
				for _, tt := range tests {
					FixProfileAttributes(tt.args.a, tt.args.b, tt.args.c, tt.args.info)
					Expect(reflect.DeepEqual(tt.args.a, tt.want.a)).To(BeTrue())
					Expect(reflect.DeepEqual(tt.args.b, tt.want.b)).To(BeTrue())
					Expect(reflect.DeepEqual(tt.args.c, tt.want.c)).To(BeTrue())
				}
			})
		})
	})

	Describe("Test SyncIFNameByUuid", func() {
		Context("When uuid is the same", func() {
			It("Should copy interface name from current to profile", func() {
				profile := &starlingxv1.HostProfileSpec{
					Interfaces: &starlingxv1.InterfaceInfo{
						Ethernet: starlingxv1.EthernetList{
							{
								CommonInterfaceInfo: starlingxv1.CommonInterfaceInfo{
									Name: "ProfEthName",
									UUID: "EthUUID",
								},
								Port: starlingxv1.EthernetPortInfo{},
							},
						},
					},
				}
				ethPortName := "EthPortName"
				currEthName := "CurrEthName"
				current := &starlingxv1.HostProfileSpec{
					Interfaces: &starlingxv1.InterfaceInfo{
						Ethernet: starlingxv1.EthernetList{
							{
								CommonInterfaceInfo: starlingxv1.CommonInterfaceInfo{
									Name: currEthName,
									UUID: "EthUUID",
								},
								Port: starlingxv1.EthernetPortInfo{
									Name: ethPortName,
								},
							},
						},
					},
				}
				SyncIFNameByUuid(profile, current)
				Expect(profile.Interfaces.Ethernet[0].Port.Name).To(Equal(ethPortName))
				Expect(profile.Interfaces.Ethernet[0].CommonInterfaceInfo.Name).
					To(Equal(currEthName))

			})
		})
		Context("if uuid is not same", func() {
			It("Should not copy interface name from current to profile", func() {
				profPortName := "profPortName"
				profEthName := "profEthName"
				profile := &starlingxv1.HostProfileSpec{
					Interfaces: &starlingxv1.InterfaceInfo{
						Ethernet: starlingxv1.EthernetList{
							{
								CommonInterfaceInfo: starlingxv1.CommonInterfaceInfo{
									Name: profEthName,
									UUID: "EthUUID1",
								},
								Port: starlingxv1.EthernetPortInfo{
									Name: profPortName,
								},
							},
						},
					},
				}
				currPortName := "EthPortName"
				currEthName := "CurrEthName"
				current := &starlingxv1.HostProfileSpec{
					Interfaces: &starlingxv1.InterfaceInfo{
						Ethernet: starlingxv1.EthernetList{
							{
								CommonInterfaceInfo: starlingxv1.CommonInterfaceInfo{
									Name: currEthName,
									UUID: "EthUUID",
								},
								Port: starlingxv1.EthernetPortInfo{
									Name: currPortName,
								},
							},
						},
					},
				}
				SyncIFNameByUuid(profile, current)
				Expect(profile.Interfaces.Ethernet[0].Port.Name).To(Equal(profPortName))
				Expect(profile.Interfaces.Ethernet[0].CommonInterfaceInfo.Name).
					To(Equal(profEthName))

			})
		})
	})

	Describe("Test FillEmptyUuidbyName", func() {
		Context("When name is the same", func() {
			It("Should copy interface uuid from current to profile", func() {
				profile := &starlingxv1.HostProfileSpec{
					Interfaces: &starlingxv1.InterfaceInfo{
						Ethernet: starlingxv1.EthernetList{
							{
								CommonInterfaceInfo: starlingxv1.CommonInterfaceInfo{
									Name: "foo",
									UUID: "",
								},
							},
						},
						Bond: starlingxv1.BondList{
							{
								CommonInterfaceInfo: starlingxv1.CommonInterfaceInfo{
									Name: "foo",
									UUID: "",
								},
							},
						},
						VF: starlingxv1.VFList{
							{
								CommonInterfaceInfo: starlingxv1.CommonInterfaceInfo{
									Name: "foo",
									UUID: "",
								},
							},
						},
						VLAN: starlingxv1.VLANList{
							{
								CommonInterfaceInfo: starlingxv1.CommonInterfaceInfo{
									Name: "foo",
									UUID: "",
								},
							},
						},
					},
				}
				current := &starlingxv1.HostProfileSpec{
					Interfaces: &starlingxv1.InterfaceInfo{
						Ethernet: starlingxv1.EthernetList{
							{
								CommonInterfaceInfo: starlingxv1.CommonInterfaceInfo{
									Name: "foo",
									UUID: "bar1",
								},
							},
						},
						Bond: starlingxv1.BondList{
							{
								CommonInterfaceInfo: starlingxv1.CommonInterfaceInfo{
									Name: "foo",
									UUID: "bar2",
								},
							},
						},
						VF: starlingxv1.VFList{
							{
								CommonInterfaceInfo: starlingxv1.CommonInterfaceInfo{
									Name: "foo",
									UUID: "bar3",
								},
							},
						},
						VLAN: starlingxv1.VLANList{
							{
								CommonInterfaceInfo: starlingxv1.CommonInterfaceInfo{
									Name: "foo",
									UUID: "bar4",
								},
							},
						},
					},
				}
				FillEmptyUuidbyName(profile, current)
				Expect(
					profile.Interfaces.VF[0].CommonInterfaceInfo.UUID,
				).To(Equal("bar3"))
				Expect(
					profile.Interfaces.Bond[0].CommonInterfaceInfo.UUID,
				).To(Equal("bar2"))
				Expect(
					profile.Interfaces.VLAN[0].CommonInterfaceInfo.UUID,
				).To(Equal("bar4"))
				Expect(
					profile.Interfaces.Ethernet[0].CommonInterfaceInfo.UUID,
				).To(Equal("bar1"))
			})
		})
		Context("if name is not same", func() {
			It("Should not copy interface uuid from current to profile", func() {
				profile := &starlingxv1.HostProfileSpec{
					Interfaces: &starlingxv1.InterfaceInfo{
						Ethernet: starlingxv1.EthernetList{
							{
								CommonInterfaceInfo: starlingxv1.CommonInterfaceInfo{
									Name: "foo",
									UUID: "",
								},
							},
						},
						Bond: starlingxv1.BondList{
							{
								CommonInterfaceInfo: starlingxv1.CommonInterfaceInfo{
									Name: "foo",
									UUID: "",
								},
							},
						},
						VF: starlingxv1.VFList{
							{
								CommonInterfaceInfo: starlingxv1.CommonInterfaceInfo{
									Name: "foo",
									UUID: "",
								},
							},
						},
						VLAN: starlingxv1.VLANList{
							{
								CommonInterfaceInfo: starlingxv1.CommonInterfaceInfo{
									Name: "foo",
									UUID: "",
								},
							},
						},
					},
				}
				current := &starlingxv1.HostProfileSpec{
					Interfaces: &starlingxv1.InterfaceInfo{
						Ethernet: starlingxv1.EthernetList{
							{
								CommonInterfaceInfo: starlingxv1.CommonInterfaceInfo{
									Name: "foo2",
									UUID: "bar",
								},
							},
						},
						Bond: starlingxv1.BondList{
							{
								CommonInterfaceInfo: starlingxv1.CommonInterfaceInfo{
									Name: "foo2",
									UUID: "bar2",
								},
							},
						},
						VF: starlingxv1.VFList{
							{
								CommonInterfaceInfo: starlingxv1.CommonInterfaceInfo{
									Name: "foo2",
									UUID: "bar3",
								},
							},
						},
						VLAN: starlingxv1.VLANList{
							{
								CommonInterfaceInfo: starlingxv1.CommonInterfaceInfo{
									Name: "foo2",
									UUID: "bar4",
								},
							},
						},
					},
				}
				SyncIFNameByUuid(profile, current)
				Expect(profile.Interfaces.Ethernet[0].CommonInterfaceInfo.UUID).
					To(Equal(""))
				Expect(profile.Interfaces.Bond[0].CommonInterfaceInfo.UUID).
					To(Equal(""))
				Expect(profile.Interfaces.VF[0].CommonInterfaceInfo.UUID).
					To(Equal(""))
				Expect(profile.Interfaces.VLAN[0].CommonInterfaceInfo.UUID).
					To(Equal(""))
			})
		})
		Context("if uuid is not empty, that is the normal case", func() {
			It("the profile's uuid should not be updated", func() {
				profile := &starlingxv1.HostProfileSpec{
					Interfaces: &starlingxv1.InterfaceInfo{
						Ethernet: starlingxv1.EthernetList{
							{
								CommonInterfaceInfo: starlingxv1.CommonInterfaceInfo{
									Name: "foo",
									UUID: "bar",
								},
							},
						},
						Bond: starlingxv1.BondList{
							{
								CommonInterfaceInfo: starlingxv1.CommonInterfaceInfo{
									Name: "foo",
									UUID: "bar2",
								},
							},
						},
						VF: starlingxv1.VFList{
							{
								CommonInterfaceInfo: starlingxv1.CommonInterfaceInfo{
									Name: "foo",
									UUID: "bar3",
								},
							},
						},
						VLAN: starlingxv1.VLANList{
							{
								CommonInterfaceInfo: starlingxv1.CommonInterfaceInfo{
									Name: "foo",
									UUID: "bar4",
								},
							},
						},
					},
				}
				current := &starlingxv1.HostProfileSpec{
					Interfaces: &starlingxv1.InterfaceInfo{
						Ethernet: starlingxv1.EthernetList{
							{
								CommonInterfaceInfo: starlingxv1.CommonInterfaceInfo{
									Name: "foo2",
									UUID: "bar",
								},
							},
						},
						Bond: starlingxv1.BondList{
							{
								CommonInterfaceInfo: starlingxv1.CommonInterfaceInfo{
									Name: "foo2",
									UUID: "bar2",
								},
							},
						},
						VF: starlingxv1.VFList{
							{
								CommonInterfaceInfo: starlingxv1.CommonInterfaceInfo{
									Name: "foo2",
									UUID: "bar3",
								},
							},
						},
						VLAN: starlingxv1.VLANList{
							{
								CommonInterfaceInfo: starlingxv1.CommonInterfaceInfo{
									Name: "foo2",
									UUID: "bar4",
								},
							},
						},
					},
				}
				SyncIFNameByUuid(profile, current)
				Expect(profile.Interfaces.Ethernet[0].CommonInterfaceInfo.UUID).
					To(Equal("bar"))
				Expect(profile.Interfaces.Bond[0].CommonInterfaceInfo.UUID).
					To(Equal("bar2"))
				Expect(profile.Interfaces.VF[0].CommonInterfaceInfo.UUID).
					To(Equal("bar3"))
				Expect(profile.Interfaces.VLAN[0].CommonInterfaceInfo.UUID).
					To(Equal("bar4"))
			})
		})
	})
})
