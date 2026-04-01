/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2022, 2024-2026 Wind River Systems, Inc. */
package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/ptpinstances"
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

const ptpInstanceResponse = `{
	"uuid": "test-ptp-uuid",
	"id": 1,
	"name": "ptp4l-instance-0",
	"service": "ptp4l",
	"parameters": {}
}`

const ptpInstanceListResponse = `{
	"ptp_instances": [` + ptpInstanceResponse + `]
}`

func newPtpInstanceFixtureServer() (*httptest.Server, *gophercloud.ServiceClient) {
	mux := http.NewServeMux()
	mux.HandleFunc("/ptp_instances", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodGet:
			_, _ = fmt.Fprint(w, ptpInstanceListResponse)
		case http.MethodPost:
			var body map[string]interface{}
			_ = json.NewDecoder(r.Body).Decode(&body)
			name, _ := body["name"].(string)
			svc, _ := body["service"].(string)
			resp := fmt.Sprintf(`{"uuid":"new-ptp-uuid","id":2,"name":"%s","service":"%s","parameters":{}}`, name, svc)
			w.WriteHeader(http.StatusOK)
			_, _ = fmt.Fprint(w, resp)
		}
	})
	mux.HandleFunc("/ptp_instances/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodGet:
			_, _ = fmt.Fprint(w, ptpInstanceResponse)
		case http.MethodPatch:
			_, _ = fmt.Fprint(w, ptpInstanceResponse)
		case http.MethodDelete:
			w.WriteHeader(http.StatusNoContent)
		}
	})

	server := httptest.NewServer(mux)
	sc := &gophercloud.ServiceClient{
		ProviderClient: &gophercloud.ProviderClient{TokenID: "test-token"},
		Endpoint:       server.URL + "/",
	}
	return server, sc
}

func newPtpInstanceReconciler() *PtpInstanceReconciler {
	dm := &cloudManager.Dummymanager{}
	logger := log.Log.WithName("test")
	return &PtpInstanceReconciler{
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

var _ = Describe("PtpInstance controller", func() {
	const (
		timeout  = time.Second * 30
		interval = time.Millisecond * 500
	)

	Describe("Create PtpInstance", func() {
		Context("with single section data", func() {
			It("should be created successfully", func() {
				ctx := context.Background()
				key := types.NamespacedName{
					Name:      "foo",
					Namespace: "default",
				}
				created := &starlingxv1.PtpInstance{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "foo",
						Namespace: "default",
					},
					Spec: starlingxv1.PtpInstanceSpec{
						Service:            "ptp4l",
						InstanceParameters: map[string][]string{"global": {"domainNumber=24", "clientOnly=0"}},
					}}
				Expect(k8sClient.Create(ctx, created)).To(Succeed())

				fetched := &starlingxv1.PtpInstance{}
				Eventually(func() bool {
					err := k8sClient.Get(ctx, key, fetched)
					if err != nil {
						return false
					}
					_, found := comm.ListIntersect(fetched.ObjectMeta.Finalizers, []string{PtpInstanceFinalizerName})
					return found
				}, timeout, interval).Should(BeTrue())
			})
		})
	})

	Describe("Create PtpInstance", func() {
		Context("with multiple section data", func() {
			It("should create successfully", func() {
				ctx := context.Background()
				key := types.NamespacedName{
					Name:      "bar",
					Namespace: "default",
				}
				created := &starlingxv1.PtpInstance{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "bar",
						Namespace: "default",
					},
					Spec: starlingxv1.PtpInstanceSpec{
						Service: "ptp4l",
						InstanceParameters: map[string][]string{
							"global": {"domainNumber=24", "clientOnly=0"},
							"unicast_master_table_1": {
								"table_id=1",
								"UDPv4=1.2.3.4", "UDPv4=2.3.4.5",
								"L2=00:01:FF:00:01:CD", "L2=00:02:FF:00:01:CD",
								"UDPv6=ffff::1", "UDPv6=ffff::2",
								"peer_address=::1"},
						},
					},
				}
				Expect(k8sClient.Create(ctx, created)).To(Succeed())

				fetched := &starlingxv1.PtpInstance{}
				Eventually(func() bool {
					err := k8sClient.Get(ctx, key, fetched)
					if err != nil {
						return false
					}
					_, found := comm.ListIntersect(fetched.ObjectMeta.Finalizers, []string{PtpInstanceFinalizerName})
					return found
				}, timeout, interval).Should(BeTrue())
			})
		})
	})

	Describe("FindExistingPTPInstance", func() {
		var (
			server     *httptest.Server
			gcClient   *gophercloud.ServiceClient
			reconciler *PtpInstanceReconciler
		)

		BeforeEach(func() {
			server, gcClient = newPtpInstanceFixtureServer()
			reconciler = newPtpInstanceReconciler()
		})
		AfterEach(func() { server.Close() })

		Context("when the resource has a status ID", func() {
			It("should fetch by UUID", func() {
				id := "test-ptp-uuid"
				instance := &starlingxv1.PtpInstance{
					ObjectMeta: metav1.ObjectMeta{Name: "ptp4l-instance-0", Namespace: "default"},
					Spec:       starlingxv1.PtpInstanceSpec{Service: "ptp4l"},
				}
				instance.Status.ID = &id
				found, err := reconciler.FindExistingPTPInstance(gcClient, instance)
				Expect(err).ToNot(HaveOccurred())
				Expect(found).ToNot(BeNil())
				Expect(found.UUID).To(Equal("test-ptp-uuid"))
			})
		})

		Context("when the resource has no status ID", func() {
			It("should find by name from list", func() {
				instance := &starlingxv1.PtpInstance{
					ObjectMeta: metav1.ObjectMeta{Name: "ptp4l-instance-0", Namespace: "default"},
					Spec:       starlingxv1.PtpInstanceSpec{Service: "ptp4l"},
				}
				found, err := reconciler.FindExistingPTPInstance(gcClient, instance)
				Expect(err).ToNot(HaveOccurred())
				Expect(found).ToNot(BeNil())
			})
		})

		Context("when the resource does not exist", func() {
			It("should return nil", func() {
				instance := &starlingxv1.PtpInstance{
					ObjectMeta: metav1.ObjectMeta{Name: "nonexistent", Namespace: "default"},
					Spec:       starlingxv1.PtpInstanceSpec{Service: "ptp4l"},
				}
				found, err := reconciler.FindExistingPTPInstance(gcClient, instance)
				Expect(err).ToNot(HaveOccurred())
				Expect(found).To(BeNil())
			})
		})
	})

	Describe("ReconcileNew", func() {
		var (
			server     *httptest.Server
			gcClient   *gophercloud.ServiceClient
			reconciler *PtpInstanceReconciler
		)

		BeforeEach(func() {
			server, gcClient = newPtpInstanceFixtureServer()
			reconciler = newPtpInstanceReconciler()
		})
		AfterEach(func() { server.Close() })

		Context("when creating a ptp instance", func() {
			It("should succeed", func() {
				instance := &starlingxv1.PtpInstance{
					ObjectMeta: metav1.ObjectMeta{Name: "pi-new", Namespace: "default"},
					Spec:       starlingxv1.PtpInstanceSpec{Service: "ptp4l"},
				}
				result, err := reconciler.ReconcileNew(gcClient, instance)
				Expect(err).ToNot(HaveOccurred())
				Expect(result).ToNot(BeNil())
				Expect(result.UUID).To(Equal("new-ptp-uuid"))
			})
		})

		Context("when already reconciled and StopAfterInSync", func() {
			It("should return error", func() {
				instance := &starlingxv1.PtpInstance{
					ObjectMeta: metav1.ObjectMeta{Name: "pi-new-stop", Namespace: "default"},
					Spec:       starlingxv1.PtpInstanceSpec{Service: "ptp4l"},
				}
				instance.Status.Reconciled = true
				_, err := reconciler.ReconcileNew(gcClient, instance)
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Describe("ReconciledDeleted", func() {
		var (
			server     *httptest.Server
			gcClient   *gophercloud.ServiceClient
			reconciler *PtpInstanceReconciler
		)

		BeforeEach(func() {
			server, gcClient = newPtpInstanceFixtureServer()
			reconciler = newPtpInstanceReconciler()
		})
		AfterEach(func() { server.Close() })

		Context("when the resource has a finalizer and instance exists", func() {
			It("should delete and remove the finalizer", func() {
				instance := &starlingxv1.PtpInstance{
					ObjectMeta: metav1.ObjectMeta{
						Name:       "pi-delete",
						Namespace:  "default",
						Finalizers: []string{PtpInstanceFinalizerName},
					},
					Spec: starlingxv1.PtpInstanceSpec{Service: "ptp4l"},
				}
				Expect(k8sClient.Create(ctx, instance)).To(Succeed())
				Expect(k8sClient.Delete(ctx, instance)).To(Succeed())

				Eventually(func() bool {
					err := k8sClient.Get(ctx, client.ObjectKeyFromObject(instance), instance)
					return err == nil && !instance.DeletionTimestamp.IsZero()
				}, timeout, interval).Should(BeTrue())

				existing := &ptpinstances.PTPInstance{UUID: "test-ptp-uuid", Name: "pi-delete"}
				err := reconciler.ReconciledDeleted(gcClient, instance, existing)
				Expect(err).ToNot(HaveOccurred())
				Expect(instance.Finalizers).ToNot(ContainElement(PtpInstanceFinalizerName))
			})
		})

		Context("when the instance is nil", func() {
			It("should just remove the finalizer", func() {
				instance := &starlingxv1.PtpInstance{
					ObjectMeta: metav1.ObjectMeta{
						Name:       "pi-delete-nil",
						Namespace:  "default",
						Finalizers: []string{PtpInstanceFinalizerName},
					},
					Spec: starlingxv1.PtpInstanceSpec{Service: "ptp4l"},
				}
				Expect(k8sClient.Create(ctx, instance)).To(Succeed())
				Expect(k8sClient.Delete(ctx, instance)).To(Succeed())

				Eventually(func() bool {
					err := k8sClient.Get(ctx, client.ObjectKeyFromObject(instance), instance)
					return err == nil && !instance.DeletionTimestamp.IsZero()
				}, timeout, interval).Should(BeTrue())

				err := reconciler.ReconciledDeleted(gcClient, instance, nil)
				Expect(err).ToNot(HaveOccurred())
			})
		})
	})

	Describe("ReconcileResource", func() {
		var (
			server     *httptest.Server
			gcClient   *gophercloud.ServiceClient
			reconciler *PtpInstanceReconciler
		)

		BeforeEach(func() {
			server, gcClient = newPtpInstanceFixtureServer()
			reconciler = newPtpInstanceReconciler()
		})
		AfterEach(func() { server.Close() })

		Context("when the resource exists on the system", func() {
			It("should reconcile successfully", func() {
				instance := &starlingxv1.PtpInstance{
					ObjectMeta: metav1.ObjectMeta{Name: "pi-reconcile-exist", Namespace: "default"},
					Spec:       starlingxv1.PtpInstanceSpec{Service: "ptp4l"},
				}
				Expect(k8sClient.Create(ctx, instance)).To(Succeed())

				Eventually(func() bool {
					_ = k8sClient.Get(ctx, client.ObjectKeyFromObject(instance), instance)
					return len(instance.Finalizers) > 0
				}, timeout, interval).Should(BeTrue())

				id := "test-ptp-uuid"
				instance.Status.ID = &id
				err := reconciler.ReconcileResource(gcClient, instance)
				Expect(err).ToNot(HaveOccurred())
			})
		})

		Context("when the resource does not exist on the system", func() {
			It("should create a new ptp instance", func() {
				instance := &starlingxv1.PtpInstance{
					ObjectMeta: metav1.ObjectMeta{Name: "pi-reconcile-new", Namespace: "default"},
					Spec:       starlingxv1.PtpInstanceSpec{Service: "ptp4l"},
				}
				Expect(k8sClient.Create(ctx, instance)).To(Succeed())

				Eventually(func() bool {
					_ = k8sClient.Get(ctx, client.ObjectKeyFromObject(instance), instance)
					return len(instance.Finalizers) > 0
				}, timeout, interval).Should(BeTrue())

				err := reconciler.ReconcileResource(gcClient, instance)
				Expect(err).ToNot(HaveOccurred())
			})
		})
	})
})
