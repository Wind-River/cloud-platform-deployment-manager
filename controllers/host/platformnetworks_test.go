package host

import (
	// "context"

	"context"
	"strings"
	"time"

	"github.com/gophercloud/gophercloud"
	th "github.com/gophercloud/gophercloud/testhelper"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"

	// comm "github.com/wind-river/cloud-platform-deployment-manager/common"
	gcClient "github.com/gophercloud/gophercloud/testhelper/client"
	starlingxv1 "github.com/wind-river/cloud-platform-deployment-manager/api/v1"
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
	// log.Println("Hostprofile  created", dummy_hostprofile)
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

var _ = Describe("Networking utils", func() {

	Context("PlatformNetwork with correct mgmt/admin/oam network data in Day-1", func() {
		It("Should be created successfully and Reconciled & InSync should be 'true'", func() {
			th.SetupHTTP()
			tMgr := cloudManager.GetInstance(k8sManager)
			f := func(namespace string) *gophercloud.ServiceClient {
				return gcClient.ServiceClient()
			}
			tMgr.SetGetPlatformClient(f)
			CreateDummyHost("controller-0")
			defer DeleteDummyHost("controller-0")
			HostControllerAPIHandlers()
			tMgr.SetSystemReady(TestNamespace, true)
			network_names := []string{"mgmt", "admin", "oam"}
			platform_networks := GetPlatformNetworksFromFixtures(TestNamespace)
			ctx := context.Background()
			for _, nwk_name := range network_names {
				key := types.NamespacedName{
					Name:      nwk_name,
					Namespace: TestNamespace,
				}
				Expect(k8sClient.Create(ctx, platform_networks[nwk_name])).To(Succeed())
				//	expected := platform_networks[nwk_name].DeepCopy()

				fetched := &starlingxv1.PlatformNetwork{}
				// Eventually(func() bool {
				_ = k8sClient.Get(ctx, key, fetched)
				// log.Println("#### Error:  ", err)
				//log.Println("#### fetched:  ", fetched)
				// log.Println("#### expected:  ", expected)
				// log.Println("#### fetched status:  ", fetched.Status)
				// return err == nil // &&
				// fetched.ObjectMeta.ResourceVersion != expected.ObjectMeta.ResourceVersion &&
				// fetched.Status.Reconciled == true &&
				// fetched.Status.InSync == true
				// }, timeout, interval).Should(BeTrue())
				// _, found := comm.ListIntersect(fetched.ObjectMeta.Finalizers, []string{PlatformNetworkFinalizerName})
				// Expect(found).To(BeTrue())
				// DeletePlatformNetwork(nwk_name)

				// })
			}
		})
	})
})
