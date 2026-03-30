/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2022, 2024-2026 Wind River Systems, Inc. */
package v1

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	starlingxv1 "github.com/wind-river/cloud-platform-deployment-manager/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("PtpInterface webhook", func() {

	const (
		timeout  = time.Second * 30
		interval = time.Millisecond * 500
	)

	Context("when creating with valid parameters", func() {
		It("should accept key=value parameters", func() {
			key := types.NamespacedName{Name: "ptp-iface-valid", Namespace: "default"}

			created := &starlingxv1.PtpInterface{
				ObjectMeta: metav1.ObjectMeta{Name: "ptp-iface-valid", Namespace: "default"},
				Spec: starlingxv1.PtpInterfaceSpec{
					PtpInstance:         "ptp4l-inst",
					InterfaceParameters: []string{"delay_mechanism=E2E", "network_transport=UDPv4"},
				},
			}
			Expect(k8sClient.Create(ctx, created)).To(Succeed())

			fetched := &starlingxv1.PtpInterface{}
			Eventually(func() bool {
				return k8sClient.Get(ctx, key, fetched) == nil
			}, timeout, interval).Should(BeTrue())
			Expect(fetched).To(Equal(created))

			Expect(k8sClient.Delete(ctx, fetched)).To(Succeed())
		})

		It("should accept empty parameters", func() {
			key := types.NamespacedName{Name: "ptp-iface-empty", Namespace: "default"}

			created := &starlingxv1.PtpInterface{
				ObjectMeta: metav1.ObjectMeta{Name: "ptp-iface-empty", Namespace: "default"},
				Spec: starlingxv1.PtpInterfaceSpec{
					PtpInstance: "ptp4l-inst",
				},
			}
			Expect(k8sClient.Create(ctx, created)).To(Succeed())

			fetched := &starlingxv1.PtpInterface{}
			Eventually(func() bool {
				return k8sClient.Get(ctx, key, fetched) == nil
			}, timeout, interval).Should(BeTrue())

			Expect(k8sClient.Delete(ctx, fetched)).To(Succeed())
		})
	})

	Context("when creating with invalid parameters", func() {
		It("should reject parameter missing equals sign", func() {
			created := &starlingxv1.PtpInterface{
				ObjectMeta: metav1.ObjectMeta{Name: "ptp-iface-noeq", Namespace: "default"},
				Spec: starlingxv1.PtpInterfaceSpec{
					PtpInstance:         "ptp4l-inst",
					InterfaceParameters: []string{"invalid_param"},
				},
			}
			err := k8sClient.Create(ctx, created)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Parameters must come in the form"))
		})

		It("should reject parameter starting with equals sign", func() {
			created := &starlingxv1.PtpInterface{
				ObjectMeta: metav1.ObjectMeta{Name: "ptp-iface-eqstart", Namespace: "default"},
				Spec: starlingxv1.PtpInterfaceSpec{
					PtpInstance:         "ptp4l-inst",
					InterfaceParameters: []string{"=value"},
				},
			}
			err := k8sClient.Create(ctx, created)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Parameters must come in the form"))
		})

		It("should reject duplicate parameter keys", func() {
			created := &starlingxv1.PtpInterface{
				ObjectMeta: metav1.ObjectMeta{Name: "ptp-iface-dup", Namespace: "default"},
				Spec: starlingxv1.PtpInterfaceSpec{
					PtpInstance:         "ptp4l-inst",
					InterfaceParameters: []string{"delay_mechanism=E2E", "delay_mechanism=P2P"},
				},
			}
			err := k8sClient.Create(ctx, created)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("duplicate parameter keys are not allowed"))
		})

		It("should reject parameter with multiple equals signs", func() {
			created := &starlingxv1.PtpInterface{
				ObjectMeta: metav1.ObjectMeta{Name: "ptp-iface-multi-eq", Namespace: "default"},
				Spec: starlingxv1.PtpInterfaceSpec{
					PtpInstance:         "ptp4l-inst",
					InterfaceParameters: []string{"key=val=extra"},
				},
			}
			err := k8sClient.Create(ctx, created)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Parameters must come in the form"))
		})
	})

	Context("when updating", func() {
		It("should accept valid parameters", func() {
			key := types.NamespacedName{Name: "ptp-iface-update", Namespace: "default"}

			created := &starlingxv1.PtpInterface{
				ObjectMeta: metav1.ObjectMeta{Name: "ptp-iface-update", Namespace: "default"},
				Spec: starlingxv1.PtpInterfaceSpec{
					PtpInstance:         "ptp4l-inst",
					InterfaceParameters: []string{"delay_mechanism=E2E"},
				},
			}
			Expect(k8sClient.Create(ctx, created)).To(Succeed())

			fetched := &starlingxv1.PtpInterface{}
			Eventually(func() bool {
				return k8sClient.Get(ctx, key, fetched) == nil
			}, timeout, interval).Should(BeTrue())

			fetched.Spec.InterfaceParameters = []string{"delay_mechanism=P2P", "network_transport=L2"}
			Expect(k8sClient.Update(ctx, fetched)).To(Succeed())

			Expect(k8sClient.Delete(ctx, fetched)).To(Succeed())
		})

		It("should reject invalid parameters", func() {
			key := types.NamespacedName{Name: "ptp-iface-bad-upd", Namespace: "default"}

			created := &starlingxv1.PtpInterface{
				ObjectMeta: metav1.ObjectMeta{Name: "ptp-iface-bad-upd", Namespace: "default"},
				Spec: starlingxv1.PtpInterfaceSpec{
					PtpInstance:         "ptp4l-inst",
					InterfaceParameters: []string{"delay_mechanism=E2E"},
				},
			}
			Expect(k8sClient.Create(ctx, created)).To(Succeed())

			fetched := &starlingxv1.PtpInterface{}
			Eventually(func() bool {
				return k8sClient.Get(ctx, key, fetched) == nil
			}, timeout, interval).Should(BeTrue())

			fetched.Spec.InterfaceParameters = []string{"no_equals"}
			Expect(k8sClient.Update(ctx, fetched)).ToNot(Succeed())

			Expect(k8sClient.Delete(ctx, fetched)).To(Succeed())
		})
	})

	Context("when deleting", func() {
		It("should allow delete", func() {
			v := &PtpInterfaceCustomValidator{}
			obj := &starlingxv1.PtpInterface{
				ObjectMeta: metav1.ObjectMeta{Name: "ptp-iface-del", Namespace: "default"},
				Spec:       starlingxv1.PtpInterfaceSpec{PtpInstance: "ptp4l-inst"},
			}
			warnings, err := v.ValidateDelete(ctx, obj)
			Expect(err).ToNot(HaveOccurred())
			Expect(warnings).To(BeNil())
		})
	})
})
