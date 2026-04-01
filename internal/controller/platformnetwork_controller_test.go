/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2022-2026 Wind River Systems, Inc. */
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

func newPlatformNetworkReconciler(dm *cloudManager.Dummymanager) *PlatformNetworkReconciler {
	logger := log.Log.WithName("test")
	return &PlatformNetworkReconciler{
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

var _ = Describe("Platformnetwork controller", func() {
	const (
		timeout  = time.Second * 30
		interval = time.Millisecond * 500
	)

	Context("with PlatformNetwork data", func() {
		It("should be created successfully", func() {
			ctx := context.Background()
			key := types.NamespacedName{
				Name:      "foo",
				Namespace: "default",
			}
			created := &starlingxv1.PlatformNetwork{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: "default",
				},
				Spec: starlingxv1.PlatformNetworkSpec{
					Type:                   "mgmt",
					Dynamic:                true,
					AssociatedAddressPools: []string{"management-ipv4", "management-ipv6"},
				}}
			Expect(k8sClient.Create(ctx, created)).To(Succeed())

			fetched := &starlingxv1.PlatformNetwork{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, key, fetched)
				if err != nil {
					return false
				}
				_, found := comm.ListIntersect(fetched.ObjectMeta.Finalizers, []string{PlatformNetworkFinalizerName})
				return found
			}, timeout, interval).Should(BeTrue())
		})
	})

	Describe("statusUpdateRequired", func() {
		It("should return false when status has not changed", func() {
			dm := &cloudManager.Dummymanager{}
			reconciler := newPlatformNetworkReconciler(dm)
			instance := &starlingxv1.PlatformNetwork{}
			old := instance.Status.DeepCopy()
			Expect(reconciler.statusUpdateRequired(instance, old)).To(BeFalse())
		})

		It("should return true when InSync changes", func() {
			dm := &cloudManager.Dummymanager{}
			reconciler := newPlatformNetworkReconciler(dm)
			instance := &starlingxv1.PlatformNetwork{}
			old := instance.Status.DeepCopy()
			instance.Status.InSync = true
			Expect(reconciler.statusUpdateRequired(instance, old)).To(BeTrue())
		})
	})

	Describe("UpdateInsyncStatus", func() {
		It("should update when status changed", func() {
			dm := &cloudManager.Dummymanager{}
			reconciler := newPlatformNetworkReconciler(dm)
			instance := &starlingxv1.PlatformNetwork{
				ObjectMeta: metav1.ObjectMeta{Name: "pn-insync", Namespace: "default"},
				Spec: starlingxv1.PlatformNetworkSpec{
					Type:                   "mgmt",
					Dynamic:                true,
					AssociatedAddressPools: []string{"management-ipv4"},
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

		It("should not update when status unchanged", func() {
			dm := &cloudManager.Dummymanager{}
			reconciler := newPlatformNetworkReconciler(dm)
			instance := &starlingxv1.PlatformNetwork{
				ObjectMeta: metav1.ObjectMeta{Name: "pn-insync-noop", Namespace: "default"},
				Spec: starlingxv1.PlatformNetworkSpec{
					Type:                   "mgmt",
					Dynamic:                true,
					AssociatedAddressPools: []string{"management-ipv4"},
				},
			}
			Expect(k8sClient.Create(ctx, instance)).To(Succeed())

			old := instance.Status.DeepCopy()
			err := reconciler.UpdateInsyncStatus(nil, instance, old)
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Describe("ReconcileResource", func() {
		Context("when the resource is being deleted", func() {
			It("should remove the finalizer", func() {
				dm := &cloudManager.Dummymanager{}
				reconciler := newPlatformNetworkReconciler(dm)
				instance := &starlingxv1.PlatformNetwork{
					ObjectMeta: metav1.ObjectMeta{
						Name:       "pn-delete",
						Namespace:  "default",
						Finalizers: []string{PlatformNetworkFinalizerName},
					},
					Spec: starlingxv1.PlatformNetworkSpec{
						Type:                   "mgmt",
						Dynamic:                true,
						AssociatedAddressPools: []string{"management-ipv4"},
					},
				}
				Expect(k8sClient.Create(ctx, instance)).To(Succeed())
				Expect(k8sClient.Delete(ctx, instance)).To(Succeed())

				Eventually(func() bool {
					err := k8sClient.Get(ctx, client.ObjectKeyFromObject(instance), instance)
					return err == nil && !instance.DeletionTimestamp.IsZero()
				}, timeout, interval).Should(BeTrue())

				err := reconciler.ReconcileResource(nil, instance, "default", false)
				Expect(err).ToNot(HaveOccurred())
				Expect(instance.Finalizers).ToNot(ContainElement(PlatformNetworkFinalizerName))
			})
		})

		Context("when the resource generation changed", func() {
			It("should notify the active host", func() {
				activeHost := &starlingxv1.Host{
					ObjectMeta: metav1.ObjectMeta{Name: "controller-0-pn", Namespace: "default"},
				}
				Expect(k8sClient.Create(ctx, activeHost)).To(Succeed())

				dm := &cloudManager.Dummymanager{ActiveHost: activeHost}
				reconciler := newPlatformNetworkReconciler(dm)
				instance := &starlingxv1.PlatformNetwork{
					ObjectMeta: metav1.ObjectMeta{Name: "pn-notify", Namespace: "default"},
					Spec: starlingxv1.PlatformNetworkSpec{
						Type:                   "oam",
						Dynamic:                false,
						AssociatedAddressPools: []string{"oam-ipv4"},
					},
				}
				Expect(k8sClient.Create(ctx, instance)).To(Succeed())

				Eventually(func() bool {
					_ = k8sClient.Get(ctx, client.ObjectKeyFromObject(instance), instance)
					return len(instance.Finalizers) > 0
				}, timeout, interval).Should(BeTrue())

				err := reconciler.ReconcileResource(nil, instance, "default", false)
				Expect(err).ToNot(HaveOccurred())
				Expect(instance.Status.ObservedGeneration).To(Equal(instance.Generation))
			})
		})
	})
})
