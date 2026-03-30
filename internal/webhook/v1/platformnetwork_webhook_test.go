/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019-2026 Wind River Systems, Inc. */
package v1

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	starlingxv1 "github.com/wind-river/cloud-platform-deployment-manager/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("PlatformNetwork webhook", func() {

	const (
		timeout  = time.Second * 30
		interval = time.Millisecond * 500
	)

	Context("when creating with valid data", func() {
		It("should accept valid PlatformNetwork", func() {
			key := types.NamespacedName{Name: "pn-valid", Namespace: "default"}

			created := &starlingxv1.PlatformNetwork{
				ObjectMeta: metav1.ObjectMeta{Name: "pn-valid", Namespace: "default"},
				Spec: starlingxv1.PlatformNetworkSpec{
					Type:                   "mgmt",
					Dynamic:                true,
					AssociatedAddressPools: []string{"mgmt-ipv4", "mgmt-ipv6"},
				},
			}
			Expect(k8sClient.Create(ctx, created)).To(Succeed())

			fetched := &starlingxv1.PlatformNetwork{}
			Eventually(func() bool {
				return k8sClient.Get(ctx, key, fetched) == nil
			}, timeout, interval).Should(BeTrue())
			Expect(fetched).To(Equal(created))

			Expect(k8sClient.Delete(ctx, fetched)).To(Succeed())
		})

		It("should default Dynamic to false when omitted", func() {
			key := types.NamespacedName{Name: "pn-no-dynamic", Namespace: "default"}

			created := &starlingxv1.PlatformNetwork{
				ObjectMeta: metav1.ObjectMeta{Name: "pn-no-dynamic", Namespace: "default"},
				Spec: starlingxv1.PlatformNetworkSpec{
					Type:                   "mgmt",
					AssociatedAddressPools: []string{"mgmt-ipv4", "mgmt-ipv6"},
				},
			}
			Expect(k8sClient.Create(ctx, created)).To(Succeed())

			fetched := &starlingxv1.PlatformNetwork{}
			Eventually(func() bool {
				return k8sClient.Get(ctx, key, fetched) == nil
			}, timeout, interval).Should(BeTrue())
			Expect(fetched.Spec.Dynamic).To(BeFalse())

			Expect(k8sClient.Delete(ctx, fetched)).To(Succeed())
		})
	})

	Context("when creating with invalid data", func() {
		It("should reject invalid NetworkType", func() {
			created := &starlingxv1.PlatformNetwork{
				ObjectMeta: metav1.ObjectMeta{Name: "pn-bad-type", Namespace: "default"},
				Spec: starlingxv1.PlatformNetworkSpec{
					Type:                   "dynamic",
					AssociatedAddressPools: []string{"mgmt-ipv4", "mgmt-ipv6"},
				},
			}
			Expect(k8sClient.Create(ctx, created)).ShouldNot(Succeed())
		})

		It("should reject missing AssociatedAddressPools", func() {
			created := &starlingxv1.PlatformNetwork{
				ObjectMeta: metav1.ObjectMeta{Name: "pn-no-pools", Namespace: "default"},
				Spec: starlingxv1.PlatformNetworkSpec{
					Type:    "mgmt",
					Dynamic: true,
				},
			}
			Expect(k8sClient.Create(ctx, created)).ShouldNot(Succeed())
		})

		It("should reject random NetworkType", func() {
			created := &starlingxv1.PlatformNetwork{
				ObjectMeta: metav1.ObjectMeta{Name: "pn-random", Namespace: "default"},
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
