/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2022, 2024-2026 Wind River Systems, Inc. */
package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/datanetworks"
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

const dataNetworkResponse = `{
	"uuid": "test-dn-uuid",
	"name": "group0-data0",
	"description": "",
	"network_type": "flat",
	"mtu": 1500
}`

const dataNetworkListResponse = `{
	"datanetworks": [` + dataNetworkResponse + `]
}`

func newDataNetworkFixtureServer() (*httptest.Server, *gophercloud.ServiceClient) {
	mux := http.NewServeMux()
	mux.HandleFunc("/datanetworks", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodGet:
			_, _ = fmt.Fprint(w, dataNetworkListResponse)
		case http.MethodPost:
			var body map[string]interface{}
			_ = json.NewDecoder(r.Body).Decode(&body)
			name, _ := body["name"].(string)
			ntype, _ := body["network_type"].(string)
			resp := fmt.Sprintf(`{"uuid":"new-dn-uuid","name":"%s","network_type":"%s","mtu":1500,"description":""}`, name, ntype)
			w.WriteHeader(http.StatusOK)
			_, _ = fmt.Fprint(w, resp)
		}
	})
	mux.HandleFunc("/datanetworks/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodGet:
			_, _ = fmt.Fprint(w, dataNetworkResponse)
		case http.MethodPatch:
			_, _ = fmt.Fprint(w, dataNetworkResponse)
		case http.MethodDelete:
			id := strings.TrimPrefix(r.URL.Path, "/datanetworks/")
			if id == "fail-delete" {
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = fmt.Fprint(w, `{"error": "internal error"}`)
			} else {
				w.WriteHeader(http.StatusNoContent)
			}
		}
	})

	server := httptest.NewServer(mux)
	sc := &gophercloud.ServiceClient{
		ProviderClient: &gophercloud.ProviderClient{TokenID: "test-token"},
		Endpoint:       server.URL + "/",
	}
	return server, sc
}

func newDataNetworkReconciler() *DataNetworkReconciler {
	dm := &cloudManager.Dummymanager{}
	logger := log.Log.WithName("test")
	return &DataNetworkReconciler{
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

func newGopherDataNetwork(name, ntype string) *datanetworks.DataNetwork {
	return &datanetworks.DataNetwork{
		ID:   "test-dn-uuid",
		Name: name,
		Type: ntype,
		MTU:  1500,
	}
}

var _ = Describe("Datanetwork controller", func() {
	const (
		timeout  = time.Second * 30
		interval = time.Millisecond * 500
	)

	Context("with DataNetwork data", func() {
		It("should be created successfully", func() {
			ctx := context.Background()
			key := types.NamespacedName{
				Name:      "foo",
				Namespace: "default",
			}
			mtu := 1500
			description := "This is a sample description"

			created := &starlingxv1.DataNetwork{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: "default",
				},
				Spec: starlingxv1.DataNetworkSpec{
					Type:        "flat",
					Description: &description,
					MTU:         &mtu,
				}}
			Expect(k8sClient.Create(ctx, created)).To(Succeed())

			fetched := &starlingxv1.DataNetwork{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, key, fetched)
				if err != nil {
					return false
				}
				_, found := comm.ListIntersect(fetched.ObjectMeta.Finalizers, []string{DataNetworkFinalizerName})
				return found
			}, timeout, interval).Should(BeTrue())
		})
	})

	Describe("FindExistingResource", func() {
		var (
			server     *httptest.Server
			gcClient   *gophercloud.ServiceClient
			reconciler *DataNetworkReconciler
		)

		BeforeEach(func() {
			server, gcClient = newDataNetworkFixtureServer()
			reconciler = newDataNetworkReconciler()
		})

		AfterEach(func() {
			server.Close()
		})

		Context("when the resource has a status ID", func() {
			It("should fetch by ID", func() {
				id := "test-dn-uuid"
				instance := &starlingxv1.DataNetwork{
					ObjectMeta: metav1.ObjectMeta{Name: "group0-data0", Namespace: "default"},
					Spec:       starlingxv1.DataNetworkSpec{Type: "flat"},
				}
				instance.Status.ID = &id
				network, err := reconciler.FindExistingResource(gcClient, instance)
				Expect(err).ToNot(HaveOccurred())
				Expect(network).ToNot(BeNil())
				Expect(network.ID).To(Equal("test-dn-uuid"))
			})
		})

		Context("when the resource has no status ID", func() {
			It("should find by name and type from list", func() {
				instance := &starlingxv1.DataNetwork{
					ObjectMeta: metav1.ObjectMeta{Name: "group0-data0", Namespace: "default"},
					Spec:       starlingxv1.DataNetworkSpec{Type: "flat"},
				}
				network, err := reconciler.FindExistingResource(gcClient, instance)
				Expect(err).ToNot(HaveOccurred())
				Expect(network).ToNot(BeNil())
				Expect(network.Name).To(Equal("group0-data0"))
			})
		})

		Context("when the resource does not exist in the list", func() {
			It("should return nil", func() {
				instance := &starlingxv1.DataNetwork{
					ObjectMeta: metav1.ObjectMeta{Name: "nonexistent", Namespace: "default"},
					Spec:       starlingxv1.DataNetworkSpec{Type: "flat"},
				}
				network, err := reconciler.FindExistingResource(gcClient, instance)
				Expect(err).ToNot(HaveOccurred())
				Expect(network).To(BeNil())
			})
		})
	})

	Describe("ReconcileNew", func() {
		var (
			server     *httptest.Server
			gcClient   *gophercloud.ServiceClient
			reconciler *DataNetworkReconciler
		)

		BeforeEach(func() {
			server, gcClient = newDataNetworkFixtureServer()
			reconciler = newDataNetworkReconciler()
		})

		AfterEach(func() {
			server.Close()
		})

		Context("when creating a flat data network", func() {
			It("should succeed", func() {
				instance := &starlingxv1.DataNetwork{
					ObjectMeta: metav1.ObjectMeta{Name: "dn-new-flat", Namespace: "default"},
					Spec:       starlingxv1.DataNetworkSpec{Type: "flat"},
				}
				network, err := reconciler.ReconcileNew(gcClient, instance)
				Expect(err).ToNot(HaveOccurred())
				Expect(network).ToNot(BeNil())
				Expect(network.ID).To(Equal("new-dn-uuid"))
			})
		})

		Context("when already reconciled and StopAfterInSync", func() {
			It("should return ChangeAfterInSync error", func() {
				instance := &starlingxv1.DataNetwork{
					ObjectMeta: metav1.ObjectMeta{Name: "dn-new-stop", Namespace: "default"},
					Spec:       starlingxv1.DataNetworkSpec{Type: "flat"},
				}
				instance.Status.Reconciled = true

				_, err := reconciler.ReconcileNew(gcClient, instance)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("provisioning ignored"))
			})
		})
	})

	Describe("ReconcileUpdated", func() {
		var (
			server     *httptest.Server
			gcClient   *gophercloud.ServiceClient
			reconciler *DataNetworkReconciler
		)

		BeforeEach(func() {
			server, gcClient = newDataNetworkFixtureServer()
			reconciler = newDataNetworkReconciler()
		})

		AfterEach(func() {
			server.Close()
		})

		Context("when no update is required", func() {
			It("should return nil", func() {
				instance := &starlingxv1.DataNetwork{
					ObjectMeta: metav1.ObjectMeta{Name: "dn-update-noop", Namespace: "default"},
					Spec:       starlingxv1.DataNetworkSpec{Type: "flat"},
				}
				Expect(k8sClient.Create(ctx, instance)).To(Succeed())

				Eventually(func() bool {
					_ = k8sClient.Get(ctx, client.ObjectKeyFromObject(instance), instance)
					return len(instance.Finalizers) > 0
				}, timeout, interval).Should(BeTrue())

				current := &starlingxv1.DataNetwork{}
				Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(instance), current)).To(Succeed())

				network := newGopherDataNetwork("dn-update-noop", "flat")
				err := reconciler.ReconcileUpdated(gcClient, current, network)
				Expect(err).ToNot(HaveOccurred())
			})
		})
	})

	Describe("ReconciledDeleted", func() {
		var (
			server     *httptest.Server
			gcClient   *gophercloud.ServiceClient
			reconciler *DataNetworkReconciler
		)

		BeforeEach(func() {
			server, gcClient = newDataNetworkFixtureServer()
			reconciler = newDataNetworkReconciler()
		})

		AfterEach(func() {
			server.Close()
		})

		Context("when the resource has a finalizer and network exists", func() {
			It("should delete the network and remove the finalizer", func() {
				instance := &starlingxv1.DataNetwork{
					ObjectMeta: metav1.ObjectMeta{
						Name:       "dn-delete",
						Namespace:  "default",
						Finalizers: []string{DataNetworkFinalizerName},
					},
					Spec: starlingxv1.DataNetworkSpec{Type: "flat"},
				}
				Expect(k8sClient.Create(ctx, instance)).To(Succeed())
				Expect(k8sClient.Delete(ctx, instance)).To(Succeed())

				Eventually(func() bool {
					err := k8sClient.Get(ctx, client.ObjectKeyFromObject(instance), instance)
					return err == nil && !instance.DeletionTimestamp.IsZero()
				}, timeout, interval).Should(BeTrue())

				network := newGopherDataNetwork("dn-delete", "flat")
				err := reconciler.ReconciledDeleted(gcClient, instance, network)
				Expect(err).ToNot(HaveOccurred())
				Expect(instance.Finalizers).ToNot(ContainElement(DataNetworkFinalizerName))
			})
		})

		Context("when the network is nil", func() {
			It("should just remove the finalizer", func() {
				instance := &starlingxv1.DataNetwork{
					ObjectMeta: metav1.ObjectMeta{
						Name:       "dn-delete-nil",
						Namespace:  "default",
						Finalizers: []string{DataNetworkFinalizerName},
					},
					Spec: starlingxv1.DataNetworkSpec{Type: "flat"},
				}
				Expect(k8sClient.Create(ctx, instance)).To(Succeed())
				Expect(k8sClient.Delete(ctx, instance)).To(Succeed())

				Eventually(func() bool {
					err := k8sClient.Get(ctx, client.ObjectKeyFromObject(instance), instance)
					return err == nil && !instance.DeletionTimestamp.IsZero()
				}, timeout, interval).Should(BeTrue())

				err := reconciler.ReconciledDeleted(gcClient, instance, nil)
				Expect(err).ToNot(HaveOccurred())
				Expect(instance.Finalizers).ToNot(ContainElement(DataNetworkFinalizerName))
			})
		})
	})

	Describe("ReconcileResource", func() {
		var (
			server     *httptest.Server
			gcClient   *gophercloud.ServiceClient
			reconciler *DataNetworkReconciler
		)

		BeforeEach(func() {
			server, gcClient = newDataNetworkFixtureServer()
			reconciler = newDataNetworkReconciler()
		})

		AfterEach(func() {
			server.Close()
		})

		Context("when the resource exists on the system", func() {
			It("should reconcile successfully", func() {
				instance := &starlingxv1.DataNetwork{
					ObjectMeta: metav1.ObjectMeta{Name: "dn-reconcile-exist", Namespace: "default"},
					Spec:       starlingxv1.DataNetworkSpec{Type: "flat"},
				}
				Expect(k8sClient.Create(ctx, instance)).To(Succeed())

				Eventually(func() bool {
					_ = k8sClient.Get(ctx, client.ObjectKeyFromObject(instance), instance)
					return len(instance.Finalizers) > 0
				}, timeout, interval).Should(BeTrue())

				// FindExistingResource will list and not find "dn-reconcile-exist"
				// in the fixture, so it will call ReconcileNew instead.
				// To test the "exists" path, set a status ID so it fetches by UUID.
				id := "test-dn-uuid"
				instance.Status.ID = &id
				err := reconciler.ReconcileResource(gcClient, instance)
				Expect(err).ToNot(HaveOccurred())
			})
		})

		Context("when the resource does not exist on the system", func() {
			It("should create a new data network", func() {
				instance := &starlingxv1.DataNetwork{
					ObjectMeta: metav1.ObjectMeta{Name: "dn-reconcile-new", Namespace: "default"},
					Spec:       starlingxv1.DataNetworkSpec{Type: "flat"},
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

	Describe("removeDataNetworkFinalizer", func() {
		Context("when the resource has a finalizer", func() {
			It("should remove it", func() {
				instance := &starlingxv1.DataNetwork{
					ObjectMeta: metav1.ObjectMeta{
						Name:       "dn-rm-finalizer",
						Namespace:  "default",
						Finalizers: []string{DataNetworkFinalizerName},
					},
					Spec: starlingxv1.DataNetworkSpec{Type: "flat"},
				}
				Expect(k8sClient.Create(ctx, instance)).To(Succeed())

				reconciler := newDataNetworkReconciler()
				reconciler.removeDataNetworkFinalizer(instance)
				Expect(instance.Finalizers).ToNot(ContainElement(DataNetworkFinalizerName))
			})
		})
	})
})
