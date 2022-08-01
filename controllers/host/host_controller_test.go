/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019-2022 Wind River Systems, Inc. */
package host

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	starlingxv1 "github.com/wind-river/cloud-platform-deployment-manager/api/v1"
	comm "github.com/wind-river/cloud-platform-deployment-manager/common"
)

var _ = Describe("Host controller", func() {

	const (
		timeout  = time.Second * 10
		interval = time.Millisecond * 250
	)

	Context("Host with data", func() {
		It("Should created successfully", func() {
			ctx := context.Background()
			key := types.NamespacedName{
				Name:      "foo",
				Namespace: "default",
			}
			bootMac := "01:02:03:04:05:06"
			bmAddress := "192.168.9.9"
			match := starlingxv1.MatchInfo{
				BootMAC: &bootMac,
			}
			bmType := "bmc"
			created := &starlingxv1.Host{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: "default",
				},
				Spec: starlingxv1.HostSpec{
					Profile: "some-profile",
					Match:   &match,
					Overrides: &starlingxv1.HostProfileSpec{
						Addresses: []starlingxv1.AddressInfo{
							{Interface: "enp0s3", Address: "1.2.3.10", Prefix: 24},
						},
						BoardManagement: &starlingxv1.BMInfo{
							Type:    &bmType,
							Address: &bmAddress,
						},
					},
				}}
			Expect(k8sClient.Create(ctx, created)).To(Succeed())

			expected := created.DeepCopy()

			fetched := &starlingxv1.Host{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, key, fetched)
				return err == nil &&
					fetched.ObjectMeta.ResourceVersion != expected.ObjectMeta.ResourceVersion
			}, timeout, interval).Should(BeTrue())
			_, found := comm.ListIntersect(fetched.ObjectMeta.Finalizers, []string{HostFinalizerName})
			Expect(found).To(BeTrue())
		})
	})
})
