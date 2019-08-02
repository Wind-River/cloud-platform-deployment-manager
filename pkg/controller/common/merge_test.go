/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019 Wind River Systems, Inc. */

package common

import (
	"encoding/json"
	"github.com/imdario/mergo"
	"github.com/wind-river/titanium-deployment-manager/pkg/apis/starlingx/v1beta1"
	"reflect"
	"testing"
)

func Test_MergeTransformer(t *testing.T) {
	type args struct {
		dst *v1beta1.HostProfileSpec
		src *v1beta1.HostProfileSpec
	}
	base1 := "base1"
	base2 := "base2"
	tests := []struct {
		name       string
		args       args
		wantErr    bool
		wantStruct v1beta1.HostProfileSpec
	}{
		{name: "string-overwrite-nil-pointer",
			args: args{
				dst: &v1beta1.HostProfileSpec{
					Base: nil,
				},
				src: &v1beta1.HostProfileSpec{
					Base: &base1,
				},
			},
			wantErr: false,
			wantStruct: v1beta1.HostProfileSpec{
				Base: &base1,
			},
		},
		{name: "string-no-overwrite-from-nil-pointer",
			args: args{
				dst: &v1beta1.HostProfileSpec{
					Base: &base1,
				},
				src: &v1beta1.HostProfileSpec{
					Base: nil,
				},
			},
			wantErr: false,
			wantStruct: v1beta1.HostProfileSpec{
				Base: &base1,
			},
		},
		{name: "string-overwrite",
			args: args{
				dst: &v1beta1.HostProfileSpec{
					Base: &base1,
				},
				src: &v1beta1.HostProfileSpec{
					Base: &base2,
				},
			},
			wantErr: false,
			wantStruct: v1beta1.HostProfileSpec{
				Base: &base2,
			},
		},
		{name: "struct-overwrite-nil",
			args: args{
				dst: &v1beta1.HostProfileSpec{
					Storage: nil,
				},
				src: &v1beta1.HostProfileSpec{
					Storage: &v1beta1.ProfileStorageInfo{
						FileSystems: &v1beta1.FileSystemList{
							v1beta1.FileSystemInfo{Name: "backup"},
						},
					},
				},
			},
			wantErr: false,
			wantStruct: v1beta1.HostProfileSpec{
				Storage: &v1beta1.ProfileStorageInfo{
					FileSystems: &v1beta1.FileSystemList{
						v1beta1.FileSystemInfo{Name: "backup"},
					},
				},
			},
		},
		{name: "struct-no-overwrite-from-nil",
			args: args{
				dst: &v1beta1.HostProfileSpec{
					Storage: &v1beta1.ProfileStorageInfo{
						FileSystems: &v1beta1.FileSystemList{
							v1beta1.FileSystemInfo{Name: "backup"},
						},
					},
				},
				src: &v1beta1.HostProfileSpec{
					Storage: nil,
				},
			},
			wantErr: false,
			wantStruct: v1beta1.HostProfileSpec{
				Storage: &v1beta1.ProfileStorageInfo{
					FileSystems: &v1beta1.FileSystemList{
						v1beta1.FileSystemInfo{Name: "backup"},
					},
				},
			},
		},
		{name: "slice-pointer-overwrite-nil",
			args: args{
				dst: &v1beta1.HostProfileSpec{
					Storage: &v1beta1.ProfileStorageInfo{},
				},
				src: &v1beta1.HostProfileSpec{
					Storage: &v1beta1.ProfileStorageInfo{
						FileSystems: &v1beta1.FileSystemList{
							v1beta1.FileSystemInfo{Name: "backup"},
						},
					},
				},
			},
			wantErr: false,
			wantStruct: v1beta1.HostProfileSpec{
				Storage: &v1beta1.ProfileStorageInfo{
					FileSystems: &v1beta1.FileSystemList{
						v1beta1.FileSystemInfo{Name: "backup"},
					},
				},
			},
		},
		{name: "slice-pointer-no-overwrite-from-nil",
			args: args{
				dst: &v1beta1.HostProfileSpec{
					Storage: &v1beta1.ProfileStorageInfo{
						FileSystems: &v1beta1.FileSystemList{
							v1beta1.FileSystemInfo{Name: "backup"},
						},
					},
				},
				src: &v1beta1.HostProfileSpec{
					Storage: &v1beta1.ProfileStorageInfo{},
				},
			},
			wantErr: false,
			wantStruct: v1beta1.HostProfileSpec{
				Storage: &v1beta1.ProfileStorageInfo{
					FileSystems: &v1beta1.FileSystemList{
						v1beta1.FileSystemInfo{Name: "backup"},
					},
				},
			},
		},
		{name: "slice-pointer-with-key-merge",
			args: args{
				dst: &v1beta1.HostProfileSpec{
					Storage: &v1beta1.ProfileStorageInfo{
						FileSystems: &v1beta1.FileSystemList{
							v1beta1.FileSystemInfo{Name: "backup", Size: 10},
						},
					},
				},
				src: &v1beta1.HostProfileSpec{
					Storage: &v1beta1.ProfileStorageInfo{
						FileSystems: &v1beta1.FileSystemList{
							v1beta1.FileSystemInfo{Name: "backup", Size: 20},
						},
					},
				},
			},
			wantErr: false,
			wantStruct: v1beta1.HostProfileSpec{
				Storage: &v1beta1.ProfileStorageInfo{
					FileSystems: &v1beta1.FileSystemList{
						v1beta1.FileSystemInfo{Name: "backup", Size: 20},
					},
				},
			},
		},
		{name: "slice-pointer-with-key-append",
			args: args{
				dst: &v1beta1.HostProfileSpec{
					Storage: &v1beta1.ProfileStorageInfo{
						FileSystems: &v1beta1.FileSystemList{
							v1beta1.FileSystemInfo{Name: "backup"},
						},
					},
				},
				src: &v1beta1.HostProfileSpec{
					Storage: &v1beta1.ProfileStorageInfo{
						FileSystems: &v1beta1.FileSystemList{
							v1beta1.FileSystemInfo{Name: "docker"},
						},
					},
				},
			},
			wantErr: false,
			wantStruct: v1beta1.HostProfileSpec{
				Storage: &v1beta1.ProfileStorageInfo{
					FileSystems: &v1beta1.FileSystemList{
						v1beta1.FileSystemInfo{Name: "backup"},
						v1beta1.FileSystemInfo{Name: "docker"},
					},
				},
			},
		},
		{name: "slice-without-key",
			args: args{
				dst: &v1beta1.HostProfileSpec{
					ProfileBaseAttributes: v1beta1.ProfileBaseAttributes{
						SubFunctions: []string{"sub1", "sub2"},
					},
				},
				src: &v1beta1.HostProfileSpec{},
			},
			wantErr: false,
			wantStruct: v1beta1.HostProfileSpec{
				ProfileBaseAttributes: v1beta1.ProfileBaseAttributes{
					SubFunctions: []string{"sub1", "sub2"},
				},
			},
		},
		{name: "slice-without-key-replace",
			args: args{
				dst: &v1beta1.HostProfileSpec{
					ProfileBaseAttributes: v1beta1.ProfileBaseAttributes{
						SubFunctions: []string{"sub1", "sub2"},
					},
				},
				src: &v1beta1.HostProfileSpec{
					ProfileBaseAttributes: v1beta1.ProfileBaseAttributes{
						SubFunctions: []string{"sub10", "sub20"},
					},
				},
			},
			wantErr: false,
			wantStruct: v1beta1.HostProfileSpec{
				ProfileBaseAttributes: v1beta1.ProfileBaseAttributes{
					SubFunctions: []string{"sub10", "sub20"},
				},
			},
		},
		{name: "slice-without-key-reset-to-empty",
			args: args{
				dst: &v1beta1.HostProfileSpec{
					ProfileBaseAttributes: v1beta1.ProfileBaseAttributes{
						SubFunctions: []string{"sub1", "sub2"},
					},
				},
				src: &v1beta1.HostProfileSpec{
					ProfileBaseAttributes: v1beta1.ProfileBaseAttributes{
						SubFunctions: []string{},
					},
				},
			},
			wantErr: false,
			wantStruct: v1beta1.HostProfileSpec{
				ProfileBaseAttributes: v1beta1.ProfileBaseAttributes{
					SubFunctions: []string{},
				},
			},
		},
		{name: "slice-without-key-overwrite-empty",
			args: args{
				dst: &v1beta1.HostProfileSpec{
					ProfileBaseAttributes: v1beta1.ProfileBaseAttributes{
						SubFunctions: []string{},
					},
				},
				src: &v1beta1.HostProfileSpec{
					ProfileBaseAttributes: v1beta1.ProfileBaseAttributes{
						SubFunctions: []string{"sub1", "sub2"},
					},
				},
			},
			wantErr: false,
			wantStruct: v1beta1.HostProfileSpec{
				ProfileBaseAttributes: v1beta1.ProfileBaseAttributes{
					SubFunctions: []string{"sub1", "sub2"},
				},
			},
		},
		{name: "slice-with-key-no-overwrite-from-nil",
			args: args{
				dst: &v1beta1.HostProfileSpec{
					Interfaces: &v1beta1.InterfaceInfo{
						Ethernet: v1beta1.EthernetList{
							v1beta1.EthernetInfo{
								CommonInterfaceInfo: v1beta1.CommonInterfaceInfo{Name: "eth0"},
								Port:                v1beta1.EthernetPortInfo{Name: "eth0"},
							},
						}}},
				src: &v1beta1.HostProfileSpec{},
			},
			wantErr: false,
			wantStruct: v1beta1.HostProfileSpec{
				Interfaces: &v1beta1.InterfaceInfo{
					Ethernet: v1beta1.EthernetList{
						v1beta1.EthernetInfo{
							CommonInterfaceInfo: v1beta1.CommonInterfaceInfo{Name: "eth0"},
							Port:                v1beta1.EthernetPortInfo{Name: "eth0"},
						},
					},
				},
			},
		},
		{name: "slice-with-key-overwrite-nil",
			args: args{
				dst: &v1beta1.HostProfileSpec{},
				src: &v1beta1.HostProfileSpec{
					Interfaces: &v1beta1.InterfaceInfo{
						Ethernet: v1beta1.EthernetList{
							v1beta1.EthernetInfo{
								CommonInterfaceInfo: v1beta1.CommonInterfaceInfo{Name: "eth0"},
								Port:                v1beta1.EthernetPortInfo{Name: "eth0"},
							},
						},
					},
				},
			},
			wantErr: false,
			wantStruct: v1beta1.HostProfileSpec{
				Interfaces: &v1beta1.InterfaceInfo{
					Ethernet: v1beta1.EthernetList{
						v1beta1.EthernetInfo{
							CommonInterfaceInfo: v1beta1.CommonInterfaceInfo{Name: "eth0"},
							Port:                v1beta1.EthernetPortInfo{Name: "eth0"},
						},
					},
				},
			},
		},
		{name: "slice-with-key-overwrite-empty",
			args: args{
				dst: &v1beta1.HostProfileSpec{
					Interfaces: &v1beta1.InterfaceInfo{
						Ethernet: v1beta1.EthernetList{}},
				},
				src: &v1beta1.HostProfileSpec{
					Interfaces: &v1beta1.InterfaceInfo{
						Ethernet: v1beta1.EthernetList{
							v1beta1.EthernetInfo{
								CommonInterfaceInfo: v1beta1.CommonInterfaceInfo{Name: "eth0"},
								Port:                v1beta1.EthernetPortInfo{Name: "eth0"},
							},
						},
					},
				},
			},
			wantErr: false,
			wantStruct: v1beta1.HostProfileSpec{
				Interfaces: &v1beta1.InterfaceInfo{
					Ethernet: v1beta1.EthernetList{
						v1beta1.EthernetInfo{
							CommonInterfaceInfo: v1beta1.CommonInterfaceInfo{Name: "eth0"},
							Port:                v1beta1.EthernetPortInfo{Name: "eth0"},
						},
					},
				},
			},
		},
		{name: "slice-with-key-overwrite-with-empty",
			args: args{
				dst: &v1beta1.HostProfileSpec{
					Interfaces: &v1beta1.InterfaceInfo{
						Ethernet: v1beta1.EthernetList{
							v1beta1.EthernetInfo{
								CommonInterfaceInfo: v1beta1.CommonInterfaceInfo{Name: "eth0"},
								Port:                v1beta1.EthernetPortInfo{Name: "eth0"},
							},
						},
					},
				},
				src: &v1beta1.HostProfileSpec{
					Interfaces: &v1beta1.InterfaceInfo{
						Ethernet: v1beta1.EthernetList{}},
				},
			},
			wantErr: false,
			wantStruct: v1beta1.HostProfileSpec{
				Interfaces: &v1beta1.InterfaceInfo{
					Ethernet: v1beta1.EthernetList{},
				},
			},
		},
		{name: "slice-with-key-merge-elements",
			args: args{
				dst: &v1beta1.HostProfileSpec{
					Interfaces: &v1beta1.InterfaceInfo{
						Ethernet: v1beta1.EthernetList{
							v1beta1.EthernetInfo{
								CommonInterfaceInfo: v1beta1.CommonInterfaceInfo{Name: "eth0"},
								Port:                v1beta1.EthernetPortInfo{Name: "eth0"},
							},
						},
					},
				},
				src: &v1beta1.HostProfileSpec{
					Interfaces: &v1beta1.InterfaceInfo{
						Ethernet: v1beta1.EthernetList{
							v1beta1.EthernetInfo{
								CommonInterfaceInfo: v1beta1.CommonInterfaceInfo{Name: "mgmt0"},
								Port:                v1beta1.EthernetPortInfo{Name: "eth0"},
							},
						},
					},
				},
			},
			wantErr: false,
			wantStruct: v1beta1.HostProfileSpec{
				Interfaces: &v1beta1.InterfaceInfo{
					Ethernet: v1beta1.EthernetList{
						v1beta1.EthernetInfo{
							CommonInterfaceInfo: v1beta1.CommonInterfaceInfo{Name: "mgmt0"},
							Port:                v1beta1.EthernetPortInfo{Name: "eth0"},
						},
					},
				},
			},
		},
		{name: "slice-with-key-append-elements",
			args: args{
				dst: &v1beta1.HostProfileSpec{
					Interfaces: &v1beta1.InterfaceInfo{
						Ethernet: v1beta1.EthernetList{
							v1beta1.EthernetInfo{
								CommonInterfaceInfo: v1beta1.CommonInterfaceInfo{Name: "eth0"},
								Port:                v1beta1.EthernetPortInfo{Name: "eth0"},
							},
						},
					},
				},
				src: &v1beta1.HostProfileSpec{
					Interfaces: &v1beta1.InterfaceInfo{
						Ethernet: v1beta1.EthernetList{
							v1beta1.EthernetInfo{
								CommonInterfaceInfo: v1beta1.CommonInterfaceInfo{Name: "mgmt0"},
								Port:                v1beta1.EthernetPortInfo{Name: "eth0"},
							},
							v1beta1.EthernetInfo{
								CommonInterfaceInfo: v1beta1.CommonInterfaceInfo{Name: "eth1"},
								Port:                v1beta1.EthernetPortInfo{Name: "eth1"},
							},
						},
					},
				},
			},
			wantErr: false,
			wantStruct: v1beta1.HostProfileSpec{
				Interfaces: &v1beta1.InterfaceInfo{
					Ethernet: v1beta1.EthernetList{
						v1beta1.EthernetInfo{
							CommonInterfaceInfo: v1beta1.CommonInterfaceInfo{Name: "mgmt0"},
							Port:                v1beta1.EthernetPortInfo{Name: "eth0"},
						},
						v1beta1.EthernetInfo{
							CommonInterfaceInfo: v1beta1.CommonInterfaceInfo{Name: "eth1"},
							Port:                v1beta1.EthernetPortInfo{Name: "eth1"},
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := mergo.Merge(tt.args.dst, tt.args.src, mergo.WithOverride, mergo.WithTransformers(DefaultMergeTransformer)); (err != nil) != tt.wantErr {
				t.Errorf("mergo.Merge() error = %v, wantErr %v", err, tt.wantErr)
			} else if reflect.DeepEqual(*tt.args.dst, tt.wantStruct) == false {
				dstBuf, err := json.Marshal(*tt.args.dst)
				if err != nil {
					t.Errorf("failed to encode dstBuf")
				}
				wantBuf, _ := json.Marshal(tt.wantStruct)
				t.Errorf("mergo.Merg() mismatch got = %s, want = %s", string(dstBuf), string(wantBuf))
			}
		})
	}
}
