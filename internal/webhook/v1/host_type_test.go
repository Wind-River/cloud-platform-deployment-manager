/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019-2022, 2024-2026 Wind River Systems, Inc. */
package v1

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	starlingxv1 "github.com/wind-river/cloud-platform-deployment-manager/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("Datanetwork controller", func() {

	const (
		timeout  = time.Second * 30
		interval = time.Millisecond * 500
	)

	Context("with Host data", func() {
		It("Should be created successfully", func() {
			ctx := context.Background()
			key := types.NamespacedName{
				Name:      "foo",
				Namespace: "default",
			}
			bootMac := "01:02:03:04:05:06"
			bmAddress := "192.168.9.9"
			match := starlingxv1.MatchInfo{
				BootMAC: &bootMac,
			}
			bmType := "bmc"
			created := &starlingxv1.Host{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: "default",
				},
				Spec: starlingxv1.HostSpec{
					Profile: "some-profile",
					Match:   &match,
					Overrides: &starlingxv1.HostProfileSpec{
						Addresses: []starlingxv1.AddressInfo{
							starlingxv1.AddressInfo{Interface: "enp0s3", Address: "1.2.3.10", Prefix: 24},
						},
						BoardManagement: &starlingxv1.BMInfo{
							Type:    &bmType,
							Address: &bmAddress,
						},
					},
				}}
			Expect(k8sClient.Create(ctx, created)).To(Succeed())

			fetched := &starlingxv1.Host{}

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
	})
})
