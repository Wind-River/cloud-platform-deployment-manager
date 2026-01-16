/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019-2022, 2024-2025 Wind River Systems, Inc. */
package v1

import (
	"context"
	"encoding/json"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("PtpInstance controller", func() {

	const (
		timeout  = time.Second * 30
		interval = time.Millisecond * 500
	)

	Describe("PtpInstanceSpec", func() {
		Context("UnmarshalJSON", func() {
			It("Should unmarshall if parameters are omit", func() {
				var spec PtpInstanceSpec
				err := json.Unmarshal([]byte(`{"service": "ptp4l"}`), &spec)

				Expect(err).To(BeNil())
				Expect(spec.InstanceParameters).To(BeNil())

				Expect(spec.Service).To(Equal("ptp4l"))
			})

			It("Should unmarshall if parameters are an empty array", func() {
				var spec PtpInstanceSpec
				expected := map[string][]string{}

				err := json.Unmarshal([]byte(`{"service": "ptp4l", "parameters": []}`), &spec)

				Expect(err).To(BeNil())
				Expect(spec.InstanceParameters).To(Equal(expected))

				Expect(spec.Service).To(Equal("ptp4l"))
			})

			It("Should unmarshall array paramaters format", func() {
				jsonSpec := `{"service": "ptp4l", "parameters": ["param1", "param2", "param3"]}`
				expected := map[string][]string{
					"global": {"param1", "param2", "param3"},
				}

				var spec PtpInstanceSpec
				err := json.Unmarshal([]byte(jsonSpec), &spec)

				Expect(err).To(BeNil())
				Expect(spec.InstanceParameters).To(Equal(expected))

				Expect(spec.Service).To(Equal("ptp4l"))
			})

			It("Should unmarshall sectioned paramaters format", func() {
				jsonSpec := `{
				    "service": "ptp4l",
				    "parameters": {"global": ["param1", "param2", "param3"]}
				}`
				expected := map[string][]string{
					"global": {"param1", "param2", "param3"},
				}

				var spec PtpInstanceSpec
				err := json.Unmarshal([]byte(jsonSpec), &spec)

				Expect(err).To(BeNil())
				Expect(spec.InstanceParameters).To(Equal(expected))

				Expect(spec.Service).To(Equal("ptp4l"))
			})

			It("Should unmarshall sectioned paramaters format with multiple paramaters", func() {
				jsonSpec := `{
				    "service": "ptp4l",
				    "parameters": {
				        "global": ["param1", "param2", "param3"],
				        "unicast_master_table_x": ["param1", "param2"]
				    }
				}`
				expected := map[string][]string{
					"global": {"param1", "param2", "param3"},
					"unicast_master_table_x": {"param1", "param2"},
				}

				var spec PtpInstanceSpec
				err := json.Unmarshal([]byte(jsonSpec), &spec)

				Expect(err).To(BeNil())
				Expect(spec.InstanceParameters).To(Equal(expected))

				Expect(spec.Service).To(Equal("ptp4l"))
			})
		})
	})

	Describe("PtpInstance", func() {
		Context("with single section data", func() {
			It("Should created successfully", func() {
				ctx := context.Background()
				key := types.NamespacedName{
					Name:      "foo",
					Namespace: "default",
				}
				created := &PtpInstance{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "foo",
						Namespace: "default",
					},
					Spec: PtpInstanceSpec{
						Service:            "ptp4l",
						InstanceParameters: map[string][]string{"global": []string{"domainNumber=24", "clientOnly=0"}},
					}}
				Expect(k8sClient.Create(ctx, created)).Should(Succeed())

				fetched := &PtpInstance{}

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

	Describe("PtpInstance", func() {
		Context("with multiple section data and repetitive UDPv4, UDPv6, L2", func() {
			It("Should created successfully", func() {
				ctx := context.Background()
				key := types.NamespacedName{
					Name:      "bar",
					Namespace: "default",
				}
				created := &PtpInstance{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "bar",
						Namespace: "default",
					},
					Spec: PtpInstanceSpec{
						Service: "ptp4l",
						InstanceParameters: map[string][]string{
							"global": []string{"domainNumber=24", "clientOnly=0"},
							"unicast_master_table_1": []string{
								"table_id=1",
								"UDPv4=1.2.3.4", "UDPv4=2.3.4.5",
								"L2=00:01:FF:00:01:CD", "L2=00:02:FF:00:01:CD",
								"UDPv6=ffff::1", "UDPv6=ffff::2",
								"peer_address=::1"},
							"unicast_master_table_2": []string{
								"table_id=2",
								"UDPv4=1.2.3.4",
								"peer_address=::2"},
						},
					}}
				Expect(k8sClient.Create(ctx, created)).Should(Succeed())

				fetched := &PtpInstance{}

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

	Describe("PtpInstance", func() {
		Context("with unicast_master_table prefixed section and repetitive other than L2/UDPv4/UDPv6 keys", func() {
			It("Should fail", func() {
				ctx := context.Background()
				created := &PtpInstance{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "bar1",
						Namespace: "default",
					},
					Spec: PtpInstanceSpec{
						Service: "ptp4l",
						InstanceParameters: map[string][]string{
							"global": []string{"domainNumber=24", "clientOnly=0"},
							"unicast_master_table_1": []string{
								"table_id=1", "table_id=2",
								"UDPv4=1.2.3.4",
								"L2=00:01:FF:00:01:CD",
								"UDPv6=ffff::2",
								"peer_address=::1", "peer_address=::2"},
						},
					}}
				err := (k8sClient.Create(ctx, created))
				Expect(err).ToNot(BeNil())
				Expect(err.Error()).To(ContainSubstring("duplicate parameter keys are not allowed for table_id=2."))
			})
		})
	})
})
