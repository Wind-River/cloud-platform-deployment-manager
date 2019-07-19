/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019 Wind River Systems, Inc. */

package host

import (
	"reflect"
	"testing"

	starlingxv1beta1 "github.com/wind-river/titanium-deployment-manager/pkg/apis/starlingx/v1beta1"
)

func TestMergeProfiles(t *testing.T) {
	admin1 := "locked"
	location1 := "vbox"
	interfaces1 := starlingxv1beta1.InterfaceInfo{
		Ethernet: []starlingxv1beta1.EthernetInfo{{CommonInterfaceInfo: starlingxv1beta1.CommonInterfaceInfo{Name: "eth0"}, Port: starlingxv1beta1.EthernetPortInfo{Name: "eth0"}}},
	}
	filesystems1 := starlingxv1beta1.FileSystemList{
		starlingxv1beta1.FileSystemInfo{
			Name: "backup",
			Size: 10,
		},
		starlingxv1beta1.FileSystemInfo{
			Name: "docker",
			Size: 10,
		},
	}
	osd1 := starlingxv1beta1.OSDList{
		starlingxv1beta1.OSDInfo{
			Function: "osd",
			Path:     "/dev/sda",
		},
	}
	profile1 := starlingxv1beta1.HostProfileSpec{
		ProfileBaseAttributes: starlingxv1beta1.ProfileBaseAttributes{
			AdministrativeState: &admin1,
			Location:            &location1,
		},
		Interfaces: &interfaces1,
		Storage: &starlingxv1beta1.ProfileStorageInfo{
			OSDs:        &osd1,
			FileSystems: &filesystems1,
		},
	}
	personality2 := "controller"
	admin2 := "unlocked"
	interfaces2 := starlingxv1beta1.InterfaceInfo{
		Ethernet: []starlingxv1beta1.EthernetInfo{{CommonInterfaceInfo: starlingxv1beta1.CommonInterfaceInfo{Name: "mgmt0"}, Port: starlingxv1beta1.EthernetPortInfo{Name: "eth0"}}},
	}
	filesystems2 := starlingxv1beta1.FileSystemList{
		starlingxv1beta1.FileSystemInfo{
			Name: "backup",
			Size: 20,
		},
	}
	vg2 := starlingxv1beta1.VolumeGroupList{
		starlingxv1beta1.VolumeGroupInfo{
			Name: "nova-local",
			PhysicalVolumes: starlingxv1beta1.PhysicalVolumeList{
				starlingxv1beta1.PhysicalVolumeInfo{
					Path: "/dev/sda",
					Type: "disk",
				},
			},
		},
	}
	profile2 := starlingxv1beta1.HostProfileSpec{
		ProfileBaseAttributes: starlingxv1beta1.ProfileBaseAttributes{
			AdministrativeState: &admin2,
			Personality:         &personality2,
		},
		Interfaces: &interfaces2,
		Storage: &starlingxv1beta1.ProfileStorageInfo{
			VolumeGroups: &vg2,
			FileSystems:  &filesystems2,
		},
	}
	filesystems3 := starlingxv1beta1.FileSystemList{
		starlingxv1beta1.FileSystemInfo{
			Name: "backup",
			Size: 20,
		},
		starlingxv1beta1.FileSystemInfo{
			Name: "docker",
			Size: 10,
		},
	}
	result := starlingxv1beta1.HostProfileSpec{
		ProfileBaseAttributes: starlingxv1beta1.ProfileBaseAttributes{
			AdministrativeState: &admin2,
			Personality:         &personality2,
			Location:            &location1,
		},
		Interfaces: &interfaces2,
		Storage: &starlingxv1beta1.ProfileStorageInfo{
			OSDs:         &osd1,
			VolumeGroups: &vg2,
			FileSystems:  &filesystems3,
		},
	}
	type args struct {
		a *starlingxv1beta1.HostProfileSpec
		b *starlingxv1beta1.HostProfileSpec
	}
	tests := []struct {
		name    string
		args    args
		want    *starlingxv1beta1.HostProfileSpec
		wantErr bool
	}{
		{name: "simple",
			args:    args{a: &profile1, b: &profile2},
			want:    &result,
			wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := MergeProfiles(tt.args.a, tt.args.b)
			if (err != nil) != tt.wantErr {
				t.Errorf("MergeProfiles() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("MergeProfiles() = %v, want %v", got, tt.want)
			}
			if got != nil && got.DeepEqual(tt.want) == false {
				t.Errorf("Profile.DeepEqual() disagrees with reflect.DeepEqual()")
			}
		})
	}
}
