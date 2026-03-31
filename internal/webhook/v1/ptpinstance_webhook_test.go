/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019-2022, 2024-2026 Wind River Systems, Inc. */
package v1

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	starlingxv1 "github.com/wind-river/cloud-platform-deployment-manager/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("PtpInstance webhook", func() {

	const (
		timeout  = time.Second * 30
		interval = time.Millisecond * 500
	)

	Context("when creating with valid parameters", func() {
		It("should accept single section data", func() {
			key := types.NamespacedName{Name: "foo", Namespace: "default"}

			created := &starlingxv1.PtpInstance{
				ObjectMeta: metav1.ObjectMeta{Name: "foo", Namespace: "default"},
				Spec: starlingxv1.PtpInstanceSpec{
					Service:            "ptp4l",
					InstanceParameters: map[string][]string{"global": {"domainNumber=24", "clientOnly=0"}},
				},
			}
			Expect(k8sClient.Create(ctx, created)).To(Succeed())

			fetched := &starlingxv1.PtpInstance{}
			Eventually(func() bool {
				return k8sClient.Get(ctx, key, fetched) == nil
			}, timeout, interval).Should(BeTrue())
			Expect(fetched).To(Equal(created))

			Expect(k8sClient.Delete(ctx, fetched)).To(Succeed())
		})

		It("should accept multiple sections with repetitive UDPv4, UDPv6, L2", func() {
			key := types.NamespacedName{Name: "bar", Namespace: "default"}

			created := &starlingxv1.PtpInstance{
				ObjectMeta: metav1.ObjectMeta{Name: "bar", Namespace: "default"},
				Spec: starlingxv1.PtpInstanceSpec{
					Service: "ptp4l",
					InstanceParameters: map[string][]string{
						"global": {"domainNumber=24", "clientOnly=0"},
						"unicast_master_table_1": {
							"table_id=1",
							"UDPv4=1.2.3.4", "UDPv4=2.3.4.5",
							"L2=00:01:FF:00:01:CD", "L2=00:02:FF:00:01:CD",
							"UDPv6=ffff::1", "UDPv6=ffff::2",
							"peer_address=::1"},
						"unicast_master_table_2": {
							"table_id=2",
							"UDPv4=1.2.3.4",
							"peer_address=::2"},
					},
				},
			}
			Expect(k8sClient.Create(ctx, created)).To(Succeed())

			fetched := &starlingxv1.PtpInstance{}
			Eventually(func() bool {
				return k8sClient.Get(ctx, key, fetched) == nil
			}, timeout, interval).Should(BeTrue())
			Expect(fetched).To(Equal(created))

			Expect(k8sClient.Delete(ctx, fetched)).To(Succeed())
		})
	})

	Context("when creating with invalid parameters", func() {
		It("should reject duplicate non-repeatable keys in unicast_master_table", func() {
			created := &starlingxv1.PtpInstance{
				ObjectMeta: metav1.ObjectMeta{Name: "bar1", Namespace: "default"},
				Spec: starlingxv1.PtpInstanceSpec{
					Service: "ptp4l",
					InstanceParameters: map[string][]string{
						"global": {"domainNumber=24", "clientOnly=0"},
						"unicast_master_table_1": {
							"table_id=1", "table_id=2",
							"UDPv4=1.2.3.4",
							"L2=00:01:FF:00:01:CD",
							"UDPv6=ffff::2",
							"peer_address=::1", "peer_address=::2"},
					},
				},
			}
			err := k8sClient.Create(ctx, created)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("duplicate parameter keys are not allowed for table_id=2."))
		})
	})
})

var _ = Describe("PtpInstance webhook wrappers", func() {
	Context("when calling validators directly", func() {
		It("should accept ValidateUpdate", func() {
			v := &PtpInstanceCustomValidator{}
			obj := &starlingxv1.PtpInstance{Spec: starlingxv1.PtpInstanceSpec{Service: "ptp4l"}}
			_, err := v.ValidateUpdate(ctx, obj, obj)
			Expect(err).ToNot(HaveOccurred())
		})
		It("should accept ValidateDelete", func() {
			v := &PtpInstanceCustomValidator{}
			obj := &starlingxv1.PtpInstance{}
			_, err := v.ValidateDelete(ctx, obj)
			Expect(err).ToNot(HaveOccurred())
		})
	})
})
