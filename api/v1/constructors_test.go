/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019-2023 Wind River Systems, Inc. */

package v1

import (
	"reflect"
	"strings"

	"github.com/alecthomas/units"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/addresses"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/addresspools"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/cephmonitors"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/certificates"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/controllerFilesystems"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/cpus"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/datanetworks"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/disks"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/dns"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/drbd"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/hostFilesystems"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/hosts"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/interfaceNetworks"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/interfaces"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/kernel"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/labels"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/licenses"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/memory"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/networks"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/ntp"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/osds"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/partitions"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/physicalvolumes"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/ptp"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/ptpinstances"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/ptpinterfaces"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/routes"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/serviceparameters"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/storagebackends"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/storagetiers"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/volumegroups"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	common "github.com/wind-river/cloud-platform-deployment-manager/common"
	"github.com/wind-river/cloud-platform-deployment-manager/platform"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Constructor utils for kind", func() {

	Describe("stripPartitionNumber utility", func() {
		Context("with partition and disk data", func() {
			It("should removes the \"-partNNN\" successfully", func() {
				type args struct {
					path string
				}
				tests := []struct {
					name string
					args args
					want string
				}{
					{
						name: "partition",
						args: args{"/dev/disk/by-path/pci-0000:00:0d.0-ata-1.0-part2"},
						want: "/dev/disk/by-path/pci-0000:00:0d.0-ata-1.0",
					},
					{
						name: "disk",
						args: args{"/dev/disk/by-path/pci-0000:00:0d.0-ata-1.0"},
						want: "/dev/disk/by-path/pci-0000:00:0d.0-ata-1.0",
					},
				}
				for _, tt := range tests {
					got := stripPartitionNumber(tt.args.path)
					Expect(got).To(Equal(tt.want))
				}
			})
		})
	})
	Describe("FixDevicePath utility", func() {
		Context("with path data", func() {
			It("should convert it to the newer format", func() {
				type args struct {
					path string
					host platform.HostInfo
				}
				host1 := platform.HostInfo{
					Disks: []disks.Disk{
						{DeviceNode: "/dev/sdaa",
							DevicePath: "/dev/disk/by-path/pci-0000:00:14.0-usb-0:1:1.0-scsi-0:0:0:0"},
						{DeviceNode: "/dev/sda",
							DevicePath: "/dev/disk/by-path/pci-0000:00:15.0-usb-0:1:1.0-scsi-0:0:0:0"},
						{DeviceNode: "/dev/sdbb",
							DevicePath: "/dev/disk/by-path/pci-0000:00:16.0-usb-0:1:1.0-scsi-0:0:0:0"},
						{DeviceNode: "/dev/sdb",
							DevicePath: "/dev/disk/by-path/pci-0000:00:17.0-usb-0:1:1.0-scsi-0:0:0:0"},
						{DeviceNode: "/dev/mapper/mpatha",
							DevicePath: "/dev/disk/by-id/pci-0000:00:18.0-usb-0:1:1.0-scsi-0:0:0:0"},
					},
				}
				tests := []struct {
					name string
					args args
					want string
				}{
					{name: "short-form-no-overlap",
						args: args{path: "sda", host: host1},
						want: "/dev/disk/by-path/pci-0000:00:15.0-usb-0:1:1.0-scsi-0:0:0:0"},
					{name: "short-form",
						args: args{path: "sdaa", host: host1},
						want: "/dev/disk/by-path/pci-0000:00:14.0-usb-0:1:1.0-scsi-0:0:0:0"},
					{name: "long-form-no-overlap",
						args: args{path: "/dev/sdb", host: host1},
						want: "/dev/disk/by-path/pci-0000:00:17.0-usb-0:1:1.0-scsi-0:0:0:0"},
					{name: "long-form",
						args: args{path: "/dev/sdbb", host: host1},
						want: "/dev/disk/by-path/pci-0000:00:16.0-usb-0:1:1.0-scsi-0:0:0:0"},
					{name: "already-in-new-format",
						args: args{path: "/dev/disk/by-path/pci-0000:00:16.0-usb-0:1:1.0-scsi-0:0:0:0", host: host1},
						want: "/dev/disk/by-path/pci-0000:00:16.0-usb-0:1:1.0-scsi-0:0:0:0"},
					{name: "not-found",
						args: args{path: "/dev/sdc", host: host1},
						want: "/dev/sdc"},
					{name: "imcomplete-mapper-not-found",
						args: args{path: "mpatha", host: host1},
						want: "mpatha"},
					{name: "mapper-form",
						args: args{path: "/dev/mapper/mpatha", host: host1},
						want: "/dev/disk/by-id/pci-0000:00:18.0-usb-0:1:1.0-scsi-0:0:0:0"},
					{name: "path-form",
						args: args{path: "/dev/disk/by-id/pci-0000:00:18.0-usb-0:1:1.0-scsi-0:0:0:0", host: host1},
						want: "/dev/disk/by-id/pci-0000:00:18.0-usb-0:1:1.0-scsi-0:0:0:0"},
					{name: "imcomplete-path-form-not-found",
						args: args{path: "pci-0000:00:18.0-usb-0:1:1.0-scsi-0:0:0:0", host: host1},
						want: "pci-0000:00:18.0-usb-0:1:1.0-scsi-0:0:0:0"},
				}
				for _, tt := range tests {
					got := FixDevicePath(tt.args.path, tt.args.host)
					Expect(reflect.DeepEqual(got, tt.want)).To(BeTrue())
				}
			})
		})
	})

	Describe("Test IsDefaultServiceParameter", func() {
		Context("When filtering default service parameters", func() {
			It("should return true", func() {
				tests := common.DefaultParameters
				for _, test := range tests {
					singleServiceParameter := ServiceParameterInfo{
						Service:   test.Service,
						Section:   test.Section,
						ParamName: test.ParamName,
					}
					got := IsDefaultServiceParameter(&singleServiceParameter)
					Expect(got).To(BeTrue())
				}
			})
		})
		Context("When filtering not default service parameters", func() {
			It("should return false", func() {
				singleServiceParameter := ServiceParameterInfo{
					Service:   "foo",
					Section:   "bar",
					ParamName: "foobar",
				}
				got := IsDefaultServiceParameter(&singleServiceParameter)
				Expect(got).To(BeFalse())
			})
		})
	})

	Describe("Test Megabytes", func() {
		Context("When the pagesize is PageSize1G", func() {
			var test PageSize = PageSize1G
			It("should return 1024", func() {
				want := 1024
				got := test.Megabytes()
				Expect(reflect.DeepEqual(got, want)).To(BeTrue())

			})
		})
		Context("When the pagesize is PageSize2M", func() {
			It("should return 2", func() {
				test := PageSize(PageSize2M)
				want := 2
				got := test.Megabytes()
				Expect(reflect.DeepEqual(got, want)).To(BeTrue())

			})
		})
		Context("When the pagesize is other than PageSize2M and PageSize1G", func() {
			It("should return 0", func() {
				test := PageSize("5MB")
				want := 0
				got := test.Megabytes()
				Expect(reflect.DeepEqual(got, want)).To(BeTrue())

			})
		})
	})

	Describe("Test Bytes", func() {
		Context("When the pagesize is PageSize1G", func() {
			var test PageSize = PageSize1G
			It("should return int(units.Gibibyte)", func() {
				want := int(units.Gibibyte)
				got := test.Bytes()
				Expect(reflect.DeepEqual(got, want)).To(BeTrue())

			})
		})
		Context("When the pagesize is PageSize2M", func() {
			It("should return 2 * int(units.Mebibyte)", func() {
				test := PageSize(PageSize2M)
				want := 2 * int(units.Mebibyte)
				got := test.Bytes()
				Expect(reflect.DeepEqual(got, want)).To(BeTrue())

			})
		})
		Context("When the pagesize is PageSize4K", func() {
			It("should return 4 * int(units.KiB)", func() {
				test := PageSize(PageSize4K)
				want := 4 * int(units.KiB)
				got := test.Bytes()
				Expect(reflect.DeepEqual(got, want)).To(BeTrue())

			})
		})
		Context("When the pagesize is other than PageSize2M,PageSize1G and PageSize4K", func() {
			It("should return 0", func() {
				test := PageSize("5MB")
				want := 0
				got := test.Bytes()
				Expect(reflect.DeepEqual(got, want)).To(BeTrue())

			})
		})
	})

	Describe("Test parseLabelInfo", func() {
		Context("When host labels are present", func() {
			It("should check if profile labels are same as host lables", func() {
				profile := &HostProfileSpec{}
				host := platform.HostInfo{}
				hostLabels := make([]labels.Label, 2)
				hostLabels[0].Key = "label1"
				hostLabels[0].Value = "label1"
				hostLabels[1].Key = "label2"
				hostLabels[1].Value = "label2"
				host.Labels = hostLabels
				err := parseLabelInfo(profile, host)
				Expect(err).To(BeNil())
				Expect(len(profile.Labels)).To(Equal(len(hostLabels)))

				profileLabels1 := make([]labels.Label, len(profile.Labels))
				profileLabels2 := make([]labels.Label, len(profile.Labels))

				profileLabels1[0].Key = "label1"
				profileLabels1[0].Value = "label1"
				profileLabels1[1].Key = "label2"
				profileLabels1[1].Value = "label2"

				profileLabels2[0].Key = "label2"
				profileLabels2[0].Value = "label2"
				profileLabels2[1].Key = "label1"
				profileLabels2[1].Value = "label1"

				profileLabels := make([]labels.Label, len(profile.Labels))

				itr := 0
				for j, v := range profile.Labels {
					profileLabels[itr].Key = j
					profileLabels[itr].Value = v
					itr++
				}
				flag := false
				if reflect.DeepEqual(profileLabels, profileLabels1) || reflect.DeepEqual(profileLabels, profileLabels2) {
					flag = true
				}
				Expect(flag).To(BeTrue())
			})
		})
	})

	Describe("Test parseAddressInfo", func() {
		Context("When host address is not a systemAddress", func() {
			It("should check if profile address is same as host address", func() {
				profile := &HostProfileSpec{}
				host := platform.HostInfo{}
				host.Addresses = make([]addresses.Address, 1)

				host.Addresses[0] = addresses.Address{
					InterfaceName: "InterfaceName",
					Address:       "Address",
					Prefix:        1,
				}
				err := parseAddressInfo(profile, host)
				profileAddresses := make([]addresses.Address, len(profile.Addresses))

				itr := 0
				for _, v := range profile.Addresses {
					profileAddresses[itr].Address = v.Address
					profileAddresses[itr].InterfaceName = v.Interface
					profileAddresses[itr].Prefix = v.Prefix
					itr++
				}
				Expect(err).To(BeNil())
				Expect(profileAddresses).To(Equal(host.Addresses))
			})
		})
		Context("When host address is  a systemAddress", func() {
			It("should check if profile address is nil", func() {
				profile := &HostProfileSpec{}
				host := platform.HostInfo{}
				poolID := "poolID"
				address := addresses.Address{
					InterfaceName: "InterfaceName",
					Address:       "Address",
					Prefix:        1,
					PoolUUID:      &poolID,
				}
				host.Addresses = make([]addresses.Address, 1)
				host.Addresses[0] = address
				err := parseAddressInfo(profile, host)
				Expect(err).To(BeNil())
				Expect(profile.Addresses).To(BeNil())
			})
		})
	})

	Describe("Test parseRouteInfo", func() {
		Context("When host route data is present in the host", func() {
			It("should check if profile routes are same as host routes", func() {
				profile := &HostProfileSpec{}
				host := platform.HostInfo{}
				host.Routes = make([]routes.Route, 1)
				hostRoutes := routes.Route{
					InterfaceName: "InterfaceName",
					Network:       "oam",
					Prefix:        1,
					Gateway:       "Gateway",
					Metric:        1,
				}
				host.Routes[0] = hostRoutes
				err := parseRouteInfo(profile, host)
				profileRoutes := make([]routes.Route, len(profile.Routes))

				itr := 0
				for _, v := range profile.Routes {
					profileRoutes[itr].Network = v.Network
					profileRoutes[itr].InterfaceName = v.Interface
					profileRoutes[itr].Prefix = v.Prefix
					profileRoutes[itr].Gateway = v.Gateway
					profileRoutes[itr].Metric = *v.Metric
					itr++
				}
				Expect(err).To(BeNil())
				Expect(profileRoutes).To(Equal(host.Routes))
			})
		})
	})

	Describe("Test parseProcessorInfo", func() {
		//TBD: When cpu function is application
		Context("When the cpu function not application", func() {
			It("should check if profile processors are same as host cpu data", func() {
				profile := &HostProfileSpec{}

				hostCpus := make([]cpus.CPU, 1)
				hostCpus[0].Function = cpus.CPUFunctionPlatform
				hostCpus[0].Processor = 0
				hostCpus[0].Thread = 0
				h := platform.HostInfo{
					CPU: hostCpus,
				}
				want := make([]ProcessorInfo, 1)
				data := ProcessorFunctionInfo{
					Function: strings.ToLower(hostCpus[0].Function),
					Count:    1,
				}
				want[0].Node = 0
				want[0].Functions = make([]ProcessorFunctionInfo, 0)
				want[0].Functions = append(want[0].Functions, data)
				list := ProcessorNodeList(want)
				err := parseProcessorInfo(profile, h)
				Expect(err).To(BeNil())
				Expect(profile.Processors).To(Equal(list))
			})
		})
	})

	Describe("Test NewPlatformNetworkSpec", func() {
		Context("When network dynamic is false", func() {
			It("should give allocation type as AllocationOrderStatic", func() {
				gateway := "gateway"
				ranges := make([]AllocationRange, 0)
				obj := AllocationRange{
					Start: "192.168.1.100",
					End:   "192.168.1.200",
				}
				ranges = append(ranges, obj)
				obj = AllocationRange{
					Start: "10.0.0.1",
					End:   "10.0.0.255",
				}
				ranges = append(ranges, obj)
				poolRanges := make([][]string, 2)
				poolRanges[0] = []string{"192.168.1.100", "192.168.1.200"}
				poolRanges[1] = []string{"10.0.0.1", "10.0.0.255"}

				pool := addresspools.AddressPool{
					Network:            "192.168.1.0/24",
					FloatingAddress:    "192.168.1.255",
					Controller0Address: "192.168.1.1",
					Controller1Address: "192.168.1.2",
					Prefix:             24,
					Gateway:            &gateway,
					Order:              "random",
					Ranges:             poolRanges,
				}

				network := networks.Network{
					Dynamic: false,
					Type:    "mgmt",
				}
				expSpec := PlatformNetworkSpec{
					Type:               "mgmt",
					Subnet:             "192.168.1.0/24",
					FloatingAddress:    "192.168.1.255",
					Controller0Address: "192.168.1.1",
					Controller1Address: "192.168.1.2",
					Prefix:             24,
					Gateway:            &gateway,
					Allocation: AllocationInfo{
						Type:   networks.AllocationOrderStatic,
						Order:  &pool.Order,
						Ranges: ranges,
					},
				}
				pnSpec, err := NewPlatformNetworkSpec(pool, network)
				Expect(err).To(BeNil())
				Expect(*pnSpec).To(Equal(expSpec))
			})
		})
	})

	Describe("Test NewPlatformNetwork", func() {
		Context("When network dynamic is false", func() {
			It("should give allocation type as AllocationOrderStatic", func() {
				namespace := "NameSpace"
				gateway := "192.168.1.1"
				ranges := make([]AllocationRange, 0)
				obj := AllocationRange{
					Start: "192.168.1.100",
					End:   "192.168.1.200",
				}
				ranges = append(ranges, obj)
				obj = AllocationRange{
					Start: "10.0.0.1",
					End:   "10.0.0.255",
				}
				ranges = append(ranges, obj)
				poolRanges := make([][]string, 2)
				poolRanges[0] = []string{"192.168.1.100", "192.168.1.200"}
				poolRanges[1] = []string{"10.0.0.1", "10.0.0.255"}

				pool := addresspools.AddressPool{
					Network:            "192.168.1.0/24",
					FloatingAddress:    "192.168.1.255",
					Controller0Address: "192.168.1.1",
					Controller1Address: "192.168.1.2",
					Prefix:             24,
					Gateway:            &gateway,
					Order:              "random",
					Ranges:             poolRanges,
				}

				network := networks.Network{
					Name:    "NetworkName",
					Dynamic: false,
					Type:    "mgmt",
				}
				expSpec := PlatformNetworkSpec{
					Type:               "mgmt",
					Subnet:             "192.168.1.0/24",
					FloatingAddress:    "192.168.1.255",
					Controller0Address: "192.168.1.1",
					Controller1Address: "192.168.1.2",
					Prefix:             24,
					Gateway:            &gateway,
					Allocation: AllocationInfo{
						Type:   networks.AllocationOrderStatic,
						Order:  &pool.Order,
						Ranges: ranges,
					},
				}
				expPn := PlatformNetwork{
					TypeMeta: metav1.TypeMeta{
						APIVersion: APIVersion,
						Kind:       KindPlatformNetwork,
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      network.Name,
						Namespace: namespace,
						Labels: map[string]string{
							ControllerToolsLabel: ControllerToolsVersion,
						},
					},
					Spec: expSpec,
				}
				pn, err := NewPlatformNetwork(namespace, pool, network)
				Expect(err).To(BeNil())
				Expect(*pn).To(Equal(expPn))
			})
		})
	})

	Describe("Test NewPtpInterfaceSpec", func() {
		Context("When params are not nil", func() {
			It("should give spec InterfaceParams as PTPInterface params", func() {
				params := []string{"param1", "param2"}
				PTPint := ptpinterfaces.PTPInterface{
					PTPInstanceName: "PTPInstanceName",
					Parameters:      params,
				}

				expSpec := PtpInterfaceSpec{
					PtpInstance:         PTPint.PTPInstanceName,
					InterfaceParameters: params,
				}

				ptpSpec, err := NewPtpInterfaceSpec(PTPint)
				Expect(err).To(BeNil())
				Expect(*ptpSpec).To(Equal(expSpec))
			})
		})
	})

	Describe("Test NewPTPInstance", func() {
		Context("When params are not nil", func() {
			It("should give spec InterfaceParams as PTPInterface params", func() {
				name := "PTPInterfacename"
				namespace := "PTPInterfaceNameSpace"

				params := []string{"param1", "param2"}
				PTPint := ptpinterfaces.PTPInterface{
					PTPInstanceName: "PTPInstanceName",
					Parameters:      params,
				}

				expSpec := PtpInterfaceSpec{
					PtpInstance:         PTPint.PTPInstanceName,
					InterfaceParameters: params,
				}
				expPtpInterface := PtpInterface{
					TypeMeta: metav1.TypeMeta{
						APIVersion: APIVersion,
						Kind:       KindPTPInterface,
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      name,
						Namespace: namespace,
						Labels: map[string]string{
							ControllerToolsLabel: ControllerToolsVersion,
						},
					},
					Spec: expSpec,
				}
				ptpInterface, err := NewPTPInterface(name, namespace, PTPint)
				Expect(err).To(BeNil())
				Expect(*ptpInterface).To(Equal(expPtpInterface))
			})
		})
	})

	Describe("Test NewPtpInstanceSpec", func() {
		Context("When params are not nil", func() {
			It("should give spec InterfaceParams as PTPInstance params", func() {
				params := []string{"param1", "param2"}
				inst := ptpinstances.PTPInstance{
					Service:    "PTPInstanceService",
					Parameters: params,
				}
				expSpec := PtpInstanceSpec{
					Service:            inst.Service,
					InstanceParameters: params,
				}

				ptpInSpec, err := NewPtpInstanceSpec(inst)
				Expect(err).To(BeNil())
				Expect(*ptpInSpec).To(Equal(expSpec))
			})
		})
	})

	Describe("Test NewPTPInstance", func() {
		Context("When params are not nil", func() {
			It("should give spec InstanceParams as PTPInstance params", func() {
				name := "NewPTPInstancename"
				namespace := "NewPTPInstanceNameSpace"

				params := []string{"param1", "param2"}
				inst := ptpinstances.PTPInstance{
					Service:    "PTPInstanceService",
					Parameters: params,
				}
				expSpec := PtpInstanceSpec{
					Service:            inst.Service,
					InstanceParameters: params,
				}
				expPtpInstance := PtpInstance{
					TypeMeta: metav1.TypeMeta{
						APIVersion: APIVersion,
						Kind:       KindPTPInstance,
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      name,
						Namespace: namespace,
						Labels: map[string]string{
							ControllerToolsLabel: ControllerToolsVersion,
						},
					},
					Spec: expSpec,
				}
				ptpInstance, err := NewPTPInstance(name, namespace, inst)
				Expect(err).To(BeNil())
				Expect(*ptpInstance).To(Equal(expPtpInstance))
			})
		})
	})

	Describe("Test NewDataNetworkSpec", func() {
		Context("When needs to get instance of DataNetworkSpec", func() {
			It("Returns instance of DataNetworkSpec without an error", func() {
				mode := datanetworks.EndpointModeDynamic
				uDPPortNo := 1234
				ttl := 11
				multicastGrp := "MulticastGroup"
				net := datanetworks.DataNetwork{
					Type:           datanetworks.TypeVxLAN,
					MTU:            1600,
					Description:    "New DataNetwork Spec",
					Mode:           &mode,
					UDPPortNumber:  &uDPPortNo,
					TTL:            &ttl,
					MulticastGroup: &multicastGrp,
				}

				expSpec := DataNetworkSpec{
					Type:        net.Type,
					Description: &net.Description,
					MTU:         &net.MTU,
					VxLAN: &VxLANInfo{
						EndpointMode:   &mode,
						UDPPortNumber:  &uDPPortNo,
						TTL:            &ttl,
						MulticastGroup: &multicastGrp,
					},
				}

				dnSpec, err := NewDataNetworkSpec(net)
				Expect(err).To(BeNil())
				Expect(*dnSpec).To(Equal(expSpec))
			})
		})
	})

	Describe("Test NewDataNetwork", func() {
		Context("When needs to get instance of DataNetwork", func() {
			It("Returns instance of DataNetwork without an error", func() {
				name := "NewDataNetworkName"
				namespace := "NewDataNetworkNameSpace"

				mode := datanetworks.EndpointModeDynamic
				uDPPortNo := 1234
				ttl := 11
				multicastGrp := "MulticastGroup"
				net := datanetworks.DataNetwork{
					Type:           datanetworks.TypeVxLAN,
					MTU:            1600,
					Description:    "New DataNetwork Spec",
					Mode:           &mode,
					UDPPortNumber:  &uDPPortNo,
					TTL:            &ttl,
					MulticastGroup: &multicastGrp,
				}

				expSpec := DataNetworkSpec{
					Type:        net.Type,
					Description: &net.Description,
					MTU:         &net.MTU,
					VxLAN: &VxLANInfo{
						EndpointMode:   &mode,
						UDPPortNumber:  &uDPPortNo,
						TTL:            &ttl,
						MulticastGroup: &multicastGrp,
					},
				}

				expDN := DataNetwork{
					TypeMeta: metav1.TypeMeta{
						APIVersion: APIVersion,
						Kind:       KindDataNetwork,
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      name,
						Namespace: namespace,
						Labels: map[string]string{
							ControllerToolsLabel: ControllerToolsVersion,
						},
					},
					Spec: expSpec,
				}
				dn, err := NewDataNetwork(name, namespace, net)
				Expect(err).To(BeNil())
				Expect(*dn).To(Equal(expDN))
			})
		})
	})

	Describe("Test NewHostSpec", func() {
		Context("When needs to get instance of HostSpec", func() {
			It("Returns instance of HostSpec without an error", func() {
				hostname := "kernelName"
				bootMAC := "01:02:03:04"
				hostInfo := platform.HostInfo{
					Kernel: kernel.Kernel{
						Hostname: hostname,
					},
				}

				hostInfo.ID = "hostID"
				hostInfo.BootMAC = bootMAC
				expSpec := HostSpec{
					Profile: hostInfo.ID,
					Overrides: &HostProfileSpec{
						ProfileBaseAttributes: ProfileBaseAttributes{
							BootMAC: &bootMAC,
						},
					},
				}

				hSpec, err := NewHostSpec(hostInfo)
				Expect(err).To(BeNil())
				Expect(*hSpec).To(Equal(expSpec))
			})
		})
	})

	Describe("Test NewHost", func() {
		Context("When needs to get instance of Host", func() {
			It("Returns instance of Host without an error", func() {
				name := "NewHostName"
				namespace := "NewHostNameSpace"

				hostname := "kernelName"
				bootMAC := "01:02:03:04"
				hostInfo := platform.HostInfo{
					Kernel: kernel.Kernel{
						Hostname: hostname,
					},
				}

				hostInfo.ID = "hostID"
				hostInfo.BootMAC = bootMAC
				expSpec := HostSpec{
					Profile: hostInfo.ID,
					Overrides: &HostProfileSpec{
						ProfileBaseAttributes: ProfileBaseAttributes{
							BootMAC: &bootMAC,
						},
					},
				}
				expHost := Host{
					TypeMeta: metav1.TypeMeta{
						APIVersion: APIVersion,
						Kind:       KindHost,
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      name,
						Namespace: namespace,
						Labels: map[string]string{
							ControllerToolsLabel: ControllerToolsVersion,
						},
					},
					Spec: expSpec,
				}

				hostGot, err := NewHost(name, namespace, hostInfo)
				Expect(err).To(BeNil())
				Expect(*hostGot).To(Equal(expHost))
			})
		})
	})

	Describe("Test NewSystemStatus", func() {
		Context("When needs to get instance of SystemStatus", func() {
			It("Returns instance of SystemStatus without an error", func() {
				systemInfo := platform.SystemInfo{}
				sysType := "singleNode"
				sysMode := "aio-sx"
				systemInfo.SystemType = sysType
				systemInfo.SystemMode = sysMode
				expStatus := SystemStatus{
					SystemType: sysType,
					SystemMode: sysMode,
				}
				sysStatus, err := NewSystemStatus(systemInfo)
				Expect(err).To(BeNil())
				Expect(*sysStatus).To(Equal(expStatus))
			})
		})
	})

	Describe("Test NewBMSecret", func() {
		Context("When needs to get instance of BMSecret", func() {
			It("Returns instance of BMSecret without an error", func() {
				var name, namespace, username string = "BMSecretName", "BMSecretNameSpace", "BMSecretUserName"
				fakePassword := []byte("")
				expSecret := v1.Secret{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "v1",
						Kind:       "Secret",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      name,
						Namespace: namespace,
					},
					Type: v1.SecretTypeBasicAuth,
					Data: map[string][]byte{
						v1.BasicAuthUsernameKey: []byte(username),
						v1.BasicAuthPasswordKey: fakePassword,
					},
				}
				secretGot, err := NewBMSecret(name, namespace, username)
				Expect(err).To(BeNil())
				Expect(*secretGot).To(Equal(expSecret))
			})
		})
	})

	Describe("Test NewLicenseSecret", func() {
		Context("When needs to get instance of LicenseSecret", func() {
			It("Returns instance of License Secret without an error", func() {
				var name, namespace, content string = "LicenseSecretName", "LicenseSecretNameSpace", "Content"
				expLicSecret := v1.Secret{
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
						SecretLicenseContentKey: []byte(content),
					},
				}
				secretGot, err := NewLicenseSecret(name, namespace, content)
				Expect(err).To(BeNil())
				Expect(*secretGot).To(Equal(expLicSecret))
			})
		})
	})

	Describe("Test NewCertificateSecret", func() {
		Context("When needs to get instance of Certificate secret", func() {
			It("Returns instance of Certificate Secret without an error", func() {
				var name, namespace string = "CertificateSecretName", "CertificateSecretNameSpace"
				fakeInput := []byte("")

				expCertSecret := v1.Secret{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "v1",
						Kind:       "Secret",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      name,
						Namespace: namespace,
					},
					Type: v1.SecretTypeTLS,
					Data: map[string][]byte{
						v1.TLSCertKey:              fakeInput,
						v1.TLSPrivateKeyKey:        fakeInput,
						v1.ServiceAccountRootCAKey: fakeInput,
					},
				}
				secretGot, err := NewCertificateSecret(name, namespace)
				Expect(err).To(BeNil())
				Expect(*secretGot).To(Equal(expCertSecret))
			})
		})
	})

	Describe("Test parseLicenseInfo", func() {
		Context("When license is given populate spec with it", func() {
			It("Adds license to the spec", func() {
				spec := &SystemSpec{}
				license := &licenses.License{
					Content: "content",
				}
				want := LicenseInfo{Secret: SystemDefaultLicenseName}
				err := parseLicenseInfo(spec, license)
				Expect(err).To(BeNil())
				Expect(*spec.License).To(Equal(want))
			})
		})
	})

	Describe("Test parseStorageBackendInfo", func() {
		Context("When storageBackends is given populate spec with them", func() {
			It("Add storageBackends to the spec", func() {
				spec := &SystemSpec{}
				network, rep1, rep2 := "mgmt", 1, 2
				storageBackendsIn := []storagebackends.StorageBackend{
					{
						Name:    "strBnd1",
						Network: network,
						Backend: "strBackend1",
						Capabilities: storagebackends.Capabilities{
							Replication: "1",
						},
					},
					{
						Name:    "strBnd2",
						Network: network,
						Backend: "strBackend2",
						Capabilities: storagebackends.Capabilities{
							Replication: "2",
						},
					},
				}
				storageBackends := []StorageBackend{
					{
						Name:              "strBnd1",
						Type:              "strBackend1",
						Network:           &network,
						ReplicationFactor: &rep1,
					},
					{
						Name:              "strBnd2",
						Type:              "strBackend2",
						Network:           &network,
						ReplicationFactor: &rep2,
					},
				}

				want := StorageBackendList(storageBackends)
				err := parseStorageBackendInfo(spec, storageBackendsIn)
				Expect(err).To(BeNil())
				Expect(*spec.Storage.Backends).To(Equal(want))
			})
		})
	})

	Describe("Test parseFileSystemInfo", func() {
		Context("When Filesystems is given populate spec with them", func() {
			It("Adds filesystems to the spec", func() {
				spec := &SystemSpec{}
				fileSystems := []controllerFilesystems.FileSystem{
					{
						Name: "fsName1",
						Size: 100,
					},
					{
						Name: "fsName2",
						Size: 200,
					},
				}

				fsInfo := []ControllerFileSystemInfo{
					{
						Name: "fsName1",
						Size: 100,
					},
					{
						Name: "fsName2",
						Size: 200,
					},
				}

				want := ControllerFileSystemList(fsInfo)
				err := parseFileSystemInfo(spec, fileSystems)
				Expect(err).To(BeNil())
				Expect(*spec.Storage.FileSystems).To(Equal(want))
			})
		})
	})

	Describe("Test parseServiceParameterInfo", func() {
		Context("When non empty ServiceParameterInfo is given", func() {
			It("populates the spec with Serviceparamters", func() {
				spec := &SystemSpec{}
				resource1, resource2, personality1, personality2 := "resource1", "resource2", "personality1", "personality2"
				serviceParams := []serviceparameters.ServiceParameter{
					{
						Service:     "service1",
						Section:     "Section1",
						ParamName:   "ParamName1",
						ParamValue:  "ParamValue1",
						Resource:    &resource1,
						Personality: &personality1,
					},
					{
						Service:     "service2",
						Section:     "Section2",
						ParamName:   "ParamName2",
						ParamValue:  "ParamValue2",
						Resource:    &resource2,
						Personality: &personality2,
					},
				}
				serviceInfo := []ServiceParameterInfo{
					{
						Service:     "service1",
						Section:     "Section1",
						ParamName:   "ParamName1",
						ParamValue:  "ParamValue1",
						Resource:    &resource1,
						Personality: &personality1,
					},
					{
						Service:     "service2",
						Section:     "Section2",
						ParamName:   "ParamName2",
						ParamValue:  "ParamValue2",
						Resource:    &resource2,
						Personality: &personality2,
					},
				}
				want := ServiceParameterList(serviceInfo)
				err := parseServiceParameterInfo(spec, serviceParams)
				Expect(err).To(BeNil())
				Expect(*spec.ServiceParameters).To(Equal(want))
			})
		})
	})

	Describe("Test parseCertificateInfo", func() {
		Context("When non empty CertificateInfo is given", func() {
			It("Populates the spec with CertificateInfo", func() {
				spec := &SystemSpec{}
				certificates := []certificates.Certificate{
					{
						Type:      "T1",
						Signature: "hash1",
					},
					{
						Type:      "T2",
						Signature: "hash2",
					},
				}
				certInfo := []CertificateInfo{
					{
						Type:      "T1",
						Signature: "hash1",
						Secret:    "T1-cert-secret-0",
					},
					{
						Type:      "T2",
						Signature: "hash2",
						Secret:    "T2-cert-secret-1",
					},
				}
				want := CertificateList(certInfo)
				err := parseCertificateInfo(spec, certificates)
				Expect(err).To(BeNil())
				Expect(*spec.Certificates).To(Equal(want))
			})
		})
	})

	Describe("Test parseCertificateInfo filtering certificates", func() {
		Context("When non empty CertificateInfo is given with openstack_CA/openldap/docker_registry/ssl cert included", func() {
			It("Populates the spec with CertificateInfo without the openstack_CA/openldap/docker_registry/ssl cert", func() {
				spec := &SystemSpec{}
				certificates := []certificates.Certificate{
					{
						Type:      "T1",
						Signature: "hash1",
					},
					{
						Type:      "openstack_ca",
						Signature: "hash2",
					},
					{
						Type:      "openldap",
						Signature: "hash3",
					},
					{
						Type:      "docker_registry",
						Signature: "hash4",
					},
					{
						Type:      "ssl",
						Signature: "hash5",
					},
				}
				certInfo := []CertificateInfo{
					{
						Type:      "T1",
						Signature: "hash1",
						Secret:    "T1-cert-secret-0",
					},
				}
				want := CertificateList(certInfo)
				err := parseCertificateInfo(spec, certificates)
				Expect(err).To(BeNil())
				Expect(*spec.Certificates).To(Equal(want))
			})
		})
	})

	Describe("Test parseMonitorInfo", func() {
		Context("When one of the monitor hostname is same as host name", func() {
			It("Sucessfully adds the monitor info to the profile spec", func() {
				size := 1
				profile := &HostProfileSpec{
					Storage: &ProfileStorageInfo{},
				}
				host := platform.HostInfo{
					Host: hosts.Host{
						Hostname: "hostname1",
					},
					Monitors: []cephmonitors.CephMonitor{
						{
							Hostname: "hostname1",
							Size:     size,
						},
						{
							Hostname: "hostname",
							Size:     size,
						},
					},
				}
				want := MonitorInfo{
					Size: &size,
				}
				err := parseMonitorInfo(profile, host)
				Expect(err).To(BeNil())
				Expect(*profile.Storage.Monitor).To(Equal(want))
			})
		})
	})
	Describe("Test parseHostFileSystemInfo", func() {
		Context("When filesystems is not nil", func() {
			It("Sucessfully parses hostFileSystem Info without any error", func() {
				spec := &HostProfileSpec{
					Storage: &ProfileStorageInfo{},
				}
				fileSystems := []hostFilesystems.FileSystem{
					{
						Name: "FsName1",
						Size: 1,
					},
					{
						Name: "FsName2",
						Size: 2,
					},
				}
				want := FileSystemList{
					{
						Name: "FsName1",
						Size: 1,
					},
					{
						Name: "FsName2",
						Size: 2,
					},
				}
				err := parseHostFileSystemInfo(spec, fileSystems)
				Expect(err).To(BeNil())
				Expect(*spec.Storage.FileSystems).To(Equal(want))
			})
		})
	})
	Describe("Test NewNamespace", func() {
		Context("When the new namespace is to be created with name", func() {
			It("Sucessfully create the namespace instance without error", func() {
				name := "newns"
				expNS := &v1.Namespace{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "v1",
						Kind:       "Namespace",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name: name,
					},
				}
				gotNS, err := NewNamespace(name)
				Expect(err).To(BeNil())
				Expect(gotNS).To(Equal(expNS))
			})
		})
	})
	Describe("Test parseBoardManagementInfo", func() {
		Context("When BMType is not nil", func() {
			It("Host BM info is added to the Profile boardmanagement without any error", func() {
				profile := &HostProfileSpec{
					BoardManagement: &BMInfo{},
				}

				bMUsername, bMAddress, bMType := "BMUsername", "BMAddress", "BMType"
				host := platform.HostInfo{
					Host: hosts.Host{
						BMUsername: &bMUsername,
						BMAddress:  &bMAddress,
						BMType:     &bMType,
					},
				}

				exp := BMInfo{
					Type:    &bMType,
					Address: &bMAddress,
					Credentials: &BMCredentials{
						Password: &BMPasswordInfo{
							Secret: "bmc-secret",
						},
					},
				}

				err := parseBoardManagementInfo(profile, host)
				Expect(err).To(BeNil())
				Expect(*profile.BoardManagement).To(Equal(exp))
			})
		})
		Context("When BMType is nil", func() {
			It("Profile boardmanagement is assigned to nil values without any error", func() {
				profile := &HostProfileSpec{
					BoardManagement: &BMInfo{},
				}
				host := platform.HostInfo{
					Host: hosts.Host{},
				}
				bmType := "none"
				exp := BMInfo{
					Type:        &bmType,
					Address:     nil,
					Credentials: nil,
				}
				err := parseBoardManagementInfo(profile, host)
				Expect(err).To(BeNil())
				Expect(*profile.BoardManagement).To(Equal(exp))
			})
		})
	})
	Describe("Test autoGenerateBMSecretName", func() {
		Context("When autoGenerateBMSecretName is called", func() {
			It("Returns bmc-secret string", func() {
				exp := "bmc-secret"
				got := autoGenerateBMSecretName()
				Expect(got).To(Equal(exp))
			})
		})
	})
	Describe("Test parseOSDInfo", func() {
		Context("When OSDInfo is not nil", func() {
			It("profile osds are populated with host data", func() {
				clusterName := "clusterName"
				profile := &HostProfileSpec{
					Storage: &ProfileStorageInfo{},
				}
				storeTiers := map[string]*storagetiers.StorageTier{}
				storeTiers[clusterName] = &storagetiers.StorageTier{
					ID: "TierUUID",
				}
				location := "journal_loc"
				path := "/a/b/c"
				host := platform.HostInfo{
					OSDs: []osds.OSD{
						{
							TierUUID: "TierUUID",
							DiskID:   "DiskID",
							JournalInfo: osds.JournalInfo{
								Location: &location,
								Path:     &path,
							},
							Function: "journal",
						},
					},
					StorageTiers: storeTiers,
					Disks: []disks.Disk{
						{
							ID:         "DiskID",
							DevicePath: "/a/b/c",
						},
					},
				}

				exp := OSDList{
					{
						Function:    "journal",
						ClusterName: &clusterName,
						Path:        "/a/b/c",
						Journal: &JournalInfo{
							Location: path,
							Size:     0,
						},
					},
				}
				err := parseOSDInfo(profile, host)
				Expect(err).To(BeNil())
				Expect(*profile.Storage.OSDs).To(Equal(exp))
			})
		})
	})
	Describe("Test parseMemoryInfo", func() {
		Context("When Memory Info is not nil", func() {
			It("Stores memoryInfo to profile memory", func() {
				personality := hosts.PersonalityWorker
				profile := &HostProfileSpec{
					ProfileBaseAttributes: ProfileBaseAttributes{
						Personality: &personality,
					},
					Memory: MemoryNodeList{},
				}
				vM2MHugepagesPending := 1
				host := platform.HostInfo{
					Memory: []memory.Memory{
						{
							Processor:             1,
							Platform:              1,
							VM2MHugepagesCount:    1,
							VM2MHugepagesPending:  &vM2MHugepagesPending,
							VSwitchHugepagesSize:  2,
							VM1GHugepagesCount:    1,
							VSwitchHugepagesCount: 3,
						},
					},
				}
				exp := MemoryNodeList{
					{
						Node: 1,
						Functions: MemoryFunctionList{
							{
								Function:  memory.MemoryFunctionPlatform,
								PageSize:  string(PageSize4K),
								PageCount: (1 * int(units.Mebibyte)) / PageSize4K.Bytes(),
							},
							{
								Function:  memory.MemoryFunctionVSwitch,
								PageSize:  string(PageSize2M),
								PageCount: 3,
							},
							{
								Function:  memory.MemoryFunctionVM,
								PageSize:  string(PageSize2M),
								PageCount: vM2MHugepagesPending,
							},
							{
								Function:  memory.MemoryFunctionVM,
								PageSize:  string(PageSize1G),
								PageCount: 1,
							},
						},
					},
				}
				err := parseMemoryInfo(profile, host)
				Expect(err).To(BeNil())
				Expect(profile.Memory).To(Equal(exp))
			})
		})
	})
	Describe("Test parsePhysicalVolumeInfo", func() {
		Context("When physcial volumes deviceUUID is same as partitions ID", func() {
			It("Stores physical volume data into group physcial volumes without error", func() {

				group := &VolumeGroupInfo{
					PhysicalVolumes: PhysicalVolumeList{},
				}
				vg := &volumegroups.VolumeGroup{
					ID: "vgID",
				}
				host := platform.HostInfo{
					PhysicalVolumes: []physicalvolumes.PhysicalVolume{
						{
							ID:            "vID",
							VolumeGroupID: "vgID",
							Type:          physicalvolumes.PVTypePartition,
							DevicePath:    "/a/b/c",
							DeviceUUID:    "DeviceUUID",
						},
					},
					Partitions: []partitions.DiskPartition{
						{
							ID:         "DeviceUUID",
							DevicePath: "/a/b/c",
							Size:       1024,
						},
					},
				}
				size := 1
				exp := PhysicalVolumeList{
					{
						Type: physicalvolumes.PVTypePartition,
						Path: "/a/b/c",
						Size: &size,
					},
				}
				err := parsePhysicalVolumeInfo(group, vg, host)
				Expect(err).To(BeNil())
				Expect(group.PhysicalVolumes).To(Equal(exp))
			})
		})
		Context("When there is no match with deviceUUID of physcial volumes andf partitions ID", func() {
			It("Throws failed to lookup partition DeviceUUID error", func() {

				group := &VolumeGroupInfo{
					PhysicalVolumes: PhysicalVolumeList{},
				}
				vg := &volumegroups.VolumeGroup{
					ID: "vgID",
				}
				host := platform.HostInfo{
					PhysicalVolumes: []physicalvolumes.PhysicalVolume{
						{
							ID:            "vID",
							VolumeGroupID: "vgID",
							Type:          physicalvolumes.PVTypePartition,
							DevicePath:    "/a/b/c",
							DeviceUUID:    "DeviceUUID",
						},
					},
					Partitions: []partitions.DiskPartition{
						{
							ID:         "DeviceUUID1",
							DevicePath: "/a/b/c",
							Size:       1024,
						},
					},
				}
				err := parsePhysicalVolumeInfo(group, vg, host)
				msg := ErrMissingSystemResource{"failed to lookup partition DeviceUUID"}
				Expect(err).To(Equal(msg))

			})
		})
	})

	Describe("Test NewMissingSystemResource func", func() {
		Context("When a non-empty message string is given", func() {
			It("Returns ErrMissingSystemResource with message value as input message string", func() {
				msg := "error msg"
				exp := ErrMissingSystemResource{
					message: "error msg",
				}
				out := NewMissingSystemResource(msg)
				Expect(out).To(Equal(exp))
			})
		})
	})

	Describe("Test Error func", func() {
		Context("When Error func is tested", func() {
			It("returns the message associated with an error of this type.", func() {
				in := ErrMissingSystemResource{
					message: "error msg",
				}
				exp := "error msg"
				out := in.Error()
				Expect(out).To(Equal(exp))
			})
		})
	})

	Describe("Test parseVolumeGroupInfo", func() {
		Context("When volumeGroupInfo of host is not nil", func() {
			It("Stored volumegroup data into the profile storage volume gropus without error", func() {
				profile := &HostProfileSpec{
					Storage: &ProfileStorageInfo{
						VolumeGroups: &VolumeGroupList{},
					},
				}
				lvmType := "LVMType"
				host := platform.HostInfo{
					VolumeGroups: []volumegroups.VolumeGroup{
						{
							LVMInfo: volumegroups.LVMInfo{
								Name: "volumegroupName",
							},
							Capabilities: volumegroups.Capabilities{
								LVMType: &lvmType,
							},
							ID: "vgID",
						},
					},
					PhysicalVolumes: []physicalvolumes.PhysicalVolume{
						{
							ID:            "vID",
							VolumeGroupID: "vgID",
							Type:          physicalvolumes.PVTypePartition,
							DevicePath:    "/a/b/c",
							DeviceUUID:    "DeviceUUID",
						},
					},
					Partitions: []partitions.DiskPartition{
						{
							ID:         "DeviceUUID",
							DevicePath: "/a/b/c",
							Size:       1024,
						},
					},
				}
				size := 1
				exp := VolumeGroupList{
					{
						Name:    "volumegroupName",
						LVMType: &lvmType,
						PhysicalVolumes: PhysicalVolumeList{
							{
								Type: "partition",
								Path: "/a/b/c",
								Size: &size,
							},
						},
					},
				}

				err := parseVolumeGroupInfo(profile, host)
				Expect(err).To(BeNil())
				Expect(*profile.Storage.VolumeGroups).To(Equal(exp))
			})
		})
	})

	Describe("Test parseStorageInfo", func() {
		Context("When Storage Info os host is not nil", func() {
			It("Stores storageInfo to profile spec", func() {
				profile := &HostProfileSpec{
					Storage: &ProfileStorageInfo{
						VolumeGroups: &VolumeGroupList{},
					},
				}
				lvmType := "LVMType"
				host := platform.HostInfo{
					VolumeGroups: []volumegroups.VolumeGroup{
						{
							LVMInfo: volumegroups.LVMInfo{
								Name: "volumegroupName",
							},
							Capabilities: volumegroups.Capabilities{
								LVMType: &lvmType,
							},
							ID: "vgID",
						},
					},
					PhysicalVolumes: []physicalvolumes.PhysicalVolume{
						{
							ID:            "vID",
							VolumeGroupID: "vgID",
							Type:          physicalvolumes.PVTypePartition,
							DevicePath:    "/a/b/c",
							DeviceUUID:    "DeviceUUID",
						},
					},
					Partitions: []partitions.DiskPartition{
						{
							ID:         "DeviceUUID",
							DevicePath: "/a/b/c",
							Size:       1024,
						},
					},
					Host: hosts.Host{
						Personality: "worker",
					},
				}
				err := parseStorageInfo(profile, host)
				Expect(err).To(BeNil())
			})
		})
	})

	Describe("Test parseInterfaceInfo", func() {
		Context("When host interface data is not nil", func() {
			It("Stores interface data into profile spec", func() {
				profile := &HostProfileSpec{
					Storage: &ProfileStorageInfo{
						VolumeGroups: &VolumeGroupList{},
					},
				}
				iPv4Pool := "01:22:33:12"
				iPv6Pool := "11:29:43:12"
				pTPRole := "ptp master"
				vID := 1234
				aEMode := "AEMode"
				aETransmitHash := "AETransmitHash"
				aEPrimReselect := "AEPrimReselect"
				vFCount := 5
				vFDriver := "VFDriver"
				maxTxRate := 7
				host := platform.HostInfo{
					PTPInterfaces: []ptpinterfaces.PTPInterface{
						{
							Name:           "PTPInterfaceName",
							InterfaceNames: []string{"hostName/infID"},
						},
						{
							Name:           "PTPInterfaceName1",
							InterfaceNames: []string{"hostName/infID1"},
						},
					},
					InterfaceNetworks: []interfaceNetworks.InterfaceNetwork{
						{
							InterfaceUUID: "infID",
							NetworkName:   "infNetName",
						},
						{
							InterfaceUUID: "infID1",
							NetworkName:   "infNetName2",
						},
					},
					Pools: []addresspools.AddressPool{
						{
							ID:   iPv4Pool,
							Name: "poolName4",
						},
						{
							ID:   iPv6Pool,
							Name: "poolName6",
						},
					},
					Interfaces: []interfaces.Interface{
						{
							ID:       "infID",
							Name:     "infName",
							Class:    interfaces.IFClassPlatform,
							MTU:      4,
							IPv4Pool: &iPv4Pool,
							IPv6Pool: &iPv6Pool,
							PTPRole:  &pTPRole,
							Type:     "ethernet",
							Uses:     []string{"use1", "use2"},
						},
						{
							ID:    "infID1",
							Name:  "infName1",
							Class: interfaces.IFClassPlatform,
							MTU:   10,
							Type:  interfaces.IFTypeVLAN,
							Uses:  []string{"use3"},
							VID:   &vID,
						},
						{
							ID:             "infID2",
							Name:           "infName2",
							Class:          interfaces.IFClassPlatform,
							MTU:            9,
							Type:           interfaces.IFTypeAE,
							Uses:           []string{"use4"},
							AEMode:         &aEMode,
							AETransmitHash: &aETransmitHash,
							AEPrimReselect: &aEPrimReselect,
						},
						{
							ID:    "infID3",
							Name:  "infName3",
							Class: interfaces.IFClassPlatform,
							MTU:   10,
							Type:  interfaces.IFTypeVirtual,
							Uses:  []string{"use5"},
						},
						{
							ID:        "infID4",
							Name:      "infName4",
							Class:     interfaces.IFClassPlatform,
							MTU:       7,
							Type:      interfaces.IFTypeVF,
							Uses:      []string{"use6"},
							VFCount:   &vFCount,
							VFDriver:  &vFDriver,
							MaxTxRate: &maxTxRate,
						},
					},

					Host: hosts.Host{
						Hostname:    "hostName",
						Personality: "worker",
					},
				}
				err := parseInterfaceInfo(profile, host)
				Expect(err).To(BeNil())
			})
		})
	})
	Describe("Test NewSystemSpec", func() {
		Context("When the system info is used to create systemspec", func() {
			It("Returns the systemSpec with systemInfo data without any error", func() {
				network := "mgmt"
				vSwitchType := "vswitch_type"
				description := "systemInfo description"
				location := "/a/b/c"
				latitude := "45.02"
				longitude := "98.70"
				contact := "fakeName"
				mode := "duplex"
				transport := "duplex"
				mechanism := "PTPMechanism"
				repFactor := 1
				sysInfo := platform.SystemInfo{
					DRBD: &drbd.DRBD{
						LinkUtilization: 27,
					},
					DNS: &dns.DNS{
						Nameservers: "DNS1,DNS2",
					},
					NTP: &ntp.NTP{
						NTPServers: "NTP1,NTP2",
					},
					PTP: &ptp.PTP{
						Mode:      mode,
						Transport: transport,
						Mechanism: mechanism,
					},
					Certificates: []certificates.Certificate{
						{
							Type:      "T1",
							Signature: "hash1",
						},
					},
					ServiceParameters: []serviceparameters.ServiceParameter{
						{
							Service: "service1",
							Section: "Section1",
						},
					},
					FileSystems: []controllerFilesystems.FileSystem{
						{
							Name: "fsName1",
							Size: 100,
						},
					},
					StorageBackends: []storagebackends.StorageBackend{
						{
							Name:    "strBnd1",
							Network: network,
							Backend: "strBackend1",
							Capabilities: storagebackends.Capabilities{
								Replication: "1",
							},
						},
					},
					License: &licenses.License{
						Content: "content",
					},
				}

				sysInfo.Location = location
				sysInfo.Description = description
				sysInfo.Contact = contact
				sysInfo.Latitude = latitude
				sysInfo.Longitude = longitude
				sysInfo.Capabilities.VSwitchType = vSwitchType
				expSpec := SystemSpec{
					Description: &description,
					Location:    &location,
					Latitude:    &latitude,
					Longitude:   &longitude,
					Contact:     &contact,
					DNSServers:  &DNSServerList{"DNS1", "DNS2"},
					NTPServers:  &NTPServerList{"NTP1", "NTP2"},
					PTP: &PTPInfo{
						Mode:      &mode,
						Transport: &transport,
						Mechanism: &mechanism,
					},
					Certificates: &CertificateList{
						{
							Type:      "T1",
							Secret:    "T1-cert-secret-0",
							Signature: "hash1",
						},
					},
					License: &LicenseInfo{
						Secret: "system-license",
					},
					ServiceParameters: &ServiceParameterList{
						{Service: "service1", Section: "Section1", ParamName: "", ParamValue: "", Personality: nil, Resource: nil},
					},
					Storage: &SystemStorageInfo{
						Backends: &StorageBackendList{
							{Name: "strBnd1", Type: "strBackend1", Services: nil, ReplicationFactor: &repFactor, PartitionSize: nil, Network: &network},
						},
						DRBD: &DRBDConfiguration{
							LinkUtilization: 27,
						},
						FileSystems: &ControllerFileSystemList{
							{Name: "fsName1", Size: 100},
						},
					},
					VSwitchType: &vSwitchType,
				}

				outSpec, err := NewSystemSpec(sysInfo)
				Expect(err).To(BeNil())
				Expect(*outSpec).To(Equal(expSpec))

			})
		})
	})
	Describe("Test NewSystem", func() {
		Context("When the new system instance needs to be obtained from the system info", func() {
			It("Returns the system instance without any error", func() {
				namespace := "ns"
				name := "systemName"
				network := "mgmt"
				vSwitchType := "vswitch_type"
				description := "systemInfo description"
				location := "/a/b/c"
				latitude := "45.02"
				longitude := "98.70"
				contact := "fakeName"
				mode := "operationMode"
				transport := "transportMode"
				mechanism := "PTPMechanism"
				repFactor := 1
				sysInfo := platform.SystemInfo{
					DRBD: &drbd.DRBD{
						LinkUtilization: 27,
					},
					DNS: &dns.DNS{
						Nameservers: "DNS1,DNS2",
					},
					NTP: &ntp.NTP{
						NTPServers: "NTP1,NTP2",
					},
					PTP: &ptp.PTP{
						Mode:      mode,
						Transport: transport,
						Mechanism: mechanism,
					},
					Certificates: []certificates.Certificate{
						{
							Type:      "T1",
							Signature: "hash1",
						},
					},
					ServiceParameters: []serviceparameters.ServiceParameter{
						{
							Service: "service1",
							Section: "Section1",
						},
					},
					FileSystems: []controllerFilesystems.FileSystem{
						{
							Name: "fsName1",
							Size: 100,
						},
					},
					StorageBackends: []storagebackends.StorageBackend{
						{
							Name:    "strBnd1",
							Network: network,
							Backend: "strBackend1",
							Capabilities: storagebackends.Capabilities{
								Replication: "1",
							},
						},
					},
					License: &licenses.License{
						Content: "content",
					},
				}

				sysInfo.Location = location
				sysInfo.Description = description
				sysInfo.Contact = contact
				sysInfo.Latitude = latitude
				sysInfo.Longitude = longitude
				sysInfo.Capabilities.VSwitchType = vSwitchType
				expSys := System{
					TypeMeta: metav1.TypeMeta{
						APIVersion: APIVersion,
						Kind:       KindSystem,
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      name,
						Namespace: namespace,
						Labels: map[string]string{
							ControllerToolsLabel: ControllerToolsVersion,
						},
					},
					Spec: SystemSpec{
						Description: &description,
						Location:    &location,
						Latitude:    &latitude,
						Longitude:   &longitude,
						Contact:     &contact,
						DNSServers:  &DNSServerList{"DNS1", "DNS2"},
						NTPServers:  &NTPServerList{"NTP1", "NTP2"},
						PTP: &PTPInfo{
							Mode:      &mode,
							Transport: &transport,
							Mechanism: &mechanism,
						},
						Certificates: &CertificateList{
							{
								Type:      "T1",
								Secret:    "T1-cert-secret-0",
								Signature: "hash1",
							},
						},
						License: &LicenseInfo{
							Secret: "system-license",
						},
						ServiceParameters: &ServiceParameterList{
							{Service: "service1", Section: "Section1", ParamName: "", ParamValue: "", Personality: nil, Resource: nil},
						},
						Storage: &SystemStorageInfo{
							Backends: &StorageBackendList{
								{Name: "strBnd1", Type: "strBackend1", Services: nil, ReplicationFactor: &repFactor, PartitionSize: nil, Network: &network},
							},
							DRBD: &DRBDConfiguration{
								LinkUtilization: 27,
							},
							FileSystems: &ControllerFileSystemList{
								{Name: "fsName1", Size: 100},
							},
						},
						VSwitchType: &vSwitchType,
					},
				}
				outSys, err := NewSystem(namespace, name, sysInfo)
				Expect(err).To(BeNil())
				Expect(*outSys).To(Equal(expSys))
			})
		})
	})
	Describe("Test NewHostProfileSpec", func() {
		Context("When the host info is used to create HostProfileSpec", func() {
			It("Returns the HostProfileSpec with host info data without any error", func() {
				administrativeState := "Admin"
				personality := "worker"
				bootMAC := "00:11:22:33"
				installOutput := "InstallOutput"
				provisionedKernel := "ProvisionedKernel"
				hwSettle := "HwSettle"
				appArmor := "AppArmor"
				maxCPUMhzConfigured := "26"
				locName := "locName"
				rootDevice := "rootDevice"
				bootDevice := "BootDevice"
				fakeClock := "fakeClock"
				params := []string{"param1", "param2"}
				worker := "worker"
				console := "console"
				powerOn := true
				bmiInfoType := "none"
				host := platform.HostInfo{
					Host: hosts.Host{
						Hostname:            "hostName",
						Personality:         personality,
						SubFunctions:        "func1,func2",
						AdministrativeState: administrativeState,
						BootMAC:             bootMAC,
						InstallOutput:       installOutput,
					},
					Kernel: kernel.Kernel{
						ProvisionedKernel: provisionedKernel,
					},
					PTPInstances: []ptpinstances.PTPInstance{
						{
							Service:    "PTPInstanceService",
							Parameters: params,
						},
					},
				}

				host.BootDevice = bootDevice
				host.RootDevice = rootDevice
				host.ClockSynchronization = &fakeClock
				host.Location.Name = &locName
				host.AppArmor = appArmor
				host.MaxCPUMhzConfigured = maxCPUMhzConfigured
				host.HwSettle = hwSettle
				host.Personality = worker
				host.Console = console
				expSpec := HostProfileSpec{
					Base: nil,
					ProfileBaseAttributes: ProfileBaseAttributes{
						Personality:          &worker,
						AdministrativeState:  &administrativeState,
						SubFunctions:         []SubFunction{"func1", "func2"},
						Location:             &locName,
						Labels:               nil,
						InstallOutput:        &installOutput,
						Console:              &console,
						BootDevice:           &bootDevice,
						PowerOn:              &powerOn,
						ProvisioningMode:     nil,
						BootMAC:              &bootMAC,
						PtpInstances:         PtpInstanceItemList{""},
						RootDevice:           &rootDevice,
						ClockSynchronization: &fakeClock,
						MaxCPUMhzConfigured:  &maxCPUMhzConfigured,
						AppArmor:             &appArmor,
						HwSettle:             &hwSettle,
						Kernel:               &provisionedKernel,
					},
					BoardManagement: &BMInfo{Type: &bmiInfoType, Address: nil, Credentials: nil},
					Processors:      nil,
					Memory:          nil,
					Storage:         &ProfileStorageInfo{Monitor: nil, OSDs: nil, VolumeGroups: nil, FileSystems: &FileSystemList{}},
					Interfaces:      &InterfaceInfo{Ethernet: nil, VLAN: nil, Bond: nil, VF: nil},
					Addresses:       nil,
					Routes:          nil,
				}
				outSpec, err := NewHostProfileSpec(host)
				Expect(err).To(BeNil())
				Expect(*outSpec).To(Equal(expSpec))
			})
		})
	})
	Describe("Test NewHostProfile", func() {
		Context("When the new hostprofile instance needs to be obtained from the host info", func() {
			It("Returns the hostProfile instance without any error", func() {
				name, namespace := "hostName", "hostNS"
				administrativeState := "Admin"
				personality := "worker"
				bootMAC := "00:11:22:33"
				installOutput := "InstallOutput"
				provisionedKernel := "ProvisionedKernel"
				hwSettle := "HwSettle"
				appArmor := "AppArmor"
				maxCPUMhzConfigured := "26"
				locName := "locName"
				rootDevice := "rootDevice"
				bootDevice := "BootDevice"
				fakeClock := "fakeClock"
				params := []string{"param1", "param2"}
				worker := "worker"
				console := "console"
				powerOn := true
				bmiInfoType := "none"
				host := platform.HostInfo{
					Host: hosts.Host{
						Hostname:            "hostName",
						Personality:         personality,
						SubFunctions:        "func1,func2",
						AdministrativeState: administrativeState,
						BootMAC:             bootMAC,
						InstallOutput:       installOutput,
					},
					Kernel: kernel.Kernel{
						ProvisionedKernel: provisionedKernel,
					},
					PTPInstances: []ptpinstances.PTPInstance{
						{
							Service:    "PTPInstanceService",
							Parameters: params,
						},
					},
				}

				host.BootDevice = bootDevice
				host.RootDevice = rootDevice
				host.ClockSynchronization = &fakeClock
				host.Location.Name = &locName
				host.AppArmor = appArmor
				host.MaxCPUMhzConfigured = maxCPUMhzConfigured
				host.HwSettle = hwSettle
				host.Personality = worker
				host.Console = console
				hostProfileSpec := HostProfileSpec{
					Base: nil,
					ProfileBaseAttributes: ProfileBaseAttributes{
						Personality:          &worker,
						AdministrativeState:  &administrativeState,
						SubFunctions:         []SubFunction{"func1", "func2"},
						Location:             &locName,
						Labels:               nil,
						InstallOutput:        &installOutput,
						Console:              &console,
						BootDevice:           &bootDevice,
						PowerOn:              &powerOn,
						ProvisioningMode:     nil,
						BootMAC:              &bootMAC,
						PtpInstances:         PtpInstanceItemList{""},
						RootDevice:           &rootDevice,
						ClockSynchronization: &fakeClock,
						MaxCPUMhzConfigured:  &maxCPUMhzConfigured,
						AppArmor:             &appArmor,
						HwSettle:             &hwSettle,
						Kernel:               &provisionedKernel,
					},
					BoardManagement: &BMInfo{Type: &bmiInfoType, Address: nil, Credentials: nil},
					Processors:      nil,
					Memory:          nil,
					Storage:         &ProfileStorageInfo{Monitor: nil, OSDs: nil, VolumeGroups: nil, FileSystems: &FileSystemList{}},
					Interfaces:      &InterfaceInfo{Ethernet: nil, VLAN: nil, Bond: nil, VF: nil},
					Addresses:       nil,
					Routes:          nil,
				}
				exp := HostProfile{
					TypeMeta: metav1.TypeMeta{
						APIVersion: APIVersion,
						Kind:       KindHostProfile,
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      name + "-profile",
						Namespace: namespace,
						Labels: map[string]string{
							ControllerToolsLabel: ControllerToolsVersion,
						},
					},
					Spec: hostProfileSpec,
				}
				out, err := NewHostProfile(name, namespace, host)
				Expect(err).To(BeNil())
				Expect(*out).To(Equal(exp))
			})
		})
	})
})
