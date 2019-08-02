/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019 Wind River Systems, Inc. */

package host

import (
	"encoding/json"
	"reflect"
	"testing"

	starlingxv1beta1 "github.com/wind-river/titanium-deployment-manager/pkg/apis/starlingx/v1beta1"
)

func TestMergeProfiles(t *testing.T) {
	admin1 := "locked"
	location1 := "vbox"
	bmType1 := "bmc"
	bm1 := starlingxv1beta1.BMInfo{
		Type: &bmType1,
	}
	personality2 := "controller"
	admin2 := "unlocked"
	type args struct {
		a *starlingxv1beta1.HostProfileSpec
		b *starlingxv1beta1.HostProfileSpec
	}
	size1 := 1
	osds1 := starlingxv1beta1.OSDList{
		starlingxv1beta1.OSDInfo{
			Function: "osd",
			Path:     "/dev/sda",
		},
	}
	vgs1 := starlingxv1beta1.VolumeGroupList{
		starlingxv1beta1.VolumeGroupInfo{
			Name: "nova-local",
			PhysicalVolumes: starlingxv1beta1.PhysicalVolumeList{
				starlingxv1beta1.PhysicalVolumeInfo{
					Type: "disk",
					Path: "/dev/sdb",
				},
			},
		},
	}
	fs1 := starlingxv1beta1.FileSystemList{
		starlingxv1beta1.FileSystemInfo{
			Name: "backup",
			Size: 10,
		},
	}
	osds2 := starlingxv1beta1.OSDList{
		starlingxv1beta1.OSDInfo{
			Function: "osd",
			Path:     "/dev/sda",
			Journal:  &starlingxv1beta1.JournalInfo{Size: 10},
		},
		starlingxv1beta1.OSDInfo{
			Function: "osd",
			Path:     "/dev/sdc",
		},
	}
	vgs2 := starlingxv1beta1.VolumeGroupList{
		starlingxv1beta1.VolumeGroupInfo{
			Name: "nova-local",
			PhysicalVolumes: starlingxv1beta1.PhysicalVolumeList{
				starlingxv1beta1.PhysicalVolumeInfo{
					Type: "disk",
					Path: "/dev/sdd",
				},
				starlingxv1beta1.PhysicalVolumeInfo{
					Type: "partition",
					Path: "/dev/sde",
					Size: &size1,
				},
			},
		},
		starlingxv1beta1.VolumeGroupInfo{
			Name: "spare",
			PhysicalVolumes: starlingxv1beta1.PhysicalVolumeList{
				starlingxv1beta1.PhysicalVolumeInfo{
					Type: "disk",
					Path: "/dev/sdb",
				},
			},
		},
	}
	fs2 := starlingxv1beta1.FileSystemList{
		starlingxv1beta1.FileSystemInfo{
			Name: "backup",
			Size: 20,
		},
		starlingxv1beta1.FileSystemInfo{
			Name: "docker",
			Size: 5,
		},
	}
	tests := []struct {
		name    string
		args    args
		want    *starlingxv1beta1.HostProfileSpec
		wantErr bool
	}{
		{name: "basic",
			args: args{
				a: &starlingxv1beta1.HostProfileSpec{
					ProfileBaseAttributes: starlingxv1beta1.ProfileBaseAttributes{
						AdministrativeState: &admin1,
						Location:            &location1,
					},
					BoardManagement: &bm1,
				},
				b: &starlingxv1beta1.HostProfileSpec{
					ProfileBaseAttributes: starlingxv1beta1.ProfileBaseAttributes{
						AdministrativeState: &admin2,
						Personality:         &personality2,
					},
				},
			},
			want: &starlingxv1beta1.HostProfileSpec{
				ProfileBaseAttributes: starlingxv1beta1.ProfileBaseAttributes{
					AdministrativeState: &admin2,
					Personality:         &personality2,
					Location:            &location1,
				},
				BoardManagement: &bm1,
			},
			wantErr: false},
		{name: "interface-merge",
			args: args{a: &starlingxv1beta1.HostProfileSpec{
				Interfaces: &starlingxv1beta1.InterfaceInfo{
					Ethernet: []starlingxv1beta1.EthernetInfo{
						{CommonInterfaceInfo: starlingxv1beta1.CommonInterfaceInfo{Name: "eth0"}, Port: starlingxv1beta1.EthernetPortInfo{Name: "eth0"}},
						{CommonInterfaceInfo: starlingxv1beta1.CommonInterfaceInfo{Name: "eth1"}, Port: starlingxv1beta1.EthernetPortInfo{Name: "eth1"}}},
					VLAN: []starlingxv1beta1.VLANInfo{
						{CommonInterfaceInfo: starlingxv1beta1.CommonInterfaceInfo{Name: "vlan1"}, Lower: "eth0", VID: 1},
						{CommonInterfaceInfo: starlingxv1beta1.CommonInterfaceInfo{Name: "vlan2"}, Lower: "eth1", VID: 2}},
					Bond: []starlingxv1beta1.BondInfo{
						{CommonInterfaceInfo: starlingxv1beta1.CommonInterfaceInfo{Name: "bond0"}, Mode: "balanced", Members: []string{"eth0", "eth1"}},
						{CommonInterfaceInfo: starlingxv1beta1.CommonInterfaceInfo{Name: "bond1"}, Mode: "balanced", Members: []string{"eth2", "eth3"}}},
				}},
				b: &starlingxv1beta1.HostProfileSpec{
					Interfaces: &starlingxv1beta1.InterfaceInfo{
						Ethernet: []starlingxv1beta1.EthernetInfo{
							{CommonInterfaceInfo: starlingxv1beta1.CommonInterfaceInfo{Name: "mgmt0"}, Port: starlingxv1beta1.EthernetPortInfo{Name: "eth0"}},
							{CommonInterfaceInfo: starlingxv1beta1.CommonInterfaceInfo{Name: "cluster0"}, Port: starlingxv1beta1.EthernetPortInfo{Name: "eth2"}}},
						VLAN: []starlingxv1beta1.VLANInfo{
							{CommonInterfaceInfo: starlingxv1beta1.CommonInterfaceInfo{Name: "vlan1"}, Lower: "mgmt0", VID: 1},
							{CommonInterfaceInfo: starlingxv1beta1.CommonInterfaceInfo{Name: "vlan3"}, Lower: "cluster0", VID: 3}},
						Bond: []starlingxv1beta1.BondInfo{
							{CommonInterfaceInfo: starlingxv1beta1.CommonInterfaceInfo{Name: "bond0"}, Mode: "802.3ad", Members: []string{"eth10", "eth11"}},
							{CommonInterfaceInfo: starlingxv1beta1.CommonInterfaceInfo{Name: "bond2"}, Mode: "802.3ad", Members: []string{"eth12", "eth13"}}},
					},
				},
			},
			wantErr: false,
			want: &starlingxv1beta1.HostProfileSpec{
				Interfaces: &starlingxv1beta1.InterfaceInfo{
					Ethernet: []starlingxv1beta1.EthernetInfo{
						{CommonInterfaceInfo: starlingxv1beta1.CommonInterfaceInfo{Name: "mgmt0"}, Port: starlingxv1beta1.EthernetPortInfo{Name: "eth0"}},
						{CommonInterfaceInfo: starlingxv1beta1.CommonInterfaceInfo{Name: "eth1"}, Port: starlingxv1beta1.EthernetPortInfo{Name: "eth1"}},
						{CommonInterfaceInfo: starlingxv1beta1.CommonInterfaceInfo{Name: "cluster0"}, Port: starlingxv1beta1.EthernetPortInfo{Name: "eth2"}}},
					VLAN: []starlingxv1beta1.VLANInfo{
						{CommonInterfaceInfo: starlingxv1beta1.CommonInterfaceInfo{Name: "vlan1"}, Lower: "mgmt0", VID: 1},
						{CommonInterfaceInfo: starlingxv1beta1.CommonInterfaceInfo{Name: "vlan2"}, Lower: "eth1", VID: 2},
						{CommonInterfaceInfo: starlingxv1beta1.CommonInterfaceInfo{Name: "vlan3"}, Lower: "cluster0", VID: 3}},
					Bond: []starlingxv1beta1.BondInfo{
						{CommonInterfaceInfo: starlingxv1beta1.CommonInterfaceInfo{Name: "bond0"}, Mode: "802.3ad", Members: []string{"eth10", "eth11"}},
						{CommonInterfaceInfo: starlingxv1beta1.CommonInterfaceInfo{Name: "bond1"}, Mode: "balanced", Members: []string{"eth2", "eth3"}},
						{CommonInterfaceInfo: starlingxv1beta1.CommonInterfaceInfo{Name: "bond2"}, Mode: "802.3ad", Members: []string{"eth12", "eth13"}}},
				},
			},
		},
		{name: "storage-merge",
			args: args{
				a: &starlingxv1beta1.HostProfileSpec{
					Storage: &starlingxv1beta1.ProfileStorageInfo{
						Monitor:      &starlingxv1beta1.MonitorInfo{Size: &size1},
						OSDs:         &osds1,
						VolumeGroups: &vgs1,
						FileSystems:  &fs1,
					},
				},
				b: &starlingxv1beta1.HostProfileSpec{
					Storage: &starlingxv1beta1.ProfileStorageInfo{
						Monitor:      &starlingxv1beta1.MonitorInfo{Size: &size1},
						OSDs:         &osds2,
						VolumeGroups: &vgs2,
						FileSystems:  &fs2,
					},
				},
			},
			wantErr: false,
			want: &starlingxv1beta1.HostProfileSpec{
				Storage: &starlingxv1beta1.ProfileStorageInfo{
					Monitor:      &starlingxv1beta1.MonitorInfo{Size: &size1},
					OSDs:         &osds2,
					VolumeGroups: &vgs2,
					FileSystems:  &fs2,
				},
			},
		},

		{name: "processor-merge",
			args: args{
				a: &starlingxv1beta1.HostProfileSpec{
					Processors: starlingxv1beta1.ProcessorNodeList{
						starlingxv1beta1.ProcessorInfo{
							Node: 0,
							Functions: starlingxv1beta1.ProcessorFunctionList{
								starlingxv1beta1.ProcessorFunctionInfo{
									Function: "vswitch",
									Count:    1,
								},
							},
						},
					},
				},
				b: &starlingxv1beta1.HostProfileSpec{
					Processors: starlingxv1beta1.ProcessorNodeList{
						starlingxv1beta1.ProcessorInfo{
							Node: 0,
							Functions: starlingxv1beta1.ProcessorFunctionList{
								starlingxv1beta1.ProcessorFunctionInfo{
									Function: "vswitch",
									Count:    2,
								},
								starlingxv1beta1.ProcessorFunctionInfo{
									Function: "platform",
									Count:    1,
								},
							},
						},
						starlingxv1beta1.ProcessorInfo{
							Node: 1,
							Functions: starlingxv1beta1.ProcessorFunctionList{
								starlingxv1beta1.ProcessorFunctionInfo{
									Function: "vswitch",
									Count:    4,
								},
								starlingxv1beta1.ProcessorFunctionInfo{
									Function: "platform",
									Count:    2,
								},
							},
						},
					},
				},
			},
			wantErr: false,
			want: &starlingxv1beta1.HostProfileSpec{
				Processors: starlingxv1beta1.ProcessorNodeList{
					starlingxv1beta1.ProcessorInfo{
						Node: 0,
						Functions: starlingxv1beta1.ProcessorFunctionList{
							starlingxv1beta1.ProcessorFunctionInfo{
								Function: "vswitch",
								Count:    2,
							},
							starlingxv1beta1.ProcessorFunctionInfo{
								Function: "platform",
								Count:    1,
							},
						},
					},
					starlingxv1beta1.ProcessorInfo{
						Node: 1,
						Functions: starlingxv1beta1.ProcessorFunctionList{
							starlingxv1beta1.ProcessorFunctionInfo{
								Function: "vswitch",
								Count:    4,
							},
							starlingxv1beta1.ProcessorFunctionInfo{
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
				a: &starlingxv1beta1.HostProfileSpec{
					Memory: starlingxv1beta1.MemoryNodeList{
						starlingxv1beta1.MemoryNodeInfo{
							Node: 0,
							Functions: starlingxv1beta1.MemoryFunctionList{
								starlingxv1beta1.MemoryFunctionInfo{
									Function:  "vswitch",
									PageSize:  "2MB",
									PageCount: 16,
								},
								starlingxv1beta1.MemoryFunctionInfo{
									Function:  "vswitch",
									PageSize:  "1GB",
									PageCount: 1,
								},
							},
						},
					},
				},
				b: &starlingxv1beta1.HostProfileSpec{
					Memory: starlingxv1beta1.MemoryNodeList{
						starlingxv1beta1.MemoryNodeInfo{
							Node: 0,
							Functions: starlingxv1beta1.MemoryFunctionList{
								starlingxv1beta1.MemoryFunctionInfo{
									Function:  "vswitch",
									PageSize:  "2MB",
									PageCount: 128,
								},
								starlingxv1beta1.MemoryFunctionInfo{
									Function:  "platform",
									PageSize:  "2MB",
									PageCount: 32,
								},
							},
						},
						starlingxv1beta1.MemoryNodeInfo{
							Node: 1,
							Functions: starlingxv1beta1.MemoryFunctionList{
								starlingxv1beta1.MemoryFunctionInfo{
									Function:  "vswitch",
									PageSize:  "2MB",
									PageCount: 128,
								},
								starlingxv1beta1.MemoryFunctionInfo{
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
			want: &starlingxv1beta1.HostProfileSpec{
				Memory: starlingxv1beta1.MemoryNodeList{
					starlingxv1beta1.MemoryNodeInfo{
						Node: 0,
						Functions: starlingxv1beta1.MemoryFunctionList{
							starlingxv1beta1.MemoryFunctionInfo{
								Function:  "vswitch",
								PageSize:  "2MB",
								PageCount: 128,
							},
							starlingxv1beta1.MemoryFunctionInfo{
								Function:  "vswitch",
								PageSize:  "1GB",
								PageCount: 1,
							},
							starlingxv1beta1.MemoryFunctionInfo{
								Function:  "platform",
								PageSize:  "2MB",
								PageCount: 32,
							},
						},
					},
					starlingxv1beta1.MemoryNodeInfo{
						Node: 1,
						Functions: starlingxv1beta1.MemoryFunctionList{
							starlingxv1beta1.MemoryFunctionInfo{
								Function:  "vswitch",
								PageSize:  "2MB",
								PageCount: 128,
							},
							starlingxv1beta1.MemoryFunctionInfo{
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
				a: &starlingxv1beta1.HostProfileSpec{
					Addresses: starlingxv1beta1.AddressList{
						{Address: "1.2.3.4", Prefix: 24, Interface: "eth0"},
						{Address: "10.20.30.40", Prefix: 24, Interface: "eth1"},
					},
				},
				b: &starlingxv1beta1.HostProfileSpec{
					Addresses: starlingxv1beta1.AddressList{
						{Address: "10.20.30.40", Prefix: 32, Interface: "eth2"},
						{Address: "fd00::40", Prefix: 64, Interface: "eth1"},
					},
				},
			},
			wantErr: false,
			want: &starlingxv1beta1.HostProfileSpec{
				Addresses: starlingxv1beta1.AddressList{
					{Address: "1.2.3.4", Prefix: 24, Interface: "eth0"},
					{Address: "10.20.30.40", Prefix: 32, Interface: "eth2"},
					{Address: "fd00::40", Prefix: 64, Interface: "eth1"},
				},
			},
		},
		{name: "routes-merge",
			args: args{
				a: &starlingxv1beta1.HostProfileSpec{
					Routes: starlingxv1beta1.RouteList{
						{Network: "10.10.10.0", Prefix: 24, Gateway: "10.10.10.1", Interface: "eth0"},
						{Network: "172.16.0.0", Prefix: 16, Gateway: "172.16.0.1", Interface: "eth1"},
					},
				},
				b: &starlingxv1beta1.HostProfileSpec{
					Routes: starlingxv1beta1.RouteList{
						{Network: "10.10.10.0", Prefix: 24, Gateway: "10.10.10.2", Interface: "eth0"},
						{Network: "172.16.0.0", Prefix: 24, Gateway: "172.16.0.1", Interface: "eth2"},
						{Network: "fd00:1::", Prefix: 64, Gateway: "fd00:1::1", Interface: "eth1"},
					},
				},
			},
			wantErr: false,
			want: &starlingxv1beta1.HostProfileSpec{
				Routes: starlingxv1beta1.RouteList{
					{Network: "10.10.10.0", Prefix: 24, Gateway: "10.10.10.2", Interface: "eth0"},
					{Network: "172.16.0.0", Prefix: 16, Gateway: "172.16.0.1", Interface: "eth1"},
					{Network: "172.16.0.0", Prefix: 24, Gateway: "172.16.0.1", Interface: "eth2"},
					{Network: "fd00:1::", Prefix: 64, Gateway: "fd00:1::1", Interface: "eth1"},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := MergeProfiles(tt.args.a, tt.args.b)
			if (err != nil) != tt.wantErr {
				t.Errorf("MergeProfiles() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				gotBuf, _ := json.Marshal(got)
				wantBuf, _ := json.Marshal(tt.want)
				t.Errorf("MergeProfiles(), got = %s, want = %s", gotBuf, wantBuf)

			} else if got != nil && got.DeepEqual(tt.want) == false {
				gotBuf, _ := json.Marshal(got)
				wantBuf, _ := json.Marshal(tt.want)
				t.Errorf("Profile.DeepEqual() disagrees with reflect.DeepEqual(), got = %s, want = %s", gotBuf, wantBuf)
			}
		})
	}
}
