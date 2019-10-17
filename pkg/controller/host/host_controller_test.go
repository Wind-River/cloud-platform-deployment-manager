/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019 Wind River Systems, Inc. */

package host

import (
	"context"
	"testing"
	"time"

	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/hosts"
	"github.com/onsi/gomega"
	starlingxv1 "github.com/wind-river/cloud-platform-deployment-manager/pkg/apis/starlingx/v1"
	"github.com/wind-river/cloud-platform-deployment-manager/pkg/controller/common"
	cloudManager "github.com/wind-river/cloud-platform-deployment-manager/pkg/manager"
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
	instance := &starlingxv1.Host{ObjectMeta: metav1.ObjectMeta{Name: "foo", Namespace: "default"}}

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
	defer func() { _ = c.Delete(context.TODO(), instance) }()
	g.Eventually(requests, timeout).Should(gomega.Receive(gomega.Equal(expectedRequest)))
}

func TestReconcileHost_CompareAttributes(t *testing.T) {
	type fields struct {
		Client                 client.Client
		scheme                 *runtime.Scheme
		CloudManager           cloudManager.CloudManager
		ReconcilerErrorHandler common.ReconcilerErrorHandler
		ReconcilerEventLogger  common.ReconcilerEventLogger
		hosts                  []hosts.Host
	}
	namespace := "test"
	mgr := cloudManager.NewPlatformManager(nil)
	mgr.SetSystemReady(namespace, true)
	mgr.SetSystemType(namespace, cloudManager.SystemTypeAllInOne)
	emptyProfile := starlingxv1.HostProfileSpec{
		BoardManagement: &starlingxv1.BMInfo{},
		Interfaces: &starlingxv1.InterfaceInfo{
			Ethernet: starlingxv1.EthernetList{
				starlingxv1.EthernetInfo{
					CommonInterfaceInfo: starlingxv1.CommonInterfaceInfo{},
				},
			},
		},
		Routes: starlingxv1.RouteList{
			starlingxv1.RouteInfo{},
		},
		Storage: &starlingxv1.ProfileStorageInfo{
			Monitor: &starlingxv1.MonitorInfo{},
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
	a := starlingxv1.HostProfileSpec{
		ProfileBaseAttributes: starlingxv1.ProfileBaseAttributes{
			Labels: map[string]string{
				"openstack-compute-node":  "true",
				"openstack-control-plane": "true",
			},
		},
		Memory: starlingxv1.MemoryNodeList{
			starlingxv1.MemoryNodeInfo{
				Node: 0,
				Functions: starlingxv1.MemoryFunctionList{
					starlingxv1.MemoryFunctionInfo{
						Function:  "platform",
						PageSize:  "4KB",
						PageCount: 1000,
					},
					starlingxv1.MemoryFunctionInfo{
						Function:  "vm",
						PageSize:  "4KB",
						PageCount: 2000,
					},
					starlingxv1.MemoryFunctionInfo{
						Function:  "vm",
						PageSize:  "2M",
						PageCount: 3000,
					},
					starlingxv1.MemoryFunctionInfo{
						Function:  "vswitch",
						PageSize:  "1GB",
						PageCount: 1,
					},
				},
			},
			starlingxv1.MemoryNodeInfo{
				Node: 1,
				Functions: starlingxv1.MemoryFunctionList{
					starlingxv1.MemoryFunctionInfo{
						Function:  "platform",
						PageSize:  "4KB",
						PageCount: 2000,
					},
					starlingxv1.MemoryFunctionInfo{
						Function:  "vm",
						PageSize:  "4KB",
						PageCount: 3000,
					},
					starlingxv1.MemoryFunctionInfo{
						Function:  "vm",
						PageSize:  "2M",
						PageCount: 4000,
					},
					starlingxv1.MemoryFunctionInfo{
						Function:  "vswitch",
						PageSize:  "1GB",
						PageCount: 2,
					},
				},
			},
		},
		Processors: starlingxv1.ProcessorNodeList{
			starlingxv1.ProcessorInfo{
				Node: 0,
				Functions: starlingxv1.ProcessorFunctionList{
					starlingxv1.ProcessorFunctionInfo{
						Function: "vswitch",
						Count:    0,
					},
					starlingxv1.ProcessorFunctionInfo{
						Function: "platform",
						Count:    1,
					},
				},
			},
			starlingxv1.ProcessorInfo{
				Node: 1,
				Functions: starlingxv1.ProcessorFunctionList{
					starlingxv1.ProcessorFunctionInfo{
						Function: "vswitch",
						Count:    0,
					},
					starlingxv1.ProcessorFunctionInfo{
						Function: "platform",
						Count:    1,
					},
				},
			},
		},
		Interfaces: &starlingxv1.InterfaceInfo{
			Ethernet: starlingxv1.EthernetList{
				starlingxv1.EthernetInfo{
					CommonInterfaceInfo: starlingxv1.CommonInterfaceInfo{
						Name:             "eth2",
						PlatformNetworks: &starlingxv1.StringList{"net2", "net1", "net0"},
						DataNetworks:     &starlingxv1.StringList{"data2", "data1", "data0"},
					},
				},
				starlingxv1.EthernetInfo{
					CommonInterfaceInfo: starlingxv1.CommonInterfaceInfo{
						Name:             "eth1",
						PlatformNetworks: &starlingxv1.StringList{"net2", "net1", "net0"},
						DataNetworks:     &starlingxv1.StringList{"data2", "data1", "data0"},
					},
				},
				starlingxv1.EthernetInfo{
					CommonInterfaceInfo: starlingxv1.CommonInterfaceInfo{
						Name:             "eth0",
						PlatformNetworks: &starlingxv1.StringList{"net2", "net1", "net0"},
						DataNetworks:     &starlingxv1.StringList{"data2", "data1", "data0"},
					},
				},
			},
			VLAN: starlingxv1.VLANList{
				starlingxv1.VLANInfo{
					CommonInterfaceInfo: starlingxv1.CommonInterfaceInfo{
						Name:             "vlan2",
						PlatformNetworks: &starlingxv1.StringList{"net2", "net1", "net0"},
						DataNetworks:     &starlingxv1.StringList{"data2", "data1", "data0"},
					},
					VID: 3,
				},
				starlingxv1.VLANInfo{
					CommonInterfaceInfo: starlingxv1.CommonInterfaceInfo{
						Name:             "vlan1",
						PlatformNetworks: &starlingxv1.StringList{"net2", "net1", "net0"},
						DataNetworks:     &starlingxv1.StringList{"data2", "data1", "data0"},
					},
					VID: 2,
				},
				starlingxv1.VLANInfo{
					CommonInterfaceInfo: starlingxv1.CommonInterfaceInfo{
						Name:             "vlan0",
						PlatformNetworks: &starlingxv1.StringList{"net2", "net1", "net0"},
						DataNetworks:     &starlingxv1.StringList{"data2", "data1", "data0"},
					},
					VID: 1,
				},
			},
			Bond: starlingxv1.BondList{
				starlingxv1.BondInfo{
					CommonInterfaceInfo: starlingxv1.CommonInterfaceInfo{
						Name:             "Bond2",
						PlatformNetworks: &starlingxv1.StringList{"net2", "net1", "net0"},
						DataNetworks:     &starlingxv1.StringList{"data2", "data1", "data0"},
					},
					Members: []string{"eth5", "eth4"},
				},
				starlingxv1.BondInfo{
					CommonInterfaceInfo: starlingxv1.CommonInterfaceInfo{
						Name:             "Bond1",
						PlatformNetworks: &starlingxv1.StringList{"net2", "net1", "net0"},
						DataNetworks:     &starlingxv1.StringList{"data2", "data1", "data0"},
					},
					Members: []string{"eth3", "eth2"},
				},
				starlingxv1.BondInfo{
					CommonInterfaceInfo: starlingxv1.CommonInterfaceInfo{
						Name:             "Bond0",
						PlatformNetworks: &starlingxv1.StringList{"net2", "net1", "net0"},
						DataNetworks:     &starlingxv1.StringList{"data2", "data1", "data0"},
					},
					Members: []string{"eth1", "eth0"},
				},
			},
		},
		Routes: starlingxv1.RouteList{
			starlingxv1.RouteInfo{
				Interface: "eth1",
				Network:   "2.2.3.0",
				Prefix:    28,
				Gateway:   "2.2.3.1",
			},
			starlingxv1.RouteInfo{
				Interface: "eth0",
				Network:   "1.2.3.0",
				Prefix:    24,
				Gateway:   "1.2.3.1",
			},
		},
		Addresses: starlingxv1.AddressList{
			starlingxv1.AddressInfo{
				Interface: "eth1",
				Address:   "2.2.3.10",
				Prefix:    28,
			},
			starlingxv1.AddressInfo{
				Interface: "eth0",
				Address:   "1.2.3.10",
				Prefix:    24,
			},
		},
		Storage: &starlingxv1.ProfileStorageInfo{
			OSDs: &starlingxv1.OSDList{
				starlingxv1.OSDInfo{
					ClusterName: &aClusterName2,
				},
				starlingxv1.OSDInfo{
					ClusterName: &aClusterName1,
				},
			},
			VolumeGroups: &starlingxv1.VolumeGroupList{
				starlingxv1.VolumeGroupInfo{
					LVMType:                  &aLVMType1,
					ConcurrentDiskOperations: &aConcurrentOperations1,
					PhysicalVolumes: starlingxv1.PhysicalVolumeList{
						starlingxv1.PhysicalVolumeInfo{
							Size: &aVolumeSize2,
						},
						starlingxv1.PhysicalVolumeInfo{
							Size: &aVolumeSize1,
						},
					},
				},
				starlingxv1.VolumeGroupInfo{
					LVMType:                  &aLVMType2,
					ConcurrentDiskOperations: &aConcurrentOperations2,
				},
			},
			FileSystems: &starlingxv1.FileSystemList{
				starlingxv1.FileSystemInfo{
					Name: "docker",
					Size: 20,
				},
				starlingxv1.FileSystemInfo{
					Name: "backup",
					Size: 10,
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
	b := starlingxv1.HostProfileSpec{
		ProfileBaseAttributes: starlingxv1.ProfileBaseAttributes{
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
		BoardManagement: &starlingxv1.BMInfo{
			Address: &bAddress,
			Credentials: &starlingxv1.BMCredentials{
				Password: &starlingxv1.BMPasswordInfo{
					Secret: "bmc-secret",
				},
			},
		},
		Interfaces: &starlingxv1.InterfaceInfo{
			Ethernet: starlingxv1.EthernetList{
				starlingxv1.EthernetInfo{
					CommonInterfaceInfo: starlingxv1.CommonInterfaceInfo{
						MTU: &bMTU,
					},
				},
			},
		},
		Routes: starlingxv1.RouteList{
			starlingxv1.RouteInfo{
				Metric: &bMetric,
			},
		},
		Storage: &starlingxv1.ProfileStorageInfo{
			Monitor: &starlingxv1.MonitorInfo{
				Size: &bMonitorSize,
			},
			OSDs: &starlingxv1.OSDList{
				starlingxv1.OSDInfo{
					ClusterName: &bClusterName1,
				},
			},
			VolumeGroups: &starlingxv1.VolumeGroupList{
				starlingxv1.VolumeGroupInfo{
					LVMType:                  &bLVMType1,
					ConcurrentDiskOperations: &bConcurrentOperations1,
					PhysicalVolumes: starlingxv1.PhysicalVolumeList{
						starlingxv1.PhysicalVolumeInfo{
							Size: &bVolumeSize1,
						},
					},
				},
			},
			FileSystems: &starlingxv1.FileSystemList{
				starlingxv1.FileSystemInfo{
					Name: "backup",
					Size: 10,
				},
			},
		},
	}
	c := starlingxv1.HostProfileSpec{
		ProfileBaseAttributes: starlingxv1.ProfileBaseAttributes{
			Labels: map[string]string{
				"openstack-control-plane": "true",
				"openstack-compute-node":  "true",
			},
		},
		Memory: starlingxv1.MemoryNodeList{
			starlingxv1.MemoryNodeInfo{
				Node: 1,
				Functions: starlingxv1.MemoryFunctionList{
					starlingxv1.MemoryFunctionInfo{
						Function:  "vswitch",
						PageSize:  "1GB",
						PageCount: 2,
					},
					starlingxv1.MemoryFunctionInfo{
						Function:  "vm",
						PageSize:  "2M",
						PageCount: 4000,
					},
					starlingxv1.MemoryFunctionInfo{
						Function:  "vm",
						PageSize:  "4KB",
						PageCount: 3000,
					},
					starlingxv1.MemoryFunctionInfo{
						Function:  "platform",
						PageSize:  "4KB",
						PageCount: 2000,
					},
				},
			},
			starlingxv1.MemoryNodeInfo{
				Node: 0,
				Functions: starlingxv1.MemoryFunctionList{
					starlingxv1.MemoryFunctionInfo{
						Function:  "vswitch",
						PageSize:  "1GB",
						PageCount: 1,
					},
					starlingxv1.MemoryFunctionInfo{
						Function:  "vm",
						PageSize:  "2M",
						PageCount: 3000,
					},
					starlingxv1.MemoryFunctionInfo{
						Function:  "vm",
						PageSize:  "4KB",
						PageCount: 2000,
					},
					starlingxv1.MemoryFunctionInfo{
						Function:  "platform",
						PageSize:  "4KB",
						PageCount: 1000,
					},
				},
			},
		},
		Processors: starlingxv1.ProcessorNodeList{
			starlingxv1.ProcessorInfo{
				Node: 1,
				Functions: starlingxv1.ProcessorFunctionList{
					starlingxv1.ProcessorFunctionInfo{
						Function: "platform",
						Count:    1,
					},
					starlingxv1.ProcessorFunctionInfo{
						Function: "vswitch",
						Count:    0,
					},
				},
			},
			starlingxv1.ProcessorInfo{
				Node: 0,
				Functions: starlingxv1.ProcessorFunctionList{
					starlingxv1.ProcessorFunctionInfo{
						Function: "platform",
						Count:    1,
					},
					starlingxv1.ProcessorFunctionInfo{
						Function: "vswitch",
						Count:    0,
					},
				},
			},
		},
		Interfaces: &starlingxv1.InterfaceInfo{
			Ethernet: starlingxv1.EthernetList{
				starlingxv1.EthernetInfo{
					CommonInterfaceInfo: starlingxv1.CommonInterfaceInfo{
						Name:             "eth0",
						PlatformNetworks: &starlingxv1.StringList{"net0", "net1", "net2"},
						DataNetworks:     &starlingxv1.StringList{"data0", "data1", "data2"},
					},
				},
				starlingxv1.EthernetInfo{
					CommonInterfaceInfo: starlingxv1.CommonInterfaceInfo{
						Name:             "eth1",
						PlatformNetworks: &starlingxv1.StringList{"net0", "net1", "net2"},
						DataNetworks:     &starlingxv1.StringList{"data0", "data1", "data2"},
					},
				},
				starlingxv1.EthernetInfo{
					CommonInterfaceInfo: starlingxv1.CommonInterfaceInfo{
						Name:             "eth2",
						PlatformNetworks: &starlingxv1.StringList{"net0", "net1", "net2"},
						DataNetworks:     &starlingxv1.StringList{"data0", "data1", "data2"},
					},
				},
			},
			VLAN: starlingxv1.VLANList{
				starlingxv1.VLANInfo{
					CommonInterfaceInfo: starlingxv1.CommonInterfaceInfo{
						Name:             "vlan0",
						PlatformNetworks: &starlingxv1.StringList{"net0", "net1", "net2"},
						DataNetworks:     &starlingxv1.StringList{"data0", "data1", "data2"},
					},
					VID: 1,
				},
				starlingxv1.VLANInfo{
					CommonInterfaceInfo: starlingxv1.CommonInterfaceInfo{
						Name:             "vlan1",
						PlatformNetworks: &starlingxv1.StringList{"net0", "net1", "net2"},
						DataNetworks:     &starlingxv1.StringList{"data0", "data1", "data2"},
					},
					VID: 2,
				},
				starlingxv1.VLANInfo{
					CommonInterfaceInfo: starlingxv1.CommonInterfaceInfo{
						Name:             "vlan2",
						PlatformNetworks: &starlingxv1.StringList{"net0", "net1", "net2"},
						DataNetworks:     &starlingxv1.StringList{"data0", "data1", "data2"},
					},
					VID: 3,
				},
			},
			Bond: starlingxv1.BondList{
				starlingxv1.BondInfo{
					CommonInterfaceInfo: starlingxv1.CommonInterfaceInfo{
						Name:             "Bond0",
						PlatformNetworks: &starlingxv1.StringList{"net0", "net1", "net2"},
						DataNetworks:     &starlingxv1.StringList{"data0", "data1", "data2"},
					},
					Members: []string{"eth0", "eth1"},
				},
				starlingxv1.BondInfo{
					CommonInterfaceInfo: starlingxv1.CommonInterfaceInfo{
						Name:             "Bond1",
						PlatformNetworks: &starlingxv1.StringList{"net0", "net1", "net2"},
						DataNetworks:     &starlingxv1.StringList{"data0", "data1", "data2"},
					},
					Members: []string{"eth2", "eth3"},
				},
				starlingxv1.BondInfo{
					CommonInterfaceInfo: starlingxv1.CommonInterfaceInfo{
						Name:             "Bond2",
						PlatformNetworks: &starlingxv1.StringList{"net0", "net1", "net2"},
						DataNetworks:     &starlingxv1.StringList{"data0", "data1", "data2"},
					},
					Members: []string{"eth4", "eth5"},
				},
			},
		},
		Routes: starlingxv1.RouteList{
			starlingxv1.RouteInfo{
				Interface: "eth0",
				Network:   "1.2.3.0",
				Prefix:    24,
				Gateway:   "1.2.3.1",
			},
			starlingxv1.RouteInfo{
				Interface: "eth1",
				Network:   "2.2.3.0",
				Prefix:    28,
				Gateway:   "2.2.3.1",
			},
		},
		Addresses: starlingxv1.AddressList{
			starlingxv1.AddressInfo{
				Interface: "eth0",
				Address:   "1.2.3.10",
				Prefix:    24,
			},
			starlingxv1.AddressInfo{
				Interface: "eth1",
				Address:   "2.2.3.10",
				Prefix:    28,
			},
		},
		Storage: &starlingxv1.ProfileStorageInfo{
			OSDs: &starlingxv1.OSDList{
				starlingxv1.OSDInfo{
					ClusterName: &bClusterName1,
				},
				starlingxv1.OSDInfo{
					ClusterName: &bClusterName2,
				},
			},
			VolumeGroups: &starlingxv1.VolumeGroupList{
				starlingxv1.VolumeGroupInfo{
					LVMType:                  &bLVMType1,
					ConcurrentDiskOperations: &bConcurrentOperations1,
					PhysicalVolumes: starlingxv1.PhysicalVolumeList{
						starlingxv1.PhysicalVolumeInfo{
							Size: &bVolumeSize1,
						},
						starlingxv1.PhysicalVolumeInfo{
							Size: &bVolumeSize2,
						},
					},
				},
				starlingxv1.VolumeGroupInfo{
					LVMType:                  &bLVMType2,
					ConcurrentDiskOperations: &bConcurrentOperations2,
				},
			},
			FileSystems: &starlingxv1.FileSystemList{
				starlingxv1.FileSystemInfo{
					Name: "backup",
					Size: 10,
				},
				starlingxv1.FileSystemInfo{
					Name: "docker",
					Size: 20,
				},
			},
		},
	}
	type args struct {
		in          *starlingxv1.HostProfileSpec
		other       *starlingxv1.HostProfileSpec
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
			fields: fields{CloudManager: mgr},
			args:   args{&emptyProfile, &b, namespace, "worker"},
			want:   true},
		// Check that the deepequal methods properly respect the
		// unordered-array annotation
		{name: "unordered-array",
			fields: fields{CloudManager: mgr},
			args:   args{&a, &c, namespace, "worker"},
			want:   true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &ReconcileHost{
				Client:                 tt.fields.Client,
				scheme:                 tt.fields.scheme,
				CloudManager:           tt.fields.CloudManager,
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
		CloudManager           cloudManager.CloudManager
		ReconcilerErrorHandler common.ReconcilerErrorHandler
		ReconcilerEventLogger  common.ReconcilerEventLogger
		hosts                  []hosts.Host
	}
	namespace := "test"
	mgr := cloudManager.NewPlatformManager(nil)
	mgr.SetSystemReady(namespace, true)
	mgr.SetSystemType(namespace, cloudManager.SystemTypeAllInOne)
	aStorage := starlingxv1.ProfileStorageInfo{
		OSDs: &starlingxv1.OSDList{
			starlingxv1.OSDInfo{
				Path: "/dev/path/to/some/device",
			},
		},
	}
	a := starlingxv1.HostProfileSpec{
		Storage: &aStorage,
	}
	bStorage := starlingxv1.ProfileStorageInfo{
		OSDs: &starlingxv1.OSDList{
			starlingxv1.OSDInfo{
				Path: "/dev/path/to/some/other/device",
			},
		},
	}
	b := starlingxv1.HostProfileSpec{
		Storage: &bStorage,
	}
	cStorage := starlingxv1.ProfileStorageInfo{
		FileSystems: &starlingxv1.FileSystemList{
			starlingxv1.FileSystemInfo{
				Name: "backup",
				Size: 10,
			},
		},
	}
	c := starlingxv1.HostProfileSpec{
		Storage: &cStorage,
	}
	dStorage := starlingxv1.ProfileStorageInfo{
		FileSystems: &starlingxv1.FileSystemList{
			starlingxv1.FileSystemInfo{
				Name: "backup",
				Size: 20,
			},
		},
	}
	d := starlingxv1.HostProfileSpec{
		Storage: &dStorage,
	}
	type args struct {
		in          *starlingxv1.HostProfileSpec
		other       *starlingxv1.HostProfileSpec
		namespace   string
		personality string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{name: "osds",
			fields: fields{CloudManager: mgr},
			args:   args{&a, &b, namespace, "worker"},
			want:   false},
		{name: "filesystems",
			fields: fields{CloudManager: mgr},
			args:   args{&c, &d, namespace, "worker"},
			want:   false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &ReconcileHost{
				Client:                 tt.fields.Client,
				scheme:                 tt.fields.scheme,
				CloudManager:           tt.fields.CloudManager,
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
		CloudManager           cloudManager.CloudManager
		ReconcilerErrorHandler common.ReconcilerErrorHandler
		ReconcilerEventLogger  common.ReconcilerEventLogger
		hosts                  []hosts.Host
	}
	namespace := "test"
	mgr := cloudManager.NewPlatformManager(nil)
	mgr.SetSystemReady(namespace, true)
	mgr.SetSystemType(namespace, cloudManager.SystemTypeAllInOne)
	a := starlingxv1.HostProfileSpec{
		Processors: starlingxv1.ProcessorNodeList{
			starlingxv1.ProcessorInfo{
				Node: 0,
				Functions: starlingxv1.ProcessorFunctionList{
					starlingxv1.ProcessorFunctionInfo{
						Function: "vswitch",
						Count:    1,
					},
				},
			},
		},
	}
	b := starlingxv1.HostProfileSpec{
		Processors: starlingxv1.ProcessorNodeList{
			starlingxv1.ProcessorInfo{
				Node: 0,
				Functions: starlingxv1.ProcessorFunctionList{
					starlingxv1.ProcessorFunctionInfo{
						Function: "vswitch",
						Count:    0,
					},
				},
			},
		},
	}
	type args struct {
		in          *starlingxv1.HostProfileSpec
		other       *starlingxv1.HostProfileSpec
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
			fields: fields{CloudManager: mgr},
			args:   args{&a, &b, namespace, "worker"},
			want:   false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &ReconcileHost{
				Client:                 tt.fields.Client,
				scheme:                 tt.fields.scheme,
				CloudManager:           tt.fields.CloudManager,
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
