/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2023-2024 Wind River Systems, Inc. */

package build

import (
	"os"
	"reflect"
	"testing"

	"github.com/gophercloud/gophercloud"
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

		var expectPlatformNetworkFilters []PlatformNetworkFilter
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
})
