/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2024 Wind River Systems, Inc. */
package v1

import (
	"errors"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("system_webhook functions", func() {

	Describe("validateBackendServices function is tested", func() {
		Context("When services belong to the backendType ", func() {
			It("returns true", func() {
				backendType := "ceph"
				services := []string{"cinder", "nova"}
				err := validateBackendServices(backendType, services)

				Expect(err).To(BeNil())
			})
		})
		Context("When services doesnt belong to the backendType ", func() {
			It("Returns the error that <service> service not allowed with the <backendType> backend", func() {
				backendType := "lvm"
				services := []string{"cinder", "nova"}
				err := validateBackendServices(backendType, services)

				msg := errors.New("nova service not allowed with lvm backend.")
				Expect(err).To(Equal(msg))

			})
		})
	})
	Describe("validateBackendAttributes function is tested", func() {
		Context("When the type is ceph and partitionSize and replicationFactor is specified", func() {
			It("Validtes backend attributes sucessfully without any error", func() {
				prtSize := 20
				repFac := 2
				backend := StorageBackend{
					PartitionSize:     &prtSize,
					ReplicationFactor: &repFac,
					Type:              ceph,
				}

				err := validateBackendAttributes(backend)
				Expect(err).To(BeNil())
			})
		})
		Context("When partitionSize is present when the Type is file", func() {
			It("Should return the error partitionSize only permitted with ceph backend", func() {
				prtSize := 20
				backend := StorageBackend{
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
				backend := StorageBackend{
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
				backend := StorageBackend{
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
				backend := StorageBackend{
					Deployment: deploymentModel,
					Type:       file,
				}

				err := validateBackendAttributes(backend)
				msg := errors.New("deployment is only permitted with ceph-rook backend")
				Expect(err).To(Equal(msg))
			})
		})
	})
	Describe("validateStorageBackends function is tested", func() {
		Context("When backend type is unique", func() {
			It("Returns nil", func() {
				prtSize := 20
				repFac := 2

				obj := &System{
					Spec: SystemSpec{
						Storage: &SystemStorageInfo{
							Backends: &StorageBackendList{
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
				Expect(err).To(BeNil())
			})
		})
		Context("When backend type is duplicated", func() {
			It("Returns error that backend services may only be specified once.", func() {
				prtSize := 20
				repFac := 2

				obj := &System{
					Spec: SystemSpec{
						Storage: &SystemStorageInfo{
							Backends: &StorageBackendList{
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
			It("Returns error that they are not supported at the same time.", func() {
				repFac := 10
				deployment := "open"

				obj := &System{
					Spec: SystemSpec{
						Storage: &SystemStorageInfo{
							Backends: &StorageBackendList{
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
			It("Returns success.", func() {
				obj := &System{
					Spec: SystemSpec{
						Storage: &SystemStorageInfo{
							Backends: &StorageBackendList{
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
	Describe("validateStorage function is tested", func() {
		Context("When Backends is not nil and services are belonging to the backend type", func() {
			It("Returns nil error", func() {
				prtSize := 20
				repFac := 2
				obj := &System{
					Spec: SystemSpec{
						Storage: &SystemStorageInfo{
							Backends: &StorageBackendList{
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
				Expect(err).To(BeNil())
			})
		})
	})
})
