/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2024 Wind River Systems, Inc. */
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

var _ = Describe("AddressPool controller", func() {
	const (
		timeout  = time.Second * 30
		interval = time.Millisecond * 500
	)

	Context("AddressPool with data", func() {
		It("Should created successfully", func() {
			ctx := context.Background()
			key := types.NamespacedName{
				Name:      "foo",
				Namespace: "default",
			}

			floating_address := "10.10.10.2"
			allocation_order := "random"
			spec := starlingxv1.AddressPoolSpec{
				Subnet:             "10.10.10.0",
				FloatingAddress:    &floating_address,
				Controller0Address: &floating_address,
				Controller1Address: &floating_address,
				Prefix:             24,
				Gateway:            &floating_address,
				Allocation: starlingxv1.AllocationInfo{
					Order: &allocation_order,
				},
			}

			spec.Allocation.Ranges = []starlingxv1.AllocationRange{starlingxv1.AllocationRange{
				Start: floating_address,
				End:   floating_address,
			}}

			created := &starlingxv1.AddressPool{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: "default",
				},
				Spec: spec}
			Expect(k8sClient.Create(ctx, created)).To(Succeed())

			expected := created.DeepCopy()

			fetched := &starlingxv1.AddressPool{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, key, fetched)
				return err == nil &&
					fetched.ObjectMeta.ResourceVersion != expected.ObjectMeta.ResourceVersion
			}, timeout, interval).Should(BeTrue())
			_, found := comm.ListIntersect(fetched.ObjectMeta.Finalizers, []string{AddressPoolFinalizerName})
			Expect(found).To(BeTrue())
		})
	})

})
