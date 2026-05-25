/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2026 Wind River Systems, Inc. */

package host

import (
	"encoding/json"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/routes"
	starlingxv1 "github.com/wind-river/cloud-platform-deployment-manager/api/v1"
	v1info "github.com/wind-river/cloud-platform-deployment-manager/platform"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var _ = Describe("BuildHostDefaults", func() {
	var (
		r        *HostReconciler
		instance *starlingxv1.Host
		host     v1info.HostInfo
	)

	BeforeEach(func() {
		scheme := runtime.NewScheme()
		err := starlingxv1.AddToScheme(scheme)
		Expect(err).ToNot(HaveOccurred())

		instance = &starlingxv1.Host{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-host",
				Namespace: "default",
			},
		}

		fakeClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithStatusSubresource(instance).
			WithObjects(instance).
			Build()

		r = &HostReconciler{
			Client: fakeClient,
		}

		fakeClock := "ntp"
		host = v1info.HostInfo{}
		host.Hostname = "test-host"
		host.ID = "test-host-id"
		host.Personality = "worker"
		host.SubFunctions = "worker"
		host.AdministrativeState = "locked"
		host.BootMAC = "00:11:22:33:44:55"
		host.Console = "tty0"
		host.InstallOutput = "text"
		host.BootDevice = "sda"
		host.RootDevice = "sda"
		host.ClockSynchronization = &fakeClock

		// Routes from system API
		host.Routes = []routes.Route{
			{
				InterfaceName: "eth0",
				Network:       "10.10.10.0",
				Prefix:        24,
				Gateway:       "10.10.10.1",
				Metric:        1,
			},
			{
				InterfaceName: "eth1",
				Network:       "172.16.0.0",
				Prefix:        16,
				Gateway:       "172.16.0.1",
				Metric:        5,
			},
		}
	})

	It("should clear Routes from defaults", func() {
		defaults, err := r.BuildHostDefaults(instance, host)
		Expect(err).ToNot(HaveOccurred())
		Expect(defaults).NotTo(BeNil())
		Expect(defaults.Routes).To(BeNil())
	})

	It("should persist nil Routes in stored status defaults", func() {
		_, err := r.BuildHostDefaults(instance, host)
		Expect(err).ToNot(HaveOccurred())

		stored, err := r.GetHostDefaults(instance)
		Expect(err).ToNot(HaveOccurred())
		Expect(stored).NotTo(BeNil())
		Expect(stored.Routes).To(BeNil())
	})

	It("should still populate non-route attributes", func() {
		defaults, err := r.BuildHostDefaults(instance, host)
		Expect(err).ToNot(HaveOccurred())
		Expect(defaults).NotTo(BeNil())
		Expect(defaults.Personality).NotTo(BeNil())
		Expect(*defaults.Personality).To(Equal("worker"))
		Expect(defaults.BootMAC).NotTo(BeNil())
		Expect(*defaults.BootMAC).To(Equal("00:11:22:33:44:55"))
	})

	It("should store valid JSON in instance status", func() {
		_, err := r.BuildHostDefaults(instance, host)
		Expect(err).ToNot(HaveOccurred())
		Expect(instance.Status.Defaults).NotTo(BeNil())

		var parsed starlingxv1.HostProfileSpec
		err = json.Unmarshal([]byte(*instance.Status.Defaults), &parsed)
		Expect(err).ToNot(HaveOccurred())
		Expect(parsed.Routes).To(BeNil())
	})
})
