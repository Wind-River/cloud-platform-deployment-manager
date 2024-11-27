/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019-2022 Wind River Systems, Inc. */
package v1

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("Datanetwork controller", func() {

	const (
		timeout  = time.Second * 30
		interval = time.Millisecond * 250
	)
	ctx := context.Background()
	Context("DataNetwork with data", func() {
		It("Should created successfully", func() {
			key := types.NamespacedName{
				Name:      "foo",
				Namespace: "default",
			}
			mtu := 1500
			description := "This is a sample description"

			created := &DataNetwork{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: "default",
				},
				Spec: DataNetworkSpec{
					Type:        "flat",
					Description: &description,
					MTU:         &mtu,
				}}
			Expect(k8sClient.Create(ctx, created)).To(Succeed())

			fetched := &DataNetwork{}

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
