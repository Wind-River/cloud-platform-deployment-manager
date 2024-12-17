/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2023-2024 Wind River Systems, Inc. */

package build

import (
	"fmt"
	"net/http"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/licenses"
	th "github.com/gophercloud/gophercloud/testhelper"
	gcClient "github.com/gophercloud/gophercloud/testhelper/client"
	starlingxv1 "github.com/wind-river/cloud-platform-deployment-manager/api/v1"
	"github.com/wind-river/cloud-platform-deployment-manager/controllers/manager"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestBuild(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Build Suite")
}

var _ = BeforeSuite(func() {
	th.SetupHTTP()
	StartAPIHandlers()
}, 60)

var expYaml = `---
apiVersion: v1
kind: namespace
metadata:
  name: fakens
spec: {}
---
apiVersion: v1
data:
  password: ""
  username: dXNlcm5hbWU=
kind: Secret
metadata:
  name: secret1
  namespace: fakens
type: kubernetes.io/basic-auth
---
apiVersion: v1
data:
  Fake Data: 'Warning: Incomplete secret, please replace it with the secret content'
kind: Secret
metadata:
  name: incomsec1
  namespace: bar
type: fake type
---
metadata:
  name: sys1
  namespace: default
spec: {}
---
metadata:
  name: pn1
  namespace: default
spec:
  associatedAddressPools: null
  dynamic: false
  type: mgmt
---
metadata:
  name: pn1
  namespace: default
spec:
  allocation:
    order: random
    ranges:
    - end: 192.168.204.254
      start: 192.168.204.2
  controller0Address: 192.168.204.3
  controller1Address: 192.168.204.4
  floatingAddress: 192.168.204.2
  gateway: 192.168.204.1
  prefix: 24
  subnet: 192.168.204.0
---
metadata:
  name: dn1
  namespace: default
spec:
  type: ""
---
metadata:
  name: ptpinst1
  namespace: default
spec:
  service: ""
---
metadata:
  name: ptpinf1
  namespace: default
spec:
  ptpinstance: ""
---
metadata:
  name: hostprofile1
  namespace: default
spec: {}
---
metadata:
  name: host
  namespace: default
spec:
  profile: ""
---
`

const DataNetworkListBody = `
{
	"datanetworks": [
		{
  			"name": "data1"
		}
    ]
}
`
const AddrPoolListBody = `
{
    "addrpools": [
        {
            "name": "management",
	    "uuid": "aa277c8e-7421-4721-ae6a-347771fe4fa6"
        }
    ]
}
`
const PlatformNetworkListBody = `
{
    "networks": [
        {
			"dynamic": true,
			"name": "mgmt",
			"pool_uuid": "aa277c8e-7421-4721-ae6a-347771fe4fa6",
			"type": "mgmt",
			"uuid": "a48a7b6d-9cfa-24a4-8d48-f0e25d35984a"
        }
    ]
}
`
const PTPInstanceListBody = `
{
	"ptp_instances": [
		{
			"uuid": "fa5defce-2546-4786-ae58-7bb08e2105fc", 
		 	"service": "phc2sys", 
		 	"type": "ptp-instance",
		 	"name": "phc2sys1"
		}
	]
}
`
const PTPInterfaceListBody = `
{
	"ptp_interfaces": [
		{
			"ptp_instance_uuid": "fa5defce-2546-4786-ae58-7bb08e2105fc",
			"interface_names": [],
			"ptp_instance_id": 1,
			"uuid": "b7d51ba0-35d7-4bab-9e27-a8b701587c54",
			"parameters": [],
			"created_at": "2022-01-19T20:42:18.638033+00:00",
			"updated_at": null,
			"capabilities": {},
			"hostnames": [],
			"ptp_instance_name": "phc2sys1",
			"type": "ptp-interface",
			"name": "ptpint1"
		}
	]
}
`
const NetworkAddressPoolListBody = `
{
    "network_addresspools": [
        {
			"uuid": "11111111-a6e5-425e-9317-995da88d6694",
			"network_uuid": "a48a7b6d-9cfa-24a4-8d48-f0e25d35984a",
			"address_pool_uuid": "aa277c8e-7421-4721-ae6a-347771fe4fa6",
			"network_name": "oam",
			"address_pool_name": "oam-ipv4"
	}
    ]
}
`

func HandleResourceRequests(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		w.Header().Add("Content-Type", "application/json")
		path := r.URL.Path
		if strings.HasSuffix(path, "/"+"datanetworks") {
			fmt.Fprint(w, DataNetworkListBody)
			break
		}
		if strings.HasSuffix(path, "/"+"networks") {
			fmt.Fprint(w, PlatformNetworkListBody)
			break
		}
		if strings.HasSuffix(path, "/"+"addrpools") {
			fmt.Fprint(w, AddrPoolListBody)
			break
		}
		if strings.HasSuffix(path, "/"+"ptp_instances") {
			fmt.Fprint(w, PTPInstanceListBody)
			break
		}
		if strings.HasSuffix(path, "/"+"ptp_interfaces") {
			fmt.Fprint(w, PTPInterfaceListBody)
			break
		}
		if strings.HasSuffix(path, "/"+"network_addresspools") {
			fmt.Fprint(w, NetworkAddressPoolListBody)
			break
		}
	default:
		http.Error(w, `{"error": "Method not allowed"}`, http.StatusMethodNotAllowed)
	}
}

func StartAPIHandlers() {

	th.Mux.HandleFunc("/datanetworks", HandleResourceRequests)
	th.Mux.HandleFunc("/networks", HandleResourceRequests)
	th.Mux.HandleFunc("/addrpools", HandleResourceRequests)
	th.Mux.HandleFunc("/ptp_instances", HandleResourceRequests)
	th.Mux.HandleFunc("/ptp_interfaces", HandleResourceRequests)
	th.Mux.HandleFunc("/network_addresspools", HandleResourceRequests)
}

var _ = Describe("Test Build utilities:", func() {
	Describe("Test New NewDeploymentBuilder", func() {

		client := gophercloud.ServiceClient{}
		got := NewDeploymentBuilder(&client, "foo", "bar", os.Stdout)

		expectSystemFilter := []SystemFilter{
			NewServiceParametersSystemFilter()}
		Expect(reflect.DeepEqual(
			got.systemFilters, expectSystemFilter)).To(BeTrue())

		expectHostFilter := []HostFilter{
			NewController0Filter(),
			NewLoopbackInterfaceFilter(),
			NewLocationFilter(),
			NewAddressFilter(),
			NewBMAddressFilter(),
			NewStorageMonitorFilter(),
			NewInterfaceRemoveUuidFilter(),
			NewHostKernelFilter(),
		}
		Expect(reflect.DeepEqual(
			got.hostFilters, expectHostFilter)).To(BeTrue())

		expectPlatformNetworkFilters := []PlatformNetworkFilter{
			NewAddressPoolFilter(),
		}
		Expect(reflect.DeepEqual(
			got.platformNetworkFilters, expectPlatformNetworkFilters)).To(BeTrue())
	})

	Describe("Test parse incomplete secrets", func() {
		warningMsg := "Warning: Incomplete secret, please replace it with the secret content"
		Context("when there are incomplete TLS secrets", func() {
			It("should return secret with warning message", func() {
				fakeInput := []byte("")
				secret := v1.Secret{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "v1",
						Kind:       "Secret",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "foo",
						Namespace: "bar",
					},
					Type: v1.SecretTypeTLS,
					Data: map[string][]byte{
						v1.TLSCertKey:              fakeInput,
						v1.TLSPrivateKeyKey:        fakeInput,
						v1.ServiceAccountRootCAKey: fakeInput,
					},
				}
				got := parseIncompleteSecret(&secret)
				expect := IncompleteSecret{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "v1",
						Kind:       "Secret",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "foo",
						Namespace: "bar",
					},
					Type: v1.SecretTypeTLS,
					Data: map[string]string{
						v1.TLSCertKey:              warningMsg,
						v1.TLSPrivateKeyKey:        warningMsg,
						v1.ServiceAccountRootCAKey: warningMsg,
					},
				}
				Expect(got.TypeMeta).To(Equal(expect.TypeMeta))
				Expect(got.ObjectMeta).To(Equal(expect.ObjectMeta))
				Expect(got.Type).To(Equal(expect.Type))
				Expect(got.Data).To(Equal(expect.Data))
			})
		})

		Context("when there are incomplete bm secrets", func() {
			It("should return secret with warning password", func() {
				fakeInput := []byte("")
				fakeUser := []byte("username")
				secret := v1.Secret{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "v1",
						Kind:       "Secret",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "foo",
						Namespace: "bar",
					},
					Type: v1.SecretTypeBasicAuth,
					Data: map[string][]byte{
						v1.BasicAuthUsernameKey: []byte(fakeUser),
						v1.BasicAuthPasswordKey: fakeInput,
					},
				}
				got := parseIncompleteSecret(&secret)
				expect := IncompleteSecret{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "v1",
						Kind:       "Secret",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "foo",
						Namespace: "bar",
					},
					Type: v1.SecretTypeBasicAuth,
					Data: map[string]string{
						v1.BasicAuthUsernameKey: string([]byte(fakeUser)),
						v1.BasicAuthPasswordKey: warningMsg,
					},
				}
				Expect(got.TypeMeta).To(Equal(expect.TypeMeta))
				Expect(got.ObjectMeta).To(Equal(expect.ObjectMeta))
				Expect(got.Type).To(Equal(expect.Type))
				Expect(got.Data).To(Equal(expect.Data))
			})
		})

		Context("when there are incomplete other secrets", func() {
			It("should return secret with fake datum and error message", func() {
				fakeInput := []byte("")
				fakeUser := []byte("username")
				secret := v1.Secret{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "v1",
						Kind:       "Secret",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "foo",
						Namespace: "bar",
					},
					Type: "fake type",
					Data: map[string][]byte{
						v1.BasicAuthUsernameKey: []byte(fakeUser),
						v1.BasicAuthPasswordKey: fakeInput,
					},
				}
				got := parseIncompleteSecret(&secret)
				expect := IncompleteSecret{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "v1",
						Kind:       "Secret",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "foo",
						Namespace: "bar",
					},
					Type: "fake type",
					Data: map[string]string{
						"Fake Data": warningMsg,
					},
				}
				Expect(got.TypeMeta).To(Equal(expect.TypeMeta))
				Expect(got.ObjectMeta).To(Equal(expect.ObjectMeta))
				Expect(got.Type).To(Equal(expect.Type))
				Expect(got.Data).To(Equal(expect.Data))
			})
		})
	})

	Describe("Test NewEndpointSecretFromEnv", func() {
		Context("When NewEndpointSecretFromEnv is tested", func() {
			It("Tests NewEndpointSecretFromEnv", func() {
				name := "name"
				namespace := "namespace"
				username := os.Getenv(manager.UsernameKey)
				password := os.Getenv(manager.PasswordKey)
				want := &v1.Secret{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "v1",
						Kind:       "Secret",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      name,
						Namespace: namespace,
					},
					Type: v1.SecretTypeOpaque,
					Data: map[string][]byte{
						manager.UsernameKey: []byte(username),
						manager.PasswordKey: []byte(password),
					},
					StringData: map[string]string{
						manager.RegionNameKey:  os.Getenv(manager.RegionNameKey),
						manager.DomainNameKey:  os.Getenv(manager.DomainNameKey),
						manager.ProjectNameKey: os.Getenv(manager.ProjectNameKey),
						manager.AuthUrlKey:     os.Getenv(manager.AuthUrlKey),
						manager.InterfaceKey:   os.Getenv(manager.InterfaceKey),
					},
				}
				keystoneRegion := os.Getenv(manager.KeystoneRegionNameKey)
				if keystoneRegion != "" {
					want.StringData[manager.KeystoneRegionNameKey] = keystoneRegion
				}

				secret, err := NewEndpointSecretFromEnv(name, namespace)
				Expect(err).To(BeNil())
				Expect(secret).To(Equal(want))
			})
		})
	})

	Describe("Test buildEndpointSecret", func() {
		Context("When buildEndpointSecret is tested", func() {
			It("Tests buildEndpointSecret", func() {
				gClient := &gophercloud.ServiceClient{}
				db := &DeploymentBuilder{
					client:    gClient,
					name:      "name",
					namespace: "namespace",
				}
				dSecrets := make([]*v1.Secret, 1)

				fakeInput := []byte("")
				dSecrets[0] = &v1.Secret{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "v1",
						Kind:       "Secret",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "foo",
						Namespace: "bar",
					},
					Type: v1.SecretTypeTLS,
					Data: map[string][]byte{
						v1.TLSCertKey: fakeInput,
					},
				}
				d := &Deployment{
					Secrets: dSecrets,
				}
				name := manager.SystemEndpointSecretName
				username := os.Getenv(manager.UsernameKey)
				password := os.Getenv(manager.PasswordKey)
				secret2 := v1.Secret{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "v1",
						Kind:       "Secret",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      name,
						Namespace: db.namespace,
					},
					Type: v1.SecretTypeOpaque,
					Data: map[string][]byte{
						manager.UsernameKey: []byte(username),
						manager.PasswordKey: []byte(password),
					},
					StringData: map[string]string{
						manager.RegionNameKey:  os.Getenv(manager.RegionNameKey),
						manager.DomainNameKey:  os.Getenv(manager.DomainNameKey),
						manager.ProjectNameKey: os.Getenv(manager.ProjectNameKey),
						manager.AuthUrlKey:     os.Getenv(manager.AuthUrlKey),
						manager.InterfaceKey:   os.Getenv(manager.InterfaceKey),
					},
				}
				keystoneRegion := os.Getenv(manager.KeystoneRegionNameKey)
				if keystoneRegion != "" {
					secret2.StringData[manager.KeystoneRegionNameKey] = keystoneRegion
				}
				want := append(dSecrets, &secret2)
				err := db.buildEndpointSecret(d)
				Expect(err).To(BeNil())
				Expect(d.Secrets).To(Equal(want))
			})
		})
	})
	Describe("Test isInterfaceInUse", func() {
		Context("when bonds Members has the ifname interface", func() {
			It("It returns true indicating that interface is in use", func() {
				ifname := "ifname"
				info := &starlingxv1.InterfaceInfo{}
				bonds := make(starlingxv1.BondList, 1)
				bonds[0] = starlingxv1.BondInfo{
					Members: []string{"1", "ifname", "2"},
				}
				info.Bond = bonds
				out := isInterfaceInUse(ifname, info)
				Expect(out).To(Equal(true))
			})
		})
	})

	Describe("Test isInterfaceInUse", func() {
		Context("when VLAN lower defines the ifname interface", func() {
			It("It returns true indicating that interface is in use", func() {
				ifname := "ifname"
				info := &starlingxv1.InterfaceInfo{}
				info.VF = make(starlingxv1.VFList, 1)
				bonds := make(starlingxv1.BondList, 1)
				info.VLAN = make(starlingxv1.VLANList, 1)
				info.VLAN[0].Lower = "ifname"
				info.VF[0].Lower = "VFLower"
				bonds[0] = starlingxv1.BondInfo{
					Members: []string{"1", "2"},
				}
				info.Bond = bonds
				out := isInterfaceInUse(ifname, info)
				Expect(out).To(Equal(true))
			})
		})
	})

	Describe("Test isInterfaceInUse", func() {
		Context("when VF lower defines the ifname interface", func() {
			It("It returns true indicating that interface is in use", func() {
				ifname := "ifname"
				info := &starlingxv1.InterfaceInfo{}
				info.VF = make(starlingxv1.VFList, 1)
				bonds := make(starlingxv1.BondList, 1)
				info.VLAN = make(starlingxv1.VLANList, 1)
				info.VLAN[0].Lower = "VLANLower"
				info.VF[0].Lower = "ifname"
				bonds[0] = starlingxv1.BondInfo{
					Members: []string{"1", "2"},
				}
				info.Bond = bonds
				out := isInterfaceInUse(ifname, info)
				Expect(out).To(Equal(true))
			})
		})
	})

	Describe("Test isInterfaceInUse", func() {
		Context("when interface is not in use", func() {
			It("It returns false indicating that interface is not in use", func() {
				ifname := "ifname"
				info := &starlingxv1.InterfaceInfo{}
				info.VF = make(starlingxv1.VFList, 1)
				bonds := make(starlingxv1.BondList, 1)
				info.VLAN = make(starlingxv1.VLANList, 1)
				info.VLAN[0].Lower = "VLANLower"
				info.VF[0].Lower = "VFLower"
				bonds[0] = starlingxv1.BondInfo{
					Members: []string{"1", "2"},
				}
				info.Bond = bonds
				out := isInterfaceInUse(ifname, info)
				Expect(out).To(Equal(false))
			})
		})
	})
	Describe("Test buildNamespace", func() {
		Context("When deployment builder is with non-empty namespace", func() {
			It("Fills the deployment namespace with the deploymentbuilder namespace", func() {
				name := "fakens"
				db := &DeploymentBuilder{
					namespace: name,
				}
				d := Deployment{}
				ns := v1.Namespace{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "v1",
						Kind:       "Namespace",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name: name,
					},
				}
				err := db.buildNamespace(&d)
				Expect(err).To(BeNil())
				Expect(d.Namespace).To(Equal(ns))
			})
		})
	})

	Describe("Test buildLicenseSecret", func() {
		Context("When deployment builder is with non-empty secrets", func() {
			It("Fills the deployment secrets with the deploymentbuilder secrets", func() {
				secrets := make([]*v1.Secret, 0)
				secName := "fakeSecret"
				content := "LicenseContent"
				license := &licenses.License{
					Content: content,
				}
				ns := "fakens"
				db := &DeploymentBuilder{
					namespace: ns,
				}
				d := Deployment{
					System: starlingxv1.System{
						Spec: starlingxv1.SystemSpec{
							License: &starlingxv1.LicenseInfo{
								Secret: secName,
							},
						},
					},
					Secrets: secrets,
				}
				expSec := &v1.Secret{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "v1",
						Kind:       "Secret",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      secName,
						Namespace: ns,
					},
					Type: v1.SecretTypeOpaque,
					Data: map[string][]byte{
						"content": []byte(content),
					},
				}
				expSecrets := make([]*v1.Secret, 0)
				expSecrets = append(expSecrets, expSec)
				err := db.buildLicenseSecret(&d, license)
				Expect(err).To(BeNil())
				Expect(d.Secrets).To(Equal(expSecrets))
			})
		})
	})

	Describe("Test buildCertificateSecrets", func() {
		Context("When deployment spec has non empty certificates", func() {
			It("Adds them to the incomplete secrets in the deployment", func() {
				secrets := make([]*v1.Secret, 0)
				secName := "fakeSecret"

				ns := "fakens"
				db := &DeploymentBuilder{
					namespace: ns,
				}
				d := Deployment{
					System: starlingxv1.System{
						Spec: starlingxv1.SystemSpec{
							Certificates: &starlingxv1.CertificateList{
								{
									Secret: secName,
								},
							},
						},
					},
					Secrets: secrets,
				}
				fakeInput := []byte("")

				secret := v1.Secret{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "v1",
						Kind:       "Secret",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      secName,
						Namespace: ns,
					},
					Type: v1.SecretTypeTLS,
					Data: map[string][]byte{
						v1.TLSCertKey:              fakeInput,
						v1.TLSPrivateKeyKey:        fakeInput,
						v1.ServiceAccountRootCAKey: fakeInput,
					},
				}
				warningMsg := "Warning: Incomplete secret, please replace it with the secret content"
				expInCompleteSec := IncompleteSecret{
					TypeMeta:   secret.TypeMeta,
					ObjectMeta: secret.ObjectMeta,
					Type:       secret.Type,
					Data: map[string]string{
						v1.TLSCertKey:              warningMsg,
						v1.TLSPrivateKeyKey:        warningMsg,
						v1.ServiceAccountRootCAKey: warningMsg,
					},
				}
				expInCompleteSecrets := make([]*IncompleteSecret, 0)
				expInCompleteSecrets = append(expInCompleteSecrets, &expInCompleteSec)
				err := db.buildCertificateSecrets(&d)
				Expect(err).To(BeNil())
				Expect(d.IncompleteSecrets).To(Equal(expInCompleteSecrets))
			})
		})
	})

	Describe("Test parseIncompleteSecret", func() {
		Context("When type of secret is SecretTypeTLS", func() {
			It("Adds warning msg for keys tls.crt,tls.key and ca.crt in data", func() {
				warningMsg := "Warning: Incomplete secret, please replace it with the secret content"
				secret := &v1.Secret{
					Type: v1.SecretTypeTLS,
				}
				expInComSecret := IncompleteSecret{
					TypeMeta:   secret.TypeMeta,
					ObjectMeta: secret.ObjectMeta,
					Type:       secret.Type,
					Data: map[string]string{
						v1.TLSCertKey:              warningMsg,
						v1.TLSPrivateKeyKey:        warningMsg,
						v1.ServiceAccountRootCAKey: warningMsg,
					},
				}
				out := parseIncompleteSecret(secret)
				Expect(*out).To(Equal(expInComSecret))
			})
		})
		Context("When type of secret is SecretTypeBasicAuth", func() {
			It("Adds warning msg to username and password keys of data", func() {
				warningMsg := "Warning: Incomplete secret, please replace it with the secret content"
				secret := &v1.Secret{
					Type: v1.SecretTypeBasicAuth,
				}
				expInComSecret := IncompleteSecret{
					TypeMeta:   secret.TypeMeta,
					ObjectMeta: secret.ObjectMeta,
					Type:       secret.Type,
					Data: map[string]string{
						v1.BasicAuthUsernameKey: string(secret.Data["username"]),
						v1.BasicAuthPasswordKey: warningMsg,
					},
				}
				out := parseIncompleteSecret(secret)
				Expect(*out).To(Equal(expInComSecret))
			})
		})
		Context("When type of secret is other than SecretTypeBasicAuth and SecretTypeTLS", func() {
			It("Returns empty Data", func() {
				warningMsg := "Warning: Incomplete secret, please replace it with the secret content"
				secret := &v1.Secret{
					Type: v1.SecretTypeDockercfg,
				}
				expInComSecret := IncompleteSecret{
					TypeMeta:   secret.TypeMeta,
					ObjectMeta: secret.ObjectMeta,
					Type:       secret.Type,
					Data: map[string]string{
						"Fake Data": warningMsg,
					},
				}
				out := parseIncompleteSecret(secret)
				Expect(*out).To(Equal(expInComSecret))
			})
		})
	})

	Describe("Test buildDataNetworks", func() {
		Context("When non-empty datanetworks is present is returned by fake client", func() {
			It("Fills the deployment with fake client returned datanetworks", func() {
				ns := "fakens"
				db := &DeploymentBuilder{
					client:    gcClient.ServiceClient(),
					namespace: ns,
				}
				d := Deployment{}
				mtu := 0
				expDNs := make([]*starlingxv1.DataNetwork, 0)
				expDN := starlingxv1.DataNetwork{
					TypeMeta: metav1.TypeMeta{
						APIVersion: starlingxv1.APIVersion,
						Kind:       starlingxv1.KindDataNetwork,
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "data1",
						Namespace: ns,
						Labels: map[string]string{
							starlingxv1.ControllerToolsLabel: starlingxv1.ControllerToolsVersion,
						},
					},
					Spec: starlingxv1.DataNetworkSpec{
						MTU: &mtu,
					},
				}
				expDNs = append(expDNs, &expDN)
				err := db.buildDataNetworks(&d)
				Expect(err).To(BeNil())
				Expect(d.DataNetworks).To(Equal(expDNs))
			})
		})
	})

	Describe("Test buildPlatformNetworks", func() {
		Context("When non-empty platformnetworks is present is returned by fake client", func() {
			It("Fills the deployment with fake client returned platformnetworks", func() {
				ns := "fakens"
				name := "mgmt"
				network_type := "mgmt"
				db := &DeploymentBuilder{
					client:    gcClient.ServiceClient(),
					namespace: ns,
				}
				d := Deployment{}
				expPNs := make([]*starlingxv1.PlatformNetwork, 0)
				expPN := starlingxv1.PlatformNetwork{
					TypeMeta: metav1.TypeMeta{
						APIVersion: starlingxv1.APIVersion,
						Kind:       starlingxv1.KindPlatformNetwork,
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      name,
						Namespace: ns,
						Labels: map[string]string{
							starlingxv1.ControllerToolsLabel: starlingxv1.ControllerToolsVersion,
						},
					},
					Spec: starlingxv1.PlatformNetworkSpec{
						Dynamic:                true,
						Type:                   network_type,
						AssociatedAddressPools: []string{},
					},
				}
				expPNs = append(expPNs, &expPN)
				err := db.buildPlatformNetworks(&d)
				Expect(err).To(BeNil())
				Expect(d.PlatformNetworks).To(Equal(expPNs))
			})
		})
	})

	Describe("Test buildPTPInstances", func() {
		Context("When non-empty PTPInstances is present is returned by fake client", func() {
			It("Fills the deployment with fake client returned PTPInstances", func() {
				ns := "fakens"
				name := "phc2sys1"
				service := "phc2sys"
				db := &DeploymentBuilder{
					client:    gcClient.ServiceClient(),
					namespace: ns,
				}
				d := Deployment{}
				expPtpInsts := make([]*starlingxv1.PtpInstance, 0)
				expPtpInst := starlingxv1.PtpInstance{
					TypeMeta: metav1.TypeMeta{
						APIVersion: starlingxv1.APIVersion,
						Kind:       starlingxv1.KindPTPInstance,
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      name,
						Namespace: ns,
						Labels: map[string]string{
							starlingxv1.ControllerToolsLabel: starlingxv1.ControllerToolsVersion,
						},
					},
					Spec: starlingxv1.PtpInstanceSpec{
						Service: service,
					},
				}
				expPtpInsts = append(expPtpInsts, &expPtpInst)
				err := db.buildPTPInstances(&d)
				Expect(err).To(BeNil())
				Expect(d.PtpInstances).To(Equal(expPtpInsts))
			})
		})
	})
	Describe("Test buildPTPInterfaces", func() {
		Context("When non-empty PTPInterfaces is present is returned by fake client", func() {
			It("Fills the deployment with fake client returned PTPInterfaces", func() {
				ns := "fakens"
				name := "ptpint1"
				ptpInstanceName := "phc2sys1"
				db := &DeploymentBuilder{
					client:    gcClient.ServiceClient(),
					namespace: ns,
				}
				d := Deployment{}
				expPtpInfs := make([]*starlingxv1.PtpInterface, 0)
				expPtpInf := starlingxv1.PtpInterface{
					TypeMeta: metav1.TypeMeta{
						APIVersion: starlingxv1.APIVersion,
						Kind:       starlingxv1.KindPTPInterface,
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      name,
						Namespace: ns,
						Labels: map[string]string{
							starlingxv1.ControllerToolsLabel: starlingxv1.ControllerToolsVersion,
						},
					},
					Spec: starlingxv1.PtpInterfaceSpec{
						PtpInstance: ptpInstanceName,
					},
				}
				expPtpInfs = append(expPtpInfs, &expPtpInf)
				err := db.buildPTPInterfaces(&d)
				Expect(err).To(BeNil())
				Expect(d.PtpInterfaces).To(Equal(expPtpInfs))
			})
		})
	})
	Describe("Test simplifyHostProfiles", func() {
		Context("When deployment has non-empty profiles", func() {
			It("Simplifies the host profiles", func() {
				ns := "fakens"
				profName := "hostProfile1"
				db := &DeploymentBuilder{
					client:    gcClient.ServiceClient(),
					namespace: ns,
				}
				d := Deployment{
					Hosts: []*starlingxv1.Host{
						{
							Spec: starlingxv1.HostSpec{
								Profile: profName,
							},
						},
					},
					Profiles: []*starlingxv1.HostProfile{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name:      profName,
								Namespace: ns,
								Labels: map[string]string{
									starlingxv1.ControllerToolsLabel: starlingxv1.ControllerToolsVersion,
								},
							},
						},
					},
				}
				expProfiles := []*starlingxv1.HostProfile{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      profName,
							Namespace: ns,
							Labels: map[string]string{
								starlingxv1.ControllerToolsLabel: starlingxv1.ControllerToolsVersion,
							},
						},
					},
				}

				err := db.simplifyHostProfiles(&d)
				Expect(err).To(BeNil())
				Expect(d.Profiles).To(Equal(expProfiles))
			})
		})
	})

	Describe("Test ToYAML", func() {
		Context("When non empty resource instances are given", func() {
			It("Modifies all the instances to the yaml format", func() {
				warningMsg := "Warning: Incomplete secret, please replace it with the secret content"
				namespace := "fakens"
				fakePassword := []byte("")
				floating_address := "192.168.204.2"
				controller0_address := "192.168.204.3"
				controller1_address := "192.168.204.4"
				gateway := "192.168.204.1"
				allocation_order := "random"
				d := &Deployment{
					Namespace: v1.Namespace{
						TypeMeta: metav1.TypeMeta{
							APIVersion: "v1",
							Kind:       "namespace",
						},
						ObjectMeta: metav1.ObjectMeta{
							Name: namespace,
						},
					},
					Secrets: []*v1.Secret{
						{
							TypeMeta: metav1.TypeMeta{
								APIVersion: "v1",
								Kind:       "Secret",
							},
							ObjectMeta: metav1.ObjectMeta{
								Name:      "secret1",
								Namespace: namespace,
							},
							Type: v1.SecretTypeBasicAuth,
							Data: map[string][]byte{
								v1.BasicAuthUsernameKey: []byte("username"),
								v1.BasicAuthPasswordKey: fakePassword,
							},
						},
					},

					IncompleteSecrets: []*IncompleteSecret{
						{
							TypeMeta: metav1.TypeMeta{
								APIVersion: "v1",
								Kind:       "Secret",
							},
							ObjectMeta: metav1.ObjectMeta{
								Name:      "incomsec1",
								Namespace: "bar",
							},
							Type: "fake type",
							Data: map[string]string{
								"Fake Data": warningMsg,
							},
						},
					},
					System: starlingxv1.System{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "sys1",
							Namespace: "default",
						},
					},
					PlatformNetworks: []*starlingxv1.PlatformNetwork{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "pn1",
								Namespace: "default",
							},
							Spec: starlingxv1.PlatformNetworkSpec{
								Type: "mgmt",
							},
						},
					},
					AddressPools: []*starlingxv1.AddressPool{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "pn1",
								Namespace: "default",
							},
							Spec: starlingxv1.AddressPoolSpec{
								Subnet:             "192.168.204.0",
								FloatingAddress:    &floating_address,
								Controller0Address: &controller0_address,
								Controller1Address: &controller1_address,
								Prefix:             24,
								Gateway:            &gateway,
								Allocation: starlingxv1.AllocationInfo{
									Order:  &allocation_order,
									Ranges: []starlingxv1.AllocationRange{{Start: "192.168.204.2", End: "192.168.204.254"}},
								},
							},
						},
					},

					DataNetworks: []*starlingxv1.DataNetwork{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "dn1",
								Namespace: "default",
							},
						},
					},
					PtpInstances: []*starlingxv1.PtpInstance{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "ptpinst1",
								Namespace: "default",
							},
						},
					},
					PtpInterfaces: []*starlingxv1.PtpInterface{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "ptpinf1",
								Namespace: "default",
							},
						},
					},
					Profiles: []*starlingxv1.HostProfile{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "hostprofile1",
								Namespace: "default",
							},
						},
					},
					Hosts: []*starlingxv1.Host{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "host",
								Namespace: "default",
							},
						},
					},
				}
				out, err := d.ToYAML()
				Expect(err).To(BeNil())
				Expect(out).To(Equal(expYaml))
			})
		})
	})
})
