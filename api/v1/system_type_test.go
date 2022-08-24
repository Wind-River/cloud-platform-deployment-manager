/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019-2022 Wind River Systems, Inc. */
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
		timeout  = time.Second * 10
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
})
