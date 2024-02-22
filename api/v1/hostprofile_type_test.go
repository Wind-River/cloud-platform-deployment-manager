/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019-2022 Wind River Systems, Inc. */
package v1

import (
	"context"
	"time"

	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/clusters"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("HostProfile controller", func() {

	const (
		timeout  = time.Second * 10
		interval = time.Millisecond * 250
	)
	Context("Test GetClusterName with clusterName not nil", func() {
		It("gives cluserName of OSDInfo", func() {
			name := "ClusterName"
			in := &OSDInfo{
				ClusterName: &name,
			}
			want := name
			got := in.GetClusterName()
			Expect(got).To(Equal(want))
		})
	})

	Context("Test GetClusterName with clusterName  nil", func() {
		It("gives cluserName as CephClusterName", func() {

			in := &OSDInfo{
				ClusterName: nil,
			}
			want := clusters.CephClusterName
			got := in.GetClusterName()
			Expect(got).To(Equal(want))
		})
	})

	Context("Test StringsToPtpInstanceItemList", func() {
		It("It gives PtpInstanceItemList from input string array", func() {
			strArr := []string{"random1", "random2"}
			want := []PtpInstanceItem{"random1", "random2"}
			wantList := PtpInstanceItemList(want)
			got := StringsToPtpInstanceItemList(strArr)
			Expect(got).To(Equal(wantList))
		})
	})

	Context("Test SubFunctionFromString", func() {
		It("Gives subfunction from string", func() {
			str := "randomString"
			want := SubFunction(str)
			got := SubFunctionFromString(str)
			Expect(got).To(Equal(want))
		})
	})

	Context("HostProfile with data", func() {
		It("Should created successfully", func() {
			ctx := context.Background()
			key := types.NamespacedName{
				Name:      "foo",
				Namespace: "default",
			}
			created := &HostProfile{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: "default",
				}}
			Expect(k8sClient.Create(ctx, created)).Should(Succeed())

			fetched := &HostProfile{}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, key, fetched)
				return err == nil
			}, timeout, interval).Should(BeTrue())
			Expect(fetched).To(Equal(created))

			updated := fetched.DeepCopy()
			updated.Labels = map[string]string{"hello": "world"}
			Expect(k8sClient.Update(ctx, updated)).To(Succeed())

			Eventually(func() bool {
				err := k8sClient.Get(ctx, key, fetched)
				return err == nil
			}, timeout, interval).Should(BeTrue())
			Expect(fetched).To(Equal(updated))

			Expect(k8sClient.Delete(ctx, fetched)).To(Succeed())
			Eventually(func() bool {
				err := k8sClient.Get(ctx, key, fetched)
				return err == nil
			}, timeout, interval).Should(BeFalse())
		})
		It("Should created successfully", func() {
			ctx := context.Background()
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
					args:    args{console: "ttyS0"},
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
				if tt.wantErr {
					Expect(k8sClient.Create(ctx, created)).Error()
				} else {
					Expect(k8sClient.Create(ctx, created)).To(Succeed())
					key.Name = tt.name
					Eventually(func() bool {
						err := k8sClient.Get(ctx, key, fetched)
						return err == nil
					}, timeout, interval).Should(BeTrue())
					Expect(fetched.Spec.Console).To(Equal(created.Spec.Console))
				}
			}
		})
	})
})
