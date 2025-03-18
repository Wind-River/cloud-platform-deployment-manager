/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2024-2025 Wind River Systems, Inc. */
package v1

import (
	"errors"

	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/datanetworks"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("datanetwork_webhook functions", func() {

	Describe("validateDataNetwork function is tested", func() {
		Context("When the type of dataNetwork is vxlan", func() {
			It("Sucessfully validates the Data Network", func() {
				r := &DataNetwork{
					Spec: DataNetworkSpec{
						Type: datanetworks.TypeVxLAN,
					},
				}
				err := r.validateDataNetwork()
				Expect(err).To(BeNil())
			})
		})
		Context("When the dataNetwork type is not vxlan but has vxlan info", func() {
			It("Should throw the error VxLAN attributes are only allowed for VxLAN type data networks", func() {
				uDPPortNumber := 8472
				r := &DataNetwork{
					Spec: DataNetworkSpec{
						Type: datanetworks.TypeVLAN,
						VxLAN: &VxLANInfo{
							UDPPortNumber: &uDPPortNumber,
						},
					},
				}
				err := r.validateDataNetwork()
				msg := errors.New("VxLAN attributes are only allowed for VxLAN type data networks.")
				Expect(err).To(Equal(msg))
			})
		})
	})
})
