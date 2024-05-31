/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2022-2024 Wind River Systems, Inc. */
package controllers

import (
	// "context"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"strings"

	starlingxv1 "github.com/wind-river/cloud-platform-deployment-manager/api/v1"
	// comm "github.com/wind-river/cloud-platform-deployment-manager/common"
	cloudManager "github.com/wind-river/cloud-platform-deployment-manager/controllers/manager"
)

const TestNamespace = "default"

func IntroducePlatformNetworkChange(platform_network *starlingxv1.PlatformNetwork) {
	if platform_network.Spec.Dynamic == true {
		platform_network.Spec.Dynamic = false
	} else {
		platform_network.Spec.Dynamic = true
	}
}

func CreateDummyHost(hostname string) {
	annotations := make(map[string]string)
	annotations["kubectl.kubernetes.io/last-applied-configuration"] = `{"status":{"deploymentScope":"bootstrap"}}`

	personality_controller := "controller"
	interfaces := &starlingxv1.InterfaceInfo{}
	dummy_hostprofile := &starlingxv1.HostProfile{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "dummy-profile-for-" + hostname,
			Namespace: TestNamespace,
		},
		Spec: starlingxv1.HostProfileSpec{
			ProfileBaseAttributes: starlingxv1.ProfileBaseAttributes{
				Personality: &personality_controller,
			},
			Interfaces: interfaces,
		}}
	Expect(k8sClient.Create(ctx, dummy_hostprofile)).To(Succeed())

	bootMac := "01:02:03:04:05:06"
	match := starlingxv1.MatchInfo{
		BootMAC: &bootMac,
	}
	dummy_host := &starlingxv1.Host{
		ObjectMeta: metav1.ObjectMeta{
			Name:        hostname,
			Namespace:   TestNamespace,
			Annotations: annotations,
		},
		Spec: starlingxv1.HostSpec{
			Profile: "dummy-profile-for-" + hostname,
			Match:   &match,
			Overrides: &starlingxv1.HostProfileSpec{
				Addresses: []starlingxv1.AddressInfo{
					{Interface: "enp0s3", Address: "1.2.3.10", Prefix: 24},
				},
			},
		}}
	Expect(k8sClient.Create(ctx, dummy_host)).To(Succeed())
}

func DeleteDummyHost(hostname string) {
	for _, key := range []string{hostname, "dummy-profile-for-" + hostname} {
		crd_key := types.NamespacedName{
			Name:      key,
			Namespace: TestNamespace,
		}

		if strings.Contains(key, "profile") {
			crd_fetched := &starlingxv1.HostProfile{}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, crd_key, crd_fetched)
				if err == nil {
					crd_fetched.ObjectMeta.Finalizers = []string{}
					err = k8sClient.Update(ctx, crd_fetched)
					if err == nil {
						err = k8sClient.Delete(ctx, crd_fetched)
						if err == nil {
							return true
						}
					}
				}
				return false
			}, timeout, interval).Should(BeTrue())

			Eventually(func() bool {
				err := k8sClient.Get(ctx, crd_key, crd_fetched)
				return err != nil
			}, timeout, interval).Should(BeTrue())

		} else {
			crd_fetched := &starlingxv1.Host{}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, crd_key, crd_fetched)
				if err == nil {
					crd_fetched.ObjectMeta.Finalizers = []string{}
					err = k8sClient.Update(ctx, crd_fetched)
					if err == nil {
						err = k8sClient.Delete(ctx, crd_fetched)
						if err == nil {
							return true
						}
					}
				}
				return false
			}, timeout, interval).Should(BeTrue())

			Eventually(func() bool {
				err := k8sClient.Get(ctx, crd_key, crd_fetched)
				return err != nil
			}, timeout, interval).Should(BeTrue())
		}
	}
}

func DeletePlatformNetwork(nwk_name string) {
	key := types.NamespacedName{
		Name:      nwk_name,
		Namespace: TestNamespace,
	}

	fetched := &starlingxv1.PlatformNetwork{}

	Eventually(func() bool {
		err := k8sClient.Get(ctx, key, fetched)
		if err == nil {
			fetched.ObjectMeta.Finalizers = []string{}
			err = k8sClient.Update(ctx, fetched)
			if err == nil {
				err = k8sClient.Delete(ctx, fetched)
				if err == nil {
					return true
				}
			}
		}
		return false
	}, timeout, interval).Should(BeTrue())

	Eventually(func() bool {
		err := k8sClient.Get(ctx, key, fetched)
		return err != nil
	}, timeout, interval).Should(BeTrue())
}

func SimulateVIMStrategyAction(hostname string, expect_strategy string) {
	host_key := types.NamespacedName{
		Name:      hostname,
		Namespace: TestNamespace,
	}
	host_instance := &starlingxv1.Host{}

	Eventually(func() bool {
		err := k8sClient.Get(ctx, host_key, host_instance)
		Expect(err).To(BeNil())
		return (host_instance.Status.StrategyRequired == expect_strategy)
	}, timeout, interval).Should(BeTrue())

	if host_instance.Status.StrategyRequired == cloudManager.StrategyLockRequired {
		HostsListBodyResponse = strings.Replace(HostsListBody, `"administrative": "unlocked",`, `"administrative": "locked",`, 1)
		HostsListBodyResponse = strings.Replace(HostsListBodyResponse, `"operational": "enabled"`, `"operational": "disabled"`, 1)
	} else {
		HostsListBodyResponse = HostsListBody
	}
}

var _ = Describe("Platformnetwork controller", func() {

	const (
		timeout  = time.Second * 10
		interval = time.Millisecond * 250
	)

	// Context("PlatformNetwork with correct mgmt/admin/oam network data in Day-1", func() {
	// 	It("Should be created successfully and Reconciled & InSync should be 'true'", func() {

	// 		tMgr := cloudManager.GetInstance(k8sManager)
	// 		StartPlatformNetworkAPIHandlers()
	// 		tMgr.SetSystemReady(TestNamespace, true)

	// 		network_names := []string{"mgmt", "admin", "oam"}
	// 		platform_networks := GetPlatformNetworksFromFixtures(TestNamespace)
	// 		ctx := context.Background()

	// 		for _, nwk_name := range network_names {
	// 			key := types.NamespacedName{
	// 				Name:      nwk_name,
	// 				Namespace: TestNamespace,
	// 			}

	// 			Expect(k8sClient.Create(ctx, platform_networks[nwk_name])).To(Succeed())

	// 			expected := platform_networks[nwk_name].DeepCopy()

	// 			fetched := &starlingxv1.PlatformNetwork{}
	// 			Eventually(func() bool {
	// 				err := k8sClient.Get(ctx, key, fetched)
	// 				return err == nil &&
	// 					fetched.ObjectMeta.ResourceVersion != expected.ObjectMeta.ResourceVersion &&
	// 					fetched.Status.Reconciled == true &&
	// 					fetched.Status.InSync == true
	// 			}, timeout, interval).Should(BeTrue())
	// 			_, found := comm.ListIntersect(fetched.ObjectMeta.Finalizers, []string{PlatformNetworkFinalizerName})
	// 			Expect(found).To(BeTrue())

	// 			DeletePlatformNetwork(nwk_name)
	// 		}
	// 	})
	// })

	// Context("PlatformNetwork with incorrect mgmt network data in Day-1", func() {
	// 	It("Should be created successfully and Reconciled should be 'true' and Insync should be 'false'", func() {

	// 		network_names := []string{"mgmt", "admin", "oam"}
	// 		platform_networks := GetPlatformNetworksFromFixtures(TestNamespace)
	// 		ctx := context.Background()

	// 		for _, nwk_name := range network_names {
	// 			key := types.NamespacedName{
	// 				Name:      nwk_name,
	// 				Namespace: TestNamespace,
	// 			}

	// 			IntroducePlatformNetworkChange(platform_networks[nwk_name])

	// 			Expect(k8sClient.Create(ctx, platform_networks[nwk_name])).To(Succeed())

	// 			expected := platform_networks[nwk_name].DeepCopy()

	// 			fetched := &starlingxv1.PlatformNetwork{}
	// 			Eventually(func() bool {
	// 				err := k8sClient.Get(ctx, key, fetched)
	// 				return err == nil &&
	// 					fetched.ObjectMeta.ResourceVersion != expected.ObjectMeta.ResourceVersion &&
	// 					fetched.Status.Reconciled == true &&
	// 					fetched.Status.InSync == false
	// 			}, timeout, interval).Should(BeTrue())
	// 			_, found := comm.ListIntersect(fetched.ObjectMeta.Finalizers, []string{PlatformNetworkFinalizerName})
	// 			Expect(found).To(BeTrue())

	// 			DeletePlatformNetwork(nwk_name)
	// 		}
	// 	})
	// })

	// Context("PlatformNetwork with incorrect network data for network other than oam / mgmt/ admin in Day-1", func() {
	// 	It("Should be created successfully and Reconciled should be 'true' and Insync should be 'true'", func() {
	// 		network_names := []string{"pxeboot"}
	// 		platform_networks := GetPlatformNetworksFromFixtures(TestNamespace)
	// 		ctx := context.Background()

	// 		for _, nwk_name := range network_names {
	// 			key := types.NamespacedName{
	// 				Name:      nwk_name,
	// 				Namespace: TestNamespace,
	// 			}

	// 			platform_networks[nwk_name].Spec.FloatingAddress = "100.100.100.100"

	// 			Expect(k8sClient.Create(ctx, platform_networks[nwk_name])).To(Succeed())

	// 			expected := platform_networks[nwk_name].DeepCopy()

	// 			fetched := &starlingxv1.PlatformNetwork{}
	// 			Eventually(func() bool {
	// 				err := k8sClient.Get(ctx, key, fetched)
	// 				return err == nil &&
	// 					fetched.ObjectMeta.ResourceVersion != expected.ObjectMeta.ResourceVersion &&
	// 					fetched.Status.Reconciled == true &&
	// 					fetched.Status.InSync == true
	// 			}, timeout, interval).Should(BeTrue())
	// 			_, found := comm.ListIntersect(fetched.ObjectMeta.Finalizers, []string{PlatformNetworkFinalizerName})
	// 			Expect(found).To(BeTrue())

	// 			DeletePlatformNetwork(nwk_name)
	// 		}
	// 	})
	// })

	// Context("PlatformNetwork with admin/oam network data in Day-2 (Network Reconfiguration)", func() {
	// 	It("Should be created successfully and Reconciled & InSync should be 'true'", func() {

	// 		network_names := []string{"admin", "oam"}
	// 		platform_networks := GetPlatformNetworksFromFixtures(TestNamespace)
	// 		ctx := context.Background()

	// 		CreateDummyHost("controller-0")
	// 		defer DeleteDummyHost("controller-0")

	// 		annotations := make(map[string]string)
	// 		annotations["kubectl.kubernetes.io/last-applied-configuration"] = `{"status":{"deploymentScope":"principal"}}`

	// 		for _, nwk_name := range network_names {
	// 			key := types.NamespacedName{
	// 				Name:      nwk_name,
	// 				Namespace: TestNamespace,
	// 			}

	// 			IntroducePlatformNetworkChange(platform_networks[nwk_name])

	// 			platform_networks[nwk_name].ObjectMeta.Annotations = annotations

	// 			Expect(k8sClient.Create(ctx, platform_networks[nwk_name])).To(Succeed())

	// 			expected := platform_networks[nwk_name].DeepCopy()

	// 			fetched := &starlingxv1.PlatformNetwork{}
	// 			Eventually(func() bool {
	// 				err := k8sClient.Get(ctx, key, fetched)
	// 				return err == nil &&
	// 					fetched.ObjectMeta.ResourceVersion != expected.ObjectMeta.ResourceVersion &&
	// 					fetched.Status.Reconciled == true &&
	// 					fetched.Status.InSync == true
	// 			}, timeout, interval).Should(BeTrue())
	// 			_, found := comm.ListIntersect(fetched.ObjectMeta.Finalizers, []string{PlatformNetworkFinalizerName})
	// 			Expect(found).To(BeTrue())

	// 			DeletePlatformNetwork(nwk_name)
	// 		}
	// 	})
	// })

	// Context("Reconfigure admin network on AIO-SX - Day 2", func() {
	// 	It("Should not trigger lock / unlock of host", func() {
	// 		platform_networks := GetPlatformNetworksFromFixtures(TestNamespace)
	// 		ctx := context.Background()

	// 		CreateDummyHost("controller-0")
	// 		defer DeleteDummyHost("controller-0")

	// 		nwk_name := "admin"
	// 		annotations := make(map[string]string)
	// 		annotations["kubectl.kubernetes.io/last-applied-configuration"] = `{"status":{"deploymentScope":"principal"}}`

	// 		key := types.NamespacedName{
	// 			Name:      nwk_name,
	// 			Namespace: TestNamespace,
	// 		}

	// 		IntroducePlatformNetworkChange(platform_networks[nwk_name])

	// 		platform_networks[nwk_name].ObjectMeta.Annotations = annotations

	// 		Expect(k8sClient.Create(ctx, platform_networks[nwk_name])).To(Succeed())

	// 		expected := platform_networks[nwk_name].DeepCopy()

	// 		SimulateVIMStrategyAction("controller-0", cloudManager.StrategyNotRequired)

	// 		fetched := &starlingxv1.PlatformNetwork{}
	// 		Eventually(func() bool {
	// 			err := k8sClient.Get(ctx, key, fetched)
	// 			return err == nil &&
	// 				fetched.ObjectMeta.ResourceVersion != expected.ObjectMeta.ResourceVersion &&
	// 				fetched.Status.Reconciled == true &&
	// 				fetched.Status.InSync == true
	// 		}, timeout, interval).Should(BeTrue())
	// 		_, found := comm.ListIntersect(fetched.ObjectMeta.Finalizers, []string{PlatformNetworkFinalizerName})
	// 		Expect(found).To(BeTrue())

	// 		DeletePlatformNetwork(nwk_name)
	// 	})
	// })

	// Context("Reconfigure OAM network on AIO-SX - Day 2", func() {
	// 	It("Should not trigger lock / unlock of host", func() {
	// 		platform_networks := GetPlatformNetworksFromFixtures(TestNamespace)
	// 		ctx := context.Background()

	// 		CreateDummyHost("controller-0")
	// 		defer DeleteDummyHost("controller-0")

	// 		nwk_name := "oam"
	// 		annotations := make(map[string]string)
	// 		annotations["kubectl.kubernetes.io/last-applied-configuration"] = `{"status":{"deploymentScope":"principal"}}`

	// 		key := types.NamespacedName{
	// 			Name:      nwk_name,
	// 			Namespace: TestNamespace,
	// 		}

	// 		IntroducePlatformNetworkChange(platform_networks[nwk_name])

	// 		platform_networks[nwk_name].ObjectMeta.Annotations = annotations

	// 		Expect(k8sClient.Create(ctx, platform_networks[nwk_name])).To(Succeed())

	// 		expected := platform_networks[nwk_name].DeepCopy()

	// 		SimulateVIMStrategyAction("controller-0", cloudManager.StrategyNotRequired)

	// 		fetched := &starlingxv1.PlatformNetwork{}
	// 		Eventually(func() bool {
	// 			err := k8sClient.Get(ctx, key, fetched)
	// 			return err == nil &&
	// 				fetched.ObjectMeta.ResourceVersion != expected.ObjectMeta.ResourceVersion &&
	// 				fetched.Status.Reconciled == true &&
	// 				fetched.Status.InSync == true
	// 		}, timeout, interval).Should(BeTrue())
	// 		_, found := comm.ListIntersect(fetched.ObjectMeta.Finalizers, []string{PlatformNetworkFinalizerName})
	// 		Expect(found).To(BeTrue())

	// 		DeletePlatformNetwork(nwk_name)
	// 	})
	// })

	// Context("Reconfigure management network on AIO-SX - Day 2", func() {
	// 	It("Should trigger lock / unlock of host", func() {
	// 		tMgr := cloudManager.GetInstance(k8sManager)
	// 		platform_networks := GetPlatformNetworksFromFixtures(TestNamespace)

	// 		ctx := context.Background()

	// 		CreateDummyHost("controller-0")
	// 		defer DeleteDummyHost("controller-0")

	// 		nwk_name := "mgmt"
	// 		annotations := make(map[string]string)
	// 		annotations["kubectl.kubernetes.io/last-applied-configuration"] = `{"status":{"deploymentScope":"principal"}}`

	// 		key := types.NamespacedName{
	// 			Name:      nwk_name,
	// 			Namespace: TestNamespace,
	// 		}

	// 		IntroducePlatformNetworkChange(platform_networks[nwk_name])

	// 		platform_networks[nwk_name].ObjectMeta.Annotations = annotations

	// 		Expect(k8sClient.Create(ctx, platform_networks[nwk_name])).To(Succeed())

	// 		expected := platform_networks[nwk_name].DeepCopy()

	// 		SimulateVIMStrategyAction("controller-0", cloudManager.StrategyLockRequired)

	// 		Expect(tMgr.IsPlatformNetworkReconciling()).To(BeTrue())

	// 		fetched := &starlingxv1.PlatformNetwork{}
	// 		// Timeout in this case is set to 45 seconds because we are returning NewResourceConfigurationDependency
	// 		// error while waiting for VIM strategy to lock the host.
	// 		// This means reconciliation is attempted after every 20 seconds. Setting the timeout to
	// 		// 45 seconds would allow at least two retries before marking the test case as failure.
	// 		Eventually(func() bool {
	// 			err := k8sClient.Get(ctx, key, fetched)
	// 			return err == nil &&
	// 				fetched.ObjectMeta.ResourceVersion != expected.ObjectMeta.ResourceVersion &&
	// 				fetched.Status.Reconciled == true &&
	// 				fetched.Status.InSync == true
	// 		}, time.Second*45, interval).Should(BeTrue())
	// 		_, found := comm.ListIntersect(fetched.ObjectMeta.Finalizers, []string{PlatformNetworkFinalizerName})
	// 		Expect(found).To(BeTrue())

	// 		Expect(tMgr.IsPlatformNetworkReconciling()).To(BeFalse())

	// 		DeletePlatformNetwork(nwk_name)
	// 	})
	// })

	// Context("Reconfigure management network on AIO-DX - Day 2", func() {
	// 	It("Should NOT attempt reconciliation", func() {
	// 		SingleSystemBodyResponse = strings.Replace(SingleSystemBody, `"system_mode": "simplex",`, `"system_mode": "duplex",`, 1)
	// 		platform_networks := GetPlatformNetworksFromFixtures(TestNamespace)
	// 		ctx := context.Background()

	// 		CreateDummyHost("controller-0")
	// 		defer DeleteDummyHost("controller-0")

	// 		nwk_name := "mgmt"
	// 		annotations := make(map[string]string)
	// 		annotations["kubectl.kubernetes.io/last-applied-configuration"] = `{"status":{"deploymentScope":"principal"}}`

	// 		key := types.NamespacedName{
	// 			Name:      nwk_name,
	// 			Namespace: TestNamespace,
	// 		}

	// 		IntroducePlatformNetworkChange(platform_networks[nwk_name])

	// 		platform_networks[nwk_name].ObjectMeta.Annotations = annotations

	// 		Expect(k8sClient.Create(ctx, platform_networks[nwk_name])).To(Succeed())

	// 		expected := platform_networks[nwk_name].DeepCopy()

	// 		fetched := &starlingxv1.PlatformNetwork{}

	// 		Eventually(func() bool {
	// 			err := k8sClient.Get(ctx, key, fetched)
	// 			return err == nil &&
	// 				fetched.ObjectMeta.ResourceVersion != expected.ObjectMeta.ResourceVersion &&
	// 				fetched.Status.Reconciled == false &&
	// 				fetched.Status.InSync == false
	// 		}, timeout, interval).Should(BeTrue())
	// 		_, found := comm.ListIntersect(fetched.ObjectMeta.Finalizers, []string{PlatformNetworkFinalizerName})
	// 		Expect(found).To(BeTrue())

	// 		DeletePlatformNetwork(nwk_name)
	// 	})
	// })

	// Context("Reconfigure networks other than management on AIO-DX - Day 2", func() {
	// 	It("Should reoncile the networks", func() {
	// 		SingleSystemBodyResponse = strings.Replace(SingleSystemBody, `"system_mode": "simplex",`, `"system_mode": "duplex",`, 1)
	// 		platform_networks := GetPlatformNetworksFromFixtures(TestNamespace)
	// 		ctx := context.Background()

	// 		CreateDummyHost("controller-0")
	// 		defer DeleteDummyHost("controller-0")

	// 		nwk_name := "pxeboot"
	// 		annotations := make(map[string]string)
	// 		annotations["kubectl.kubernetes.io/last-applied-configuration"] = `{"status":{"deploymentScope":"principal"}}`

	// 		key := types.NamespacedName{
	// 			Name:      nwk_name,
	// 			Namespace: TestNamespace,
	// 		}

	// 		IntroducePlatformNetworkChange(platform_networks[nwk_name])

	// 		platform_networks[nwk_name].ObjectMeta.Annotations = annotations

	// 		Expect(k8sClient.Create(ctx, platform_networks[nwk_name])).To(Succeed())

	// 		expected := platform_networks[nwk_name].DeepCopy()

	// 		fetched := &starlingxv1.PlatformNetwork{}

	// 		Eventually(func() bool {
	// 			err := k8sClient.Get(ctx, key, fetched)
	// 			return err == nil &&
	// 				fetched.ObjectMeta.ResourceVersion != expected.ObjectMeta.ResourceVersion &&
	// 				fetched.Status.Reconciled == true &&
	// 				fetched.Status.InSync == true
	// 		}, timeout, interval).Should(BeTrue())
	// 		_, found := comm.ListIntersect(fetched.ObjectMeta.Finalizers, []string{PlatformNetworkFinalizerName})
	// 		Expect(found).To(BeTrue())

	// 		DeletePlatformNetwork(nwk_name)
	// 	})
	// })

	// Context("Test Restore In Progress", func() {
	// 	It("Should update inSync/deploymentScope/strategyRequired without reconciling", func() {
	// 		platform_networks := GetPlatformNetworksFromFixtures(TestNamespace)
	// 		ctx := context.Background()

	// 		CreateDummyHost("controller-0")
	// 		defer DeleteDummyHost("controller-0")

	// 		nwk_name := "pxeboot"
	// 		annotations := make(map[string]string)
	// 		annotations["kubectl.kubernetes.io/last-applied-configuration"] = `{"status":{"deploymentScope":"bootstrap"}}`
	// 		annotations["deployment-manager/restore-in-progress"] = `{"inSync": false, "reconciled": false, "deploymentScope": "principal"}`

	// 		key := types.NamespacedName{
	// 			Name:      nwk_name,
	// 			Namespace: TestNamespace,
	// 		}

	// 		IntroducePlatformNetworkChange(platform_networks[nwk_name])

	// 		platform_networks[nwk_name].ObjectMeta.Annotations = annotations

	// 		Expect(k8sClient.Create(ctx, platform_networks[nwk_name])).To(Succeed())

	// 		fetched := &starlingxv1.PlatformNetwork{}

	// 		Eventually(func() bool {
	// 			err := k8sClient.Get(ctx, key, fetched)
	// 			return err == nil &&
	// 				fetched.Status.Reconciled == true &&
	// 				fetched.Status.InSync == false &&
	// 				fetched.Status.DeploymentScope == "bootstrap" &&
	// 				len(fetched.Annotations) == 1
	// 		}, timeout, interval).Should(BeTrue())
	// 		_, found := comm.ListIntersect(fetched.ObjectMeta.Finalizers, []string{PlatformNetworkFinalizerName})
	// 		Expect(found).To(BeTrue())

	// 		DeletePlatformNetwork(nwk_name)
	// 	})
	// })

})
