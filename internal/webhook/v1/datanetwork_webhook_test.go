/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2024-2026 Wind River Systems, Inc. */
package v1

import (
	"errors"

	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/datanetworks"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	starlingxv1 "github.com/wind-river/cloud-platform-deployment-manager/api/v1"
)

var _ = Describe("DataNetworkWebhook", func() {

	Describe("ValidateDataNetwork", func() {
		Context("When the type of dataNetwork is vxlan", func() {
			It("should successfully validate the Data Network", func() {
				r := &starlingxv1.DataNetwork{
					Spec: starlingxv1.DataNetworkSpec{
						Type: datanetworks.TypeVxLAN,
					},
				}
				err := validateDataNetwork(r)
				Expect(err).ToNot(HaveOccurred())
			})
		})
		Context("When the dataNetwork type is not vxlan but has vxlan info", func() {
			It("Should throw the error VxLAN attributes are only allowed for VxLAN type data networks", func() {
				uDPPortNumber := 8472
				r := &starlingxv1.DataNetwork{
					Spec: starlingxv1.DataNetworkSpec{
						Type: datanetworks.TypeVLAN,
						VxLAN: &starlingxv1.VxLANInfo{
							UDPPortNumber: &uDPPortNumber,
						},
					},
				}
				err := validateDataNetwork(r)
				msg := errors.New("VxLAN attributes are only allowed for VxLAN type data networks")
				Expect(err).To(Equal(msg))
			})
		})
	})
})
