/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2026 Wind River Systems, Inc. */
package controller

import (
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/datanetworks"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/ptpinstances"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/ptpinterfaces"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	starlingxv1 "github.com/wind-river/cloud-platform-deployment-manager/api/v1"
	cloudManager "github.com/wind-river/cloud-platform-deployment-manager/internal/controller/manager"
)

var _ = Describe("instanceUpdateRequired", func() {
	Context("when name differs", func() {
		It("should return true", func() {
			instance := &starlingxv1.PtpInstance{
				ObjectMeta: metav1.ObjectMeta{Name: "ptp4l-instance-1"},
				Spec:       starlingxv1.PtpInstanceSpec{Service: "ptp4l"},
			}
			current := &ptpinstances.PTPInstance{Name: "ptp4l-instance-0", Service: "ptp4l"}
			Expect(instanceUpdateRequired(instance, current)).To(BeTrue())
		})
	})

	Context("when service differs", func() {
		It("should return true", func() {
			instance := &starlingxv1.PtpInstance{
				ObjectMeta: metav1.ObjectMeta{Name: "ptp4l-instance-0"},
				Spec:       starlingxv1.PtpInstanceSpec{Service: "phc2sys"},
			}
			current := &ptpinstances.PTPInstance{Name: "ptp4l-instance-0", Service: "ptp4l"}
			Expect(instanceUpdateRequired(instance, current)).To(BeTrue())
		})
	})

	Context("when all fields match", func() {
		It("should return false", func() {
			instance := &starlingxv1.PtpInstance{
				ObjectMeta: metav1.ObjectMeta{Name: "ptp4l-instance-0"},
				Spec:       starlingxv1.PtpInstanceSpec{Service: "ptp4l"},
			}
			current := &ptpinstances.PTPInstance{Name: "ptp4l-instance-0", Service: "ptp4l"}
			Expect(instanceUpdateRequired(instance, current)).To(BeFalse())
		})
	})
})

var _ = Describe("interfaceUpdateRequired", func() {
	Context("when name differs", func() {
		It("should return true", func() {
			instance := &starlingxv1.PtpInterface{
				ObjectMeta: metav1.ObjectMeta{Name: "ptp-iface-1"},
				Spec:       starlingxv1.PtpInterfaceSpec{PtpInstance: "ptp4l-instance-0"},
			}
			current := &ptpinterfaces.PTPInterface{Name: "ptp-iface-0", PTPInstanceName: "ptp4l-instance-0"}
			Expect(interfaceUpdateRequired(instance, current)).To(BeTrue())
		})
	})

	Context("when ptp instance differs", func() {
		It("should return true", func() {
			instance := &starlingxv1.PtpInterface{
				ObjectMeta: metav1.ObjectMeta{Name: "ptp-iface-0"},
				Spec:       starlingxv1.PtpInterfaceSpec{PtpInstance: "ptp4l-instance-1"},
			}
			current := &ptpinterfaces.PTPInterface{Name: "ptp-iface-0", PTPInstanceName: "ptp4l-instance-0"}
			Expect(interfaceUpdateRequired(instance, current)).To(BeTrue())
		})
	})

	Context("when all fields match", func() {
		It("should return false", func() {
			instance := &starlingxv1.PtpInterface{
				ObjectMeta: metav1.ObjectMeta{Name: "ptp-iface-0"},
				Spec:       starlingxv1.PtpInterfaceSpec{PtpInstance: "ptp4l-instance-0"},
			}
			current := &ptpinterfaces.PTPInterface{Name: "ptp-iface-0", PTPInstanceName: "ptp4l-instance-0"}
			Expect(interfaceUpdateRequired(instance, current)).To(BeFalse())
		})
	})
})

var _ = Describe("dataNetworkUpdateRequired", func() {
	var reconciler *DataNetworkReconciler

	BeforeEach(func() {
		reconciler = &DataNetworkReconciler{
			Client: k8sClient,
		}
	})

	Context("when name differs", func() {
		It("should return true", func() {
			instance := &starlingxv1.DataNetwork{
				ObjectMeta: metav1.ObjectMeta{Name: "group0-data0", Namespace: "default"},
				Spec:       starlingxv1.DataNetworkSpec{Type: "flat"},
			}
			Expect(k8sClient.Create(ctx, instance)).To(Succeed())
			current := &datanetworks.DataNetwork{Name: "group0-data1", Type: "flat"}
			_, result := dataNetworkUpdateRequired(instance, current, reconciler)
			Expect(result).To(BeTrue())
		})
	})

	Context("when type differs", func() {
		It("should return true", func() {
			instance := &starlingxv1.DataNetwork{
				ObjectMeta: metav1.ObjectMeta{Name: "group0-data1", Namespace: "default"},
				Spec:       starlingxv1.DataNetworkSpec{Type: "vlan"},
			}
			Expect(k8sClient.Create(ctx, instance)).To(Succeed())
			current := &datanetworks.DataNetwork{Name: "group0-data1", Type: "flat"}
			_, result := dataNetworkUpdateRequired(instance, current, reconciler)
			Expect(result).To(BeTrue())
		})
	})

	Context("when MTU differs", func() {
		It("should return true", func() {
			mtu := 9000
			instance := &starlingxv1.DataNetwork{
				ObjectMeta: metav1.ObjectMeta{Name: "group0-data2", Namespace: "default"},
				Spec:       starlingxv1.DataNetworkSpec{Type: "flat", MTU: &mtu},
			}
			Expect(k8sClient.Create(ctx, instance)).To(Succeed())
			current := &datanetworks.DataNetwork{Name: "group0-data2", Type: "flat", MTU: 1500}
			opts, result := dataNetworkUpdateRequired(instance, current, reconciler)
			Expect(result).To(BeTrue())
			Expect(*opts.MTU).To(Equal(9000))
		})
	})

	Context("when description differs", func() {
		It("should return true", func() {
			desc := "new description"
			instance := &starlingxv1.DataNetwork{
				ObjectMeta: metav1.ObjectMeta{Name: "group0-data3", Namespace: "default"},
				Spec:       starlingxv1.DataNetworkSpec{Type: "flat", Description: &desc},
			}
			Expect(k8sClient.Create(ctx, instance)).To(Succeed())
			current := &datanetworks.DataNetwork{Name: "group0-data3", Type: "flat", Description: "old"}
			opts, result := dataNetworkUpdateRequired(instance, current, reconciler)
			Expect(result).To(BeTrue())
			Expect(*opts.Description).To(Equal("new description"))
		})
	})

	Context("when vxlan fields differ", func() {
		It("should detect endpoint mode change", func() {
			mode := "static"
			currentMode := "dynamic"
			instance := &starlingxv1.DataNetwork{
				ObjectMeta: metav1.ObjectMeta{Name: "group0-data4", Namespace: "default"},
				Spec: starlingxv1.DataNetworkSpec{
					Type: datanetworks.TypeVxLAN,
					VxLAN: &starlingxv1.VxLANInfo{
						EndpointMode: &mode,
					},
				},
			}
			Expect(k8sClient.Create(ctx, instance)).To(Succeed())
			current := &datanetworks.DataNetwork{
				Name: "group0-data4", Type: datanetworks.TypeVxLAN, Mode: &currentMode,
			}
			opts, result := dataNetworkUpdateRequired(instance, current, reconciler)
			Expect(result).To(BeTrue())
			Expect(*opts.Mode).To(Equal("static"))
		})

		It("should detect TTL change", func() {
			ttl := 10
			currentTTL := 5
			instance := &starlingxv1.DataNetwork{
				ObjectMeta: metav1.ObjectMeta{Name: "group0-data6", Namespace: "default"},
				Spec: starlingxv1.DataNetworkSpec{
					Type:  datanetworks.TypeVxLAN,
					VxLAN: &starlingxv1.VxLANInfo{TTL: &ttl},
				},
			}
			Expect(k8sClient.Create(ctx, instance)).To(Succeed())
			current := &datanetworks.DataNetwork{
				Name: "group0-data6", Type: datanetworks.TypeVxLAN, TTL: &currentTTL,
			}
			opts, result := dataNetworkUpdateRequired(instance, current, reconciler)
			Expect(result).To(BeTrue())
			Expect(*opts.TTL).To(Equal(10))
		})

		It("should detect UDP port change", func() {
			port := 8472
			currentPort := 4789
			instance := &starlingxv1.DataNetwork{
				ObjectMeta: metav1.ObjectMeta{Name: "group0-data7", Namespace: "default"},
				Spec: starlingxv1.DataNetworkSpec{
					Type:  datanetworks.TypeVxLAN,
					VxLAN: &starlingxv1.VxLANInfo{UDPPortNumber: &port},
				},
			}
			Expect(k8sClient.Create(ctx, instance)).To(Succeed())
			current := &datanetworks.DataNetwork{
				Name: "group0-data7", Type: datanetworks.TypeVxLAN, UDPPortNumber: &currentPort,
			}
			opts, result := dataNetworkUpdateRequired(instance, current, reconciler)
			Expect(result).To(BeTrue())
			Expect(*opts.PortNumber).To(Equal(8472))
		})

		It("should detect multicast group change", func() {
			mcast := "239.0.0.1"
			currentMcast := "239.0.0.2"
			instance := &starlingxv1.DataNetwork{
				ObjectMeta: metav1.ObjectMeta{Name: "group0-data8", Namespace: "default"},
				Spec: starlingxv1.DataNetworkSpec{
					Type:  datanetworks.TypeVxLAN,
					VxLAN: &starlingxv1.VxLANInfo{MulticastGroup: &mcast},
				},
			}
			Expect(k8sClient.Create(ctx, instance)).To(Succeed())
			current := &datanetworks.DataNetwork{
				Name: "group0-data8", Type: datanetworks.TypeVxLAN, MulticastGroup: &currentMcast,
			}
			opts, result := dataNetworkUpdateRequired(instance, current, reconciler)
			Expect(result).To(BeTrue())
			Expect(*opts.MulticastGroup).To(Equal("239.0.0.1"))
		})
	})

	Context("when all fields match", func() {
		It("should return false", func() {
			instance := &starlingxv1.DataNetwork{
				ObjectMeta: metav1.ObjectMeta{Name: "group0-data5", Namespace: "default"},
				Spec:       starlingxv1.DataNetworkSpec{Type: "flat"},
			}
			Expect(k8sClient.Create(ctx, instance)).To(Succeed())
			current := &datanetworks.DataNetwork{Name: "group0-data5", Type: "flat"}
			_, result := dataNetworkUpdateRequired(instance, current, reconciler)
			Expect(result).To(BeFalse())
		})
	})
})

var _ = Describe("instanceParameterUpdateRequired", func() {
	var reconciler *PtpInstanceReconciler

	BeforeEach(func() {
		reconciler = &PtpInstanceReconciler{
			Client:       k8sClient,
			CloudManager: &cloudManager.Dummymanager{},
		}
	})

	Context("when parameters are added", func() {
		It("should return true with added map populated", func() {
			instance := &starlingxv1.PtpInstance{
				ObjectMeta: metav1.ObjectMeta{Name: "ptp4l-param-add", Namespace: "default"},
				Spec: starlingxv1.PtpInstanceSpec{
					Service: "ptp4l",
					InstanceParameters: map[string][]string{
						"global": {"domainNumber=24", "slaveOnly=1"},
					},
				},
			}
			Expect(k8sClient.Create(ctx, instance)).To(Succeed())
			current := &ptpinstances.PTPInstance{
				Name:       "ptp4l-param-add",
				Service:    "ptp4l",
				Parameters: map[string][]string{},
			}
			added, _, result := instanceParameterUpdateRequired(instance, current, reconciler)
			Expect(result).To(BeTrue())
			Expect(added).To(HaveKey("global"))
		})
	})

	Context("when parameters are removed", func() {
		It("should return true with removed map populated", func() {
			instance := &starlingxv1.PtpInstance{
				ObjectMeta: metav1.ObjectMeta{Name: "ptp4l-param-rm", Namespace: "default"},
				Spec: starlingxv1.PtpInstanceSpec{
					Service:            "ptp4l",
					InstanceParameters: map[string][]string{},
				},
			}
			Expect(k8sClient.Create(ctx, instance)).To(Succeed())
			current := &ptpinstances.PTPInstance{
				Name:    "ptp4l-param-rm",
				Service: "ptp4l",
				Parameters: map[string][]string{
					"global": {"domainNumber=24"},
				},
			}
			_, removed, result := instanceParameterUpdateRequired(instance, current, reconciler)
			Expect(result).To(BeTrue())
			Expect(removed).To(HaveKey("global"))
		})
	})

	Context("when parameters match", func() {
		It("should return false", func() {
			params := map[string][]string{"global": {"domainNumber=24"}}
			instance := &starlingxv1.PtpInstance{
				ObjectMeta: metav1.ObjectMeta{Name: "ptp4l-param-eq", Namespace: "default"},
				Spec: starlingxv1.PtpInstanceSpec{
					Service:            "ptp4l",
					InstanceParameters: params,
				},
			}
			Expect(k8sClient.Create(ctx, instance)).To(Succeed())
			current := &ptpinstances.PTPInstance{
				Name:       "ptp4l-param-eq",
				Service:    "ptp4l",
				Parameters: params,
			}
			_, _, result := instanceParameterUpdateRequired(instance, current, reconciler)
			Expect(result).To(BeFalse())
		})
	})
})

var _ = Describe("intefaceParameterUpdateRequired", func() {
	var reconciler *PtpInterfaceReconciler

	BeforeEach(func() {
		reconciler = &PtpInterfaceReconciler{
			Client:       k8sClient,
			CloudManager: &cloudManager.Dummymanager{},
		}
	})

	Context("when parameters are added", func() {
		It("should return true with added list populated", func() {
			instance := &starlingxv1.PtpInterface{
				ObjectMeta: metav1.ObjectMeta{Name: "ptp-iface-param-add", Namespace: "default"},
				Spec: starlingxv1.PtpInterfaceSpec{
					PtpInstance:         "ptp4l-instance-0",
					InterfaceParameters: []string{"masterOnly=1"},
				},
			}
			Expect(k8sClient.Create(ctx, instance)).To(Succeed())
			current := &ptpinterfaces.PTPInterface{
				Name:            "ptp-iface-param-add",
				PTPInstanceName: "ptp4l-instance-0",
				Parameters:      []string{},
			}
			added, _, result := intefaceParameterUpdateRequired(instance, current, reconciler)
			Expect(result).To(BeTrue())
			Expect(added).To(ContainElement("masterOnly=1"))
		})
	})

	Context("when parameters are removed", func() {
		It("should return true with removed list populated", func() {
			instance := &starlingxv1.PtpInterface{
				ObjectMeta: metav1.ObjectMeta{Name: "ptp-iface-param-rm", Namespace: "default"},
				Spec: starlingxv1.PtpInterfaceSpec{
					PtpInstance:         "ptp4l-instance-0",
					InterfaceParameters: []string{},
				},
			}
			Expect(k8sClient.Create(ctx, instance)).To(Succeed())
			current := &ptpinterfaces.PTPInterface{
				Name:            "ptp-iface-param-rm",
				PTPInstanceName: "ptp4l-instance-0",
				Parameters:      []string{"masterOnly=1"},
			}
			_, removed, result := intefaceParameterUpdateRequired(instance, current, reconciler)
			Expect(result).To(BeTrue())
			Expect(removed).To(ContainElement("masterOnly=1"))
		})
	})

	Context("when parameters match", func() {
		It("should return false", func() {
			instance := &starlingxv1.PtpInterface{
				ObjectMeta: metav1.ObjectMeta{Name: "ptp-iface-param-eq", Namespace: "default"},
				Spec: starlingxv1.PtpInterfaceSpec{
					PtpInstance:         "ptp4l-instance-0",
					InterfaceParameters: []string{"masterOnly=1"},
				},
			}
			Expect(k8sClient.Create(ctx, instance)).To(Succeed())
			current := &ptpinterfaces.PTPInterface{
				Name:            "ptp-iface-param-eq",
				PTPInstanceName: "ptp4l-instance-0",
				Parameters:      []string{"masterOnly=1"},
			}
			_, _, result := intefaceParameterUpdateRequired(instance, current, reconciler)
			Expect(result).To(BeFalse())
		})
	})
})

var _ = Describe("statusUpdateRequired", func() {
	Describe("DataNetwork", func() {
		var reconciler *DataNetworkReconciler

		BeforeEach(func() {
			reconciler = &DataNetworkReconciler{
				Client:       k8sClient,
				CloudManager: &cloudManager.Dummymanager{},
			}
		})

		Context("when ID changes", func() {
			It("should return true", func() {
				instance := &starlingxv1.DataNetwork{
					ObjectMeta: metav1.ObjectMeta{Name: "dn-status-id", Namespace: "default"},
					Spec:       starlingxv1.DataNetworkSpec{Type: "flat"},
				}
				network := &datanetworks.DataNetwork{ID: "new-uuid"}
				result := reconciler.statusUpdateRequired(instance, network, false)
				Expect(result).To(BeTrue())
				Expect(*instance.Status.ID).To(Equal("new-uuid"))
			})
		})

		Context("when inSync changes", func() {
			It("should return true", func() {
				instance := &starlingxv1.DataNetwork{
					ObjectMeta: metav1.ObjectMeta{Name: "dn-status-sync", Namespace: "default"},
					Spec:       starlingxv1.DataNetworkSpec{Type: "flat"},
				}
				result := reconciler.statusUpdateRequired(instance, nil, true)
				Expect(result).To(BeTrue())
				Expect(instance.Status.InSync).To(BeTrue())
			})
		})

		Context("when becoming inSync and not yet reconciled", func() {
			It("should set reconciled to true", func() {
				instance := &starlingxv1.DataNetwork{
					ObjectMeta: metav1.ObjectMeta{Name: "dn-status-recon", Namespace: "default"},
					Spec:       starlingxv1.DataNetworkSpec{Type: "flat"},
				}
				result := reconciler.statusUpdateRequired(instance, nil, true)
				Expect(result).To(BeTrue())
				Expect(instance.Status.Reconciled).To(BeTrue())
			})
		})

		Context("when nothing changes", func() {
			It("should return false", func() {
				instance := &starlingxv1.DataNetwork{
					ObjectMeta: metav1.ObjectMeta{Name: "dn-status-noop", Namespace: "default"},
					Spec:       starlingxv1.DataNetworkSpec{Type: "flat"},
				}
				result := reconciler.statusUpdateRequired(instance, nil, false)
				Expect(result).To(BeFalse())
			})
		})
	})

	Describe("PtpInstance", func() {
		var reconciler *PtpInstanceReconciler

		BeforeEach(func() {
			reconciler = &PtpInstanceReconciler{
				Client:       k8sClient,
				CloudManager: &cloudManager.Dummymanager{},
			}
		})

		Context("when ID changes", func() {
			It("should return true", func() {
				instance := &starlingxv1.PtpInstance{
					ObjectMeta: metav1.ObjectMeta{Name: "pi-status-id", Namespace: "default"},
					Spec:       starlingxv1.PtpInstanceSpec{Service: "ptp4l"},
				}
				current := &ptpinstances.PTPInstance{UUID: "new-uuid"}
				result := reconciler.statusUpdateRequired(instance, current, false)
				Expect(result).To(BeTrue())
				Expect(*instance.Status.ID).To(Equal("new-uuid"))
			})
		})

		Context("when becoming inSync and not yet reconciled", func() {
			It("should set reconciled to true", func() {
				instance := &starlingxv1.PtpInstance{
					ObjectMeta: metav1.ObjectMeta{Name: "pi-status-recon", Namespace: "default"},
					Spec:       starlingxv1.PtpInstanceSpec{Service: "ptp4l"},
				}
				result := reconciler.statusUpdateRequired(instance, nil, true)
				Expect(result).To(BeTrue())
				Expect(instance.Status.Reconciled).To(BeTrue())
			})
		})
	})

	Describe("PtpInterface", func() {
		var reconciler *PtpInterfaceReconciler

		BeforeEach(func() {
			reconciler = &PtpInterfaceReconciler{
				Client:       k8sClient,
				CloudManager: &cloudManager.Dummymanager{},
			}
		})

		Context("when ID changes", func() {
			It("should return true", func() {
				instance := &starlingxv1.PtpInterface{
					ObjectMeta: metav1.ObjectMeta{Name: "pif-status-id", Namespace: "default"},
					Spec:       starlingxv1.PtpInterfaceSpec{PtpInstance: "ptp4l-instance-0"},
				}
				current := &ptpinterfaces.PTPInterface{UUID: "new-uuid"}
				result := reconciler.statusUpdateRequired(instance, current, false)
				Expect(result).To(BeTrue())
				Expect(*instance.Status.ID).To(Equal("new-uuid"))
			})
		})

		Context("when becoming inSync and not yet reconciled", func() {
			It("should set reconciled to true", func() {
				instance := &starlingxv1.PtpInterface{
					ObjectMeta: metav1.ObjectMeta{Name: "pif-status-recon", Namespace: "default"},
					Spec:       starlingxv1.PtpInterfaceSpec{PtpInstance: "ptp4l-instance-0"},
				}
				result := reconciler.statusUpdateRequired(instance, nil, true)
				Expect(result).To(BeTrue())
				Expect(instance.Status.Reconciled).To(BeTrue())
			})
		})
	})
})
