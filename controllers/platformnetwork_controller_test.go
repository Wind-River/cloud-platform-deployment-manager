/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2022 Wind River Systems, Inc. */
package controllers

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	starlingxv1 "github.com/wind-river/cloud-platform-deployment-manager/api/v1"
	comm "github.com/wind-river/cloud-platform-deployment-manager/common"
)

var _ = Describe("Platformnetwork controller", func() {

	const (
		timeout  = time.Second * 10
		interval = time.Millisecond * 250
	)

	Context("PlatformNetwork with data", func() {
		It("Should created successfully", func() {
			ctx := context.Background()
			key := types.NamespacedName{
				Name:      "foo",
				Namespace: "default",
			}
			order := "sequential"
			created := &starlingxv1.PlatformNetwork{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: "default",
				},
				Spec: starlingxv1.PlatformNetworkSpec{
					Type:   "mgmt",
					Subnet: "1.2.3.0",
					Prefix: 24,
					Allocation: starlingxv1.AllocationInfo{
						Type:  "static",
						Order: &order,
						Ranges: []starlingxv1.AllocationRange{
							{Start: "1.2.3.10", End: "1.2.3.49"},
							{Start: "1.2.3.129", End: "1.2.3.139"}},
					},
				},
			}
			Expect(k8sClient.Create(ctx, created)).To(Succeed())

			expected := created.DeepCopy()

			fetched := &starlingxv1.PlatformNetwork{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, key, fetched)
				return err == nil &&
					fetched.ObjectMeta.ResourceVersion != expected.ObjectMeta.ResourceVersion
			}, timeout, interval).Should(BeTrue())
			_, found := comm.ListIntersect(fetched.ObjectMeta.Finalizers, []string{PlatformNetworkFinalizerName})
			Expect(found).To(BeTrue())
		})
	})
})
