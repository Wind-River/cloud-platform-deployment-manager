/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2022-2024 Wind River Systems, Inc. */
package controllers

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	starlingxv1 "github.com/wind-river/cloud-platform-deployment-manager/api/v1"
	comm "github.com/wind-river/cloud-platform-deployment-manager/common"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("Platformnetwork controller", func() {
	const (
		timeout  = time.Second * 30
		interval = time.Millisecond * 500
	)

	Context("PlatformNetwork with data", func() {
		It("Should created successfully", func() {
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
					AssociatedAddressPools: []string{"management-ipv4", "management-ipv6"},
				}}
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
