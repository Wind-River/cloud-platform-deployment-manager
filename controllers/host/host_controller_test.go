/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019-2022 Wind River Systems, Inc. */
package host

import (
	"context"
	"reflect"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	starlingxv1 "github.com/wind-river/cloud-platform-deployment-manager/api/v1"
	comm "github.com/wind-river/cloud-platform-deployment-manager/common"
)

var _ = Describe("Host controller", func() {

	const (
		timeout  = time.Second * 10
		interval = time.Millisecond * 250
	)

	Context("Host with data", func() {
		It("Should created successfully", func() {
			ctx := context.Background()
			key := types.NamespacedName{
				Name:      "foo",
				Namespace: "default",
			}
			bootMac := "01:02:03:04:05:06"
			bmAddress := "192.168.9.9"
			match := starlingxv1.MatchInfo{
				BootMAC: &bootMac,
			}
			bmType := "bmc"
			created := &starlingxv1.Host{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: "default",
				},
				Spec: starlingxv1.HostSpec{
					Profile: "some-profile",
					Match:   &match,
					Overrides: &starlingxv1.HostProfileSpec{
						Addresses: []starlingxv1.AddressInfo{
							{Interface: "enp0s3", Address: "1.2.3.10", Prefix: 24},
						},
						BoardManagement: &starlingxv1.BMInfo{
							Type:    &bmType,
							Address: &bmAddress,
						},
					},
				}}
			Expect(k8sClient.Create(ctx, created)).To(Succeed())

			expected := created.DeepCopy()

			fetched := &starlingxv1.Host{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, key, fetched)
				return err == nil &&
					fetched.ObjectMeta.ResourceVersion != expected.ObjectMeta.ResourceVersion
			}, timeout, interval).Should(BeTrue())
			_, found := comm.ListIntersect(fetched.ObjectMeta.Finalizers, []string{HostFinalizerName})
			Expect(found).To(BeTrue())
		})
	})

	Context("Admin network multi-netting creation - day2", func() {
		Describe("hasAdminNetworkChange", func() {
			It("Should detect admin network change", func() {
				eth0Profile := starlingxv1.EthernetInfo{
					CommonInterfaceInfo: starlingxv1.CommonInterfaceInfo{
						Name:             "eth0",
						PlatformNetworks: &starlingxv1.PlatformNetworkItemList{"admin"},
					},
				}

				profileInterfaces := &starlingxv1.InterfaceInfo{
					Ethernet: []starlingxv1.EthernetInfo{eth0Profile},
				}

				eth0Current := starlingxv1.EthernetInfo{
					CommonInterfaceInfo: starlingxv1.CommonInterfaceInfo{
						Name:             "eth0",
						PlatformNetworks: &starlingxv1.PlatformNetworkItemList{},
					},
				}

				currentInterfaces := &starlingxv1.InterfaceInfo{
					Ethernet: []starlingxv1.EthernetInfo{eth0Current},
				}

				expectedEthInfo := &starlingxv1.EthernetInfo{
					CommonInterfaceInfo: starlingxv1.CommonInterfaceInfo{
						UUID:             "",
						Name:             "eth0",
						Class:            "",
						MTU:              nil,
						PlatformNetworks: &starlingxv1.PlatformNetworkItemList{"admin"},
						DataNetworks:     nil,
						PTPRole:          nil,
						PtpInterfaces:    nil,
					},
					VFCount:  nil,
					VFDriver: nil,
					Port: starlingxv1.EthernetPortInfo{
						Name: "",
					},
					Lower: "",
				}

				interfaceName, commonInterfaceInfo, hasChange := hasAdminNetworkChange(profileInterfaces, currentInterfaces)

				Expect(hasChange).To(BeTrue())
				Expect(reflect.DeepEqual(*commonInterfaceInfo, expectedEthInfo.CommonInterfaceInfo)).To(BeTrue())
				Expect(interfaceName).To(Equal("eth0"))
			})

			It("Should detect VLAN admin network change", func() {
				vlan0Profile := starlingxv1.VLANInfo{
					CommonInterfaceInfo: starlingxv1.CommonInterfaceInfo{
						Name:             "vlan0",
						PlatformNetworks: &starlingxv1.PlatformNetworkItemList{"admin"},
					},
				}

				profileInterfaces := &starlingxv1.InterfaceInfo{
					VLAN: []starlingxv1.VLANInfo{vlan0Profile},
				}

				vlan0Current := starlingxv1.VLANInfo{
					CommonInterfaceInfo: starlingxv1.CommonInterfaceInfo{
						Name:             "vlan0",
						PlatformNetworks: &starlingxv1.PlatformNetworkItemList{},
					},
				}

				currentInterfaces := &starlingxv1.InterfaceInfo{
					VLAN: []starlingxv1.VLANInfo{vlan0Current},
				}

				expectedVLANInfo := &starlingxv1.VLANInfo{
					CommonInterfaceInfo: starlingxv1.CommonInterfaceInfo{
						UUID:             "",
						Name:             "vlan0",
						Class:            "",
						MTU:              nil,
						PlatformNetworks: &starlingxv1.PlatformNetworkItemList{"admin"},
						DataNetworks:     nil,
						PTPRole:          nil,
						PtpInterfaces:    nil,
					},
					VID: 1,
				}

				interfaceName, commonInterfaceInfo, hasChange := hasAdminNetworkChange(profileInterfaces, currentInterfaces)

				Expect(hasChange).To(BeTrue())
				Expect(reflect.DeepEqual(*commonInterfaceInfo, expectedVLANInfo.CommonInterfaceInfo)).To(BeTrue())
				Expect(interfaceName).To(Equal("vlan0"))
			})

			It("Should detect Bond admin network change", func() {
				mgmt0Profile := starlingxv1.BondInfo{
					CommonInterfaceInfo: starlingxv1.CommonInterfaceInfo{
						Name:             "mgmt0",
						PlatformNetworks: &starlingxv1.PlatformNetworkItemList{"mgmt", "admin"},
					},
				}

				profileInterfaces := &starlingxv1.InterfaceInfo{
					Bond: []starlingxv1.BondInfo{mgmt0Profile},
				}

				mgmt0Current := starlingxv1.BondInfo{
					CommonInterfaceInfo: starlingxv1.CommonInterfaceInfo{
						Name:             "mgmt0",
						PlatformNetworks: &starlingxv1.PlatformNetworkItemList{"mgmt"},
					},
				}

				currentInterfaces := &starlingxv1.InterfaceInfo{
					Bond: []starlingxv1.BondInfo{mgmt0Current},
				}

				expectedBondInfo := &starlingxv1.BondInfo{
					CommonInterfaceInfo: starlingxv1.CommonInterfaceInfo{
						UUID:             "",
						Name:             "mgmt0",
						Class:            "",
						MTU:              nil,
						PlatformNetworks: &starlingxv1.PlatformNetworkItemList{"mgmt", "admin"},
						DataNetworks:     nil,
						PTPRole:          nil,
						PtpInterfaces:    nil,
					},
					Members: []string{"enp0s8", "enp0s8"},
				}

				interfaceName, commonInterfaceInfo, hasChange := hasAdminNetworkChange(profileInterfaces, currentInterfaces)

				Expect(hasChange).To(BeTrue())
				Expect(reflect.DeepEqual(*commonInterfaceInfo, expectedBondInfo.CommonInterfaceInfo)).To(BeTrue())
				Expect(interfaceName).To(Equal("mgmt0"))
			})

			It("Should not detect admin network change different interface", func() {
				eth0Profile := starlingxv1.EthernetInfo{
					CommonInterfaceInfo: starlingxv1.CommonInterfaceInfo{
						Name:             "eth0",
						PlatformNetworks: &starlingxv1.PlatformNetworkItemList{"mgmt"},
					},
				}

				eth1Profile := starlingxv1.EthernetInfo{
					CommonInterfaceInfo: starlingxv1.CommonInterfaceInfo{
						Name:             "eth1",
						PlatformNetworks: &starlingxv1.PlatformNetworkItemList{"admin"},
					},
				}

				profileInterfaces := &starlingxv1.InterfaceInfo{
					Ethernet: []starlingxv1.EthernetInfo{eth0Profile, eth1Profile},
				}
				currentInterfaces := &starlingxv1.InterfaceInfo{
					Ethernet: []starlingxv1.EthernetInfo{eth0Profile},
				}

				interfaceName, commonInterfaceInfo, hasChange := hasAdminNetworkChange(profileInterfaces, currentInterfaces)

				Expect(hasChange).To(BeFalse())
				Expect(commonInterfaceInfo).To(BeNil())
				Expect(interfaceName).To(Equal(""))
			})

			It("Should not detect admin network change w/o interface changes", func() {
				eth0Profile := starlingxv1.EthernetInfo{
					CommonInterfaceInfo: starlingxv1.CommonInterfaceInfo{
						Name:             "eth0",
						PlatformNetworks: &starlingxv1.PlatformNetworkItemList{"admin"},
					},
				}

				profileInterfaces := &starlingxv1.InterfaceInfo{
					Ethernet: []starlingxv1.EthernetInfo{eth0Profile},
				}
				currentInterfaces := &starlingxv1.InterfaceInfo{
					Ethernet: []starlingxv1.EthernetInfo{eth0Profile},
				}

				interfaceName, commonInterfaceInfo, hasChange := hasAdminNetworkChange(profileInterfaces, currentInterfaces)

				Expect(hasChange).To(BeFalse())
				Expect(commonInterfaceInfo).To(BeNil())
				Expect(interfaceName).To(Equal(""))
			})

			It("Should not detect admin network change with interface changes", func() {
				eth0Current := starlingxv1.EthernetInfo{
					CommonInterfaceInfo: starlingxv1.CommonInterfaceInfo{
						Name:             "eth0",
						PlatformNetworks: &starlingxv1.PlatformNetworkItemList{"mgmt"},
					},
				}

				eth0Profile := starlingxv1.EthernetInfo{
					CommonInterfaceInfo: starlingxv1.CommonInterfaceInfo{
						Name:             "eth0",
						PlatformNetworks: &starlingxv1.PlatformNetworkItemList{"mgmt, oam"},
					},
				}

				profileInterfaces := &starlingxv1.InterfaceInfo{
					Ethernet: []starlingxv1.EthernetInfo{eth0Profile},
				}
				currentInterfaces := &starlingxv1.InterfaceInfo{
					Ethernet: []starlingxv1.EthernetInfo{eth0Current},
				}

				interfaceName, commonInterfaceInfo, hasChange := hasAdminNetworkChange(profileInterfaces, currentInterfaces)

				Expect(hasChange).To(BeFalse())
				Expect(commonInterfaceInfo).To(BeNil())
				Expect(interfaceName).To(Equal(""))
			})
		})

		Describe("findEthernetInfoByName", func() {
			It("Should find EthernetInfo by name", func() {
				eth0Profile := starlingxv1.EthernetInfo{
					CommonInterfaceInfo: starlingxv1.CommonInterfaceInfo{
						Name: "eth0",
					},
				}
				eth1Profile := starlingxv1.EthernetInfo{
					CommonInterfaceInfo: starlingxv1.CommonInterfaceInfo{
						Name: "eth1",
					},
				}
				ethList := []starlingxv1.EthernetInfo{
					eth0Profile,
					eth1Profile,
				}
				name := "eth1"

				result := findEthernetInfoByName(ethList, name)

				Expect(result).NotTo(BeNil())
				Expect(result.Name).To(Equal(name))
			})

			It("Should not find EthernetInfo by name", func() {
				eth0Profile := starlingxv1.EthernetInfo{
					CommonInterfaceInfo: starlingxv1.CommonInterfaceInfo{
						Name: "eth0",
					},
				}
				eth1Profile := starlingxv1.EthernetInfo{
					CommonInterfaceInfo: starlingxv1.CommonInterfaceInfo{
						Name: "eth1",
					},
				}
				ethList := []starlingxv1.EthernetInfo{
					eth0Profile,
					eth1Profile,
				}
				name := "eth2"

				result := findEthernetInfoByName(ethList, name)

				Expect(result).To(BeNil())
			})
		})

		Describe("containsAdminPlatformNetwork", func() {
			It("Should contain admin platform network", func() {
				platformNetworks := starlingxv1.PlatformNetworkItemList{"admin"}
				containsAdmin := containsAdminPlatformNetwork(platformNetworks)

				Expect(containsAdmin).To(BeTrue())
			})

			It("Should not contain admin platform network", func() {
				platformNetworks := starlingxv1.PlatformNetworkItemList{"management", "data"}
				containsAdmin := containsAdminPlatformNetwork(platformNetworks)

				Expect(containsAdmin).To(BeFalse())
			})
		})
	})
})
