/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019 Wind River Systems, Inc. */

package v1beta1

import (
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
