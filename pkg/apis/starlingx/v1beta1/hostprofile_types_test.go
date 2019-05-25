/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019 Wind River Systems, Inc. */

package v1beta1

import (
	"reflect"
	"testing"

	"github.com/onsi/gomega"
	"golang.org/x/net/context"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func TestStorageHostProfile(t *testing.T) {
	key := types.NamespacedName{
		Name:      "foo",
		Namespace: "default",
	}
	created := &HostProfile{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "foo",
			Namespace: "default",
		}}
	g := gomega.NewGomegaWithT(t)

	// Test Create
	fetched := &HostProfile{}
	g.Expect(c.Create(context.TODO(), created)).NotTo(gomega.HaveOccurred())

	g.Expect(c.Get(context.TODO(), key, fetched)).NotTo(gomega.HaveOccurred())
	g.Expect(fetched).To(gomega.Equal(created))

	// Test Updating the Labels
	updated := fetched.DeepCopy()
	updated.Labels = map[string]string{"hello": "world"}
	g.Expect(c.Update(context.TODO(), updated)).NotTo(gomega.HaveOccurred())

	g.Expect(c.Get(context.TODO(), key, fetched)).NotTo(gomega.HaveOccurred())
	g.Expect(fetched).To(gomega.Equal(updated))

	// Test Delete
	g.Expect(c.Delete(context.TODO(), fetched)).NotTo(gomega.HaveOccurred())
	g.Expect(c.Get(context.TODO(), key, fetched)).To(gomega.HaveOccurred())
}

func TestEthernetList_SortByNetworkCount(t *testing.T) {
	before := EthernetList{
		EthernetInfo{
			CommonInterfaceInfo: CommonInterfaceInfo{
				Name:             "mgmt0",
				PlatformNetworks: &StringList{"a", "b", "c"},
			},
		},
		EthernetInfo{
			CommonInterfaceInfo: CommonInterfaceInfo{
				Name: "lo",
			},
		},
		EthernetInfo{
			CommonInterfaceInfo: CommonInterfaceInfo{
				Name:             "enp0s3",
				PlatformNetworks: &StringList{"d", "e"},
			},
		},
	}
	after := EthernetList{
		EthernetInfo{
			CommonInterfaceInfo: CommonInterfaceInfo{
				Name: "lo",
			},
		},
		EthernetInfo{
			CommonInterfaceInfo: CommonInterfaceInfo{
				Name:             "enp0s3",
				PlatformNetworks: &StringList{"d", "e"},
			},
		},
		EthernetInfo{
			CommonInterfaceInfo: CommonInterfaceInfo{
				Name:             "mgmt0",
				PlatformNetworks: &StringList{"a", "b", "c"},
			},
		},
	}
	tests := []struct {
		name string
		in   EthernetList
		want EthernetList
	}{
		{name: "simple",
			in:   before,
			want: after},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.in.SortByNetworkCount(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("EthernetList.SortByNetworkCount() = %v, want %v", got, tt.want)
			}
		})
	}
}
