/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2024-2026 Wind River Systems, Inc. */
package v1

import (
	"errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	starlingxv1 "github.com/wind-river/cloud-platform-deployment-manager/api/v1"
)

var _ = Describe("SystemWebhook", func() {

	Describe("ValidateBackendServices", func() {
		Context("When services belong to the backendType ", func() {
			It("should return true", func() {
				backendType := "ceph"
				services := []string{"cinder", "nova"}
				err := validateBackendServices(backendType, services)

				Expect(err).ToNot(HaveOccurred())
			})
		})
		Context("When services doesnt belong to the backendType ", func() {
			It("should return the error that service is not allowed with the backendType backend", func() {
				backendType := "lvm"
				services := []string{"cinder", "nova"}
				err := validateBackendServices(backendType, services)

				msg := errors.New("nova service not allowed with lvm backend.")
				Expect(err).To(Equal(msg))

			})
		})
	})
	Describe("ValidateBackendAttributes", func() {
		Context("When the type is ceph and partitionSize and replicationFactor is specified", func() {
			It("should validate backend attributes successfully without any error", func() {
				prtSize := 20
				repFac := 2
				backend := starlingxv1.StorageBackend{
					PartitionSize:     &prtSize,
					ReplicationFactor: &repFac,
					Type:              ceph,
				}

				err := validateBackendAttributes(backend)
				Expect(err).ToNot(HaveOccurred())
			})
		})
		Context("When partitionSize is present when the Type is file", func() {
			It("Should return the error partitionSize only permitted with ceph backend", func() {
				prtSize := 20
				backend := starlingxv1.StorageBackend{
					PartitionSize: &prtSize,
					Type:          file,
				}

				err := validateBackendAttributes(backend)
				msg := errors.New("partitionSize is only permitted with ceph backend")
				Expect(err).To(Equal(msg))
			})
		})
		Context("When replicationFactor is present when the Type is file", func() {
			It("Should return the error ReplicationFactor only permitted with ceph and ceph-rook backends", func() {
				repFac := 2
				backend := starlingxv1.StorageBackend{
					ReplicationFactor: &repFac,
					Type:              file,
				}

				err := validateBackendAttributes(backend)
				msg := errors.New("replicationFactor is only permitted with ceph and ceph-rook backends")
				Expect(err).To(Equal(msg))
			})
		})
		Context("When replicationFactor is present when the Type is ceph-rook", func() {
			It("Should returns success", func() {
				repFac := 2
				backend := starlingxv1.StorageBackend{
					ReplicationFactor: &repFac,
					Type:              rook,
				}

				err := validateBackendAttributes(backend)
				Expect(err).To(Succeed())
			})
		})
		Context("When deployment is present when the Type is file", func() {
			It("Should return the error Deployment only permitted with ceph-rook backend", func() {
				deploymentModel := "open"
				backend := starlingxv1.StorageBackend{
					Deployment: deploymentModel,
					Type:       file,
				}

				err := validateBackendAttributes(backend)
				msg := errors.New("deployment is only permitted with ceph-rook backend")
				Expect(err).To(Equal(msg))
			})
		})
	})
	Describe("ValidateStorageBackends", func() {
		Context("When backend type is unique", func() {
			It("should return nil", func() {
				prtSize := 20
				repFac := 2

				obj := &starlingxv1.System{
					Spec: starlingxv1.SystemSpec{
						Storage: &starlingxv1.SystemStorageInfo{
							Backends: starlingxv1.StorageBackendList{
								{
									PartitionSize:     &prtSize,
									ReplicationFactor: &repFac,
									Type:              ceph,
									Services:          []string{"cinder", "nova"},
								},
							},
						},
					},
				}
				err := validateStorageBackends(obj)
				Expect(err).ToNot(HaveOccurred())
			})
		})
		Context("When backend type is duplicated", func() {
			It("should return error that backend services may only be specified once", func() {
				prtSize := 20
				repFac := 2

				obj := &starlingxv1.System{
					Spec: starlingxv1.SystemSpec{
						Storage: &starlingxv1.SystemStorageInfo{
							Backends: starlingxv1.StorageBackendList{
								{
									PartitionSize:     &prtSize,
									ReplicationFactor: &repFac,
									Type:              ceph,
									Services:          []string{"cinder", "nova"},
								},
								{
									PartitionSize:     &prtSize,
									ReplicationFactor: &repFac,
									Type:              ceph,
									Services:          []string{"swift"},
								},
							},
						},
					},
				}
				err := validateStorageBackends(obj)
				msg := errors.New("backend services may only be specified once.")
				Expect(err).To(Equal(msg))
			})
		})
		Context("When ceph and ceph-rook backends are added", func() {
			It("should return error that they are not supported at the same time", func() {
				repFac := 10
				deployment := "open"

				obj := &starlingxv1.System{
					Spec: starlingxv1.SystemSpec{
						Storage: &starlingxv1.SystemStorageInfo{
							Backends: starlingxv1.StorageBackendList{
								{
									Type:     ceph,
									Services: []string{"cinder", "nova"},
								},
								{
									Type:              rook,
									ReplicationFactor: &repFac,
									Deployment:        deployment,
									Services:          []string{"block", "filesystem"},
								},
							},
						},
					},
				}
				err := validateStorageBackends(obj)
				msg := errors.New("ceph and ceph-rook backends are not supported at the same time")
				Expect(err).To(Equal(msg))
			})
		})
		Context("When ceph and file backends are added", func() {
			It("should return success", func() {
				obj := &starlingxv1.System{
					Spec: starlingxv1.SystemSpec{
						Storage: &starlingxv1.SystemStorageInfo{
							Backends: starlingxv1.StorageBackendList{
								{
									Type:     ceph,
									Services: []string{"cinder", "nova"},
								},
								{
									Type: file,
								},
							},
						},
					},
				}
				err := validateStorageBackends(obj)
				Expect(err).To(Succeed())
			})
		})
	})
	Describe("ValidateStorage", func() {
		Context("When Backends is not nil and services are belonging to the backend type", func() {
			It("should return nil error", func() {
				prtSize := 20
				repFac := 2
				obj := &starlingxv1.System{
					Spec: starlingxv1.SystemSpec{
						Storage: &starlingxv1.SystemStorageInfo{
							Backends: starlingxv1.StorageBackendList{
								{
									PartitionSize:     &prtSize,
									ReplicationFactor: &repFac,
									Type:              ceph,
									Services:          []string{"cinder", "nova"},
								},
							},
						},
					},
				}

				err := validateStorage(obj)
				Expect(err).ToNot(HaveOccurred())
			})
		})
	})
})
