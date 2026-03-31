/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2022, 2024-2026 Wind River Systems, Inc. */
package system

import (
	"context"

	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/dns"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/drbd"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/hosts"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/ntp"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/ptp"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/serviceparameters"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/system"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	starlingxv1 "github.com/wind-river/cloud-platform-deployment-manager/api/v1"
)

var _ = Describe("System controller", func() {
	Describe("DnsUpdateRequired", func() {
		Context("when dns servers are configured", func() {
			It("should return true if DNS arrays have different sizes", func() {
				spec := starlingxv1.SystemSpec{
					DNSServers: []string{},
				}
				info := dns.DNS{Nameservers: "1.1.1.1,8.8.8.8,8.8.4.4"}
				opts, required := dnsUpdateRequired(&spec, &info)
				Expect(required).To(BeTrue())
				Expect(*opts.Nameservers).To(Equal("NC"))
			})
		})

		It("should return false when current and desired DNS servers are identical", func() {
			spec := starlingxv1.SystemSpec{
				DNSServers: []string{"1.1.1.1", "8.8.8.8", "8.8.4.4"},
			}
			info := dns.DNS{Nameservers: "1.1.1.1,8.8.8.8,8.8.4.4"}
			opts, required := dnsUpdateRequired(&spec, &info)
			Expect(opts.Nameservers).To(BeNil())
			Expect(required).To(BeFalse())
		})

		It("should return true when current DNS is empty and desired has DNS servers", func() {
			spec := starlingxv1.SystemSpec{
				DNSServers: []string{"1.1.1.1", "8.8.8.8", "8.8.4.4"},
			}
			info := dns.DNS{Nameservers: ""}
			opts, required := dnsUpdateRequired(&spec, &info)
			Expect(required).To(BeTrue())
			Expect(*opts.Nameservers).To(Equal("1.1.1.1,8.8.8.8,8.8.4.4"))
		})

		It("should return true when current has DNS servers but desired has none", func() {
			spec := starlingxv1.SystemSpec{DNSServers: []string{}}
			info := dns.DNS{Nameservers: "1.1.1.1,8.8.8.8,8.8.4.4"}
			opts, required := dnsUpdateRequired(&spec, &info)
			Expect(required).To(BeTrue())
			Expect(*opts.Nameservers).To(Equal("NC"))
		})
	})

	Describe("NtpUpdateRequired", func() {
		Context("when NTP servers are configured", func() {
			It("should return true if NTP arrays have different sizes", func() {
				spec := starlingxv1.SystemSpec{NTPServers: []string{}}
				info := ntp.NTP{NTPServers: "0.ubuntu.pool.ntp.org,1.ubuntu.pool.ntp.org"}
				opts, required := ntpUpdateRequired(&spec, &info)
				Expect(required).To(BeTrue())
				Expect(*opts.NTPServers).To(Equal("NC"))
			})

			It("should return false when current and desired NTP servers are identical", func() {
				spec := starlingxv1.SystemSpec{
					NTPServers: []string{"0.ubuntu.pool.ntp.org", "1.ubuntu.pool.ntp.org"},
				}
				info := ntp.NTP{NTPServers: "0.ubuntu.pool.ntp.org,1.ubuntu.pool.ntp.org"}
				opts, required := ntpUpdateRequired(&spec, &info)
				Expect(opts.NTPServers).To(BeNil())
				Expect(required).To(BeFalse())
			})

			It("should return true when current NTP is empty and desired has NTP servers", func() {
				spec := starlingxv1.SystemSpec{
					NTPServers: []string{"0.ubuntu.pool.ntp.org", "1.ubuntu.pool.ntp.org"},
				}
				info := ntp.NTP{NTPServers: ""}
				opts, required := ntpUpdateRequired(&spec, &info)
				Expect(required).To(BeTrue())
				Expect(*opts.NTPServers).To(Equal("0.ubuntu.pool.ntp.org,1.ubuntu.pool.ntp.org"))
			})

			It("should return true when current has NTP servers but desired has none", func() {
				spec := starlingxv1.SystemSpec{NTPServers: []string{}}
				info := ntp.NTP{NTPServers: "0.ubuntu.pool.ntp.org,1.ubuntu.pool.ntp.org"}
				opts, required := ntpUpdateRequired(&spec, &info)
				Expect(required).To(BeTrue())
				Expect(*opts.NTPServers).To(Equal("NC"))
			})
		})

	})

	Context("with System data", func() {
		It("should be created successfully", func() {
			ctx := context.Background()
			description := string("A sample description")
			location := string("A sample location")
			contact := string("A sample contact")
			latitude := string("45.35189954974955")
			longitude := string("-75.91866628453701")
			dnsServers := []string{"8.8.8.8", "4.4.4.4"}
			ntpServers := []string{"time.ntp.org", "1.2.3.4"}
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
					DNSServers:  dnsServers,
					NTPServers:  ntpServers,
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

	Context("when cleaning deprecated certificates", func() {
		It("should filter all deprecated certificate types successfully", func() {
			certs := []starlingxv1.CertificateInfo{
				{
					Type:      starlingxv1.PlatformCertificate,
					Signature: "ssl_10886226602156394257",
				},
				{
					Type:      starlingxv1.PlatformCACertificate,
					Signature: "ssl_ca_10886226602156394257",
				},
				{
					Type:      starlingxv1.DockerCertificate,
					Signature: "docker_registry_10886226602156394257",
				},
			}
			expOutCerts := []starlingxv1.CertificateInfo{
				{
					Type:      starlingxv1.PlatformCACertificate,
					Signature: "ssl_ca_10886226602156394257",
				},
			}
			outCerts := cleanDeprecatedCerts(certs)
			Expect(outCerts).To(Equal(expOutCerts))
		})
	})

	Context("when validating a valid deployment for Ceph Rook backend", func() {
		It("should return successfully", func() {
			nameBackend := string("ceph-rook-foobar")
			typeBackend := string("ceph-rook")
			deploymentModel := string("controller")
			created := &starlingxv1.System{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foobar",
					Namespace: "default",
				},
				Spec: starlingxv1.SystemSpec{
					Storage: &starlingxv1.SystemStorageInfo{
						Backends: starlingxv1.StorageBackendList{
							{
								Name:       nameBackend,
								Type:       typeBackend,
								Deployment: deploymentModel,
							},
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, created)).To(Succeed())
		})
	})

	Context("when validating an invalid deployment for Ceph Rook backend", func() {
		It("should return error", func() {
			nameBackend := string("ceph-rook-foobarfoo")
			typeBackend := string("ceph-rook")
			deploymentModel := string("incorrect-deployment-model")
			created := &starlingxv1.System{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foobarfoo",
					Namespace: "default",
				},
				Spec: starlingxv1.SystemSpec{
					Storage: &starlingxv1.SystemStorageInfo{
						Backends: starlingxv1.StorageBackendList{
							{
								Name:       nameBackend,
								Type:       typeBackend,
								Deployment: deploymentModel,
							},
						},
					},
				},
			}
			err := k8sClient.Create(ctx, created)
			Expect(err).To(HaveOccurred())
		})
	})

	Context("when creating ceph-float Controller Filesystem", func() {
		It("should return successfully", func() {
			nameBackend := string("ceph-rook-storage")
			typeBackend := string("ceph-rook")
			deploymentModel := string("controller")
			nameControllerFs := string("ceph-float")
			sizeControllerFs := 10
			created := &starlingxv1.System{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "rook",
					Namespace: "default",
				},
				Spec: starlingxv1.SystemSpec{
					Storage: &starlingxv1.SystemStorageInfo{
						Backends: starlingxv1.StorageBackendList{
							{
								Name:       nameBackend,
								Type:       typeBackend,
								Deployment: deploymentModel,
							},
						},
						FileSystems: starlingxv1.ControllerFileSystemList{
							{
								Name: nameControllerFs,
								Size: sizeControllerFs,
							},
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, created)).To(Succeed())
		})
	})
	Context("when current has runtimeCerts", func() {
		It("should remove the runtime certs and return the remaining", func() {
			specCerts := []starlingxv1.CertificateInfo{
				{
					Type:      "ssl_ca",
					Secret:    "ssl-ca-secret-1",
					Signature: "",
				},
				{
					Type:      "ssl_ca",
					Secret:    "ssl-ca-secret-2",
					Signature: "",
				},
				{
					Type:      "ssl_ca",
					Secret:    "ssl-ca-secret-3",
					Signature: "",
				},
			}
			currentCerts := []starlingxv1.CertificateInfo{
				{
					Type:      "ssl_ca",
					Secret:    "ssl-ca-secret-1",
					Signature: "ssl_ca_0011",
				},
				{
					Type:      "ssl_ca",
					Secret:    "ssl-ca-secret-2",
					Signature: "ssl_ca_0012",
				},
				{
					Type:      "ssl_ca",
					Secret:    "ssl-ca-secret-0",
					Signature: "ssl_ca_0013",
				},
			}

			expRes := []starlingxv1.CertificateInfo{
				{
					Type:      "ssl_ca",
					Secret:    "ssl-ca-secret-1",
					Signature: "ssl_ca_0011",
				},
				{
					Type:      "ssl_ca",
					Secret:    "ssl-ca-secret-2",
					Signature: "ssl_ca_0012",
				},
			}
			res := FixCertsToManage(specCerts, currentCerts)
			Expect(res).To(Equal(expRes))
		})
	})

	Describe("systemUpdateRequired", func() {
		var (
			instance *starlingxv1.System
			spec     *starlingxv1.SystemSpec
			current  *system.System
		)

		BeforeEach(func() {
			instance = &starlingxv1.System{
				ObjectMeta: metav1.ObjectMeta{Name: "test-system"},
			}
			spec = &starlingxv1.SystemSpec{}
			current = &system.System{Name: "test-system"}
		})

		Context("when no fields differ", func() {
			It("should return false", func() {
				_, result := systemUpdateRequired(instance, spec, current)
				Expect(result).To(BeFalse())
			})
		})

		Context("when the instance name differs", func() {
			It("should return true with Name set", func() {
				instance.Name = "new-name"
				opts, result := systemUpdateRequired(instance, spec, current)
				Expect(result).To(BeTrue())
				Expect(*opts.Name).To(Equal("new-name"))
			})
		})

		Context("when spec fields differ", func() {
			It("should detect description change", func() {
				desc := "new-desc"
				spec.Description = &desc
				current.Description = "old-desc"
				opts, result := systemUpdateRequired(instance, spec, current)
				Expect(result).To(BeTrue())
				Expect(*opts.Description).To(Equal("new-desc"))
			})

			It("should detect contact change", func() {
				contact := "new-contact"
				spec.Contact = &contact
				opts, result := systemUpdateRequired(instance, spec, current)
				Expect(result).To(BeTrue())
				Expect(*opts.Contact).To(Equal("new-contact"))
			})

			It("should detect location change", func() {
				loc := "new-loc"
				spec.Location = &loc
				opts, result := systemUpdateRequired(instance, spec, current)
				Expect(result).To(BeTrue())
				Expect(*opts.Location).To(Equal("new-loc"))
			})

			It("should detect latitude change", func() {
				lat := "45.0"
				spec.Latitude = &lat
				opts, result := systemUpdateRequired(instance, spec, current)
				Expect(result).To(BeTrue())
				Expect(*opts.Latitude).To(Equal("45.0"))
			})

			It("should detect longitude change", func() {
				lon := "-75.0"
				spec.Longitude = &lon
				opts, result := systemUpdateRequired(instance, spec, current)
				Expect(result).To(BeTrue())
				Expect(*opts.Longitude).To(Equal("-75.0"))
			})

			It("should detect vswitch type change", func() {
				vswitch := "ovs-dpdk"
				spec.VSwitchType = &vswitch
				current.Capabilities.VSwitchType = "none"
				opts, result := systemUpdateRequired(instance, spec, current)
				Expect(result).To(BeTrue())
				Expect(*opts.VSwitchType).To(Equal("ovs-dpdk"))
			})
		})

		Context("when spec field matches current", func() {
			It("should return false", func() {
				desc := "same"
				spec.Description = &desc
				current.Description = "same"
				_, result := systemUpdateRequired(instance, spec, current)
				Expect(result).To(BeFalse())
			})
		})
	})

	Describe("ptpUpdateRequired", func() {
		Context("when spec is nil", func() {
			It("should return false", func() {
				_, result := ptpUpdateRequired(nil, &ptp.PTP{})
				Expect(result).To(BeFalse())
			})
		})

		Context("when mode differs", func() {
			It("should return true with Mode set", func() {
				mode := "hardware"
				spec := &starlingxv1.PTPInfo{Mode: &mode}
				opts, result := ptpUpdateRequired(spec, &ptp.PTP{Mode: "software"})
				Expect(result).To(BeTrue())
				Expect(*opts.Mode).To(Equal("hardware"))
			})
		})

		Context("when mechanism differs", func() {
			It("should return true with Mechanism set", func() {
				mech := "p2p"
				spec := &starlingxv1.PTPInfo{Mechanism: &mech}
				opts, result := ptpUpdateRequired(spec, &ptp.PTP{Mechanism: "e2e"})
				Expect(result).To(BeTrue())
				Expect(*opts.Mechanism).To(Equal("p2p"))
			})
		})

		Context("when transport differs", func() {
			It("should return true with Transport set", func() {
				transport := "l2"
				spec := &starlingxv1.PTPInfo{Transport: &transport}
				opts, result := ptpUpdateRequired(spec, &ptp.PTP{Transport: "udp"})
				Expect(result).To(BeTrue())
				Expect(*opts.Transport).To(Equal("l2"))
			})
		})

		Context("when all fields match", func() {
			It("should return false", func() {
				mode := "hardware"
				spec := &starlingxv1.PTPInfo{Mode: &mode}
				_, result := ptpUpdateRequired(spec, &ptp.PTP{Mode: "hardware"})
				Expect(result).To(BeFalse())
			})
		})
	})

	Describe("drbdUpdateRequired", func() {
		Context("when storage is nil", func() {
			It("should return false", func() {
				spec := &starlingxv1.SystemSpec{}
				_, result := drbdUpdateRequired(spec, &drbd.DRBD{})
				Expect(result).To(BeFalse())
			})
		})

		Context("when DRBD is nil", func() {
			It("should return false", func() {
				spec := &starlingxv1.SystemSpec{
					Storage: &starlingxv1.SystemStorageInfo{},
				}
				_, result := drbdUpdateRequired(spec, &drbd.DRBD{})
				Expect(result).To(BeFalse())
			})
		})

		Context("when link utilization differs", func() {
			It("should return true", func() {
				spec := &starlingxv1.SystemSpec{
					Storage: &starlingxv1.SystemStorageInfo{
						DRBD: &starlingxv1.DRBDConfiguration{LinkUtilization: 80},
					},
				}
				opts, result := drbdUpdateRequired(spec, &drbd.DRBD{LinkUtilization: 60})
				Expect(result).To(BeTrue())
				Expect(opts.LinkUtilization).To(Equal(80))
			})
		})

		Context("when link utilization matches", func() {
			It("should return false", func() {
				spec := &starlingxv1.SystemSpec{
					Storage: &starlingxv1.SystemStorageInfo{
						DRBD: &starlingxv1.DRBDConfiguration{LinkUtilization: 60},
					},
				}
				_, result := drbdUpdateRequired(spec, &drbd.DRBD{LinkUtilization: 60})
				Expect(result).To(BeFalse())
			})
		})
	})

	Describe("serviceparametersUpdateRequired", func() {
		Context("when spec is nil", func() {
			It("should return false", func() {
				_, result := serviceparametersUpdateRequired(nil, &serviceparameters.ServiceParameter{})
				Expect(result).To(BeFalse())
			})
		})

		Context("when param value differs", func() {
			It("should return true", func() {
				spec := &starlingxv1.ServiceParameterInfo{ParamValue: "new-val"}
				current := &serviceparameters.ServiceParameter{ParamValue: "old-val"}
				opts, result := serviceparametersUpdateRequired(spec, current)
				Expect(result).To(BeTrue())
				Expect(*opts.ParamValue).To(Equal("new-val"))
			})
		})

		Context("when resource differs", func() {
			It("should return true", func() {
				res := "new-resource"
				oldRes := "old-resource"
				spec := &starlingxv1.ServiceParameterInfo{
					ParamValue: "same",
					Resource:   &res,
				}
				current := &serviceparameters.ServiceParameter{
					ParamValue: "same",
					Resource:   &oldRes,
				}
				opts, result := serviceparametersUpdateRequired(spec, current)
				Expect(result).To(BeTrue())
				Expect(*opts.Resource).To(Equal("new-resource"))
			})
		})

		Context("when personality differs", func() {
			It("should return true", func() {
				pers := "controller"
				oldPers := "storage"
				spec := &starlingxv1.ServiceParameterInfo{
					ParamValue:  "same",
					Personality: &pers,
				}
				current := &serviceparameters.ServiceParameter{
					ParamValue:  "same",
					Personality: &oldPers,
				}
				opts, result := serviceparametersUpdateRequired(spec, current)
				Expect(result).To(BeTrue())
				Expect(*opts.Personality).To(Equal("controller"))
			})
		})

		Context("when all fields match", func() {
			It("should return false", func() {
				res := "same-res"
				spec := &starlingxv1.ServiceParameterInfo{
					ParamValue: "same",
					Resource:   &res,
				}
				current := &serviceparameters.ServiceParameter{
					ParamValue: "same",
					Resource:   &res,
				}
				_, result := serviceparametersUpdateRequired(spec, current)
				Expect(result).To(BeFalse())
			})
		})
	})

	Describe("ControllerNodesAvailable", func() {
		Context("when enough controllers are unlocked/enabled/available", func() {
			It("should return true", func() {
				hostList := []hosts.Host{
					{
						Hostname:            "controller-0",
						Personality:         hosts.PersonalityController,
						AdministrativeState: hosts.AdminUnlocked,
						OperationalStatus:   hosts.OperEnabled,
						AvailabilityStatus:  hosts.AvailAvailable,
					},
					{
						Hostname:            "controller-1",
						Personality:         hosts.PersonalityController,
						AdministrativeState: hosts.AdminUnlocked,
						OperationalStatus:   hosts.OperEnabled,
						AvailabilityStatus:  hosts.AvailAvailable,
					},
				}
				Expect(ControllerNodesAvailable(hostList, 2)).To(BeTrue())
			})
		})

		Context("when not enough controllers are available", func() {
			It("should return false", func() {
				hostList := []hosts.Host{
					{
						Hostname:            "controller-0",
						Personality:         hosts.PersonalityController,
						AdministrativeState: hosts.AdminUnlocked,
						OperationalStatus:   hosts.OperEnabled,
						AvailabilityStatus:  hosts.AvailAvailable,
					},
					{
						Hostname:            "controller-1",
						Personality:         hosts.PersonalityController,
						AdministrativeState: hosts.AdminUnlocked,
						OperationalStatus:   hosts.OperEnabled,
						AvailabilityStatus:  "degraded",
					},
				}
				Expect(ControllerNodesAvailable(hostList, 2)).To(BeFalse())
			})
		})

		Context("when worker nodes are present but not controllers", func() {
			It("should return false", func() {
				hostList := []hosts.Host{
					{
						Hostname:            "compute-0",
						Personality:         "worker",
						AdministrativeState: hosts.AdminUnlocked,
						OperationalStatus:   hosts.OperEnabled,
						AvailabilityStatus:  hosts.AvailAvailable,
					},
				}
				Expect(ControllerNodesAvailable(hostList, 1)).To(BeFalse())
			})
		})

		Context("when the host list is empty", func() {
			It("should return false for required > 0", func() {
				Expect(ControllerNodesAvailable([]hosts.Host{}, 1)).To(BeFalse())
			})

			It("should return true for required = 0", func() {
				Expect(ControllerNodesAvailable([]hosts.Host{}, 0)).To(BeTrue())
			})
		})
	})

	Describe("MergeSystemSpecs", func() {
		Context("when merging two specs", func() {
			It("should override fields from b into a", func() {
				desc := "from-a"
				descB := "from-b"
				a := &starlingxv1.SystemSpec{Description: &desc}
				b := &starlingxv1.SystemSpec{Description: &descB}
				result, err := MergeSystemSpecs(a, b)
				Expect(err).ToNot(HaveOccurred())
				Expect(*result.Description).To(Equal("from-b"))
			})

			It("should keep fields from a when b is empty", func() {
				desc := "from-a"
				a := &starlingxv1.SystemSpec{Description: &desc}
				b := &starlingxv1.SystemSpec{}
				result, err := MergeSystemSpecs(a, b)
				Expect(err).ToNot(HaveOccurred())
				Expect(*result.Description).To(Equal("from-a"))
			})
		})
	})

	Describe("FillOptionalMergedSystemSpec", func() {
		Context("when a ceph backend has no network", func() {
			It("should fill the default mgmt network", func() {
				spec := &starlingxv1.SystemSpec{
					Storage: &starlingxv1.SystemStorageInfo{
						Backends: starlingxv1.StorageBackendList{
							{Name: "ceph-store", Type: "ceph"},
						},
					},
				}
				result, err := FillOptionalMergedSystemSpec(spec)
				Expect(err).ToNot(HaveOccurred())
				Expect(*result.Storage.Backends[0].Network).To(Equal("mgmt"))
			})
		})

		Context("when a non-ceph backend has no network", func() {
			It("should not fill the network", func() {
				spec := &starlingxv1.SystemSpec{
					Storage: &starlingxv1.SystemStorageInfo{
						Backends: starlingxv1.StorageBackendList{
							{Name: "lvm-store", Type: "lvm"},
						},
					},
				}
				result, err := FillOptionalMergedSystemSpec(spec)
				Expect(err).ToNot(HaveOccurred())
				Expect(result.Storage.Backends[0].Network).To(BeNil())
			})
		})

		Context("when storage is nil", func() {
			It("should return the spec unchanged", func() {
				spec := &starlingxv1.SystemSpec{}
				result, err := FillOptionalMergedSystemSpec(spec)
				Expect(err).ToNot(HaveOccurred())
				Expect(result.Storage).To(BeNil())
			})
		})
	})
})
