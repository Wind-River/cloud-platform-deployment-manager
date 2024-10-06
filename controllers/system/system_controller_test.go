/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2022,2024 Wind River Systems, Inc. */
package system

import (
	"context"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	starlingxv1 "github.com/wind-river/cloud-platform-deployment-manager/api/v1"
)

var _ = Describe("System controller", func() {

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

	Context("Test clean_deprecated_certificates func", func() {
		It("Should filter all deprecated certficate types successfully", func() {
			certs := starlingxv1.CertificateList{
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
			expOutCerts := starlingxv1.CertificateList{
				{
					Type:      starlingxv1.PlatformCACertificate,
					Signature: "ssl_ca_10886226602156394257",
				},
			}
			outCerts := clean_deprecated_certificates(certs)
			Expect(outCerts).To(Equal(expOutCerts))
		})
	})

	Context("Test valid deployment for Ceph Rook backend", func() {
		It("Should return successfully", func() {
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
						Backends: &starlingxv1.StorageBackendList{
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

	Context("Test invalid deployment for Ceph Rook backend", func() {
		It("Should return error", func() {
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
						Backends: &starlingxv1.StorageBackendList{
							{
								Name:       nameBackend,
								Type:       typeBackend,
								Deployment: deploymentModel,
							},
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, created)).Error()
		})
	})

	Context("Test creating ceph-float Controller Filesystem", func() {
		It("Should return successfully", func() {
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
						Backends: &starlingxv1.StorageBackendList{
							{
								Name:       nameBackend,
								Type:       typeBackend,
								Deployment: deploymentModel,
							},
						},
						FileSystems: &starlingxv1.ControllerFileSystemList{
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
	Context("For the FixCertsToManage func when current has runtimeCerts", func() {
		It("Should remove the rutime certs and return the remaining", func() {
			specCerts := &starlingxv1.CertificateList{
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
			currentCerts := &starlingxv1.CertificateList{
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

			expRes := starlingxv1.CertificateList{
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
})
