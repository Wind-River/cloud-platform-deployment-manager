/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019-2022 Wind River Systems, Inc. */
package host

import (
	"reflect"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/addresses"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/ports"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/routes"

	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/interfaces"
	starlingxv1 "github.com/wind-river/cloud-platform-deployment-manager/api/v1"
	v1info "github.com/wind-river/cloud-platform-deployment-manager/platform"
)

var _ = Describe("Networking utils", func() {
	Describe("findConfiguredInterface utility", func() {
		Context("with interface data", func() {
			It("should find in the list", func() {

				vid10 := 10
				vid20 := 20
				vfcount1 := 1
				vfdriver1 := "netdevice"
				vfdriver2 := "vfio"
				sample := starlingxv1.HostProfileSpec{
					Interfaces: &starlingxv1.InterfaceInfo{
						Ethernet: starlingxv1.EthernetList{
							starlingxv1.EthernetInfo{
								CommonInterfaceInfo: starlingxv1.CommonInterfaceInfo{
									Name: "lo",
								},
								Port: starlingxv1.EthernetPortInfo{
									Name: "lo",
								},
							},
							starlingxv1.EthernetInfo{
								CommonInterfaceInfo: starlingxv1.CommonInterfaceInfo{
									Name: "mgmt0",
								},
								Port: starlingxv1.EthernetPortInfo{
									Name: "eth0",
								},
							},
							starlingxv1.EthernetInfo{
								CommonInterfaceInfo: starlingxv1.CommonInterfaceInfo{
									Name: "eth1",
								},
								Port: starlingxv1.EthernetPortInfo{
									Name: "eth1",
								},
							},
							starlingxv1.EthernetInfo{
								CommonInterfaceInfo: starlingxv1.CommonInterfaceInfo{
									Name: "eth2",
								},
								Port: starlingxv1.EthernetPortInfo{
									Name: "eth2",
								},
							},
							starlingxv1.EthernetInfo{
								CommonInterfaceInfo: starlingxv1.CommonInterfaceInfo{
									Name: "member3",
								},
								Port: starlingxv1.EthernetPortInfo{
									Name: "eth3",
								},
							},
							starlingxv1.EthernetInfo{
								CommonInterfaceInfo: starlingxv1.CommonInterfaceInfo{
									Name: "member4",
								},
								Port: starlingxv1.EthernetPortInfo{
									Name: "eth4",
								},
							},
							starlingxv1.EthernetInfo{
								CommonInterfaceInfo: starlingxv1.CommonInterfaceInfo{
									Name: "sriov0",
								},
								Port: starlingxv1.EthernetPortInfo{
									Name: "eth5",
								},
							},
						},
						VLAN: starlingxv1.VLANList{
							starlingxv1.VLANInfo{
								CommonInterfaceInfo: starlingxv1.CommonInterfaceInfo{
									Name: "cluster0",
								},
								Lower: "mgmt0",
								VID:   10,
							},
							starlingxv1.VLANInfo{
								CommonInterfaceInfo: starlingxv1.CommonInterfaceInfo{
									Name: "cluster0",
								},
								Lower: "bond0",
								VID:   20,
							},
						},
						Bond: starlingxv1.BondList{
							starlingxv1.BondInfo{
								CommonInterfaceInfo: starlingxv1.CommonInterfaceInfo{
									Name: "infra0",
								},
								Members: []string{"eth1", "eth2"},
							},
							starlingxv1.BondInfo{
								CommonInterfaceInfo: starlingxv1.CommonInterfaceInfo{
									Name: "bond0",
								},
								Members: []string{"member3", "member4"},
							},
						},
						VF: starlingxv1.VFList{
							starlingxv1.VFInfo{
								CommonInterfaceInfo: starlingxv1.CommonInterfaceInfo{
									Name: "sriov1",
								},
								Lower:    "sriov0",
								VFCount:  2,
								VFDriver: &vfdriver2,
							},
							starlingxv1.VFInfo{
								CommonInterfaceInfo: starlingxv1.CommonInterfaceInfo{
									Name: "sriov2",
								},
								Lower:    "sriov0",
								VFCount:  1,
								VFDriver: &vfdriver1,
							},
						},
					},
				}
				info := v1info.HostInfo{
					Ports: []ports.Port{
						{Name: "eth0",
							InterfaceID: "uuid-eth0",
						},
						{Name: "eth1",
							InterfaceID: "uuid-eth1",
						},
						{Name: "eth2",
							InterfaceID: "uuid-eth2",
						},
						{Name: "eth3",
							InterfaceID: "uuid-eth3",
						},
						{Name: "eth4",
							InterfaceID: "uuid-eth4",
						},
						{Name: "eth5",
							InterfaceID: "uuid-eth5",
						},
					},
					Interfaces: []interfaces.Interface{
						{Name: "lo",
							Type: interfaces.IFTypeVirtual,
						},
						{Name: "eth0",
							ID:   "uuid-eth0",
							Type: interfaces.IFTypeEthernet,
						},
						{Name: "eth1",
							ID:   "uuid-eth1",
							Type: interfaces.IFTypeEthernet,
						},
						{Name: "eth2",
							ID:   "uuid-eth2",
							Type: interfaces.IFTypeEthernet,
						},
						{Name: "eth3",
							ID:   "uuid-eth3",
							Type: interfaces.IFTypeEthernet,
						},
						{Name: "eth4",
							ID:   "uuid-eth4",
							Type: interfaces.IFTypeEthernet,
						},
						{Name: "sriov0",
							ID:   "uuid-eth5",
							Type: interfaces.IFTypeEthernet,
						},
						{Name: "cluster0",
							ID:   "uuid-cluster0",
							Type: interfaces.IFTypeVLAN,
							Uses: []string{"eth0"},
						},
						{Name: "vid20",
							ID:   "uuid-vid20",
							Type: interfaces.IFTypeVLAN,
							Uses: []string{"infra0"},
						},
						{Name: "infra0",
							ID:   "uuid-infra0",
							Type: interfaces.IFTypeAE,
							Uses: []string{"eth1", "eth2"},
						},
						{Name: "bond0",
							ID:   "uuid-bond0",
							Type: interfaces.IFTypeAE,
							Uses: []string{"eth3", "eth4"},
						},
						{Name: "sriov1",
							ID:   "uuid-sriov1",
							Type: interfaces.IFTypeVF,
							Uses: []string{"sriov0"},
						},
						{Name: "sriov2",
							ID:   "uuid-sriov2",
							Type: interfaces.IFTypeVF,
							Uses: []string{"sriov1"},
						},
					},
				}
				type args struct {
					profile *starlingxv1.HostProfileSpec
					iface   *interfaces.Interface
					host    *v1info.HostInfo
				}
				tests := []struct {
					name  string
					args  args
					want  *starlingxv1.CommonInterfaceInfo
					want1 bool
				}{
					{name: "find-loopback",
						args: args{
							profile: &sample,
							iface: &interfaces.Interface{
								ID:   "uuid-lo",
								Name: "lo",
								Type: interfaces.IFTypeVirtual,
							},
							host: &info,
						},
						want:  &sample.Interfaces.Ethernet[0].CommonInterfaceInfo,
						want1: true,
					},
					{name: "find-loopback-not-configured",
						args: args{
							profile: &sample,
							iface: &interfaces.Interface{
								ID:   "uuid-lo1",
								Name: "lo1",
								Type: interfaces.IFTypeVirtual,
							},
							host: &info,
						},
						want:  nil,
						want1: false,
					},
					{name: "find-ethernet",
						args: args{
							profile: &sample,
							iface: &interfaces.Interface{
								ID:   "uuid-eth1",
								Name: "eth1",
								Type: interfaces.IFTypeEthernet,
							},
							host: &info,
						},
						want:  &sample.Interfaces.Ethernet[2].CommonInterfaceInfo,
						want1: true,
					},
					{name: "find-ethernet-renamed-interface",
						args: args{
							profile: &sample,
							iface: &interfaces.Interface{
								ID:   "uuid-eth0",
								Name: "eth0",
								Type: interfaces.IFTypeEthernet,
							},
							host: &info,
						},
						want:  &sample.Interfaces.Ethernet[1].CommonInterfaceInfo,
						want1: true,
					},
					{name: "find-ethernet-not-configured",
						args: args{
							profile: &sample,
							iface: &interfaces.Interface{
								ID:   "uuid-eth9",
								Name: "eth9",
								Type: interfaces.IFTypeEthernet,
							},
							host: &info,
						},
						want:  nil,
						want1: false,
					},
					{name: "find-vlan",
						args: args{
							profile: &sample,
							iface: &interfaces.Interface{
								ID:   "uuid-cluster0",
								Name: "cluster0",
								Type: interfaces.IFTypeVLAN,
								VID:  &vid10,
								Uses: []string{"eth0"},
							},
							host: &info,
						},
						want:  &sample.Interfaces.VLAN[0].CommonInterfaceInfo,
						want1: true,
					},
					{name: "find-vlan-different-lower",
						args: args{
							profile: &sample,
							iface: &interfaces.Interface{
								ID:   "uuid-vid20",
								Name: "vid20",
								Type: interfaces.IFTypeVLAN,
								VID:  &vid20,
								Uses: []string{"infra0"},
							},
							host: &info,
						},
						want:  nil,
						want1: false,
					},
					{name: "find-bond",
						args: args{
							profile: &sample,
							iface: &interfaces.Interface{
								ID:   "uuid-infra0",
								Name: "infra0",
								Type: interfaces.IFTypeAE,
								Uses: []string{"eth1", "eth2"},
							},
							host: &info,
						},
						want:  &sample.Interfaces.Bond[0].CommonInterfaceInfo,
						want1: true,
					},
					{name: "find-bond-renamed-members",
						args: args{
							profile: &sample,
							iface: &interfaces.Interface{
								ID:   "uuid-bond0",
								Name: "bond0",
								Type: interfaces.IFTypeAE,
								Uses: []string{"eth3", "eth4"},
							},
							host: &info,
						},
						want:  &sample.Interfaces.Bond[1].CommonInterfaceInfo,
						want1: true,
					},
					{name: "find-bond-renamed-members",
						args: args{
							profile: &sample,
							iface: &interfaces.Interface{
								ID:   "uuid-bond0",
								Name: "bond0",
								Type: interfaces.IFTypeAE,
								Uses: []string{"eth3", "eth4"},
							},
							host: &info,
						},
						want:  &sample.Interfaces.Bond[1].CommonInterfaceInfo,
						want1: true,
					},
					{name: "find-bond-not-configured",
						args: args{
							profile: &sample,
							iface: &interfaces.Interface{
								ID:   "uuid-bond10",
								Name: "bond10",
								Type: interfaces.IFTypeAE,
								Uses: []string{"eth9", "eth10"},
							},
							host: &info,
						},
						want:  nil,
						want1: false,
					},
					{name: "find-vf",
						args: args{
							profile: &sample,
							iface: &interfaces.Interface{
								ID:      "uuid-sriov1",
								Name:    "sriov1",
								Type:    interfaces.IFTypeVF,
								VFCount: &vfcount1,
								Uses:    []string{"sriov0"},
							},
							host: &info,
						},
						want:  &sample.Interfaces.VF[0].CommonInterfaceInfo,
						want1: true,
					},
					{name: "find-vf-different-lower",
						args: args{
							profile: &sample,
							iface: &interfaces.Interface{
								ID:      "uuid-sriov2",
								Name:    "sriov2",
								Type:    interfaces.IFTypeVF,
								VFCount: &vfcount1,
								Uses:    []string{"sriov1"},
							},
							host: &info,
						},
						want:  nil,
						want1: false,
					},
					{name: "default",
						args: args{
							profile: &sample,
							iface: &interfaces.Interface{
								ID:      "uuid-sriov2",
								Name:    "sriov2",
								Type:    "default",
								VFCount: &vfcount1,
								Uses:    []string{"sriov1"},
							},
							host: &info,
						},
						want:  nil,
						want1: false,
					},
				}

				for _, tt := range tests {
					got, got1 := findConfiguredInterface(tt.args.iface, tt.args.profile, tt.args.host)
					Expect(reflect.DeepEqual(got, tt.want)).To(BeTrue())
					Expect(got1).To(Equal(tt.want1))
				}
			})
		})
	})

	Describe("findConfiguredRoute utility", func() {
		Context("with route data", func() {
			It("should find in the list", func() {

				metric1 := 1
				metric2 := 2
				sample := starlingxv1.HostProfileSpec{
					Routes: starlingxv1.RouteList{
						starlingxv1.RouteInfo{
							Interface: "eth0",
							Network:   "10.10.10.0",
							Prefix:    24,
							Gateway:   "10.10.10.1",
							Metric:    nil,
						},
						starlingxv1.RouteInfo{
							Interface: "eth0",
							Network:   "fd00:1::",
							Prefix:    64,
							Gateway:   "fd00:1::1",
							Metric:    nil,
						},
						starlingxv1.RouteInfo{
							Interface: "eth0",
							Network:   "11.11.11.0",
							Prefix:    24,
							Gateway:   "11.11.11.1",
							Metric:    &metric1,
						},
						starlingxv1.RouteInfo{
							Interface: "eth0",
							Network:   "fd00:11::",
							Prefix:    64,
							Gateway:   "fd00:11::1",
							Metric:    &metric2,
						},
					},
				}

				type args struct {
					route   routes.Route
					profile *starlingxv1.HostProfileSpec
				}
				tests := []struct {
					name  string
					args  args
					want  *starlingxv1.RouteInfo
					want1 bool
				}{
					{name: "find-ipv4",
						args: args{
							route: routes.Route{
								ID:            "uuid0",
								Network:       "10.10.10.0",
								Prefix:        24,
								Gateway:       "10.10.10.1",
								Metric:        1,
								InterfaceName: "eth0",
							},
							profile: &sample,
						},
						want:  &sample.Routes[0],
						want1: true,
					},
					{name: "find-ipv4-with-metric",
						args: args{
							route: routes.Route{
								ID:            "uuid2",
								Network:       "11.11.11.0",
								Prefix:        24,
								Gateway:       "11.11.11.1",
								Metric:        1,
								InterfaceName: "eth0",
							},
							profile: &sample,
						},
						want:  &sample.Routes[2],
						want1: true,
					},
					{name: "find-ipv4-with-wrong-metric",
						args: args{
							route: routes.Route{
								ID:            "uuid10",
								Network:       "11.11.11.0",
								Prefix:        24,
								Gateway:       "11.11.11.1",
								Metric:        10,
								InterfaceName: "eth0",
							},
							profile: &sample,
						},
						want:  nil,
						want1: false,
					},
					{name: "find-ipv4-with-wrong-gateway",
						args: args{
							route: routes.Route{
								ID:            "uuid10",
								Network:       "11.11.11.0",
								Prefix:        24,
								Gateway:       "11.11.11.254",
								Metric:        1,
								InterfaceName: "eth0",
							},
							profile: &sample,
						},
						want:  nil,
						want1: false,
					},
					{name: "find-ipv4-with-wrong-prefix",
						args: args{
							route: routes.Route{
								ID:            "uuid10",
								Network:       "11.11.11.0",
								Prefix:        32,
								Gateway:       "11.11.11.1",
								Metric:        1,
								InterfaceName: "eth0",
							},
							profile: &sample,
						},
						want:  nil,
						want1: false,
					},
					{name: "find-ipv4-with-wrong-interface",
						args: args{
							route: routes.Route{
								ID:            "uuid10",
								Network:       "11.11.11.0",
								Prefix:        24,
								Gateway:       "11.11.11.1",
								Metric:        1,
								InterfaceName: "eth10",
							},
							profile: &sample,
						},
						want:  nil,
						want1: false,
					},
					{name: "find-ipv6",
						args: args{
							route: routes.Route{
								ID:            "uuid1",
								Network:       "fd00:1::",
								Prefix:        64,
								Gateway:       "fd00:1::1",
								Metric:        1,
								InterfaceName: "eth0",
							},
							profile: &sample,
						},
						want:  &sample.Routes[1],
						want1: true,
					},
					{name: "find-ipv6-with-metric",
						args: args{
							route: routes.Route{
								ID:            "uuid0",
								Network:       "fd00:11::",
								Prefix:        64,
								Gateway:       "fd00:11::1",
								Metric:        2,
								InterfaceName: "eth0",
							},
							profile: &sample,
						},
						want:  &sample.Routes[3],
						want1: true,
					},
					{name: "find-ipv6-with-case-insensitive",
						args: args{
							route: routes.Route{
								ID:            "uuid0",
								Network:       "FD00:11::",
								Prefix:        64,
								Gateway:       "FD00:11::1",
								Metric:        2,
								InterfaceName: "eth0",
							},
							profile: &sample,
						},
						want:  &sample.Routes[3],
						want1: true,
					},
					{name: "find-ipv6-with-wrong-metric",
						args: args{
							route: routes.Route{
								ID:            "uuid0",
								Network:       "fd00:11::",
								Prefix:        64,
								Gateway:       "fd00:11::1",
								Metric:        20,
								InterfaceName: "eth0",
							},
							profile: &sample,
						},
						want:  nil,
						want1: false,
					},
					{name: "find-ipv6-with-wrong-gateway",
						args: args{
							route: routes.Route{
								ID:            "uuid0",
								Network:       "fd00:11::",
								Prefix:        64,
								Gateway:       "fd00:11::1111",
								Metric:        2,
								InterfaceName: "eth0",
							},
							profile: &sample,
						},
						want:  nil,
						want1: false,
					},
					{name: "find-ipv6-with-wrong-prefix",
						args: args{
							route: routes.Route{
								ID:            "uuid0",
								Network:       "fd00:11::",
								Prefix:        128,
								Gateway:       "fd00:11::1",
								Metric:        2,
								InterfaceName: "eth0",
							},
							profile: &sample,
						},
						want:  nil,
						want1: false,
					},
					{name: "find-ipv6-with-wrong-interface",
						args: args{
							route: routes.Route{
								ID:            "uuid0",
								Network:       "fd00:11::",
								Prefix:        64,
								Gateway:       "fd00:11::1",
								Metric:        2,
								InterfaceName: "eth10",
							},
							profile: &sample,
						},
						want:  nil,
						want1: false,
					},
				}
				for _, tt := range tests {
					got, got1 := findConfiguredRoute(tt.args.route, tt.args.profile)
					Expect(reflect.DeepEqual(got, tt.want)).To(BeTrue())
					Expect(got1).To(Equal(tt.want1))
				}
			})
		})
	})

	Describe("findConfiguredAddress utility", func() {
		Context("with address data", func() {
			It("should find in the list", func() {
				sample := starlingxv1.HostProfileSpec{
					Addresses: starlingxv1.AddressList{
						starlingxv1.AddressInfo{
							Interface: "eth0",
							Address:   "10.10.10.10",
							Prefix:    24,
						},
						starlingxv1.AddressInfo{
							Interface: "eth0",
							Address:   "fd00:1::10",
							Prefix:    64,
						},
						starlingxv1.AddressInfo{
							Interface: "eth0",
							Address:   "11.11.11.10",
							Prefix:    24,
						},
						starlingxv1.AddressInfo{
							Interface: "eth0",
							Address:   "fd00:11::10",
							Prefix:    64,
						},
					},
				}

				type args struct {
					Address addresses.Address
					profile *starlingxv1.HostProfileSpec
				}
				tests := []struct {
					name  string
					args  args
					want  *starlingxv1.AddressInfo
					want1 bool
				}{
					{name: "find-ipv4",
						args: args{
							Address: addresses.Address{
								ID:            "uuid0",
								Address:       "10.10.10.10",
								Prefix:        24,
								InterfaceName: "eth0",
							},
							profile: &sample,
						},
						want:  &sample.Addresses[0],
						want1: true,
					},
					{name: "find-ipv4-wrong-prefix",
						args: args{
							Address: addresses.Address{
								ID:            "uuid2",
								Address:       "11.11.11.11",
								Prefix:        32,
								InterfaceName: "eth0",
							},
							profile: &sample,
						},
						want:  nil,
						want1: false,
					},
					{name: "find-ipv6",
						args: args{
							Address: addresses.Address{
								ID:            "uuid1",
								Address:       "fd00:1::10",
								Prefix:        64,
								InterfaceName: "eth0",
							},
							profile: &sample,
						},
						want:  &sample.Addresses[1],
						want1: true,
					},
					{name: "find-ipv6-with-case-insensitive",
						args: args{
							Address: addresses.Address{
								ID:            "uuid0",
								Address:       "FD00:11::10",
								Prefix:        64,
								InterfaceName: "eth0",
							},
							profile: &sample,
						},
						want:  &sample.Addresses[3],
						want1: true,
					},
					{name: "find-ipv6-with-wrong-prefix",
						args: args{
							Address: addresses.Address{
								ID:            "uuid0",
								Address:       "fd00:11::10",
								Prefix:        128,
								InterfaceName: "eth0",
							},
							profile: &sample,
						},
						want:  nil,
						want1: false,
					},
				}
				for _, tt := range tests {
					got, got1 := findConfiguredAddress(tt.args.Address, tt.args.profile)
					Expect(reflect.DeepEqual(got, tt.want)).To(BeTrue())
					Expect(got1).To(Equal(tt.want1))
				}
			})
		})
	})
})
