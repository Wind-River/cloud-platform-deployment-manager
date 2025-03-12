/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2024-2025 Wind River Systems, Inc. */
package v1

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("Addresspool controller", func() {

	const (
		timeout  = time.Second * 30
		interval = time.Millisecond * 500
	)

	Context("Addresspool with all data", func() {
		It("Should be created successfully", func() {
			ctx := context.Background()
			key := types.NamespacedName{
				Name:      "foo",
				Namespace: "default",
			}

			floating_address := "192.168.204.2"
			controller0_address := "192.168.204.3"
			controller1_address := "192.168.204.4"
			gateway := "192.168.204.1"
			allocation_order := "random"

			created := &AddressPool{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: "default",
				},
				Spec: AddressPoolSpec{
					Subnet:             "192.168.204.0",
					FloatingAddress:    &floating_address,
					Controller0Address: &controller0_address,
					Controller1Address: &controller1_address,
					Prefix:             24,
					Gateway:            &gateway,
					Allocation: AllocationInfo{
						Order:  &allocation_order,
						Ranges: []AllocationRange{{Start: "192.168.204.2", End: "192.168.204.254"}},
					},
				},
			}
			Expect(k8sClient.Create(ctx, created)).Should(Succeed())

			fetched := &AddressPool{}

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
