/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019-2025 Wind River Systems, Inc. */
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

var _ = Describe("Platformnetwork controller", func() {

	const (
		timeout  = time.Second * 30
		interval = time.Millisecond * 500
	)

	Context("PlatformNetwork with all data", func() {
		It("Should be created successfully", func() {
			ctx := context.Background()
			key := types.NamespacedName{
				Name:      "foo",
				Namespace: "default",
			}
			created := &starlingxv1.PlatformNetwork{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: "default",
				},
				Spec: starlingxv1.PlatformNetworkSpec{
					Type:                   "mgmt",
					Dynamic:                true,
					AssociatedAddressPools: []string{"mgmt-ipv4", "mgmt-ipv6"},
				},
			}
			Expect(k8sClient.Create(ctx, created)).Should(Succeed())

			fetched := &starlingxv1.PlatformNetwork{}

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

	Context("Create PlatformNetwork Without NetworkType", func() {
		It("Should fail", func() {
			ctx := context.Background()
			created := &starlingxv1.PlatformNetwork{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: "default",
				},
				Spec: starlingxv1.PlatformNetworkSpec{
					Type:                   "dynamic",
					AssociatedAddressPools: []string{"mgmt-ipv4", "mgmt-ipv6"},
				},
			}
			Expect(k8sClient.Create(ctx, created)).ShouldNot(Succeed())
		})
	})

	Context("Create PlatformNetwork Without Dynamic", func() {
		It("PlatformNetwork.Spec.Dynamic should be false", func() {
			ctx := context.Background()
			created := &starlingxv1.PlatformNetwork{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: "default",
				},
				Spec: starlingxv1.PlatformNetworkSpec{
					Type:                   "mgmt",
					AssociatedAddressPools: []string{"mgmt-ipv4", "mgmt-ipv6"},
				},
			}
			Expect(k8sClient.Create(ctx, created)).Should(Succeed())

			fetched := &starlingxv1.PlatformNetwork{}
			key := types.NamespacedName{
				Name:      "foo",
				Namespace: "default",
			}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, key, fetched)
				return err == nil
			}, timeout, interval).Should(BeTrue())
			Expect(fetched.Spec.Dynamic).To(BeFalse())

		})
	})

	Context("Create PlatformNetwork Without AssociatedAddressPools", func() {
		It("Should fail", func() {
			ctx := context.Background()
			created := &starlingxv1.PlatformNetwork{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: "default",
				},
				Spec: starlingxv1.PlatformNetworkSpec{
					Type:    "mgmt",
					Dynamic: true,
				},
			}
			Expect(k8sClient.Create(ctx, created)).ShouldNot(Succeed())
		})
	})

	Context("Create PlatformNetwork With Random NetworkType", func() {
		It("Should fail", func() {
			ctx := context.Background()
			created := &starlingxv1.PlatformNetwork{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: "default",
				},
				Spec: starlingxv1.PlatformNetworkSpec{
					Type:                   "abc",
					Dynamic:                false,
					AssociatedAddressPools: []string{"mgmt-ipv4", "mgmt-ipv6"},
				},
			}
			Expect(k8sClient.Create(ctx, created)).ShouldNot(Succeed())
		})
	})
})
