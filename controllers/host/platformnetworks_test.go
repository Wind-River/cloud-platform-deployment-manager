package host

import (
	// "context"

	"context"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"

	starlingxv1 "github.com/wind-river/cloud-platform-deployment-manager/api/v1"
	comm "github.com/wind-river/cloud-platform-deployment-manager/common"
	utils "github.com/wind-river/cloud-platform-deployment-manager/common"
	"github.com/wind-river/cloud-platform-deployment-manager/controllers"
	cloudManager "github.com/wind-river/cloud-platform-deployment-manager/controllers/manager"
)

var k8sManager, _ = ctrl.NewManager(cfg, ctrl.Options{
	Scheme:             scheme.Scheme,
	MetricsBindAddress: "0",
})

const PlatformNetworkFinalizerName = "platformnetwork.finalizers.windriver.com"

const TestNamespace = "default"
const (
	timeout  = time.Second * 10
	interval = time.Millisecond * 250
)

func IntroducePlatformNetworkChange(platform_network *starlingxv1.PlatformNetwork) {
	if platform_network.Spec.Dynamic == true {
		platform_network.Spec.Dynamic = false
	} else {
		platform_network.Spec.Dynamic = true
	}
}

func IntroduceAddrPoolChange(addrpool *starlingxv1.AddressPool) {
	if utils.IsIPv4(addrpool.Spec.Subnet) {
		addrpool.Spec.Subnet = "100.100.100.100"
	} else {
		addrpool.Spec.Subnet = "100::100"
	}
}

func CreateDummyHost(hostname string) {
	annotations := make(map[string]string)
	annotations["kubectl.kubernetes.io/last-applied-configuration"] = `{"status":{"deploymentScope":"bootstrap"}}`

	personality_controller := "controller"
	admin_state := "unlocked"
	interfaces := &starlingxv1.InterfaceInfo{}
	dummy_hostprofile := &starlingxv1.HostProfile{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "dummy-profile-for-" + hostname,
			Namespace: TestNamespace,
		},
		Spec: starlingxv1.HostProfileSpec{
			ProfileBaseAttributes: starlingxv1.ProfileBaseAttributes{
				Personality:         &personality_controller,
				AdministrativeState: &admin_state,
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

	Eventually(func() bool {
		crd_key := types.NamespacedName{
			Name:      hostname,
			Namespace: TestNamespace,
		}
		crd_fetched := &starlingxv1.Host{}
		err := k8sClient.Get(ctx, crd_key, crd_fetched)
		return err == nil && crd_fetched.Status.Defaults != nil
	}, timeout, interval).Should(BeTrue())

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

func DeleteAddressPool(pool_name string) {
	key := types.NamespacedName{
		Name:      pool_name,
		Namespace: TestNamespace,
	}

	fetched := &starlingxv1.AddressPool{}

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

var _ = Describe("Networking utils", func() {

	Context(" Verify reconfiguration is blocked for all networks in Day-1", func() {
		It("Should be created successfully and Reconciled should be true & InSync should be 'false'", func() {
			tMgr := cloudManager.GetInstance(k8sManager)
			HostControllerAPIHandlers()
			tMgr.SetSystemReady(TestNamespace, true)

			CreateDummyHost("controller-0")
			defer DeleteDummyHost("controller-0")

			network_names := []string{"mgmt", "admin", "oam"}
			platform_networks, address_pools := GetPlatformNetworksFromFixtures(TestNamespace)
			ctx := context.Background()
			for _, nwk_name := range network_names {

				key_net := types.NamespacedName{
					Name:      nwk_name,
					Namespace: TestNamespace,
				}
				Expect(k8sClient.Create(ctx, platform_networks[nwk_name])).To(Succeed())

				expected_platformnetwork := platform_networks[nwk_name].DeepCopy()

				for _, pool := range address_pools[nwk_name] {
					IntroduceAddrPoolChange(pool)

					Expect(k8sClient.Create(ctx, pool)).To(Succeed())
				}

				fetched_net := &starlingxv1.PlatformNetwork{}
				Eventually(func() bool {
					err := k8sClient.Get(ctx, key_net, fetched_net)
					return err == nil &&
						fetched_net.ObjectMeta.ResourceVersion != expected_platformnetwork.ObjectMeta.ResourceVersion &&
						fetched_net.Status.Reconciled == true &&
						fetched_net.Status.InSync == true
				}, timeout, interval).Should(BeTrue())
				_, found := comm.ListIntersect(fetched_net.ObjectMeta.Finalizers, []string{controllers.PlatformNetworkFinalizerName})
				Expect(found).To(BeTrue())

				for _, pool := range address_pools[nwk_name] {
					expected_addrpool := pool.DeepCopy()

					key_pool := types.NamespacedName{
						Name:      pool.Name,
						Namespace: TestNamespace,
					}

					fetched_pool := &starlingxv1.AddressPool{}
					Eventually(func() bool {
						err := k8sClient.Get(ctx, key_pool, fetched_pool)
						return err == nil &&
							fetched_pool.ObjectMeta.ResourceVersion != expected_addrpool.ObjectMeta.ResourceVersion &&
							fetched_pool.Status.Reconciled == true &&
							fetched_pool.Status.InSync == false
					}, timeout, interval).Should(BeTrue())
					_, found = comm.ListIntersect(fetched_pool.ObjectMeta.Finalizers, []string{controllers.AddressPoolFinalizerName})
					Expect(found).To(BeTrue())

					DeleteAddressPool(pool.Name)
				}

				DeletePlatformNetwork(nwk_name)
			}
		})
	})

	Context("Verify fresh configuration is successful for cluster-host in Day-1", func() {
		It("Should be created successfully and Reconciled should be 'true' and Insync should be 'true'", func() {
			CreateDummyHost("controller-0")
			defer DeleteDummyHost("controller-0")

			network_names := []string{"cluster-host"}
			platform_networks, address_pools := GetPlatformNetworksFromFixtures(TestNamespace)
			ctx := context.Background()
			for _, nwk_name := range network_names {

				key_net := types.NamespacedName{
					Name:      nwk_name,
					Namespace: TestNamespace,
				}
				Expect(k8sClient.Create(ctx, platform_networks[nwk_name])).To(Succeed())

				expected_platformnetwork := platform_networks[nwk_name].DeepCopy()

				for _, pool := range address_pools[nwk_name] {
					IntroduceAddrPoolChange(pool)

					Expect(k8sClient.Create(ctx, pool)).To(Succeed())
				}

				fetched_net := &starlingxv1.PlatformNetwork{}
				Eventually(func() bool {
					err := k8sClient.Get(ctx, key_net, fetched_net)
					return err == nil &&
						fetched_net.ObjectMeta.ResourceVersion != expected_platformnetwork.ObjectMeta.ResourceVersion &&
						fetched_net.Status.Reconciled == true &&
						fetched_net.Status.InSync == true
				}, timeout, interval).Should(BeTrue())
				_, found := comm.ListIntersect(fetched_net.ObjectMeta.Finalizers, []string{controllers.PlatformNetworkFinalizerName})
				Expect(found).To(BeTrue())

				for _, pool := range address_pools[nwk_name] {
					expected_addrpool := pool.DeepCopy()

					key_pool := types.NamespacedName{
						Name:      pool.Name,
						Namespace: TestNamespace,
					}

					fetched_pool := &starlingxv1.AddressPool{}
					Eventually(func() bool {
						err := k8sClient.Get(ctx, key_pool, fetched_pool)
						return err == nil &&
							fetched_pool.ObjectMeta.ResourceVersion != expected_addrpool.ObjectMeta.ResourceVersion &&
							fetched_pool.Status.Reconciled == true &&
							fetched_pool.Status.InSync == true
					}, timeout, interval).Should(BeTrue())
					_, found = comm.ListIntersect(fetched_pool.ObjectMeta.Finalizers, []string{controllers.AddressPoolFinalizerName})
					Expect(found).To(BeTrue())

					DeleteAddressPool(pool.Name)
				}
				DeletePlatformNetwork(nwk_name)
			}
		})
	})

	Context(" Verify fresh configuration is blocked for networks such as oam / admin / mgmt", func() {
		It("Should be created successfully and Reconciled should be true & InSync should be 'false'", func() {
			CreateDummyHost("controller-0")
			defer DeleteDummyHost("controller-0")

			s := `
			{
				"gateway_address": null,
				"network": "192.168.204.0",
				"name": "management",
				"ranges": [
				  [
					"192.168.204.2",
					"192.168.204.50"
				  ]
				],
				"floating_address": "192.168.204.2",
				"controller0_address": "192.168.204.3",
				"controller1_address": "192.168.204.4",
				"prefix": 24,
				"order": "random",
				"uuid": "aa277c8e-7421-4721-ae6a-347771fe4fa6"
			  }
			  `
			s1 := `
			  {
				"uuid": "22222222-2222-425e-9317-995da88d6694",
				"network_uuid": "a48a7b6d-9cfa-24a4-8d48-f0e25d35984a",
				"address_pool_uuid": "aa277c8e-7421-4721-ae6a-347771fe4fa6",
				"network_name": "mgmt",
				"address_pool_name": "management"
			}
			`
			AddrPoolListBody = strings.Replace(AddrPoolListBody, s, "", 1)
			NetworkAddressPoolListBody = strings.Replace(NetworkAddressPoolListBody, s1, "", 1)

			network_names := []string{"mgmt"}
			platform_networks, address_pools := GetPlatformNetworksFromFixtures(TestNamespace)
			ctx := context.Background()
			for _, nwk_name := range network_names {

				key_net := types.NamespacedName{
					Name:      nwk_name,
					Namespace: TestNamespace,
				}
				Expect(k8sClient.Create(ctx, platform_networks[nwk_name])).To(Succeed())

				expected_platformnetwork := platform_networks[nwk_name].DeepCopy()

				for _, pool := range address_pools[nwk_name] {
					IntroduceAddrPoolChange(pool)

					Expect(k8sClient.Create(ctx, pool)).To(Succeed())
				}

				fetched_net := &starlingxv1.PlatformNetwork{}
				Eventually(func() bool {
					err := k8sClient.Get(ctx, key_net, fetched_net)
					return err == nil &&
						fetched_net.ObjectMeta.ResourceVersion != expected_platformnetwork.ObjectMeta.ResourceVersion &&
						fetched_net.Status.Reconciled == true &&
						fetched_net.Status.InSync == true
				}, timeout, interval).Should(BeTrue())
				_, found := comm.ListIntersect(fetched_net.ObjectMeta.Finalizers, []string{controllers.PlatformNetworkFinalizerName})
				Expect(found).To(BeTrue())
				for _, pool := range address_pools[nwk_name] {
					expected_addrpool := pool.DeepCopy()

					key_pool := types.NamespacedName{
						Name:      pool.Name,
						Namespace: TestNamespace,
					}

					fetched_pool := &starlingxv1.AddressPool{}
					Eventually(func() bool {
						err := k8sClient.Get(ctx, key_pool, fetched_pool)
						return err == nil &&
							fetched_pool.ObjectMeta.ResourceVersion != expected_addrpool.ObjectMeta.ResourceVersion &&
							fetched_pool.Status.Reconciled == true &&
							fetched_pool.Status.InSync == false
					}, timeout, interval).Should(BeTrue())
					_, found = comm.ListIntersect(fetched_pool.ObjectMeta.Finalizers, []string{controllers.AddressPoolFinalizerName})
					Expect(found).To(BeTrue())

					DeleteAddressPool(pool.Name)
				}

				DeletePlatformNetwork(nwk_name)
			}
		})
	})

	Context(" Verify IPv6 addresspool associated with pxeboot fails to reconcile without requeuing the request with fresh configuration in Day1", func() {
		It("Should be created successfully and Reconciled should be false & InSync should be 'false'", func() {
			CreateDummyHost("controller-0")
			defer DeleteDummyHost("controller-0")

			network_names := []string{"pxeboot"}
			platform_networks, address_pools := GetPlatformNetworksFromFixtures(TestNamespace)
			ctx := context.Background()
			for _, nwk_name := range network_names {

				key_net := types.NamespacedName{
					Name:      nwk_name,
					Namespace: TestNamespace,
				}
				Expect(k8sClient.Create(ctx, platform_networks[nwk_name])).To(Succeed())

				expected_platformnetwork := platform_networks[nwk_name].DeepCopy()

				for _, pool := range address_pools[nwk_name] {
					IntroduceAddrPoolChange(pool)

					Expect(k8sClient.Create(ctx, pool)).To(Succeed())
				}

				fetched_net := &starlingxv1.PlatformNetwork{}
				Eventually(func() bool {
					err := k8sClient.Get(ctx, key_net, fetched_net)
					return err == nil &&
						fetched_net.ObjectMeta.ResourceVersion != expected_platformnetwork.ObjectMeta.ResourceVersion &&
						fetched_net.Status.Reconciled == true &&
						fetched_net.Status.InSync == true
				}, timeout, interval).Should(BeTrue())
				_, found := comm.ListIntersect(fetched_net.ObjectMeta.Finalizers, []string{controllers.PlatformNetworkFinalizerName})
				Expect(found).To(BeTrue())
				for _, pool := range address_pools[nwk_name] {
					expected_addrpool := pool.DeepCopy()

					key_pool := types.NamespacedName{
						Name:      pool.Name,
						Namespace: TestNamespace,
					}

					fetched_pool := &starlingxv1.AddressPool{}
					Eventually(func() bool {
						err := k8sClient.Get(ctx, key_pool, fetched_pool)
						return err == nil &&
							fetched_pool.ObjectMeta.ResourceVersion != expected_addrpool.ObjectMeta.ResourceVersion &&
							fetched_pool.Status.Reconciled == true &&
							fetched_pool.Status.InSync == false
					}, timeout, interval).Should(BeTrue())
					_, found = comm.ListIntersect(fetched_pool.ObjectMeta.Finalizers, []string{controllers.AddressPoolFinalizerName})
					Expect(found).To(BeTrue())

					DeleteAddressPool(pool.Name)
				}

				DeletePlatformNetwork(nwk_name)
			}
		})
	})

	Context("Verify oam fails to reconcile if either 'floatingAddress' or 'gateway' is missing from the addresspool spec on AIO-SX.", func() {
		It("Should be created successfully and Reconciled should be true & InSync should be 'false'", func() {
			CreateDummyHost("controller-0")
			defer DeleteDummyHost("controller-0")

			network_names := []string{"oam"}
			platform_networks, address_pools := GetPlatformNetworksFromFixtures(TestNamespace)
			ctx := context.Background()
			for _, nwk_name := range network_names {

				key_net := types.NamespacedName{
					Name:      nwk_name,
					Namespace: TestNamespace,
				}
				Expect(k8sClient.Create(ctx, platform_networks[nwk_name])).To(Succeed())

				expected_platformnetwork := platform_networks[nwk_name].DeepCopy()

				for _, pool := range address_pools[nwk_name] {
					IntroduceAddrPoolChange(pool)

					Expect(k8sClient.Create(ctx, pool)).To(Succeed())
				}

				fetched_net := &starlingxv1.PlatformNetwork{}
				Eventually(func() bool {
					err := k8sClient.Get(ctx, key_net, fetched_net)
					return err == nil &&
						fetched_net.ObjectMeta.ResourceVersion != expected_platformnetwork.ObjectMeta.ResourceVersion &&
						fetched_net.Status.Reconciled == true &&
						fetched_net.Status.InSync == true
				}, timeout, interval).Should(BeTrue())
				_, found := comm.ListIntersect(fetched_net.ObjectMeta.Finalizers, []string{controllers.PlatformNetworkFinalizerName})
				Expect(found).To(BeTrue())
				for _, pool := range address_pools[nwk_name] {
					expected_addrpool := pool.DeepCopy()

					key_pool := types.NamespacedName{
						Name:      pool.Name,
						Namespace: TestNamespace,
					}

					fetched_pool := &starlingxv1.AddressPool{}
					Eventually(func() bool {
						err := k8sClient.Get(ctx, key_pool, fetched_pool)
						return err == nil &&
							fetched_pool.ObjectMeta.ResourceVersion != expected_addrpool.ObjectMeta.ResourceVersion &&
							fetched_pool.Status.Reconciled == true &&
							fetched_pool.Status.InSync == false
					}, timeout, interval).Should(BeTrue())
					_, found = comm.ListIntersect(fetched_pool.ObjectMeta.Finalizers, []string{controllers.AddressPoolFinalizerName})
					Expect(found).To(BeTrue())

					DeleteAddressPool(pool.Name)
				}

				DeletePlatformNetwork(nwk_name)
			}
		})
	})

	Context("Verify pxeboot fails to reconcile if any of 'floatingAddress', controller0Address' or 'controller1Address' are missing from the addresspool spec", func() {
		It("Should be created successfully and Reconciled should be false & InSync should be 'false'", func() {
			CreateDummyHost("controller-0")
			defer DeleteDummyHost("controller-0")

			network_names := []string{"pxeboot"}
			platform_networks, address_pools := GetPlatformNetworksFromFixtures(TestNamespace)

			ctx := context.Background()
			for _, nwk_name := range network_names {

				key_net := types.NamespacedName{
					Name:      nwk_name,
					Namespace: TestNamespace,
				}
				Expect(k8sClient.Create(ctx, platform_networks[nwk_name])).To(Succeed())

				expected_platformnetwork := platform_networks[nwk_name].DeepCopy()

				for _, pool := range address_pools[nwk_name] {
					IntroduceAddrPoolChange(pool)

					Expect(k8sClient.Create(ctx, pool)).To(Succeed())
				}

				fetched_net := &starlingxv1.PlatformNetwork{}
				Eventually(func() bool {
					err := k8sClient.Get(ctx, key_net, fetched_net)
					return err == nil &&
						fetched_net.ObjectMeta.ResourceVersion != expected_platformnetwork.ObjectMeta.ResourceVersion &&
						fetched_net.Status.Reconciled == true &&
						fetched_net.Status.InSync == true
				}, timeout, interval).Should(BeTrue())
				_, found := comm.ListIntersect(fetched_net.ObjectMeta.Finalizers, []string{controllers.PlatformNetworkFinalizerName})
				Expect(found).To(BeTrue())
				for _, pool := range address_pools[nwk_name] {
					expected_addrpool := pool.DeepCopy()

					key_pool := types.NamespacedName{
						Name:      pool.Name,
						Namespace: TestNamespace,
					}

					fetched_pool := &starlingxv1.AddressPool{}
					Eventually(func() bool {
						err := k8sClient.Get(ctx, key_pool, fetched_pool)
						return err == nil &&
							fetched_pool.ObjectMeta.ResourceVersion != expected_addrpool.ObjectMeta.ResourceVersion &&
							fetched_pool.Status.Reconciled == true &&
							fetched_pool.Status.InSync == false
					}, timeout, interval).Should(BeTrue())
					_, found = comm.ListIntersect(fetched_pool.ObjectMeta.Finalizers, []string{controllers.AddressPoolFinalizerName})
					Expect(found).To(BeTrue())

					DeleteAddressPool(pool.Name)
				}

				DeletePlatformNetwork(nwk_name)
			}
		})
	})

})
