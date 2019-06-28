/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019 Wind River Systems, Inc. */

package host

import (
	"context"
	"testing"
	"time"

	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/hosts"
	"github.com/onsi/gomega"
	starlingxv1beta1 "github.com/wind-river/titanium-deployment-manager/pkg/apis/starlingx/v1beta1"
	"github.com/wind-river/titanium-deployment-manager/pkg/controller/common"
	titaniumManager "github.com/wind-river/titanium-deployment-manager/pkg/manager"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var c client.Client

var expectedRequest = reconcile.Request{NamespacedName: types.NamespacedName{Name: "foo", Namespace: "default"}}

const timeout = time.Second * 5

func TestReconcile(t *testing.T) {
	g := gomega.NewGomegaWithT(t)
	instance := &starlingxv1beta1.Host{ObjectMeta: metav1.ObjectMeta{Name: "foo", Namespace: "default"}}

	// Setup the Manager and Controller.  Wrap the Controller Reconcile function so it writes each request to a
	// channel when it is finished.
	mgr, err := manager.New(cfg, manager.Options{})
	g.Expect(err).NotTo(gomega.HaveOccurred())
	c = mgr.GetClient()

	recFn, requests := SetupTestReconcile(newReconciler(mgr))
	g.Expect(add(mgr, recFn)).NotTo(gomega.HaveOccurred())

	stopMgr, mgrStopped := StartTestManager(mgr, g)

	defer func() {
		close(stopMgr)
		mgrStopped.Wait()
	}()

	// Create the Host object and expect the Reconcile and Deployment to be created
	err = c.Create(context.TODO(), instance)
	// The instance object may not be a valid object because it might be missing some required fields.
	// Please modify the instance object by adding required fields and then remove the following if statement.
	if apierrors.IsInvalid(err) {
		t.Logf("failed to create object, got an invalid object error: %v", err)
		return
	}
	g.Expect(err).NotTo(gomega.HaveOccurred())
	defer c.Delete(context.TODO(), instance)
	g.Eventually(requests, timeout).Should(gomega.Receive(gomega.Equal(expectedRequest)))
}

func TestReconcileHost_CompareAttributes(t *testing.T) {
	type fields struct {
		Client                 client.Client
		scheme                 *runtime.Scheme
		TitaniumManager        titaniumManager.TitaniumManager
		ReconcilerErrorHandler common.ReconcilerErrorHandler
		ReconcilerEventLogger  common.ReconcilerEventLogger
		hosts                  []hosts.Host
	}
	namespace := "test"
	mgr := titaniumManager.NewPlatformManager(nil)
	mgr.SetSystemReady(namespace, true)
	mgr.SetSystemType(namespace, titaniumManager.SystemTypeAllInOne)
	emptyProfile := starlingxv1beta1.HostProfileSpec{
		BoardManagement: &starlingxv1beta1.BMInfo{},
		Interfaces: &starlingxv1beta1.InterfaceInfo{
			Ethernet: starlingxv1beta1.EthernetList{
				starlingxv1beta1.EthernetInfo{
					CommonInterfaceInfo: starlingxv1beta1.CommonInterfaceInfo{},
				},
			},
		},
		Routes: starlingxv1beta1.RouteList{
			starlingxv1beta1.RouteInfo{},
		},
		Storage: &starlingxv1beta1.ProfileStorageInfo{
			Monitor: &starlingxv1beta1.MonitorInfo{},
			OSDs: starlingxv1beta1.OSDList{
				starlingxv1beta1.OSDInfo{},
			},
			VolumeGroups: starlingxv1beta1.VolumeGroupList{
				starlingxv1beta1.VolumeGroupInfo{
					PhysicalVolumes: starlingxv1beta1.PhysicalVolumeList{
						starlingxv1beta1.PhysicalVolumeInfo{},
					},
				},
			},
		},
	}
	aClusterName1 := "ceph_cluster"
	aClusterName2 := "other_cluster"
	aConcurrentOperations1 := 2
	aLVMType1 := "thin"
	aConcurrentOperations2 := 3
	aLVMType2 := "thick"
	aVolumeSize1 := 100
	aVolumeSize2 := 10
	a := starlingxv1beta1.HostProfileSpec{
		ProfileBaseAttributes: starlingxv1beta1.ProfileBaseAttributes{
			Labels: map[string]string{
				"openstack-compute-node":  "true",
				"openstack-control-plane": "true",
			},
		},
		Memory: starlingxv1beta1.MemoryNodeList{
			starlingxv1beta1.MemoryNodeInfo{
				Node: 0,
				Functions: starlingxv1beta1.MemoryFunctionList{
					starlingxv1beta1.MemoryFunctionInfo{
						Function:  "platform",
						PageSize:  "4KB",
						PageCount: 1000,
					},
					starlingxv1beta1.MemoryFunctionInfo{
						Function:  "vm",
						PageSize:  "4KB",
						PageCount: 2000,
					},
					starlingxv1beta1.MemoryFunctionInfo{
						Function:  "vm",
						PageSize:  "2M",
						PageCount: 3000,
					},
					starlingxv1beta1.MemoryFunctionInfo{
						Function:  "vswitch",
						PageSize:  "1GB",
						PageCount: 1,
					},
				},
			},
			starlingxv1beta1.MemoryNodeInfo{
				Node: 1,
				Functions: starlingxv1beta1.MemoryFunctionList{
					starlingxv1beta1.MemoryFunctionInfo{
						Function:  "platform",
						PageSize:  "4KB",
						PageCount: 2000,
					},
					starlingxv1beta1.MemoryFunctionInfo{
						Function:  "vm",
						PageSize:  "4KB",
						PageCount: 3000,
					},
					starlingxv1beta1.MemoryFunctionInfo{
						Function:  "vm",
						PageSize:  "2M",
						PageCount: 4000,
					},
					starlingxv1beta1.MemoryFunctionInfo{
						Function:  "vswitch",
						PageSize:  "1GB",
						PageCount: 2,
					},
				},
			},
		},
		Processors: starlingxv1beta1.ProcessorNodeList{
			starlingxv1beta1.ProcessorInfo{
				Node: 0,
				Functions: starlingxv1beta1.ProcessorFunctionList{
					starlingxv1beta1.ProcessorFunctionInfo{
						Function: "vswitch",
						Count:    0,
					},
					starlingxv1beta1.ProcessorFunctionInfo{
						Function: "platform",
						Count:    1,
					},
				},
			},
			starlingxv1beta1.ProcessorInfo{
				Node: 1,
				Functions: starlingxv1beta1.ProcessorFunctionList{
					starlingxv1beta1.ProcessorFunctionInfo{
						Function: "vswitch",
						Count:    0,
					},
					starlingxv1beta1.ProcessorFunctionInfo{
						Function: "platform",
						Count:    1,
					},
				},
			},
		},
		Interfaces: &starlingxv1beta1.InterfaceInfo{
			Ethernet: starlingxv1beta1.EthernetList{
				starlingxv1beta1.EthernetInfo{
					CommonInterfaceInfo: starlingxv1beta1.CommonInterfaceInfo{
						Name:             "eth2",
						PlatformNetworks: &starlingxv1beta1.StringList{"net2", "net1", "net0"},
						DataNetworks:     &starlingxv1beta1.StringList{"data2", "data1", "data0"},
					},
				},
				starlingxv1beta1.EthernetInfo{
					CommonInterfaceInfo: starlingxv1beta1.CommonInterfaceInfo{
						Name:             "eth1",
						PlatformNetworks: &starlingxv1beta1.StringList{"net2", "net1", "net0"},
						DataNetworks:     &starlingxv1beta1.StringList{"data2", "data1", "data0"},
					},
				},
				starlingxv1beta1.EthernetInfo{
					CommonInterfaceInfo: starlingxv1beta1.CommonInterfaceInfo{
						Name:             "eth0",
						PlatformNetworks: &starlingxv1beta1.StringList{"net2", "net1", "net0"},
						DataNetworks:     &starlingxv1beta1.StringList{"data2", "data1", "data0"},
					},
				},
			},
			VLAN: starlingxv1beta1.VLANList{
				starlingxv1beta1.VLANInfo{
					CommonInterfaceInfo: starlingxv1beta1.CommonInterfaceInfo{
						Name:             "vlan2",
						PlatformNetworks: &starlingxv1beta1.StringList{"net2", "net1", "net0"},
						DataNetworks:     &starlingxv1beta1.StringList{"data2", "data1", "data0"},
					},
					VID: 3,
				},
				starlingxv1beta1.VLANInfo{
					CommonInterfaceInfo: starlingxv1beta1.CommonInterfaceInfo{
						Name:             "vlan1",
						PlatformNetworks: &starlingxv1beta1.StringList{"net2", "net1", "net0"},
						DataNetworks:     &starlingxv1beta1.StringList{"data2", "data1", "data0"},
					},
					VID: 2,
				},
				starlingxv1beta1.VLANInfo{
					CommonInterfaceInfo: starlingxv1beta1.CommonInterfaceInfo{
						Name:             "vlan0",
						PlatformNetworks: &starlingxv1beta1.StringList{"net2", "net1", "net0"},
						DataNetworks:     &starlingxv1beta1.StringList{"data2", "data1", "data0"},
					},
					VID: 1,
				},
			},
			Bond: starlingxv1beta1.BondList{
				starlingxv1beta1.BondInfo{
					CommonInterfaceInfo: starlingxv1beta1.CommonInterfaceInfo{
						Name:             "Bond2",
						PlatformNetworks: &starlingxv1beta1.StringList{"net2", "net1", "net0"},
						DataNetworks:     &starlingxv1beta1.StringList{"data2", "data1", "data0"},
					},
					Members: []string{"eth5", "eth4"},
				},
				starlingxv1beta1.BondInfo{
					CommonInterfaceInfo: starlingxv1beta1.CommonInterfaceInfo{
						Name:             "Bond1",
						PlatformNetworks: &starlingxv1beta1.StringList{"net2", "net1", "net0"},
						DataNetworks:     &starlingxv1beta1.StringList{"data2", "data1", "data0"},
					},
					Members: []string{"eth3", "eth2"},
				},
				starlingxv1beta1.BondInfo{
					CommonInterfaceInfo: starlingxv1beta1.CommonInterfaceInfo{
						Name:             "Bond0",
						PlatformNetworks: &starlingxv1beta1.StringList{"net2", "net1", "net0"},
						DataNetworks:     &starlingxv1beta1.StringList{"data2", "data1", "data0"},
					},
					Members: []string{"eth1", "eth0"},
				},
			},
		},
		Routes: starlingxv1beta1.RouteList{
			starlingxv1beta1.RouteInfo{
				Interface: "eth1",
				Network:   "2.2.3.0",
				Prefix:    28,
				Gateway:   "2.2.3.1",
			},
			starlingxv1beta1.RouteInfo{
				Interface: "eth0",
				Network:   "1.2.3.0",
				Prefix:    24,
				Gateway:   "1.2.3.1",
			},
		},
		Addresses: starlingxv1beta1.AddressList{
			starlingxv1beta1.AddressInfo{
				Interface: "eth1",
				Address:   "2.2.3.10",
				Prefix:    28,
			},
			starlingxv1beta1.AddressInfo{
				Interface: "eth0",
				Address:   "1.2.3.10",
				Prefix:    24,
			},
		},
		Storage: &starlingxv1beta1.ProfileStorageInfo{
			OSDs: starlingxv1beta1.OSDList{
				starlingxv1beta1.OSDInfo{
					ClusterName: &aClusterName2,
				},
				starlingxv1beta1.OSDInfo{
					ClusterName: &aClusterName1,
				},
			},
			VolumeGroups: starlingxv1beta1.VolumeGroupList{
				starlingxv1beta1.VolumeGroupInfo{
					LVMType:                  &aLVMType1,
					ConcurrentDiskOperations: &aConcurrentOperations1,
					PhysicalVolumes: starlingxv1beta1.PhysicalVolumeList{
						starlingxv1beta1.PhysicalVolumeInfo{
							Size: &aVolumeSize2,
						},
						starlingxv1beta1.PhysicalVolumeInfo{
							Size: &aVolumeSize1,
						},
					},
				},
				starlingxv1beta1.VolumeGroupInfo{
					LVMType:                  &aLVMType2,
					ConcurrentDiskOperations: &aConcurrentOperations2,
				},
			},
		},
	}
	bClusterName1 := "ceph_cluster"
	bClusterName2 := "other_cluster"
	bAddress := "1.2.3.4"
	bConcurrentOperations1 := 2
	bLVMType1 := "thin"
	bConcurrentOperations2 := 3
	bLVMType2 := "thick"
	bMonitorSize := 20
	bMTU := 1500
	bMetric := 1
	bPersonality := "controller"
	bAdministrativeState := "locked"
	bLocation := "vbox"
	bConsole := "tty0"
	bRootDevice := "/dev/sda"
	bBootMAC := "01:02:03:04:05:06"
	bInstallOutput := "text"
	bBootDevice := "/dev/sda"
	bPowerOn := true
	bProvisioningMode := "static"
	bVolumeSize1 := 100
	bVolumeSize2 := 10
	b := starlingxv1beta1.HostProfileSpec{
		ProfileBaseAttributes: starlingxv1beta1.ProfileBaseAttributes{
			Personality:         &bPersonality,
			AdministrativeState: &bAdministrativeState,
			Location:            &bLocation,
			InstallOutput:       &bInstallOutput,
			Console:             &bConsole,
			BootDevice:          &bBootDevice,
			PowerOn:             &bPowerOn,
			ProvisioningMode:    &bProvisioningMode,
			BootMAC:             &bBootMAC,
			RootDevice:          &bRootDevice,
		},
		BoardManagement: &starlingxv1beta1.BMInfo{
			Address: &bAddress,
			Credentials: &starlingxv1beta1.BMCredentials{
				Password: &starlingxv1beta1.BMPasswordInfo{
					Secret: "bmc-secret",
				},
			},
		},
		Interfaces: &starlingxv1beta1.InterfaceInfo{
			Ethernet: starlingxv1beta1.EthernetList{
				starlingxv1beta1.EthernetInfo{
					CommonInterfaceInfo: starlingxv1beta1.CommonInterfaceInfo{
						MTU: &bMTU,
					},
				},
			},
		},
		Routes: starlingxv1beta1.RouteList{
			starlingxv1beta1.RouteInfo{
				Metric: &bMetric,
			},
		},
		Storage: &starlingxv1beta1.ProfileStorageInfo{
			Monitor: &starlingxv1beta1.MonitorInfo{
				Size: &bMonitorSize,
			},
			OSDs: starlingxv1beta1.OSDList{
				starlingxv1beta1.OSDInfo{
					ClusterName: &bClusterName1,
				},
			},
			VolumeGroups: starlingxv1beta1.VolumeGroupList{
				starlingxv1beta1.VolumeGroupInfo{
					LVMType:                  &bLVMType1,
					ConcurrentDiskOperations: &bConcurrentOperations1,
					PhysicalVolumes: starlingxv1beta1.PhysicalVolumeList{
						starlingxv1beta1.PhysicalVolumeInfo{
							Size: &bVolumeSize1,
						},
					},
				},
			},
		},
	}
	c := starlingxv1beta1.HostProfileSpec{
		ProfileBaseAttributes: starlingxv1beta1.ProfileBaseAttributes{
			Labels: map[string]string{
				"openstack-control-plane": "true",
				"openstack-compute-node":  "true",
			},
		},
		Memory: starlingxv1beta1.MemoryNodeList{
			starlingxv1beta1.MemoryNodeInfo{
				Node: 1,
				Functions: starlingxv1beta1.MemoryFunctionList{
					starlingxv1beta1.MemoryFunctionInfo{
						Function:  "vswitch",
						PageSize:  "1GB",
						PageCount: 2,
					},
					starlingxv1beta1.MemoryFunctionInfo{
						Function:  "vm",
						PageSize:  "2M",
						PageCount: 4000,
					},
					starlingxv1beta1.MemoryFunctionInfo{
						Function:  "vm",
						PageSize:  "4KB",
						PageCount: 3000,
					},
					starlingxv1beta1.MemoryFunctionInfo{
						Function:  "platform",
						PageSize:  "4KB",
						PageCount: 2000,
					},
				},
			},
			starlingxv1beta1.MemoryNodeInfo{
				Node: 0,
				Functions: starlingxv1beta1.MemoryFunctionList{
					starlingxv1beta1.MemoryFunctionInfo{
						Function:  "vswitch",
						PageSize:  "1GB",
						PageCount: 1,
					},
					starlingxv1beta1.MemoryFunctionInfo{
						Function:  "vm",
						PageSize:  "2M",
						PageCount: 3000,
					},
					starlingxv1beta1.MemoryFunctionInfo{
						Function:  "vm",
						PageSize:  "4KB",
						PageCount: 2000,
					},
					starlingxv1beta1.MemoryFunctionInfo{
						Function:  "platform",
						PageSize:  "4KB",
						PageCount: 1000,
					},
				},
			},
		},
		Processors: starlingxv1beta1.ProcessorNodeList{
			starlingxv1beta1.ProcessorInfo{
				Node: 1,
				Functions: starlingxv1beta1.ProcessorFunctionList{
					starlingxv1beta1.ProcessorFunctionInfo{
						Function: "platform",
						Count:    1,
					},
					starlingxv1beta1.ProcessorFunctionInfo{
						Function: "vswitch",
						Count:    0,
					},
				},
			},
			starlingxv1beta1.ProcessorInfo{
				Node: 0,
				Functions: starlingxv1beta1.ProcessorFunctionList{
					starlingxv1beta1.ProcessorFunctionInfo{
						Function: "platform",
						Count:    1,
					},
					starlingxv1beta1.ProcessorFunctionInfo{
						Function: "vswitch",
						Count:    0,
					},
				},
			},
		},
		Interfaces: &starlingxv1beta1.InterfaceInfo{
			Ethernet: starlingxv1beta1.EthernetList{
				starlingxv1beta1.EthernetInfo{
					CommonInterfaceInfo: starlingxv1beta1.CommonInterfaceInfo{
						Name:             "eth0",
						PlatformNetworks: &starlingxv1beta1.StringList{"net0", "net1", "net2"},
						DataNetworks:     &starlingxv1beta1.StringList{"data0", "data1", "data2"},
					},
				},
				starlingxv1beta1.EthernetInfo{
					CommonInterfaceInfo: starlingxv1beta1.CommonInterfaceInfo{
						Name:             "eth1",
						PlatformNetworks: &starlingxv1beta1.StringList{"net0", "net1", "net2"},
						DataNetworks:     &starlingxv1beta1.StringList{"data0", "data1", "data2"},
					},
				},
				starlingxv1beta1.EthernetInfo{
					CommonInterfaceInfo: starlingxv1beta1.CommonInterfaceInfo{
						Name:             "eth2",
						PlatformNetworks: &starlingxv1beta1.StringList{"net0", "net1", "net2"},
						DataNetworks:     &starlingxv1beta1.StringList{"data0", "data1", "data2"},
					},
				},
			},
			VLAN: starlingxv1beta1.VLANList{
				starlingxv1beta1.VLANInfo{
					CommonInterfaceInfo: starlingxv1beta1.CommonInterfaceInfo{
						Name:             "vlan0",
						PlatformNetworks: &starlingxv1beta1.StringList{"net0", "net1", "net2"},
						DataNetworks:     &starlingxv1beta1.StringList{"data0", "data1", "data2"},
					},
					VID: 1,
				},
				starlingxv1beta1.VLANInfo{
					CommonInterfaceInfo: starlingxv1beta1.CommonInterfaceInfo{
						Name:             "vlan1",
						PlatformNetworks: &starlingxv1beta1.StringList{"net0", "net1", "net2"},
						DataNetworks:     &starlingxv1beta1.StringList{"data0", "data1", "data2"},
					},
					VID: 2,
				},
				starlingxv1beta1.VLANInfo{
					CommonInterfaceInfo: starlingxv1beta1.CommonInterfaceInfo{
						Name:             "vlan2",
						PlatformNetworks: &starlingxv1beta1.StringList{"net0", "net1", "net2"},
						DataNetworks:     &starlingxv1beta1.StringList{"data0", "data1", "data2"},
					},
					VID: 3,
				},
			},
			Bond: starlingxv1beta1.BondList{
				starlingxv1beta1.BondInfo{
					CommonInterfaceInfo: starlingxv1beta1.CommonInterfaceInfo{
						Name:             "Bond0",
						PlatformNetworks: &starlingxv1beta1.StringList{"net0", "net1", "net2"},
						DataNetworks:     &starlingxv1beta1.StringList{"data0", "data1", "data2"},
					},
					Members: []string{"eth0", "eth1"},
				},
				starlingxv1beta1.BondInfo{
					CommonInterfaceInfo: starlingxv1beta1.CommonInterfaceInfo{
						Name:             "Bond1",
						PlatformNetworks: &starlingxv1beta1.StringList{"net0", "net1", "net2"},
						DataNetworks:     &starlingxv1beta1.StringList{"data0", "data1", "data2"},
					},
					Members: []string{"eth2", "eth3"},
				},
				starlingxv1beta1.BondInfo{
					CommonInterfaceInfo: starlingxv1beta1.CommonInterfaceInfo{
						Name:             "Bond2",
						PlatformNetworks: &starlingxv1beta1.StringList{"net0", "net1", "net2"},
						DataNetworks:     &starlingxv1beta1.StringList{"data0", "data1", "data2"},
					},
					Members: []string{"eth4", "eth5"},
				},
			},
		},
		Routes: starlingxv1beta1.RouteList{
			starlingxv1beta1.RouteInfo{
				Interface: "eth0",
				Network:   "1.2.3.0",
				Prefix:    24,
				Gateway:   "1.2.3.1",
			},
			starlingxv1beta1.RouteInfo{
				Interface: "eth1",
				Network:   "2.2.3.0",
				Prefix:    28,
				Gateway:   "2.2.3.1",
			},
		},
		Addresses: starlingxv1beta1.AddressList{
			starlingxv1beta1.AddressInfo{
				Interface: "eth0",
				Address:   "1.2.3.10",
				Prefix:    24,
			},
			starlingxv1beta1.AddressInfo{
				Interface: "eth1",
				Address:   "2.2.3.10",
				Prefix:    28,
			},
		},
		Storage: &starlingxv1beta1.ProfileStorageInfo{
			OSDs: starlingxv1beta1.OSDList{
				starlingxv1beta1.OSDInfo{
					ClusterName: &bClusterName1,
				},
				starlingxv1beta1.OSDInfo{
					ClusterName: &bClusterName2,
				},
			},
			VolumeGroups: starlingxv1beta1.VolumeGroupList{
				starlingxv1beta1.VolumeGroupInfo{
					LVMType:                  &bLVMType1,
					ConcurrentDiskOperations: &bConcurrentOperations1,
					PhysicalVolumes: starlingxv1beta1.PhysicalVolumeList{
						starlingxv1beta1.PhysicalVolumeInfo{
							Size: &bVolumeSize1,
						},
						starlingxv1beta1.PhysicalVolumeInfo{
							Size: &bVolumeSize2,
						},
					},
				},
				starlingxv1beta1.VolumeGroupInfo{
					LVMType:                  &bLVMType2,
					ConcurrentDiskOperations: &bConcurrentOperations2,
				},
			},
		},
	}
	type args struct {
		in          *starlingxv1beta1.HostProfileSpec
		other       *starlingxv1beta1.HostProfileSpec
		namespace   string
		personality string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		// Check that the deepequal methods properly respect the
		// ignore-nil-fields annotation
		{name: "ignore-nil-fields",
			fields: fields{TitaniumManager: mgr},
			args:   args{&emptyProfile, &b, namespace, "worker"},
			want:   true},
		// Check that the deepequal methods properly respect the
		// unordered-array annotation
		{name: "unordered-array",
			fields: fields{TitaniumManager: mgr},
			args:   args{&a, &c, namespace, "worker"},
			want:   true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &ReconcileHost{
				Client:                 tt.fields.Client,
				scheme:                 tt.fields.scheme,
				TitaniumManager:        tt.fields.TitaniumManager,
				ReconcilerErrorHandler: tt.fields.ReconcilerErrorHandler,
				ReconcilerEventLogger:  tt.fields.ReconcilerEventLogger,
				hosts:                  tt.fields.hosts,
			}
			if got := r.CompareAttributes(tt.args.in, tt.args.other, tt.args.namespace, tt.args.personality); got != tt.want {
				t.Errorf("CompareAttributes() = %v, want %v", got, tt.want)
			}

			if got := tt.args.in.DeepEqual(tt.args.other); got != tt.want {
				t.Errorf("CompareAttributes != DeepEqual")
			}
		})
	}
}

func TestReconcileHost_CompareEnabledAttributes(t *testing.T) {
	type fields struct {
		Client                 client.Client
		scheme                 *runtime.Scheme
		TitaniumManager        titaniumManager.TitaniumManager
		ReconcilerErrorHandler common.ReconcilerErrorHandler
		ReconcilerEventLogger  common.ReconcilerEventLogger
		hosts                  []hosts.Host
	}
	namespace := "test"
	mgr := titaniumManager.NewPlatformManager(nil)
	mgr.SetSystemReady(namespace, true)
	mgr.SetSystemType(namespace, titaniumManager.SystemTypeAllInOne)
	aStorage := starlingxv1beta1.ProfileStorageInfo{
		OSDs: starlingxv1beta1.OSDList{
			starlingxv1beta1.OSDInfo{
				Path: "/dev/path/to/some/device",
			},
		},
	}
	a := starlingxv1beta1.HostProfileSpec{
		Storage: &aStorage,
	}
	bStorage := starlingxv1beta1.ProfileStorageInfo{
		OSDs: starlingxv1beta1.OSDList{
			starlingxv1beta1.OSDInfo{
				Path: "/dev/path/to/some/other/device",
			},
		},
	}
	b := starlingxv1beta1.HostProfileSpec{
		Storage: &bStorage,
	}
	type args struct {
		in          *starlingxv1beta1.HostProfileSpec
		other       *starlingxv1beta1.HostProfileSpec
		namespace   string
		personality string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{name: "simple",
			fields: fields{TitaniumManager: mgr},
			args:   args{&a, &b, namespace, "worker"},
			want:   false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &ReconcileHost{
				Client:                 tt.fields.Client,
				scheme:                 tt.fields.scheme,
				TitaniumManager:        tt.fields.TitaniumManager,
				ReconcilerErrorHandler: tt.fields.ReconcilerErrorHandler,
				ReconcilerEventLogger:  tt.fields.ReconcilerEventLogger,
				hosts:                  tt.fields.hosts,
			}
			if got := r.CompareEnabledAttributes(tt.args.in, tt.args.other, tt.args.namespace, tt.args.personality); got != tt.want {
				t.Errorf("CompareEnabledAttributes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReconcileHost_CompareDisabledAttributes(t *testing.T) {
	type fields struct {
		Client                 client.Client
		scheme                 *runtime.Scheme
		TitaniumManager        titaniumManager.TitaniumManager
		ReconcilerErrorHandler common.ReconcilerErrorHandler
		ReconcilerEventLogger  common.ReconcilerEventLogger
		hosts                  []hosts.Host
	}
	namespace := "test"
	mgr := titaniumManager.NewPlatformManager(nil)
	mgr.SetSystemReady(namespace, true)
	mgr.SetSystemType(namespace, titaniumManager.SystemTypeAllInOne)
	a := starlingxv1beta1.HostProfileSpec{
		Processors: starlingxv1beta1.ProcessorNodeList{
			starlingxv1beta1.ProcessorInfo{
				Node: 0,
				Functions: starlingxv1beta1.ProcessorFunctionList{
					starlingxv1beta1.ProcessorFunctionInfo{
						Function: "vswitch",
						Count:    1,
					},
				},
			},
		},
	}
	b := starlingxv1beta1.HostProfileSpec{
		Processors: starlingxv1beta1.ProcessorNodeList{
			starlingxv1beta1.ProcessorInfo{
				Node: 0,
				Functions: starlingxv1beta1.ProcessorFunctionList{
					starlingxv1beta1.ProcessorFunctionInfo{
						Function: "vswitch",
						Count:    0,
					},
				},
			},
		},
	}
	type args struct {
		in          *starlingxv1beta1.HostProfileSpec
		other       *starlingxv1beta1.HostProfileSpec
		namespace   string
		personality string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{name: "simple",
			fields: fields{TitaniumManager: mgr},
			args:   args{&a, &b, namespace, "worker"},
			want:   false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &ReconcileHost{
				Client:                 tt.fields.Client,
				scheme:                 tt.fields.scheme,
				TitaniumManager:        tt.fields.TitaniumManager,
				ReconcilerErrorHandler: tt.fields.ReconcilerErrorHandler,
				ReconcilerEventLogger:  tt.fields.ReconcilerEventLogger,
				hosts:                  tt.fields.hosts,
			}
			if got := r.CompareDisabledAttributes(tt.args.in, tt.args.other, tt.args.namespace, tt.args.personality); got != tt.want {
				t.Errorf("CompareEnabledAttributes() = %v, want %v", got, tt.want)
			}
		})
	}
}
