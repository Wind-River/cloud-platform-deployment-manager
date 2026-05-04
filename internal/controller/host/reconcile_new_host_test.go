/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2026 Wind River Systems, Inc. */
package host

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/hosts"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/log"

	starlingxv1 "github.com/wind-river/cloud-platform-deployment-manager/api/v1"
	common "github.com/wind-river/cloud-platform-deployment-manager/internal/controller/common"
	cloudManager "github.com/wind-river/cloud-platform-deployment-manager/internal/controller/manager"
)

func newTestHostReconciler(hostList []hosts.Host) *HostReconciler {
	dm := &cloudManager.Dummymanager{}
	logger := log.Log.WithName("test-reconcile-new-host")
	return &HostReconciler{
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
		hosts: hostList,
	}
}

func newTestServiceClient(handler http.Handler) (*httptest.Server, *gophercloud.ServiceClient) {
	server := httptest.NewServer(handler)
	sc := &gophercloud.ServiceClient{
		ProviderClient: &gophercloud.ProviderClient{},
		Endpoint:       server.URL + "/",
	}
	return server, sc
}

func newHostInstance(name, namespace string, reconciled bool, annotations map[string]string) *starlingxv1.Host {
	bootMAC := "01:02:03:04:05:06"
	return &starlingxv1.Host{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   namespace,
			Annotations: annotations,
		},
		Spec: starlingxv1.HostSpec{
			Match: &starlingxv1.MatchInfo{
				BootMAC: &bootMAC,
			},
		},
		Status: starlingxv1.HostStatus{
			Reconciled: reconciled,
		},
	}
}

var _ = Describe("ReconcileNewHost", func() {

	Context("when host exists in inventory with a hostname", func() {
		It("should return the existing host without changes", func() {
			existingHost := hosts.Host{
				ID:       "existing-id",
				Hostname: "controller-0",
				BootMAC:  "01:02:03:04:05:06",
			}
			r := newTestHostReconciler([]hosts.Host{existingHost})

			personality := "controller"
			mode := starlingxv1.ProvioningModeStatic
			profile := &starlingxv1.HostProfileSpec{
				ProfileBaseAttributes: starlingxv1.ProfileBaseAttributes{
					Personality:      &personality,
					ProvisioningMode: &mode,
				},
			}
			instance := newHostInstance("controller-0", "default", false, nil)

			server, client := newTestServiceClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, "unexpected call", http.StatusInternalServerError)
			}))
			defer server.Close()

			host, err := r.ReconcileNewHost(client, instance, profile)
			Expect(err).ToNot(HaveOccurred())
			Expect(host).ToNot(BeNil())
			Expect(host.Hostname).To(Equal("controller-0"))
		})
	})

	Context("when host is not found and provisioning mode is dynamic", func() {
		It("should start a dynamic host monitor and return nil host", func() {
			r := newTestHostReconciler([]hosts.Host{})
			dm := r.CloudManager.(*cloudManager.Dummymanager)

			mode := starlingxv1.ProvioningModeDynamic
			personality := "compute"
			profile := &starlingxv1.HostProfileSpec{
				ProfileBaseAttributes: starlingxv1.ProfileBaseAttributes{
					Personality:      &personality,
					ProvisioningMode: &mode,
				},
			}
			instance := newHostInstance("compute-0", "default", false, nil)

			server, client := newTestServiceClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, "unexpected call", http.StatusInternalServerError)
			}))
			defer server.Close()

			host, err := r.ReconcileNewHost(client, instance, profile)
			Expect(err).ToNot(HaveOccurred())
			Expect(host).To(BeNil())
			Expect(dm.MonitorStarted).To(BeTrue())
			Expect(dm.MonitorMessage).To(Equal("waiting for dynamic host to appear in inventory"))
			dm.CancelMonitor(instance)
			Expect(dm.MonitorStarted).To(BeFalse())
		})
	})

	Context("when host is not found, static mode, provisioning not allowed", func() {
		It("should start a provisioning allowed monitor", func() {
			// No controller unlocked/enabled → provisioning not allowed
			r := newTestHostReconciler([]hosts.Host{})
			dm := r.CloudManager.(*cloudManager.Dummymanager)

			mode := starlingxv1.ProvioningModeStatic
			personality := "compute"
			profile := &starlingxv1.HostProfileSpec{
				ProfileBaseAttributes: starlingxv1.ProfileBaseAttributes{
					Personality:      &personality,
					ProvisioningMode: &mode,
				},
			}
			instance := newHostInstance("compute-0", "default", false, nil)

			server, client := newTestServiceClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, "unexpected call", http.StatusInternalServerError)
			}))
			defer server.Close()

			host, err := r.ReconcileNewHost(client, instance, profile)
			Expect(err).ToNot(HaveOccurred())
			Expect(host).To(BeNil())
			Expect(dm.MonitorStarted).To(BeTrue())
			Expect(dm.MonitorMessage).To(Equal("waiting for system to allow creating static hosts"))
			dm.CancelMonitor(instance)
			Expect(dm.MonitorStarted).To(BeFalse())
		})
	})

	Context("when host is not found, static mode, provisioning allowed, already reconciled, StopAfterInSync, no annotation", func() {
		It("should return ChangeAfterInSync error", func() {
			controllerHost := hosts.Host{
				Hostname:            hosts.Controller0,
				AdministrativeState: hosts.AdminUnlocked,
				OperationalStatus:   hosts.OperEnabled,
			}
			r := newTestHostReconciler([]hosts.Host{controllerHost})

			mode := starlingxv1.ProvioningModeStatic
			personality := "compute"
			profile := &starlingxv1.HostProfileSpec{
				ProfileBaseAttributes: starlingxv1.ProfileBaseAttributes{
					Personality:      &personality,
					ProvisioningMode: &mode,
				},
			}
			instance := newHostInstance("compute-0", "default", true, nil)

			server, client := newTestServiceClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, "unexpected call", http.StatusInternalServerError)
			}))
			defer server.Close()

			host, err := r.ReconcileNewHost(client, instance, profile)
			Expect(err).To(HaveOccurred())
			Expect(host).To(BeNil())
			_, ok := err.(common.ChangeAfterReconciled)
			Expect(ok).To(BeTrue())
		})
	})

	Context("when host is not found, static mode, provisioning allowed, already reconciled, has ReconcileAfterInSync annotation", func() {
		It("should create the host via API", func() {
			controllerHost := hosts.Host{
				Hostname:            hosts.Controller0,
				AdministrativeState: hosts.AdminUnlocked,
				OperationalStatus:   hosts.OperEnabled,
			}
			r := newTestHostReconciler([]hosts.Host{controllerHost})

			mode := starlingxv1.ProvioningModeStatic
			personality := "compute"
			subfunctions := []starlingxv1.SubFunction{"compute"}
			profile := &starlingxv1.HostProfileSpec{
				ProfileBaseAttributes: starlingxv1.ProfileBaseAttributes{
					Personality:      &personality,
					SubFunctions:     subfunctions,
					ProvisioningMode: &mode,
				},
			}
			annotations := map[string]string{
				cloudManager.ReconcileAfterInSync: "true",
			}
			instance := newHostInstance("compute-0", "default", true, annotations)

			createdHost := hosts.Host{
				ID:       "new-host-id",
				Hostname: "compute-0",
			}
			server, client := newTestServiceClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if r.Method == http.MethodPost {
					resp, _ := json.Marshal(createdHost)
					_, _ = fmt.Fprint(w, string(resp))
				}
			}))
			defer server.Close()

			host, err := r.ReconcileNewHost(client, instance, profile)
			Expect(err).ToNot(HaveOccurred())
			Expect(host).ToNot(BeNil())
			Expect(host.ID).To(Equal("new-host-id"))
		})
	})

	Context("when host is not found, static mode, provisioning allowed, not reconciled", func() {
		It("should create the host via API", func() {
			controllerHost := hosts.Host{
				Hostname:            hosts.Controller0,
				AdministrativeState: hosts.AdminUnlocked,
				OperationalStatus:   hosts.OperEnabled,
			}
			r := newTestHostReconciler([]hosts.Host{controllerHost})

			mode := starlingxv1.ProvioningModeStatic
			personality := "compute"
			subfunctions := []starlingxv1.SubFunction{"compute"}
			profile := &starlingxv1.HostProfileSpec{
				ProfileBaseAttributes: starlingxv1.ProfileBaseAttributes{
					Personality:      &personality,
					SubFunctions:     subfunctions,
					ProvisioningMode: &mode,
				},
			}
			instance := newHostInstance("compute-0", "default", false, nil)

			createdHost := hosts.Host{
				ID:       "new-host-id",
				Hostname: "compute-0",
			}
			server, client := newTestServiceClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if r.Method == http.MethodPost {
					resp, _ := json.Marshal(createdHost)
					_, _ = fmt.Fprint(w, string(resp))
				}
			}))
			defer server.Close()

			host, err := r.ReconcileNewHost(client, instance, profile)
			Expect(err).ToNot(HaveOccurred())
			Expect(host).ToNot(BeNil())
			Expect(host.ID).To(Equal("new-host-id"))
		})
	})

	Context("when host is not found, static mode, provisioning allowed, with BMC and PowerOn", func() {
		It("should create the host and send reinstall action", func() {
			controllerHost := hosts.Host{
				Hostname:            hosts.Controller0,
				AdministrativeState: hosts.AdminUnlocked,
				OperationalStatus:   hosts.OperEnabled,
			}
			r := newTestHostReconciler([]hosts.Host{controllerHost})

			mode := starlingxv1.ProvioningModeStatic
			personality := "compute"
			subfunctions := []starlingxv1.SubFunction{"compute"}
			bmType := "bmc"
			bmAddr := "192.168.9.9"
			powerOn := true
			profile := &starlingxv1.HostProfileSpec{
				ProfileBaseAttributes: starlingxv1.ProfileBaseAttributes{
					Personality:      &personality,
					SubFunctions:     subfunctions,
					ProvisioningMode: &mode,
					PowerOn:          &powerOn,
				},
				BoardManagement: &starlingxv1.BMInfo{
					Type:    &bmType,
					Address: &bmAddr,
				},
			}
			instance := newHostInstance("compute-0", "default", false, nil)

			createdHost := hosts.Host{
				ID:       "new-host-id",
				Hostname: "compute-0",
			}
			callCount := 0
			server, client := newTestServiceClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				callCount++
				// First POST = create, second PATCH = update (reinstall action)
				resp, _ := json.Marshal(createdHost)
				_, _ = fmt.Fprint(w, string(resp))
			}))
			defer server.Close()

			host, err := r.ReconcileNewHost(client, instance, profile)
			Expect(err).ToNot(HaveOccurred())
			Expect(host).ToNot(BeNil())
			Expect(callCount).To(BeNumerically(">=", 2))
		})
	})

	Context("when host is not found, static mode, API create fails", func() {
		It("should return an error", func() {
			controllerHost := hosts.Host{
				Hostname:            hosts.Controller0,
				AdministrativeState: hosts.AdminUnlocked,
				OperationalStatus:   hosts.OperEnabled,
			}
			r := newTestHostReconciler([]hosts.Host{controllerHost})

			mode := starlingxv1.ProvioningModeStatic
			personality := "compute"
			subfunctions := []starlingxv1.SubFunction{"compute"}
			profile := &starlingxv1.HostProfileSpec{
				ProfileBaseAttributes: starlingxv1.ProfileBaseAttributes{
					Personality:      &personality,
					SubFunctions:     subfunctions,
					ProvisioningMode: &mode,
				},
			}
			instance := newHostInstance("compute-0", "default", false, nil)

			server, client := newTestServiceClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, "internal error", http.StatusInternalServerError)
			}))
			defer server.Close()

			host, err := r.ReconcileNewHost(client, instance, profile)
			Expect(err).To(HaveOccurred())
			Expect(host).To(BeNil())
		})
	})

	Context("when host is not found, static mode, provisioning allowed, BMC+PowerOn, reinstall update fails", func() {
		It("should return an error after creating the host", func() {
			controllerHost := hosts.Host{
				Hostname:            hosts.Controller0,
				AdministrativeState: hosts.AdminUnlocked,
				OperationalStatus:   hosts.OperEnabled,
			}
			r := newTestHostReconciler([]hosts.Host{controllerHost})

			mode := starlingxv1.ProvioningModeStatic
			personality := "compute"
			subfunctions := []starlingxv1.SubFunction{"compute"}
			bmType := "bmc"
			bmAddr := "192.168.9.9"
			powerOn := true
			profile := &starlingxv1.HostProfileSpec{
				ProfileBaseAttributes: starlingxv1.ProfileBaseAttributes{
					Personality:      &personality,
					SubFunctions:     subfunctions,
					ProvisioningMode: &mode,
					PowerOn:          &powerOn,
				},
				BoardManagement: &starlingxv1.BMInfo{
					Type:    &bmType,
					Address: &bmAddr,
				},
			}
			instance := newHostInstance("compute-0", "default", false, nil)

			createdHost := hosts.Host{
				ID:       "new-host-id",
				Hostname: "compute-0",
			}
			callCount := 0
			server, client := newTestServiceClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				callCount++
				w.Header().Set("Content-Type", "application/json")
				if callCount == 1 {
					// Create succeeds
					resp, _ := json.Marshal(createdHost)
					_, _ = fmt.Fprint(w, string(resp))
				} else {
					// Reinstall update fails
					http.Error(w, "power-on failed", http.StatusInternalServerError)
				}
			}))
			defer server.Close()

			host, err := r.ReconcileNewHost(client, instance, profile)
			Expect(err).To(HaveOccurred())
			Expect(host).To(BeNil())
			Expect(err.Error()).To(ContainSubstring("failed to power-on host"))
		})
	})

	Context("when host is not found, static mode, buildInitialHostOpts fails", func() {
		It("should return an error from buildInitialHostOpts", func() {
			controllerHost := hosts.Host{
				Hostname:            hosts.Controller0,
				AdministrativeState: hosts.AdminUnlocked,
				OperationalStatus:   hosts.OperEnabled,
			}
			r := newTestHostReconciler([]hosts.Host{controllerHost})

			mode := starlingxv1.ProvioningModeStatic
			// nil Personality triggers an error in UpdateRequired/buildInitialHostOpts
			profile := &starlingxv1.HostProfileSpec{
				ProfileBaseAttributes: starlingxv1.ProfileBaseAttributes{
					ProvisioningMode: &mode,
				},
			}
			instance := newHostInstance("compute-0", "default", false, nil)

			server, client := newTestServiceClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, "unexpected call", http.StatusInternalServerError)
			}))
			defer server.Close()

			host, err := r.ReconcileNewHost(client, instance, profile)
			Expect(err).To(HaveOccurred())
			Expect(host).To(BeNil())
		})
	})

	Context("when host exists with empty hostname, provisioning allowed, ReconcileAttributes fails", func() {
		It("should return the host and the error", func() {
			existingHost := hosts.Host{
				ID:                  "found-host-id",
				Hostname:            "",
				BootMAC:             "01:02:03:04:05:06",
				AdministrativeState: hosts.AdminLocked,
				OperationalStatus:   hosts.OperDisabled,
			}
			controllerHost := hosts.Host{
				Hostname:            hosts.Controller0,
				AdministrativeState: hosts.AdminUnlocked,
				OperationalStatus:   hosts.OperEnabled,
			}
			r := newTestHostReconciler([]hosts.Host{existingHost, controllerHost})

			mode := starlingxv1.ProvioningModeStatic
			personality := "compute"
			subfunctions := []starlingxv1.SubFunction{"compute"}
			profile := &starlingxv1.HostProfileSpec{
				ProfileBaseAttributes: starlingxv1.ProfileBaseAttributes{
					Personality:      &personality,
					SubFunctions:     subfunctions,
					ProvisioningMode: &mode,
				},
			}
			instance := newHostInstance("compute-0", "default", false, nil)

			server, client := newTestServiceClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, "update failed", http.StatusInternalServerError)
			}))
			defer server.Close()

			host, err := r.ReconcileNewHost(client, instance, profile)
			Expect(err).To(HaveOccurred())
			Expect(host).ToNot(BeNil())
			Expect(host.ID).To(Equal("found-host-id"))
		})
	})

	Context("when host exists with empty hostname, provisioning allowed, BMC+PowerOn, reinstall succeeds", func() {
		It("should reconcile attributes and send reinstall action", func() {
			existingHost := hosts.Host{
				ID:                  "found-host-id",
				Hostname:            "",
				BootMAC:             "01:02:03:04:05:06",
				AdministrativeState: hosts.AdminLocked,
				OperationalStatus:   hosts.OperDisabled,
			}
			controllerHost := hosts.Host{
				Hostname:            hosts.Controller0,
				AdministrativeState: hosts.AdminUnlocked,
				OperationalStatus:   hosts.OperEnabled,
			}
			r := newTestHostReconciler([]hosts.Host{existingHost, controllerHost})

			mode := starlingxv1.ProvioningModeStatic
			personality := "compute"
			subfunctions := []starlingxv1.SubFunction{"compute"}
			bmType := "bmc"
			bmAddr := "192.168.9.9"
			powerOn := true
			profile := &starlingxv1.HostProfileSpec{
				ProfileBaseAttributes: starlingxv1.ProfileBaseAttributes{
					Personality:      &personality,
					SubFunctions:     subfunctions,
					ProvisioningMode: &mode,
					PowerOn:          &powerOn,
				},
				BoardManagement: &starlingxv1.BMInfo{
					Type:    &bmType,
					Address: &bmAddr,
				},
			}
			instance := newHostInstance("compute-0", "default", false, nil)

			updatedHost := hosts.Host{
				ID:       "found-host-id",
				Hostname: "compute-0",
			}
			callCount := 0
			server, client := newTestServiceClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				callCount++
				resp, _ := json.Marshal(updatedHost)
				_, _ = fmt.Fprint(w, string(resp))
			}))
			defer server.Close()

			host, err := r.ReconcileNewHost(client, instance, profile)
			Expect(err).ToNot(HaveOccurred())
			Expect(host).ToNot(BeNil())
			// PATCH for ReconcileAttributes + PATCH for reinstall
			Expect(callCount).To(BeNumerically(">=", 2))
		})
	})

	Context("when host exists with empty hostname, provisioning allowed, BMC+PowerOn, reinstall fails", func() {
		It("should return an error after reconciling attributes", func() {
			existingHost := hosts.Host{
				ID:                  "found-host-id",
				Hostname:            "",
				BootMAC:             "01:02:03:04:05:06",
				AdministrativeState: hosts.AdminLocked,
				OperationalStatus:   hosts.OperDisabled,
			}
			controllerHost := hosts.Host{
				Hostname:            hosts.Controller0,
				AdministrativeState: hosts.AdminUnlocked,
				OperationalStatus:   hosts.OperEnabled,
			}
			r := newTestHostReconciler([]hosts.Host{existingHost, controllerHost})

			mode := starlingxv1.ProvioningModeStatic
			personality := "compute"
			subfunctions := []starlingxv1.SubFunction{"compute"}
			bmType := "bmc"
			bmAddr := "192.168.9.9"
			powerOn := true
			profile := &starlingxv1.HostProfileSpec{
				ProfileBaseAttributes: starlingxv1.ProfileBaseAttributes{
					Personality:      &personality,
					SubFunctions:     subfunctions,
					ProvisioningMode: &mode,
					PowerOn:          &powerOn,
				},
				BoardManagement: &starlingxv1.BMInfo{
					Type:    &bmType,
					Address: &bmAddr,
				},
			}
			instance := newHostInstance("compute-0", "default", false, nil)

			updatedHost := hosts.Host{
				ID:       "found-host-id",
				Hostname: "compute-0",
			}
			callCount := 0
			server, client := newTestServiceClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				callCount++
				w.Header().Set("Content-Type", "application/json")
				if callCount == 1 {
					// ReconcileAttributes succeeds
					resp, _ := json.Marshal(updatedHost)
					_, _ = fmt.Fprint(w, string(resp))
				} else {
					// Reinstall fails
					http.Error(w, "power-on failed", http.StatusInternalServerError)
				}
			}))
			defer server.Close()

			host, err := r.ReconcileNewHost(client, instance, profile)
			Expect(err).To(HaveOccurred())
			Expect(host).To(BeNil())
			Expect(err.Error()).To(ContainSubstring("failed to power-on host"))
		})
	})

	Context("when host exists with empty hostname, provisioning allowed", func() {
		It("should call ReconcileAttributes to set initial attributes", func() {
			existingHost := hosts.Host{
				ID:                  "found-host-id",
				Hostname:            "",
				BootMAC:             "01:02:03:04:05:06",
				AdministrativeState: hosts.AdminLocked,
				OperationalStatus:   hosts.OperDisabled,
			}
			controllerHost := hosts.Host{
				Hostname:            hosts.Controller0,
				AdministrativeState: hosts.AdminUnlocked,
				OperationalStatus:   hosts.OperEnabled,
			}
			r := newTestHostReconciler([]hosts.Host{existingHost, controllerHost})

			mode := starlingxv1.ProvioningModeStatic
			personality := "compute"
			subfunctions := []starlingxv1.SubFunction{"compute"}
			profile := &starlingxv1.HostProfileSpec{
				ProfileBaseAttributes: starlingxv1.ProfileBaseAttributes{
					Personality:      &personality,
					SubFunctions:     subfunctions,
					ProvisioningMode: &mode,
				},
			}
			instance := newHostInstance("compute-0", "default", false, nil)

			updatedHost := hosts.Host{
				ID:       "found-host-id",
				Hostname: "compute-0",
			}
			server, client := newTestServiceClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				resp, _ := json.Marshal(updatedHost)
				_, _ = fmt.Fprint(w, string(resp))
			}))
			defer server.Close()

			host, err := r.ReconcileNewHost(client, instance, profile)
			Expect(err).ToNot(HaveOccurred())
			Expect(host).ToNot(BeNil())
			Expect(host.ID).To(Equal("found-host-id"))
		})
	})

	Context("when host exists with empty hostname, provisioning not allowed", func() {
		It("should start a provisioning allowed monitor", func() {
			existingHost := hosts.Host{
				ID:       "found-host-id",
				Hostname: "",
				BootMAC:  "01:02:03:04:05:06",
			}
			// No controller unlocked/enabled
			r := newTestHostReconciler([]hosts.Host{existingHost})
			dm := r.CloudManager.(*cloudManager.Dummymanager)

			mode := starlingxv1.ProvioningModeStatic
			personality := "compute"
			profile := &starlingxv1.HostProfileSpec{
				ProfileBaseAttributes: starlingxv1.ProfileBaseAttributes{
					Personality:      &personality,
					ProvisioningMode: &mode,
				},
			}
			instance := newHostInstance("compute-0", "default", false, nil)

			server, client := newTestServiceClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, "unexpected call", http.StatusInternalServerError)
			}))
			defer server.Close()

			host, err := r.ReconcileNewHost(client, instance, profile)
			Expect(err).ToNot(HaveOccurred())
			Expect(host).ToNot(BeNil())
			Expect(host.ID).To(Equal("found-host-id"))
			Expect(dm.MonitorStarted).To(BeTrue())
			Expect(dm.MonitorMessage).To(Equal("waiting for system to allow host provisioning"))
			dm.CancelMonitor(instance)
			Expect(dm.MonitorStarted).To(BeFalse())
		})
	})
})

var _ = Describe("reinstallAllowed", func() {
	var r *HostReconciler

	BeforeEach(func() {
		r = newTestHostReconciler([]hosts.Host{})
	})

	It("should return false when BoardManagement is nil", func() {
		powerOn := true
		profile := &starlingxv1.HostProfileSpec{
			ProfileBaseAttributes: starlingxv1.ProfileBaseAttributes{
				PowerOn: &powerOn,
			},
		}
		host := &hosts.Host{ID: "test-id"}
		Expect(r.reinstallAllowed(host, profile)).To(BeFalse())
	})

	It("should return false when PowerOn is nil", func() {
		bmType := "bmc"
		profile := &starlingxv1.HostProfileSpec{
			BoardManagement: &starlingxv1.BMInfo{Type: &bmType},
		}
		host := &hosts.Host{ID: "test-id"}
		Expect(r.reinstallAllowed(host, profile)).To(BeFalse())
	})

	It("should return false when PowerOn is false", func() {
		bmType := "bmc"
		powerOn := false
		profile := &starlingxv1.HostProfileSpec{
			ProfileBaseAttributes: starlingxv1.ProfileBaseAttributes{
				PowerOn: &powerOn,
			},
			BoardManagement: &starlingxv1.BMInfo{Type: &bmType},
		}
		host := &hosts.Host{ID: "test-id"}
		Expect(r.reinstallAllowed(host, profile)).To(BeFalse())
	})

	It("should return false when host is nil", func() {
		bmType := "bmc"
		powerOn := true
		profile := &starlingxv1.HostProfileSpec{
			ProfileBaseAttributes: starlingxv1.ProfileBaseAttributes{
				PowerOn: &powerOn,
			},
			BoardManagement: &starlingxv1.BMInfo{Type: &bmType},
		}
		Expect(r.reinstallAllowed(nil, profile)).To(BeFalse())
	})

	It("should return true when InventoryState is nil", func() {
		bmType := "bmc"
		powerOn := true
		profile := &starlingxv1.HostProfileSpec{
			ProfileBaseAttributes: starlingxv1.ProfileBaseAttributes{
				PowerOn: &powerOn,
			},
			BoardManagement: &starlingxv1.BMInfo{Type: &bmType},
		}
		host := &hosts.Host{ID: "test-id", InventoryState: nil}
		Expect(r.reinstallAllowed(host, profile)).To(BeTrue())
	})

	It("should return true when InventoryState is empty string", func() {
		bmType := "bmc"
		powerOn := true
		profile := &starlingxv1.HostProfileSpec{
			ProfileBaseAttributes: starlingxv1.ProfileBaseAttributes{
				PowerOn: &powerOn,
			},
			BoardManagement: &starlingxv1.BMInfo{Type: &bmType},
		}
		empty := ""
		host := &hosts.Host{ID: "test-id", InventoryState: &empty}
		Expect(r.reinstallAllowed(host, profile)).To(BeTrue())
	})

	It("should return false when host is already inventoried", func() {
		bmType := "bmc"
		powerOn := true
		profile := &starlingxv1.HostProfileSpec{
			ProfileBaseAttributes: starlingxv1.ProfileBaseAttributes{
				PowerOn: &powerOn,
			},
			BoardManagement: &starlingxv1.BMInfo{Type: &bmType},
		}
		inventoried := "inventoried"
		host := &hosts.Host{ID: "test-id", InventoryState: &inventoried}
		Expect(r.reinstallAllowed(host, profile)).To(BeFalse())
	})
})
