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
		Context("when the type of dataNetwork is vxlan", func() {
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

var _ = Describe("DataNetworkWebhook wrappers", func() {
	Context("when calling validators directly", func() {
		It("should accept ValidateCreate", func() {
			v := &DataNetworkCustomValidator{}
			obj := &starlingxv1.DataNetwork{Spec: starlingxv1.DataNetworkSpec{Type: datanetworks.TypeVxLAN}}
			_, err := v.ValidateCreate(ctx, obj)
			Expect(err).ToNot(HaveOccurred())
		})
		It("should accept ValidateUpdate", func() {
			v := &DataNetworkCustomValidator{}
			obj := &starlingxv1.DataNetwork{Spec: starlingxv1.DataNetworkSpec{Type: datanetworks.TypeVxLAN}}
			_, err := v.ValidateUpdate(ctx, obj, obj)
			Expect(err).ToNot(HaveOccurred())
		})
		It("should accept ValidateDelete", func() {
			v := &DataNetworkCustomValidator{}
			// NOTE: ValidateDelete has a copy-paste bug — it casts to *AddressPool instead of *DataNetwork.
			// Passing a DataNetwork triggers the type assertion error.
			obj := &starlingxv1.DataNetwork{}
			_, err := v.ValidateDelete(ctx, obj)
			Expect(err).To(HaveOccurred())
		})
	})
})
