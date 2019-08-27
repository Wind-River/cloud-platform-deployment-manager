/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019 Wind River Systems, Inc. */

package v1beta1

import (
	"testing"

	"github.com/onsi/gomega"
	"golang.org/x/net/context"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func TestStorageHostProfile(t *testing.T) {
	key := types.NamespacedName{
		Name:      "foo",
		Namespace: "default",
	}
	created := &HostProfile{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "foo",
			Namespace: "default",
		}}
	g := gomega.NewGomegaWithT(t)

	// Test Create
	fetched := &HostProfile{}
	g.Expect(c.Create(context.TODO(), created)).NotTo(gomega.HaveOccurred())

	g.Expect(c.Get(context.TODO(), key, fetched)).NotTo(gomega.HaveOccurred())
	g.Expect(fetched).To(gomega.Equal(created))

	// Test Updating the Labels
	updated := fetched.DeepCopy()
	updated.Labels = map[string]string{"hello": "world"}
	g.Expect(c.Update(context.TODO(), updated)).NotTo(gomega.HaveOccurred())

	g.Expect(c.Get(context.TODO(), key, fetched)).NotTo(gomega.HaveOccurred())
	g.Expect(fetched).To(gomega.Equal(updated))

	// Test Delete
	g.Expect(c.Delete(context.TODO(), fetched)).NotTo(gomega.HaveOccurred())
	g.Expect(c.Get(context.TODO(), key, fetched)).To(gomega.HaveOccurred())
}

func TestStorageHostProfileConsoleRegex(t *testing.T) {
	type args struct {
		console string
	}

	key := types.NamespacedName{
		Name:      "foo",
		Namespace: "default",
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{name: "empty",
			args:    args{console: ""},
			wantErr: false},
		{name: "serial-simple",
			args:    args{console: "ttyS0"},
			wantErr: false},
		{name: "serial-simple-double-digit",
			args:    args{console: "ttyS0"},
			wantErr: false},
		{name: "serial-missing-baud",
			args:    args{console: "ttyS0,"},
			wantErr: true},
		{name: "serial-with-baud",
			args:    args{console: "ttyS0,115200"},
			wantErr: false},
		{name: "serial-with-baud-and-parity",
			args:    args{console: "ttyS0,115200n8"},
			wantErr: false},
		{name: "serial-double-digit-with-baud",
			args:    args{console: "ttyS01,115200"},
			wantErr: false},
		{name: "serial-double-digit-with-baud-and-parity",
			args:    args{console: "ttyS01,115200n8"},
			wantErr: false},
		{name: "serial-invalid",
			args:    args{console: "tty"},
			wantErr: true},
		{name: "serial-invalid-numbers",
			args:    args{console: "1111"},
			wantErr: true},
		{name: "graphical-simple",
			args:    args{console: "tty0"},
			wantErr: false},
		{name: "graphical-double-digit",
			args:    args{console: "tty01"},
			wantErr: false},
		{name: "parallel-simple",
			args:    args{console: "lp0"},
			wantErr: false},
		{name: "parallel-double-digit",
			args:    args{console: "lp01"},
			wantErr: false},
		{name: "usb-simple",
			args:    args{console: "ttyUSB0"},
			wantErr: false},
		{name: "usb-simple-double-digit",
			args:    args{console: "ttyUSB0"},
			wantErr: false},
		{name: "usb-missing-baud",
			args:    args{console: "ttyUSB0,"},
			wantErr: true},
		{name: "usb-with-baud",
			args:    args{console: "ttyUSB0,115200"},
			wantErr: false},
		{name: "usb-with-baud-and-parity",
			args:    args{console: "ttyUSB0,115200n8"},
			wantErr: false},
		{name: "usb-double-digit-with-baud",
			args:    args{console: "ttyUSB01,115200"},
			wantErr: false},
		{name: "usb-double-digit-with-baud-and-parity",
			args:    args{console: "ttyUSB01,115200n8"},
			wantErr: false},
		{name: "usb-invalid",
			args:    args{console: "usb0"},
			wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			created := &HostProfile{
				ObjectMeta: metav1.ObjectMeta{
					Name:      tt.name,
					Namespace: "default",
				},
				Spec: HostProfileSpec{
					ProfileBaseAttributes: ProfileBaseAttributes{
						Console: &tt.args.console,
					},
				}}
			fetched := &HostProfile{}
			err := c.Create(context.TODO(), created)
			if tt.wantErr != (err != nil) {
				t.Errorf("failed to create HostProfile, error=%s", err)
			} else if !tt.wantErr {
				key.Name = tt.name
				err = c.Get(context.TODO(), key, fetched)
				if err != nil {
					t.Errorf("failed to get HostProfile, error=%s", err)
				} else if *fetched.Spec.Console != *created.Spec.Console {
					t.Errorf("console attribute mismatch; got=%s expected=%s",
						*fetched.Spec.Console, *created.Spec.Console)
				}
			}
		})
	}
}
