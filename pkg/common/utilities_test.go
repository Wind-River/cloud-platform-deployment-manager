/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019 Wind River Systems, Inc. */

package common

import (
	"reflect"
	"testing"
)

func TestIsIPv4(t *testing.T) {
	type args struct {
		address string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{name: "any",
			args: args{address: "0.0.0.0"},
			want: true},
		{name: "normal",
			args: args{address: "1.2.3.4"},
			want: true},
		{name: "ipv6-to-ipv4-mapped",
			args: args{address: "::ffff:1.2.3.4"},
			want: true},
		{name: "invalid-fqdn",
			args: args{address: "a.b.c.d"},
			want: false},
		{name: "invalid-too-many-octets",
			args: args{address: "1.2.3.4.5"},
			want: false},
		{name: "invalid-ipv6",
			args: args{address: "fd00::1"},
			want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsIPv4(tt.args.address); got != tt.want {
				t.Errorf("IsIPv4() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsIPv6(t *testing.T) {
	type args struct {
		address string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{name: "any",
			args: args{address: "::"},
			want: true},
		{name: "normal",
			args: args{address: "fd00::1:2:3:4"},
			want: true},
		{name: "ipv6-to-ipv4-mapped",
			args: args{address: "::ffff:1.2.3.4"},
			want: false},
		{name: "invalid-fqdn",
			args: args{address: "a.b.c.d"},
			want: false},
		{name: "invalid-too-many-octets",
			args: args{address: "a:b:c:d:e:f:g:h:i"},
			want: false},
		{name: "invalid-too-many-expansions",
			args: args{address: "fd00::1::2"},
			want: false},
		{name: "invalid-ipv6",
			args: args{address: "fd00::asdf"},
			want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsIPv6(tt.args.address); got != tt.want {
				t.Errorf("IsIPv6() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_ListDelta(t *testing.T) {
	empty := make([]string, 0)
	type args struct {
		a []string
		b []string
	}
	tests := []struct {
		name        string
		args        args
		wantAdded   []string
		wantRemoved []string
		wantSame    []string
	}{
		{name: "add_first",
			args: args{a: []string{},
				b: []string{"1"}},
			wantAdded:   []string{"1"},
			wantRemoved: empty,
			wantSame:    empty},
		{name: "add_second",
			args: args{a: []string{"2"},
				b: []string{"1", "2"}},
			wantAdded:   []string{"1"},
			wantRemoved: empty,
			wantSame:    []string{"2"}},
		{name: "add_multiple",
			args: args{a: []string{"2"},
				b: []string{"1", "2", "3", "4"}},
			wantAdded:   []string{"1", "3", "4"},
			wantRemoved: empty,
			wantSame:    []string{"2"}},
		{name: "remove_last",
			args: args{a: []string{"1"},
				b: []string{}},
			wantAdded:   empty,
			wantRemoved: []string{"1"},
			wantSame:    empty},
		{name: "remove_one",
			args: args{a: []string{"1", "2"},
				b: []string{"2"}},
			wantAdded:   empty,
			wantRemoved: []string{"1"},
			wantSame:    []string{"2"}},
		{name: "remove_multiple",
			args: args{a: []string{"1", "2", "3", "4"},
				b: []string{"2"}},
			wantAdded:   empty,
			wantRemoved: []string{"1", "3", "4"},
			wantSame:    []string{"2"}},
		{name: "identical",
			args: args{a: []string{"1"},
				b: []string{"1"}},
			wantAdded:   empty,
			wantRemoved: empty,
			wantSame:    []string{"1"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotAdded, gotRemoved, gotSame := ListDelta(tt.args.a, tt.args.b)
			if !reflect.DeepEqual(gotAdded, tt.wantAdded) {
				t.Errorf("ListDelta() gotAdded = %v, want %v", gotAdded, tt.wantAdded)
			}
			if !reflect.DeepEqual(gotRemoved, tt.wantRemoved) {
				t.Errorf("ListDelta() gotRemoved = %v, want %v", gotRemoved, tt.wantRemoved)
			}
			if !reflect.DeepEqual(gotSame, tt.wantSame) {
				t.Errorf("ListDelta() gotSame = %v, want %v", gotSame, tt.wantSame)
			}
		})
	}
}

func Test_ListChanged(t *testing.T) {
	type args struct {
		a []string
		b []string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{name: "compare_with_empty",
			args: args{a: []string{},
				b: []string{"1"}},
			want: true},
		{name: "compare_with_non_empty",
			args: args{a: []string{"2"},
				b: []string{"1", "2"}},
			want: true},
		{name: "compare_to_empty",
			args: args{a: []string{"1"},
				b: []string{}},
			want: true},
		{name: "compare_to_non_empty",
			args: args{a: []string{"1", "2"},
				b: []string{"2"}},
			want: true},
		{name: "identical",
			args: args{a: []string{"1"},
				b: []string{"1"}},
			want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ListChanged(tt.args.a, tt.args.b); got != tt.want {
				t.Errorf("ListChanged() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestListIntersect(t *testing.T) {
	type args struct {
		a []string
		b []string
	}
	tests := []struct {
		name  string
		args  args
		want  []string
		want1 bool
	}{
		{name: "same",
			args: args{a: []string{"a", "b"},
				b: []string{"b", "a"}},
			want:  []string{"a", "b"},
			want1: true},
		{name: "empty",
			args: args{a: []string{},
				b: []string{}},
			want:  []string{},
			want1: false},
		{name: "intersect",
			args: args{a: []string{"a", "b"},
				b: []string{"b", "c"}},
			want:  []string{"b"},
			want1: true},
		{name: "no-intersect",
			args: args{a: []string{"a", "b"},
				b: []string{"c", "d"}},
			want:  []string{},
			want1: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := ListIntersect(tt.args.a, tt.args.b)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ListIntersect() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("ListIntersect() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func TestComparePartitionPaths(t *testing.T) {
	type args struct {
		a string
		b string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{name: "same-disks",
			args: args{a: "/dev/disk/by-path/pci-0000:00:1f.2-ata-5.0",
				b: "/dev/disk/by-path/pci-0000:00:1f.2-ata-5.0"},
			want: true},
		{name: "different-disks",
			args: args{a: "/dev/disk/by-path/pci-0000:00:1f.2-ata-5.0",
				b: "/dev/disk/by-path/pci-0000:00:1f.3-ata-6.0"},
			want: false},
		{name: "same-disk-partitions",
			args: args{a: "/dev/disk/by-path/pci-0000:00:1f.2-ata-5.0-part1",
				b: "/dev/disk/by-path/pci-0000:00:1f.2-ata-5.0-part1"},
			want: true},
		{name: "different-disks-partitions",
			args: args{a: "/dev/disk/by-path/pci-0000:00:1f.2-ata-5.0-part1",
				b: "/dev/disk/by-path/pci-0000:00:1f.3-ata-6.0-part1"},
			want: false},
		{name: "different-partitions",
			args: args{a: "/dev/disk/by-path/pci-0000:00:1f.2-ata-5.0-part1",
				b: "/dev/disk/by-path/pci-0000:00:1f.2-ata-5.0-part2"},
			want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ComparePartitionPaths(tt.args.a, tt.args.b); got != tt.want {
				t.Errorf("ComparePartitionPaths() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestContainsString(t *testing.T) {
	type args struct {
		slice []string
		s     string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{name: "included",
			args: args{slice: []string{"abc", "def"},
				s: "abc"},
			want: true},
		{name: "not-included",
			args: args{slice: []string{"abc", "def"},
				s: "xyz"},
			want: false},
		{name: "substring-not-included",
			args: args{slice: []string{"abc", "def"},
				s: "a"},
			want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ContainsString(tt.args.slice, tt.args.s); got != tt.want {
				t.Errorf("ContainsString() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRemoveString(t *testing.T) {
	type args struct {
		slice []string
		s     string
	}
	tests := []struct {
		name       string
		args       args
		wantResult []string
	}{
		{name: "included",
			args: args{slice: []string{"abc", "def"},
				s: "abc"},
			wantResult: []string{"def"}},
		{name: "not-included",
			args: args{slice: []string{"abc", "def"},
				s: "xyz"},
			wantResult: []string{"abc", "def"}},
		{name: "substring-not-included",
			args: args{slice: []string{"abc", "def"},
				s: "a"},
			wantResult: []string{"abc", "def"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if gotResult := RemoveString(tt.args.slice, tt.args.s); !reflect.DeepEqual(gotResult, tt.wantResult) {
				t.Errorf("RemoveString() = %v, want %v", gotResult, tt.wantResult)
			}
		})
	}
}
