/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2022 Wind River Systems, Inc. */
package system

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	starlingxv1 "github.com/wind-river/cloud-platform-deployment-manager/api/v1"
)

var _ = Describe("System controller", func() {

	const (
		timeout  = time.Second * 10
		interval = time.Millisecond * 250
	)

	Context("System with data", func() {
		It("Should created successfully", func() {
			ctx := context.Background()
			description := string("A sample description")
			location := string("A sample location")
			contact := string("A sample contact")
			latitude := string("45.35189954974955")
			longitude := string("-75.91866628453701")
			dnsServers := starlingxv1.StringsToDNSServerList([]string{"8.8.8.8", "4.4.4.4"})
			ntpServers := starlingxv1.StringsToNTPServerList([]string{"time.ntp.org", "1.2.3.4"})
			ptpMode := "hardware"
			created := &starlingxv1.System{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: "default",
				},
				Spec: starlingxv1.SystemSpec{
					Description: &description,
					Location:    &location,
					Latitude:    &latitude,
					Longitude:   &longitude,
					Contact:     &contact,
					DNSServers:  &dnsServers,
					NTPServers:  &ntpServers,
					PTP: &starlingxv1.PTPInfo{
						Mode: &ptpMode,
					},
				},
			}
			Expect(k8sClient.Create(ctx, created)).To(Succeed())

			// Mock is needed for the further testing
			// Currently there is no update in System instance
			// So we test only for create
		})
	})
})
