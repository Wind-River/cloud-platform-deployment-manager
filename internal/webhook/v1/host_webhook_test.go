/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2024-2025 Wind River Systems, Inc. */
package v1

import (
	"errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	starlingxv1 "github.com/wind-river/cloud-platform-deployment-manager/api/v1"
)

var _ = Describe("system_webhook functions", func() {

	Describe("validateMatchBMInfo function is tested", func() {
		Context("When board management address is not nil", func() {
			It("validates succesfully without error", func() {
				bmAddr := "192.13.24.39"
				r := &starlingxv1.Host{
					Spec: starlingxv1.HostSpec{
						Match: &starlingxv1.MatchInfo{
							BoardManagement: &starlingxv1.MatchBMInfo{
								Address: &bmAddr,
							},
						},
					},
				}
				err := validateMatchBMInfo(r)
				Expect(err).To(BeNil())
			})
		})
		Context("When the board management address is nil", func() {
			It("Throws the error board management address must be supplied in match criteria", func() {
				r := &starlingxv1.Host{
					Spec: starlingxv1.HostSpec{
						Match: &starlingxv1.MatchInfo{
							BoardManagement: &starlingxv1.MatchBMInfo{},
						},
					},
				}
				msg := errors.New("board management address must be supplied in match criteria")
				err := validateMatchBMInfo(r)
				Expect(err).To(Equal(msg))
			})
		})
	})
	Describe("validateMatchDMIInfo function is tested", func() {
		Context("When serial no and assetTag are not nil", func() {
			It("successfull without any error", func() {
				serialNo := "12345"
				astTag := "90"
				r := &starlingxv1.Host{
					Spec: starlingxv1.HostSpec{
						Match: &starlingxv1.MatchInfo{
							DMI: &starlingxv1.MatchDMIInfo{
								SerialNumber: &serialNo,
								AssetTag:     &astTag,
							},
						},
					},
				}
				err := validateMatchDMIInfo(r)
				Expect(err).To(BeNil())
			})
		})
		Context("When both serial no and assetTag are nil", func() {
			It("Throws the error DMI Serial Number or Asset Tag must be supplied in match criteria", func() {
				astTag := "90"
				r := &starlingxv1.Host{
					Spec: starlingxv1.HostSpec{
						Match: &starlingxv1.MatchInfo{
							DMI: &starlingxv1.MatchDMIInfo{
								AssetTag: &astTag,
							},
						},
					},
				}
				msg := errors.New("DMI Serial Number or Asset Tag must be supplied in match criteria")
				err := validateMatchDMIInfo(r)
				Expect(err).To(Equal(msg))
			})
		})
	})
	Describe("validateMatchInfo function is tested", func() {
		Context("When board management,DMI and bootMac info is not nil ", func() {
			It("Successfully validates match info without any error", func() {
				bmAddr := "192.13.24.39"
				serialNo := "12345"
				astTag := "90"
				bootMac := "01:02:03:04:05:06"

				r := &starlingxv1.Host{
					Spec: starlingxv1.HostSpec{
						Match: &starlingxv1.MatchInfo{
							BoardManagement: &starlingxv1.MatchBMInfo{
								Address: &bmAddr,
							},
							DMI: &starlingxv1.MatchDMIInfo{
								SerialNumber: &serialNo,
								AssetTag:     &astTag,
							},
							BootMAC: &bootMac,
						},
					},
				}
				err := validateMatchInfo(r)
				Expect(err).To(BeNil())
			})
		})
		Context("When all board management,DMI and bootMac info are nil", func() {
			It("Returns the error host must be configured with at least 1 match criteria", func() {
				r := &starlingxv1.Host{
					Spec: starlingxv1.HostSpec{
						Match: &starlingxv1.MatchInfo{},
					},
				}
				msg := errors.New("host must be configured with at least 1 match criteria")
				err := validateMatchInfo(r)
				Expect(err).To(Equal(msg))
			})
		})
	})
	Describe("validateHost function is tested", func() {
		Context("When match info is not nil", func() {
			It("It validates the host match info succesfully without any error", func() {
				bmAddr := "192.13.24.39"
				serialNo := "12345"
				astTag := "90"
				bootMac := "01:02:03:04:05:06"

				r := &starlingxv1.Host{
					Spec: starlingxv1.HostSpec{
						Match: &starlingxv1.MatchInfo{
							BoardManagement: &starlingxv1.MatchBMInfo{
								Address: &bmAddr,
							},
							DMI: &starlingxv1.MatchDMIInfo{
								SerialNumber: &serialNo,
								AssetTag:     &astTag,
							},
							BootMAC: &bootMac,
						},
					},
				}
				err := validateHost(r)
				Expect(err).To(BeNil())
			})
		})
	})
})
