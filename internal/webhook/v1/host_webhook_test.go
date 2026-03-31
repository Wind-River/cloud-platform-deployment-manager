/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2024-2026 Wind River Systems, Inc. */
package v1

import (
	"errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	starlingxv1 "github.com/wind-river/cloud-platform-deployment-manager/api/v1"
)

var _ = Describe("HostWebhook", func() {

	Describe("ValidateMatchBMInfo", func() {
		Context("When board management address is not nil", func() {
			It("should validate successfully without error", func() {
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
				Expect(err).ToNot(HaveOccurred())
			})
		})
		Context("When the board management address is nil", func() {
			It("should return error that board management address must be supplied in match criteria", func() {
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
	Describe("ValidateMatchDMIInfo", func() {
		Context("When serial no and assetTag are not nil", func() {
			It("should succeed without any error", func() {
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
				Expect(err).ToNot(HaveOccurred())
			})
		})
		Context("When both serial no and assetTag are nil", func() {
			It("should return error that DMI Serial Number or Asset Tag must be supplied in match criteria", func() {
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
	Describe("ValidateMatchInfo", func() {
		Context("When board management,DMI and bootMac info is not nil ", func() {
			It("should validate match info successfully without any error", func() {
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
				Expect(err).ToNot(HaveOccurred())
			})
		})
		Context("when only bootMAC is provided", func() {
			It("should validate successfully without error", func() {
				bootMac := "01:02:03:04:05:06"
				r := &starlingxv1.Host{
					Spec: starlingxv1.HostSpec{
						Match: &starlingxv1.MatchInfo{
							BootMAC: &bootMac,
						},
					},
				}
				err := validateMatchInfo(r)
				Expect(err).ToNot(HaveOccurred())
			})
		})
		Context("When all board management,DMI and bootMac info are nil", func() {
			It("should return error that host must be configured with at least 1 match criteria", func() {
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
		Context("when board management address is nil", func() {
			It("should propagate the error", func() {
				r := &starlingxv1.Host{
					Spec: starlingxv1.HostSpec{
						Match: &starlingxv1.MatchInfo{
							BoardManagement: &starlingxv1.MatchBMInfo{},
						},
					},
				}
				err := validateMatchInfo(r)
				Expect(err).To(HaveOccurred())
			})
		})
		Context("when DMI fields are incomplete", func() {
			It("should propagate the error", func() {
				r := &starlingxv1.Host{
					Spec: starlingxv1.HostSpec{
						Match: &starlingxv1.MatchInfo{
							DMI: &starlingxv1.MatchDMIInfo{},
						},
					},
				}
				err := validateMatchInfo(r)
				Expect(err).To(HaveOccurred())
			})
		})
	})
	Describe("ValidateHost", func() {
		Context("when match info is not nil", func() {
			It("should validate the host match info successfully without any error", func() {
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
				Expect(err).ToNot(HaveOccurred())
			})
		})
		Context("when match info is nil", func() {
			It("should succeed without error", func() {
				r := &starlingxv1.Host{
					Spec: starlingxv1.HostSpec{},
				}
				err := validateHost(r)
				Expect(err).ToNot(HaveOccurred())
			})
		})
		Context("when match info has invalid data", func() {
			It("should propagate the error", func() {
				r := &starlingxv1.Host{
					Spec: starlingxv1.HostSpec{
						Match: &starlingxv1.MatchInfo{},
					},
				}
				err := validateHost(r)
				Expect(err).To(HaveOccurred())
			})
		})
	})
})

var _ = Describe("HostWebhook wrappers", func() {
	Context("when calling validators directly", func() {
		It("should accept ValidateCreate", func() {
			v := &HostCustomValidator{}
			obj := &starlingxv1.Host{Spec: starlingxv1.HostSpec{}}
			_, err := v.ValidateCreate(ctx, obj)
			Expect(err).ToNot(HaveOccurred())
		})
		It("should accept ValidateUpdate", func() {
			v := &HostCustomValidator{}
			obj := &starlingxv1.Host{Spec: starlingxv1.HostSpec{}}
			_, err := v.ValidateUpdate(ctx, obj, obj)
			Expect(err).ToNot(HaveOccurred())
		})
		It("should accept ValidateDelete", func() {
			v := &HostCustomValidator{}
			obj := &starlingxv1.Host{}
			_, err := v.ValidateDelete(ctx, obj)
			Expect(err).ToNot(HaveOccurred())
		})
	})
})
