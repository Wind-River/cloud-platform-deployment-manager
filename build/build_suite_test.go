/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2023 Wind River Systems, Inc. */

package build

import (
	"os"
	"reflect"
	"testing"

	"github.com/gophercloud/gophercloud"
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
		}
		Expect(reflect.DeepEqual(
			got.hostFilters, expectHostFilter)).To(BeTrue())
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

})
