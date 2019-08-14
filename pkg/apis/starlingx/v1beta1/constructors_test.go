/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019 Wind River Systems, Inc. */

package v1beta1

import (
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/disks"
	"github.com/wind-river/titanium-deployment-manager/pkg/platform"
	"reflect"
	"testing"
)

func Test_stripPartitionNumber(t *testing.T) {
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
		t.Run(tt.name, func(t *testing.T) {
			if got := stripPartitionNumber(tt.args.path); got != tt.want {
				t.Errorf("stripPartitionNumber() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_fixDevicePath(t *testing.T) {
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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := fixDevicePath(tt.args.path, tt.args.host); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("fixDevicePath() = %v, want %v", got, tt.want)
			}
		})
	}
}
