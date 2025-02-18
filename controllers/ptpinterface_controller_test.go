/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2022, 2024-2025 Wind River Systems, Inc. */
package controllers

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

var _ = Describe("PtpInterface controller", func() {

	const (
		timeout  = time.Second * 30
		interval = time.Millisecond * 500
	)

	Context("PtpInstance with data", func() {
		It("Should created successfully", func() {
			ctx := context.Background()
			key := types.NamespacedName{
				Name:      "foo",
				Namespace: "default",
			}
			created := &starlingxv1.PtpInterface{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: "default",
				},
				Spec: starlingxv1.PtpInterfaceSpec{
					PtpInstance:         "ptp1",
					InterfaceParameters: []string{"foo1=bar1", "foo2=bar2"},
				}}
			Expect(k8sClient.Create(ctx, created)).To(Succeed())

			expected := created.DeepCopy()

			fetched := &starlingxv1.PtpInterface{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, key, fetched)
				return err == nil &&
					fetched.ObjectMeta.ResourceVersion != expected.ObjectMeta.ResourceVersion
			}, timeout, interval).Should(BeTrue())
			_, found := comm.ListIntersect(fetched.ObjectMeta.Finalizers, []string{PtpInterfaceFinalizerName})
			Expect(found).To(BeTrue())
		})
	})
})
