/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2022,2025 Wind River Systems, Inc. */
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

var _ = Describe("PtpInstance controller", func() {

	const (
		timeout  = time.Second * 30
		interval = time.Millisecond * 500
	)

	Describe("Create PtpInstance", func() {
		Context("with single section data", func() {
			It("Should created successfully", func() {
				ctx := context.Background()
				key := types.NamespacedName{
					Name:      "foo",
					Namespace: "default",
				}
				created := &starlingxv1.PtpInstance{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "foo",
						Namespace: "default",
					},
					Spec: starlingxv1.PtpInstanceSpec{
						Service:            "ptp4l",
						InstanceParameters: map[string][]string{"global": []string{"domainNumber=24", "clientOnly=0"}},
					}}
				Expect(k8sClient.Create(ctx, created)).To(Succeed())

				expected := created.DeepCopy()

				fetched := &starlingxv1.PtpInstance{}
				Eventually(func() bool {
					err := k8sClient.Get(ctx, key, fetched)
					return err == nil &&
						fetched.ObjectMeta.ResourceVersion != expected.ObjectMeta.ResourceVersion
				}, timeout, interval).Should(BeTrue())
				_, found := comm.ListIntersect(fetched.ObjectMeta.Finalizers, []string{PtpInstanceFinalizerName})
				Expect(found).To(BeTrue())
			})
		})
	})
	Describe("Create PtpInstance", func() {
		Context("with multiple section data", func() {
			It("Should create successfully", func() {
				ctx := context.Background()
				key := types.NamespacedName{
					Name:      "bar",
					Namespace: "default",
				}
				created := &starlingxv1.PtpInstance{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "bar",
						Namespace: "default",
					},
					Spec: starlingxv1.PtpInstanceSpec{
						Service: "ptp4l",
						InstanceParameters: map[string][]string{
							"global": []string{"domainNumber=24", "clientOnly=0"},
							"unicast_master_table_1": []string{
								"table_id=1",
								"UDPv4=1.2.3.4", "UDPv4=2.3.4.5",
								"L2=00:01:FF:00:01:CD", "L2=00:02:FF:00:01:CD",
								"UDPv6=ffff::1", "UDPv6=ffff::2",
								"peer_address=::1"},
						},
					},
				}
				Expect(k8sClient.Create(ctx, created)).To(Succeed())

				expected := created.DeepCopy()

				fetched := &starlingxv1.PtpInstance{}
				Eventually(func() bool {
					err := k8sClient.Get(ctx, key, fetched)
					return err == nil &&
						fetched.ObjectMeta.ResourceVersion != expected.ObjectMeta.ResourceVersion
				}, timeout, interval).Should(BeTrue())
				_, found := comm.ListIntersect(fetched.ObjectMeta.Finalizers, []string{PtpInstanceFinalizerName})
				Expect(found).To(BeTrue())
			})
		})
	})
})
