/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019-2024 Wind River Systems, Inc. */
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

	Context("System with data", func() {
		It("Should created successfully", func() {
			ctx := context.Background()
			key := types.NamespacedName{
				Name:      "foo",
				Namespace: "default",
			}
			description := string("A sample description")
			location := string("A sample location")
			contact := string("A sample contact")
			latitude := string("45.35189954974955")
			longitude := string("-75.91866628453701")
			dnsServers := StringsToDNSServerList([]string{"8.8.8.8", "4.4.4.4"})
			ntpServers := StringsToNTPServerList([]string{"time.ntp.org", "1.2.3.4"})
			ptpMode := "hardware"
			created := &System{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: "default",
				},
				Spec: SystemSpec{
					Description: &description,
					Location:    &location,
					Latitude:    &latitude,
					Longitude:   &longitude,
					Contact:     &contact,
					DNSServers:  &dnsServers,
					NTPServers:  &ntpServers,
					PTP: &PTPInfo{
						Mode: &ptpMode,
					},
				},
			}
			Expect(k8sClient.Create(ctx, created)).To(Succeed())

			fetched := &System{}

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
	Describe("Test DeepEqual function", func() {
		Context("When both type and secret are same but signature is different in 2 entities", func() {
			It("Returns true", func() {
				other := &CertificateInfo{
					Type:   "ssl",
					Secret: "secret",
				}
				in := &CertificateInfo{
					Signature: "",
					Type:      "ssl",
					Secret:    "secret",
				}
				out := in.DeepEqual(other)
				Expect(out).To(BeTrue())
			})
		})
		Context("When type, secret  and signature are same in 2 entities", func() {
			It("Returns true", func() {
				other := &CertificateInfo{
					Type:      "ssl",
					Secret:    "secret",
					Signature: "signature",
				}
				in := &CertificateInfo{
					Signature: "signature",
					Type:      "ssl",
					Secret:    "secret",
				}
				out := in.DeepEqual(other)
				Expect(out).To(BeTrue())
			})
		})
	})
	Describe("Test IsKeyEqual function", func() {
		Context("When both type and secret are same but signature is different in 2 entities", func() {
			It("Returns true", func() {
				x := CertificateInfo{
					Type:   "ssl",
					Secret: "secret",
				}
				in := CertificateInfo{
					Signature: "",
					Type:      "ssl",
					Secret:    "secret",
				}
				out := in.IsKeyEqual(x)
				Expect(out).To(BeTrue())
			})
		})
		Context("When type, secret  and signature are same in 2 entities", func() {
			It("Returns true", func() {
				x := CertificateInfo{
					Signature: "signature",
					Type:      "ssl",
					Secret:    "secret",
				}
				in := CertificateInfo{
					Signature: "signature",
					Type:      "ssl",
					Secret:    "secret",
				}
				out := in.IsKeyEqual(x)
				Expect(out).To(BeTrue())
			})
		})
	})
	Describe("Test DNSServerListToStrings function", func() {
		Context("When list of 2 DNSservers are given", func() {
			It("String array of the given DNSservers is returned", func() {
				item1 := DNSServer("DNSServer")
				item2 := DNSServer("DNSServer1")
				list := make([]DNSServer, 0)
				list = append(list, item1, item2)
				items := DNSServerList(list)
				exp := []string{"DNSServer", "DNSServer1"}
				out := DNSServerListToStrings(items)
				Expect(out).To(Equal(exp))
			})
		})
	})
	Describe("Test NTPServerListToStrings function", func() {
		Context("When list of 2 NTPservers are given", func() {
			It("String array of the given NTPservers is returned", func() {
				item1 := NTPServer("NTPServer")
				item2 := NTPServer("NTPServer1")
				list := make([]NTPServer, 0)
				list = append(list, item1, item2)
				items := NTPServerList(list)
				exp := []string{"NTPServer", "NTPServer1"}
				out := NTPServerListToStrings(items)
				Expect(out).To(Equal(exp))
			})
		})
	})
})
