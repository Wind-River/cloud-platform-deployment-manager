/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019 Wind River Systems, Inc. */

package manager

import (
	"testing"
)

func Test_getNextCount(t *testing.T) {
	type args struct {
		value string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{name: "empty value",
			args: args{value: ""},
			want: "1",
		},
		{name: "integer value",
			args: args{value: "1"},
			want: "2"},
		{name: "alphanumeric value",
			args: args{value: "foobar"},
			want: "1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getNextCount(tt.args.value); got != tt.want {
				t.Errorf("getNextCount() = %v, want %v", got, tt.want)
			}
		})
	}
}
