/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2022, 2024-2026 Wind River Systems, Inc. */
package controller

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/ptpinterfaces"
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

const ptpInterfaceResponse = `{
	"uuid": "test-ptpif-uuid",
	"id": 1,
	"name": "ptp-iface-0",
	"ptp_instance_uuid": "test-ptp-uuid",
	"ptp_instance_name": "ptp4l-instance-0",
	"parameters": []
}`

const ptpInterfaceListResponse = `{
	"ptp_interfaces": [` + ptpInterfaceResponse + `]
}`

func newPtpInterfaceFixtureServer() (*httptest.Server, *gophercloud.ServiceClient) {
	mux := http.NewServeMux()

	// ptp_instances endpoint needed by findPTPInstanceByName
	mux.HandleFunc("/ptp_instances", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, ptpInstanceListResponse)
	})
	mux.HandleFunc("/ptp_instances/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, ptpInstanceResponse)
	})

	mux.HandleFunc("/ptp_interfaces", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodGet:
			_, _ = fmt.Fprint(w, ptpInterfaceListResponse)
		case http.MethodPost:
			var body map[string]interface{}
			_ = json.NewDecoder(r.Body).Decode(&body)
			name, _ := body["name"].(string)
			resp := fmt.Sprintf(`{"uuid":"new-ptpif-uuid","id":2,"name":"%s","ptp_instance_uuid":"test-ptp-uuid","ptp_instance_name":"ptp4l-instance-0","parameters":[]}`, name)
			w.WriteHeader(http.StatusOK)
			_, _ = fmt.Fprint(w, resp)
		}
	})
	mux.HandleFunc("/ptp_interfaces/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodGet:
			_, _ = fmt.Fprint(w, ptpInterfaceResponse)
		case http.MethodPatch:
			_, _ = fmt.Fprint(w, ptpInterfaceResponse)
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

func newPtpInterfaceReconciler() *PtpInterfaceReconciler {
	dm := &cloudManager.Dummymanager{}
	logger := log.Log.WithName("test")
	return &PtpInterfaceReconciler{
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

var _ = Describe("PtpInterface controller", func() {
	const (
		timeout  = time.Second * 30
		interval = time.Millisecond * 500
	)

	Context("with PtpInterface data", func() {
		It("should be created successfully", func() {
			key := types.NamespacedName{
				Name:      "ptp-iface-create",
				Namespace: "default",
			}
			created := &starlingxv1.PtpInterface{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ptp-iface-create",
					Namespace: "default",
				},
				Spec: starlingxv1.PtpInterfaceSpec{
					PtpInstance: "ptp4l-instance-0",
				}}
			Expect(k8sClient.Create(ctx, created)).To(Succeed())

			fetched := &starlingxv1.PtpInterface{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, key, fetched)
				if err != nil {
					return false
				}
				_, found := comm.ListIntersect(fetched.ObjectMeta.Finalizers, []string{PtpInterfaceFinalizerName})
				return found
			}, timeout, interval).Should(BeTrue())
		})
	})

	Describe("FindExistingPTPInterface", func() {
		var (
			server     *httptest.Server
			gcClient   *gophercloud.ServiceClient
			reconciler *PtpInterfaceReconciler
		)

		BeforeEach(func() {
			server, gcClient = newPtpInterfaceFixtureServer()
			reconciler = newPtpInterfaceReconciler()
		})
		AfterEach(func() { server.Close() })

		Context("when the resource has a status ID", func() {
			It("should fetch by UUID", func() {
				id := "test-ptpif-uuid"
				instance := &starlingxv1.PtpInterface{
					ObjectMeta: metav1.ObjectMeta{Name: "ptp-iface-0", Namespace: "default"},
					Spec:       starlingxv1.PtpInterfaceSpec{PtpInstance: "ptp4l-instance-0"},
				}
				instance.Status.ID = &id
				found, err := reconciler.FindExistingPTPInterface(gcClient, instance)
				Expect(err).ToNot(HaveOccurred())
				Expect(found).ToNot(BeNil())
				Expect(found.UUID).To(Equal("test-ptpif-uuid"))
			})
		})

		Context("when the resource has no status ID", func() {
			It("should find by name from list", func() {
				instance := &starlingxv1.PtpInterface{
					ObjectMeta: metav1.ObjectMeta{Name: "ptp-iface-0", Namespace: "default"},
					Spec:       starlingxv1.PtpInterfaceSpec{PtpInstance: "ptp4l-instance-0"},
				}
				found, err := reconciler.FindExistingPTPInterface(gcClient, instance)
				Expect(err).ToNot(HaveOccurred())
				Expect(found).ToNot(BeNil())
			})
		})

		Context("when the resource does not exist", func() {
			It("should return nil", func() {
				instance := &starlingxv1.PtpInterface{
					ObjectMeta: metav1.ObjectMeta{Name: "nonexistent", Namespace: "default"},
					Spec:       starlingxv1.PtpInterfaceSpec{PtpInstance: "ptp4l-instance-0"},
				}
				found, err := reconciler.FindExistingPTPInterface(gcClient, instance)
				Expect(err).ToNot(HaveOccurred())
				Expect(found).To(BeNil())
			})
		})
	})

	Describe("ReconcileNew", func() {
		var (
			server     *httptest.Server
			gcClient   *gophercloud.ServiceClient
			reconciler *PtpInterfaceReconciler
		)

		BeforeEach(func() {
			server, gcClient = newPtpInterfaceFixtureServer()
			reconciler = newPtpInterfaceReconciler()
		})
		AfterEach(func() { server.Close() })

		Context("when creating a ptp interface", func() {
			It("should succeed", func() {
				instance := &starlingxv1.PtpInterface{
					ObjectMeta: metav1.ObjectMeta{Name: "ptpif-new", Namespace: "default"},
					Spec:       starlingxv1.PtpInterfaceSpec{PtpInstance: "ptp4l-instance-0"},
				}
				result, err := reconciler.ReconcileNew(gcClient, instance)
				Expect(err).ToNot(HaveOccurred())
				Expect(result).ToNot(BeNil())
				Expect(result.UUID).To(Equal("new-ptpif-uuid"))
			})
		})

		Context("when already reconciled and StopAfterInSync", func() {
			It("should return error", func() {
				instance := &starlingxv1.PtpInterface{
					ObjectMeta: metav1.ObjectMeta{Name: "ptpif-new-stop", Namespace: "default"},
					Spec:       starlingxv1.PtpInterfaceSpec{PtpInstance: "ptp4l-instance-0"},
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
			reconciler *PtpInterfaceReconciler
		)

		BeforeEach(func() {
			server, gcClient = newPtpInterfaceFixtureServer()
			reconciler = newPtpInterfaceReconciler()
		})
		AfterEach(func() { server.Close() })

		Context("when the resource has a finalizer and interface exists", func() {
			It("should delete and remove the finalizer", func() {
				instance := &starlingxv1.PtpInterface{
					ObjectMeta: metav1.ObjectMeta{
						Name:       "ptpif-delete",
						Namespace:  "default",
						Finalizers: []string{PtpInterfaceFinalizerName},
					},
					Spec: starlingxv1.PtpInterfaceSpec{PtpInstance: "ptp4l-instance-0"},
				}
				Expect(k8sClient.Create(ctx, instance)).To(Succeed())
				Expect(k8sClient.Delete(ctx, instance)).To(Succeed())

				Eventually(func() bool {
					err := k8sClient.Get(ctx, client.ObjectKeyFromObject(instance), instance)
					return err == nil && !instance.DeletionTimestamp.IsZero()
				}, timeout, interval).Should(BeTrue())

				existing := &ptpinterfaces.PTPInterface{UUID: "test-ptpif-uuid", Name: "ptpif-delete"}
				err := reconciler.ReconciledDeleted(gcClient, instance, existing)
				Expect(err).ToNot(HaveOccurred())
				Expect(instance.Finalizers).ToNot(ContainElement(PtpInterfaceFinalizerName))
			})
		})

		Context("when the interface is nil", func() {
			It("should just remove the finalizer", func() {
				instance := &starlingxv1.PtpInterface{
					ObjectMeta: metav1.ObjectMeta{
						Name:       "ptpif-delete-nil",
						Namespace:  "default",
						Finalizers: []string{PtpInterfaceFinalizerName},
					},
					Spec: starlingxv1.PtpInterfaceSpec{PtpInstance: "ptp4l-instance-0"},
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
			reconciler *PtpInterfaceReconciler
		)

		BeforeEach(func() {
			server, gcClient = newPtpInterfaceFixtureServer()
			reconciler = newPtpInterfaceReconciler()
		})
		AfterEach(func() { server.Close() })

		Context("when the resource exists on the system", func() {
			It("should reconcile successfully", func() {
				instance := &starlingxv1.PtpInterface{
					ObjectMeta: metav1.ObjectMeta{Name: "ptpif-reconcile-exist", Namespace: "default"},
					Spec:       starlingxv1.PtpInterfaceSpec{PtpInstance: "ptp4l-instance-0"},
				}
				Expect(k8sClient.Create(ctx, instance)).To(Succeed())

				Eventually(func() bool {
					_ = k8sClient.Get(ctx, client.ObjectKeyFromObject(instance), instance)
					return len(instance.Finalizers) > 0
				}, timeout, interval).Should(BeTrue())

				id := "test-ptpif-uuid"
				instance.Status.ID = &id
				err := reconciler.ReconcileResource(gcClient, instance)
				Expect(err).ToNot(HaveOccurred())
			})
		})

		Context("when the resource does not exist on the system", func() {
			It("should create a new ptp interface", func() {
				instance := &starlingxv1.PtpInterface{
					ObjectMeta: metav1.ObjectMeta{Name: "ptpif-reconcile-new", Namespace: "default"},
					Spec:       starlingxv1.PtpInterfaceSpec{PtpInstance: "ptp4l-instance-0"},
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

	Describe("findPTPInstanceByName", func() {
		var (
			server   *httptest.Server
			gcClient *gophercloud.ServiceClient
		)

		BeforeEach(func() {
			server, gcClient = newPtpInterfaceFixtureServer()
		})
		AfterEach(func() { server.Close() })

		Context("when the instance exists", func() {
			It("should return it", func() {
				found, err := findPTPInstanceByName(gcClient, "ptp4l-instance-0")
				Expect(err).ToNot(HaveOccurred())
				Expect(found).ToNot(BeNil())
				Expect(found.Name).To(Equal("ptp4l-instance-0"))
			})
		})

		Context("when the instance does not exist", func() {
			It("should return nil", func() {
				found, err := findPTPInstanceByName(gcClient, "nonexistent")
				Expect(err).ToNot(HaveOccurred())
				Expect(found).To(BeNil())
			})
		})
	})
})
