/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019 Wind River Systems, Inc. */

package host

import (
	"reflect"
	"testing"
)

func Test_listDelta(t *testing.T) {
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
			gotAdded, gotRemoved, gotSame := listDelta(tt.args.a, tt.args.b)
			if !reflect.DeepEqual(gotAdded, tt.wantAdded) {
				t.Errorf("listDelta() gotAdded = %v, want %v", gotAdded, tt.wantAdded)
			}
			if !reflect.DeepEqual(gotRemoved, tt.wantRemoved) {
				t.Errorf("listDelta() gotRemoved = %v, want %v", gotRemoved, tt.wantRemoved)
			}
			if !reflect.DeepEqual(gotSame, tt.wantSame) {
				t.Errorf("listDelta() gotSame = %v, want %v", gotSame, tt.wantSame)
			}
		})
	}
}

func Test_listChanged(t *testing.T) {
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
			if got := listChanged(tt.args.a, tt.args.b); got != tt.want {
				t.Errorf("listChanged() = %v, want %v", got, tt.want)
			}
		})
	}
}
