/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2022, 2024-2026 Wind River Systems, Inc. */
package controller

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	starlingxv1 "github.com/wind-river/cloud-platform-deployment-manager/api/v1"
	comm "github.com/wind-river/cloud-platform-deployment-manager/common"
	"github.com/wind-river/cloud-platform-deployment-manager/internal/controller/common"
	cloudManager "github.com/wind-river/cloud-platform-deployment-manager/internal/controller/manager"
)

func newAddressPoolReconciler() *AddressPoolReconciler {
	dm := &cloudManager.Dummymanager{}
	logger := log.Log.WithName("test")
	return &AddressPoolReconciler{
		Client:       k8sClient,
		CloudManager: dm,
		ReconcilerErrorHandler: &common.ErrorHandler{
			CloudManager: dm,
			Logger:       logger,
		},
		ReconcilerEventLogger: &common.EventLogger{
			EventRecorder: record.NewFakeRecorder(100),
			Logger:        logger,
		},
	}
}

var _ = Describe("AddressPool controller", func() {
	const (
		timeout  = time.Second * 30
		interval = time.Millisecond * 500
	)

	Context("with AddressPool data", func() {
		It("should be created successfully", func() {
			ctx := context.Background()
			key := types.NamespacedName{
				Name:      "foo",
				Namespace: "default",
			}

			floating_address := "10.10.10.2"
			allocation_order := "random"
			spec := starlingxv1.AddressPoolSpec{
				Subnet:             "10.10.10.0",
				FloatingAddress:    &floating_address,
				Controller0Address: &floating_address,
				Controller1Address: &floating_address,
				Prefix:             24,
				Gateway:            &floating_address,
				Allocation: starlingxv1.AllocationInfo{
					Order: &allocation_order,
				},
			}

			spec.Allocation.Ranges = []starlingxv1.AllocationRange{starlingxv1.AllocationRange{
				Start: floating_address,
				End:   floating_address,
			}}

			created := &starlingxv1.AddressPool{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: "default",
				},
				Spec: spec}
			Expect(k8sClient.Create(ctx, created)).To(Succeed())

			fetched := &starlingxv1.AddressPool{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, key, fetched)
				if err != nil {
					return false
				}
				_, found := comm.ListIntersect(fetched.ObjectMeta.Finalizers, []string{AddressPoolFinalizerName})
				return found
			}, timeout, interval).Should(BeTrue())
		})
	})

	Describe("statusUpdateRequired", func() {
		var reconciler *AddressPoolReconciler

		BeforeEach(func() {
			reconciler = newAddressPoolReconciler()
		})

		Context("when status has not changed", func() {
			It("should return false", func() {
				instance := &starlingxv1.AddressPool{}
				old := instance.Status.DeepCopy()
				Expect(reconciler.statusUpdateRequired(instance, old)).To(BeFalse())
			})
		})

		Context("when InSync changes", func() {
			It("should return true", func() {
				instance := &starlingxv1.AddressPool{}
				old := instance.Status.DeepCopy()
				instance.Status.InSync = true
				Expect(reconciler.statusUpdateRequired(instance, old)).To(BeTrue())
			})
		})

		Context("when Reconciled changes", func() {
			It("should return true", func() {
				instance := &starlingxv1.AddressPool{}
				old := instance.Status.DeepCopy()
				instance.Status.Reconciled = true
				Expect(reconciler.statusUpdateRequired(instance, old)).To(BeTrue())
			})
		})

		Context("when ID changes", func() {
			It("should return true", func() {
				id := "some-uuid"
				instance := &starlingxv1.AddressPool{}
				old := instance.Status.DeepCopy()
				instance.Status.ID = &id
				Expect(reconciler.statusUpdateRequired(instance, old)).To(BeTrue())
			})
		})
	})

	Describe("UpdateInsyncStatus", func() {
		var reconciler *AddressPoolReconciler

		BeforeEach(func() {
			reconciler = newAddressPoolReconciler()
		})

		Context("when status changed", func() {
			It("should update the resource status", func() {
				floating := "10.10.10.12"
				instance := &starlingxv1.AddressPool{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "ap-insync-changed",
						Namespace: "default",
					},
					Spec: starlingxv1.AddressPoolSpec{
						Subnet:          "10.10.10.0",
						Prefix:          24,
						FloatingAddress: &floating,
					},
				}
				Expect(k8sClient.Create(ctx, instance)).To(Succeed())

				Eventually(func() bool {
					_ = k8sClient.Get(ctx, client.ObjectKeyFromObject(instance), instance)
					return len(instance.Finalizers) > 0
				}, timeout, interval).Should(BeTrue())

				old := instance.Status.DeepCopy()
				instance.Status.InSync = true
				err := reconciler.UpdateInsyncStatus(nil, instance, old)
				Expect(err).ToNot(HaveOccurred())
			})
		})

		Context("when status has not changed", func() {
			It("should not update", func() {
				floating := "10.10.10.13"
				instance := &starlingxv1.AddressPool{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "ap-insync-same",
						Namespace: "default",
					},
					Spec: starlingxv1.AddressPoolSpec{
						Subnet:          "10.10.10.0",
						Prefix:          24,
						FloatingAddress: &floating,
					},
				}
				Expect(k8sClient.Create(ctx, instance)).To(Succeed())

				old := instance.Status.DeepCopy()
				err := reconciler.UpdateInsyncStatus(nil, instance, old)
				Expect(err).ToNot(HaveOccurred())
			})
		})
	})

	Describe("ReconcileResource", func() {
		Context("when the resource is being deleted", func() {
			It("should remove the finalizer", func() {
				floating := "10.10.10.14"
				instance := &starlingxv1.AddressPool{
					ObjectMeta: metav1.ObjectMeta{
						Name:       "ap-delete",
						Namespace:  "default",
						Finalizers: []string{AddressPoolFinalizerName},
					},
					Spec: starlingxv1.AddressPoolSpec{
						Subnet:          "10.10.10.0",
						Prefix:          24,
						FloatingAddress: &floating,
					},
				}
				Expect(k8sClient.Create(ctx, instance)).To(Succeed())

				Expect(k8sClient.Delete(ctx, instance)).To(Succeed())

				Eventually(func() bool {
					err := k8sClient.Get(ctx, client.ObjectKeyFromObject(instance), instance)
					return err == nil && !instance.DeletionTimestamp.IsZero()
				}, timeout, interval).Should(BeTrue())

				reconciler := newAddressPoolReconciler()
				err := reconciler.ReconcileResource(nil, instance, "default")
				Expect(err).ToNot(HaveOccurred())
				Expect(instance.Finalizers).ToNot(ContainElement(AddressPoolFinalizerName))
			})
		})

		Context("when the resource is created and generation changed", func() {
			It("should notify the active host and update observed generation", func() {
				floating := "10.10.10.15"
				instance := &starlingxv1.AddressPool{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "ap-notify",
						Namespace: "default",
					},
					Spec: starlingxv1.AddressPoolSpec{
						Subnet:          "10.10.10.0",
						Prefix:          24,
						FloatingAddress: &floating,
					},
				}
				Expect(k8sClient.Create(ctx, instance)).To(Succeed())

				Eventually(func() bool {
					_ = k8sClient.Get(ctx, client.ObjectKeyFromObject(instance), instance)
					return len(instance.Finalizers) > 0
				}, timeout, interval).Should(BeTrue())

				activeHost := &starlingxv1.Host{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "controller-0-ap",
						Namespace: "default",
					},
				}
				Expect(k8sClient.Create(ctx, activeHost)).To(Succeed())

				dm := &cloudManager.Dummymanager{ActiveHost: activeHost}
				logger := log.Log.WithName("test")
				reconciler := &AddressPoolReconciler{
					Client:       k8sClient,
					CloudManager: dm,
					ReconcilerErrorHandler: &common.ErrorHandler{
						CloudManager: dm,
						Logger:       logger,
					},
					ReconcilerEventLogger: &common.EventLogger{
						EventRecorder: record.NewFakeRecorder(100),
						Logger:        logger,
					},
				}

				err := reconciler.ReconcileResource(nil, instance, "default")
				Expect(err).ToNot(HaveOccurred())
				Expect(instance.Status.ObservedGeneration).To(Equal(instance.Generation))
			})
		})
	})
})
