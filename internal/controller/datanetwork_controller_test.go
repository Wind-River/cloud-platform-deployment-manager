/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2022, 2024-2026 Wind River Systems, Inc. */
package controller

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	starlingxv1 "github.com/wind-river/cloud-platform-deployment-manager/api/v1"
	comm "github.com/wind-river/cloud-platform-deployment-manager/common"
)

var _ = Describe("Datanetwork controller", func() {

	const (
		timeout  = time.Second * 30
		interval = time.Millisecond * 500
	)

	Context("with DataNetwork data", func() {
		It("should be created successfully", func() {
			ctx := context.Background()
			key := types.NamespacedName{
				Name:      "foo",
				Namespace: "default",
			}
			mtu := 1500
			description := "This is a sample description"

			created := &starlingxv1.DataNetwork{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: "default",
				},
				Spec: starlingxv1.DataNetworkSpec{
					Type:        "flat",
					Description: &description,
					MTU:         &mtu,
				}}
			Expect(k8sClient.Create(ctx, created)).To(Succeed())

			fetched := &starlingxv1.DataNetwork{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, key, fetched)
				if err != nil {
					return false
				}
				_, found := comm.ListIntersect(fetched.ObjectMeta.Finalizers, []string{DataNetworkFinalizerName})
				return found
			}, timeout, interval).Should(BeTrue())
		})
	})
})
